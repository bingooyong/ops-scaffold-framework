package repository

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"gorm.io/gorm"
)

// MetricsRepository 监控指标数据访问接口
type MetricsRepository interface {
	// Create 创建指标记录
	Create(ctx context.Context, metrics *model.Metrics) error
	// BatchCreate 批量创建指标记录
	BatchCreate(ctx context.Context, metrics []*model.Metrics) error
	// GetByID 根据ID获取指标
	GetByID(ctx context.Context, id uint) (*model.Metrics, error)
	// List 获取指标列表
	List(ctx context.Context, page, pageSize int) ([]*model.Metrics, int64, error)
	// ListByNodeID 根据节点ID获取指标列表
	ListByNodeID(ctx context.Context, nodeID string, page, pageSize int) ([]*model.Metrics, int64, error)
	// ListByNodeIDAndType 根据节点ID和类型获取指标列表
	ListByNodeIDAndType(ctx context.Context, nodeID, metricsType string, page, pageSize int) ([]*model.Metrics, int64, error)
	// ListByTimeRange 根据时间范围获取指标
	ListByTimeRange(ctx context.Context, nodeID string, metricsType string, start, end time.Time) ([]*model.Metrics, error)
	// GetLatestByNodeID 获取节点最新指标
	GetLatestByNodeID(ctx context.Context, nodeID string) ([]*model.Metrics, error)
	// GetLatestByNodeIDAndType 获取节点指定类型的最新指标
	GetLatestByNodeIDAndType(ctx context.Context, nodeID, metricsType string) (*model.Metrics, error)
	// DeleteOlderThan 删除指定时间之前的指标（用于数据清理）
	DeleteOlderThan(ctx context.Context, duration time.Duration) (int64, error)
	// GetAverageByNodeIDAndType 获取指定时间范围内的平均值
	GetAverageByNodeIDAndType(ctx context.Context, nodeID, metricsType string, start, end time.Time) (map[string]float64, error)
}

// metricsRepository 监控指标数据访问实现
type metricsRepository struct {
	db *gorm.DB
}

// NewMetricsRepository 创建监控指标数据访问实例
func NewMetricsRepository(db *gorm.DB) MetricsRepository {
	return &metricsRepository{db: db}
}

// Create 创建指标记录
func (r *metricsRepository) Create(ctx context.Context, metrics *model.Metrics) error {
	return r.db.WithContext(ctx).Create(metrics).Error
}

// BatchCreate 批量创建指标记录
func (r *metricsRepository) BatchCreate(ctx context.Context, metrics []*model.Metrics) error {
	return r.db.WithContext(ctx).Create(&metrics).Error
}

// GetByID 根据ID获取指标
func (r *metricsRepository) GetByID(ctx context.Context, id uint) (*model.Metrics, error) {
	var metrics model.Metrics
	err := r.db.WithContext(ctx).First(&metrics, id).Error
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

// List 获取指标列表
func (r *metricsRepository) List(ctx context.Context, page, pageSize int) ([]*model.Metrics, int64, error) {
	var metrics []*model.Metrics
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.Metrics{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(pageSize).
		Order("timestamp DESC").
		Find(&metrics).Error

	return metrics, total, err
}

// ListByNodeID 根据节点ID获取指标列表
func (r *metricsRepository) ListByNodeID(ctx context.Context, nodeID string, page, pageSize int) ([]*model.Metrics, int64, error) {
	var metrics []*model.Metrics
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Metrics{}).Where("node_id = ?", nodeID)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("timestamp DESC").
		Find(&metrics).Error

	return metrics, total, err
}

// ListByNodeIDAndType 根据节点ID和类型获取指标列表
func (r *metricsRepository) ListByNodeIDAndType(ctx context.Context, nodeID, metricsType string, page, pageSize int) ([]*model.Metrics, int64, error) {
	var metrics []*model.Metrics
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Metrics{}).
		Where("node_id = ? AND type = ?", nodeID, metricsType)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("timestamp DESC").
		Find(&metrics).Error

	return metrics, total, err
}

// ListByTimeRange 根据时间范围获取指标
func (r *metricsRepository) ListByTimeRange(ctx context.Context, nodeID string, metricsType string, start, end time.Time) ([]*model.Metrics, error) {
	var metrics []*model.Metrics

	query := r.db.WithContext(ctx).
		Where("node_id = ? AND timestamp BETWEEN ? AND ?", nodeID, start, end)

	if metricsType != "" {
		query = query.Where("type = ?", metricsType)
	}

	err := query.Order("timestamp ASC").Find(&metrics).Error
	return metrics, err
}

// GetLatestByNodeID 获取节点最新指标
func (r *metricsRepository) GetLatestByNodeID(ctx context.Context, nodeID string) ([]*model.Metrics, error) {
	var metrics []*model.Metrics

	// 获取每种类型的最新指标
	err := r.db.WithContext(ctx).Raw(`
		SELECT m1.*
		FROM metrics m1
		INNER JOIN (
			SELECT node_id, type, MAX(timestamp) as max_timestamp
			FROM metrics
			WHERE node_id = ?
			GROUP BY node_id, type
		) m2 ON m1.node_id = m2.node_id
			AND m1.type = m2.type
			AND m1.timestamp = m2.max_timestamp
	`, nodeID).Scan(&metrics).Error

	return metrics, err
}

// GetLatestByNodeIDAndType 获取节点指定类型的最新指标
func (r *metricsRepository) GetLatestByNodeIDAndType(ctx context.Context, nodeID, metricsType string) (*model.Metrics, error) {
	var metrics model.Metrics

	err := r.db.WithContext(ctx).
		Where("node_id = ? AND type = ?", nodeID, metricsType).
		Order("timestamp DESC").
		First(&metrics).Error

	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

// DeleteOlderThan 删除指定时间之前的指标（用于数据清理）
func (r *metricsRepository) DeleteOlderThan(ctx context.Context, duration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-duration)

	result := r.db.WithContext(ctx).
		Where("timestamp < ?", cutoffTime).
		Delete(&model.Metrics{})

	return result.RowsAffected, result.Error
}

// GetAverageByNodeIDAndType 获取指定时间范围内的平均值
func (r *metricsRepository) GetAverageByNodeIDAndType(ctx context.Context, nodeID, metricsType string, start, end time.Time) (map[string]float64, error) {
	// 这个方法需要根据具体的指标类型和values字段结构来实现
	// 这里提供一个基本的框架，实际使用时需要根据JSON字段的具体结构来解析
	var metrics []*model.Metrics

	err := r.db.WithContext(ctx).
		Where("node_id = ? AND type = ? AND timestamp BETWEEN ? AND ?", nodeID, metricsType, start, end).
		Find(&metrics).Error

	if err != nil {
		return nil, err
	}

	// 计算平均值
	averages := make(map[string]float64)
	counts := make(map[string]int)

	for _, m := range metrics {
		for key, value := range m.Values {
			if val, ok := value.(float64); ok {
				averages[key] += val
				counts[key]++
			}
		}
	}

	// 计算平均值
	for key := range averages {
		if counts[key] > 0 {
			averages[key] /= float64(counts[key])
		}
	}

	return averages, nil
}
