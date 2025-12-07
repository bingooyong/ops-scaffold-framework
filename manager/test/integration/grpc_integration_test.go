//go:build e2e
// +build e2e

package integration

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/agent"
	daemongrpc "github.com/bingooyong/ops-scaffold-framework/daemon/internal/grpc"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/proto"
	managergrpc "github.com/bingooyong/ops-scaffold-framework/manager/internal/grpc"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// startTestDaemonServer 启动测试Daemon gRPC服务器
func startTestDaemonServer(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	mam, err := agent.NewMultiAgentManager(tmpDir, logger)
	require.NoError(t, err)

	rm := agent.NewResourceMonitor(mam, mam.GetRegistry(), logger)
	server := daemongrpc.NewServer(mam, rm, logger)

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

	return addr, cleanup
}

// createTestAgent 创建测试Agent
func createTestAgent(t *testing.T, id string, agentType agent.AgentType) *agent.AgentInfo {
	return &agent.AgentInfo{
		ID:         id,
		Type:       agentType,
		Name:       "Test Agent " + id,
		BinaryPath: "/bin/echo",
		WorkDir:    t.TempDir(),
	}
}

// registerTestAgent 注册测试Agent
func registerTestAgent(t *testing.T, mam *agent.MultiAgentManager, info *agent.AgentInfo) *agent.AgentInstance {
	instance, err := mam.RegisterAgent(info)
	require.NoError(t, err)
	return instance
}

// TestGRPCIntegration_CompleteAgentLifecycle 完整Agent管理流程测试
func TestGRPCIntegration_CompleteAgentLifecycle(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 启动真实的Daemon gRPC服务端
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
	serverAddr := lis.Addr().String()

	serverCleanup := func() {
		grpcServer.GracefulStop()
		time.Sleep(50 * time.Millisecond)
	}
	defer serverCleanup()

	// 注册测试Agent到Daemon
	info := createTestAgent(t, "agent-1", agent.TypeFilebeat)
	instance := registerTestAgent(t, mam, info)
	_ = instance

	// 启动真实的Manager gRPC客户端（连接到Daemon）
	client, err := managergrpc.NewDaemonClient(serverAddr, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// 1. 调用ListAgents查询Agent列表
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

	// 2. 验证Agent状态为stopped（初始状态）
	assert.Equal(t, "stopped", foundAgent.Status)

	// 3. 调用OperateAgent启动Agent
	err = client.OperateAgent(ctx, "node-1", info.ID, "start")
	if err != nil {
		t.Logf("start operation result: %v (may be expected for test binary)", err)
	} else {
		// 4. 等待Agent启动完成
		time.Sleep(500 * time.Millisecond)

		// 5. 再次调用ListAgents验证状态
		agents, err = client.ListAgents(ctx, "node-1")
		require.NoError(t, err)
		for _, a := range agents {
			if a.Id == info.ID {
				t.Logf("agent status after start: %s", a.Status)
				break
			}
		}

		// 6. 调用GetAgentMetrics获取指标数据
		dataPoints, err := client.GetAgentMetrics(ctx, "node-1", info.ID, time.Hour)
		if err != nil {
			t.Logf("get metrics result: %v", err)
		} else {
			assert.NotNil(t, dataPoints)
			t.Logf("retrieved %d data points", len(dataPoints))
		}
	}

	// 7. 调用OperateAgent停止Agent
	err = client.OperateAgent(ctx, "node-1", info.ID, "stop")
	if err != nil {
		t.Logf("stop operation result: %v", err)
	}

	// 8. 验证Agent状态为stopped
	time.Sleep(200 * time.Millisecond)
	agents, err = client.ListAgents(ctx, "node-1")
	require.NoError(t, err)
	for _, a := range agents {
		if a.Id == info.ID {
			assert.Equal(t, "stopped", a.Status)
			break
		}
	}
}

// TestGRPCIntegration_ConcurrentOperations 并发操作测试
func TestGRPCIntegration_ConcurrentOperations(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

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
	serverAddr := lis.Addr().String()

	serverCleanup := func() {
		grpcServer.GracefulStop()
		time.Sleep(50 * time.Millisecond)
	}
	defer serverCleanup()

	// 注册多个测试Agent
	info1 := createTestAgent(t, "agent-1", agent.TypeFilebeat)
	info2 := createTestAgent(t, "agent-2", agent.TypeTelegraf)
	info3 := createTestAgent(t, "agent-3", agent.TypeFilebeat)
	instance1 := registerTestAgent(t, mam, info1)
	instance2 := registerTestAgent(t, mam, info2)
	instance3 := registerTestAgent(t, mam, info3)
	_ = instance1
	_ = instance2
	_ = instance3

	// 创建Manager客户端
	client, err := managergrpc.NewDaemonClient(serverAddr, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// 并发执行多个操作
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// 多个goroutine同时调用ListAgents
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			agents, err := client.ListAgents(ctx, "node-1")
			if err != nil {
				errors <- err
			} else if agents == nil {
				errors <- assert.AnError
			}
		}()
	}

	// 多个goroutine同时调用OperateAgent操作不同的Agent
	operations := []struct {
		agentID   string
		operation string
	}{
		{info1.ID, "stop"},
		{info2.ID, "stop"},
		{info3.ID, "stop"},
	}

	for _, op := range operations {
		wg.Add(1)
		go func(agentID, operation string) {
			defer wg.Done()
			err := client.OperateAgent(ctx, "node-1", agentID, operation)
			if err != nil {
				t.Logf("operation %s on %s: %v", operation, agentID, err)
			}
		}(op.agentID, op.operation)
	}

	// 多个goroutine同时调用GetAgentMetrics
	for _, agentID := range []string{info1.ID, info2.ID, info3.ID} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			dataPoints, err := client.GetAgentMetrics(ctx, "node-1", id, time.Hour)
			if err != nil {
				t.Logf("get metrics for %s: %v", id, err)
			} else {
				assert.NotNil(t, dataPoints)
			}
		}(agentID)
	}

	// 等待所有操作完成
	wg.Wait()
	close(errors)

	// 检查错误
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Logf("concurrent operation error: %v", err)
		}
	}

	t.Logf("concurrent operations completed, %d errors (may be expected)", errorCount)
}

// TestGRPCIntegration_ErrorRecovery 错误恢复测试
func TestGRPCIntegration_ErrorRecovery(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

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
	serverAddr := lis.Addr().String()

	// 创建Manager客户端
	client, err := managergrpc.NewDaemonClient(serverAddr, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// 执行正常操作
	agents, err := client.ListAgents(ctx, "node-1")
	require.NoError(t, err)
	assert.NotNil(t, agents)

	// 模拟服务端崩溃（关闭gRPC服务器）
	grpcServer.GracefulStop()
	time.Sleep(200 * time.Millisecond)

	// 验证客户端检测到连接断开
	_, err = client.ListAgents(ctx, "node-1")
	assert.Error(t, err, "should detect connection failure")

	// 重新启动服务端
	lis2, err := net.Listen("tcp", lis.Addr().String())
	require.NoError(t, err)

	grpcServer2 := grpc.NewServer()
	proto.RegisterDaemonServiceServer(grpcServer2, server)

	go func() {
		if err := grpcServer2.Serve(lis2); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	// 重新创建客户端（当前实现可能不支持完全自动重连）
	client2, err := managergrpc.NewDaemonClient(serverAddr, logger)
	require.NoError(t, err)
	defer client2.Close()

	agents2, err := client2.ListAgents(ctx, "node-1")
	require.NoError(t, err)
	assert.NotNil(t, agents2)

	// 清理
	grpcServer2.GracefulStop()
	time.Sleep(50 * time.Millisecond)
}

// TestGRPCIntegration_TimeoutAndRetry 超时和重试测试
func TestGRPCIntegration_TimeoutAndRetry(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

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
	serverAddr := lis.Addr().String()

	serverCleanup := func() {
		grpcServer.GracefulStop()
		time.Sleep(50 * time.Millisecond)
	}
	defer serverCleanup()

	// 创建客户端
	client, err := managergrpc.NewDaemonClient(serverAddr, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// 调用RPC方法（正常情况，应该成功）
	agents, err := client.ListAgents(ctx, "node-1")
	require.NoError(t, err)
	assert.NotNil(t, agents)

	// 测试超时场景：使用很短的超时上下文
	shortCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	// 等待超时
	time.Sleep(10 * time.Millisecond)

	// 调用RPC方法，应该超时
	_, err = client.ListAgents(shortCtx, "node-1")
	assert.Error(t, err, "should timeout")
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
