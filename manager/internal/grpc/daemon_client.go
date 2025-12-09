package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	// defaultTimeout 默认超时时间
	defaultTimeout = 30 * time.Second
	// operateAgentTimeout Agent操作超时时间(需要大于Agent优雅停止的30秒等待时间)
	// 设置为90秒，避免与Daemon的60-120秒keepalive冲突，提供足够的缓冲时间
	operateAgentTimeout = 90 * time.Second
	// keepaliveTime keepalive时间间隔(设置为45秒，避免与操作超时冲突)
	keepaliveTime = 45 * time.Second
	// keepaliveTimeout keepalive超时时间
	keepaliveTimeout = 15 * time.Second
	// maxMsgSize 最大消息大小(10MB)
	maxMsgSize = 10 * 1024 * 1024
	// initialWindowSize 初始窗口大小(1MB)
	initialWindowSize = 1 << 20
)

// DaemonClient Daemon gRPC客户端
type DaemonClient struct {
	conn    *grpc.ClientConn
	client  daemonpb.DaemonServiceClient
	address string
	logger  *zap.Logger
	mu      sync.RWMutex // 保护连接状态
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDaemonClient 创建Daemon gRPC客户端
func NewDaemonClient(address string, logger *zap.Logger) (*DaemonClient, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// 配置keepalive参数
	keepaliveParams := keepalive.ClientParameters{
		Time:                keepaliveTime,
		Timeout:             keepaliveTimeout,
		PermitWithoutStream: true,
	}

	// 配置重试策略
	retryPolicy := `{
		"methodConfig": [{
			"name": [{"service": "daemon.DaemonService"}],
			"waitForReady": true,
			"retryPolicy": {
				"MaxAttempts": 3,
				"InitialBackoff": "0.1s",
				"MaxBackoff": "1s",
				"BackoffMultiplier": 2.0,
				"RetryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
			}
		}]
	}`

	// 创建gRPC连接
	// 注意：根据任务说明，Daemon监听9091端口，这里使用insecure credentials，实际生产环境应使用TLS
	conn, err := grpc.Dial(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepaliveParams),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMsgSize),
			grpc.MaxCallSendMsgSize(maxMsgSize),
		),
		grpc.WithInitialWindowSize(initialWindowSize),
		grpc.WithInitialConnWindowSize(initialWindowSize),
		grpc.WithUnaryInterceptor(UnaryClientInterceptor(logger)),
		grpc.WithDefaultServiceConfig(retryPolicy),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial daemon at %s: %w", address, err)
	}

	// 创建客户端
	client := daemonpb.NewDaemonServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())

	dc := &DaemonClient{
		conn:    conn,
		client:  client,
		address: address,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}

	// 启动连接状态监控
	go dc.monitorConnection()

	return dc, nil
}

// ensureConnection 确保连接可用，如果断开则尝试重连
func (c *DaemonClient) ensureConnection(ctx context.Context) error {
	c.mu.RLock()
	state := c.conn.GetState()
	c.mu.RUnlock()

	// 如果连接已就绪，直接返回
	if state == connectivity.Ready {
		return nil
	}

	// 如果连接正在连接中，等待（使用超时避免永久阻塞）
	if state == connectivity.Connecting {
		// 创建带超时的 context，最多等待 5 秒
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		
		if !c.conn.WaitForStateChange(waitCtx, state) {
			// 超时或 context 取消
			if waitCtx.Err() == context.DeadlineExceeded {
				c.logger.Warn("connection wait timeout",
					zap.String("address", c.address),
					zap.String("state", state.String()))
				return fmt.Errorf("%w: connection wait timeout", ErrConnectionFailed)
			}
			return ErrConnectionFailed
		}
		state = c.conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
	}

	// 如果连接断开或失败，尝试重连
	if state == connectivity.TransientFailure || state == connectivity.Idle || state == connectivity.Shutdown {
		c.mu.Lock()
		defer c.mu.Unlock()

		// 再次检查状态（可能在获取锁的过程中状态已改变）
		state = c.conn.GetState()
		if state == connectivity.Ready {
			return nil
		}

		// 关闭旧连接
		if c.conn != nil {
			if err := c.conn.Close(); err != nil {
				c.logger.Warn("failed to close old connection",
					zap.String("address", c.address),
					zap.Error(err))
				// 即使关闭失败，也继续重连
			}
		}

		// 重新创建连接
		keepaliveParams := keepalive.ClientParameters{
			Time:                keepaliveTime,
			Timeout:             keepaliveTimeout,
			PermitWithoutStream: true,
		}

		// 配置重试策略
		retryPolicy := `{
			"methodConfig": [{
				"name": [{"service": "daemon.DaemonService"}],
				"waitForReady": true,
				"retryPolicy": {
					"MaxAttempts": 3,
					"InitialBackoff": "0.1s",
					"MaxBackoff": "1s",
					"BackoffMultiplier": 2.0,
					"RetryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
				}
			}]
		}`

		// 创建带超时的context
		dialCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(
			dialCtx,
			c.address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithKeepaliveParams(keepaliveParams),
			grpc.WithBlock(), // 等待连接建立
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(maxMsgSize),
				grpc.MaxCallSendMsgSize(maxMsgSize),
			),
			grpc.WithInitialWindowSize(initialWindowSize),
			grpc.WithInitialConnWindowSize(initialWindowSize),
			grpc.WithUnaryInterceptor(UnaryClientInterceptor(c.logger)),
			grpc.WithDefaultServiceConfig(retryPolicy),
		)
		if err != nil {
			c.logger.Error("failed to reconnect to daemon",
				zap.String("address", c.address),
				zap.Error(err))
			return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
		}

		c.conn = conn
		c.client = daemonpb.NewDaemonServiceClient(conn)
		c.logger.Info("reconnected to daemon", zap.String("address", c.address))
	}

	return nil
}

// ListAgents 列举所有Agent
func (c *DaemonClient) ListAgents(ctx context.Context, nodeID string) ([]*daemonpb.AgentInfo, error) {
	// 确保连接可用
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	// 设置超时
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// 调用gRPC方法
	response, err := c.client.ListAgents(timeoutCtx, &daemonpb.ListAgentsRequest{})
	if err != nil {
		c.logger.Warn("failed to list agents",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return nil, convertGRPCError(err)
	}

	// 记录日志
	c.logger.Debug("list agents success",
		zap.String("node_id", nodeID),
		zap.Int("count", len(response.Agents)))

	return response.Agents, nil
}

// OperateAgent 操作Agent(启动/停止/重启)
func (c *DaemonClient) OperateAgent(ctx context.Context, nodeID, agentID, operation string) error {
	// 参数验证
	if nodeID == "" {
		return fmt.Errorf("%w: nodeID is required", ErrInvalidArgument)
	}
	if agentID == "" {
		return fmt.Errorf("%w: agentID is required", ErrInvalidArgument)
	}
	if operation == "" {
		return fmt.Errorf("%w: operation is required", ErrInvalidArgument)
	}

	// 验证操作类型
	validOperations := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
	}
	if !validOperations[operation] {
		return fmt.Errorf("%w: invalid operation %s, must be one of: start, stop, restart", ErrInvalidArgument, operation)
	}

	// 确保连接可用
	if err := c.ensureConnection(ctx); err != nil {
		return err
	}

	// 设置超时 (Agent操作需要更长的超时时间,因为优雅停止可能需要30秒)
	timeoutCtx, cancel := context.WithTimeout(ctx, operateAgentTimeout)
	defer cancel()

	// 构建请求
	req := &daemonpb.AgentOperationRequest{
		AgentId:   agentID,
		Operation: operation,
	}

	// 调用gRPC方法
	c.logger.Debug("calling daemon OperateAgent gRPC",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.String("operation", operation),
		zap.String("address", c.address))

	callStart := time.Now()
	response, err := c.client.OperateAgent(timeoutCtx, req)
	callDuration := time.Since(callStart)

	if err != nil {
		// 检查是否是超时错误
		if err == context.DeadlineExceeded || err == context.Canceled {
			c.logger.Error("operate agent timeout or cancelled",
				zap.String("node_id", nodeID),
				zap.String("agent_id", agentID),
				zap.String("operation", operation),
				zap.String("address", c.address),
				zap.Duration("duration", callDuration),
				zap.Error(err))
		} else {
			c.logger.Error("failed to operate agent",
				zap.String("node_id", nodeID),
				zap.String("agent_id", agentID),
				zap.String("operation", operation),
				zap.String("address", c.address),
				zap.Duration("duration", callDuration),
				zap.Error(err))
		}
		return convertGRPCError(err)
	}

	// 检查响应
	if !response.Success {
		errMsg := response.ErrorMessage
		if errMsg == "" {
			errMsg = fmt.Sprintf("operation %s failed", operation)
		}
		c.logger.Error("agent operation failed",
			zap.String("node_id", nodeID),
			zap.String("agent_id", agentID),
			zap.String("operation", operation),
			zap.String("address", c.address),
			zap.Duration("duration", callDuration),
			zap.String("error", errMsg))
		return fmt.Errorf("agent operation failed: %s", errMsg)
	}

	// 记录日志
	c.logger.Info("agent operation success",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.String("operation", operation),
		zap.String("address", c.address),
		zap.Duration("duration", callDuration))

	return nil
}

// GetAgentMetrics 获取Agent资源使用指标
func (c *DaemonClient) GetAgentMetrics(ctx context.Context, nodeID, agentID string, duration time.Duration) ([]*daemonpb.ResourceDataPoint, error) {
	// 参数验证
	if nodeID == "" {
		return nil, fmt.Errorf("%w: nodeID is required", ErrInvalidArgument)
	}
	if agentID == "" {
		return nil, fmt.Errorf("%w: agentID is required", ErrInvalidArgument)
	}
	if duration <= 0 {
		return nil, fmt.Errorf("%w: duration must be greater than 0", ErrInvalidArgument)
	}

	// 确保连接可用
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	// 设置超时
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// 构建请求
	req := &daemonpb.AgentMetricsRequest{
		AgentId:         agentID,
		DurationSeconds: int64(duration.Seconds()),
	}

	// 调用gRPC方法
	response, err := c.client.GetAgentMetrics(timeoutCtx, req)
	if err != nil {
		c.logger.Warn("failed to get agent metrics",
			zap.String("node_id", nodeID),
			zap.String("agent_id", agentID),
			zap.Error(err))
		return nil, convertGRPCError(err)
	}

	// 记录日志
	c.logger.Debug("get agent metrics success",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.Int("data_points", len(response.DataPoints)))

	return response.DataPoints, nil
}

// Close 关闭客户端连接
func (c *DaemonClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 先取消监控goroutine
	if c.cancel != nil {
		c.cancel()
	}

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// monitorConnection 监控连接状态变化
func (c *DaemonClient) monitorConnection() {
	c.mu.RLock()
	state := c.conn.GetState()
	c.mu.RUnlock()

	c.logger.Debug("initial connection state",
		zap.String("address", c.address),
		zap.String("state", state.String()))

	for {
		// 等待状态变化
		c.mu.RLock()
		if !c.conn.WaitForStateChange(c.ctx, state) {
			c.mu.RUnlock()
			// context已取消，退出监控
			return
		}

		state = c.conn.GetState()
		c.mu.RUnlock()

		c.logger.Info("connection state changed",
			zap.String("address", c.address),
			zap.String("new_state", state.String()))

		// 如果连接失败，记录警告
		if state == connectivity.TransientFailure {
			c.logger.Warn("connection in transient failure state",
				zap.String("address", c.address))
		} else if state == connectivity.Shutdown {
			c.logger.Warn("connection shutdown",
				zap.String("address", c.address))
			return
		}
	}
}

// DaemonClientPool Daemon客户端连接池
type DaemonClientPool struct {
	clients map[string]*DaemonClient
	mu      sync.RWMutex
	logger  *zap.Logger
}

// NewDaemonClientPool 创建Daemon客户端连接池
func NewDaemonClientPool(logger *zap.Logger) *DaemonClientPool {
	return &DaemonClientPool{
		clients: make(map[string]*DaemonClient),
		logger:  logger,
	}
}

// GetClient 获取或创建客户端
// nodeID: 节点ID，用于标识连接
// address: Daemon地址，格式为 "host:port"
func (p *DaemonClientPool) GetClient(nodeID, address string) (service.DaemonClient, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("%w: nodeID is required", ErrInvalidArgument)
	}
	if address == "" {
		return nil, fmt.Errorf("%w: address is required", ErrInvalidArgument)
	}

	// 先尝试读锁获取现有连接
	p.mu.RLock()
	if client, exists := p.clients[nodeID]; exists {
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	// 使用写锁创建新连接
	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查，可能在获取写锁的过程中其他goroutine已创建
	if client, exists := p.clients[nodeID]; exists {
		return client, nil
	}

	// 创建新客户端
	client, err := NewDaemonClient(address, p.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create daemon client: %w", err)
	}

	// 存储到map
	p.clients[nodeID] = client

	p.logger.Info("created new daemon client",
		zap.String("node_id", nodeID),
		zap.String("address", address))

	return client, nil
}

// CloseClient 关闭指定节点的客户端
func (p *DaemonClientPool) CloseClient(nodeID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	client, exists := p.clients[nodeID]
	if !exists {
		return nil // 客户端不存在，无需关闭
	}

	// 关闭连接
	if err := client.Close(); err != nil {
		p.logger.Warn("failed to close daemon client",
			zap.String("node_id", nodeID),
			zap.Error(err))
	}

	// 从map中删除
	delete(p.clients, nodeID)

	p.logger.Info("closed daemon client", zap.String("node_id", nodeID))
	return nil
}

// CloseAll 关闭所有客户端
func (p *DaemonClientPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for nodeID, client := range p.clients {
		if err := client.Close(); err != nil {
			p.logger.Warn("failed to close daemon client",
				zap.String("node_id", nodeID),
				zap.Error(err))
		}
	}

	// 清空map
	p.clients = make(map[string]*DaemonClient)

	p.logger.Info("closed all daemon clients")
}
