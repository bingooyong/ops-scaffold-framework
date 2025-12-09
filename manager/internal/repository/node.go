package repository

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"gorm.io/gorm"
)

// NodeRepository 节点数据访问接口
type NodeRepository interface {
	// Create 创建节点
	Create(ctx context.Context, node *model.Node) error
	// GetByID 根据ID获取节点
	GetByID(ctx context.Context, id uint) (*model.Node, error)
	// GetByNodeID 根据NodeID获取节点
	GetByNodeID(ctx context.Context, nodeID string) (*model.Node, error)
	// Update 更新节点
	Update(ctx context.Context, node *model.Node) error
	// Delete 删除节点（软删除）
	Delete(ctx context.Context, id uint) error
	// List 获取节点列表
	List(ctx context.Context, page, pageSize int) ([]*model.Node, int64, error)
	// ListByStatus 根据状态获取节点列表
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Node, int64, error)
	// ListByLabels 根据标签获取节点列表
	ListByLabels(ctx context.Context, labels map[string]string, page, pageSize int) ([]*model.Node, int64, error)
	// UpdateStatus 更新节点状态
	UpdateStatus(ctx context.Context, nodeID string, status string) error
	// UpdateHeartbeat 更新心跳时间
	UpdateHeartbeat(ctx context.Context, nodeID string) error
	// UpdateVersions 更新版本信息
	UpdateVersions(ctx context.Context, nodeID, daemonVersion, agentVersion string) error
	// GetOfflineNodes 获取离线节点（超过指定时间未心跳）
	GetOfflineNodes(ctx context.Context, duration time.Duration) ([]*model.Node, error)
	// CountByStatus 统计各状态节点数量
	CountByStatus(ctx context.Context) (map[string]int64, error)
}

// nodeRepository 节点数据访问实现
type nodeRepository struct {
	db *gorm.DB
}

// NewNodeRepository 创建节点数据访问实例
func NewNodeRepository(db *gorm.DB) NodeRepository {
	return &nodeRepository{db: db}
}

// Create 创建节点
func (r *nodeRepository) Create(ctx context.Context, node *model.Node) error {
	return r.db.WithContext(ctx).Create(node).Error
}

// GetByID 根据ID获取节点
func (r *nodeRepository) GetByID(ctx context.Context, id uint) (*model.Node, error) {
	var node model.Node
	err := r.db.WithContext(ctx).First(&node, id).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetByNodeID 根据NodeID获取节点
func (r *nodeRepository) GetByNodeID(ctx context.Context, nodeID string) (*model.Node, error) {
	var node model.Node
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// Update 更新节点
func (r *nodeRepository) Update(ctx context.Context, node *model.Node) error {
	return r.db.WithContext(ctx).Save(node).Error
}

// Delete 删除节点（软删除）
func (r *nodeRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Node{}, id).Error
}

// List 获取节点列表
func (r *nodeRepository) List(ctx context.Context, page, pageSize int) ([]*model.Node, int64, error) {
	var nodes []*model.Node
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.Node{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&nodes).Error

	return nodes, total, err
}

// ListByStatus 根据状态获取节点列表
func (r *nodeRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Node, int64, error) {
	var nodes []*model.Node
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Node{}).Where("status = ?", status)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&nodes).Error

	return nodes, total, err
}

// ListByLabels 根据标签获取节点列表
func (r *nodeRepository) ListByLabels(ctx context.Context, labels map[string]string, page, pageSize int) ([]*model.Node, int64, error) {
	var nodes []*model.Node
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Node{})

	// 构建标签查询条件
	for key, value := range labels {
		query = query.Where("JSON_EXTRACT(labels, ?) = ?", "$."+key, value)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&nodes).Error

	return nodes, total, err
}

// UpdateStatus 更新节点状态
func (r *nodeRepository) UpdateStatus(ctx context.Context, nodeID string, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.Node{}).
		Where("node_id = ?", nodeID).
		Update("status", status).
		Error
}

// UpdateHeartbeat 更新心跳时间
func (r *nodeRepository) UpdateHeartbeat(ctx context.Context, nodeID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Node{}).
		Where("node_id = ?", nodeID).
		Update("last_seen_at", now).
		Error
}

// UpdateVersions 更新版本信息
func (r *nodeRepository) UpdateVersions(ctx context.Context, nodeID, daemonVersion, agentVersion string) error {
	updates := map[string]interface{}{
		"daemon_version": daemonVersion,
		"agent_version":  agentVersion,
	}
	return r.db.WithContext(ctx).
		Model(&model.Node{}).
		Where("node_id = ?", nodeID).
		Updates(updates).
		Error
}

// GetOfflineNodes 获取离线节点（超过指定时间未心跳）
func (r *nodeRepository) GetOfflineNodes(ctx context.Context, duration time.Duration) ([]*model.Node, error) {
	var nodes []*model.Node
	cutoffTime := time.Now().Add(-duration)

	err := r.db.WithContext(ctx).
		Where("last_seen_at < ? AND status != ?", cutoffTime, "offline").
		Find(&nodes).Error

	return nodes, err
}

// CountByStatus 统计各状态节点数量
func (r *nodeRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	type StatusCount struct {
		Status string
		Count  int64
	}

	var results []StatusCount
	err := r.db.WithContext(ctx).
		Model(&model.Node{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		counts[result.Status] = result.Count
	}

	return counts, nil
}
