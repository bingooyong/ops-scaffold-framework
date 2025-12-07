package collector

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"github.com/shirou/gopsutil/v3/disk"
	"go.uber.org/zap"
)

// DiskCollector 磁盘采集器
type DiskCollector struct {
	enabled     bool
	interval    time.Duration
	mountPoints []string
	logger      *zap.Logger
}

// NewDiskCollector 创建磁盘采集器
func NewDiskCollector(enabled bool, interval time.Duration, mountPoints []string, logger *zap.Logger) *DiskCollector {
	return &DiskCollector{
		enabled:     enabled,
		interval:    interval,
		mountPoints: mountPoints,
		logger:      logger,
	}
}

// Name 返回采集器名称
func (c *DiskCollector) Name() string {
	return "disk"
}

// Interval 返回采集间隔
func (c *DiskCollector) Interval() time.Duration {
	return c.interval
}

// Enabled 返回是否启用
func (c *DiskCollector) Enabled() bool {
	return c.enabled
}

// Collect 执行磁盘采集
func (c *DiskCollector) Collect(ctx context.Context) (*types.Metrics, error) {
	// 获取所有分区
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		c.logger.Error("failed to get disk partitions", zap.Error(err))
		return nil, err
	}

	// 聚合数据
	var totalBytes uint64
	var usedBytes uint64
	var totalReadBytes uint64
	var totalWriteBytes uint64

	// 明细数据（保留完整信息供未来扩展）
	diskDetails := make([]map[string]interface{}, 0)

	for _, partition := range partitions {
		// 如果指定了挂载点，则只采集指定的
		if len(c.mountPoints) > 0 {
			found := false
			for _, mp := range c.mountPoints {
				if partition.Mountpoint == mp {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 获取使用情况
		usage, err := disk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			c.logger.Warn("failed to get disk usage",
				zap.String("mountpoint", partition.Mountpoint),
				zap.Error(err))
			continue
		}

		// 累加到汇总数据
		totalBytes += usage.Total
		usedBytes += usage.Used

		// 获取IO统计（可选）
		var readBytes, writeBytes uint64
		ioCounters, err := disk.IOCountersWithContext(ctx, partition.Device)
		if err != nil {
			c.logger.Debug("failed to get disk io counters",
				zap.String("device", partition.Device),
				zap.Error(err))
		} else {
			for _, io := range ioCounters {
				readBytes = io.ReadBytes
				writeBytes = io.WriteBytes
				totalReadBytes += readBytes
				totalWriteBytes += writeBytes
				break
			}
		}

		// 保存明细数据
		diskInfo := map[string]interface{}{
			"mountpoint":   partition.Mountpoint,
			"device":       partition.Device,
			"fstype":       partition.Fstype,
			"total":        usage.Total,
			"used":         usage.Used,
			"free":         usage.Free,
			"used_percent": usage.UsedPercent,
			"read_bytes":   readBytes,
			"write_bytes":  writeBytes,
		}
		diskDetails = append(diskDetails, diskInfo)
	}

	// 计算使用率
	var usagePercent float64
	if totalBytes > 0 {
		usagePercent = float64(usedBytes) / float64(totalBytes) * 100
	}

	// 返回数据：同时包含汇总数据（扁平化）和明细数据（数组）
	metrics := &types.Metrics{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Values: map[string]interface{}{
			// 汇总数据（扁平化，供前端快速展示）
			"total_bytes":     totalBytes,
			"used_bytes":      usedBytes,
			"free_bytes":      totalBytes - usedBytes,
			"usage_percent":   usagePercent,
			"read_bytes":      totalReadBytes,
			"write_bytes":     totalWriteBytes,
			"partition_count": len(diskDetails),
			// 明细数据（数组，供未来扩展使用）
			"details":         diskDetails,
		},
	}

	c.logger.Debug("disk metrics collected",
		zap.Int("partitions", len(diskDetails)),
		zap.Float64("usage_percent", usagePercent),
	)

	return metrics, nil
}
