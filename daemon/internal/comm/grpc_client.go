//go:build !e2e && !grpc_test
// +build !e2e,!grpc_test

package comm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	managerpb "github.com/bingooyong/ops-scaffold-framework/daemon/pkg/proto/manager"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// GRPCClient gRPC客户端
// 使用 daemon 自己的 proto 定义，避免依赖 manager 模块
// 该客户端为长连接设计, 支持后台心跳/指标上报与断线自动重连
type GRPCClient struct {
	conn   *grpc.ClientConn
	client managerpb.ManagerServiceClient
	config *config.ManagerConfig
	nodeID string
	logger *zap.Logger

	// 为了支持后台 loop 与自动重连
	reconnectInterval time.Duration
}

// NewGRPCClient 创建gRPC客户端
func NewGRPCClient(cfg *config.ManagerConfig, logger *zap.Logger) *GRPCClient {
	return &GRPCClient{
		config:            cfg,
		logger:            logger,
		reconnectInterval: 5 * time.Second,
	}
}

// Connect 连接到Manager(建立或重新建立连接)
func (c *GRPCClient) Connect(ctx context.Context) error {
	// 如果已有连接且健康, 直接返回
	if c.conn != nil {
		st := c.conn.GetState()
		if st == connectivity.Ready {
			return nil
		}

		// 断开旧连接,准备重连
		_ = c.conn.Close()
		c.conn = nil
	}

	// 加载TLS证书
	var opts []grpc.DialOption

	if c.config.TLS.CertFile != "" && c.config.TLS.KeyFile != "" && c.config.TLS.CAFile != "" {
		cert, err := tls.LoadX509KeyPair(c.config.TLS.CertFile, c.config.TLS.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load client cert: %w", err)
		}

		caCert, err := os.ReadFile(c.config.TLS.CAFile)
		if err != nil {
			return fmt.Errorf("failed to read CA cert: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to append CA cert")
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      certPool,
			MinVersion:   tls.VersionTLS13,
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		// 开发环境：不使用TLS
		c.logger.Warn("TLS not configured, using insecure connection")
		opts = append(opts, grpc.WithInsecure())
	}

	// 添加keepalive参数(调整为30秒避免too_many_pings错误)
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                30 * time.Second, // keepalive ping间隔,与服务端MinTime(20秒)匹配
		Timeout:             10 * time.Second, // keepalive超时时间
		PermitWithoutStream: true,             // 允许无流时发送ping
	}))

	// 建立连接
	conn, err := grpc.DialContext(ctx, c.config.Address, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial manager: %w", err)
	}

	c.conn = conn
	c.client = managerpb.NewManagerServiceClient(conn)
	c.logger.Info("connected to manager", zap.String("address", c.config.Address))

	return nil
}

// Close 关闭连接
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Register 注册节点
func (c *GRPCClient) Register(ctx context.Context, nodeID string, info *types.NodeInfo) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("gRPC client not connected")
	}

	c.logger.Info("registering node",
		zap.String("node_id", nodeID),
		zap.String("hostname", info.Hostname),
		zap.String("ip", info.IP))

	// 构建注册请求
	req := &managerpb.RegisterNodeRequest{
		NodeId:        nodeID,
		Hostname:      info.Hostname,
		Ip:            info.IP,
		Os:            info.OS,
		Arch:          info.Arch,
		Labels:        info.Labels,
		DaemonVersion: info.DaemonVer,
		AgentVersion:  info.AgentVer,
	}

	// 调用gRPC服务
	resp, err := c.client.RegisterNode(ctx, req)
	if err != nil {
		c.logger.Error("failed to register node", zap.Error(err))
		return "", fmt.Errorf("gRPC call failed: %w", err)
	}

	if !resp.Success {
		c.logger.Error("node registration failed", zap.String("message", resp.Message))
		return "", fmt.Errorf("registration failed: %s", resp.Message)
	}

	c.nodeID = nodeID
	c.logger.Info("node registered successfully", zap.String("node_id", c.nodeID))

	return c.nodeID, nil
}

// Heartbeat 发送心跳
func (c *GRPCClient) Heartbeat(ctx context.Context) error {
	if c.nodeID == "" {
		return fmt.Errorf("node not registered")
	}

	if c.client == nil || c.conn == nil {
		return fmt.Errorf("gRPC client not connected")
	}

	c.logger.Debug("sending heartbeat", zap.String("node_id", c.nodeID))

	// 构建心跳请求
	req := &managerpb.HeartbeatRequest{
		NodeId:    c.nodeID,
		Timestamp: time.Now().Unix(),
	}

	// 调用gRPC服务
	resp, err := c.client.Heartbeat(ctx, req)
	if err != nil {
		c.logger.Error("failed to send heartbeat", zap.Error(err))
		return fmt.Errorf("gRPC call failed: %w", err)
	}

	if !resp.Success {
		c.logger.Warn("heartbeat failed", zap.String("message", resp.Message))
		return fmt.Errorf("heartbeat failed: %s", resp.Message)
	}

	return nil
}

// ReportMetrics 上报指标
func (c *GRPCClient) ReportMetrics(ctx context.Context, metrics map[string]*types.Metrics) error {
	if c.nodeID == "" {
		return fmt.Errorf("node not registered")
	}

	if c.client == nil || c.conn == nil {
		return fmt.Errorf("gRPC client not connected")
	}

	c.logger.Debug("reporting metrics",
		zap.String("node_id", c.nodeID),
		zap.Int("count", len(metrics)))

	// 转换指标数据
	var metricData []*managerpb.MetricData
	for name, m := range metrics {
		// 转换values map[string]interface{} -> map[string]float64
		values := make(map[string]float64)
		for k, v := range m.Values {
			switch val := v.(type) {
			case float64:
				values[k] = val
			case float32:
				values[k] = float64(val)
			case int:
				values[k] = float64(val)
			case int64:
				values[k] = float64(val)
			case uint64:
				values[k] = float64(val)
			default:
				// 尝试转换为float64
				if f, ok := val.(float64); ok {
					values[k] = f
				} else {
					c.logger.Warn("skipping non-numeric metric value",
						zap.String("metric", name),
						zap.String("key", k),
						zap.Any("value", v))
				}
			}
		}

		metricData = append(metricData, &managerpb.MetricData{
			Type:      name,
			Timestamp: m.Timestamp.Unix(),
			Values:    values,
		})
	}

	// 构建上报请求
	req := &managerpb.ReportMetricsRequest{
		NodeId:  c.nodeID,
		Metrics: metricData,
	}

	// 调用gRPC服务
	resp, err := c.client.ReportMetrics(ctx, req)
	if err != nil {
		c.logger.Error("failed to report metrics", zap.Error(err))
		return fmt.Errorf("gRPC call failed: %w", err)
	}

	if !resp.Success {
		c.logger.Warn("metrics report failed", zap.String("message", resp.Message))
		return fmt.Errorf("metrics report failed: %s", resp.Message)
	}

	return nil
}

// GetNodeID 获取节点ID
func (c *GRPCClient) GetNodeID() string {
	return c.nodeID
}

// IsConnected 检查连接状态
func (c *GRPCClient) IsConnected() bool {
	if c.conn == nil {
		return false
	}
	st := c.conn.GetState()
	return st == connectivity.Ready
}
