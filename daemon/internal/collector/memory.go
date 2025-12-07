package collector

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// MemoryCollector 内存采集器
type MemoryCollector struct {
	enabled  bool
	interval time.Duration
	logger   *zap.Logger
}

// NewMemoryCollector 创建内存采集器
func NewMemoryCollector(enabled bool, interval time.Duration, logger *zap.Logger) *MemoryCollector {
	return &MemoryCollector{
		enabled:  enabled,
		interval: interval,
		logger:   logger,
	}
}

// Name 返回采集器名称
func (c *MemoryCollector) Name() string {
	return "memory"
}

// Interval 返回采集间隔
func (c *MemoryCollector) Interval() time.Duration {
	return c.interval
}

// Enabled 返回是否启用
func (c *MemoryCollector) Enabled() bool {
	return c.enabled
}

// Collect 执行内存采集
func (c *MemoryCollector) Collect(ctx context.Context) (*types.Metrics, error) {
	// 获取虚拟内存信息
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		c.logger.Error("failed to get virtual memory", zap.Error(err))
		return nil, err
	}

	// 获取Swap内存信息
	swap, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		c.logger.Error("failed to get swap memory", zap.Error(err))
		return nil, err
	}

	// 组装指标数据（使用统一的命名规范）
	values := map[string]interface{}{
		// 主要字段（与 disk/network 保持一致的命名）
		"total_bytes":    vm.Total,
		"used_bytes":     vm.Used,
		"free_bytes":     vm.Free,
		"available_bytes": vm.Available,
		"usage_percent":  vm.UsedPercent,
		// 详细字段
		"cached":       vm.Cached,
		"buffers":      vm.Buffers,
		"swap_total":   swap.Total,
		"swap_used":    swap.Used,
		"swap_free":    swap.Free,
		"swap_percent": swap.UsedPercent,
		// 兼容旧字段名（用于向后兼容）
		"total":     vm.Total,
		"available": vm.Available,
		"used":      vm.Used,
		"free":      vm.Free,
	}

	metrics := &types.Metrics{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Values:    values,
	}

	return metrics, nil
}
