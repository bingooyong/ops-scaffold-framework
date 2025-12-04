package daemon

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/agent"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/collector"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/comm"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Daemon 守护进程
type Daemon struct {
	config            *config.Config
	logger            *zap.Logger
	nodeID            string
	collectorManager  *collector.Manager
	agentManager      *agent.Manager
	healthChecker     *agent.HealthChecker
	heartbeatReceiver *agent.HeartbeatReceiver
	grpcClient        *comm.GRPCClient
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

// New 创建Daemon实例
func New(cfg *config.Config, logger *zap.Logger) (*Daemon, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 生成或加载节点ID
	nodeID := cfg.Daemon.ID
	if nodeID == "" {
		nodeID = uuid.New().String()
	}

	// 创建采集器
	collectors := createCollectors(cfg, logger)
	collectorMgr := collector.NewManager(collectors, logger)

	// 创建Agent管理器
	agentMgr := agent.NewManager(&cfg.Agent, logger)

	// 创建健康检查器
	healthChecker := agent.NewHealthChecker(&cfg.Agent.HealthCheck, agentMgr, logger)

	// 创建心跳接收器
	heartbeatReceiver := agent.NewHeartbeatReceiver(cfg.Agent.SocketPath, healthChecker, logger)

	// 创建gRPC客户端
	grpcClient := comm.NewGRPCClient(&cfg.Manager, logger)

	d := &Daemon{
		config:            cfg,
		logger:            logger,
		nodeID:            nodeID,
		collectorManager:  collectorMgr,
		agentManager:      agentMgr,
		healthChecker:     healthChecker,
		heartbeatReceiver: heartbeatReceiver,
		grpcClient:        grpcClient,
		ctx:               ctx,
		cancel:            cancel,
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

	// 3. 启动Agent进程（如果配置了Agent）
	if d.config.Agent.BinaryPath != "" {
		if err := d.agentManager.Start(d.ctx); err != nil {
			d.logger.Error("failed to start agent", zap.Error(err))
			// 不中断启动，继续运行采集器
		} else {
			// 4. 启动心跳接收器
			if err := d.heartbeatReceiver.Start(); err != nil {
				d.logger.Error("failed to start heartbeat receiver", zap.Error(err))
			}

			// 5. 启动健康检查
			d.healthChecker.Start()
		}
	} else {
		d.logger.Info("agent management disabled (no binary path configured)")
	}

	// 6. 启动采集器
	d.collectorManager.Start()

	// 7. 连接Manager并注册（如果配置了Manager地址）
	if d.config.Manager.Address != "" {
		if err := d.connectAndRegister(); err != nil {
			d.logger.Error("failed to connect to manager", zap.Error(err))
			// 不中断启动，后台会重试连接
		}
	} else {
		d.logger.Info("manager connection disabled (no address configured)")
	}

	// 8. 启动后台任务（如果配置了Manager）
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
	d.heartbeatReceiver.Stop()
	d.healthChecker.Stop()
	d.collectorManager.Stop()

	// 注意：不停止Agent进程，让它继续运行
	d.logger.Info("agent will continue running after daemon stops")

	// 关闭gRPC连接
	if err := d.grpcClient.Close(); err != nil {
		d.logger.Error("failed to close grpc client", zap.Error(err))
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
