package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MetricsCleaner 监控指标清理服务
type MetricsCleaner struct {
	db            *gorm.DB
	logger        *zap.Logger
	retentionDays int
}

// NewMetricsCleaner 创建监控指标清理服务实例
func NewMetricsCleaner(db *gorm.DB, retentionDays int, logger *zap.Logger) *MetricsCleaner {
	return &MetricsCleaner{
		db:            db,
		logger:        logger,
		retentionDays: retentionDays,
	}
}

// CleanExpiredPartitions 清理过期的分区
// 使用 ALTER TABLE DROP PARTITION 删除整个分区，比 DELETE 语句高效
func (c *MetricsCleaner) CleanExpiredPartitions(ctx context.Context) error {
	// 计算过期日期
	expireDate := time.Now().AddDate(0, 0, -c.retentionDays)
	c.logger.Info("starting partition cleanup",
		zap.Int("retention_days", c.retentionDays),
		zap.Time("expire_date", expireDate))

	// 查询当前分区列表
	partitions, err := c.getPartitions(ctx)
	if err != nil {
		c.logger.Error("failed to get partitions", zap.Error(err))
		return fmt.Errorf("failed to get partitions: %w", err)
	}

	// 解析分区名称，提取日期
	partitionDateRegex := regexp.MustCompile(`^p(\d{8})$`)
	var expiredPartitions []string
	var errors []error

	for _, partition := range partitions {
		// 跳过 future 分区
		if partition == "p_future" {
			continue
		}

		// 解析分区日期
		matches := partitionDateRegex.FindStringSubmatch(partition)
		if len(matches) != 2 {
			c.logger.Warn("invalid partition name format", zap.String("partition", partition))
			continue
		}

		dateStr := matches[1]
		partitionDate, err := time.Parse("20060102", dateStr)
		if err != nil {
			c.logger.Warn("failed to parse partition date",
				zap.String("partition", partition),
				zap.String("date_str", dateStr),
				zap.Error(err))
			continue
		}

		// 判断是否过期
		if partitionDate.Before(expireDate) {
			expiredPartitions = append(expiredPartitions, partition)
		}
	}

	if len(expiredPartitions) == 0 {
		c.logger.Info("no expired partitions to clean")
		return nil
	}

	c.logger.Info("found expired partitions",
		zap.Int("count", len(expiredPartitions)),
		zap.Strings("partitions", expiredPartitions))

	// 删除过期分区
	for _, partition := range expiredPartitions {
		if err := c.dropPartition(ctx, partition); err != nil {
			c.logger.Error("failed to drop partition",
				zap.String("partition", partition),
				zap.Error(err))
			errors = append(errors, fmt.Errorf("failed to drop partition %s: %w", partition, err))
			// 继续处理其他分区
			continue
		}

		c.logger.Info("dropped expired partition",
			zap.String("partition", partition),
			zap.Int("retention_days", c.retentionDays))
	}

	// 返回汇总错误
	if len(errors) > 0 {
		return fmt.Errorf("failed to drop %d partitions: %v", len(errors), errors)
	}

	c.logger.Info("partition cleanup completed",
		zap.Int("dropped_count", len(expiredPartitions)))
	return nil
}

// getPartitions 获取当前所有分区名称
func (c *MetricsCleaner) getPartitions(ctx context.Context) ([]string, error) {
	var partitions []string

	// 查询 information_schema.PARTITIONS 表
	query := `
		SELECT DISTINCT PARTITION_NAME
		FROM information_schema.PARTITIONS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'metrics'
		  AND PARTITION_NAME IS NOT NULL
		ORDER BY PARTITION_NAME
	`

	rows, err := c.db.WithContext(ctx).Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var partitionName string
		if err := rows.Scan(&partitionName); err != nil {
			return nil, err
		}
		partitions = append(partitions, partitionName)
	}

	return partitions, rows.Err()
}

// dropPartition 删除指定分区
func (c *MetricsCleaner) dropPartition(ctx context.Context, partitionName string) error {
	sql := fmt.Sprintf("ALTER TABLE metrics DROP PARTITION %s", partitionName)
	result := c.db.WithContext(ctx).Exec(sql)
	if result.Error != nil {
		return result.Error
	}

	// 记录影响的行数（分区删除不返回行数，但可以记录操作成功）
	c.logger.Debug("partition dropped",
		zap.String("partition", partitionName),
		zap.Int64("rows_affected", result.RowsAffected))

	return nil
}

