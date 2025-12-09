package collector

import (
	"context"
	"runtime"
	"strings"
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

	// 用于去重的 map，key 为设备基础名称（去除分区号），value 为已处理的最大总空间
	// 在 macOS 上，同一个物理磁盘可能有多个虚拟挂载点，需要去重
	processedDevices := make(map[string]struct {
		total uint64
		used  uint64
	})

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

		// 在 macOS 上，过滤掉虚拟挂载点（如 /System/Volumes/*）
		// 这些虚拟挂载点通常指向同一个物理磁盘，会导致重复计算
		if runtime.GOOS == "darwin" {
			if strings.HasPrefix(partition.Mountpoint, "/System/Volumes/") {
				c.logger.Debug("skipping virtual mount point on macOS",
					zap.String("mountpoint", partition.Mountpoint))
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

		// 在 macOS 上，通过设备路径去重（去除分区号）
		// 例如：/dev/disk3s1s1, /dev/disk3s6, /dev/disk3s2 都指向同一个物理磁盘 disk3
		// 我们只统计每个物理磁盘一次，选择总空间最大的分区（通常是主分区）
		if runtime.GOOS == "darwin" {
			// 提取设备基础名称（去除分区号）
			// /dev/disk3s1s1 -> disk3, /dev/disk3s6 -> disk3
			deviceBase := partition.Device
			if idx := strings.Index(deviceBase, "s"); idx > 0 {
				deviceBase = deviceBase[:idx]
			}

			// 如果这个设备已经处理过
			if existing, exists := processedDevices[deviceBase]; exists {
				// 如果当前分区的总空间小于等于已处理的，则跳过（避免重复计算）
				if usage.Total <= existing.total {
					c.logger.Debug("skipping duplicate device partition",
						zap.String("device", partition.Device),
						zap.String("mountpoint", partition.Mountpoint),
						zap.Uint64("total", usage.Total),
						zap.Uint64("existing_total", existing.total))
					continue
				}
				// 如果当前分区的总空间更大，说明这是主分区，需要替换之前的值
				// 先减去之前的值
				totalBytes -= existing.total
				usedBytes -= existing.used
			}

			// 记录或更新已处理的设备（使用总空间最大的分区）
			processedDevices[deviceBase] = struct {
				total uint64
				used  uint64
			}{
				total: usage.Total,
				used:  usage.Used,
			}
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
			"details": diskDetails,
		},
	}

	c.logger.Debug("disk metrics collected",
		zap.Int("partitions", len(diskDetails)),
		zap.Float64("usage_percent", usagePercent),
	)

	return metrics, nil
}
