//go:build !grpc_test
// +build !grpc_test

package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/agent"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// ManagerClient Manager gRPC客户端(用于调用Manager的DaemonService)
type ManagerClient struct {
	conn   *grpc.ClientConn
	client daemonpb.DaemonServiceClient
	config *config.ManagerConfig
	logger *zap.Logger
}

// NewManagerClient 创建Manager gRPC客户端
func NewManagerClient(cfg *config.ManagerConfig, logger *zap.Logger) *ManagerClient {
	return &ManagerClient{
		config: cfg,
		logger: logger,
	}
}

// Connect 连接到Manager
func (c *ManagerClient) Connect(ctx context.Context) error {
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

	// 添加keepalive参数
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}))

	// 建立连接
	conn, err := grpc.DialContext(ctx, c.config.Address, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial manager: %w", err)
	}

	c.conn = conn
	c.client = daemonpb.NewDaemonServiceClient(conn)
	c.logger.Info("connected to manager daemon service", zap.String("address", c.config.Address))

	return nil
}

// Close 关闭连接
func (c *ManagerClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SyncAgentStates 同步Agent状态到Manager
func (c *ManagerClient) SyncAgentStates(ctx context.Context, nodeID string, states []*agent.AgentState) error {
	if c.client == nil {
		return fmt.Errorf("gRPC client not connected")
	}

	c.logger.Debug("syncing agent states to manager",
		zap.String("node_id", nodeID),
		zap.Int("count", len(states)))

	// 转换AgentState为protobuf AgentState
	var protoStates []*daemonpb.AgentState
	for _, state := range states {
		protoState := &daemonpb.AgentState{
			AgentId:       state.AgentID,
			Status:        string(state.Status),
			Pid:           int32(state.PID),
			LastHeartbeat: 0,
		}

		// 转换LastHeartbeat时间戳
		if !state.LastHeartbeat.IsZero() {
			protoState.LastHeartbeat = state.LastHeartbeat.Unix()
		}

		protoStates = append(protoStates, protoState)
	}

	// 构建同步请求
	req := &daemonpb.SyncAgentStatesRequest{
		NodeId: nodeID,
		States: protoStates,
	}

	// 调用gRPC服务
	resp, err := c.client.SyncAgentStates(ctx, req)
	if err != nil {
		c.logger.Error("failed to sync agent states", zap.Error(err))
		return fmt.Errorf("gRPC call failed: %w", err)
	}

	if !resp.Success {
		c.logger.Warn("agent states sync failed", zap.String("message", resp.Message))
		return fmt.Errorf("sync failed: %s", resp.Message)
	}

	c.logger.Debug("agent states synced successfully",
		zap.String("node_id", nodeID),
		zap.Int("count", len(states)))

	return nil
}
