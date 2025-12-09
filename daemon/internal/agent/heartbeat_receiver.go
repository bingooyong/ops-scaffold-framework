package agent

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Heartbeat Agent心跳数据结构体
type Heartbeat struct {
	AgentID   string    // Agent唯一标识符
	PID       int       // Agent进程ID
	Status    string    // Agent运行状态(running/stopped/error)
	CPU       float64   // CPU使用率(百分比)
	Memory    uint64    // 内存占用(字节)
	Timestamp time.Time // 心跳时间戳
}

// HeartbeatRequest HTTP请求结构体(用于JSON解析)
type HeartbeatRequest struct {
	AgentID   string  `json:"agent_id"`  // Agent唯一标识符
	PID       int     `json:"pid"`       // Agent进程ID
	Status    string  `json:"status"`    // Agent运行状态
	CPU       float64 `json:"cpu"`       // CPU使用率(百分比)
	Memory    uint64  `json:"memory"`    // 内存占用(字节)
	Timestamp string  `json:"timestamp"` // 心跳时间戳(JSON时间格式,可选)
}

// HeartbeatResponse HTTP响应结构体
type HeartbeatResponse struct {
	Success   bool      `json:"success"`   // 处理是否成功
	Message   string    `json:"message"`   // 响应消息
	Timestamp time.Time `json:"timestamp"` // 服务器处理时间
}

// HeartbeatStats 心跳统计信息
type HeartbeatStats struct {
	TotalReceived    int64         `json:"total_received"`     // 总接收数量
	TotalProcessed   int64         `json:"total_processed"`    // 总处理数量
	TotalErrors      int64         `json:"total_errors"`       // 总错误数量
	LastReceivedTime time.Time     `json:"last_received_time"` // 最后接收时间
	AverageLatency   time.Duration `json:"average_latency"`    // 平均处理延迟
}

// HTTPHeartbeatReceiver HTTP心跳接收器
// 提供HTTP endpoint接收Agent心跳上报,使用worker pool并发处理
// 用于多Agent场景,与Unix socket版本的HeartbeatReceiver不同
type HTTPHeartbeatReceiver struct {
	// multiManager 多Agent管理器引用(用于更新心跳)
	multiManager *MultiAgentManager

	// registry Agent注册表引用(用于验证agent_id)
	registry *AgentRegistry

	// logger 日志记录器
	logger *zap.Logger

	// workerPool 工作池channel(用于并发处理)
	workerPool chan *Heartbeat

	// workerCount 工作协程数量
	workerCount int

	// mu 保护统计数据的并发访问
	mu sync.RWMutex

	// stats 统计信息
	stats HeartbeatStats

	// totalLatency 总延迟时间(用于计算平均延迟)
	totalLatency atomic.Int64

	// wg 等待所有worker goroutine退出
	wg sync.WaitGroup

	// stopCh 停止信号channel
	stopCh chan struct{}

	// stopped 是否已停止
	stopped atomic.Bool
}

// NewHTTPHeartbeatReceiver 创建新的HTTP心跳接收器
func NewHTTPHeartbeatReceiver(
	multiManager *MultiAgentManager,
	registry *AgentRegistry,
	logger *zap.Logger,
) *HTTPHeartbeatReceiver {
	// 从环境变量或配置获取worker数量(默认10)
	workerCount := 10
	// TODO: 可以从配置中读取worker数量

	hr := &HTTPHeartbeatReceiver{
		multiManager: multiManager,
		registry:     registry,
		logger:       logger,
		workerPool:   make(chan *Heartbeat, workerCount*100), // channel容量为worker数量的100倍,防止心跳丢失
		workerCount:  workerCount,
		stopCh:       make(chan struct{}),
	}

	// 启动worker goroutines
	hr.startWorkers()

	logger.Info("heartbeat receiver created",
		zap.Int("worker_count", workerCount),
		zap.Int("channel_capacity", cap(hr.workerPool)))

	return hr
}

// HandleHeartbeat HTTP handler处理心跳请求
func (hr *HTTPHeartbeatReceiver) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	// 检查HTTP方法
	if r.Method != http.MethodPost {
		hr.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed, expected POST")
		return
	}

	// 检查是否已停止
	if hr.stopped.Load() {
		hr.writeErrorResponse(w, http.StatusServiceUnavailable, "heartbeat receiver is stopped")
		return
	}

	// 解析JSON请求体
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hr.logger.Warn("failed to decode heartbeat request",
			zap.Error(err))
		hr.writeErrorResponse(w, http.StatusBadRequest, "invalid JSON format: "+err.Error())
		return
	}

	// 验证请求数据
	if err := hr.validateRequest(&req); err != nil {
		hr.logger.Warn("invalid heartbeat request",
			zap.String("agent_id", req.AgentID),
			zap.Error(err))
		hr.writeErrorResponse(w, http.StatusBadRequest, "validation failed: "+err.Error())
		return
	}

	// 转换为Heartbeat结构体
	hb := hr.convertToHeartbeat(&req)

	// 更新统计信息(接收)
	hr.mu.Lock()
	hr.stats.TotalReceived++
	hr.stats.LastReceivedTime = time.Now()
	hr.mu.Unlock()

	// 发送到工作池channel(非阻塞)
	select {
	case hr.workerPool <- hb:
		// 成功发送到channel
		hr.writeSuccessResponse(w, "heartbeat received")
	case <-hr.stopCh:
		// 接收器已停止
		hr.writeErrorResponse(w, http.StatusServiceUnavailable, "heartbeat receiver is stopped")
		return
	default:
		// channel满,记录WARNING并返回503
		hr.logger.Warn("heartbeat worker pool is full, dropping heartbeat",
			zap.String("agent_id", req.AgentID),
			zap.Int("channel_capacity", cap(hr.workerPool)))
		hr.writeErrorResponse(w, http.StatusServiceUnavailable, "worker pool is full, please retry later")
		return
	}
}

// validateRequest 验证心跳请求数据
func (hr *HTTPHeartbeatReceiver) validateRequest(req *HeartbeatRequest) error {
	// 验证agent_id非空
	if req.AgentID == "" {
		return &ValidationError{Field: "agent_id", Message: "agent_id is required"}
	}

	// 验证agent_id在注册表中存在
	if !hr.registry.Exists(req.AgentID) {
		return &ValidationError{Field: "agent_id", Message: "agent_id not found in registry"}
	}

	// 验证PID > 0
	if req.PID <= 0 {
		return &ValidationError{Field: "pid", Message: "pid must be greater than 0"}
	}

	// 验证CPU >= 0 且 <= 100
	if req.CPU < 0 || req.CPU > 100 {
		return &ValidationError{Field: "cpu", Message: "cpu must be between 0 and 100"}
	}

	// 验证Memory >= 0
	if req.Memory < 0 {
		return &ValidationError{Field: "memory", Message: "memory must be greater than or equal to 0"}
	}

	return nil
}

// convertToHeartbeat 将HeartbeatRequest转换为Heartbeat
func (hr *HTTPHeartbeatReceiver) convertToHeartbeat(req *HeartbeatRequest) *Heartbeat {
	hb := &Heartbeat{
		AgentID: req.AgentID,
		PID:     req.PID,
		Status:  req.Status,
		CPU:     req.CPU,
		Memory:  req.Memory,
	}

	// 解析时间戳(如果提供)
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			hb.Timestamp = t
		} else {
			// 解析失败,使用服务器时间
			hr.logger.Warn("failed to parse timestamp, using server time",
				zap.String("agent_id", req.AgentID),
				zap.String("timestamp", req.Timestamp),
				zap.Error(err))
			hb.Timestamp = time.Now()
		}
	} else {
		// 未提供时间戳,使用服务器时间
		hb.Timestamp = time.Now()
	}

	return hb
}

// processHeartbeat 处理单个心跳(在工作协程中调用)
func (hr *HTTPHeartbeatReceiver) processHeartbeat(hb *Heartbeat) {
	startTime := time.Now()

	// 调用multiManager.UpdateHeartbeat更新元数据
	if err := hr.multiManager.UpdateHeartbeat(hb.AgentID, hb.Timestamp, hb.CPU, hb.Memory); err != nil {
		hr.logger.Error("failed to update heartbeat",
			zap.String("agent_id", hb.AgentID),
			zap.Error(err))

		// 更新错误统计
		atomic.AddInt64(&hr.stats.TotalErrors, 1)
		return
	}

	// 计算处理延迟
	latency := time.Since(startTime)

	// 更新统计信息(处理成功)
	atomic.AddInt64(&hr.stats.TotalProcessed, 1)
	hr.totalLatency.Add(latency.Nanoseconds())

	// 更新平均延迟
	hr.mu.Lock()
	processed := hr.stats.TotalProcessed
	if processed > 0 {
		avgLatencyNs := hr.totalLatency.Load() / processed
		hr.stats.AverageLatency = time.Duration(avgLatencyNs)
	}
	hr.mu.Unlock()

	hr.logger.Debug("heartbeat processed",
		zap.String("agent_id", hb.AgentID),
		zap.Duration("latency", latency))
}

// startWorkers 启动worker goroutines
func (hr *HTTPHeartbeatReceiver) startWorkers() {
	for i := 0; i < hr.workerCount; i++ {
		hr.wg.Add(1)
		go hr.worker(i)
	}

	hr.logger.Info("heartbeat receiver workers started",
		zap.Int("worker_count", hr.workerCount))
}

// worker worker goroutine处理心跳
func (hr *HTTPHeartbeatReceiver) worker(id int) {
	defer hr.wg.Done()

	hr.logger.Debug("heartbeat worker started",
		zap.Int("worker_id", id))

	for {
		select {
		case hb, ok := <-hr.workerPool:
			if !ok {
				// channel已关闭,退出
				hr.logger.Debug("heartbeat worker stopped",
					zap.Int("worker_id", id))
				return
			}
			// 处理心跳
			hr.processHeartbeat(hb)

		case <-hr.stopCh:
			// 收到停止信号,退出
			hr.logger.Debug("heartbeat worker received stop signal",
				zap.Int("worker_id", id))
			return
		}
	}
}

// Stop 停止心跳接收器(优雅关闭)
func (hr *HTTPHeartbeatReceiver) Stop() {
	if hr.stopped.Swap(true) {
		// 已经停止
		return
	}

	hr.logger.Info("stopping heartbeat receiver")

	// 关闭stopCh通知所有worker
	close(hr.stopCh)

	// 等待所有正在处理的心跳完成
	// 注意: 不关闭workerPool channel,让worker自然退出
	// 这样可以确保正在处理的心跳能够完成

	// 等待所有worker goroutine退出
	hr.wg.Wait()

	hr.logger.Info("heartbeat receiver stopped")
}

// GetStats 获取统计信息
func (hr *HTTPHeartbeatReceiver) GetStats() HeartbeatStats {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	// 返回统计信息的副本
	return HeartbeatStats{
		TotalReceived:    hr.stats.TotalReceived,
		TotalProcessed:   hr.stats.TotalProcessed,
		TotalErrors:      hr.stats.TotalErrors,
		LastReceivedTime: hr.stats.LastReceivedTime,
		AverageLatency:   hr.stats.AverageLatency,
	}
}

// HandleStats HTTP handler返回统计信息
func (hr *HTTPHeartbeatReceiver) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		hr.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed, expected GET")
		return
	}

	stats := hr.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// writeSuccessResponse 写入成功响应
func (hr *HTTPHeartbeatReceiver) writeSuccessResponse(w http.ResponseWriter, message string) {
	resp := HeartbeatResponse{
		Success:   true,
		Message:   message,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// writeErrorResponse 写入错误响应
func (hr *HTTPHeartbeatReceiver) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	resp := HeartbeatResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
