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

	// 聚合数据
	var totalBytesSent uint64
	var totalBytesRecv uint64
	var totalPacketsSent uint64
	var totalPacketsRecv uint64
	var totalErrIn uint64
	var totalErrOut uint64
	var totalDropIn uint64
	var totalDropOut uint64

	// 明细数据（保留完整信息供未来扩展）
	interfaceDetails := make([]map[string]interface{}, 0)

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

		// 累加到汇总数据
		totalBytesSent += io.BytesSent
		totalBytesRecv += io.BytesRecv
		totalPacketsSent += io.PacketsSent
		totalPacketsRecv += io.PacketsRecv
		totalErrIn += io.Errin
		totalErrOut += io.Errout
		totalDropIn += io.Dropin
		totalDropOut += io.Dropout

		// 保存明细数据
		interfaceInfo := map[string]interface{}{
			"interface":    io.Name,
			"bytes_sent":   io.BytesSent,
			"bytes_recv":   io.BytesRecv,
			"packets_sent": io.PacketsSent,
			"packets_recv": io.PacketsRecv,
			"error_in":     io.Errin,
			"error_out":    io.Errout,
			"drop_in":      io.Dropin,
			"drop_out":     io.Dropout,
		}
		interfaceDetails = append(interfaceDetails, interfaceInfo)
	}

	// 返回数据：同时包含汇总数据（扁平化）和明细数据（数组）
	metrics := &types.Metrics{
		Name:      c.Name(),
		Timestamp: time.Now(),
		Values: map[string]interface{}{
			// 汇总数据（扁平化，供前端快速展示）
			"tx_bytes":        totalBytesSent,   // 发送字节 (前端使用)
			"rx_bytes":        totalBytesRecv,   // 接收字节 (前端使用)
			"bytes_sent":      totalBytesSent,   // 发送字节 (别名)
			"bytes_recv":      totalBytesRecv,   // 接收字节 (别名)
			"packets_sent":    totalPacketsSent,
			"packets_recv":    totalPacketsRecv,
			"error_in":        totalErrIn,
			"error_out":       totalErrOut,
			"drop_in":         totalDropIn,
			"drop_out":        totalDropOut,
			"interface_count": len(interfaceDetails),
			// 明细数据（数组，供未来扩展使用）
			"details":         interfaceDetails,
		},
	}

	c.logger.Debug("network metrics collected",
		zap.Int("interfaces", len(interfaceDetails)),
		zap.Uint64("tx_bytes", totalBytesSent),
		zap.Uint64("rx_bytes", totalBytesRecv),
	)

	return metrics, nil
}
