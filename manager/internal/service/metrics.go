package service

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"go.uber.org/zap"
)

// MetricsService 监控指标服务接口
type MetricsService interface {
	// Create 创建指标记录
	Create(ctx context.Context, metrics *model.Metrics) error
	// BatchCreate 批量创建指标记录
	BatchCreate(ctx context.Context, metrics []*model.Metrics) error
	// GetLatestByNodeID 获取节点最新指标
	GetLatestByNodeID(ctx context.Context, nodeID string) ([]*model.Metrics, error)
	// GetLatestByNodeIDAndType 获取节点指定类型的最新指标
	GetLatestByNodeIDAndType(ctx context.Context, nodeID, metricsType string) (*model.Metrics, error)
	// ListByNodeID 根据节点ID获取指标列表
	ListByNodeID(ctx context.Context, nodeID string, page, pageSize int) ([]*model.Metrics, int64, error)
	// ListByTimeRange 根据时间范围获取指标
	ListByTimeRange(ctx context.Context, nodeID string, metricsType string, start, end time.Time) ([]*model.Metrics, error)
	// GetAverageByNodeIDAndType 获取指定时间范围内的平均值
	GetAverageByNodeIDAndType(ctx context.Context, nodeID, metricsType string, start, end time.Time) (map[string]float64, error)
	// CleanOldMetrics 清理旧指标数据
	CleanOldMetrics(ctx context.Context, retentionDays int) (int64, error)
}

// metricsService 监控指标服务实现
type metricsService struct {
	metricsRepo repository.MetricsRepository
	logger      *zap.Logger
}

// NewMetricsService 创建监控指标服务实例
func NewMetricsService(
	metricsRepo repository.MetricsRepository,
	logger *zap.Logger,
) MetricsService {
	return &metricsService{
		metricsRepo: metricsRepo,
		logger:      logger,
	}
}

// Create 创建指标记录
func (s *metricsService) Create(ctx context.Context, metrics *model.Metrics) error {
	if err := s.metricsRepo.Create(ctx, metrics); err != nil {
		s.logger.Error("failed to create metrics", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "创建指标失败", err)
	}
	return nil
}

// BatchCreate 批量创建指标记录
func (s *metricsService) BatchCreate(ctx context.Context, metrics []*model.Metrics) error {
	if err := s.metricsRepo.BatchCreate(ctx, metrics); err != nil {
		s.logger.Error("failed to batch create metrics", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "批量创建指标失败", err)
	}
	s.logger.Debug("batch created metrics", zap.Int("count", len(metrics)))
	return nil
}

// GetLatestByNodeID 获取节点最新指标
func (s *metricsService) GetLatestByNodeID(ctx context.Context, nodeID string) ([]*model.Metrics, error) {
	metrics, err := s.metricsRepo.GetLatestByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Error("failed to get latest metrics", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "查询最新指标失败", err)
	}
	return metrics, nil
}

// GetLatestByNodeIDAndType 获取节点指定类型的最新指标
func (s *metricsService) GetLatestByNodeIDAndType(ctx context.Context, nodeID, metricsType string) (*model.Metrics, error) {
	metrics, err := s.metricsRepo.GetLatestByNodeIDAndType(ctx, nodeID, metricsType)
	if err != nil {
		s.logger.Error("failed to get latest metrics by type", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "查询最新指标失败", err)
	}
	return metrics, nil
}

// ListByNodeID 根据节点ID获取指标列表
func (s *metricsService) ListByNodeID(ctx context.Context, nodeID string, page, pageSize int) ([]*model.Metrics, int64, error) {
	metrics, total, err := s.metricsRepo.ListByNodeID(ctx, nodeID, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list metrics", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询指标列表失败", err)
	}
	return metrics, total, nil
}

// ListByTimeRange 根据时间范围获取指标
func (s *metricsService) ListByTimeRange(ctx context.Context, nodeID string, metricsType string, start, end time.Time) ([]*model.Metrics, error) {
	metrics, err := s.metricsRepo.ListByTimeRange(ctx, nodeID, metricsType, start, end)
	if err != nil {
		s.logger.Error("failed to list metrics by time range", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "查询指标失败", err)
	}
	return metrics, nil
}

// GetAverageByNodeIDAndType 获取指定时间范围内的平均值
func (s *metricsService) GetAverageByNodeIDAndType(ctx context.Context, nodeID, metricsType string, start, end time.Time) (map[string]float64, error) {
	averages, err := s.metricsRepo.GetAverageByNodeIDAndType(ctx, nodeID, metricsType, start, end)
	if err != nil {
		s.logger.Error("failed to get average metrics", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "计算平均值失败", err)
	}
	return averages, nil
}

// CleanOldMetrics 清理旧指标数据
func (s *metricsService) CleanOldMetrics(ctx context.Context, retentionDays int) (int64, error) {
	duration := time.Duration(retentionDays) * 24 * time.Hour
	deleted, err := s.metricsRepo.DeleteOlderThan(ctx, duration)
	if err != nil {
		s.logger.Error("failed to clean old metrics", zap.Error(err))
		return 0, errors.Wrap(errors.ErrDatabase, "清理旧指标失败", err)
	}
	s.logger.Info("cleaned old metrics", zap.Int64("deleted", deleted))
	return deleted, nil
}
