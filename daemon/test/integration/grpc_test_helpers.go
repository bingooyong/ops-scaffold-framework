package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/agent"
	daemongrpc "github.com/bingooyong/ops-scaffold-framework/daemon/internal/grpc"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// startTestDaemonServer 启动测试Daemon gRPC服务器
// 返回服务器地址和清理函数
func startTestDaemonServer(t *testing.T) (string, func()) {
	// 创建临时目录
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	// 创建MultiAgentManager
	mam, err := agent.NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}

	// 创建ResourceMonitor
	rm := agent.NewResourceMonitor(mam, mam.GetRegistry(), logger)

	// 创建gRPC服务器实现
	server := daemongrpc.NewServer(mam, rm, logger)

	// 创建gRPC服务器
	grpcServer := grpc.NewServer()
	proto.RegisterDaemonServiceServer(grpcServer, server)

	// 在随机端口启动服务器
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	// 启动服务器
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 返回地址和清理函数
	addr := lis.Addr().String()
	cleanup := func() {
		grpcServer.GracefulStop()
		time.Sleep(50 * time.Millisecond)
	}

	return addr, cleanup
}

// startTestManagerServer 启动测试Manager gRPC服务器
// 注意：这个函数需要Manager端的实现，当前可能不存在，先提供框架
func startTestManagerServer(t *testing.T) (string, func()) {
	// TODO: 实现Manager gRPC服务器启动逻辑
	// 这需要创建临时数据库、AgentService、DaemonServer等
	t.Skip("Manager gRPC server setup not yet implemented")
	return "", func() {}
}

// createTestAgent 创建测试Agent
func createTestAgent(t *testing.T, id string, agentType agent.AgentType) *agent.AgentInfo {
	return &agent.AgentInfo{
		ID:         id,
		Type:       agentType,
		Name:       fmt.Sprintf("Test Agent %s", id),
		BinaryPath: "/bin/echo", // 使用系统命令作为测试二进制
		WorkDir:    t.TempDir(),
	}
}

// createTestAgentWithBinary 创建测试Agent（指定二进制路径）
func createTestAgentWithBinary(t *testing.T, id string, agentType agent.AgentType, binaryPath string) *agent.AgentInfo {
	return &agent.AgentInfo{
		ID:         id,
		Type:       agentType,
		Name:       fmt.Sprintf("Test Agent %s", id),
		BinaryPath: binaryPath,
		WorkDir:    t.TempDir(),
	}
}

// waitForCondition 等待条件满足
// condition: 返回true表示条件满足
// timeout: 超时时间
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		select {
		case <-ticker.C:
			// 继续等待
		case <-time.After(time.Until(deadline)):
			t.Fatalf("condition not met within timeout %v", timeout)
		}
	}

	t.Fatalf("condition not met within timeout %v", timeout)
}

// waitForAgentStatus 等待Agent状态变为指定状态
// server: Daemon gRPC服务器实例
func waitForAgentStatus(
	t *testing.T,
	server *daemongrpc.Server,
	agentID string,
	expectedStatus string,
	timeout time.Duration,
) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	waitForCondition(t, func() bool {
		req := &proto.ListAgentsRequest{}
		resp, err := server.ListAgents(ctx, req)
		if err != nil {
			return false
		}

		for _, agent := range resp.Agents {
			if agent.Id == agentID {
				return agent.Status == expectedStatus
			}
		}
		return false
	}, timeout)
}

// registerTestAgent 注册测试Agent到MultiAgentManager
func registerTestAgent(t *testing.T, mam *agent.MultiAgentManager, info *agent.AgentInfo) *agent.AgentInstance {
	instance, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("failed to register agent %s: %v", info.ID, err)
	}
	return instance
}
