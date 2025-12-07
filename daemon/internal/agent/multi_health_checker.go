package agent

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
)

// MultiHealthChecker 多Agent健康检查器
// 支持监控多个Agent实例的健康状态，为每个Agent创建独立的健康检查goroutine
type MultiHealthChecker struct {
	// multiAgentManager 多Agent管理器
	multiAgentManager *MultiAgentManager

	// agentConfigs 每个Agent的健康检查配置
	// key: Agent ID
	// value: HealthCheckConfig
	agentConfigs map[string]*config.HealthCheckConfig

	// healthStatuses 每个Agent的健康状态
	// key: Agent ID
	// value: AgentHealthStatus
	healthStatuses map[string]*AgentHealthStatus

	// heartbeats 每个Agent的最后心跳时间
	// key: Agent ID
	// value: 最后心跳时间
	heartbeats map[string]time.Time

	// mu 保护healthStatuses和heartbeats的并发访问锁
	mu sync.RWMutex

	// logger 日志记录器
	logger *zap.Logger

	// ctx 上下文
	ctx context.Context

	// cancel 取消函数
	cancel context.CancelFunc

	// wg 等待组
	wg sync.WaitGroup
}

// AgentHealthStatus Agent健康状态
type AgentHealthStatus struct {
	AgentID            string
	Status             types.HealthStatus
	LastCheck          time.Time
	LastHeartbeat      time.Time
	OverThresholdSince time.Time
	CPUPercent         float64
	MemoryRSS          uint64
	mu                 sync.RWMutex
}

// MultiHealthCheckerConfig 多Agent健康检查器配置
type MultiHealthCheckerConfig struct {
	// AgentConfigs 每个Agent的健康检查配置
	// key: Agent ID
	// value: HealthCheckConfig
	AgentConfigs map[string]*config.HealthCheckConfig
}

// NewMultiHealthChecker 创建新的多Agent健康检查器
func NewMultiHealthChecker(multiAgentManager *MultiAgentManager, cfg *MultiHealthCheckerConfig, logger *zap.Logger) *MultiHealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	agentConfigs := make(map[string]*config.HealthCheckConfig)
	if cfg != nil && cfg.AgentConfigs != nil {
		agentConfigs = cfg.AgentConfigs
	}

	return &MultiHealthChecker{
		multiAgentManager: multiAgentManager,
		agentConfigs:      agentConfigs,
		healthStatuses:    make(map[string]*AgentHealthStatus),
		heartbeats:        make(map[string]time.Time),
		logger:            logger,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// RegisterAgent 注册Agent的健康检查配置
func (mhc *MultiHealthChecker) RegisterAgent(agentID string, healthCheckCfg *config.HealthCheckConfig) {
	mhc.mu.Lock()
	defer mhc.mu.Unlock()

	mhc.agentConfigs[agentID] = healthCheckCfg
	mhc.healthStatuses[agentID] = &AgentHealthStatus{
		AgentID: agentID,
		Status:  types.HealthStatusHealthy,
	}
	mhc.heartbeats[agentID] = time.Time{}

	mhc.logger.Info("agent health check registered",
		zap.String("agent_id", agentID),
		zap.Duration("interval", healthCheckCfg.Interval))
}

// Start 启动健康检查
func (mhc *MultiHealthChecker) Start() {
	mhc.logger.Info("starting multi-agent health checker")

	// 为每个已注册的Agent启动健康检查goroutine
	agents := mhc.multiAgentManager.ListAgents()
	for _, instance := range agents {
		info := instance.GetInfo()
		agentID := info.ID

		// 获取Agent的健康检查配置（从配置中获取，如果没有则使用默认值）
		healthCheckCfg := mhc.getAgentHealthCheckConfig(agentID)
		if healthCheckCfg == nil {
			// 使用默认配置
			healthCheckCfg = &config.HealthCheckConfig{
				Interval:          30 * time.Second,
				HeartbeatTimeout:  90 * time.Second,
				CPUThreshold:      50.0,
				MemoryThreshold:   524288000,
				ThresholdDuration: 60 * time.Second,
			}
		}

		mhc.RegisterAgent(agentID, healthCheckCfg)

		// 为每个Agent启动独立的健康检查goroutine
		mhc.wg.Add(1)
		go mhc.checkAgentHealth(agentID, healthCheckCfg)
	}

	mhc.logger.Info("multi-agent health checker started",
		zap.Int("agent_count", len(agents)))
}

// Stop 停止健康检查
func (mhc *MultiHealthChecker) Stop() {
	mhc.logger.Info("stopping multi-agent health checker")
	mhc.cancel()
	mhc.wg.Wait()
	mhc.logger.Info("multi-agent health checker stopped")
}

// ReceiveHeartbeat 接收Agent心跳
func (mhc *MultiHealthChecker) ReceiveHeartbeat(agentID string, hb *types.Heartbeat) {
	mhc.mu.Lock()
	defer mhc.mu.Unlock()

	mhc.heartbeats[agentID] = hb.Timestamp

	// 更新健康状态中的心跳时间
	if status, exists := mhc.healthStatuses[agentID]; exists {
		status.mu.Lock()
		status.LastHeartbeat = hb.Timestamp
		status.mu.Unlock()
	}

	mhc.logger.Debug("received heartbeat",
		zap.String("agent_id", agentID),
		zap.Int("pid", hb.PID),
		zap.Float64("cpu", hb.CPU),
		zap.Uint64("memory", hb.Memory))
}

// GetHealthStatus 获取指定Agent的健康状态
func (mhc *MultiHealthChecker) GetHealthStatus(agentID string) *AgentHealthStatus {
	mhc.mu.RLock()
	defer mhc.mu.RUnlock()
	return mhc.healthStatuses[agentID]
}

// GetAllHealthStatuses 获取所有Agent的健康状态
func (mhc *MultiHealthChecker) GetAllHealthStatuses() map[string]*AgentHealthStatus {
	mhc.mu.RLock()
	defer mhc.mu.RUnlock()

	result := make(map[string]*AgentHealthStatus)
	for agentID, status := range mhc.healthStatuses {
		result[agentID] = status
	}
	return result
}

// checkAgentHealth 检查单个Agent的健康状态（独立goroutine）
func (mhc *MultiHealthChecker) checkAgentHealth(agentID string, healthCheckCfg *config.HealthCheckConfig) {
	defer mhc.wg.Done()

	ticker := time.NewTicker(healthCheckCfg.Interval)
	defer ticker.Stop()

	var overThresholdSince time.Time

	mhc.logger.Info("started health check goroutine",
		zap.String("agent_id", agentID),
		zap.Duration("interval", healthCheckCfg.Interval))

	for {
		select {
		case <-mhc.ctx.Done():
			return

		case <-ticker.C:
			status := mhc.checkHealth(agentID, healthCheckCfg)

			// 更新健康状态
			mhc.updateHealthStatus(agentID, status)

			switch status {
			case types.HealthStatusDead:
				mhc.logger.Warn("agent process not running, restarting",
					zap.String("agent_id", agentID))
				if err := mhc.multiAgentManager.RestartAgent(mhc.ctx, agentID); err != nil {
					mhc.logger.Error("failed to restart agent",
						zap.String("agent_id", agentID),
						zap.Error(err))
				}

			case types.HealthStatusNoHeartbeat:
				lastHB := mhc.getLastHeartbeat(agentID)
				mhc.logger.Warn("agent heartbeat timeout, restarting",
					zap.String("agent_id", agentID),
					zap.Time("last_heartbeat", lastHB))
				if err := mhc.multiAgentManager.RestartAgent(mhc.ctx, agentID); err != nil {
					mhc.logger.Error("failed to restart agent",
						zap.String("agent_id", agentID),
						zap.Error(err))
				}

			case types.HealthStatusOverThreshold:
				if overThresholdSince.IsZero() {
					overThresholdSince = time.Now()
					mhc.logger.Warn("agent resource over threshold",
						zap.String("agent_id", agentID),
						zap.Time("since", overThresholdSince))
				} else if time.Since(overThresholdSince) > healthCheckCfg.ThresholdDuration {
					mhc.logger.Warn("agent resource over threshold for too long, restarting",
						zap.String("agent_id", agentID),
						zap.Duration("duration", time.Since(overThresholdSince)))
					if err := mhc.multiAgentManager.RestartAgent(mhc.ctx, agentID); err != nil {
						mhc.logger.Error("failed to restart agent",
							zap.String("agent_id", agentID),
							zap.Error(err))
					}
					overThresholdSince = time.Time{}
				}

			case types.HealthStatusHealthy:
				// 重置超限计时
				if !overThresholdSince.IsZero() {
					mhc.logger.Info("agent resource back to normal",
						zap.String("agent_id", agentID))
					overThresholdSince = time.Time{}
				}

				// 重置重启计数（运行正常超过5分钟）
				instance := mhc.multiAgentManager.GetAgent(agentID)
				if instance != nil {
					info := instance.GetInfo()
					lastRestart := info.GetLastRestart()
					if !lastRestart.IsZero() && time.Since(lastRestart) > 5*time.Minute {
						restartCount := info.GetRestartCount()
						if restartCount > 0 {
							mhc.logger.Info("resetting restart count",
								zap.String("agent_id", agentID),
								zap.Int("previous_count", restartCount))
							info.ResetRestartCount()
						}
					}
				}
			}
		}
	}
}

// checkHealth 检查Agent健康状态
func (mhc *MultiHealthChecker) checkHealth(agentID string, healthCheckCfg *config.HealthCheckConfig) types.HealthStatus {
	instance := mhc.multiAgentManager.GetAgent(agentID)
	if instance == nil {
		return types.HealthStatusDead
	}

	// 1. 检查进程是否存在
	if !instance.IsRunning() {
		return types.HealthStatusDead
	}

	// 2. 检查心跳（如果配置了心跳超时）
	if healthCheckCfg.HeartbeatTimeout > 0 {
		lastHB := mhc.getLastHeartbeat(agentID)
		if !lastHB.IsZero() && time.Since(lastHB) > healthCheckCfg.HeartbeatTimeout {
			return types.HealthStatusNoHeartbeat
		}
	}

	// 3. 根据Agent类型选择健康检查策略
	info := instance.GetInfo()
	status := mhc.checkHealthByType(agentID, info, healthCheckCfg, instance)

	// 4. 更新健康状态中的资源信息
	if status == types.HealthStatusOverThreshold || status == types.HealthStatusHealthy {
		mhc.updateResourceInfo(agentID, instance)
	}

	return status
}

// checkHealthByType 根据Agent类型检查健康状态
func (mhc *MultiHealthChecker) checkHealthByType(
	agentID string,
	info *AgentInfo,
	healthCheckCfg *config.HealthCheckConfig,
	instance *AgentInstance,
) types.HealthStatus {
	switch info.Type {
	case TypeNodeExporter:
		// Node Exporter: 使用HTTP端点检查
		return mhc.checkHTTPHealth(agentID, healthCheckCfg)
	default:
		// 其他类型: 使用进程和资源检查
		return mhc.checkProcessHealth(agentID, instance, healthCheckCfg)
	}
}

// checkProcessHealth 检查进程健康状态（进程检查 + 资源检查）
func (mhc *MultiHealthChecker) checkProcessHealth(
	agentID string,
	instance *AgentInstance,
	healthCheckCfg *config.HealthCheckConfig,
) types.HealthStatus {
	pid := instance.GetPID()
	if pid <= 0 {
		return types.HealthStatusDead
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		mhc.logger.Error("failed to get process",
			zap.String("agent_id", agentID),
			zap.Int("pid", pid),
			zap.Error(err))
		return types.HealthStatusDead
	}

	// 检查CPU和内存使用
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		mhc.logger.Warn("failed to get cpu percent",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}

	memInfo, err := proc.MemoryInfo()
	if err != nil {
		mhc.logger.Warn("failed to get memory info",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}

	// 更新资源信息
	mhc.mu.Lock()
	if status, exists := mhc.healthStatuses[agentID]; exists {
		status.mu.Lock()
		status.CPUPercent = cpuPercent
		if memInfo != nil {
			status.MemoryRSS = memInfo.RSS
		}
		status.mu.Unlock()
	}
	mhc.mu.Unlock()

	// 检查是否超过阈值
	if cpuPercent > healthCheckCfg.CPUThreshold || (memInfo != nil && memInfo.RSS > healthCheckCfg.MemoryThreshold) {
		mhc.logger.Warn("agent resource over threshold",
			zap.String("agent_id", agentID),
			zap.Float64("cpu_percent", cpuPercent),
			zap.Float64("cpu_threshold", healthCheckCfg.CPUThreshold),
			zap.Uint64("memory_rss", memInfo.RSS),
			zap.Uint64("memory_threshold", healthCheckCfg.MemoryThreshold))
		return types.HealthStatusOverThreshold
	}

	return types.HealthStatusHealthy
}

// checkHTTPHealth 检查HTTP健康状态（用于Node Exporter等提供HTTP端点的Agent）
func (mhc *MultiHealthChecker) checkHTTPHealth(
	agentID string,
	healthCheckCfg *config.HealthCheckConfig,
) types.HealthStatus {
	// 注意：HTTP端点检查需要从配置中获取endpoint
	// 这里暂时使用默认的/metrics端点
	// 实际实现中应该从Agent配置中获取http_endpoint

	// 默认检查逻辑：如果配置了HTTP端点，检查端点
	// 否则使用进程检查
	// 这里简化实现，使用进程检查
	instance := mhc.multiAgentManager.GetAgent(agentID)
	if instance == nil {
		return types.HealthStatusDead
	}

	return mhc.checkProcessHealth(agentID, instance, healthCheckCfg)
}

// updateHealthStatus 更新健康状态
func (mhc *MultiHealthChecker) updateHealthStatus(agentID string, status types.HealthStatus) {
	mhc.mu.Lock()
	defer mhc.mu.Unlock()

	if healthStatus, exists := mhc.healthStatuses[agentID]; exists {
		healthStatus.mu.Lock()
		healthStatus.Status = status
		healthStatus.LastCheck = time.Now()
		if status == types.HealthStatusOverThreshold && healthStatus.OverThresholdSince.IsZero() {
			healthStatus.OverThresholdSince = time.Now()
		} else if status != types.HealthStatusOverThreshold {
			healthStatus.OverThresholdSince = time.Time{}
		}
		healthStatus.mu.Unlock()
	}
}

// updateResourceInfo 更新资源信息
func (mhc *MultiHealthChecker) updateResourceInfo(agentID string, instance *AgentInstance) {
	pid := instance.GetPID()
	if pid <= 0 {
		return
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return
	}

	cpuPercent, _ := proc.CPUPercent()
	memInfo, _ := proc.MemoryInfo()

	mhc.mu.Lock()
	if status, exists := mhc.healthStatuses[agentID]; exists {
		status.mu.Lock()
		status.CPUPercent = cpuPercent
		if memInfo != nil {
			status.MemoryRSS = memInfo.RSS
		}
		status.mu.Unlock()
	}
	mhc.mu.Unlock()
}

// GetLastHeartbeat 获取最后心跳时间（公开方法，用于测试和查询）
func (mhc *MultiHealthChecker) GetLastHeartbeat(agentID string) time.Time {
	mhc.mu.RLock()
	defer mhc.mu.RUnlock()
	return mhc.heartbeats[agentID]
}

// getLastHeartbeat 获取最后心跳时间（内部方法）
func (mhc *MultiHealthChecker) getLastHeartbeat(agentID string) time.Time {
	return mhc.GetLastHeartbeat(agentID)
}

// getAgentHealthCheckConfig 获取Agent的健康检查配置
func (mhc *MultiHealthChecker) getAgentHealthCheckConfig(agentID string) *config.HealthCheckConfig {
	mhc.mu.RLock()
	defer mhc.mu.RUnlock()
	return mhc.agentConfigs[agentID]
}

// HTTPHealthCheck 执行HTTP健康检查
func HTTPHealthCheck(endpoint string, timeout time.Duration) bool {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
