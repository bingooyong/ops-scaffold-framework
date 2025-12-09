package repository

import (
	"context"
	"fmt"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"gorm.io/gorm"
)

// AgentRepository Agent数据仓库接口
type AgentRepository interface {
	// GetByID 根据ID获取Agent
	GetByID(ctx context.Context, id uint) (*model.Agent, error)

	// GetByNodeIDAndAgentID 根据节点ID和AgentID获取Agent
	GetByNodeIDAndAgentID(ctx context.Context, nodeID, agentID string) (*model.Agent, error)

	// Create 创建Agent
	Create(ctx context.Context, agent *model.Agent) error

	// Update 更新Agent
	Update(ctx context.Context, agent *model.Agent) error

	// Upsert 创建或更新Agent(如果不存在则创建,存在则更新)
	Upsert(ctx context.Context, agent *model.Agent) error

	// ListByNodeID 根据节点ID列举所有Agent
	ListByNodeID(ctx context.Context, nodeID string) ([]*model.Agent, error)

	// Delete 删除Agent
	Delete(ctx context.Context, nodeID, agentID string) error

	// ClearPID 清空Agent的PID（用于停止操作）
	ClearPID(ctx context.Context, nodeID, agentID string) error
}

// agentRepository Agent数据仓库实现
type agentRepository struct {
	db *gorm.DB
}

// NewAgentRepository 创建Agent数据仓库
func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &agentRepository{
		db: db,
	}
}

// GetByID 根据ID获取Agent
func (r *agentRepository) GetByID(ctx context.Context, id uint) (*model.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).First(&agent, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get agent by ID: %w", err)
	}
	return &agent, nil
}

// GetByNodeIDAndAgentID 根据节点ID和AgentID获取Agent
func (r *agentRepository) GetByNodeIDAndAgentID(ctx context.Context, nodeID, agentID string) (*model.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).
		Where("node_id = ? AND agent_id = ?", nodeID, agentID).
		First(&agent).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return &agent, nil
}

// Create 创建Agent
func (r *agentRepository) Create(ctx context.Context, agent *model.Agent) error {
	if err := r.db.WithContext(ctx).Create(agent).Error; err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	return nil
}

// Update 更新Agent
func (r *agentRepository) Update(ctx context.Context, agent *model.Agent) error {
	// 使用 node_id 和 agent_id 作为更新条件，确保只更新指定的 Agent
	// 只更新非零值字段，避免将空字符串写入数据库
	updates := map[string]interface{}{
		"last_sync_time": agent.LastSyncTime,
	}
	
	// 只更新非空的状态
	if agent.Status != "" {
		updates["status"] = agent.Status
	}
	
	// 只更新非零的 PID
	if agent.PID != 0 {
		updates["pid"] = agent.PID
	}
	
	// 只更新非空的 Type
	if agent.Type != "" {
		updates["type"] = agent.Type
	}
	
	// 只更新非空的 Version
	if agent.Version != "" {
		updates["version"] = agent.Version
	}
	
	// 更新 LastHeartbeat（如果设置了）
	if agent.LastHeartbeat != nil {
		updates["last_heartbeat"] = agent.LastHeartbeat
	}
	
	// 使用 node_id 和 agent_id 作为 WHERE 条件，确保只更新指定的 Agent
	result := r.db.WithContext(ctx).
		Model(&model.Agent{}).
		Where("node_id = ? AND agent_id = ?", agent.NodeID, agent.AgentID).
		Updates(updates)
	
	if result.Error != nil {
		return fmt.Errorf("failed to update agent: %w", result.Error)
	}
	
	// 检查是否真的更新了记录
	if result.RowsAffected == 0 {
		return fmt.Errorf("agent not found: node_id=%s, agent_id=%s", agent.NodeID, agent.AgentID)
	}
	
	return nil
}

// Upsert 创建或更新Agent(如果不存在则创建,存在则更新)
func (r *agentRepository) Upsert(ctx context.Context, agent *model.Agent) error {
	// 先尝试查找
	existing, err := r.GetByNodeIDAndAgentID(ctx, agent.NodeID, agent.AgentID)
	if err != nil {
		return err
	}

	if existing == nil {
		// 不存在,创建
		return r.Create(ctx, agent)
	}

	// 存在,更新
	agent.ID = existing.ID
	agent.CreatedAt = existing.CreatedAt
	return r.Update(ctx, agent)
}

// ListByNodeID 根据节点ID列举所有Agent
func (r *agentRepository) ListByNodeID(ctx context.Context, nodeID string) ([]*model.Agent, error) {
	var agents []*model.Agent
	err := r.db.WithContext(ctx).
		Where("node_id = ?", nodeID).
		Find(&agents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// Delete 删除Agent
func (r *agentRepository) Delete(ctx context.Context, nodeID, agentID string) error {
	err := r.db.WithContext(ctx).
		Where("node_id = ? AND agent_id = ?", nodeID, agentID).
		Delete(&model.Agent{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// ClearPID 清空Agent的PID（用于停止操作）
func (r *agentRepository) ClearPID(ctx context.Context, nodeID, agentID string) error {
	result := r.db.WithContext(ctx).
		Model(&model.Agent{}).
		Where("node_id = ? AND agent_id = ?", nodeID, agentID).
		Update("pid", 0)

	if result.Error != nil {
		return fmt.Errorf("failed to clear agent PID: %w", result.Error)
	}

	return nil
}
