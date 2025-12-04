package collector

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"github.com/shirou/gopsutil/v3/cpu"
	"go.uber.org/zap"
)

// CPUCollector CPU采集器
type CPUCollector struct {
	enabled  bool
	interval time.Duration
	logger   *zap.Logger
}

// NewCPUCollector 创建CPU采集器
func NewCPUCollector(enabled bool, interval time.Duration, logger *zap.Logger) *CPUCollector {
	return &CPUCollector{
		enabled:  enabled,
		interval: interval,
		logger:   logger,
	}
}

// Name 返回采集器名称
func (c *CPUCollector) Name() string {
	return "cpu"
}

// Interval 返回采集间隔
func (c *CPUCollector) Interval() time.Duration {
	return c.interval
}

// Enabled 返回是否启用
func (c *CPUCollector) Enabled() bool {
	return c.enabled
}

// Collect 执行CPU采集
func (c *CPUCollector) Collect(ctx context.Context) (*types.Metrics, error) {
	// 获取CPU使用率（采样1秒）
	percent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		c.logger.Error("failed to get cpu percent", zap.Error(err))
		return nil, err
	}

	// 获取CPU信息
	info, err := cpu.InfoWithContext(ctx)
	if err != nil {
		c.logger.Error("failed to get cpu info", zap.Error(err))
		return nil, err
	}

	// 获取CPU时间统计
	times, err := cpu.TimesWithContext(ctx, false)
	if err != nil {
		c.logger.Error("failed to get cpu times", zap.Error(err))
		return nil, err
	}

	// 获取CPU数量
	counts, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		c.logger.Error("failed to get cpu counts", zap.Error(err))
		return nil, err
	}

	// 组装指标数据
	values := make(map[string]interface{})

	// CPU使用率
	if len(percent) > 0 {
		values["usage_percent"] = percent[0]
	}

	// CPU核心数
	values["cores"] = counts

	// CPU型号
	if len(info) > 0 {
		values["model"] = info[0].ModelName
		values["mhz"] = info[0].Mhz
	}

	// CPU时间统计
	if len(times) > 0 {
		values["user"] = times[0].User
		values["system"] = times[0].System
		values["idle"] = times[0].Idle
		values["iowait"] = times[0].Iowait
	}

	metrics := &types.Metrics{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Values:    values,
	}

	return metrics, nil
}
