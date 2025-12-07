package agent

import (
	"context"
	"testing"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"go.uber.org/zap"
)

func TestNewMultiHealthChecker(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	cfg := &MultiHealthCheckerConfig{}

	mhc := NewMultiHealthChecker(mam, cfg, logger)
	if mhc == nil {
		t.Fatal("expected non-nil MultiHealthChecker")
	}
}

func TestMultiHealthChecker_RegisterAgent(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	healthCheckCfg := &config.HealthCheckConfig{
		Interval:          30 * time.Second,
		HeartbeatTimeout:  90 * time.Second,
		CPUThreshold:      50.0,
		MemoryThreshold:   524288000,
		ThresholdDuration: 60 * time.Second,
	}

	mhc.RegisterAgent("test-agent", healthCheckCfg)

	// 验证Agent已注册
	status := mhc.GetHealthStatus("test-agent")
	if status == nil {
		t.Fatal("expected non-nil health status")
	}
	if status.AgentID != "test-agent" {
		t.Errorf("expected AgentID 'test-agent', got '%s'", status.AgentID)
	}
	if status.Status != types.HealthStatusHealthy {
		t.Errorf("expected initial status HealthStatusHealthy, got %v", status.Status)
	}
}

func TestMultiHealthChecker_ReceiveHeartbeat(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	healthCheckCfg := &config.HealthCheckConfig{
		Interval:         30 * time.Second,
		HeartbeatTimeout: 90 * time.Second,
	}
	mhc.RegisterAgent("test-agent", healthCheckCfg)

	// 发送心跳
	hb := &types.Heartbeat{
		PID:       12345,
		Timestamp: time.Now(),
		CPU:       10.0,
		Memory:    1000000,
	}

	mhc.ReceiveHeartbeat("test-agent", hb)

	// 验证心跳时间已更新
	lastHB := mhc.GetLastHeartbeat("test-agent")
	if lastHB.IsZero() {
		t.Error("expected last heartbeat to be set")
	}

	// 验证健康状态中的心跳时间
	status := mhc.GetHealthStatus("test-agent")
	if status == nil {
		t.Fatal("expected non-nil health status")
	}
	status.mu.RLock()
	heartbeatTime := status.LastHeartbeat
	status.mu.RUnlock()
	if heartbeatTime.IsZero() {
		t.Error("expected LastHeartbeat to be set in health status")
	}
}

func TestMultiHealthChecker_GetHealthStatus(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	healthCheckCfg := &config.HealthCheckConfig{
		Interval: 30 * time.Second,
	}
	mhc.RegisterAgent("test-agent", healthCheckCfg)

	// 获取健康状态
	status := mhc.GetHealthStatus("test-agent")
	if status == nil {
		t.Fatal("expected non-nil health status")
	}
	if status.AgentID != "test-agent" {
		t.Errorf("expected AgentID 'test-agent', got '%s'", status.AgentID)
	}

	// 获取不存在的Agent状态
	status = mhc.GetHealthStatus("non-existent")
	if status != nil {
		t.Error("expected nil for non-existent agent")
	}
}

func TestMultiHealthChecker_GetAllHealthStatuses(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	// 注册多个Agent
	healthCheckCfg := &config.HealthCheckConfig{
		Interval: 30 * time.Second,
	}
	mhc.RegisterAgent("agent-1", healthCheckCfg)
	mhc.RegisterAgent("agent-2", healthCheckCfg)

	// 获取所有健康状态
	statuses := mhc.GetAllHealthStatuses()
	if len(statuses) != 2 {
		t.Errorf("expected 2 health statuses, got %d", len(statuses))
	}

	// 验证所有Agent都在状态列表中
	if _, exists := statuses["agent-1"]; !exists {
		t.Error("expected agent-1 in statuses")
	}
	if _, exists := statuses["agent-2"]; !exists {
		t.Error("expected agent-2 in statuses")
	}
}

func TestMultiHealthChecker_CheckHealth_Dead(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	// 注册Agent（但不创建实例，模拟进程不存在）
	healthCheckCfg := &config.HealthCheckConfig{
		Interval: 30 * time.Second,
	}
	mhc.RegisterAgent("test-agent", healthCheckCfg)

	// 检查健康状态（应该返回Dead，因为Agent实例不存在）
	status := mhc.checkHealth("test-agent", healthCheckCfg)
	if status != types.HealthStatusDead {
		t.Errorf("expected HealthStatusDead, got %v", status)
	}
}

func TestMultiHealthChecker_CheckHealth_NoHeartbeat(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	// 注册Agent并创建实例
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		BinaryPath: "/usr/bin/filebeat",
		ConfigFile: "/etc/filebeat/filebeat.yml",
		WorkDir:    "/tmp/test-agent",
	}
	instance, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 设置Agent为运行状态（但不实际启动）
	info.SetStatus(StatusRunning)
	info.SetPID(12345)

	healthCheckCfg := &config.HealthCheckConfig{
		Interval:         30 * time.Second,
		HeartbeatTimeout: 5 * time.Second, // 短超时用于测试
	}
	mhc.RegisterAgent("test-agent", healthCheckCfg)

	// 设置一个过时的心跳（超过超时时间）
	pastTime := time.Now().Add(-10 * time.Second)
	mhc.mu.Lock()
	mhc.heartbeats["test-agent"] = pastTime
	mhc.mu.Unlock()

	// 检查健康状态（应该返回NoHeartbeat）
	status := mhc.checkHealth("test-agent", healthCheckCfg)
	if status != types.HealthStatusNoHeartbeat {
		t.Errorf("expected HealthStatusNoHeartbeat, got %v", status)
	}

	_ = instance // 避免未使用变量警告
}

func TestMultiHealthChecker_CheckHealthByType(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	healthCheckCfg := &config.HealthCheckConfig{
		Interval: 30 * time.Second,
	}

	// 测试Filebeat类型（使用进程检查）
	info1 := &AgentInfo{
		ID:   "filebeat-1",
		Type: TypeFilebeat,
	}
	instance1, _ := mam.RegisterAgent(info1)
	status1 := mhc.checkHealthByType("filebeat-1", info1, healthCheckCfg, instance1)
	// 应该返回Dead（因为进程未运行）
	if status1 != types.HealthStatusDead {
		t.Errorf("expected HealthStatusDead for filebeat, got %v", status1)
	}

	// 测试Node Exporter类型（使用HTTP检查，但简化实现使用进程检查）
	info2 := &AgentInfo{
		ID:   "node-exporter-1",
		Type: TypeNodeExporter,
	}
	instance2, _ := mam.RegisterAgent(info2)
	status2 := mhc.checkHealthByType("node-exporter-1", info2, healthCheckCfg, instance2)
	// 应该返回Dead（因为进程未运行）
	if status2 != types.HealthStatusDead {
		t.Errorf("expected HealthStatusDead for node_exporter, got %v", status2)
	}

	_ = instance1
	_ = instance2
}

func TestMultiHealthChecker_UpdateHealthStatus(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	healthCheckCfg := &config.HealthCheckConfig{
		Interval: 30 * time.Second,
	}
	mhc.RegisterAgent("test-agent", healthCheckCfg)

	// 更新健康状态
	mhc.updateHealthStatus("test-agent", types.HealthStatusOverThreshold)

	status := mhc.GetHealthStatus("test-agent")
	if status == nil {
		t.Fatal("expected non-nil health status")
	}
	status.mu.RLock()
	if status.Status != types.HealthStatusOverThreshold {
		t.Errorf("expected Status HealthStatusOverThreshold, got %v", status.Status)
	}
	if status.OverThresholdSince.IsZero() {
		t.Error("expected OverThresholdSince to be set")
	}
	status.mu.RUnlock()
}

func TestBuildMultiHealthCheckerConfig(t *testing.T) {
	cfg := &config.Config{
		AgentDefaults: config.AgentDefaultsConfig{
			HealthCheck: config.HealthCheckConfig{
				Interval:          30 * time.Second,
				HeartbeatTimeout:  90 * time.Second,
				CPUThreshold:      50.0,
				MemoryThreshold:   524288000,
				ThresholdDuration: 60 * time.Second,
			},
		},
		Agents: config.AgentsConfig{
			config.AgentItemConfig{
				ID:   "agent-1",
				Type: "filebeat",
				HealthCheck: config.HealthCheckConfig{
					CPUThreshold: 40.0, // 覆盖默认值
				},
			},
			config.AgentItemConfig{
				ID:   "agent-2",
				Type: "telegraf",
				// 使用默认值
			},
		},
	}

	mhcCfg := BuildMultiHealthCheckerConfig(cfg)

	// 验证配置
	if len(mhcCfg.AgentConfigs) != 2 {
		t.Errorf("expected 2 agent configs, got %d", len(mhcCfg.AgentConfigs))
	}

	// 验证agent-1覆盖了默认值
	agent1Cfg := mhcCfg.AgentConfigs["agent-1"]
	if agent1Cfg.CPUThreshold != 40.0 {
		t.Errorf("expected CPUThreshold 40.0, got %f", agent1Cfg.CPUThreshold)
	}
	// 其他字段应该使用默认值
	if agent1Cfg.Interval != 30*time.Second {
		t.Errorf("expected Interval 30s, got %v", agent1Cfg.Interval)
	}

	// 验证agent-2使用默认值
	agent2Cfg := mhcCfg.AgentConfigs["agent-2"]
	if agent2Cfg.CPUThreshold != 50.0 {
		t.Errorf("expected CPUThreshold 50.0, got %f", agent2Cfg.CPUThreshold)
	}
}

func TestHTTPHealthCheck(t *testing.T) {
	// 测试HTTP健康检查函数
	// 注意：这个测试需要一个可用的HTTP端点，在测试环境中可能不可用
	// 这里只测试函数不会panic

	// 测试无效端点（应该返回false）
	result := HTTPHealthCheck("http://invalid-endpoint:9999", 1*time.Second)
	if result {
		t.Error("expected false for invalid endpoint")
	}
}

func TestMultiHealthChecker_Stop(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	mhc := NewMultiHealthChecker(mam, nil, logger)

	// 启动健康检查器（即使没有Agent，也应该能正常停止）
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 在goroutine中启动和停止
	done := make(chan bool)
	go func() {
		mhc.Start()
		<-ctx.Done()
		mhc.Stop()
		done <- true
	}()

	// 等待停止完成
	select {
	case <-done:
		// 成功停止
	case <-time.After(200 * time.Millisecond):
		t.Error("health checker stop timeout")
	}
}
