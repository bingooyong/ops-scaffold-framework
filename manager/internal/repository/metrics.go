package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"gorm.io/gorm"
)

// NodeMetrics 节点指标数据（用于集群概览）
type NodeMetrics struct {
	NodeID      string  `json:"node_id"`
	Hostname    string  `json:"hostname"`
	IP          string  `json:"ip"`
	Status      string  `json:"status"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	MemoryTotal float64 `json:"memory_total"` // 总内存字节数
	DiskUsage   float64 `json:"disk_usage"`
	DiskTotal   float64 `json:"disk_total"`   // 总磁盘字节数
	NetworkRx   float64 `json:"network_rx"`   // 接收字节数
	NetworkTx   float64 `json:"network_tx"`   // 发送字节数
}

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
	// ListByTimeRangeWithInterval 根据时间范围获取指标（支持采样间隔聚合）
	ListByTimeRangeWithInterval(ctx context.Context, nodeID string, metricType string, startTime, endTime time.Time, interval time.Duration) ([]*model.Metrics, error)
	// GetAggregateStats 获取聚合统计（min/max/avg）
	GetAggregateStats(ctx context.Context, nodeID string, metricType string, startTime, endTime time.Time) (min, max, avg float64, err error)
	// GetAllNodesLatestMetrics 获取所有节点的最新指标数据
	GetAllNodesLatestMetrics(ctx context.Context) ([]*NodeMetrics, error)
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

// ListByTimeRangeWithInterval 根据时间范围获取指标（支持采样间隔聚合）
// 如果 interval 为 0，返回原始数据；否则按时间桶聚合
// 预期使用索引: idx_node_id_type_timestamp (node_id, type, timestamp)
func (r *metricsRepository) ListByTimeRangeWithInterval(ctx context.Context, nodeID string, metricType string, startTime, endTime time.Time, interval time.Duration) ([]*model.Metrics, error) {
	var metrics []*model.Metrics

	// 如果 interval 为 0，直接查询原始数据
	if interval == 0 {
		err := r.db.WithContext(ctx).
			Where("node_id = ? AND type = ? AND timestamp BETWEEN ? AND ?", nodeID, metricType, startTime, endTime).
			Order("timestamp ASC").
			Find(&metrics).Error
		return metrics, err
	}

	// 使用时间桶聚合
	// 将 interval 转换为秒数
	intervalSeconds := int64(interval.Seconds())

	// 构建聚合查询 SQL
	// 注意：不同指标类型可能有不同的 value key，这里使用通用的 usage_percent
	// 如果其他类型需要不同的 key，可以在后续优化中扩展
	querySQL := `
		SELECT 
			node_id,
			type,
			FROM_UNIXTIME((UNIX_TIMESTAMP(timestamp) DIV ?) * ?) as timestamp,
			JSON_OBJECT('usage_percent', AVG(JSON_EXTRACT(values, '$.usage_percent'))) as values,
			NOW() as created_at,
			NULL as deleted_at
		FROM metrics
		WHERE node_id = ? AND type = ? AND timestamp BETWEEN ? AND ?
		GROUP BY node_id, type, UNIX_TIMESTAMP(timestamp) DIV ?
		ORDER BY timestamp ASC
	`

	// 执行原始 SQL 查询
	rows, err := r.db.WithContext(ctx).Raw(querySQL, intervalSeconds, intervalSeconds, nodeID, metricType, startTime, endTime, intervalSeconds).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 手动扫描结果
	for rows.Next() {
		var m model.Metrics
		var valuesJSON string
		var deletedAt *time.Time

		err := rows.Scan(
			&m.NodeID,
			&m.Type,
			&m.Timestamp,
			&valuesJSON,
			&m.CreatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析 JSON values
		if valuesJSON != "" {
			if err := json.Unmarshal([]byte(valuesJSON), &m.Values); err != nil {
				return nil, err
			}
		}

		// 处理 deleted_at（聚合查询中 deleted_at 为 NULL，不需要设置）
		// 聚合数据不设置 deleted_at，deletedAt 为 nil 时表示未删除

		metrics = append(metrics, &m)
	}

	return metrics, rows.Err()
}

// GetAggregateStats 获取聚合统计（min/max/avg）
// 预期使用索引: idx_node_id_type_timestamp (node_id, type, timestamp)
func (r *metricsRepository) GetAggregateStats(ctx context.Context, nodeID string, metricType string, startTime, endTime time.Time) (min, max, avg float64, err error) {
	// 使用 JSON_EXTRACT 提取 usage_percent 值进行聚合
	// 注意：不同指标类型可能有不同的 value key，这里使用通用的 usage_percent
	querySQL := `
		SELECT 
			MIN(JSON_EXTRACT(values, '$.usage_percent')) as min_value,
			MAX(JSON_EXTRACT(values, '$.usage_percent')) as max_value,
			AVG(JSON_EXTRACT(values, '$.usage_percent')) as avg_value
		FROM metrics
		WHERE node_id = ? AND type = ? AND timestamp BETWEEN ? AND ?
	`

	row := r.db.WithContext(ctx).Raw(querySQL, nodeID, metricType, startTime, endTime).Row()

	// 使用 sql.NullFloat64 处理可能的 NULL 值
	var minVal, maxVal, avgVal sql.NullFloat64
	err = row.Scan(&minVal, &maxVal, &avgVal)
	if err != nil {
		if err == sql.ErrNoRows {
			// 空数据情况，返回零值
			return 0, 0, 0, nil
		}
		return 0, 0, 0, err
	}

	// 处理 NULL 值
	if minVal.Valid {
		min = minVal.Float64
	}
	if maxVal.Valid {
		max = maxVal.Float64
	}
	if avgVal.Valid {
		avg = avgVal.Float64
	}

	return min, max, avg, nil
}

// GetAllNodesLatestMetrics 获取所有节点的最新指标数据
// 使用 LEFT JOIN 和子查询获取每个节点的最新指标
func (r *metricsRepository) GetAllNodesLatestMetrics(ctx context.Context) ([]*NodeMetrics, error) {
	// 使用子查询获取每个节点的最新指标，然后 LEFT JOIN nodes 表
	// 这样可以避免全表扫描，只查询每个节点的最新指标
	querySQL := `
		SELECT
			n.node_id,
			n.hostname,
			n.ip,
			n.status,
			COALESCE(JSON_EXTRACT(m_cpu.values, '$.usage_percent'), 0) as cpu_usage,
			COALESCE(JSON_EXTRACT(m_mem.values, '$.usage_percent'), 0) as memory_usage,
			COALESCE(JSON_EXTRACT(m_mem.values, '$.total_bytes'), 0) as memory_total,
			COALESCE(JSON_EXTRACT(m_disk.values, '$.usage_percent'), 0) as disk_usage,
			COALESCE(JSON_EXTRACT(m_disk.values, '$.total_bytes'), 0) as disk_total,
			COALESCE(JSON_EXTRACT(m_net.values, '$.rx_bytes'), 0) as network_rx,
			COALESCE(JSON_EXTRACT(m_net.values, '$.tx_bytes'), 0) as network_tx
		FROM nodes n
		LEFT JOIN (
			SELECT m1.node_id, m1.values, m1.timestamp
			FROM metrics m1
			INNER JOIN (
				SELECT node_id, MAX(timestamp) as max_timestamp
				FROM metrics
				WHERE type = 'cpu' AND deleted_at IS NULL
				GROUP BY node_id
			) m2 ON m1.node_id = m2.node_id 
				AND m1.timestamp = m2.max_timestamp 
				AND m1.type = 'cpu'
		) m_cpu ON n.node_id = m_cpu.node_id
		LEFT JOIN (
			SELECT m1.node_id, m1.values, m1.timestamp
			FROM metrics m1
			INNER JOIN (
				SELECT node_id, MAX(timestamp) as max_timestamp
				FROM metrics
				WHERE type = 'memory' AND deleted_at IS NULL
				GROUP BY node_id
			) m2 ON m1.node_id = m2.node_id 
				AND m1.timestamp = m2.max_timestamp 
				AND m1.type = 'memory'
		) m_mem ON n.node_id = m_mem.node_id
		LEFT JOIN (
			SELECT m1.node_id, m1.values, m1.timestamp
			FROM metrics m1
			INNER JOIN (
				SELECT node_id, MAX(timestamp) as max_timestamp
				FROM metrics
				WHERE type = 'disk' AND deleted_at IS NULL
				GROUP BY node_id
			) m2 ON m1.node_id = m2.node_id 
				AND m1.timestamp = m2.max_timestamp 
				AND m1.type = 'disk'
		) m_disk ON n.node_id = m_disk.node_id
		LEFT JOIN (
			SELECT m1.node_id, m1.values, m1.timestamp
			FROM metrics m1
			INNER JOIN (
				SELECT node_id, MAX(timestamp) as max_timestamp
				FROM metrics
				WHERE type = 'network' AND deleted_at IS NULL
				GROUP BY node_id
			) m2 ON m1.node_id = m2.node_id 
				AND m1.timestamp = m2.max_timestamp 
				AND m1.type = 'network'
		) m_net ON n.node_id = m_net.node_id
		WHERE n.deleted_at IS NULL
		ORDER BY n.node_id
	`

	rows, err := r.db.WithContext(ctx).Raw(querySQL).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodeMetrics []*NodeMetrics
	for rows.Next() {
		var nm NodeMetrics
		var cpuUsage, memoryUsage, memoryTotal, diskUsage, diskTotal, networkRx, networkTx sql.NullFloat64

		err := rows.Scan(
			&nm.NodeID,
			&nm.Hostname,
			&nm.IP,
			&nm.Status,
			&cpuUsage,
			&memoryUsage,
			&memoryTotal,
			&diskUsage,
			&diskTotal,
			&networkRx,
			&networkTx,
		)
		if err != nil {
			return nil, err
		}

		// 处理 NULL 值
		if cpuUsage.Valid {
			nm.CPUUsage = cpuUsage.Float64
		}
		if memoryUsage.Valid {
			nm.MemoryUsage = memoryUsage.Float64
		}
		if memoryTotal.Valid {
			nm.MemoryTotal = memoryTotal.Float64
		}
		if diskUsage.Valid {
			nm.DiskUsage = diskUsage.Float64
		}
		if diskTotal.Valid {
			nm.DiskTotal = diskTotal.Float64
		}
		if networkRx.Valid {
			nm.NetworkRx = networkRx.Float64
		}
		if networkTx.Valid {
			nm.NetworkTx = networkTx.Float64
		}

		nodeMetrics = append(nodeMetrics, &nm)
	}

	return nodeMetrics, rows.Err()
}
