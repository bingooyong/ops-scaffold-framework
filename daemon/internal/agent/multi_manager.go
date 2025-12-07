package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AgentStateChangeCallback Agent状态变化回调函数类型
type AgentStateChangeCallback func(agentID string, status AgentStatus, pid int, lastHeartbeat time.Time)

// MultiAgentManager 多Agent管理器
// 使用AgentRegistry管理多个AgentInstance，提供批量操作和单个Agent操作
type MultiAgentManager struct {
	// registry Agent注册表，存储所有Agent信息
	registry *AgentRegistry

	// instances 存储所有Agent实例管理器
	// key: Agent ID
	// value: AgentInstance指针
	instances map[string]*AgentInstance

	// mu 保护instances的并发访问锁
	mu sync.RWMutex

	// metadataStore 元数据存储接口
	metadataStore MetadataStore

	// stateChangeCallback Agent状态变化回调函数(可选)
	stateChangeCallback AgentStateChangeCallback

	// logger 日志记录器
	logger *zap.Logger
}

// NewMultiAgentManager 创建新的多Agent管理器
func NewMultiAgentManager(workDir string, logger *zap.Logger) (*MultiAgentManager, error) {
	// 创建FileMetadataStore实例
	metadataStore, err := NewFileMetadataStore(workDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata store: %w", err)
	}

	return &MultiAgentManager{
		registry:      NewAgentRegistry(),
		instances:     make(map[string]*AgentInstance),
		metadataStore: metadataStore,
		logger:        logger,
	}, nil
}

// SetStateChangeCallback 设置Agent状态变化回调函数
func (mam *MultiAgentManager) SetStateChangeCallback(callback AgentStateChangeCallback) {
	mam.stateChangeCallback = callback
}

// RegisterAgent 注册一个新的Agent
// 从AgentInfo创建AgentInstance并注册到管理器
func (mam *MultiAgentManager) RegisterAgent(info *AgentInfo) (*AgentInstance, error) {
	mam.mu.Lock()
	defer mam.mu.Unlock()

	// 检查是否已存在
	if _, exists := mam.instances[info.ID]; exists {
		return nil, &AgentExistsError{ID: info.ID}
	}

	// 创建AgentInstance
	instance := NewAgentInstance(info, mam.logger)

	// 存储到instances map
	mam.instances[info.ID] = instance

	mam.logger.Info("agent registered",
		zap.String("agent_id", info.ID),
		zap.String("agent_type", string(info.Type)),
		zap.String("agent_name", info.Name))

	return instance, nil
}

// GetAgent 根据ID获取Agent实例
func (mam *MultiAgentManager) GetAgent(id string) *AgentInstance {
	mam.mu.RLock()
	defer mam.mu.RUnlock()
	return mam.instances[id]
}

// UnregisterAgent 从管理器中移除Agent
func (mam *MultiAgentManager) UnregisterAgent(id string) error {
	mam.mu.Lock()
	defer mam.mu.Unlock()

	instance, exists := mam.instances[id]
	if !exists {
		return &AgentNotFoundError{ID: id}
	}

	// 检查Agent是否正在运行
	if instance.IsRunning() {
		return &AgentRunningError{ID: id}
	}

	// 从instances map中删除
	delete(mam.instances, id)

	mam.logger.Info("agent unregistered",
		zap.String("agent_id", id))

	return nil
}

// ListAgents 列举所有已注册的Agent
func (mam *MultiAgentManager) ListAgents() []*AgentInstance {
	mam.mu.RLock()
	defer mam.mu.RUnlock()

	result := make([]*AgentInstance, 0, len(mam.instances))
	for _, instance := range mam.instances {
		result = append(result, instance)
	}

	return result
}

// GetRegistry 获取Agent注册表
func (mam *MultiAgentManager) GetRegistry() *AgentRegistry {
	return mam.registry
}

// StartAgent 启动指定的Agent
func (mam *MultiAgentManager) StartAgent(ctx context.Context, id string) error {
	instance := mam.GetAgent(id)
	if instance == nil {
		return &AgentNotFoundError{ID: id}
	}

	err := instance.Start(ctx)
	if err != nil {
		return err
	}

	// Agent启动成功后,创建或更新元数据
	info := instance.GetInfo()
	now := time.Now()

	// 尝试获取现有元数据
	metadata, err := mam.metadataStore.GetMetadata(id)
	if err != nil {
		// 如果元数据不存在,创建新记录
		if err == os.ErrNotExist {
			metadata = &AgentMetadata{
				ID:            id,
				Type:          string(info.Type),
				Status:        "running",
				StartTime:     now,
				RestartCount:  0,
				ResourceUsage: *NewResourceUsageHistory(1440),
			}
			if err := mam.metadataStore.SaveMetadata(id, metadata); err != nil {
				mam.logger.Warn("failed to save metadata after agent start",
					zap.String("agent_id", id),
					zap.Error(err))
			}
		} else {
			mam.logger.Warn("failed to get metadata after agent start",
				zap.String("agent_id", id),
				zap.Error(err))
		}
	} else {
		// 更新现有元数据
		metadata.Status = "running"
		metadata.StartTime = now
		if err := mam.metadataStore.SaveMetadata(id, metadata); err != nil {
			mam.logger.Warn("failed to update metadata after agent start",
				zap.String("agent_id", id),
				zap.Error(err))
		}
	}

	// 通知状态变化回调
	if mam.stateChangeCallback != nil {
		info := instance.GetInfo()
		var lastHeartbeat time.Time
		metadata, err := mam.metadataStore.GetMetadata(id)
		if err == nil && metadata != nil {
			lastHeartbeat = metadata.LastHeartbeat
		}
		mam.stateChangeCallback(id, info.GetStatus(), info.GetPID(), lastHeartbeat)
	}

	return nil
}

// StopAgent 停止指定的Agent
func (mam *MultiAgentManager) StopAgent(ctx context.Context, id string, graceful bool) error {
	instance := mam.GetAgent(id)
	if instance == nil {
		return &AgentNotFoundError{ID: id}
	}

	err := instance.Stop(ctx, graceful)
	if err != nil {
		return err
	}

	// Agent停止后,更新元数据
	updates := &AgentMetadata{
		Status: "stopped",
	}
	if err := mam.metadataStore.UpdateMetadata(id, updates); err != nil {
		mam.logger.Warn("failed to update metadata after agent stop",
			zap.String("agent_id", id),
			zap.Error(err))
	}

	// 通知状态变化回调
	if mam.stateChangeCallback != nil {
		info := instance.GetInfo()
		var lastHeartbeat time.Time
		metadata, err := mam.metadataStore.GetMetadata(id)
		if err == nil && metadata != nil {
			lastHeartbeat = metadata.LastHeartbeat
		}
		mam.stateChangeCallback(id, info.GetStatus(), info.GetPID(), lastHeartbeat)
	}

	return nil
}

// RestartAgent 重启指定的Agent
func (mam *MultiAgentManager) RestartAgent(ctx context.Context, id string) error {
	instance := mam.GetAgent(id)
	if instance == nil {
		return &AgentNotFoundError{ID: id}
	}

	err := instance.Restart(ctx)
	if err != nil {
		return err
	}

	// Agent重启后,更新元数据
	info := instance.GetInfo()
	now := time.Now()
	restartCount := info.GetRestartCount()

	updates := &AgentMetadata{
		Status:       "running",
		StartTime:    now,
		RestartCount: restartCount,
	}
	if err := mam.metadataStore.UpdateMetadata(id, updates); err != nil {
		mam.logger.Warn("failed to update metadata after agent restart",
			zap.String("agent_id", id),
			zap.Error(err))
	}

	// 通知状态变化回调
	if mam.stateChangeCallback != nil {
		info := instance.GetInfo()
		var lastHeartbeat time.Time
		metadata, err := mam.metadataStore.GetMetadata(id)
		if err == nil && metadata != nil {
			lastHeartbeat = metadata.LastHeartbeat
		}
		mam.stateChangeCallback(id, info.GetStatus(), info.GetPID(), lastHeartbeat)
	}

	return nil
}

// StartAll 启动所有已注册的Agent
func (mam *MultiAgentManager) StartAll(ctx context.Context) map[string]error {
	mam.mu.RLock()
	instances := make(map[string]*AgentInstance)
	for id, instance := range mam.instances {
		instances[id] = instance
	}
	mam.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for id, instance := range instances {
		wg.Add(1)
		go func(agentID string, inst *AgentInstance) {
			defer wg.Done()
			err := inst.Start(ctx)
			mu.Lock()
			results[agentID] = err
			mu.Unlock()
			if err != nil {
				mam.logger.Error("failed to start agent",
					zap.String("agent_id", agentID),
					zap.Error(err))
			}
		}(id, instance)
	}

	wg.Wait()

	mam.logger.Info("started all agents",
		zap.Int("total", len(instances)),
		zap.Int("success", len(instances)-len(results)))

	return results
}

// StopAll 停止所有已注册的Agent
func (mam *MultiAgentManager) StopAll(ctx context.Context, graceful bool) map[string]error {
	mam.mu.RLock()
	instances := make(map[string]*AgentInstance)
	for id, instance := range mam.instances {
		instances[id] = instance
	}
	mam.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for id, instance := range instances {
		wg.Add(1)
		go func(agentID string, inst *AgentInstance) {
			defer wg.Done()
			err := inst.Stop(ctx, graceful)
			mu.Lock()
			results[agentID] = err
			mu.Unlock()
			if err != nil {
				mam.logger.Error("failed to stop agent",
					zap.String("agent_id", agentID),
					zap.Error(err))
			}
		}(id, instance)
	}

	wg.Wait()

	mam.logger.Info("stopped all agents",
		zap.Int("total", len(instances)),
		zap.Int("success", len(instances)-len(results)),
		zap.Bool("graceful", graceful))

	return results
}

// RestartAll 重启所有已注册的Agent
func (mam *MultiAgentManager) RestartAll(ctx context.Context) map[string]error {
	mam.mu.RLock()
	instances := make(map[string]*AgentInstance)
	for id, instance := range mam.instances {
		instances[id] = instance
	}
	mam.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for id, instance := range instances {
		wg.Add(1)
		go func(agentID string, inst *AgentInstance) {
			defer wg.Done()
			err := inst.Restart(ctx)
			mu.Lock()
			results[agentID] = err
			mu.Unlock()
			if err != nil {
				mam.logger.Error("failed to restart agent",
					zap.String("agent_id", agentID),
					zap.Error(err))
			}
		}(id, instance)
	}

	wg.Wait()

	mam.logger.Info("restarted all agents",
		zap.Int("total", len(instances)),
		zap.Int("success", len(instances)-len(results)))

	return results
}

// Count 返回已注册的Agent数量
func (mam *MultiAgentManager) Count() int {
	mam.mu.RLock()
	defer mam.mu.RUnlock()
	return len(mam.instances)
}

// LoadAgentsFromRegistry 从注册表加载所有Agent并创建实例
// 这个方法用于从配置加载Agent后，将AgentInfo转换为AgentInstance
func (mam *MultiAgentManager) LoadAgentsFromRegistry() error {
	agents := mam.registry.List()

	mam.mu.Lock()
	defer mam.mu.Unlock()

	for _, info := range agents {
		// 如果实例已存在，跳过
		if _, exists := mam.instances[info.ID]; exists {
			continue
		}

		// 创建AgentInstance
		instance := NewAgentInstance(info, mam.logger)
		mam.instances[info.ID] = instance

		mam.logger.Debug("agent instance created from registry",
			zap.String("agent_id", info.ID),
			zap.String("agent_type", string(info.Type)))
	}

	return nil
}

// GetAgentStatus 获取指定Agent的状态信息
func (mam *MultiAgentManager) GetAgentStatus(id string) (*AgentStatusInfo, error) {
	instance := mam.GetAgent(id)
	if instance == nil {
		return nil, &AgentNotFoundError{ID: id}
	}

	info := instance.GetInfo()
	return &AgentStatusInfo{
		ID:           info.ID,
		Type:         info.Type,
		Name:         info.Name,
		Status:       info.GetStatus(),
		PID:          info.GetPID(),
		RestartCount: info.GetRestartCount(),
		LastRestart:  info.GetLastRestart(),
		IsRunning:    instance.IsRunning(),
	}, nil
}

// GetAllAgentStatus 获取所有Agent的状态信息
func (mam *MultiAgentManager) GetAllAgentStatus() []*AgentStatusInfo {
	mam.mu.RLock()
	instances := make([]*AgentInstance, 0, len(mam.instances))
	for _, instance := range mam.instances {
		instances = append(instances, instance)
	}
	mam.mu.RUnlock()

	results := make([]*AgentStatusInfo, 0, len(instances))
	for _, instance := range instances {
		info := instance.GetInfo()
		results = append(results, &AgentStatusInfo{
			ID:           info.ID,
			Type:         info.Type,
			Name:         info.Name,
			Status:       info.GetStatus(),
			PID:          info.GetPID(),
			RestartCount: info.GetRestartCount(),
			LastRestart:  info.GetLastRestart(),
			IsRunning:    instance.IsRunning(),
		})
	}

	return results
}

// AgentStatusInfo Agent状态信息
// 用于返回Agent的当前状态，不包含敏感信息
type AgentStatusInfo struct {
	ID           string      `json:"id"`
	Type         AgentType   `json:"type"`
	Name         string      `json:"name"`
	Status       AgentStatus `json:"status"`
	PID          int         `json:"pid"`
	RestartCount int         `json:"restart_count"`
	LastRestart  time.Time   `json:"last_restart"`
	IsRunning    bool        `json:"is_running"`
}

// UpdateHeartbeat 更新Agent心跳信息(供心跳接收器调用)
func (mam *MultiAgentManager) UpdateHeartbeat(agentID string, timestamp time.Time, cpu float64, memory uint64) error {
	// 获取元数据
	metadata, err := mam.metadataStore.GetMetadata(agentID)
	if err != nil {
		if err == os.ErrNotExist {
			// 如果元数据不存在,创建新记录
			instance := mam.GetAgent(agentID)
			if instance == nil {
				return &AgentNotFoundError{ID: agentID}
			}
			info := instance.GetInfo()
			metadata = &AgentMetadata{
				ID:            agentID,
				Type:          string(info.Type),
				Status:        "running",
				StartTime:     time.Now(),
				RestartCount:  0,
				LastHeartbeat: timestamp,
				ResourceUsage: *NewResourceUsageHistory(1440),
			}
		} else {
			return fmt.Errorf("failed to get metadata: %w", err)
		}
	}

	// 更新LastHeartbeat
	metadata.LastHeartbeat = timestamp

	// 添加资源使用数据点
	metadata.ResourceUsage.AddResourceData(cpu, memory)

	// 保存更新后的元数据
	if err := mam.metadataStore.SaveMetadata(agentID, metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// GetAgentMetadata 获取指定Agent的元数据
func (mam *MultiAgentManager) GetAgentMetadata(agentID string) (*AgentMetadata, error) {
	return mam.metadataStore.GetMetadata(agentID)
}
