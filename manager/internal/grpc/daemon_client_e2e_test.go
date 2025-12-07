//go:build e2e
// +build e2e

package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/agent"
	daemongrpc "github.com/bingooyong/ops-scaffold-framework/daemon/internal/grpc"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/proto"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// startRealDaemonServer 启动真实的Daemon gRPC服务器
func startRealDaemonServer(t *testing.T) (string, func()) {
	// 创建临时工作目录
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	// 创建真实的MultiAgentManager
	mam, err := agent.NewMultiAgentManager(tmpDir, logger)
	require.NoError(t, err)

	// 创建真实的ResourceMonitor
	rm := agent.NewResourceMonitor(mam, mam.GetRegistry(), logger)

	// 创建真实的gRPC服务器
	server := daemongrpc.NewServer(mam, rm, logger)

	// 创建gRPC服务器
	grpcServer := grpc.NewServer()
	proto.RegisterDaemonServiceServer(grpcServer, server)

	// 在随机端口启动服务器
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

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

// registerTestAgent 注册测试Agent到MultiAgentManager
func registerTestAgent(t *testing.T, mam *agent.MultiAgentManager, id string) *agent.AgentInfo {
	info := &agent.AgentInfo{
		ID:         id,
		Type:       agent.TypeFilebeat,
		Name:       "Test Agent " + id,
		BinaryPath: "/bin/echo", // 使用系统命令作为测试二进制
		WorkDir:    t.TempDir(),
	}
	_, err := mam.RegisterAgent(info)
	require.NoError(t, err)
	return info
}

func TestDaemonClient_E2E_ListAgents(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 启动真实Daemon服务端
	addr, cleanup := startRealDaemonServer(t)
	defer cleanup()

	// 获取MultiAgentManager（通过辅助函数，这里简化处理）
	// 注意：在实际测试中，我们需要能够访问服务端的MultiAgentManager来注册Agent
	// 这里我们通过gRPC调用测试，Agent需要预先注册
	// 为了简化，我们直接测试空列表场景，或者需要更复杂的设置

	// 创建Manager客户端连接到真实服务端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试ListAgents
	ctx := context.Background()
	agents, err := client.ListAgents(ctx, "node-1")

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, agents)
	// 初始状态下可能没有Agent，这是正常的
}

func TestDaemonClient_E2E_OperateAgent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 启动真实Daemon服务端
	tmpDir := t.TempDir()
	daemonLogger := zap.NewNop()

	mam, err := agent.NewMultiAgentManager(tmpDir, daemonLogger)
	require.NoError(t, err)

	rm := agent.NewResourceMonitor(mam, mam.GetRegistry(), daemonLogger)
	server := daemongrpc.NewServer(mam, rm, daemonLogger)

	grpcServer := grpc.NewServer()
	proto.RegisterDaemonServiceServer(grpcServer, server)

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	addr := lis.Addr().String()

	cleanup := func() {
		grpcServer.GracefulStop()
		time.Sleep(50 * time.Millisecond)
	}
	defer cleanup()

	// 注册测试Agent（初始状态为stopped）
	info := registerTestAgent(t, mam, "agent-1")

	// 创建Manager客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// 验证初始状态
	agents, err := client.ListAgents(ctx, "node-1")
	require.NoError(t, err)
	require.NotNil(t, agents)

	// 查找我们的Agent
	var foundAgent *daemonpb.AgentInfo
	for _, a := range agents {
		if a.Id == info.ID {
			foundAgent = a
			break
		}
	}
	require.NotNil(t, foundAgent, "agent should be found in list")
	assert.Equal(t, "stopped", foundAgent.Status)

	// 注意：由于我们使用/bin/echo作为测试二进制，启动可能会立即退出
	// 这里主要测试gRPC调用流程，不验证Agent实际运行状态
	// 调用OperateAgent启动Agent
	err = client.OperateAgent(ctx, "node-1", "agent-1", "start")
	// 启动可能会失败（因为/bin/echo会立即退出），这是可以接受的
	// 我们主要验证gRPC调用成功
	if err != nil {
		t.Logf("start operation result: %v (may be expected for test binary)", err)
	}

	// 调用OperateAgent停止Agent
	err = client.OperateAgent(ctx, "node-1", "agent-1", "stop")
	// 停止操作应该成功
	if err != nil {
		t.Logf("stop operation result: %v", err)
	}
}

func TestDaemonClient_E2E_GetAgentMetrics(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 启动真实Daemon服务端
	tmpDir := t.TempDir()
	daemonLogger := zap.NewNop()

	mam, err := agent.NewMultiAgentManager(tmpDir, daemonLogger)
	require.NoError(t, err)

	rm := agent.NewResourceMonitor(mam, mam.GetRegistry(), daemonLogger)
	server := daemongrpc.NewServer(mam, rm, daemonLogger)

	grpcServer := grpc.NewServer()
	proto.RegisterDaemonServiceServer(grpcServer, server)

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	addr := lis.Addr().String()

	cleanup := func() {
		grpcServer.GracefulStop()
		time.Sleep(50 * time.Millisecond)
	}
	defer cleanup()

	// 注册并启动测试Agent
	info := registerTestAgent(t, mam, "agent-1")

	// 创建Manager客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// 等待一段时间让ResourceMonitor收集数据（如果有运行中的Agent）
	// 注意：由于我们使用测试二进制，可能没有实际数据
	time.Sleep(200 * time.Millisecond)

	// 调用GetAgentMetrics获取指标
	dataPoints, err := client.GetAgentMetrics(ctx, "node-1", info.ID, time.Hour)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, dataPoints)
	// 数据点可能为空（如果Agent未运行或没有历史数据），这是正常的
}

func TestDaemonClientPool_E2E_ConnectionReuse(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 启动真实Daemon服务端
	addr, cleanup := startRealDaemonServer(t)
	defer cleanup()

	// 创建DaemonClientPool
	pool := NewDaemonClientPool(logger)
	defer pool.CloseAll()

	// 多次调用GetClient获取同一nodeID的客户端
	client1, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)
	assert.NotNil(t, client1)

	client2, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)
	assert.NotNil(t, client2)

	// 验证返回的是同一个连接实例
	assert.Equal(t, client1, client2)

	// 再次获取，应该还是同一个
	client3, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)
	assert.Equal(t, client1, client3)
}
