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
	// GetLatestMetricsByNodeID 获取节点所有类型的最新指标（返回 map[string]*model.Metrics）
	GetLatestMetricsByNodeID(ctx context.Context, nodeID string) (map[string]*model.Metrics, error)
	// GetMetricsHistoryWithSampling 获取历史指标数据（带采样策略）
	GetMetricsHistoryWithSampling(ctx context.Context, nodeID string, metricType string, startTime, endTime time.Time) ([]*model.Metrics, error)
	// GetMetricsSummaryStats 获取指标统计摘要（min/max/avg/latest）
	GetMetricsSummaryStats(ctx context.Context, nodeID string, startTime, endTime time.Time) (map[string]interface{}, error)
	// GetClusterOverview 获取集群资源概览
	GetClusterOverview(ctx context.Context) (map[string]interface{}, error)
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

// GetLatestMetricsByNodeID 获取节点所有类型的最新指标
// 返回 map[string]*model.Metrics，key 为指标类型（cpu/memory/disk/network）
func (s *metricsService) GetLatestMetricsByNodeID(ctx context.Context, nodeID string) (map[string]*model.Metrics, error) {
	result := make(map[string]*model.Metrics)
	metricTypes := []string{"cpu", "memory", "disk", "network"}

	// 对每种指标类型获取最新记录
	for _, metricType := range metricTypes {
		metrics, err := s.metricsRepo.GetLatestByNodeIDAndType(ctx, nodeID, metricType)
		if err != nil {
			// 如果记录不存在（gorm.ErrRecordNotFound），该类型返回 nil
			// 其他错误记录日志但继续处理其他类型
			if err == gorm.ErrRecordNotFound {
				result[metricType] = nil
				continue
			}
			s.logger.Warn("failed to get latest metrics by type",
				zap.String("node_id", nodeID),
				zap.String("type", metricType),
				zap.Error(err))
			result[metricType] = nil
			continue
		}
		result[metricType] = metrics
	}

	return result, nil
}

// GetMetricsHistoryWithSampling 获取历史指标数据（带采样策略）
// 根据时间范围自动选择采样间隔，确保数据点数量适中（约 300-400 点）
func (s *metricsService) GetMetricsHistoryWithSampling(ctx context.Context, nodeID string, metricType string, startTime, endTime time.Time) ([]*model.Metrics, error) {
	// 验证时间范围不超过 30 天
	maxDuration := 30 * 24 * time.Hour
	duration := endTime.Sub(startTime)
	if duration > maxDuration {
		return nil, errors.New(errors.ErrInvalidParams, "时间范围不能超过 30 天")
	}

	if duration <= 0 {
		return nil, errors.New(errors.ErrInvalidParams, "结束时间必须大于开始时间")
	}

	// 根据时间跨度决定采样间隔
	var interval time.Duration
	switch {
	case duration <= 15*time.Minute:
		// <= 15 分钟：返回原始数据（60 秒间隔）
		interval = 0
	case duration <= 1*time.Hour:
		// <= 1 小时：返回原始数据（60 秒间隔）
		interval = 0
	case duration <= 24*time.Hour:
		// <= 1 天：使用 5 分钟聚合
		interval = 5 * time.Minute
	case duration <= 7*24*time.Hour:
		// <= 7 天：使用 30 分钟聚合
		interval = 30 * time.Minute
	case duration <= 30*24*time.Hour:
		// <= 30 天：使用 2 小时聚合
		interval = 2 * time.Hour
	default:
		// 超过 30 天已经在上面验证中返回错误
		interval = 2 * time.Hour
	}

	// 调用 Repository 层方法获取采样后的数据
	metrics, err := s.metricsRepo.ListByTimeRangeWithInterval(ctx, nodeID, metricType, startTime, endTime, interval)
	if err != nil {
		s.logger.Error("failed to get metrics history with sampling",
			zap.String("node_id", nodeID),
			zap.String("type", metricType),
			zap.Duration("interval", interval),
			zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "查询历史指标失败", err)
	}

	s.logger.Debug("retrieved metrics history with sampling",
		zap.String("node_id", nodeID),
		zap.String("type", metricType),
		zap.Duration("duration", duration),
		zap.Duration("interval", interval),
		zap.Int("count", len(metrics)))

	return metrics, nil
}

// GetMetricsSummaryStats 获取指标统计摘要（min/max/avg/latest）
// 返回每种指标类型的统计信息
func (s *metricsService) GetMetricsSummaryStats(ctx context.Context, nodeID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	metricTypes := []string{"cpu", "memory", "disk", "network"}

	// 获取所有类型的最新指标（用于 latest 值）
	latestMetrics, err := s.GetLatestMetricsByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Warn("failed to get latest metrics for summary",
			zap.String("node_id", nodeID),
			zap.Error(err))
		// 继续处理，latest 值可能为空
	}

	// 对每种指标类型获取聚合统计
	for _, metricType := range metricTypes {
		stats := make(map[string]interface{})

		// 获取聚合统计（min/max/avg）
		min, max, avg, err := s.metricsRepo.GetAggregateStats(ctx, nodeID, metricType, startTime, endTime)
		if err != nil {
			s.logger.Warn("failed to get aggregate stats",
				zap.String("node_id", nodeID),
				zap.String("type", metricType),
				zap.Error(err))
			// 该类型无数据，设置为 null
			result[metricType] = nil
			continue
		}

		// 如果 min/max/avg 都是 0，可能是空数据
		if min == 0 && max == 0 && avg == 0 {
			// 检查是否有最新数据
			if latestMetrics[metricType] != nil {
				// 有最新数据但统计为空，可能是时间范围内无数据
				stats["min"] = nil
				stats["max"] = nil
				stats["avg"] = nil
			} else {
				// 完全无数据
				result[metricType] = nil
				continue
			}
		} else {
			stats["min"] = min
			stats["max"] = max
			stats["avg"] = avg
		}

		// 获取 latest 值
		if latestMetrics[metricType] != nil {
			// 从 values 中提取 usage_percent
			if usagePercent, ok := latestMetrics[metricType].Values["usage_percent"].(float64); ok {
				stats["latest"] = usagePercent
			} else {
				stats["latest"] = nil
			}
		} else {
			stats["latest"] = nil
		}

		result[metricType] = stats
	}

	return result, nil
}

// GetClusterOverview 获取集群资源概览
func (s *metricsService) GetClusterOverview(ctx context.Context) (map[string]interface{}, error) {
	// 获取所有节点的最新指标数据
	nodeMetrics, err := s.metricsRepo.GetAllNodesLatestMetrics(ctx)
	if err != nil {
		s.logger.Error("failed to get all nodes latest metrics", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "获取集群概览数据失败", err)
	}

	// 计算聚合统计
	var totalNodes, onlineNodes, offlineNodes int
	var totalCPUUsage, totalMemoryUsage, totalDiskUsage float64
	var totalMemoryBytes, totalDiskBytes float64

	// 用于存储节点列表（包含详细指标）
	nodesList := make([]map[string]interface{}, 0, len(nodeMetrics))

	for _, nm := range nodeMetrics {
		totalNodes++

		// 统计在线/离线节点数
		if nm.Status == "online" {
			onlineNodes++
			// 只统计在线节点的资源使用率
			totalCPUUsage += nm.CPUUsage
			totalMemoryUsage += nm.MemoryUsage
			totalDiskUsage += nm.DiskUsage
			// 累加总内存和总磁盘字节数
			totalMemoryBytes += nm.MemoryTotal
			totalDiskBytes += nm.DiskTotal
		} else {
			offlineNodes++
		}

		// 构建节点数据（包含所有字段，供前端使用）
		nodeData := map[string]interface{}{
			"node_id":      nm.NodeID,
			"hostname":     nm.Hostname,
			"ip":           nm.IP,
			"status":       nm.Status,
			"cpu_usage":    nm.CPUUsage,
			"memory_usage": nm.MemoryUsage,
			"disk_usage":   nm.DiskUsage,
			"network_rx":   nm.NetworkRx,
			"network_tx":   nm.NetworkTx,
		}
		nodesList = append(nodesList, nodeData)
	}

	// 计算平均使用率（只统计在线节点）
	var avgCPU, avgMemory, avgDisk float64
	if onlineNodes > 0 {
		avgCPU = totalCPUUsage / float64(onlineNodes)
		avgMemory = totalMemoryUsage / float64(onlineNodes)
		avgDisk = totalDiskUsage / float64(onlineNodes)
	}

	// 构建返回结果
	result := map[string]interface{}{
		"aggregate": map[string]interface{}{
			"avg_cpu":         avgCPU,
			"avg_memory":      avgMemory,
			"avg_disk":        avgDisk,
			"total_memory_gb": totalMemoryBytes / (1024 * 1024 * 1024), // 集群总内存 GB
			"total_disk_gb":   totalDiskBytes / (1024 * 1024 * 1024),   // 集群总磁盘 GB
			"node_counts": map[string]int{
				"total":   totalNodes,
				"online":  onlineNodes,
				"offline": offlineNodes,
			},
		},
		"nodes": nodesList,
	}

	return result, nil
}
