package agent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
)

// ResourceDataPoint 资源数据点
// 表示某个时间点的Agent进程资源使用情况
type ResourceDataPoint struct {
	// Timestamp 采集时间戳
	Timestamp time.Time `json:"timestamp"`

	// CPU CPU使用率(百分比)
	CPU float64 `json:"cpu"`

	// MemoryRSS 内存占用RSS(字节)
	MemoryRSS uint64 `json:"memory_rss"`

	// MemoryVMS 内存占用VMS(字节)
	MemoryVMS uint64 `json:"memory_vms"`

	// DiskReadBytes 磁盘读取字节数
	DiskReadBytes uint64 `json:"disk_read_bytes"`

	// DiskWriteBytes 磁盘写入字节数
	DiskWriteBytes uint64 `json:"disk_write_bytes"`

	// OpenFiles 打开文件数
	OpenFiles int `json:"open_files"`

	// NumThreads 线程数(可选)
	NumThreads int `json:"num_threads"`
}

// ResourceThreshold 资源阈值配置
// 用于配置Agent的资源使用阈值
type ResourceThreshold struct {
	// CPUThreshold CPU使用率阈值(百分比)
	CPUThreshold float64 `json:"cpu_threshold"`

	// MemoryThreshold 内存占用阈值(字节)
	MemoryThreshold uint64 `json:"memory_threshold"`

	// OpenFilesThreshold 打开文件数阈值
	// 用于检测文件描述符泄露问题
	OpenFilesThreshold int `json:"open_files_threshold"`

	// ThresholdDuration 超过阈值持续时间(触发告警/重启)
	ThresholdDuration time.Duration `json:"threshold_duration"`
}

// ResourceAlert 资源告警
// 表示资源使用超过阈值时的告警信息
type ResourceAlert struct {
	// AgentID Agent ID
	AgentID string `json:"agent_id"`

	// Type 告警类型(CPU/Memory)
	Type string `json:"type"`

	// Value 当前值
	Value float64 `json:"value"`

	// Threshold 阈值
	Threshold float64 `json:"threshold"`

	// Duration 超过阈值持续时间
	Duration time.Duration `json:"duration"`

	// Timestamp 告警时间
	Timestamp time.Time `json:"timestamp"`
}

// ResourceMonitor 资源监控器
// 定期采集Agent进程的资源使用情况,存储到元数据中,并提供查询接口
type ResourceMonitor struct {
	// multiManager 多Agent管理器引用(用于获取Agent列表和更新元数据)
	multiManager *MultiAgentManager

	// registry Agent注册表引用(用于获取Agent信息)
	registry *AgentRegistry

	// logger 日志记录器
	logger *zap.Logger

	// interval 采集间隔(默认60秒)
	interval time.Duration

	// thresholds Agent级别的阈值配置(key为agent_id)
	thresholds map[string]*ResourceThreshold

	// exceededSince 记录每个Agent的资源超阈值开始时间
	// key: agent_id, value: map[resourceType]time.Time
	exceededSince map[string]map[string]time.Time

	// ctx 上下文(用于停止监控)
	ctx context.Context

	// cancel 取消函数
	cancel context.CancelFunc

	// wg 等待组(用于优雅停止)
	wg sync.WaitGroup

	// mu 保护并发访问
	mu sync.RWMutex
}

// NewResourceMonitor 创建新的资源监控器
func NewResourceMonitor(multiManager *MultiAgentManager, registry *AgentRegistry, logger *zap.Logger) *ResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceMonitor{
		multiManager:  multiManager,
		registry:      registry,
		logger:        logger,
		interval:      60 * time.Second,
		thresholds:    make(map[string]*ResourceThreshold),
		exceededSince: make(map[string]map[string]time.Time),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// SetInterval 设置采集间隔
func (rm *ResourceMonitor) SetInterval(interval time.Duration) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.interval = interval
}

// SetThreshold 设置指定Agent的资源阈值配置
func (rm *ResourceMonitor) SetThreshold(agentID string, threshold *ResourceThreshold) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.thresholds[agentID] = threshold
	rm.logger.Info("resource threshold set",
		zap.String("agent_id", agentID),
		zap.Float64("cpu_threshold", threshold.CPUThreshold),
		zap.Uint64("memory_threshold", threshold.MemoryThreshold),
		zap.Int("open_files_threshold", threshold.OpenFilesThreshold),
		zap.Duration("threshold_duration", threshold.ThresholdDuration))
}

// collectAgentResources 采集指定Agent的资源使用情况
func (rm *ResourceMonitor) collectAgentResources(agentID string) (*ResourceDataPoint, error) {
	// 从registry获取Agent信息
	info := rm.registry.Get(agentID)
	if info == nil {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// 获取Agent的PID
	pid := info.GetPID()
	if pid == 0 {
		return nil, fmt.Errorf("agent not running: %s", agentID)
	}

	// 使用gopsutil/process库创建Process对象
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("failed to create process object: %w", err)
	}

	// 采集各项资源指标
	dataPoint := &ResourceDataPoint{
		Timestamp: time.Now(),
	}

	// CPU使用率
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		rm.logger.Warn("failed to get cpu percent",
			zap.String("agent_id", agentID),
			zap.Int("pid", pid),
			zap.Error(err))
	} else {
		dataPoint.CPU = cpuPercent
	}

	// 内存信息
	memInfo, err := proc.MemoryInfo()
	if err != nil {
		rm.logger.Warn("failed to get memory info",
			zap.String("agent_id", agentID),
			zap.Int("pid", pid),
			zap.Error(err))
	} else if memInfo != nil {
		dataPoint.MemoryRSS = memInfo.RSS
		dataPoint.MemoryVMS = memInfo.VMS
	}

	// 磁盘I/O
	ioCounters, err := proc.IOCounters()
	if err != nil {
		rm.logger.Warn("failed to get io counters",
			zap.String("agent_id", agentID),
			zap.Int("pid", pid),
			zap.Error(err))
	} else if ioCounters != nil {
		dataPoint.DiskReadBytes = ioCounters.ReadBytes
		dataPoint.DiskWriteBytes = ioCounters.WriteBytes
	}

	// 打开文件数(使用跨平台实现)
	// 优先使用自定义实现,失败后尝试 gopsutil 的 NumFDs
	numFDs, err := getFDsWithFallback(int32(pid), rm.logger)
	if err != nil {
		// 尝试使用 gopsutil 的 NumFDs 作为备用
		gopsutilFDs, gopsutilErr := proc.NumFDs()
		if gopsutilErr != nil {
			// 两种方法都失败,仅在非 Windows 平台记录警告
			if runtime.GOOS != "windows" {
				rm.logger.Warn("failed to get num fds",
					zap.String("agent_id", agentID),
					zap.Int("pid", pid),
					zap.Error(err),
					zap.NamedError("gopsutil_error", gopsutilErr))
			}
			// OpenFiles保持为0(默认值)
		} else {
			// gopsutil 方法成功
			dataPoint.OpenFiles = int(gopsutilFDs)
		}
	} else {
		// 自定义跨平台方法成功
		dataPoint.OpenFiles = int(numFDs)
		rm.logger.Debug("successfully got num fds",
			zap.String("agent_id", agentID),
			zap.Int("pid", pid),
			zap.Int32("fds", numFDs))
	}

	// 线程数(可选)
	numThreads, err := proc.NumThreads()
	if err != nil {
		rm.logger.Debug("failed to get num threads",
			zap.String("agent_id", agentID),
			zap.Int("pid", pid),
			zap.Error(err))
	} else {
		dataPoint.NumThreads = int(numThreads)
	}

	return dataPoint, nil
}

// collectAllAgents 采集所有运行中Agent的资源使用情况
func (rm *ResourceMonitor) collectAllAgents() {
	// 从multiManager获取所有运行中的Agent列表
	instances := rm.multiManager.ListAgents()

	// 并发采集以提高效率
	var wg sync.WaitGroup
	for _, instance := range instances {
		// 只采集运行中的Agent
		if !instance.IsRunning() {
			continue
		}

		info := instance.GetInfo()
		agentID := info.ID

		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// 采集资源数据
			dataPoint, err := rm.collectAgentResources(id)
			if err != nil {
				rm.logger.Warn("failed to collect agent resources",
					zap.String("agent_id", id),
					zap.Error(err))
				return
			}

			// 更新元数据
			rm.updateAgentResourceData(id, dataPoint)
		}(agentID)
	}

	wg.Wait()
}

// updateAgentResourceData 更新Agent元数据中的资源数据
func (rm *ResourceMonitor) updateAgentResourceData(agentID string, dataPoint *ResourceDataPoint) {
	// 获取Agent元数据
	metadata, err := rm.multiManager.GetAgentMetadata(agentID)
	if err != nil {
		// 如果元数据不存在,创建新记录
		instance := rm.multiManager.GetAgent(agentID)
		if instance == nil {
			rm.logger.Warn("agent instance not found",
				zap.String("agent_id", agentID))
			return
		}

		info := instance.GetInfo()
		metadata = &AgentMetadata{
			ID:            agentID,
			Type:          string(info.Type),
			Status:        "running",
			StartTime:     time.Now(),
			RestartCount:  0,
			ResourceUsage: *NewResourceUsageHistory(1440),
		}

		rm.logger.Debug("created new metadata for agent",
			zap.String("agent_id", agentID))
	}

	// 调用AddResourceData添加数据点(使用MemoryRSS作为内存指标)
	metadata.ResourceUsage.AddResourceData(dataPoint.CPU, dataPoint.MemoryRSS)

	rm.logger.Debug("added resource data point",
		zap.String("agent_id", agentID),
		zap.Float64("cpu", dataPoint.CPU),
		zap.Uint64("memory_rss", dataPoint.MemoryRSS),
		zap.Int("total_data_points", len(metadata.ResourceUsage.Timestamps)))

	// 保存更新后的元数据
	metadataStore := rm.multiManager.metadataStore
	if err := metadataStore.SaveMetadata(agentID, metadata); err != nil {
		rm.logger.Error("failed to save metadata",
			zap.String("agent_id", agentID),
			zap.Error(err))
		return
	}

	// 检查资源阈值
	rm.checkResourceThresholds(agentID, dataPoint)
}

// checkResourceThresholds 检查资源阈值
func (rm *ResourceMonitor) checkResourceThresholds(agentID string, dataPoint *ResourceDataPoint) {
	rm.mu.RLock()
	threshold, exists := rm.thresholds[agentID]
	rm.mu.RUnlock()

	// 如果未配置阈值,使用默认值或跳过检查
	if !exists || threshold == nil {
		return
	}

	now := time.Now()

	// 检查CPU使用率
	if dataPoint.CPU > threshold.CPUThreshold {
		duration := rm.getExceededDuration(agentID, "cpu", now)
		if duration >= threshold.ThresholdDuration {
			rm.logger.Error("agent cpu over threshold for too long",
				zap.String("agent_id", agentID),
				zap.Float64("cpu", dataPoint.CPU),
				zap.Float64("threshold", threshold.CPUThreshold),
				zap.Duration("duration", duration))

			// 可选:调用MultiAgentManager触发重启
			// 这里只记录告警,实际重启逻辑由HealthChecker处理
		} else {
			rm.logger.Warn("agent cpu over threshold",
				zap.String("agent_id", agentID),
				zap.Float64("cpu", dataPoint.CPU),
				zap.Float64("threshold", threshold.CPUThreshold),
				zap.Duration("duration", duration))
		}
	} else {
		// CPU恢复正常,清除超阈值开始时间
		rm.clearExceededSince(agentID, "cpu")
	}

	// 检查内存占用
	if dataPoint.MemoryRSS > threshold.MemoryThreshold {
		duration := rm.getExceededDuration(agentID, "memory", now)
		if duration >= threshold.ThresholdDuration {
			rm.logger.Error("agent memory over threshold for too long",
				zap.String("agent_id", agentID),
				zap.Uint64("memory_rss", dataPoint.MemoryRSS),
				zap.Uint64("threshold", threshold.MemoryThreshold),
				zap.Duration("duration", duration))

			// 可选:调用MultiAgentManager触发重启
			// 这里只记录告警,实际重启逻辑由HealthChecker处理
		} else {
			rm.logger.Warn("agent memory over threshold",
				zap.String("agent_id", agentID),
				zap.Uint64("memory_rss", dataPoint.MemoryRSS),
				zap.Uint64("threshold", threshold.MemoryThreshold),
				zap.Duration("duration", duration))
		}
	} else {
		// 内存恢复正常,清除超阈值开始时间
		rm.clearExceededSince(agentID, "memory")
	}

	// 检查打开文件数(文件描述符泄露检测)
	if threshold.OpenFilesThreshold > 0 && dataPoint.OpenFiles > threshold.OpenFilesThreshold {
		duration := rm.getExceededDuration(agentID, "open_files", now)
		if duration >= threshold.ThresholdDuration {
			rm.logger.Error("agent open files over threshold for too long - possible fd leak",
				zap.String("agent_id", agentID),
				zap.Int("open_files", dataPoint.OpenFiles),
				zap.Int("threshold", threshold.OpenFilesThreshold),
				zap.Duration("duration", duration),
				zap.String("warning", "file descriptor leak detected"))

			// 文件描述符泄露是严重问题,可能需要重启Agent
		} else {
			rm.logger.Warn("agent open files over threshold",
				zap.String("agent_id", agentID),
				zap.Int("open_files", dataPoint.OpenFiles),
				zap.Int("threshold", threshold.OpenFilesThreshold),
				zap.Duration("duration", duration))
		}
	} else {
		// 打开文件数恢复正常,清除超阈值开始时间
		rm.clearExceededSince(agentID, "open_files")
	}
}

// getExceededDuration 获取资源超阈值持续时间
func (rm *ResourceMonitor) getExceededDuration(agentID string, resourceType string, now time.Time) time.Duration {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 初始化exceededSince map
	if rm.exceededSince[agentID] == nil {
		rm.exceededSince[agentID] = make(map[string]time.Time)
	}

	// 如果还没有记录超阈值开始时间,记录当前时间
	if rm.exceededSince[agentID][resourceType].IsZero() {
		rm.exceededSince[agentID][resourceType] = now
		return 0
	}

	// 计算持续时间
	return now.Sub(rm.exceededSince[agentID][resourceType])
}

// clearExceededSince 清除超阈值开始时间
func (rm *ResourceMonitor) clearExceededSince(agentID string, resourceType string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.exceededSince[agentID] != nil {
		rm.exceededSince[agentID][resourceType] = time.Time{}
	}
}

// Start 启动监控循环
func (rm *ResourceMonitor) Start() {
	rm.logger.Info("starting resource monitor",
		zap.Duration("interval", rm.interval))

	rm.wg.Add(1)
	go rm.monitorLoop()
}

// Stop 停止监控
func (rm *ResourceMonitor) Stop() {
	rm.logger.Info("stopping resource monitor")
	rm.cancel()
	rm.wg.Wait()
	rm.logger.Info("resource monitor stopped")
}

// monitorLoop 监控循环
func (rm *ResourceMonitor) monitorLoop() {
	defer rm.wg.Done()

	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()

	// 立即执行一次采集
	rm.collectAllAgents()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.collectAllAgents()
		}
	}
}

// GetResourceHistory 获取指定Agent的资源使用历史数据
func (rm *ResourceMonitor) GetResourceHistory(agentID string, duration time.Duration) ([]ResourceDataPoint, error) {
	// 获取Agent元数据
	metadata, err := rm.multiManager.GetAgentMetadata(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	// 快速复制数据,减少持锁时间
	metadata.ResourceUsage.mu.RLock()

	// 快速检查是否有数据
	if len(metadata.ResourceUsage.Timestamps) == 0 {
		metadata.ResourceUsage.mu.RUnlock()
		rm.logger.Debug("no resource history data",
			zap.String("agent_id", agentID))
		return []ResourceDataPoint{}, nil
	}

	// 快速复制切片(浅拷贝引用),在锁内完成
	timestamps := make([]time.Time, len(metadata.ResourceUsage.Timestamps))
	cpuData := make([]float64, len(metadata.ResourceUsage.CPU))
	memData := make([]uint64, len(metadata.ResourceUsage.Memory))
	copy(timestamps, metadata.ResourceUsage.Timestamps)
	copy(cpuData, metadata.ResourceUsage.CPU)
	copy(memData, metadata.ResourceUsage.Memory)
	totalPoints := len(timestamps)

	metadata.ResourceUsage.mu.RUnlock()
	// 释放锁后处理数据

	// 计算截止时间
	cutoff := time.Now().Add(-duration)

	// 构建ResourceDataPoint切片
	result := make([]ResourceDataPoint, 0)
	for i, ts := range timestamps {
		// 只包含在时间范围内的数据点
		if ts.After(cutoff) || ts.Equal(cutoff) {
			if i < len(cpuData) && i < len(memData) {
				result = append(result, ResourceDataPoint{
					Timestamp:  ts,
					CPU:        cpuData[i],
					MemoryRSS:  memData[i],
					MemoryVMS:  0, // VMS数据未在ResourceUsageHistory中存储
					OpenFiles:  0, // 其他字段未在ResourceUsageHistory中存储
					NumThreads: 0,
				})
			}
		}
	}

	rm.logger.Debug("retrieved resource history",
		zap.String("agent_id", agentID),
		zap.Duration("duration", duration),
		zap.Int("total_data_points", totalPoints),
		zap.Int("filtered_data_points", len(result)))

	return result, nil
}

// GetResourceHistoryAggregated 获取聚合后的资源使用历史数据
func (rm *ResourceMonitor) GetResourceHistoryAggregated(agentID string, duration time.Duration, interval time.Duration) ([]ResourceDataPoint, error) {
	// 获取原始历史数据
	rawData, err := rm.GetResourceHistory(agentID, duration)
	if err != nil {
		return nil, err
	}

	if len(rawData) == 0 {
		return []ResourceDataPoint{}, nil
	}

	// 按interval间隔聚合数据(计算平均值)
	result := make([]ResourceDataPoint, 0)
	currentWindowStart := rawData[0].Timestamp.Truncate(interval)
	var windowData []ResourceDataPoint

	for _, point := range rawData {
		windowStart := point.Timestamp.Truncate(interval)

		// 如果进入新的时间窗口,聚合上一个窗口的数据
		if windowStart.After(currentWindowStart) {
			if len(windowData) > 0 {
				aggregated := rm.aggregateDataPoints(windowData, currentWindowStart)
				result = append(result, aggregated)
			}
			windowData = []ResourceDataPoint{point}
			currentWindowStart = windowStart
		} else {
			windowData = append(windowData, point)
		}
	}

	// 处理最后一个窗口
	if len(windowData) > 0 {
		aggregated := rm.aggregateDataPoints(windowData, currentWindowStart)
		result = append(result, aggregated)
	}

	return result, nil
}

// aggregateDataPoints 聚合数据点(计算平均值)
func (rm *ResourceMonitor) aggregateDataPoints(points []ResourceDataPoint, timestamp time.Time) ResourceDataPoint {
	if len(points) == 0 {
		return ResourceDataPoint{Timestamp: timestamp}
	}

	var sumCPU float64
	var sumMemoryRSS uint64
	var sumMemoryVMS uint64
	var sumDiskReadBytes uint64
	var sumDiskWriteBytes uint64
	var sumOpenFiles int
	var sumNumThreads int

	for _, point := range points {
		sumCPU += point.CPU
		sumMemoryRSS += point.MemoryRSS
		sumMemoryVMS += point.MemoryVMS
		sumDiskReadBytes += point.DiskReadBytes
		sumDiskWriteBytes += point.DiskWriteBytes
		sumOpenFiles += point.OpenFiles
		sumNumThreads += point.NumThreads
	}

	count := float64(len(points))
	return ResourceDataPoint{
		Timestamp:      timestamp,
		CPU:            sumCPU / count,
		MemoryRSS:      sumMemoryRSS / uint64(count),
		MemoryVMS:      sumMemoryVMS / uint64(count),
		DiskReadBytes:  sumDiskReadBytes / uint64(count),
		DiskWriteBytes: sumDiskWriteBytes / uint64(count),
		OpenFiles:      sumOpenFiles / len(points),
		NumThreads:     sumNumThreads / len(points),
	}
}

// GetCurrentResources 立即采集指定Agent的当前资源使用情况
func (rm *ResourceMonitor) GetCurrentResources(agentID string) (*ResourceDataPoint, error) {
	return rm.collectAgentResources(agentID)
}
