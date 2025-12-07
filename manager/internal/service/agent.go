package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/database"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AgentService Agent服务
type AgentService struct {
	agentRepo repository.AgentRepository
	logger    *zap.Logger
}

// NewAgentService 创建Agent服务
func NewAgentService(agentRepo repository.AgentRepository, logger *zap.Logger) *AgentService {
	return &AgentService{
		agentRepo: agentRepo,
		logger:    logger,
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
			// 转换protobuf AgentState为业务对象
			agent := &model.Agent{
				NodeID:       nodeID,
				AgentID:      protoState.AgentId,
				Type:         "", // Agent类型从首次创建时获取,这里不更新
				Status:       protoState.Status,
				PID:          int(protoState.Pid),
				LastSyncTime: time.Now(),
			}

			// 转换LastHeartbeat时间戳
			if protoState.LastHeartbeat > 0 {
				heartbeatTime := time.Unix(protoState.LastHeartbeat, 0)
				agent.LastHeartbeat = &heartbeatTime
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
				agent.ID = existing.ID
				agent.CreatedAt = existing.CreatedAt
				// 保留Type(如果现有记录有Type)
				if existing.Type != "" {
					agent.Type = existing.Type
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
