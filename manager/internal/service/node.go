package service

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NodeService 节点服务接口
type NodeService interface {
	// Register 注册节点
	Register(ctx context.Context, node *model.Node) error
	// GetByNodeID 根据NodeID获取节点
	GetByNodeID(ctx context.Context, nodeID string) (*model.Node, error)
	// Update 更新节点信息
	Update(ctx context.Context, node *model.Node) error
	// Delete 删除节点
	Delete(ctx context.Context, id uint) error
	// List 获取节点列表
	List(ctx context.Context, page, pageSize int) ([]*model.Node, int64, error)
	// ListByStatus 根据状态获取节点列表
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Node, int64, error)
	// ListByLabels 根据标签获取节点列表
	ListByLabels(ctx context.Context, labels map[string]string, page, pageSize int) ([]*model.Node, int64, error)
	// UpdateStatus 更新节点状态
	UpdateStatus(ctx context.Context, nodeID string, status string) error
	// Heartbeat 心跳处理
	Heartbeat(ctx context.Context, nodeID string) error
	// UpdateVersions 更新版本信息
	UpdateVersions(ctx context.Context, nodeID, daemonVersion, agentVersion string) error
	// CheckOfflineNodes 检查离线节点
	CheckOfflineNodes(ctx context.Context, offlineDuration time.Duration) error
	// GetStatistics 获取节点统计信息
	GetStatistics(ctx context.Context) (map[string]int64, error)
}

// nodeService 节点服务实现
type nodeService struct {
	nodeRepo  repository.NodeRepository
	auditRepo repository.AuditLogRepository
	logger    *zap.Logger
}

// NewNodeService 创建节点服务实例
func NewNodeService(
	nodeRepo repository.NodeRepository,
	auditRepo repository.AuditLogRepository,
	logger *zap.Logger,
) NodeService {
	return &nodeService{
		nodeRepo:  nodeRepo,
		auditRepo: auditRepo,
		logger:    logger,
	}
}

// Register 注册节点
func (s *nodeService) Register(ctx context.Context, node *model.Node) error {
	// 检查节点是否已存在
	existingNode, err := s.nodeRepo.GetByNodeID(ctx, node.NodeID)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.logger.Error("failed to check node", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	// 如果节点已存在，更新节点信息
	if existingNode != nil {
		node.ID = existingNode.ID
		node.CreatedAt = existingNode.CreatedAt
		if err := s.nodeRepo.Update(ctx, node); err != nil {
			s.logger.Error("failed to update node", zap.Error(err))
			return errors.Wrap(errors.ErrDatabase, "更新节点失败", err)
		}
		s.logger.Info("node updated", zap.String("node_id", node.NodeID))
		return nil
	}

	// 创建新节点
	if err := s.nodeRepo.Create(ctx, node); err != nil {
		s.logger.Error("failed to create node", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "创建节点失败", err)
	}

	s.logger.Info("node registered", zap.String("node_id", node.NodeID))
	return nil
}

// GetByNodeID 根据NodeID获取节点
func (s *nodeService) GetByNodeID(ctx context.Context, nodeID string) (*model.Node, error) {
	node, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrNodeNotFoundMsg
		}
		s.logger.Error("failed to get node", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	return node, nil
}

// Update 更新节点信息
func (s *nodeService) Update(ctx context.Context, node *model.Node) error {
	if err := s.nodeRepo.Update(ctx, node); err != nil {
		s.logger.Error("failed to update node", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新节点失败", err)
	}
	s.logger.Info("node updated", zap.String("node_id", node.NodeID))
	return nil
}

// Delete 删除节点
func (s *nodeService) Delete(ctx context.Context, id uint) error {
	if err := s.nodeRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete node", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "删除节点失败", err)
	}
	s.logger.Info("node deleted", zap.Uint("id", id))
	return nil
}

// List 获取节点列表
func (s *nodeService) List(ctx context.Context, page, pageSize int) ([]*model.Node, int64, error) {
	nodes, total, err := s.nodeRepo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list nodes", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询节点列表失败", err)
	}
	return nodes, total, nil
}

// ListByStatus 根据状态获取节点列表
func (s *nodeService) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Node, int64, error) {
	nodes, total, err := s.nodeRepo.ListByStatus(ctx, status, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list nodes by status", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询节点列表失败", err)
	}
	return nodes, total, nil
}

// ListByLabels 根据标签获取节点列表
func (s *nodeService) ListByLabels(ctx context.Context, labels map[string]string, page, pageSize int) ([]*model.Node, int64, error) {
	nodes, total, err := s.nodeRepo.ListByLabels(ctx, labels, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list nodes by labels", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询节点列表失败", err)
	}
	return nodes, total, nil
}

// UpdateStatus 更新节点状态
func (s *nodeService) UpdateStatus(ctx context.Context, nodeID string, status string) error {
	if err := s.nodeRepo.UpdateStatus(ctx, nodeID, status); err != nil {
		s.logger.Error("failed to update node status", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新节点状态失败", err)
	}
	s.logger.Info("node status updated", zap.String("node_id", nodeID), zap.String("status", status))
	return nil
}

// Heartbeat 心跳处理
func (s *nodeService) Heartbeat(ctx context.Context, nodeID string) error {
	// 更新心跳时间
	if err := s.nodeRepo.UpdateHeartbeat(ctx, nodeID); err != nil {
		s.logger.Error("failed to update heartbeat", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新心跳失败", err)
	}

	// 如果节点之前是离线状态，更新为在线
	node, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Warn("failed to get node for status check", zap.Error(err))
		return nil
	}

	if node.Status == "offline" {
		if err := s.nodeRepo.UpdateStatus(ctx, nodeID, "online"); err != nil {
			s.logger.Warn("failed to update node status to online", zap.Error(err))
		}
	}

	return nil
}

// UpdateVersions 更新版本信息
func (s *nodeService) UpdateVersions(ctx context.Context, nodeID, daemonVersion, agentVersion string) error {
	if err := s.nodeRepo.UpdateVersions(ctx, nodeID, daemonVersion, agentVersion); err != nil {
		s.logger.Error("failed to update versions", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新版本信息失败", err)
	}
	s.logger.Info("node versions updated",
		zap.String("node_id", nodeID),
		zap.String("daemon_version", daemonVersion),
		zap.String("agent_version", agentVersion))
	return nil
}

// CheckOfflineNodes 检查离线节点
func (s *nodeService) CheckOfflineNodes(ctx context.Context, offlineDuration time.Duration) error {
	nodes, err := s.nodeRepo.GetOfflineNodes(ctx, offlineDuration)
	if err != nil {
		s.logger.Error("failed to get offline nodes", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "查询离线节点失败", err)
	}

	// 更新节点状态为离线
	for _, node := range nodes {
		if err := s.nodeRepo.UpdateStatus(ctx, node.NodeID, "offline"); err != nil {
			s.logger.Warn("failed to update node status to offline",
				zap.String("node_id", node.NodeID),
				zap.Error(err))
			continue
		}
		s.logger.Info("node marked as offline", zap.String("node_id", node.NodeID))
	}

	return nil
}

// GetStatistics 获取节点统计信息
func (s *nodeService) GetStatistics(ctx context.Context) (map[string]int64, error) {
	stats, err := s.nodeRepo.CountByStatus(ctx)
	if err != nil {
		s.logger.Error("failed to get node statistics", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "获取统计信息失败", err)
	}
	return stats, nil
}
