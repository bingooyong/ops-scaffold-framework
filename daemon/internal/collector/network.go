package collector

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"
)

// NetworkCollector 网络采集器
type NetworkCollector struct {
	enabled    bool
	interval   time.Duration
	interfaces []string
	logger     *zap.Logger
}

// NewNetworkCollector 创建网络采集器
func NewNetworkCollector(enabled bool, interval time.Duration, interfaces []string, logger *zap.Logger) *NetworkCollector {
	return &NetworkCollector{
		enabled:    enabled,
		interval:   interval,
		interfaces: interfaces,
		logger:     logger,
	}
}

// Name 返回采集器名称
func (c *NetworkCollector) Name() string {
	return "network"
}

// Interval 返回采集间隔
func (c *NetworkCollector) Interval() time.Duration {
	return c.interval
}

// Enabled 返回是否启用
func (c *NetworkCollector) Enabled() bool {
	return c.enabled
}

// Collect 执行网络采集
func (c *NetworkCollector) Collect(ctx context.Context) (*types.Metrics, error) {
	// 获取网络IO统计
	ioCounters, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		c.logger.Error("failed to get network io counters", zap.Error(err))
		return nil, err
	}

	netMetrics := make([]map[string]interface{}, 0)

	for _, io := range ioCounters {
		// 如果指定了网卡，则只采集指定的
		if len(c.interfaces) > 0 {
			found := false
			for _, iface := range c.interfaces {
				if io.Name == iface {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		netInfo := map[string]interface{}{
			"interface":    io.Name,
			"bytes_sent":   io.BytesSent,
			"bytes_recv":   io.BytesRecv,
			"packets_sent": io.PacketsSent,
			"packets_recv": io.PacketsRecv,
			"errin":        io.Errin,
			"errout":       io.Errout,
			"dropin":       io.Dropin,
			"dropout":      io.Dropout,
		}

		netMetrics = append(netMetrics, netInfo)
	}

	metrics := &types.Metrics{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Values:    map[string]interface{}{"interfaces": netMetrics},
	}

	return metrics, nil
}
