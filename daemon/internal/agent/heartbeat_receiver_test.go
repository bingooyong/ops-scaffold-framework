package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// TestHandleHeartbeat_Success 测试成功接收和处理心跳
func TestHandleHeartbeat_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 创建临时工作目录
	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建真实的MultiAgentManager
	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()

	// 注册一个测试Agent
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 注册Agent到MultiAgentManager
	agentInfo := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/test",
		WorkDir:    "/tmp",
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	// 创建心跳接收器
	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 创建测试请求
	reqBody := HeartbeatRequest{
		AgentID:   "test-agent",
		PID:       12345,
		Status:    "running",
		CPU:       50.5,
		Memory:    1024000,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 处理请求
	receiver.HandleHeartbeat(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HeartbeatResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success=true, got false")
	}

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)
}

// TestHandleHeartbeat_InvalidJSON 测试无效JSON格式
func TestHandleHeartbeat_InvalidJSON(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	receiver.HandleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleHeartbeat_InvalidAgentID 测试无效agent_id
func TestHandleHeartbeat_InvalidAgentID(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 测试空agent_id
	reqBody := HeartbeatRequest{
		AgentID: "",
		PID:     12345,
		CPU:     50.0,
		Memory:  1024000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	receiver.HandleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	// 测试不存在的agent_id
	reqBody.AgentID = "non-existent-agent"
	body, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	receiver.HandleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleHeartbeat_InvalidPID 测试无效PID
func TestHandleHeartbeat_InvalidPID(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 测试PID <= 0
	reqBody := HeartbeatRequest{
		AgentID: "test-agent",
		PID:     0,
		CPU:     50.0,
		Memory:  1024000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	receiver.HandleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleHeartbeat_InvalidResourceData 测试无效资源数据
func TestHandleHeartbeat_InvalidResourceData(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 测试CPU < 0
	reqBody := HeartbeatRequest{
		AgentID: "test-agent",
		PID:     12345,
		CPU:     -1.0,
		Memory:  1024000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	receiver.HandleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for CPU < 0, got %d", w.Code)
	}

	// 测试CPU > 100
	reqBody.CPU = 101.0
	body, _ = json.Marshal(reqBody)

	req = httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	receiver.HandleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for CPU > 100, got %d", w.Code)
	}
}

// TestProcessHeartbeat_UpdateMetadata 测试心跳更新元数据
func TestProcessHeartbeat_UpdateMetadata(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewAgentRegistry()

	_, err := registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 创建临时工作目录
	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建真实的MultiAgentManager
	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	// 注册Agent到MultiAgentManager
	agentInfo := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/test",
		WorkDir:    "/tmp",
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 创建心跳
	hb := &Heartbeat{
		AgentID:   "test-agent",
		PID:       12345,
		Status:    "running",
		CPU:       50.5,
		Memory:    1024000,
		Timestamp: time.Now(),
	}

	// 处理心跳
	receiver.processHeartbeat(hb)

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)

	// 验证元数据已更新
	metadata, err := multiManager.GetAgentMetadata("test-agent")
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}

	if metadata.LastHeartbeat.IsZero() {
		t.Error("LastHeartbeat should not be zero")
	}

	if len(metadata.ResourceUsage.CPU) == 0 {
		t.Error("ResourceUsage.CPU should not be empty")
	}

	if len(metadata.ResourceUsage.Memory) == 0 {
		t.Error("ResourceUsage.Memory should not be empty")
	}
}

// TestWorkerPool_ConcurrentProcessing 测试并发处理
func TestWorkerPool_ConcurrentProcessing(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := NewAgentRegistry()

	// 注册多个Agent
	for i := 0; i < 10; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		_, err := registry.Register(agentID, TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
		if err != nil {
			t.Fatalf("failed to register agent: %v", err)
		}
	}

	// 创建临时工作目录
	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	// 注册所有Agent到MultiAgentManager
	for i := 0; i < 10; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		agentInfo := &AgentInfo{
			ID:         agentID,
			Type:       TypeCustom,
			Name:       "Test Agent",
			BinaryPath: "/bin/test",
			WorkDir:    "/tmp",
		}
		_, err = multiManager.RegisterAgent(agentInfo)
		if err != nil {
			t.Fatalf("failed to register agent: %v", err)
		}
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 并发发送100个心跳
	var wg sync.WaitGroup
	numHeartbeats := 100
	wg.Add(numHeartbeats)

	for i := 0; i < numHeartbeats; i++ {
		go func(idx int) {
			defer wg.Done()

			agentID := fmt.Sprintf("agent-%d", idx%10)
			reqBody := HeartbeatRequest{
				AgentID: agentID,
				PID:     12345 + idx,
				Status:  "running",
				CPU:     float64(idx % 100),
				Memory:  uint64(1024000 + idx),
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			receiver.HandleHeartbeat(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
		}(i)
	}

	wg.Wait()

	// 等待所有心跳处理完成
	time.Sleep(500 * time.Millisecond)

	// 验证统计信息
	stats := receiver.GetStats()
	if stats.TotalReceived != int64(numHeartbeats) {
		t.Errorf("expected TotalReceived=%d, got %d", numHeartbeats, stats.TotalReceived)
	}

	if stats.TotalProcessed < int64(numHeartbeats) {
		t.Errorf("expected TotalProcessed>=%d, got %d", numHeartbeats, stats.TotalProcessed)
	}
}

// TestWorkerPool_ChannelFull 测试channel满时的处理
func TestWorkerPool_ChannelFull(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 创建小容量的接收器(worker数量少,channel容量小)
	receiver := &HTTPHeartbeatReceiver{
		multiManager: multiManager,
		registry:     registry,
		logger:       logger,
		workerPool:   make(chan *Heartbeat, 5), // 小容量
		workerCount:  2,                        // 少量worker
		stopCh:       make(chan struct{}),
	}
	receiver.startWorkers()
	defer receiver.Stop()

	// 快速发送大量请求,填满channel
	successCount := 0
	for i := 0; i < 20; i++ {
		reqBody := HeartbeatRequest{
			AgentID: "test-agent",
			PID:     12345,
			CPU:     50.0,
			Memory:  1024000,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		receiver.HandleHeartbeat(w, req)

		if w.Code == http.StatusOK {
			successCount++
		} else if w.Code == http.StatusServiceUnavailable {
			// 预期会有一些503错误(当channel满时)
		}
	}

	// 至少应该有一些请求成功
	if successCount == 0 {
		t.Error("expected at least some requests to succeed")
	}
}

// TestStats_Update 测试统计信息更新
func TestStats_Update(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 注册Agent到MultiAgentManager
	agentInfo := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/test",
		WorkDir:    "/tmp",
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 发送几个心跳
	for i := 0; i < 5; i++ {
		reqBody := HeartbeatRequest{
			AgentID: "test-agent",
			PID:     12345,
			CPU:     50.0,
			Memory:  1024000,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		receiver.HandleHeartbeat(w, req)
	}

	// 等待处理完成
	time.Sleep(200 * time.Millisecond)

	// 验证统计信息
	stats := receiver.GetStats()
	if stats.TotalReceived != 5 {
		t.Errorf("expected TotalReceived=5, got %d", stats.TotalReceived)
	}

	if stats.TotalProcessed != 5 {
		t.Errorf("expected TotalProcessed=5, got %d", stats.TotalProcessed)
	}

	if stats.TotalErrors != 0 {
		t.Errorf("expected TotalErrors=0, got %d", stats.TotalErrors)
	}

	if stats.LastReceivedTime.IsZero() {
		t.Error("LastReceivedTime should not be zero")
	}
}

// TestStop_GracefulShutdown 测试优雅停止
func TestStop_GracefulShutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 注册Agent到MultiAgentManager
	agentInfo := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/test",
		WorkDir:    "/tmp",
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)

	// 发送一些心跳
	for i := 0; i < 5; i++ {
		reqBody := HeartbeatRequest{
			AgentID: "test-agent",
			PID:     12345,
			CPU:     50.0,
			Memory:  1024000,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		receiver.HandleHeartbeat(w, req)
	}

	// 停止接收器
	stopStart := time.Now()
	receiver.Stop()
	stopDuration := time.Since(stopStart)

	// 验证停止是优雅的(应该等待正在处理的心跳完成)
	// 由于有5个心跳,但worker并发处理,所以总时间应该合理
	if stopDuration > 1*time.Second {
		t.Errorf("stop took too long: %v", stopDuration)
	}
}

// TestHandleStats 测试统计信息HTTP endpoint
func TestHandleStats(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 注册Agent到MultiAgentManager
	agentInfo := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/test",
		WorkDir:    "/tmp",
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 发送一个心跳
	reqBody := HeartbeatRequest{
		AgentID: "test-agent",
		PID:     12345,
		CPU:     50.0,
		Memory:  1024000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	receiver.HandleHeartbeat(w, req)

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)

	// 请求统计信息
	req = httptest.NewRequest(http.MethodGet, "/heartbeat/stats", nil)
	w = httptest.NewRecorder()
	receiver.HandleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var stats HeartbeatStats
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}

	if stats.TotalReceived == 0 {
		t.Error("expected TotalReceived > 0")
	}
}

// TestHandleHeartbeat_BurstTraffic 测试突发流量（短时间内大量心跳）
func TestHandleHeartbeat_BurstTraffic(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	agentInfo := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/test",
		WorkDir:    "/tmp",
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 发送大量心跳（突发流量）
	const numHeartbeats = 100
	successCount := 0
	errorCount := 0

	for i := 0; i < numHeartbeats; i++ {
		reqBody := HeartbeatRequest{
			AgentID: "test-agent",
			PID:     12345 + i,
			Status:  "running",
			CPU:     float64(i % 100),
			Memory:  uint64(1024000 + i*1000),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		receiver.HandleHeartbeat(w, req)

		if w.Code == http.StatusOK {
			successCount++
		} else {
			errorCount++
		}
	}

	// 等待所有心跳处理完成
	time.Sleep(500 * time.Millisecond)

	// 验证统计信息
	stats := receiver.GetStats()
	if stats.TotalReceived != int64(numHeartbeats) {
		t.Errorf("expected TotalReceived %d, got %d", numHeartbeats, stats.TotalReceived)
	}

	t.Logf("burst traffic test: %d successful, %d errors", successCount, errorCount)
}

// TestWorkerPool_WorkerFailure 测试worker失败时的处理
func TestWorkerPool_WorkerFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	agentInfo := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/test",
		WorkDir:    "/tmp",
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 发送一些有效心跳
	for i := 0; i < 5; i++ {
		reqBody := HeartbeatRequest{
			AgentID: "test-agent",
			PID:     12345,
			Status:  "running",
			CPU:     50.0,
			Memory:  1024000,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		receiver.HandleHeartbeat(w, req)
	}

	// 等待处理完成
	time.Sleep(200 * time.Millisecond)

	// 验证统计信息（应该有一些处理成功）
	stats := receiver.GetStats()
	if stats.TotalProcessed == 0 {
		t.Error("expected at least one heartbeat to be processed")
	}

	// 验证worker pool仍然正常工作
	reqBody := HeartbeatRequest{
		AgentID: "test-agent",
		PID:     12345,
		Status:  "running",
		CPU:     50.0,
		Memory:  1024000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	receiver.HandleHeartbeat(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 after worker test, got %d", w.Code)
	}
}

// TestStats_ConcurrentUpdate 测试统计信息的并发更新
func TestStats_ConcurrentUpdate(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpDir, err := os.MkdirTemp("", "heartbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	agentInfo := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/test",
		WorkDir:    "/tmp",
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	// 并发发送心跳并读取统计信息
	const numGoroutines = 20
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// 发送心跳
			reqBody := HeartbeatRequest{
				AgentID: "test-agent",
				PID:     12345 + id,
				Status:  "running",
				CPU:     float64(id % 100),
				Memory:  uint64(1024000 + id*1000),
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			receiver.HandleHeartbeat(w, req)

			// 读取统计信息
			stats := receiver.GetStats()
			_ = stats.TotalReceived // 读取统计信息

			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// 成功
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent stats update timeout")
		}
	}

	// 验证最终统计信息
	finalStats := receiver.GetStats()
	if finalStats.TotalReceived == 0 {
		t.Error("expected TotalReceived > 0")
	}

	t.Logf("concurrent stats update test completed: TotalReceived=%d, TotalProcessed=%d",
		finalStats.TotalReceived, finalStats.TotalProcessed)
}
