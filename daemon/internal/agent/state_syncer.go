package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AgentState Agent状态结构体(用于状态同步)
type AgentState struct {
	AgentID       string
	Status        AgentStatus
	PID           int
	LastHeartbeat time.Time
	Type          string // Agent类型(filebeat/telegraf/node_exporter等)
	Version       string // Agent版本号
}

// StateSyncer Agent状态同步器
// 监听Agent状态变化并定期向Manager上报状态
type StateSyncer struct {
	// multiManager 多Agent管理器引用(用于获取Agent状态)
	multiManager *MultiAgentManager

	// registry Agent注册表引用(用于获取Agent列表)
	registry *AgentRegistry

	// managerAddress Manager gRPC服务器地址
	managerAddress string

	// syncInterval 同步间隔(默认30秒)
	syncInterval time.Duration

	// pendingStates 待同步的状态缓存(key为agent_id)
	pendingStates map[string]*AgentState

	// mu 保护并发访问
	mu sync.RWMutex

	// logger 日志记录器
	logger *zap.Logger

	// ctx 上下文(用于停止同步)
	ctx context.Context

	// cancel 取消函数
	cancel context.CancelFunc

	// wg 等待组
	wg sync.WaitGroup

	// managerClient Manager gRPC客户端(用于调用Manager的SyncAgentStates方法)
	managerClient ManagerClient
}

// ManagerClient Manager gRPC客户端接口
type ManagerClient interface {
	SyncAgentStates(ctx context.Context, nodeID string, states []*AgentState) error
}

// NewStateSyncer 创建新的状态同步器
func NewStateSyncer(
	multiManager *MultiAgentManager,
	registry *AgentRegistry,
	managerAddress string,
	logger *zap.Logger,
) *StateSyncer {
	ctx, cancel := context.WithCancel(context.Background())

	return &StateSyncer{
		multiManager:   multiManager,
		registry:       registry,
		managerAddress: managerAddress,
		syncInterval:   30 * time.Second, // 默认30秒
		pendingStates:  make(map[string]*AgentState),
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// SetManagerClient 设置Manager gRPC客户端
func (ss *StateSyncer) SetManagerClient(client ManagerClient) {
	ss.managerClient = client
}

// SetSyncInterval 设置同步间隔
func (ss *StateSyncer) SetSyncInterval(interval time.Duration) {
	ss.syncInterval = interval
}

// OnAgentStateChange 监听Agent状态变化
func (ss *StateSyncer) OnAgentStateChange(agentID string, status AgentStatus, pid int, lastHeartbeat time.Time) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// 如果Status是空的（零值），使用默认值"stopped"
	if status == "" {
		status = StatusStopped
		ss.logger.Warn("agent status is empty in state change callback, using default 'stopped'",
			zap.String("agent_id", agentID))
	}

	// 获取Agent信息以获取Type
	var agentType string
	var version string
	instance := ss.multiManager.GetAgent(agentID)
	if instance != nil {
		info := instance.GetInfo()
		agentType = string(info.Type)
		// 从metadata获取Version
		metadata, err := ss.multiManager.GetAgentMetadata(agentID)
		if err == nil && metadata != nil {
			version = metadata.Version
		}
	}

	// 创建或更新待同步状态
	ss.pendingStates[agentID] = &AgentState{
		AgentID:       agentID,
		Status:        status,
		PID:           pid,
		LastHeartbeat: lastHeartbeat,
		Type:          agentType,
		Version:       version,
	}

	ss.logger.Debug("agent state changed, added to pending states",
		zap.String("agent_id", agentID),
		zap.String("status", string(status)),
		zap.Int("pid", pid),
		zap.String("type", agentType),
		zap.String("version", version))

	// 可选: 如果状态变化,立即触发一次同步(而不是等待定时同步)
	// 这里不立即触发,而是等待定时同步,避免频繁同步
}

// collectAgentStates 收集所有Agent的状态
func (ss *StateSyncer) collectAgentStates() []*AgentState {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	// 从multiManager获取所有Agent实例
	instances := ss.multiManager.ListAgents()
	ss.logger.Debug("collecting agent states",
		zap.Int("total_agents", len(instances)),
		zap.Int("pending_states", len(ss.pendingStates)))

	// 构建状态映射(用于合并)
	stateMap := make(map[string]*AgentState)

	// 遍历每个Agent,收集状态信息
	for _, instance := range instances {
		info := instance.GetInfo()
		agentID := info.ID

		// 从AgentInfo获取Status、PID、Type
		status := info.GetStatus()
		// 如果Status是空的（零值），使用默认值"stopped"
		if status == "" {
			status = StatusStopped
			ss.logger.Warn("agent status is empty, using default 'stopped'",
				zap.String("agent_id", agentID))
		}
		pid := info.GetPID()
		agentType := string(info.Type)

		// 从metadata获取Version
		// 使用超时避免阻塞状态同步循环
		var version string
		// last_heartbeat 使用当前时间，代表"Daemon 最后一次报告该 Agent 状态的时间"
		lastHeartbeat := time.Now()

		type metadataResult struct {
			metadata *AgentMetadata
			err      error
		}
		metadataChan := make(chan metadataResult, 1)

		go func(aid string) {
			md, err := ss.multiManager.GetAgentMetadata(aid)
			metadataChan <- metadataResult{metadata: md, err: err}
		}(agentID)

		select {
		case result := <-metadataChan:
			if result.err == nil && result.metadata != nil {
				version = result.metadata.Version
				// 注意：不再从 metadata 获取 lastHeartbeat
				// lastHeartbeat 应该是 Daemon 报告状态的时间，而不是 Agent 自己的心跳时间
			}
		case <-time.After(100 * time.Millisecond):
			// 超时后继续，version 使用空值，lastHeartbeat 使用当前时间
			ss.logger.Debug("metadata read timeout in collectAgentStates",
				zap.String("agent_id", agentID))
		}

		ss.logger.Debug("collected agent state",
			zap.String("agent_id", agentID),
			zap.String("status", string(status)),
			zap.Int("pid", pid),
			zap.String("type", agentType),
			zap.String("version", version),
			zap.Time("last_heartbeat", lastHeartbeat))

		stateMap[agentID] = &AgentState{
			AgentID:       agentID,
			Status:        status,
			PID:           pid,
			LastHeartbeat: lastHeartbeat,
			Type:          agentType,
			Version:       version,
		}
	}

	// 合并pendingStates(优先使用pendingStates中的状态)
	for agentID, pendingState := range ss.pendingStates {
		stateMap[agentID] = pendingState
	}

	// 转换为切片
	states := make([]*AgentState, 0, len(stateMap))
	for _, state := range stateMap {
		states = append(states, state)
	}

	ss.logger.Debug("collected agent states summary",
		zap.Int("collected_count", len(states)))

	return states
}

// mergeStates 合并当前状态和待同步状态
func (ss *StateSyncer) mergeStates(current []*AgentState, pending map[string]*AgentState) []*AgentState {
	// 构建状态映射
	stateMap := make(map[string]*AgentState)

	// 先添加当前状态
	for _, state := range current {
		stateMap[state.AgentID] = state
	}

	// 再添加待同步状态(优先使用待同步状态)
	for agentID, pendingState := range pending {
		stateMap[agentID] = pendingState
	}

	// 转换为切片
	states := make([]*AgentState, 0, len(stateMap))
	for _, state := range stateMap {
		states = append(states, state)
	}

	return states
}

// syncToManager 向Manager同步状态
func (ss *StateSyncer) syncToManager(states []*AgentState, nodeID string) error {
	if ss.managerClient == nil {
		return fmt.Errorf("manager client not set")
	}

	// 调用Manager的SyncAgentStates方法
	ctx, cancel := context.WithTimeout(ss.ctx, 10*time.Second)
	defer cancel()

	err := ss.managerClient.SyncAgentStates(ctx, nodeID, states)
	if err != nil {
		// 同步失败,保留在pendingStates中
		ss.mu.Lock()
		for _, state := range states {
			ss.pendingStates[state.AgentID] = state
		}
		ss.mu.Unlock()

		ss.logger.Warn("failed to sync agent states to manager",
			zap.String("node_id", nodeID),
			zap.Int("count", len(states)),
			zap.Error(err))
		return err
	}

	// 同步成功,清空pendingStates
	ss.mu.Lock()
	ss.pendingStates = make(map[string]*AgentState)
	ss.mu.Unlock()

	ss.logger.Info("synced agent states to manager",
		zap.String("node_id", nodeID),
		zap.Int("count", len(states)))

	return nil
}

// syncLoop 同步循环
func (ss *StateSyncer) syncLoop(nodeID string) {
	defer ss.wg.Done()

	ticker := time.NewTicker(ss.syncInterval)
	defer ticker.Stop()

	ss.logger.Info("state syncer loop started",
		zap.String("node_id", nodeID),
		zap.Duration("interval", ss.syncInterval))

	for {
		select {
		case <-ss.ctx.Done():
			ss.logger.Info("state syncer loop stopped")
			return
		case <-ticker.C:
			// 收集所有Agent状态
			states := ss.collectAgentStates()

			// 注意:即使状态为空也进行同步,让Manager知道这个节点当前没有Agent或Agent还未就绪
			if len(states) == 0 {
				ss.logger.Debug("no agent states to sync, but syncing to update node status")
			}

			// 同步到Manager
			if err := ss.syncToManager(states, nodeID); err != nil {
				ss.logger.Warn("sync failed, will retry on next interval",
					zap.Error(err))
			}
		}
	}
}

// Start 启动状态同步
func (ss *StateSyncer) Start(nodeID string) {
	if ss.managerClient == nil {
		ss.logger.Warn("manager client not set, state syncer will not start")
		return
	}

	ss.wg.Add(1)
	go ss.syncLoop(nodeID)

	ss.logger.Info("state syncer started",
		zap.String("node_id", nodeID),
		zap.Duration("interval", ss.syncInterval))
}

// Stop 停止状态同步
func (ss *StateSyncer) Stop() {
	ss.logger.Info("stopping state syncer")
	ss.cancel()
	ss.wg.Wait()
	ss.logger.Info("state syncer stopped")
}
