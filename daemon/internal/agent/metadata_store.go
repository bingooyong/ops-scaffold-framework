package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AgentMetadata Agent元数据结构体
// 维护每个Agent的完整元数据信息，包括版本、状态、启动时间、重启次数、心跳时间、资源使用历史等
type AgentMetadata struct {
	// ID Agent唯一标识符
	ID string `json:"id"`

	// Type Agent类型(filebeat/telegraf/node_exporter等)
	Type string `json:"type"`

	// Version Agent版本号(可选,从Agent进程获取或配置)
	Version string `json:"version,omitempty"`

	// Status 运行状态(running/stopped/error/starting/stopping)
	Status string `json:"status"`

	// StartTime 启动时间
	StartTime time.Time `json:"start_time"`

	// RestartCount 重启次数
	RestartCount int `json:"restart_count"`

	// LastHeartbeat 最后心跳时间
	LastHeartbeat time.Time `json:"last_heartbeat"`

	// ResourceUsage 资源使用历史记录(CPU/Memory)
	ResourceUsage ResourceUsageHistory `json:"resource_usage"`
}

// ResourceUsageHistory 资源使用历史记录
type ResourceUsageHistory struct {
	// CPU CPU使用率历史记录(百分比)
	CPU []float64 `json:"cpu"`

	// Memory 内存占用历史记录(字节)
	Memory []uint64 `json:"memory"`

	// MaxHistorySize 最大历史记录数量(默认1440,保留24小时数据,每分钟一个点)
	MaxHistorySize int `json:"max_history_size"`

	// Timestamps 时间戳记录,用于GetRecent方法
	Timestamps []time.Time `json:"timestamps"`

	// mu 保护并发访问的锁
	mu sync.RWMutex
}

// NewResourceUsageHistory 创建新的资源使用历史记录
func NewResourceUsageHistory(maxSize int) *ResourceUsageHistory {
	if maxSize <= 0 {
		maxSize = 1440 // 默认1440,保留24小时数据,每分钟一个点
	}
	return &ResourceUsageHistory{
		CPU:            make([]float64, 0, maxSize),
		Memory:         make([]uint64, 0, maxSize),
		Timestamps:     make([]time.Time, 0, maxSize),
		MaxHistorySize: maxSize,
	}
}

// AddResourceData 添加资源使用数据点,自动维护历史记录大小
func (r *ResourceUsageHistory) AddResourceData(cpu float64, memory uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.CPU = append(r.CPU, cpu)
	r.Memory = append(r.Memory, memory)
	r.Timestamps = append(r.Timestamps, now)

	// 如果超过最大大小,删除最旧的数据
	if len(r.CPU) > r.MaxHistorySize {
		r.CPU = r.CPU[1:]
		r.Memory = r.Memory[1:]
		r.Timestamps = r.Timestamps[1:]
	}
}

// GetRecentCPU 获取最近指定时间范围的CPU数据
func (r *ResourceUsageHistory) GetRecentCPU(duration time.Duration) []float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.Timestamps) == 0 {
		return []float64{}
	}

	cutoff := time.Now().Add(-duration)
	result := make([]float64, 0)

	for i, ts := range r.Timestamps {
		if ts.After(cutoff) || ts.Equal(cutoff) {
			result = append(result, r.CPU[i])
		}
	}

	return result
}

// GetRecentMemory 获取最近指定时间范围的Memory数据
func (r *ResourceUsageHistory) GetRecentMemory(duration time.Duration) []uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.Timestamps) == 0 {
		return []uint64{}
	}

	cutoff := time.Now().Add(-duration)
	result := make([]uint64, 0)

	for i, ts := range r.Timestamps {
		if ts.After(cutoff) || ts.Equal(cutoff) {
			result = append(result, r.Memory[i])
		}
	}

	return result
}

// MetadataStore 元数据存储接口
type MetadataStore interface {
	// GetMetadata 获取指定Agent的元数据
	GetMetadata(agentID string) (*AgentMetadata, error)

	// ListAllMetadata 列举所有Agent的元数据
	ListAllMetadata() ([]*AgentMetadata, error)

	// UpdateMetadata 更新指定Agent的元数据(仅更新非零值字段)
	UpdateMetadata(agentID string, updates *AgentMetadata) error

	// SaveMetadata 保存指定Agent的元数据(完整覆盖)
	SaveMetadata(agentID string, metadata *AgentMetadata) error

	// DeleteMetadata 删除指定Agent的元数据
	DeleteMetadata(agentID string) error
}

// FileMetadataStore 基于文件的元数据存储实现
type FileMetadataStore struct {
	// workDir 工作目录路径
	workDir string

	// mu 保护并发访问的读写锁
	mu sync.RWMutex

	// logger 日志记录器
	logger *zap.Logger
}

// NewFileMetadataStore 创建新的文件元数据存储
func NewFileMetadataStore(workDir string, logger *zap.Logger) (*FileMetadataStore, error) {
	metadataDir := filepath.Join(workDir, "metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return &FileMetadataStore{
		workDir: workDir,
		logger:  logger,
	}, nil
}

// getMetadataPath 获取元数据文件路径
func (f *FileMetadataStore) getMetadataPath(agentID string) string {
	return filepath.Join(f.workDir, "metadata", agentID+".json")
}

// SaveMetadata 保存元数据(原子性写入)
func (f *FileMetadataStore) SaveMetadata(agentID string, metadata *AgentMetadata) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	metadataPath := f.getMetadataPath(agentID)
	tmpPath := metadataPath + ".tmp"

	// 序列化metadata为JSON(使用json.MarshalIndent格式化)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// 写入临时文件
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// 验证临时文件可读(读取并解析测试)
	var verify AgentMetadata
	verifyData, err := os.ReadFile(tmpPath)
	if err != nil {
		os.Remove(tmpPath) // 清理临时文件
		return fmt.Errorf("failed to read temporary file for verification: %w", err)
	}

	if err := json.Unmarshal(verifyData, &verify); err != nil {
		os.Remove(tmpPath) // 清理临时文件
		return fmt.Errorf("failed to verify temporary file: %w", err)
	}

	// 使用os.Rename原子替换原文件
	if err := os.Rename(tmpPath, metadataPath); err != nil {
		os.Remove(tmpPath) // 清理临时文件
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	f.logger.Debug("metadata saved",
		zap.String("agent_id", agentID),
		zap.String("path", metadataPath))

	return nil
}

// getMetadataUnlocked 读取元数据的内部方法(不加锁,供已持有锁的方法调用)
func (f *FileMetadataStore) getMetadataUnlocked(agentID string) (*AgentMetadata, error) {
	metadataPath := f.getMetadataPath(agentID)

	// 读取JSON文件
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	// 解析为AgentMetadata结构体
	var metadata AgentMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// 如果ResourceUsage没有初始化,初始化它
	if metadata.ResourceUsage.MaxHistorySize == 0 {
		metadata.ResourceUsage = *NewResourceUsageHistory(1440)
	} else {
		// 确保切片已初始化
		if metadata.ResourceUsage.CPU == nil {
			metadata.ResourceUsage.CPU = make([]float64, 0)
		}
		if metadata.ResourceUsage.Memory == nil {
			metadata.ResourceUsage.Memory = make([]uint64, 0)
		}
		if metadata.ResourceUsage.Timestamps == nil {
			metadata.ResourceUsage.Timestamps = make([]time.Time, 0)
		}
	}

	return &metadata, nil
}

// GetMetadata 获取元数据
func (f *FileMetadataStore) GetMetadata(agentID string) (*AgentMetadata, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.getMetadataUnlocked(agentID)
}

// ListAllMetadata 列举所有元数据
func (f *FileMetadataStore) ListAllMetadata() ([]*AgentMetadata, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	metadataDir := filepath.Join(f.workDir, "metadata")

	// 扫描metadata目录
	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AgentMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	result := make([]*AgentMetadata, 0, len(entries))

	// 读取所有*.json文件
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// 提取agentID(去掉.json后缀)
		agentID := entry.Name()[:len(entry.Name())-5]

		// 读取并解析元数据(使用不加锁的内部方法,因为已经持有锁)
		metadata, err := f.getMetadataUnlocked(agentID)
		if err != nil {
			// 忽略解析失败的文件,记录WARNING日志
			f.logger.Warn("failed to parse metadata file",
				zap.String("agent_id", agentID),
				zap.String("file", entry.Name()),
				zap.Error(err))
			continue
		}

		result = append(result, metadata)
	}

	return result, nil
}

// UpdateMetadata 更新元数据(仅更新非零值字段)
func (f *FileMetadataStore) UpdateMetadata(agentID string, updates *AgentMetadata) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 读取当前元数据(使用不加锁的内部方法,因为已经持有锁)
	current, err := f.getMetadataUnlocked(agentID)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果不存在,创建新记录(注意:这里需要先释放锁再调用SaveMetadata)
			f.mu.Unlock()
			result := f.SaveMetadata(agentID, updates)
			f.mu.Lock() // 重新获取锁以保证defer正常工作
			return result
		}
		return fmt.Errorf("failed to get current metadata: %w", err)
	}

	// 合并更新字段(仅更新非零值字段)
	if updates.ID != "" {
		current.ID = updates.ID
	}
	if updates.Type != "" {
		current.Type = updates.Type
	}
	if updates.Version != "" {
		current.Version = updates.Version
	}
	if updates.Status != "" {
		current.Status = updates.Status
	}
	if !updates.StartTime.IsZero() {
		current.StartTime = updates.StartTime
	}
	// RestartCount: 由于无法区分"未设置"和"设置为0",这里采用特殊处理
	// 如果updates.RestartCount > 0,则更新
	// 如果updates.RestartCount == 0 且 current.RestartCount > 0,可能是重置操作,也更新
	// 注意:这要求调用者明确设置RestartCount值
	if updates.RestartCount > 0 || (updates.RestartCount == 0 && current.RestartCount > 0) {
		current.RestartCount = updates.RestartCount
	}
	if !updates.LastHeartbeat.IsZero() {
		current.LastHeartbeat = updates.LastHeartbeat
	}
	// ResourceUsage: 不在这里合并,应该通过AddResourceData方法单独更新

	// 保存更新后的元数据
	return f.SaveMetadata(agentID, current)
}

// DeleteMetadata 删除元数据
func (f *FileMetadataStore) DeleteMetadata(agentID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	metadataPath := f.getMetadataPath(agentID)

	// 删除元数据文件
	if err := os.Remove(metadataPath); err != nil {
		if os.IsNotExist(err) {
			// 如果文件不存在,不返回错误(幂等操作)
			return nil
		}
		return fmt.Errorf("failed to delete metadata file: %w", err)
	}

	f.logger.Debug("metadata deleted",
		zap.String("agent_id", agentID),
		zap.String("path", metadataPath))

	return nil
}
