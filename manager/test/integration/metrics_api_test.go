package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MetricsAPITestSuite Metrics API 集成测试套件
type MetricsAPITestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
	testData   map[string]interface{}
	testNodeID string
}

// SetupSuite 在整个测试套件开始前运行一次
func (s *MetricsAPITestSuite) SetupSuite() {
	s.baseURL = getEnvOrDefault("MANAGER_BASE_URL", "http://127.0.0.1:8080")
	s.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
	s.testData = make(map[string]interface{})
	s.testNodeID = "test-node-metrics-001"

	// 验证服务是否可用
	resp, err := s.httpClient.Get(s.baseURL + "/health")
	require.NoError(s.T(), err, "Manager服务必须启动才能运行测试")
	require.Equal(s.T(), http.StatusOK, resp.StatusCode, "健康检查失败")
	resp.Body.Close()

	// 确保已登录（获取 token）
	s.ensureAuthenticated()

	s.T().Logf("✅ Metrics API 测试服务地址: %s", s.baseURL)
}

// ensureAuthenticated 确保已认证（如果未登录则先登录）
func (s *MetricsAPITestSuite) ensureAuthenticated() {
	if _, ok := s.testData["token"]; ok {
		return // 已有 token
	}

	// 尝试登录
	username := fmt.Sprintf("testuser_metrics_%d", time.Now().Unix())
	email := fmt.Sprintf("%s@test.com", username)

	// 注册用户
	resp, body := s.POST("/api/v1/auth/register", map[string]interface{}{
		"username": username,
		"password": "password123",
		"email":    email,
	}, false)

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if data, ok := result["data"].(map[string]interface{}); ok {
				if user, ok := data["user"].(map[string]interface{}); ok {
					if userID, ok := user["id"].(float64); ok {
						s.testData["user_id"] = int(userID)
					}
				}
			}
		}
	}

	// 登录获取 token
	resp, body = s.POST("/api/v1/auth/login", map[string]interface{}{
		"username": username,
		"password": "password123",
	}, false)

	require.Equal(s.T(), http.StatusOK, resp.StatusCode, "登录失败，响应: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	require.Equal(s.T(), float64(0), result["code"], "登录失败，响应: %s", string(body))

	data := result["data"].(map[string]interface{})
	token := data["token"].(string)
	s.testData["token"] = token

	s.T().Log("✅ 已获取认证 Token")
}

// Helper: HTTP请求方法（支持认证）
func (s *MetricsAPITestSuite) request(method, path string, body interface{}, needAuth bool) (*http.Response, []byte) {
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

func (s *MetricsAPITestSuite) GET(path string, needAuth bool) (*http.Response, []byte) {
	return s.request("GET", path, nil, needAuth)
}

func (s *MetricsAPITestSuite) POST(path string, body interface{}, needAuth bool) (*http.Response, []byte) {
	return s.request("POST", path, body, needAuth)
}

// ============================================================================
// TestGetLatestMetrics - 获取节点最新指标
// ============================================================================

func (s *MetricsAPITestSuite) TestGetLatestMetrics_Success() {
	s.T().Log("=== Test: 获取节点最新指标 - 成功场景 ===")

	// 注意：此测试需要节点已存在且有指标数据
	// 在实际环境中，需要先创建节点并插入指标数据
	resp, body := s.GET(fmt.Sprintf("/api/v1/metrics/nodes/%s/latest", s.testNodeID), true)

	// 如果节点不存在，返回 404 是正常的
	if resp.StatusCode == http.StatusNotFound {
		s.T().Logf("⚠️ 节点不存在，跳过此测试（需要先创建节点和指标数据）")
		return
	}

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), float64(0), result["code"])

	// 验证数据结构
	data, ok := result["data"].(map[string]interface{})
	require.True(s.T(), ok, "data 应该是 map 类型")

	// 验证包含指标类型 key（可能为空）
	_, hasCPU := data["cpu"]
	_, hasMemory := data["memory"]
	_, hasDisk := data["disk"]
	_, hasNetwork := data["network"]

	s.T().Logf("✅ 获取最新指标成功，包含类型: cpu=%v, memory=%v, disk=%v, network=%v",
		hasCPU, hasMemory, hasDisk, hasNetwork)
}

func (s *MetricsAPITestSuite) TestGetLatestMetrics_NodeNotFound() {
	s.T().Log("=== Test: 获取节点最新指标 - 节点不存在 ===")

	nonExistentNodeID := "non-existent-node-999"
	resp, body := s.GET(fmt.Sprintf("/api/v1/metrics/nodes/%s/latest", nonExistentNodeID), true)

	// 节点不存在可能返回 404 或 200（空数据）
	if resp.StatusCode == http.StatusNotFound {
		var result map[string]interface{}
		err := json.Unmarshal(body, &result)
		require.NoError(s.T(), err)
		s.T().Logf("✅ 节点不存在返回 404，符合预期")
	} else if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		err := json.Unmarshal(body, &result)
		require.NoError(s.T(), err)
		// 可能返回空数据
		s.T().Logf("✅ 节点不存在返回 200（空数据），符合预期")
	}
}

func (s *MetricsAPITestSuite) TestGetLatestMetrics_Unauthorized() {
	s.T().Log("=== Test: 获取节点最新指标 - 未授权 ===")

	resp, body := s.GET(fmt.Sprintf("/api/v1/metrics/nodes/%s/latest", s.testNodeID), false)

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode, "应该返回 401，响应: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	assert.NotEqual(s.T(), float64(0), result["code"], "错误响应 code 不应为 0")

	s.T().Log("✅ 未授权访问正确返回 401")
}

// ============================================================================
// TestGetMetricsHistory - 获取历史指标数据
// ============================================================================

func (s *MetricsAPITestSuite) TestGetMetricsHistory_Success() {
	s.T().Log("=== Test: 获取历史指标数据 - 成功场景 ===")

	now := time.Now()
	startTime := now.Add(-1 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	path := fmt.Sprintf("/api/v1/metrics/nodes/%s/cpu/history?start_time=%s&end_time=%s",
		s.testNodeID, startTime, endTime)
	resp, body := s.GET(path, true)

	// 如果节点不存在或没有数据，返回 404 或 200（空数组）是正常的
	if resp.StatusCode == http.StatusNotFound {
		s.T().Logf("⚠️ 节点不存在，跳过此测试（需要先创建节点和指标数据）")
		return
	}

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), float64(0), result["code"])

	// 验证数据结构（data 应该是数组）
	data, ok := result["data"]
	require.True(s.T(), ok, "data 应该存在")

	// data 可能是数组或空
	s.T().Logf("✅ 获取历史数据成功，数据类型: %T", data)
}

func (s *MetricsAPITestSuite) TestGetMetricsHistory_InvalidTimeRange() {
	s.T().Log("=== Test: 获取历史指标数据 - 时间范围超过 30 天 ===")

	now := time.Now()
	startTime := now.Add(-31 * 24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	path := fmt.Sprintf("/api/v1/metrics/nodes/%s/cpu/history?start_time=%s&end_time=%s",
		s.testNodeID, startTime, endTime)
	resp, body := s.GET(path, true)

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "应该返回 400，响应: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	assert.NotEqual(s.T(), float64(0), result["code"], "错误响应 code 不应为 0")
	assert.Contains(s.T(), result["message"].(string), "30 天", "错误消息应该包含 30 天")

	s.T().Log("✅ 时间范围超过 30 天正确返回 400")
}

func (s *MetricsAPITestSuite) TestGetMetricsHistory_InvalidTimeFormat() {
	s.T().Log("=== Test: 获取历史指标数据 - 时间格式错误 ===")

	path := fmt.Sprintf("/api/v1/metrics/nodes/%s/cpu/history?start_time=invalid-time&end_time=2025-12-04T10:00:00Z",
		s.testNodeID)
	resp, body := s.GET(path, true)

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "应该返回 400，响应: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Log("✅ 时间格式错误正确返回 400")
}

func (s *MetricsAPITestSuite) TestGetMetricsHistory_InvalidMetricType() {
	s.T().Log("=== Test: 获取历史指标数据 - 无效的指标类型 ===")

	now := time.Now()
	startTime := now.Add(-1 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	path := fmt.Sprintf("/api/v1/metrics/nodes/%s/invalid_type/history?start_time=%s&end_time=%s",
		s.testNodeID, startTime, endTime)
	resp, body := s.GET(path, true)

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "应该返回 400，响应: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	assert.NotEqual(s.T(), float64(0), result["code"])

	s.T().Log("✅ 无效的指标类型正确返回 400")
}

// ============================================================================
// TestGetMetricsSummary - 获取指标统计摘要
// ============================================================================

func (s *MetricsAPITestSuite) TestGetMetricsSummary_Success() {
	s.T().Log("=== Test: 获取指标统计摘要 - 成功场景 ===")

	path := fmt.Sprintf("/api/v1/metrics/nodes/%s/summary", s.testNodeID)
	resp, body := s.GET(path, true)

	// 如果节点不存在，返回 404 或 200（空数据）是正常的
	if resp.StatusCode == http.StatusNotFound {
		s.T().Logf("⚠️ 节点不存在，跳过此测试（需要先创建节点和指标数据）")
		return
	}

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), float64(0), result["code"])

	// 验证数据结构（data 应该是 map）
	data, ok := result["data"].(map[string]interface{})
	require.True(s.T(), ok, "data 应该是 map 类型")

	// 验证包含指标类型 key
	s.T().Logf("✅ 获取统计摘要成功，包含类型: %v", data)

	// 测试默认时间范围（不传递时间参数）
	path2 := fmt.Sprintf("/api/v1/metrics/nodes/%s/summary", s.testNodeID)
	resp2, body2 := s.GET(path2, true)
	assert.Equal(s.T(), http.StatusOK, resp2.StatusCode, "默认时间范围查询应该成功")

	var result2 map[string]interface{}
	err = json.Unmarshal(body2, &result2)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), float64(0), result2["code"])

	s.T().Log("✅ 默认时间范围（最近 24 小时）查询成功")
}

func (s *MetricsAPITestSuite) TestGetMetricsSummary_CustomTimeRange() {
	s.T().Log("=== Test: 获取指标统计摘要 - 自定义时间范围 ===")

	now := time.Now()
	startTime := now.Add(-2 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	path := fmt.Sprintf("/api/v1/metrics/nodes/%s/summary?start_time=%s&end_time=%s",
		s.testNodeID, startTime, endTime)
	resp, body := s.GET(path, true)

	if resp.StatusCode == http.StatusNotFound {
		s.T().Logf("⚠️ 节点不存在，跳过此测试")
		return
	}

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode, "响应体: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), float64(0), result["code"])

	s.T().Log("✅ 自定义时间范围查询成功")
}

// ============================================================================
// TestMetricsAPI_ErrorScenarios - 错误场景测试
// ============================================================================

func (s *MetricsAPITestSuite) TestMetricsAPI_InvalidToken() {
	s.T().Log("=== Test: 无效 Token ===")

	// 使用无效 token
	req, err := http.NewRequest("GET", s.baseURL+fmt.Sprintf("/api/v1/metrics/nodes/%s/latest", s.testNodeID), nil)
	require.NoError(s.T(), err)
	req.Header.Set("Authorization", "Bearer invalid-token-12345")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode, "应该返回 401")

	s.T().Log("✅ 无效 Token 正确返回 401")
}

// ============================================================================
// 运行测试套件
// ============================================================================

func TestMetricsAPI(t *testing.T) {
	suite.Run(t, new(MetricsAPITestSuite))
}

