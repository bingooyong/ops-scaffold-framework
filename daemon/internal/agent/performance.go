package agent

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	// 心跳处理指标
	HeartbeatProcessed  atomic.Int64 // 已处理心跳数
	HeartbeatDropped    atomic.Int64 // 丢弃心跳数
	HeartbeatLatency    atomic.Int64 // 总延迟（纳秒）
	HeartbeatMaxLatency atomic.Int64 // 最大延迟（纳秒）

	// Agent 操作指标
	AgentOpsTotal   atomic.Int64 // 总操作数
	AgentOpsSuccess atomic.Int64 // 成功操作数
	AgentOpsFailed  atomic.Int64 // 失败操作数
	AgentOpsLatency atomic.Int64 // 总延迟（纳秒）

	// 状态同步指标
	StateSyncTotal   atomic.Int64 // 总同步次数
	StateSyncSuccess atomic.Int64 // 成功同步次数
	StateSyncFailed  atomic.Int64 // 失败同步次数
	StateSyncLatency atomic.Int64 // 总延迟（纳秒）

	// 资源监控指标
	ResourceCheckTotal   atomic.Int64 // 总检查次数
	ResourceCheckLatency atomic.Int64 // 总延迟（纳秒）

	mu sync.RWMutex
}

// NewPerformanceMetrics 创建性能指标收集器
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{}
}

// RecordHeartbeat 记录心跳处理
func (pm *PerformanceMetrics) RecordHeartbeat(latency time.Duration, dropped bool) {
	if dropped {
		pm.HeartbeatDropped.Add(1)
		return
	}

	pm.HeartbeatProcessed.Add(1)
	latencyNs := latency.Nanoseconds()
	pm.HeartbeatLatency.Add(latencyNs)

	// 更新最大延迟
	for {
		currentMax := pm.HeartbeatMaxLatency.Load()
		if latencyNs <= currentMax {
			break
		}
		if pm.HeartbeatMaxLatency.CompareAndSwap(currentMax, latencyNs) {
			break
		}
	}
}

// RecordAgentOp 记录 Agent 操作
func (pm *PerformanceMetrics) RecordAgentOp(success bool, latency time.Duration) {
	pm.AgentOpsTotal.Add(1)
	if success {
		pm.AgentOpsSuccess.Add(1)
	} else {
		pm.AgentOpsFailed.Add(1)
	}
	pm.AgentOpsLatency.Add(latency.Nanoseconds())
}

// RecordStateSync 记录状态同步
func (pm *PerformanceMetrics) RecordStateSync(success bool, latency time.Duration) {
	pm.StateSyncTotal.Add(1)
	if success {
		pm.StateSyncSuccess.Add(1)
	} else {
		pm.StateSyncFailed.Add(1)
	}
	pm.StateSyncLatency.Add(latency.Nanoseconds())
}

// RecordResourceCheck 记录资源检查
func (pm *PerformanceMetrics) RecordResourceCheck(latency time.Duration) {
	pm.ResourceCheckTotal.Add(1)
	pm.ResourceCheckLatency.Add(latency.Nanoseconds())
}

// GetStats 获取统计信息
func (pm *PerformanceMetrics) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	heartbeatProcessed := pm.HeartbeatProcessed.Load()
	heartbeatDropped := pm.HeartbeatDropped.Load()
	heartbeatLatency := pm.HeartbeatLatency.Load()
	heartbeatMaxLatency := pm.HeartbeatMaxLatency.Load()

	var avgHeartbeatLatency time.Duration
	if heartbeatProcessed > 0 {
		avgHeartbeatLatency = time.Duration(heartbeatLatency / heartbeatProcessed)
	}

	agentOpsTotal := pm.AgentOpsTotal.Load()
	agentOpsSuccess := pm.AgentOpsSuccess.Load()
	agentOpsFailed := pm.AgentOpsFailed.Load()
	agentOpsLatency := pm.AgentOpsLatency.Load()

	var avgAgentOpsLatency time.Duration
	if agentOpsTotal > 0 {
		avgAgentOpsLatency = time.Duration(agentOpsLatency / agentOpsTotal)
	}

	stateSyncTotal := pm.StateSyncTotal.Load()
	stateSyncSuccess := pm.StateSyncSuccess.Load()
	stateSyncFailed := pm.StateSyncFailed.Load()
	stateSyncLatency := pm.StateSyncLatency.Load()

	var avgStateSyncLatency time.Duration
	if stateSyncTotal > 0 {
		avgStateSyncLatency = time.Duration(stateSyncLatency / stateSyncTotal)
	}

	resourceCheckTotal := pm.ResourceCheckTotal.Load()
	resourceCheckLatency := pm.ResourceCheckLatency.Load()

	var avgResourceCheckLatency time.Duration
	if resourceCheckTotal > 0 {
		avgResourceCheckLatency = time.Duration(resourceCheckLatency / resourceCheckTotal)
	}

	return map[string]interface{}{
		"heartbeat": map[string]interface{}{
			"processed":      heartbeatProcessed,
			"dropped":        heartbeatDropped,
			"avg_latency_ns": avgHeartbeatLatency.Nanoseconds(),
			"max_latency_ns": heartbeatMaxLatency,
		},
		"agent_ops": map[string]interface{}{
			"total":          agentOpsTotal,
			"success":        agentOpsSuccess,
			"failed":         agentOpsFailed,
			"avg_latency_ns": avgAgentOpsLatency.Nanoseconds(),
		},
		"state_sync": map[string]interface{}{
			"total":          stateSyncTotal,
			"success":        stateSyncSuccess,
			"failed":         stateSyncFailed,
			"avg_latency_ns": avgStateSyncLatency.Nanoseconds(),
		},
		"resource_check": map[string]interface{}{
			"total":          resourceCheckTotal,
			"avg_latency_ns": avgResourceCheckLatency.Nanoseconds(),
		},
	}
}

// LogStats 记录统计信息到日志
func (pm *PerformanceMetrics) LogStats(logger *zap.Logger) {
	stats := pm.GetStats()
	logger.Info("performance metrics",
		zap.Any("stats", stats))
}
