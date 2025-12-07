package repository

import (
	"context"
	"fmt"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"gorm.io/gorm"
)

// AgentRepository Agent数据仓库接口
type AgentRepository interface {
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
	if err := r.db.WithContext(ctx).Save(agent).Error; err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
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
