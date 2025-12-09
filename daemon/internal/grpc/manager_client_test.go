//go:build grpc_test || e2e
// +build grpc_test e2e

package grpc

import (
	"context"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/agent"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"go.uber.org/zap"
)

// ManagerClient Manager gRPC客户端(用于调用Manager的DaemonService)
// 这是测试专用的 stub 实现，避免导入 manager 模块的 proto 包
type ManagerClient struct {
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

// Connect 连接到Manager (测试 stub)
func (c *ManagerClient) Connect(ctx context.Context) error {
	c.logger.Warn("ManagerClient.Connect called in test mode (stub implementation)")
	return nil
}

// Close 关闭连接 (测试 stub)
func (c *ManagerClient) Close() error {
	return nil
}

// SyncAgentStates 同步Agent状态到Manager (测试 stub)
func (c *ManagerClient) SyncAgentStates(ctx context.Context, nodeID string, states []*agent.AgentState) error {
	c.logger.Warn("ManagerClient.SyncAgentStates called in test mode (stub implementation)")
	return nil
}
