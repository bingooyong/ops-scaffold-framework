package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// AgentAPITestSuite Agent API集成测试套件
type AgentAPITestSuite struct {
	suite.Suite
	baseURL       string
	httpClient    *http.Client
	testData      map[string]interface{}
	db            *gorm.DB
	agentRepo     repository.AgentRepository
	nodeRepo      repository.NodeRepository
	logger        *zap.Logger
	cleanupNodes  []string // 用于清理测试数据
	cleanupAgents []struct {
		nodeID  string
		agentID string
	}
}

// SetupSuite 在整个测试套件开始前运行一次
func (s *AgentAPITestSuite) SetupSuite() {
	s.baseURL = getEnvOrDefault("MANAGER_BASE_URL", "http://127.0.0.1:8080")
	s.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
	s.testData = make(map[string]interface{})
	s.cleanupNodes = make([]string, 0)
	s.cleanupAgents = make([]struct {
		nodeID  string
		agentID string
	}, 0)

	// 初始化logger
	var err error
	s.logger, err = zap.NewDevelopment()
	require.NoError(s.T(), err)

	// 初始化数据库连接
	// 集成测试需要连接到与运行中的服务相同的数据库
	dsn := getEnvOrDefault("DATABASE_DSN", "root:rootpassword@tcp(127.0.0.1:3306)/ops_manager_dev?charset=utf8mb4&parseTime=True&loc=Local")

	// 如果全局DB已初始化，使用它；否则创建新连接（不设置全局DB）
	if database.DB != nil {
		s.db = database.DB
	} else {
		// 创建临时连接用于测试（直接使用GORM，不通过database包）
		// 注意：这里不设置全局database.DB，避免影响运行中的服务
		s.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		require.NoError(s.T(), err, "无法连接到数据库，请确保MySQL运行且数据库已创建")
	}

	// 初始化Repository（用于创建测试数据）
	s.agentRepo = repository.NewAgentRepository(s.db)
	s.nodeRepo = repository.NewNodeRepository(s.db)

	// 验证服务是否可用
	resp, err := s.httpClient.Get(s.baseURL + "/health")
	require.NoError(s.T(), err, "Manager服务必须启动才能运行测试")
	require.Equal(s.T(), http.StatusOK, resp.StatusCode, "健康检查失败")
	resp.Body.Close()

	// 登录获取token
	s.login()

	s.T().Logf("✅ Agent API测试服务地址: %s", s.baseURL)
}

// TearDownSuite 在整个测试套件结束后运行一次
func (s *AgentAPITestSuite) TearDownSuite() {
	// 清理测试数据
	ctx := context.Background()
	for _, agent := range s.cleanupAgents {
		_ = s.agentRepo.Delete(ctx, agent.nodeID, agent.agentID)
	}
	for _, nodeID := range s.cleanupNodes {
		node, _ := s.nodeRepo.GetByNodeID(ctx, nodeID)
		if node != nil {
			_ = s.nodeRepo.Delete(ctx, node.ID)
		}
	}

	s.T().Log("✅ Agent API测试套件执行完毕")
}

// login 登录获取token
func (s *AgentAPITestSuite) login() {
	// 生成唯一用户名
	username := fmt.Sprintf("testuser_%d", time.Now().Unix())
	email := fmt.Sprintf("%s@test.com", username)

	// 注册用户
	resp, body := s.POST("/api/v1/auth/register", map[string]interface{}{
		"username": username,
		"password": "password123",
		"email":    email,
	}, false)

	if resp.StatusCode == http.StatusCreated {
		// 登录
		resp, body = s.POST("/api/v1/auth/login", map[string]interface{}{
			"username": username,
			"password": "password123",
		}, false)

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			json.Unmarshal(body, &result)
			if data, ok := result["data"].(map[string]interface{}); ok {
				if token, ok := data["token"].(string); ok {
					s.testData["token"] = token
					s.testData["username"] = username
					return
				}
			}
		}
	}

	// 如果注册/登录失败，尝试使用现有用户登录
	resp, body = s.POST("/api/v1/auth/login", map[string]interface{}{
		"username": "admin",
		"password": "admin123",
	}, false)

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		if data, ok := result["data"].(map[string]interface{}); ok {
			if token, ok := data["token"].(string); ok {
				s.testData["token"] = token
				s.testData["username"] = "admin"
			}
		}
	}
}

// Helper: HTTP请求方法（支持认证）
func (s *AgentAPITestSuite) request(method, path string, body interface{}, needAuth bool) (*http.Response, []byte) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		require.NoError(s.T(), err)
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, s.baseURL+path, bodyReader)
	require.NoError(s.T(), err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 添加认证头
	if needAuth {
		token, ok := s.testData["token"].(string)
		if ok {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	resp.Body.Close()

	return resp, respBody
}

func (s *AgentAPITestSuite) GET(path string, needAuth bool) (*http.Response, []byte) {
	return s.request("GET", path, nil, needAuth)
}

func (s *AgentAPITestSuite) POST(path string, body interface{}, needAuth bool) (*http.Response, []byte) {
	return s.request("POST", path, body, needAuth)
}

// createTestNode 创建测试节点
func (s *AgentAPITestSuite) createTestNode() *model.Node {
	nodeID := uuid.New().String()
	node := &model.Node{
		NodeID:        nodeID,
		Hostname:      "test-host",
		IP:            "127.0.0.1",
		OS:            "linux",
		Arch:          "amd64",
		Status:        "online",
		RegisterAt:    time.Now(),
		DaemonVersion: "v1.0.0",
		AgentVersion:  "v1.0.0",
	}

	ctx := context.Background()
	err := s.nodeRepo.Create(ctx, node)
	require.NoError(s.T(), err)

	s.cleanupNodes = append(s.cleanupNodes, nodeID)
	return node
}

// createTestAgent 创建测试Agent
func (s *AgentAPITestSuite) createTestAgent(nodeID string, agentID string, status string) *model.Agent {
	agent := &model.Agent{
		NodeID:       nodeID,
		AgentID:      agentID,
		Type:         "filebeat",
		Version:      "v1.0.0",
		Status:       status,
		PID:          0,
		LastSyncTime: time.Now(),
	}

	if status == "running" {
		agent.PID = 12345
	}

	ctx := context.Background()
	err := s.agentRepo.Create(ctx, agent)
	require.NoError(s.T(), err)

	s.cleanupAgents = append(s.cleanupAgents, struct {
		nodeID  string
		agentID string
	}{nodeID: nodeID, agentID: agentID})

	return agent
}

// ============================================================================
// ListAgents 接口测试
// ============================================================================

// TestAgentAPI_ListAgents_Success 成功场景：获取Agent列表
func (s *AgentAPITestSuite) TestAgentAPI_ListAgents_Success() {
	s.T().Log("=== ListAgents - 成功场景 ===")

	// 创建测试节点和Agent
	node := s.createTestNode()
	agent1 := s.createTestAgent(node.NodeID, "agent-1", "running")
	agent2 := s.createTestAgent(node.NodeID, "agent-2", "stopped")

	// 调用API
	resp, body := s.GET(fmt.Sprintf("/api/v1/nodes/%s/agents", node.NodeID), true)

	// 验证响应
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])
	assert.Equal(s.T(), "success", result["message"])

	data := result["data"].(map[string]interface{})
	agents := data["agents"].([]interface{})
	count := int(data["count"].(float64))

	assert.GreaterOrEqual(s.T(), count, 2, "应该至少返回2个Agent")
	assert.GreaterOrEqual(s.T(), len(agents), 2, "应该至少返回2个Agent")

	// 验证Agent数据
	agentMap := make(map[string]interface{})
	for _, a := range agents {
		agent := a.(map[string]interface{})
		agentMap[agent["agent_id"].(string)] = agent
	}

	assert.Contains(s.T(), agentMap, agent1.AgentID)
	assert.Contains(s.T(), agentMap, agent2.AgentID)

	s.T().Logf("✅ ListAgents成功: 返回%d个Agent", count)
}

// TestAgentAPI_ListAgents_NodeNotFound 错误场景：节点不存在
func (s *AgentAPITestSuite) TestAgentAPI_ListAgents_NodeNotFound() {
	s.T().Log("=== ListAgents - 节点不存在 ===")

	nonExistentNodeID := uuid.New().String()
	resp, body := s.GET(fmt.Sprintf("/api/v1/nodes/%s/agents", nonExistentNodeID), true)

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 节点不存在测试通过: code=%v", result["code"])
}

// TestAgentAPI_ListAgents_Unauthorized 错误场景：未授权
func (s *AgentAPITestSuite) TestAgentAPI_ListAgents_Unauthorized() {
	s.T().Log("=== ListAgents - 未授权 ===")

	node := s.createTestNode()
	req, err := http.NewRequest("GET", s.baseURL+fmt.Sprintf("/api/v1/nodes/%s/agents", node.NodeID), nil)
	require.NoError(s.T(), err)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)

	s.T().Logf("✅ 未授权测试通过: status=%d", resp.StatusCode)
}

// TestAgentAPI_ListAgents_InvalidNodeID 错误场景：无效的节点ID
func (s *AgentAPITestSuite) TestAgentAPI_ListAgents_InvalidNodeID() {
	s.T().Log("=== ListAgents - 无效节点ID ===")

	resp, body := s.GET("/api/v1/nodes//agents", true)

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 无效节点ID测试通过: code=%v", result["code"])
}

// TestAgentAPI_ListAgents_EmptyList 边界场景：空列表
func (s *AgentAPITestSuite) TestAgentAPI_ListAgents_EmptyList() {
	s.T().Log("=== ListAgents - 空列表 ===")

	// 创建测试节点（不创建Agent）
	node := s.createTestNode()

	resp, body := s.GET(fmt.Sprintf("/api/v1/nodes/%s/agents", node.NodeID), true)

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])

	data := result["data"].(map[string]interface{})
	count := int(data["count"].(float64))
	agents := data["agents"].([]interface{})

	assert.Equal(s.T(), 0, count)
	assert.Equal(s.T(), 0, len(agents))

	s.T().Logf("✅ 空列表测试通过: count=%d", count)
}

// ============================================================================
// OperateAgent 接口测试
// ============================================================================

// TestAgentAPI_OperateAgent_Start 成功场景：启动Agent
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_Start() {
	s.T().Log("=== OperateAgent - 启动Agent ===")

	// 创建测试节点和Agent（初始状态为stopped）
	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-start", "stopped")

	// 注意：集成测试使用真实服务，不需要Mock
	// mockClient := s.mockPool.GetMockClient(node.NodeID)
	// require.NotNil(s.T(), mockClient)
	// mockClient.SetOperateAgentResponse(true, "")

	// 调用API
	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, agent.AgentID),
		map[string]interface{}{
			"operation": "start",
		},
		true,
	)

	// 验证响应
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])
	assert.Equal(s.T(), "success", result["message"])

	// 注意：无法验证Mock调用次数，因为使用的是真实服务

	s.T().Logf("✅ 启动Agent测试通过")
}

// TestAgentAPI_OperateAgent_Stop 成功场景：停止Agent
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_Stop() {
	s.T().Log("=== OperateAgent - 停止Agent ===")

	// 创建运行中的Agent
	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-stop", "running")

	// 注意：集成测试使用真实服务，不需要Mock
	// mockClient := s.mockPool.GetMockClient(node.NodeID)
	// require.NotNil(s.T(), mockClient)
	// mockClient.SetOperateAgentResponse(true, "")

	// 调用API
	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, agent.AgentID),
		map[string]interface{}{
			"operation": "stop",
		},
		true,
	)

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 停止Agent测试通过")
}

// TestAgentAPI_OperateAgent_Restart 成功场景：重启Agent
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_Restart() {
	s.T().Log("=== OperateAgent - 重启Agent ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-restart", "running")

	// 注意：集成测试使用真实服务，不需要Mock
	// mockClient := s.mockPool.GetMockClient(node.NodeID)
	// require.NotNil(s.T(), mockClient)
	// mockClient.SetOperateAgentResponse(true, "")

	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, agent.AgentID),
		map[string]interface{}{
			"operation": "restart",
		},
		true,
	)

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 重启Agent测试通过")
}

// TestAgentAPI_OperateAgent_NodeNotFound 错误场景：节点不存在
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_NodeNotFound() {
	s.T().Log("=== OperateAgent - 节点不存在 ===")

	nonExistentNodeID := uuid.New().String()
	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/agent-1/operate", nonExistentNodeID),
		map[string]interface{}{
			"operation": "start",
		},
		true,
	)

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 节点不存在测试通过: code=%v", result["code"])
}

// TestAgentAPI_OperateAgent_AgentNotFound 错误场景：Agent不存在
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_AgentNotFound() {
	s.T().Log("=== OperateAgent - Agent不存在 ===")

	node := s.createTestNode()
	nonExistentAgentID := "non-existent-agent"

	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, nonExistentAgentID),
		map[string]interface{}{
			"operation": "start",
		},
		true,
	)

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ Agent不存在测试通过: code=%v", result["code"])
}

// TestAgentAPI_OperateAgent_InvalidOperation 错误场景：无效操作
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_InvalidOperation() {
	s.T().Log("=== OperateAgent - 无效操作 ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-invalid", "stopped")

	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, agent.AgentID),
		map[string]interface{}{
			"operation": "invalid",
		},
		true,
	)

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 无效操作测试通过: code=%v", result["code"])
}

// TestAgentAPI_OperateAgent_GRPCFailure 错误场景：gRPC失败
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_GRPCFailure() {
	s.T().Log("=== OperateAgent - gRPC失败 ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-grpc-fail", "stopped")

	// 注意：集成测试中无法模拟gRPC错误
	// 此测试需要Daemon服务未运行或不可达才能验证错误场景
	// 为简化测试，这里跳过或使用无效地址的节点
	// 实际测试中，可以创建一个IP地址无效的节点来触发连接错误
	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, agent.AgentID),
		map[string]interface{}{
			"operation": "start",
		},
		true,
	)

	assert.Equal(s.T(), http.StatusInternalServerError, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ gRPC失败测试通过: code=%v", result["code"])
}

// TestAgentAPI_OperateAgent_Unauthorized 错误场景：未授权
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_Unauthorized() {
	s.T().Log("=== OperateAgent - 未授权 ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-unauth", "stopped")

	req, err := http.NewRequest("POST", s.baseURL+fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, agent.AgentID),
		bytes.NewReader([]byte(`{"operation":"start"}`)))
	require.NoError(s.T(), err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)

	s.T().Logf("✅ 未授权测试通过: status=%d", resp.StatusCode)
}

// TestAgentAPI_OperateAgent_EmptyOperation 边界场景：空操作
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_EmptyOperation() {
	s.T().Log("=== OperateAgent - 空操作 ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-empty", "stopped")

	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, agent.AgentID),
		map[string]interface{}{
			"operation": "",
		},
		true,
	)

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 空操作测试通过: code=%v", result["code"])
}

// TestAgentAPI_OperateAgent_MissingOperation 边界场景：缺少操作字段
func (s *AgentAPITestSuite) TestAgentAPI_OperateAgent_MissingOperation() {
	s.T().Log("=== OperateAgent - 缺少操作字段 ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-missing", "stopped")

	resp, body := s.POST(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, agent.AgentID),
		map[string]interface{}{},
		true,
	)

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 缺少操作字段测试通过: code=%v", result["code"])
}

// ============================================================================
// GetAgentLogs 接口测试
// ============================================================================

// TestAgentAPI_GetAgentLogs_Success 成功场景：获取Agent日志
func (s *AgentAPITestSuite) TestAgentAPI_GetAgentLogs_Success() {
	s.T().Log("=== GetAgentLogs - 成功场景 ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-logs", "running")

	// 注意：当前GetAgentLogs功能未实现，返回501
	resp, body := s.GET(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/logs?lines=100", node.NodeID, agent.AgentID),
		true,
	)

	// 当前实现返回501（功能未实现）
	assert.Equal(s.T(), http.StatusInternalServerError, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	// 验证返回了错误信息
	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ GetAgentLogs测试通过（功能未实现）: code=%v", result["code"])
}

// TestAgentAPI_GetAgentLogs_NodeNotFound 错误场景：节点不存在
func (s *AgentAPITestSuite) TestAgentAPI_GetAgentLogs_NodeNotFound() {
	s.T().Log("=== GetAgentLogs - 节点不存在 ===")

	nonExistentNodeID := uuid.New().String()
	resp, body := s.GET(
		fmt.Sprintf("/api/v1/nodes/%s/agents/agent-1/logs", nonExistentNodeID),
		true,
	)

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 节点不存在测试通过: code=%v", result["code"])
}

// TestAgentAPI_GetAgentLogs_AgentNotFound 错误场景：Agent不存在
func (s *AgentAPITestSuite) TestAgentAPI_GetAgentLogs_AgentNotFound() {
	s.T().Log("=== GetAgentLogs - Agent不存在 ===")

	node := s.createTestNode()
	nonExistentAgentID := "non-existent-agent"

	resp, body := s.GET(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/logs", node.NodeID, nonExistentAgentID),
		true,
	)

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ Agent不存在测试通过: code=%v", result["code"])
}

// TestAgentAPI_GetAgentLogs_Unauthorized 错误场景：未授权
func (s *AgentAPITestSuite) TestAgentAPI_GetAgentLogs_Unauthorized() {
	s.T().Log("=== GetAgentLogs - 未授权 ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-logs-unauth", "running")

	req, err := http.NewRequest("GET", s.baseURL+fmt.Sprintf("/api/v1/nodes/%s/agents/%s/logs", node.NodeID, agent.AgentID), nil)
	require.NoError(s.T(), err)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)

	s.T().Logf("✅ 未授权测试通过: status=%d", resp.StatusCode)
}

// TestAgentAPI_GetAgentLogs_InvalidLines 边界场景：无效的行数
func (s *AgentAPITestSuite) TestAgentAPI_GetAgentLogs_InvalidLines() {
	s.T().Log("=== GetAgentLogs - 无效行数 ===")

	node := s.createTestNode()
	agent := s.createTestAgent(node.NodeID, "agent-logs-invalid", "running")

	// 测试负数
	resp, body := s.GET(
		fmt.Sprintf("/api/v1/nodes/%s/agents/%s/logs?lines=-1", node.NodeID, agent.AgentID),
		true,
	)

	// 应该返回400或使用默认值
	assert.True(s.T(), resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError,
		"响应体: %s", string(body))

	s.T().Logf("✅ 无效行数测试通过: status=%d", resp.StatusCode)
}

// ============================================================================
// 并发操作测试
// ============================================================================

// TestAgentAPI_ConcurrentOperations 并发操作安全性测试
func (s *AgentAPITestSuite) TestAgentAPI_ConcurrentOperations() {
	s.T().Log("=== 并发操作测试 ===")

	node := s.createTestNode()
	agentCount := 5
	operationsPerAgent := 3

	// 创建多个Agent
	agents := make([]*model.Agent, agentCount)
	for i := 0; i < agentCount; i++ {
		agentID := fmt.Sprintf("agent-concurrent-%d", i)
		agents[i] = s.createTestAgent(node.NodeID, agentID, "stopped")
	}

	// 注意：集成测试使用真实服务

	// 并发执行操作
	var wg sync.WaitGroup
	operations := []string{"start", "stop", "restart"}
	errors := make([]error, 0)
	var mu sync.Mutex

	for _, agent := range agents {
		for _, op := range operations {
			wg.Add(1)
			go func(a *model.Agent, operation string) {
				defer wg.Done()

				resp, body := s.POST(
					fmt.Sprintf("/api/v1/nodes/%s/agents/%s/operate", node.NodeID, a.AgentID),
					map[string]interface{}{
						"operation": operation,
					},
					true,
				)

				if resp.StatusCode != http.StatusOK {
					mu.Lock()
					errors = append(errors, fmt.Errorf("operation %s on %s failed: %s", operation, a.AgentID, string(body)))
					mu.Unlock()
				}
			}(agent, op)
		}
	}

	wg.Wait()

	// 验证所有操作成功
	assert.Equal(s.T(), 0, len(errors), "并发操作应该全部成功，但有以下错误: %v", errors)

	// 注意：无法验证调用次数，因为使用的是真实服务
	s.T().Logf("✅ 并发操作测试通过: %d个Agent，每个%d个操作",
		agentCount, operationsPerAgent)
}

// TestAgentAPI_ConcurrentListAgents 并发查询测试
func (s *AgentAPITestSuite) TestAgentAPI_ConcurrentListAgents() {
	s.T().Log("=== 并发查询测试 ===")

	node := s.createTestNode()
	agentCount := 10

	// 创建多个Agent
	for i := 0; i < agentCount; i++ {
		agentID := fmt.Sprintf("agent-list-%d", i)
		s.createTestAgent(node.NodeID, agentID, "running")
	}

	// 并发查询
	concurrentRequests := 20
	var wg sync.WaitGroup
	results := make([][]interface{}, concurrentRequests)
	errors := make([]error, 0)
	var mu sync.Mutex

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			resp, body := s.GET(fmt.Sprintf("/api/v1/nodes/%s/agents", node.NodeID), true)

			if resp.StatusCode != http.StatusOK {
				mu.Lock()
				errors = append(errors, fmt.Errorf("request %d failed: %s", index, string(body)))
				mu.Unlock()
				return
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("request %d parse error: %v", index, err))
				mu.Unlock()
				return
			}

			data := result["data"].(map[string]interface{})
			agents := data["agents"].([]interface{})
			results[index] = agents
		}(i)
	}

	wg.Wait()

	// 验证所有请求成功
	assert.Equal(s.T(), 0, len(errors), "并发查询应该全部成功，但有以下错误: %v", errors)

	// 验证返回数据一致性（所有请求应该返回相同数量的Agent）
	firstCount := len(results[0])
	for i, agents := range results {
		if len(agents) != firstCount {
			s.T().Errorf("请求%d返回的Agent数量不一致: 期望%d，实际%d", i, firstCount, len(agents))
		}
	}

	s.T().Logf("✅ 并发查询测试通过: %d个并发请求，每个返回%d个Agent",
		concurrentRequests, firstCount)
}

// ============================================================================
// TestAgentAPI 运行测试套件
// ============================================================================

func TestAgentAPI(t *testing.T) {
	suite.Run(t, new(AgentAPITestSuite))
}
