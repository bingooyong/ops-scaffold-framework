package daemon

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"net"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/agent"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/collector"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/comm"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	grpcclient "github.com/bingooyong/ops-scaffold-framework/daemon/internal/grpc"
	grpcserver "github.com/bingooyong/ops-scaffold-framework/daemon/internal/grpc"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/proto"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Daemon 守护进程
type Daemon struct {
	config                *config.Config
	logger                *zap.Logger
	nodeID                string
	collectorManager      *collector.Manager
	agentManager          *agent.Manager               // 旧格式，向后兼容
	multiAgentManager     *agent.MultiAgentManager     // 新格式，多Agent管理
	healthChecker         *agent.HealthChecker         // 旧格式，向后兼容
	multiHealthChecker    *agent.MultiHealthChecker    // 新格式，多Agent健康检查
	heartbeatReceiver     *agent.HeartbeatReceiver     // 旧格式，Unix socket
	httpHeartbeatReceiver *agent.HTTPHeartbeatReceiver // 新格式，HTTP endpoint
	resourceMonitor       *agent.ResourceMonitor       // 资源监控器
	logManager            *agent.LogManager            // 日志管理器
	stateSyncer           *agent.StateSyncer           // Agent状态同步器
	httpServer            *http.Server                 // HTTP服务器
	grpcClient            *comm.GRPCClient
	managerClient         *grpcclient.ManagerClient // Manager gRPC客户端(用于上报Agent状态)
	grpcServer            *grpc.Server              // gRPC服务器
	grpcListener          net.Listener              // gRPC监听器
	ctx                   context.Context
	cancel                context.CancelFunc
	wg                    sync.WaitGroup
}

// New 创建Daemon实例
func New(cfg *config.Config, logger *zap.Logger) (*Daemon, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 生成或加载节点ID
	nodeID, err := loadOrGenerateNodeID(cfg, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load or generate node ID: %w", err)
	}

	// 创建采集器
	collectors := createCollectors(cfg, logger)
	collectorMgr := collector.NewManager(collectors, logger)

	// 创建Agent管理器（支持新格式和旧格式）
	var agentMgr *agent.Manager
	var multiAgentMgr *agent.MultiAgentManager
	var healthChecker *agent.HealthChecker
	var multiHealthChecker *agent.MultiHealthChecker
	var heartbeatReceiver *agent.HeartbeatReceiver         // 旧格式，Unix socket
	var httpHeartbeatReceiver *agent.HTTPHeartbeatReceiver // 新格式，HTTP endpoint
	var logManager *agent.LogManager                       // 日志管理器

	// 检查是否使用新格式（多Agent配置）
	if len(cfg.Agents) > 0 {
		// 使用新格式：MultiAgentManager
		logger.Info("using multi-agent configuration")
		var err error
		multiAgentMgr, err = agent.NewMultiAgentManager(cfg.Daemon.WorkDir, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create multi-agent manager: %w", err)
		}

		// 从配置加载Agent到注册表
		if err := agent.LoadAgentsFromConfig(cfg, multiAgentMgr.GetRegistry(), cfg.Daemon.WorkDir, logger); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to load agents from config: %w", err)
		}

		// 从注册表创建AgentInstance
		if err := multiAgentMgr.LoadAgentsFromRegistry(); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to load agents from registry: %w", err)
		}

		logger.Info("loaded agents from config",
			zap.Int("count", multiAgentMgr.Count()))

		// 创建日志管理器
		logManager = agent.NewLogManager(cfg.Daemon.WorkDir, logger)
		// TODO: 从配置读取日志保留天数（如果配置中有）
		// logManager.SetRetentionDays(cfg.Log.RetentionDays)

		// 为每个Agent实例设置日志轮转器
		// 默认配置：最大100MB，保留7个文件，压缩旧文件
		const defaultMaxSize = 100 * 1024 * 1024 // 100MB
		const defaultMaxFiles = 7
		for _, instance := range multiAgentMgr.ListAgents() {
			workDir := instance.GetInfo().WorkDir
			if workDir == "" {
				workDir = cfg.Daemon.WorkDir
			}
			logPath := fmt.Sprintf("%s/agents/%s/logs/agent.log", workDir, instance.GetInfo().ID)
			rotator := agent.NewLogRotator(logPath, defaultMaxSize, defaultMaxFiles, logger)
			instance.SetLogRotator(rotator)
		}

		// 创建多Agent健康检查器
		healthCheckerCfg := agent.BuildMultiHealthCheckerConfig(cfg)
		multiHealthChecker = agent.NewMultiHealthChecker(multiAgentMgr, healthCheckerCfg, logger)

		// 创建HTTP心跳接收器(用于接收Agent心跳上报)
		httpHeartbeatReceiver = agent.NewHTTPHeartbeatReceiver(multiAgentMgr, multiAgentMgr.GetRegistry(), logger)
	} else if cfg.Agent.BinaryPath != "" {
		// 使用旧格式：单Agent Manager（向后兼容）
		logger.Info("using legacy single-agent configuration")
		agentMgr = agent.NewManager(&cfg.Agent, logger)
		healthChecker = agent.NewHealthChecker(&cfg.Agent.HealthCheck, agentMgr, logger)
		heartbeatReceiver = agent.NewHeartbeatReceiver(cfg.Agent.SocketPath, healthChecker, logger)
	} else {
		logger.Info("agent management disabled (no agents configured)")
	}

	// 创建gRPC客户端(用于ManagerService)
	grpcClient := comm.NewGRPCClient(&cfg.Manager, logger)

	// 创建Manager gRPC客户端(用于DaemonService,上报Agent状态)
	var managerClient *grpcclient.ManagerClient
	var stateSyncer *agent.StateSyncer
	if multiAgentMgr != nil && cfg.Manager.Address != "" {
		managerClient = grpcclient.NewManagerClient(&cfg.Manager, logger)
		stateSyncer = agent.NewStateSyncer(
			multiAgentMgr,
			multiAgentMgr.GetRegistry(),
			cfg.Manager.Address,
			logger,
		)
		stateSyncer.SetManagerClient(managerClient)

		// 注册状态变化回调到MultiAgentManager
		multiAgentMgr.SetStateChangeCallback(func(agentID string, status agent.AgentStatus, pid int, lastHeartbeat time.Time) {
			if stateSyncer != nil {
				stateSyncer.OnAgentStateChange(agentID, status, pid, lastHeartbeat)
			}
		})
	}

	var resourceMonitor *agent.ResourceMonitor
	if multiAgentMgr != nil {
		resourceMonitor = agent.NewResourceMonitor(multiAgentMgr, multiAgentMgr.GetRegistry(), logger)
		// 从配置读取阈值配置(如果配置中有)
		// 遍历所有Agent配置,设置资源阈值
		for _, agentCfg := range cfg.Agents {
			if agentCfg.HealthCheck.CPUThreshold > 0 || agentCfg.HealthCheck.MemoryThreshold > 0 {
				threshold := &agent.ResourceThreshold{
					CPUThreshold:      agentCfg.HealthCheck.CPUThreshold,
					MemoryThreshold:   agentCfg.HealthCheck.MemoryThreshold,
					ThresholdDuration: agentCfg.HealthCheck.ThresholdDuration,
				}
				resourceMonitor.SetThreshold(agentCfg.ID, threshold)
			}
		}
	}

	d := &Daemon{
		config:                cfg,
		logger:                logger,
		nodeID:                nodeID,
		collectorManager:      collectorMgr,
		agentManager:          agentMgr,
		multiAgentManager:     multiAgentMgr,
		healthChecker:         healthChecker,
		multiHealthChecker:    multiHealthChecker,
		heartbeatReceiver:     heartbeatReceiver,
		httpHeartbeatReceiver: httpHeartbeatReceiver,
		resourceMonitor:       resourceMonitor,
		logManager:            logManager,
		stateSyncer:           stateSyncer,
		grpcClient:            grpcClient,
		managerClient:         managerClient,
		ctx:                   ctx,
		cancel:                cancel,
	}

	// 创建gRPC服务器（如果配置了MultiAgentManager）
	if multiAgentMgr != nil && resourceMonitor != nil {
		grpcPort := cfg.Daemon.GRPCPort
		if grpcPort == 0 {
			grpcPort = 9091 // 默认端口9091（避免与Manager的9090冲突）
		}

		// 创建gRPC服务器实例
		grpcServerImpl := grpcserver.NewServer(multiAgentMgr, resourceMonitor, logger)

		// 创建gRPC服务器
		d.grpcServer = grpc.NewServer()

		// 注册服务
		proto.RegisterDaemonServiceServer(d.grpcServer, grpcServerImpl)

		// 创建监听器
		addr := fmt.Sprintf(":%d", grpcPort)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create gRPC listener: %w", err)
		}
		d.grpcListener = listener

		logger.Info("gRPC server initialized",
			zap.Int("port", grpcPort))
	}

	return d, nil
}

// Start 启动Daemon
func (d *Daemon) Start() error {
	d.logger.Info("starting daemon", zap.String("node_id", d.nodeID))

	// 1. 创建工作目录
	if err := os.MkdirAll(d.config.Daemon.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// 2. 写入PID文件
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// 3. 启动Agent进程
	if d.multiAgentManager != nil {
		// 新格式：启动所有Agent
		d.logger.Info("starting all agents", zap.Int("count", d.multiAgentManager.Count()))

		// 创建工作目录
		agentsWorkDir := fmt.Sprintf("%s/agents", d.config.Daemon.WorkDir)
		if err := os.MkdirAll(agentsWorkDir, 0755); err != nil {
			d.logger.Error("failed to create agents work directory", zap.Error(err))
		}

		// 启动所有Agent
		results := d.multiAgentManager.StartAll(d.ctx)
		successCount := 0
		for agentID, err := range results {
			if err != nil {
				d.logger.Error("failed to start agent",
					zap.String("agent_id", agentID),
					zap.Error(err))
			} else {
				successCount++
			}
		}

		d.logger.Info("agents started",
			zap.Int("total", len(results)),
			zap.Int("success", successCount),
			zap.Int("failed", len(results)-successCount))

		// 启动多Agent健康检查器
		if d.multiHealthChecker != nil {
			d.multiHealthChecker.Start()
			d.logger.Info("multi-agent health checker started")
		}

		// 启动HTTP服务器(用于接收Agent心跳)
		if d.httpHeartbeatReceiver != nil {
			if err := d.startHTTPServer(); err != nil {
				d.logger.Error("failed to start HTTP server", zap.Error(err))
				// 不中断启动，继续运行
			} else {
				d.logger.Info("HTTP heartbeat receiver started")
			}
		}

		// 启动资源监控器
		if d.resourceMonitor != nil {
			d.resourceMonitor.Start()
			d.logger.Info("resource monitor started")
		}

		// 启动日志清理任务
		if d.logManager != nil {
			d.logManager.StartCleanupTask()
			d.logger.Info("log manager started")
		}
	} else if d.agentManager != nil {
		// 旧格式：启动单个Agent（向后兼容）
		if err := d.agentManager.Start(d.ctx); err != nil {
			d.logger.Error("failed to start agent", zap.Error(err))
			// 不中断启动，继续运行采集器
		} else {
			// 4. 启动心跳接收器
			if d.heartbeatReceiver != nil {
				if err := d.heartbeatReceiver.Start(); err != nil {
					d.logger.Error("failed to start heartbeat receiver", zap.Error(err))
				}
			}

			// 5. 启动健康检查
			if d.healthChecker != nil {
				d.healthChecker.Start()
			}
		}
	} else {
		d.logger.Info("agent management disabled (no agents configured)")
	}

	// 6. 启动采集器
	d.collectorManager.Start()

	// 7. 连接Manager并注册（如果配置了Manager地址）
	if d.config.Manager.Address != "" {
		if err := d.connectAndRegister(); err != nil {
			d.logger.Error("failed to connect to manager", zap.Error(err))
			// 不中断启动，后台会重试连接
		}

		// 连接ManagerClient(用于上报Agent状态)
		if d.managerClient != nil {
			ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
			if err := d.managerClient.Connect(ctx); err != nil {
				d.logger.Error("failed to connect manager client for agent state sync", zap.Error(err))
				// 不中断启动，StateSyncer会在下次同步时重试
			} else {
				// 启动StateSyncer
				if d.stateSyncer != nil {
					d.stateSyncer.Start(d.nodeID)
					d.logger.Info("state syncer started")
				}
			}
			cancel()
		}
	} else {
		d.logger.Info("manager connection disabled (no address configured)")
	}

	// 8. 启动gRPC服务器（如果已初始化）
	if d.grpcServer != nil && d.grpcListener != nil {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.logger.Info("gRPC server starting",
				zap.String("address", d.grpcListener.Addr().String()))
			if err := d.grpcServer.Serve(d.grpcListener); err != nil {
				d.logger.Error("gRPC server error",
					zap.Error(err))
			}
		}()
	}

	// 9. 启动后台任务（如果配置了Manager）
	if d.config.Manager.Address != "" {
		d.wg.Add(2)
		go d.heartbeatLoop()
		go d.reportMetricsLoop()
	}

	d.logger.Info("daemon started successfully")

	return nil
}

// Stop 停止Daemon
func (d *Daemon) Stop() {
	d.logger.Info("stopping daemon")

	// 停止后台任务
	d.cancel()
	d.wg.Wait()

	// 停止各个组件
	if d.stateSyncer != nil {
		d.stateSyncer.Stop()
	}
	if d.logManager != nil {
		d.logManager.StopCleanupTask()
	}
	if d.resourceMonitor != nil {
		d.resourceMonitor.Stop()
	}
	if d.httpHeartbeatReceiver != nil {
		d.httpHeartbeatReceiver.Stop()
	}
	if d.heartbeatReceiver != nil {
		d.heartbeatReceiver.Stop()
	}
	if d.multiHealthChecker != nil {
		d.multiHealthChecker.Stop()
	}
	if d.healthChecker != nil {
		d.healthChecker.Stop()
	}
	d.collectorManager.Stop()

	// 停止HTTP服务器
	if d.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := d.httpServer.Shutdown(ctx); err != nil {
			d.logger.Error("failed to shutdown HTTP server", zap.Error(err))
		} else {
			d.logger.Info("HTTP server stopped")
		}
	}

	// 停止gRPC服务器
	if d.grpcServer != nil {
		d.logger.Info("stopping gRPC server")
		d.grpcServer.GracefulStop()
		if d.grpcListener != nil {
			d.grpcListener.Close()
		}
		d.logger.Info("gRPC server stopped")
	}

	// 停止Agent进程
	if d.multiAgentManager != nil {
		// 新格式：停止所有Agent
		d.logger.Info("stopping all agents", zap.Int("count", d.multiAgentManager.Count()))
		results := d.multiAgentManager.StopAll(d.ctx, true)
		successCount := 0
		for agentID, err := range results {
			if err != nil {
				d.logger.Error("failed to stop agent",
					zap.String("agent_id", agentID),
					zap.Error(err))
			} else {
				successCount++
			}
		}
		d.logger.Info("agents stopped",
			zap.Int("total", len(results)),
			zap.Int("success", successCount))
	} else if d.agentManager != nil {
		// 旧格式：停止单个Agent
		// 注意：不停止Agent进程，让它继续运行（向后兼容原有行为）
		d.logger.Info("agent will continue running after daemon stops")
	}

	// 关闭gRPC连接
	if err := d.grpcClient.Close(); err != nil {
		d.logger.Error("failed to close grpc client", zap.Error(err))
	}
	if d.managerClient != nil {
		if err := d.managerClient.Close(); err != nil {
			d.logger.Error("failed to close manager client", zap.Error(err))
		}
	}

	// 删除PID文件
	os.Remove(d.config.Daemon.PIDFile)

	d.logger.Info("daemon stopped")
}

// connectAndRegister 连接Manager并注册节点
func (d *Daemon) connectAndRegister() error {
	// 连接Manager
	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	if err := d.grpcClient.Connect(ctx); err != nil {
		return err
	}

	// 注册节点
	hostname, _ := os.Hostname()
	nodeInfo := &types.NodeInfo{
		NodeID:     d.nodeID,
		Hostname:   hostname,
		IP:         d.getLocalIP(),
		OS:         d.getOS(),
		Arch:       d.getArch(),
		Labels:     make(map[string]string),
		DaemonVer:  "1.0.0", // TODO: 从version包获取
		AgentVer:   "1.0.0", // TODO: 从Agent获取
		RegisterAt: time.Now(),
	}

	nodeID, err := d.grpcClient.Register(ctx, d.nodeID, nodeInfo)
	if err != nil {
		return err
	}

	d.nodeID = nodeID
	d.logger.Info("registered to manager", zap.String("node_id", d.nodeID))

	return nil
}

// heartbeatLoop 心跳循环
func (d *Daemon) heartbeatLoop() {
	defer d.wg.Done()

	ticker := time.NewTicker(d.config.Manager.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
			if err := d.grpcClient.Heartbeat(ctx); err != nil {
				d.logger.Error("failed to send heartbeat", zap.Error(err))
			}
			cancel()
		}
	}
}

// reportMetricsLoop 指标上报循环
func (d *Daemon) reportMetricsLoop() {
	defer d.wg.Done()

	ticker := time.NewTicker(d.config.Manager.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			metrics := d.collectorManager.GetLatest()
			if len(metrics) == 0 {
				continue
			}

			ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
			if err := d.grpcClient.ReportMetrics(ctx, metrics); err != nil {
				d.logger.Error("failed to report metrics", zap.Error(err))
			}
			cancel()
		}
	}
}

// writePIDFile 写入PID文件
func (d *Daemon) writePIDFile() error {
	pid := os.Getpid()
	return os.WriteFile(d.config.Daemon.PIDFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// createCollectors 创建所有采集器
func createCollectors(cfg *config.Config, logger *zap.Logger) []collector.Collector {
	collectors := make([]collector.Collector, 0)

	// CPU采集器
	if cfg.Collectors.CPU.Enabled {
		collectors = append(collectors,
			collector.NewCPUCollector(true, cfg.Collectors.CPU.Interval, logger))
	}

	// 内存采集器
	if cfg.Collectors.Memory.Enabled {
		collectors = append(collectors,
			collector.NewMemoryCollector(true, cfg.Collectors.Memory.Interval, logger))
	}

	// 磁盘采集器
	if cfg.Collectors.Disk.Enabled {
		collectors = append(collectors,
			collector.NewDiskCollector(true, cfg.Collectors.Disk.Interval, cfg.Collectors.Disk.MountPoints, logger))
	}

	// 网络采集器
	if cfg.Collectors.Network.Enabled {
		collectors = append(collectors,
			collector.NewNetworkCollector(true, cfg.Collectors.Network.Interval, cfg.Collectors.Network.Interfaces, logger))
	}

	return collectors
}

// getLocalIP 获取本地IP
func (d *Daemon) getLocalIP() string {
	// TODO: 实现获取本地IP逻辑
	return "127.0.0.1"
}

// getOS 获取操作系统
func (d *Daemon) getOS() string {
	// TODO: 实现获取OS逻辑
	return "linux"
}

// getArch 获取架构
func (d *Daemon) getArch() string {
	// TODO: 实现获取架构逻辑
	return "amd64"
}

// startHTTPServer 启动HTTP服务器(用于接收Agent心跳)
func (d *Daemon) startHTTPServer() error {
	// 创建HTTP路由
	mux := http.NewServeMux()

	// 注册心跳路由
	mux.HandleFunc("/heartbeat", d.httpHeartbeatReceiver.HandleHeartbeat)

	// 注册统计信息路由(可选)
	mux.HandleFunc("/heartbeat/stats", d.httpHeartbeatReceiver.HandleStats)

	// 创建HTTP服务器(默认监听8081端口)
	// TODO: 可以从配置中读取端口
	addr := ":8081"
	d.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// 在goroutine中启动服务器
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.logger.Info("HTTP server starting", zap.String("addr", addr))
		if err := d.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	return nil
}

// loadOrGenerateNodeID 加载或生成节点ID
// 优先级：配置文件 > 持久化文件 > 新生成
func loadOrGenerateNodeID(cfg *config.Config, logger *zap.Logger) (string, error) {
	// 1. 优先使用配置文件中的ID
	if cfg.Daemon.ID != "" {
		logger.Info("using node ID from config", zap.String("node_id", cfg.Daemon.ID))
		return cfg.Daemon.ID, nil
	}

	// 2. 尝试从持久化文件读取
	nodeIDFile := fmt.Sprintf("%s/node_id", cfg.Daemon.WorkDir)
	if data, err := os.ReadFile(nodeIDFile); err == nil {
		nodeID := string(data)
		if nodeID != "" {
			logger.Info("loaded node ID from file", zap.String("node_id", nodeID), zap.String("file", nodeIDFile))
			return nodeID, nil
		}
	}

	// 3. 生成新的节点ID并持久化
	nodeID := uuid.New().String()

	// 确保工作目录存在
	if err := os.MkdirAll(cfg.Daemon.WorkDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create work directory: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(nodeIDFile, []byte(nodeID), 0644); err != nil {
		return "", fmt.Errorf("failed to save node ID to file: %w", err)
	}

	logger.Info("generated new node ID and saved to file", zap.String("node_id", nodeID), zap.String("file", nodeIDFile))
	return nodeID, nil
}
