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

	diskMetrics := make([]map[string]interface{}, 0)

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

		// 获取IO统计（可选）
		ioCounters, err := disk.IOCountersWithContext(ctx, partition.Device)
		if err != nil {
			c.logger.Debug("failed to get disk io counters",
				zap.String("device", partition.Device),
				zap.Error(err))
		}

		diskInfo := map[string]interface{}{
			"mountpoint":   partition.Mountpoint,
			"device":       partition.Device,
			"fstype":       partition.Fstype,
			"total":        usage.Total,
			"used":         usage.Used,
			"free":         usage.Free,
			"used_percent": usage.UsedPercent,
			"inodes_total": usage.InodesTotal,
			"inodes_used":  usage.InodesUsed,
			"inodes_free":  usage.InodesFree,
		}

		// 添加IO统计
		if len(ioCounters) > 0 {
			for _, io := range ioCounters {
				diskInfo["read_count"] = io.ReadCount
				diskInfo["write_count"] = io.WriteCount
				diskInfo["read_bytes"] = io.ReadBytes
				diskInfo["write_bytes"] = io.WriteBytes
				diskInfo["read_time"] = io.ReadTime
				diskInfo["write_time"] = io.WriteTime
				break
			}
		}

		diskMetrics = append(diskMetrics, diskInfo)
	}

	metrics := &types.Metrics{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Values:    map[string]interface{}{"disks": diskMetrics},
	}

	return metrics, nil
}
