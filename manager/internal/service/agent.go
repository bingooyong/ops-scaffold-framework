package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/database"
	pkgerrors "github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DaemonClient Daemon客户端接口，用于避免循环导入
type DaemonClient interface {
	OperateAgent(ctx context.Context, nodeID, agentID, operation string) error
	ListAgents(ctx context.Context, nodeID string) ([]*daemonpb.AgentInfo, error)
	GetAgentMetrics(ctx context.Context, nodeID, agentID string, duration time.Duration) ([]*daemonpb.ResourceDataPoint, error)
}

// DaemonClientPool Daemon客户端连接池接口，用于避免循环导入
type DaemonClientPool interface {
	GetClient(nodeID, address string) (DaemonClient, error)
	CloseClient(nodeID string) error
	CloseAll()
}

// AgentService Agent服务
type AgentService struct {
	agentRepo  repository.AgentRepository
	nodeRepo   repository.NodeRepository
	daemonPool DaemonClientPool
	logger     *zap.Logger
	daemonPort int // Daemon gRPC端口，默认9091
}

// NewAgentService 创建Agent服务
func NewAgentService(agentRepo repository.AgentRepository, nodeRepo repository.NodeRepository, daemonPool DaemonClientPool, logger *zap.Logger) *AgentService {
	return &AgentService{
		agentRepo:  agentRepo,
		nodeRepo:   nodeRepo,
		daemonPool: daemonPool,
		logger:     logger,
		daemonPort: 9091, // 默认Daemon gRPC端口
	}
}

// SyncAgentStates 同步Agent状态
func (s *AgentService) SyncAgentStates(ctx context.Context, nodeID string, states []*daemonpb.AgentState) error {
	if nodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	if len(states) == 0 {
		s.logger.Debug("no agent states to sync", zap.String("node_id", nodeID))
		return nil
	}

	// 使用事务确保数据一致性
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 创建临时repository使用事务
		txAgentRepo := repository.NewAgentRepository(tx)

		successCount := 0
		for _, protoState := range states {
			// 验证必要字段
			if protoState.AgentId == "" {
				s.logger.Warn("skipping agent state with empty agent_id",
					zap.String("node_id", nodeID))
				continue
			}

			// 获取现有记录(如果存在)
			existing, err := txAgentRepo.GetByNodeIDAndAgentID(ctx, nodeID, protoState.AgentId)
			if err != nil {
				s.logger.Warn("failed to get existing agent",
					zap.String("node_id", nodeID),
					zap.String("agent_id", protoState.AgentId),
					zap.Error(err))
				continue // 继续处理其他Agent
			}

			if existing != nil {
				// 更新现有记录
				agent := &model.Agent{
					NodeID:       nodeID,
					AgentID:      protoState.AgentId,
					LastSyncTime: time.Now(),
				}

				// 只更新非空的状态值，如果状态为空则保留现有状态
				if protoState.Status != "" {
					agent.Status = protoState.Status
				} else {
					// 如果状态为空，保留现有状态，并记录警告
					s.logger.Warn("agent status is empty in sync, keeping existing status",
						zap.String("node_id", nodeID),
						zap.String("agent_id", protoState.AgentId),
						zap.String("existing_status", existing.Status))
					agent.Status = existing.Status // 保留现有状态
				}

				// 只更新非零的 PID
				if protoState.Pid != 0 {
					agent.PID = int(protoState.Pid)
				} else {
					agent.PID = existing.PID // 保留现有 PID
				}

				// 如果同步状态中有Type，使用新的Type；否则保留现有的Type
				if protoState.Type != "" {
					agent.Type = protoState.Type
				} else {
					agent.Type = existing.Type
				}

				// 如果同步状态中有Version，使用新的Version；否则保留现有的Version
				if protoState.Version != "" {
					agent.Version = protoState.Version
				} else {
					agent.Version = existing.Version
				}

				// 转换LastHeartbeat时间戳
				if protoState.LastHeartbeat > 0 {
					heartbeatTime := time.Unix(protoState.LastHeartbeat, 0)
					agent.LastHeartbeat = &heartbeatTime
				} else {
					agent.LastHeartbeat = existing.LastHeartbeat // 保留现有心跳时间
				}

				if err := txAgentRepo.Update(ctx, agent); err != nil {
					s.logger.Warn("failed to update agent",
						zap.String("node_id", nodeID),
						zap.String("agent_id", protoState.AgentId),
						zap.Error(err))
					continue // 继续处理其他Agent
				}
			} else {
				// 创建新记录
				// 对于新记录，如果状态为空，使用默认值 "stopped"
				status := protoState.Status
				if status == "" {
					status = "stopped"
					s.logger.Warn("agent status is empty when creating new record, using default 'stopped'",
						zap.String("node_id", nodeID),
						zap.String("agent_id", protoState.AgentId))
				}

				agent := &model.Agent{
					NodeID:       nodeID,
					AgentID:      protoState.AgentId,
					Type:         protoState.Type,
					Version:      protoState.Version,
					Status:       status,
					PID:          int(protoState.Pid),
					LastSyncTime: time.Now(),
				}

				// 转换LastHeartbeat时间戳
				if protoState.LastHeartbeat > 0 {
					heartbeatTime := time.Unix(protoState.LastHeartbeat, 0)
					agent.LastHeartbeat = &heartbeatTime
				}

				if err := txAgentRepo.Create(ctx, agent); err != nil {
					s.logger.Warn("failed to create agent",
						zap.String("node_id", nodeID),
						zap.String("agent_id", protoState.AgentId),
						zap.Error(err))
					continue // 继续处理其他Agent
				}
			}

			successCount++
		}

		s.logger.Info("synced agent states",
			zap.String("node_id", nodeID),
			zap.Int("total", len(states)),
			zap.Int("success", successCount),
			zap.Int("failed", len(states)-successCount))

		return nil
	})

	if err != nil {
		s.logger.Error("failed to sync agent states in transaction",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return fmt.Errorf("failed to sync agent states: %w", err)
	}

	return nil
}

// ListAgents 获取节点下的所有Agent
func (s *AgentService) ListAgents(ctx context.Context, nodeID string) ([]*model.Agent, error) {
	if nodeID == "" {
		return nil, pkgerrors.New(pkgerrors.ErrInvalidParams, "node_id is required")
	}

	// 验证节点是否存在
	node, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Error("failed to get node",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return nil, pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to get node", err)
	}
	if node == nil {
		return nil, pkgerrors.ErrNodeNotFoundMsg
	}

	// 从数据库获取Agent列表
	agents, err := s.agentRepo.ListByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Error("failed to list agents",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return nil, pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to list agents", err)
	}

	s.logger.Info("list agents success",
		zap.String("node_id", nodeID),
		zap.Int("count", len(agents)))

	return agents, nil
}

// OperateAgent 操作Agent(启动/停止/重启)
func (s *AgentService) OperateAgent(ctx context.Context, nodeID, agentID, operation string) error {
	if nodeID == "" {
		return pkgerrors.New(pkgerrors.ErrInvalidParams, "node_id is required")
	}
	if agentID == "" {
		return pkgerrors.New(pkgerrors.ErrInvalidParams, "agent_id is required")
	}
	if operation == "" {
		return pkgerrors.New(pkgerrors.ErrInvalidParams, "operation is required")
	}

	// 验证操作类型
	validOperations := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
	}
	if !validOperations[operation] {
		return pkgerrors.New(pkgerrors.ErrInvalidParams, fmt.Sprintf("invalid operation: %s, must be one of: start, stop, restart", operation))
	}

	// 验证节点是否存在
	node, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Error("failed to get node",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to get node", err)
	}
	if node == nil {
		return pkgerrors.ErrNodeNotFoundMsg
	}

	// 验证Agent是否存在
	agent, err := s.agentRepo.GetByNodeIDAndAgentID(ctx, nodeID, agentID)
	if err != nil {
		s.logger.Error("failed to get agent",
			zap.String("node_id", nodeID),
			zap.String("agent_id", agentID),
			zap.Error(err))
		return pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to get agent", err)
	}
	if agent == nil {
		return pkgerrors.New(pkgerrors.ErrNotFound, "agent not found")
	}

	// 构建Daemon gRPC地址
	daemonAddr := fmt.Sprintf("%s:%d", node.IP, s.daemonPort)

	// 从连接池获取Daemon客户端
	daemonClient, err := s.daemonPool.GetClient(nodeID, daemonAddr)
	if err != nil {
		s.logger.Error("failed to get daemon client",
			zap.String("node_id", nodeID),
			zap.String("address", daemonAddr),
			zap.Error(err))
		return pkgerrors.Wrap(pkgerrors.ErrGRPC, "failed to connect to daemon", err)
	}

	// 调用Daemon的OperateAgent方法
	s.logger.Info("calling daemon OperateAgent",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.String("operation", operation),
		zap.String("daemon_address", daemonAddr))

	startTime := time.Now()
	err = daemonClient.OperateAgent(ctx, nodeID, agentID, operation)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("failed to operate agent",
			zap.String("node_id", nodeID),
			zap.String("agent_id", agentID),
			zap.String("operation", operation),
			zap.String("daemon_address", daemonAddr),
			zap.Duration("duration", duration),
			zap.Error(err))

		// 如果是连接错误，清理连接池中的连接，下次会重新建立
		if isConnectionError(err) {
			s.logger.Info("connection error detected, closing client",
				zap.String("node_id", nodeID),
				zap.String("daemon_address", daemonAddr))
			s.daemonPool.CloseClient(nodeID)
		}

		return pkgerrors.Wrap(pkgerrors.ErrGRPC, "failed to operate agent", err)
	}

	s.logger.Info("operate agent success",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.String("operation", operation),
		zap.String("daemon_address", daemonAddr),
		zap.Duration("duration", duration))

	// 操作成功后，立即更新数据库中的状态
	// 这样用户可以立即看到操作结果，不需要等待心跳同步
	expectedStatus := getExpectedStatusAfterOperation(operation)
	if expectedStatus != "" {
		now := time.Now()
		updateAgent := &model.Agent{
			NodeID:       nodeID,
			AgentID:      agentID,
			Status:       expectedStatus,
			LastSyncTime: now,
		}

		if err := s.agentRepo.Update(ctx, updateAgent); err != nil {
			s.logger.Warn("failed to update agent status immediately after operation",
				zap.String("node_id", nodeID),
				zap.String("agent_id", agentID),
				zap.String("operation", operation),
				zap.String("expected_status", expectedStatus),
				zap.Error(err))
			// 不返回错误，操作已成功，状态更新失败不影响操作结果
		} else {
			s.logger.Info("updated agent status immediately after operation",
				zap.String("node_id", nodeID),
				zap.String("agent_id", agentID),
				zap.String("operation", operation),
				zap.String("new_status", expectedStatus))
		}

		// 如果是停止操作，需要额外清空PID（因为Update方法不会更新零值）
		if operation == "stop" {
			if err := s.agentRepo.ClearPID(ctx, nodeID, agentID); err != nil {
				s.logger.Warn("failed to clear agent PID after stop operation",
					zap.String("node_id", nodeID),
					zap.String("agent_id", agentID),
					zap.Error(err))
			}
		}
	}

	// 注意：不再在操作后立即调用 ListAgents gRPC 来同步状态
	// 原因：
	// 1. Agent 状态应该由 Daemon 的定时心跳上报来更新数据库
	// 2. 操作成功后，我们已经更新了数据库中的预期状态
	// 3. 如果需要获取最新的实时状态，前端可以调用手动同步接口
	// 4. 这样可以保证架构的一致性，避免状态不同步的问题

	return nil
}

// isConnectionError 判断是否是连接错误
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "unavailable")
}

// GetAgentLogs 获取Agent日志
func (s *AgentService) GetAgentLogs(ctx context.Context, nodeID, agentID string, lines int) ([]string, error) {
	if nodeID == "" {
		return nil, pkgerrors.New(pkgerrors.ErrInvalidParams, "node_id is required")
	}
	if agentID == "" {
		return nil, pkgerrors.New(pkgerrors.ErrInvalidParams, "agent_id is required")
	}
	if lines <= 0 {
		lines = 100 // 默认100行
	}
	if lines > 1000 {
		lines = 1000 // 限制最大1000行
	}

	// 验证节点是否存在
	node, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Error("failed to get node",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return nil, pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to get node", err)
	}
	if node == nil {
		return nil, pkgerrors.ErrNodeNotFoundMsg
	}

	// 验证Agent是否存在
	agent, err := s.agentRepo.GetByNodeIDAndAgentID(ctx, nodeID, agentID)
	if err != nil {
		s.logger.Error("failed to get agent",
			zap.String("node_id", nodeID),
			zap.String("agent_id", agentID),
			zap.Error(err))
		return nil, pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to get agent", err)
	}
	if agent == nil {
		return nil, pkgerrors.New(pkgerrors.ErrNotFound, "agent not found")
	}

	// 功能未实现，返回错误
	// TODO: 实现获取Agent日志的功能
	return nil, pkgerrors.New(pkgerrors.ErrInternalServer, "get agent logs not implemented yet")
}

// GetAgentMetrics 获取Agent资源使用指标
func (s *AgentService) GetAgentMetrics(ctx context.Context, nodeID, agentID string, duration time.Duration) ([]*daemonpb.ResourceDataPoint, error) {
	if nodeID == "" {
		return nil, pkgerrors.New(pkgerrors.ErrInvalidParams, "node_id is required")
	}
	if agentID == "" {
		return nil, pkgerrors.New(pkgerrors.ErrInvalidParams, "agent_id is required")
	}
	if duration <= 0 {
		return nil, pkgerrors.New(pkgerrors.ErrInvalidParams, "duration must be greater than 0")
	}

	// 验证节点是否存在
	node, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Error("failed to get node",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return nil, pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to get node", err)
	}
	if node == nil {
		return nil, pkgerrors.ErrNodeNotFoundMsg
	}

	// 验证Agent是否存在
	agent, err := s.agentRepo.GetByNodeIDAndAgentID(ctx, nodeID, agentID)
	if err != nil {
		s.logger.Error("failed to get agent",
			zap.String("node_id", nodeID),
			zap.String("agent_id", agentID),
			zap.Error(err))
		return nil, pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to get agent", err)
	}
	if agent == nil {
		return nil, pkgerrors.New(pkgerrors.ErrNotFound, "agent not found")
	}

	// 构建Daemon gRPC地址
	daemonAddr := fmt.Sprintf("%s:%d", node.IP, s.daemonPort)

	// 从连接池获取Daemon客户端
	daemonClient, err := s.daemonPool.GetClient(nodeID, daemonAddr)
	if err != nil {
		s.logger.Error("failed to get daemon client",
			zap.String("node_id", nodeID),
			zap.String("address", daemonAddr),
			zap.Error(err))
		return nil, pkgerrors.Wrap(pkgerrors.ErrGRPC, "failed to connect to daemon", err)
	}

	// 调用Daemon的GetAgentMetrics方法
	s.logger.Debug("calling daemon GetAgentMetrics",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.Duration("duration", duration),
		zap.String("daemon_address", daemonAddr))

	startTime := time.Now()
	dataPoints, err := daemonClient.GetAgentMetrics(ctx, nodeID, agentID, duration)
	durationElapsed := time.Since(startTime)

	if err != nil {
		s.logger.Warn("failed to get agent metrics",
			zap.String("node_id", nodeID),
			zap.String("agent_id", agentID),
			zap.Duration("duration", duration),
			zap.String("daemon_address", daemonAddr),
			zap.Duration("duration_elapsed", durationElapsed),
			zap.Error(err))

		// 如果是连接错误，清理连接池中的连接，下次会重新建立
		if isConnectionError(err) {
			s.logger.Info("connection error detected, closing client",
				zap.String("node_id", nodeID),
				zap.String("daemon_address", daemonAddr))
			s.daemonPool.CloseClient(nodeID)
		}

		return nil, pkgerrors.Wrap(pkgerrors.ErrGRPC, "failed to get agent metrics", err)
	}

	s.logger.Info("get agent metrics success",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.Duration("duration", duration),
		zap.String("daemon_address", daemonAddr),
		zap.Int("data_points_count", len(dataPoints)),
		zap.Duration("duration_elapsed", durationElapsed))

	return dataPoints, nil
}

// SyncAgentStatesFromDaemon 从Daemon手动同步Agent状态到数据库
// 此方法用于前端手动触发同步，获取最新的Agent状态
func (s *AgentService) SyncAgentStatesFromDaemon(ctx context.Context, nodeID string) (int, error) {
	if nodeID == "" {
		return 0, pkgerrors.New(pkgerrors.ErrInvalidParams, "node_id is required")
	}

	// 验证节点是否存在
	node, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Error("failed to get node",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return 0, pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to get node", err)
	}
	if node == nil {
		return 0, pkgerrors.ErrNodeNotFoundMsg
	}

	// 构建Daemon gRPC地址
	daemonAddr := fmt.Sprintf("%s:%d", node.IP, s.daemonPort)

	// 从连接池获取Daemon客户端
	daemonClient, err := s.daemonPool.GetClient(nodeID, daemonAddr)
	if err != nil {
		s.logger.Error("failed to get daemon client",
			zap.String("node_id", nodeID),
			zap.String("address", daemonAddr),
			zap.Error(err))
		return 0, pkgerrors.Wrap(pkgerrors.ErrGRPC, "failed to connect to daemon", err)
	}

	// 调用Daemon的ListAgents方法获取最新状态
	s.logger.Info("manually syncing agent states from daemon",
		zap.String("node_id", nodeID),
		zap.String("daemon_address", daemonAddr))

	startTime := time.Now()
	agentInfos, err := daemonClient.ListAgents(ctx, nodeID)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("failed to list agents from daemon",
			zap.String("node_id", nodeID),
			zap.String("daemon_address", daemonAddr),
			zap.Duration("duration", duration),
			zap.Error(err))

		// 如果是连接错误，清理连接池中的连接
		if isConnectionError(err) {
			s.logger.Info("connection error detected, closing client",
				zap.String("node_id", nodeID),
				zap.String("daemon_address", daemonAddr))
			s.daemonPool.CloseClient(nodeID)
		}

		return 0, pkgerrors.Wrap(pkgerrors.ErrGRPC, "failed to get agent states from daemon", err)
	}

	s.logger.Info("got agent states from daemon",
		zap.String("node_id", nodeID),
		zap.Int("count", len(agentInfos)),
		zap.Duration("duration", duration))

	// 转换为AgentState格式
	states := make([]*daemonpb.AgentState, 0, len(agentInfos))
	for _, info := range agentInfos {
		s.logger.Debug("converting agent info to state",
			zap.String("agent_id", info.Id),
			zap.String("status", info.Status),
			zap.Int32("pid", info.Pid))

		state := &daemonpb.AgentState{
			AgentId:       info.Id,
			Type:          info.Type,
			Version:       info.Version,
			Status:        info.Status,
			Pid:           info.Pid,
			LastHeartbeat: info.LastHeartbeat,
		}
		states = append(states, state)
	}

	// 同步状态到数据库
	if err := s.SyncAgentStates(ctx, nodeID, states); err != nil {
		s.logger.Error("failed to sync agent states to database",
			zap.String("node_id", nodeID),
			zap.Int("count", len(states)),
			zap.Error(err))
		return 0, pkgerrors.Wrap(pkgerrors.ErrDatabase, "failed to sync agent states", err)
	}

	s.logger.Info("manually synced agent states successfully",
		zap.String("node_id", nodeID),
		zap.Int("synced_count", len(states)),
		zap.Duration("total_duration", time.Since(startTime)))

	return len(states), nil
}

// getExpectedStatusAfterOperation 根据操作类型返回预期的状态
func getExpectedStatusAfterOperation(operation string) string {
	switch strings.ToLower(operation) {
	case "start":
		return "running"
	case "stop":
		return "stopped"
	case "restart":
		return "running"
	default:
		return ""
	}
}
