package collector

import (
	"context"
	"sync"
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
	// 上一次采集的累计值，用于计算差值
	lastRxBytes uint64
	lastTxBytes uint64
	lastTime    time.Time
	mu          sync.Mutex
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

	// 计算本次采集的流量差值（相对于上一次采集）
	c.mu.Lock()
	now := time.Now()
	var deltaRxBytes, deltaTxBytes uint64
	var timeDelta time.Duration

	if c.lastTime.IsZero() {
		// 第一次采集，没有历史数据，返回0（或者返回累计值，但标记为累计值）
		// 这里返回0，表示本次采集期间没有流量变化
		deltaRxBytes = 0
		deltaTxBytes = 0
		timeDelta = 0
		c.logger.Debug("first network collection, no delta available")
	} else {
		// 计算差值
		if totalBytesRecv >= c.lastRxBytes {
			deltaRxBytes = totalBytesRecv - c.lastRxBytes
		} else {
			// 累计值重置（系统重启或计数器溢出），使用当前值
			deltaRxBytes = totalBytesRecv
			c.logger.Warn("network rx bytes counter reset or overflow",
				zap.Uint64("current", totalBytesRecv),
				zap.Uint64("last", c.lastRxBytes))
		}

		if totalBytesSent >= c.lastTxBytes {
			deltaTxBytes = totalBytesSent - c.lastTxBytes
		} else {
			// 累计值重置（系统重启或计数器溢出），使用当前值
			deltaTxBytes = totalBytesSent
			c.logger.Warn("network tx bytes counter reset or overflow",
				zap.Uint64("current", totalBytesSent),
				zap.Uint64("last", c.lastTxBytes))
		}

		timeDelta = now.Sub(c.lastTime)
	}

	// 更新上一次的值
	c.lastRxBytes = totalBytesRecv
	c.lastTxBytes = totalBytesSent
	c.lastTime = now
	c.mu.Unlock()

	// 返回数据：同时包含汇总数据（扁平化）和明细数据（数组）
	// 注意：rx_bytes 和 tx_bytes 现在表示本次采集期间的流量差值，而不是累计值
	metrics := &types.Metrics{
		Name:      c.Name(),
		Timestamp: now,
		Values: map[string]interface{}{
			// 本次采集期间的流量差值（供前端展示）
			"tx_bytes": deltaTxBytes, // 本次采集期间发送的字节数
			"rx_bytes": deltaRxBytes, // 本次采集期间接收的字节数
			// 累计值（保留，供未来扩展使用）
			"bytes_sent":      totalBytesSent, // 累计发送字节 (别名)
			"bytes_recv":      totalBytesRecv, // 累计接收字节 (别名)
			"packets_sent":    totalPacketsSent,
			"packets_recv":    totalPacketsRecv,
			"error_in":        totalErrIn,
			"error_out":       totalErrOut,
			"drop_in":         totalDropIn,
			"drop_out":        totalDropOut,
			"interface_count": len(interfaceDetails),
			// 采集间隔（秒），用于计算速率
			"interval_seconds": timeDelta.Seconds(),
			// 明细数据（数组，供未来扩展使用）
			"details": interfaceDetails,
		},
	}

	c.logger.Debug("network metrics collected",
		zap.Int("interfaces", len(interfaceDetails)),
		zap.Uint64("delta_tx_bytes", deltaTxBytes),
		zap.Uint64("delta_rx_bytes", deltaRxBytes),
		zap.Duration("time_delta", timeDelta),
		zap.Uint64("total_tx_bytes", totalBytesSent),
		zap.Uint64("total_rx_bytes", totalBytesRecv),
	)

	return metrics, nil
}
