package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockDaemonServer 模拟Daemon gRPC服务器
type mockDaemonServer struct {
	daemonpb.UnimplementedDaemonServiceServer
	agents           []*daemonpb.AgentInfo
	operationHandler func(agentID, operation string) error
	metricsHandler   func(agentID string, duration int64) ([]*daemonpb.ResourceDataPoint, error)
	delay            time.Duration
}

func (m *mockDaemonServer) ListAgents(ctx context.Context, req *daemonpb.ListAgentsRequest) (*daemonpb.ListAgentsResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return &daemonpb.ListAgentsResponse{
		Agents: m.agents,
	}, nil
}

func (m *mockDaemonServer) OperateAgent(ctx context.Context, req *daemonpb.AgentOperationRequest) (*daemonpb.AgentOperationResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.operationHandler != nil {
		if err := m.operationHandler(req.AgentId, req.Operation); err != nil {
			return &daemonpb.AgentOperationResponse{
				Success:      false,
				ErrorMessage: err.Error(),
			}, nil
		}
	}

	return &daemonpb.AgentOperationResponse{
		Success: true,
	}, nil
}

func (m *mockDaemonServer) GetAgentMetrics(ctx context.Context, req *daemonpb.AgentMetricsRequest) (*daemonpb.AgentMetricsResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.metricsHandler != nil {
		dataPoints, err := m.metricsHandler(req.AgentId, req.DurationSeconds)
		if err != nil {
			return nil, err
		}
		return &daemonpb.AgentMetricsResponse{
			AgentId:    req.AgentId,
			DataPoints: dataPoints,
		}, nil
	}

	return &daemonpb.AgentMetricsResponse{
		AgentId:    req.AgentId,
		DataPoints: []*daemonpb.ResourceDataPoint{},
	}, nil
}

// startMockServer 启动模拟服务器并返回地址
func startMockServer(t *testing.T, server *mockDaemonServer) string {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	s := grpc.NewServer()
	daemonpb.RegisterDaemonServiceServer(s, server)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("mock server error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	return lis.Addr().String()
}

func TestListAgents_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器
	mockAgents := []*daemonpb.AgentInfo{
		{
			Id:        "agent-1",
			Type:      "filebeat",
			Version:   "1.0.0",
			Status:    "running",
			Pid:       1234,
			StartTime: time.Now().Unix(),
		},
		{
			Id:        "agent-2",
			Type:      "telegraf",
			Version:   "2.0.0",
			Status:    "stopped",
			Pid:       0,
			StartTime: 0,
		},
	}

	server := &mockDaemonServer{
		agents: mockAgents,
	}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试ListAgents
	ctx := context.Background()
	agents, err := client.ListAgents(ctx, "node-1")

	// 验证结果
	assert.NoError(t, err)
	assert.Len(t, agents, 2)
	assert.Equal(t, "agent-1", agents[0].Id)
	assert.Equal(t, "agent-2", agents[1].Id)
}

func TestListAgents_Timeout(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器，设置延迟超过超时时间
	server := &mockDaemonServer{
		delay: 15 * time.Second, // 超过10秒超时
	}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试ListAgents，应该超时
	ctx := context.Background()
	agents, err := client.ListAgents(ctx, "node-1")

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, agents)
	assert.Equal(t, ErrTimeout, err)
}

func TestListAgents_ConnectionError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 使用无效地址
	client, err := NewDaemonClient("127.0.0.1:99999", logger)
	require.NoError(t, err) // 连接创建不会立即失败
	defer client.Close()

	// 测试ListAgents，应该连接失败
	ctx := context.Background()
	agents, err := client.ListAgents(ctx, "node-1")

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, agents)
}

func TestOperateAgent_Start(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器
	server := &mockDaemonServer{
		operationHandler: func(agentID, operation string) error {
			if agentID != "agent-1" {
				return status.Errorf(codes.NotFound, "agent not found")
			}
			if operation != "start" {
				return status.Errorf(codes.InvalidArgument, "invalid operation")
			}
			return nil
		},
	}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试启动Agent
	ctx := context.Background()
	err = client.OperateAgent(ctx, "node-1", "agent-1", "start")

	// 验证结果
	assert.NoError(t, err)
}

func TestOperateAgent_Stop(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器
	server := &mockDaemonServer{
		operationHandler: func(agentID, operation string) error {
			if operation != "stop" {
				return status.Errorf(codes.InvalidArgument, "invalid operation")
			}
			return nil
		},
	}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试停止Agent
	ctx := context.Background()
	err = client.OperateAgent(ctx, "node-1", "agent-1", "stop")

	// 验证结果
	assert.NoError(t, err)
}

func TestOperateAgent_Restart(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器
	server := &mockDaemonServer{
		operationHandler: func(agentID, operation string) error {
			if operation != "restart" {
				return status.Errorf(codes.InvalidArgument, "invalid operation")
			}
			return nil
		},
	}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试重启Agent
	ctx := context.Background()
	err = client.OperateAgent(ctx, "node-1", "agent-1", "restart")

	// 验证结果
	assert.NoError(t, err)
}

func TestOperateAgent_NotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器，返回NotFound错误
	server := &mockDaemonServer{
		operationHandler: func(agentID, operation string) error {
			return status.Errorf(codes.NotFound, "agent not found")
		},
	}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试操作不存在的Agent
	ctx := context.Background()
	err = client.OperateAgent(ctx, "node-1", "non-existent", "start")

	// 验证结果
	assert.Error(t, err)
	assert.Equal(t, ErrAgentNotFound, err)
}

func TestOperateAgent_InvalidOperation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器
	server := &mockDaemonServer{}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试无效操作
	ctx := context.Background()
	err = client.OperateAgent(ctx, "node-1", "agent-1", "invalid-op")

	// 验证结果
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid operation")
}

func TestOperateAgent_InvalidParameters(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器
	server := &mockDaemonServer{}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// 测试空nodeID
	err = client.OperateAgent(ctx, "", "agent-1", "start")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nodeID is required")

	// 测试空agentID
	err = client.OperateAgent(ctx, "node-1", "", "start")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agentID is required")

	// 测试空operation
	err = client.OperateAgent(ctx, "node-1", "agent-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation is required")
}

func TestGetAgentMetrics_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器
	mockDataPoints := []*daemonpb.ResourceDataPoint{
		{
			Timestamp:      time.Now().Unix(),
			Cpu:            50.5,
			MemoryRss:      1024 * 1024 * 100, // 100MB
			MemoryVms:      1024 * 1024 * 200, // 200MB
			DiskReadBytes:  1024 * 1024 * 10,
			DiskWriteBytes: 1024 * 1024 * 5,
			OpenFiles:      10,
		},
		{
			Timestamp:      time.Now().Unix() - 60,
			Cpu:            60.0,
			MemoryRss:      1024 * 1024 * 110,
			MemoryVms:      1024 * 1024 * 210,
			DiskReadBytes:  1024 * 1024 * 12,
			DiskWriteBytes: 1024 * 1024 * 6,
			OpenFiles:      12,
		},
	}

	server := &mockDaemonServer{
		metricsHandler: func(agentID string, duration int64) ([]*daemonpb.ResourceDataPoint, error) {
			return mockDataPoints, nil
		},
	}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试获取Agent指标
	ctx := context.Background()
	dataPoints, err := client.GetAgentMetrics(ctx, "node-1", "agent-1", time.Hour)

	// 验证结果
	assert.NoError(t, err)
	assert.Len(t, dataPoints, 2)
	assert.Equal(t, 50.5, dataPoints[0].Cpu)
}

func TestGetAgentMetrics_NotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器，返回NotFound错误
	server := &mockDaemonServer{
		metricsHandler: func(agentID string, duration int64) ([]*daemonpb.ResourceDataPoint, error) {
			return nil, status.Errorf(codes.NotFound, "agent not found")
		},
	}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	// 测试获取不存在的Agent指标
	ctx := context.Background()
	dataPoints, err := client.GetAgentMetrics(ctx, "node-1", "non-existent", time.Hour)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, dataPoints)
	assert.Equal(t, ErrAgentNotFound, err)
}

func TestGetAgentMetrics_InvalidParameters(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟服务器
	server := &mockDaemonServer{}
	addr := startMockServer(t, server)

	// 创建客户端
	client, err := NewDaemonClient(addr, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// 测试空nodeID
	_, err = client.GetAgentMetrics(ctx, "", "agent-1", time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nodeID is required")

	// 测试空agentID
	_, err = client.GetAgentMetrics(ctx, "node-1", "", time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agentID is required")

	// 测试无效duration
	_, err = client.GetAgentMetrics(ctx, "node-1", "agent-1", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duration must be greater than 0")

	_, err = client.GetAgentMetrics(ctx, "node-1", "agent-1", -time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duration must be greater than 0")
}

func TestDaemonClientPool_GetClient(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建连接池
	pool := NewDaemonClientPool(logger)

	// 创建模拟服务器
	server := &mockDaemonServer{}
	addr := startMockServer(t, server)

	// 获取客户端
	client1, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)
	assert.NotNil(t, client1)

	// 再次获取同一节点的客户端，应该复用连接
	client2, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)
	assert.NotNil(t, client2)
	assert.Equal(t, client1, client2) // 应该是同一个实例

	// 清理
	pool.CloseAll()
}

func TestDaemonClientPool_ReuseConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建连接池
	pool := NewDaemonClientPool(logger)

	// 创建模拟服务器
	server := &mockDaemonServer{
		agents: []*daemonpb.AgentInfo{
			{Id: "agent-1", Status: "running"},
		},
	}
	addr := startMockServer(t, server)

	// 获取客户端并调用
	client, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)

	ctx := context.Background()
	agents, err := client.ListAgents(ctx, "node-1")
	require.NoError(t, err)
	assert.Len(t, agents, 1)

	// 再次获取同一节点的客户端，应该复用连接
	client2, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)
	assert.Equal(t, client, client2)

	// 再次调用，应该正常工作
	agents2, err := client2.ListAgents(ctx, "node-1")
	require.NoError(t, err)
	assert.Len(t, agents2, 1)

	// 清理
	pool.CloseAll()
}

func TestDaemonClientPool_CloseClient(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建连接池
	pool := NewDaemonClientPool(logger)

	// 创建模拟服务器
	server := &mockDaemonServer{}
	addr := startMockServer(t, server)

	// 获取客户端
	client, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)

	// 关闭客户端
	err = pool.CloseClient("node-1")
	assert.NoError(t, err)

	// 再次获取，应该创建新连接
	client2, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)
	assert.NotEqual(t, client, client2) // 应该是新实例

	// 清理
	pool.CloseAll()
}

func TestDaemonClientPool_CloseAll(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建连接池
	pool := NewDaemonClientPool(logger)

	// 创建模拟服务器
	server := &mockDaemonServer{}
	addr := startMockServer(t, server)

	// 创建多个客户端
	_, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)

	_, err = pool.GetClient("node-2", addr)
	require.NoError(t, err)

	// 关闭所有客户端
	pool.CloseAll()

	// 再次获取，应该创建新连接
	client, err := pool.GetClient("node-1", addr)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// 清理
	pool.CloseAll()
}

func TestDaemonClientPool_InvalidParameters(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建连接池
	pool := NewDaemonClientPool(logger)

	// 测试空nodeID
	_, err := pool.GetClient("", "127.0.0.1:9091")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nodeID is required")

	// 测试空address
	_, err = pool.GetClient("node-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address is required")
}

func TestConvertGRPCError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{
			name:     "NotFound",
			err:      status.Errorf(codes.NotFound, "not found"),
			expected: ErrAgentNotFound,
		},
		{
			name:     "InvalidArgument",
			err:      status.Errorf(codes.InvalidArgument, "invalid"),
			expected: ErrInvalidArgument,
		},
		{
			name:     "DeadlineExceeded",
			err:      status.Errorf(codes.DeadlineExceeded, "timeout"),
			expected: ErrTimeout,
		},
		{
			name:     "Unavailable",
			err:      status.Errorf(codes.Unavailable, "unavailable"),
			expected: ErrConnectionFailed,
		},
		{
			name:     "Internal",
			err:      status.Errorf(codes.Internal, "internal error"),
			expected: status.Errorf(codes.Internal, "internal error"), // 返回原始错误
		},
		{
			name:     "NonGRPCError",
			err:      assert.AnError,
			expected: assert.AnError, // 返回原始错误
		},
		{
			name:     "NilError",
			err:      nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertGRPCError(tt.err)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
