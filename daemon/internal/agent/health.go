package agent

import (
	"context"
	"sync"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
)

// HealthChecker Agent健康检查器
type HealthChecker struct {
	config        *config.HealthCheckConfig
	manager       *Manager
	heartbeatCh   chan *types.Heartbeat
	lastHeartbeat time.Time
	mu            sync.RWMutex
	logger        *zap.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(cfg *config.HealthCheckConfig, manager *Manager, logger *zap.Logger) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		config:      cfg,
		manager:     manager,
		heartbeatCh: make(chan *types.Heartbeat, 1000), // 增加缓冲区大小以处理突发心跳
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动健康检查
func (h *HealthChecker) Start() {
	h.logger.Info("starting health checker",
		zap.Duration("interval", h.config.Interval))

	h.wg.Add(1)
	go h.checkLoop()
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	h.logger.Info("stopping health checker")
	h.cancel()
	h.wg.Wait()
	close(h.heartbeatCh)
	h.logger.Info("health checker stopped")
}

// ReceiveHeartbeat 接收心跳
func (h *HealthChecker) ReceiveHeartbeat(hb *types.Heartbeat) {
	select {
	case h.heartbeatCh <- hb:
	default:
		h.logger.Warn("heartbeat channel full, dropping heartbeat")
	}
}

// checkLoop 健康检查循环
func (h *HealthChecker) checkLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	var overThresholdSince time.Time

	for {
		select {
		case <-h.ctx.Done():
			return

		case hb := <-h.heartbeatCh:
			// 批量处理所有待处理的心跳，只保留最新的时间戳
			latestHB := h.processBatchedHeartbeats(hb)
			h.processHeartbeat(latestHB)

		case <-ticker.C:
			status := h.checkHealth()

			switch status {
			case types.HealthStatusDead:
				h.logger.Warn("agent process not running, restarting")
				if err := h.manager.Restart(h.ctx); err != nil {
					h.logger.Error("failed to restart agent", zap.Error(err))
				}

			case types.HealthStatusNoHeartbeat:
				h.logger.Warn("agent heartbeat timeout, restarting",
					zap.Time("last_heartbeat", h.getLastHeartbeat()))
				if err := h.manager.Restart(h.ctx); err != nil {
					h.logger.Error("failed to restart agent", zap.Error(err))
				}

			case types.HealthStatusOverThreshold:
				if overThresholdSince.IsZero() {
					overThresholdSince = time.Now()
					h.logger.Warn("agent resource over threshold",
						zap.Time("since", overThresholdSince))
				} else if time.Since(overThresholdSince) > h.config.ThresholdDuration {
					h.logger.Warn("agent resource over threshold for too long, restarting",
						zap.Duration("duration", time.Since(overThresholdSince)))
					if err := h.manager.Restart(h.ctx); err != nil {
						h.logger.Error("failed to restart agent", zap.Error(err))
					}
					overThresholdSince = time.Time{}
				}

			case types.HealthStatusHealthy:
				// 重置超限计时
				if !overThresholdSince.IsZero() {
					h.logger.Info("agent resource back to normal")
					overThresholdSince = time.Time{}
				}
				// 重启计数重置(运行正常超过5分钟)
				if time.Since(h.manager.lastRestart) > 5*time.Minute {
					if h.manager.GetRestartCount() > 0 {
						h.logger.Info("resetting restart count",
							zap.Int("previous_count", h.manager.GetRestartCount()))
						h.manager.ResetRestartCount()
					}
				}
			}
		}
	}
}

// processBatchedHeartbeats 批量处理待处理的心跳，返回最新的心跳
// 由于我们只关心最新的心跳时间戳，所以可以丢弃旧的心跳
func (h *HealthChecker) processBatchedHeartbeats(firstHB *types.Heartbeat) *types.Heartbeat {
	latestHB := firstHB
	processed := 1

	// 非阻塞地处理所有待处理的心跳
	for {
		select {
		case nextHB := <-h.heartbeatCh:
			processed++
			// 只保留最新的心跳时间戳
			if nextHB.Timestamp.After(latestHB.Timestamp) {
				latestHB = nextHB
			}
		default:
			// 没有更多待处理的心跳
			if processed > 1 {
				h.logger.Debug("processed batched heartbeats",
					zap.Int("count", processed),
					zap.Time("latest_timestamp", latestHB.Timestamp))
			}
			return latestHB
		}
	}
}

// processHeartbeat 处理心跳
func (h *HealthChecker) processHeartbeat(hb *types.Heartbeat) {
	h.mu.Lock()
	h.lastHeartbeat = hb.Timestamp
	h.mu.Unlock()

	h.logger.Debug("received heartbeat",
		zap.Int("pid", hb.PID),
		zap.Float64("cpu", hb.CPU),
		zap.Uint64("memory", hb.Memory))
}

// checkHealth 检查健康状态
func (h *HealthChecker) checkHealth() types.HealthStatus {
	// 1. 检查进程是否存在
	if !h.manager.IsRunning() {
		return types.HealthStatusDead
	}

	// 2. 检查心跳
	lastHB := h.getLastHeartbeat()
	if !lastHB.IsZero() && time.Since(lastHB) > h.config.HeartbeatTimeout {
		return types.HealthStatusNoHeartbeat
	}

	// 3. 检查资源占用
	pid := h.manager.GetPID()
	if pid > 0 {
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			h.logger.Error("failed to get process", zap.Int("pid", pid), zap.Error(err))
			return types.HealthStatusDead
		}

		cpuPercent, err := proc.CPUPercent()
		if err != nil {
			h.logger.Warn("failed to get cpu percent", zap.Error(err))
		}

		memInfo, err := proc.MemoryInfo()
		if err != nil {
			h.logger.Warn("failed to get memory info", zap.Error(err))
		}

		if cpuPercent > h.config.CPUThreshold || (memInfo != nil && memInfo.RSS > h.config.MemoryThreshold) {
			h.logger.Warn("agent resource over threshold",
				zap.Float64("cpu_percent", cpuPercent),
				zap.Float64("cpu_threshold", h.config.CPUThreshold),
				zap.Uint64("memory_rss", memInfo.RSS),
				zap.Uint64("memory_threshold", h.config.MemoryThreshold))
			return types.HealthStatusOverThreshold
		}
	}

	return types.HealthStatusHealthy
}

// getLastHeartbeat 获取最后心跳时间
func (h *HealthChecker) getLastHeartbeat() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastHeartbeat
}
