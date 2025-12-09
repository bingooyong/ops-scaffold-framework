package integration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
)

// MockDaemonClient Mock Daemon客户端实现
type MockDaemonClient struct {
	mu sync.RWMutex

	// 可配置的响应
	listAgentsResponse      []*daemonpb.AgentInfo
	operateAgentResponse    *daemonpb.AgentOperationResponse
	getAgentLogsResponse    []string
	getAgentMetricsResponse []*daemonpb.ResourceDataPoint

	// 可配置的错误
	listAgentsError      error
	operateAgentError    error
	getAgentLogsError    error
	getAgentMetricsError error

	// 记录调用次数（用于测试）
	listAgentsCallCount      int
	operateAgentCallCount    int
	getAgentLogsCallCount    int
	getAgentMetricsCallCount int
}

// NewMockDaemonClient 创建Mock Daemon客户端
func NewMockDaemonClient() *MockDaemonClient {
	return &MockDaemonClient{
		listAgentsResponse:      make([]*daemonpb.AgentInfo, 0),
		operateAgentResponse:    &daemonpb.AgentOperationResponse{Success: true},
		getAgentLogsResponse:    make([]string, 0),
		getAgentMetricsResponse: make([]*daemonpb.ResourceDataPoint, 0),
	}
}

// SetListAgentsResponse 设置ListAgents响应
func (m *MockDaemonClient) SetListAgentsResponse(agents []*daemonpb.AgentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listAgentsResponse = agents
}

// SetOperateAgentResponse 设置OperateAgent响应
func (m *MockDaemonClient) SetOperateAgentResponse(success bool, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operateAgentResponse = &daemonpb.AgentOperationResponse{
		Success:      success,
		ErrorMessage: errorMsg,
	}
}

// SetGetAgentLogsResponse 设置GetAgentLogs响应
func (m *MockDaemonClient) SetGetAgentLogsResponse(logs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getAgentLogsResponse = logs
}

// SetListAgentsError 设置ListAgents错误
func (m *MockDaemonClient) SetListAgentsError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listAgentsError = err
}

// SetOperateAgentError 设置OperateAgent错误
func (m *MockDaemonClient) SetOperateAgentError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operateAgentError = err
}

// SetGetAgentLogsError 设置GetAgentLogs错误
func (m *MockDaemonClient) SetGetAgentLogsError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getAgentLogsError = err
}

// Reset 重置Mock客户端状态
func (m *MockDaemonClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listAgentsResponse = make([]*daemonpb.AgentInfo, 0)
	m.operateAgentResponse = &daemonpb.AgentOperationResponse{Success: true}
	m.getAgentLogsResponse = make([]string, 0)
	m.listAgentsError = nil
	m.operateAgentError = nil
	m.getAgentLogsError = nil
	m.listAgentsCallCount = 0
	m.operateAgentCallCount = 0
	m.getAgentLogsCallCount = 0
}

// GetListAgentsCallCount 获取ListAgents调用次数
func (m *MockDaemonClient) GetListAgentsCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.listAgentsCallCount
}

// GetOperateAgentCallCount 获取OperateAgent调用次数
func (m *MockDaemonClient) GetOperateAgentCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.operateAgentCallCount
}

// GetGetAgentLogsCallCount 获取GetAgentLogs调用次数
func (m *MockDaemonClient) GetGetAgentLogsCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getAgentLogsCallCount
}

// OperateAgent 实现DaemonClient接口
func (m *MockDaemonClient) OperateAgent(ctx context.Context, nodeID, agentID, operation string) error {
	m.mu.Lock()
	m.operateAgentCallCount++
	shouldError := m.operateAgentError != nil
	err := m.operateAgentError
	response := m.operateAgentResponse
	m.mu.Unlock()

	if shouldError {
		return err
	}

	if !response.Success {
		if response.ErrorMessage != "" {
			return fmt.Errorf("agent operation failed: %s", response.ErrorMessage)
		}
		return fmt.Errorf("agent operation failed")
	}

	return nil
}

// ListAgents 实现DaemonClient接口
func (m *MockDaemonClient) ListAgents(ctx context.Context, nodeID string) ([]*daemonpb.AgentInfo, error) {
	m.mu.Lock()
	m.listAgentsCallCount++
	shouldError := m.listAgentsError != nil
	err := m.listAgentsError
	response := m.listAgentsResponse
	m.mu.Unlock()

	if shouldError {
		return nil, err
	}

	return response, nil
}

// GetAgentMetrics 实现DaemonClient接口
func (m *MockDaemonClient) GetAgentMetrics(ctx context.Context, nodeID, agentID string, duration time.Duration) ([]*daemonpb.ResourceDataPoint, error) {
	m.mu.Lock()
	m.getAgentMetricsCallCount++
	shouldError := m.getAgentMetricsError != nil
	err := m.getAgentMetricsError
	response := m.getAgentMetricsResponse
	m.mu.Unlock()

	if shouldError {
		return nil, err
	}

	return response, nil
}

// MockDaemonClientPool Mock Daemon客户端连接池
type MockDaemonClientPool struct {
	mu      sync.RWMutex
	clients map[string]*MockDaemonClient
}

// NewMockDaemonClientPool 创建Mock Daemon客户端连接池
func NewMockDaemonClientPool() *MockDaemonClientPool {
	return &MockDaemonClientPool{
		clients: make(map[string]*MockDaemonClient),
	}
}

// GetClient 获取或创建Mock客户端
func (p *MockDaemonClientPool) GetClient(nodeID, address string) (service.DaemonClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, exists := p.clients[nodeID]; exists {
		return client, nil
	}

	client := NewMockDaemonClient()
	p.clients[nodeID] = client
	return client, nil
}

// GetMockClient 获取Mock客户端（用于测试中设置响应）
func (p *MockDaemonClientPool) GetMockClient(nodeID string) *MockDaemonClient {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.clients[nodeID]
}

// CloseAll 关闭所有客户端
func (p *MockDaemonClientPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients = make(map[string]*MockDaemonClient)
}
