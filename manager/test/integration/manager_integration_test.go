package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Helper: 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ManagerTestSuite Manager集成测试套件
type ManagerTestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
	testData   map[string]interface{} // 存储测试过程中的数据（如token, user_id等）
}

// SetupSuite 在整个测试套件开始前运行一次
func (s *ManagerTestSuite) SetupSuite() {
	s.baseURL = getEnvOrDefault("MANAGER_BASE_URL", "http://127.0.0.1:8080")
	s.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
	s.testData = make(map[string]interface{})

	// 验证服务是否可用
	resp, err := s.httpClient.Get(s.baseURL + "/health")
	require.NoError(s.T(), err, "Manager服务必须启动才能运行测试")
	require.Equal(s.T(), http.StatusOK, resp.StatusCode, "健康检查失败")
	resp.Body.Close()

	s.T().Logf("✅ Manager测试服务地址: %s", s.baseURL)
}

// TearDownSuite 在整个测试套件结束后运行一次
func (s *ManagerTestSuite) TearDownSuite() {
	s.T().Log("✅ Manager测试套件执行完毕")
}

// Helper: HTTP请求方法（支持认证）
func (s *ManagerTestSuite) request(method, path string, body interface{}, needAuth bool) (*http.Response, []byte) {
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
		require.True(s.T(), ok, "需要先登录获取token")
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	resp.Body.Close()

	return resp, respBody
}

func (s *ManagerTestSuite) GET(path string, needAuth bool) (*http.Response, []byte) {
	return s.request("GET", path, nil, needAuth)
}

func (s *ManagerTestSuite) POST(path string, body interface{}, needAuth bool) (*http.Response, []byte) {
	return s.request("POST", path, body, needAuth)
}

func (s *ManagerTestSuite) DELETE(path string, needAuth bool) (*http.Response, []byte) {
	return s.request("DELETE", path, nil, needAuth)
}

// ============================================================================
// Phase 0: 环境检查
// ============================================================================

func (s *ManagerTestSuite) Test_00_Phase0_HealthCheck() {
	s.T().Log("=== Phase 0: 环境检查 - 健康检查 ===")

	resp, body := s.GET("/health", false)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "ok", result["status"])
	assert.NotEmpty(s.T(), result["time"])

	s.T().Logf("✅ 健康检查通过: %+v", result)
}

// ============================================================================
// Phase 1: 认证模块 (User Management)
// ============================================================================

func (s *ManagerTestSuite) Test_01_Phase1_Register() {
	s.T().Log("=== Phase 1: 认证模块 - 用户注册 ===")

	// 生成唯一用户名
	username := fmt.Sprintf("testuser_%d", time.Now().Unix())
	email := fmt.Sprintf("%s@test.com", username)

	resp, body := s.POST("/api/v1/auth/register", map[string]interface{}{
		"username": username,
		"password": "password123",
		"email":    email,
	}, false)

	assert.Equal(s.T(), http.StatusCreated, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])
	assert.Equal(s.T(), "created", result["message"])

	data := result["data"].(map[string]interface{})
	user := data["user"].(map[string]interface{})
	assert.Equal(s.T(), username, user["username"])
	assert.Equal(s.T(), email, user["email"])
	assert.Equal(s.T(), "user", user["role"])
	assert.Equal(s.T(), "active", user["status"])

	// 保存用户信息供后续测试使用
	s.testData["username"] = username
	s.testData["password"] = "password123"
	s.testData["user_id"] = user["id"]

	s.T().Logf("✅ 用户注册成功: ID=%v, username=%s", user["id"], username)
}

func (s *ManagerTestSuite) Test_02_Phase1_Login() {
	s.T().Log("=== Phase 1: 认证模块 - 用户登录 ===")

	username, ok := s.testData["username"].(string)
	require.True(s.T(), ok, "需要先注册用户")

	password, ok := s.testData["password"].(string)
	require.True(s.T(), ok)

	resp, body := s.POST("/api/v1/auth/login", map[string]interface{}{
		"username": username,
		"password": password,
	}, false)

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])
	assert.Equal(s.T(), "success", result["message"])

	data := result["data"].(map[string]interface{})
	token := data["token"].(string)
	assert.NotEmpty(s.T(), token)

	user := data["user"].(map[string]interface{})
	assert.Equal(s.T(), username, user["username"])

	// 保存token供后续测试使用
	s.testData["token"] = token

	s.T().Logf("✅ 用户登录成功: token=%s...", token[:20])
}

func (s *ManagerTestSuite) Test_03_Phase1_GetProfile() {
	s.T().Log("=== Phase 1: 认证模块 - 获取用户资料 ===")

	resp, body := s.GET("/api/v1/auth/profile", true)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])

	data := result["data"].(map[string]interface{})
	user := data["user"].(map[string]interface{})

	username, _ := s.testData["username"].(string)
	assert.Equal(s.T(), username, user["username"])

	s.T().Logf("✅ 获取用户资料成功: %s", user["username"])
}

func (s *ManagerTestSuite) Test_04_Phase1_ChangePassword() {
	s.T().Log("=== Phase 1: 认证模块 - 修改密码 ===")

	oldPassword, _ := s.testData["password"].(string)
	newPassword := "newpassword456"

	resp, body := s.POST("/api/v1/auth/change-password", map[string]interface{}{
		"old_password": oldPassword,
		"new_password": newPassword,
	}, true)

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])

	// 更新测试数据中的密码
	s.testData["password"] = newPassword

	s.T().Logf("✅ 修改密码成功")
}

func (s *ManagerTestSuite) Test_05_Phase1_LoginWithNewPassword() {
	s.T().Log("=== Phase 1: 认证模块 - 使用新密码登录 ===")

	username, _ := s.testData["username"].(string)
	newPassword, _ := s.testData["password"].(string)

	resp, body := s.POST("/api/v1/auth/login", map[string]interface{}{
		"username": username,
		"password": newPassword,
	}, false)

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])

	data := result["data"].(map[string]interface{})
	token := data["token"].(string)
	assert.NotEmpty(s.T(), token)

	// 更新token
	s.testData["token"] = token

	s.T().Logf("✅ 使用新密码登录成功")
}

// ============================================================================
// Phase 2: 节点管理模块
// ============================================================================

func (s *ManagerTestSuite) Test_10_Phase2_ListNodes() {
	s.T().Log("=== Phase 2: 节点管理 - 查询节点列表 ===")

	resp, body := s.GET("/api/v1/nodes?page=1&page_size=10", true)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])

	data := result["data"].(map[string]interface{})
	assert.NotNil(s.T(), data["list"])

	pageInfo := data["page_info"].(map[string]interface{})
	assert.NotNil(s.T(), pageInfo["total"])

	s.T().Logf("✅ 查询节点列表成功: total=%v", pageInfo["total"])
}

func (s *ManagerTestSuite) Test_11_Phase2_GetStatistics() {
	s.T().Log("=== Phase 2: 节点管理 - 获取统计信息 ===")

	resp, body := s.GET("/api/v1/nodes/statistics", true)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), float64(0), result["code"])

	data := result["data"].(map[string]interface{})
	stats := data["statistics"].(map[string]interface{})

	// 统计信息可能为空（无节点时），只检查字段存在即可
	assert.NotNil(s.T(), stats)

	s.T().Logf("✅ 获取统计信息成功: statistics=%v", stats)
}

// ============================================================================
// Phase 3: 错误场景测试
// ============================================================================

func (s *ManagerTestSuite) Test_20_Phase3_LoginWithInvalidCredentials() {
	s.T().Log("=== Phase 3: 错误场景 - 无效凭证登录 ===")

	resp, body := s.POST("/api/v1/auth/login", map[string]interface{}{
		"username": "nonexistent",
		"password": "wrongpassword",
	}, false)

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 无效凭证登录测试通过: code=%v, message=%s", result["code"], result["message"])
}

func (s *ManagerTestSuite) Test_21_Phase3_AccessWithoutToken() {
	s.T().Log("=== Phase 3: 错误场景 - 无Token访问受保护接口 ===")

	req, err := http.NewRequest("GET", s.baseURL+"/api/v1/auth/profile", nil)
	require.NoError(s.T(), err)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)

	s.T().Logf("✅ 无Token访问测试通过: status=%d", resp.StatusCode)
}

func (s *ManagerTestSuite) Test_22_Phase3_RegisterDuplicateUser() {
	s.T().Log("=== Phase 3: 错误场景 - 注册重复用户 ===")

	username, _ := s.testData["username"].(string)

	resp, body := s.POST("/api/v1/auth/register", map[string]interface{}{
		"username": username,
		"password": "somepassword",
		"email":    "another@test.com",
	}, false)

	assert.Equal(s.T(), http.StatusConflict, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Logf("✅ 注册重复用户测试通过: code=%v, message=%s", result["code"], result["message"])
}

// ============================================================================
// TestManagerIntegration 运行测试套件
// ============================================================================

func TestManagerIntegration(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}
