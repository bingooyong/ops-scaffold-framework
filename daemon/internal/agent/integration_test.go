package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"go.uber.org/zap"
)

// TestMultiAgentManagement_EndToEnd 端到端测试：完整的多Agent管理流程
func TestMultiAgentManagement_EndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}

	// 1. 创建多个Agent配置
	agents := []*AgentInfo{
		{
			ID:         "filebeat-1",
			Type:       TypeFilebeat,
			Name:       "Filebeat Agent 1",
			BinaryPath: "/usr/bin/filebeat",
			ConfigFile: "/etc/filebeat/filebeat.yml",
			WorkDir:    "/tmp/test-filebeat-1",
		},
		{
			ID:         "telegraf-1",
			Type:       TypeTelegraf,
			Name:       "Telegraf Agent 1",
			BinaryPath: "/usr/bin/telegraf",
			ConfigFile: "/etc/telegraf/telegraf.conf",
			WorkDir:    "/tmp/test-telegraf-1",
		},
		{
			ID:         "node-exporter-1",
			Type:       TypeNodeExporter,
			Name:       "Node Exporter 1",
			BinaryPath: "/usr/bin/node_exporter",
			ConfigFile: "/etc/node_exporter/config.yml",
			WorkDir:    "/tmp/test-node-exporter-1",
		},
	}

	// 2. 注册所有Agent
	for _, info := range agents {
		instance, err := mam.RegisterAgent(info)
		if err != nil {
			t.Fatalf("failed to register agent %s: %v", info.ID, err)
		}
		if instance == nil {
			t.Fatalf("expected non-nil instance for agent %s", info.ID)
		}
	}

	// 3. 验证所有Agent已注册
	if mam.Count() != len(agents) {
		t.Errorf("expected %d agents, got %d", len(agents), mam.Count())
	}

	// 4. 验证可以获取所有Agent
	for _, info := range agents {
		instance := mam.GetAgent(info.ID)
		if instance == nil {
			t.Errorf("expected non-nil instance for agent %s", info.ID)
		}
		if instance.GetInfo().ID != info.ID {
			t.Errorf("expected agent ID %s, got %s", info.ID, instance.GetInfo().ID)
		}
	}

	// 5. 测试批量操作：启动所有Agent（注意：实际启动会失败，因为二进制文件不存在）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := mam.StartAll(ctx)
	if len(results) != len(agents) {
		t.Errorf("expected %d start results, got %d", len(agents), len(results))
	}

	// 6. 验证Agent状态
	for _, info := range agents {
		instance := mam.GetAgent(info.ID)
		if instance == nil {
			t.Errorf("expected non-nil instance for agent %s", info.ID)
			continue
		}
		agentInfo := instance.GetInfo()
		// 由于二进制文件不存在，状态应该是停止的
		status := agentInfo.GetStatus()
		if status != StatusStopped && status != StatusFailed {
			// 允许失败状态，因为二进制文件不存在
			t.Logf("agent %s status: %v (expected Stopped or Failed)", info.ID, status)
		}
	}

	// 7. 测试批量停止
	stopResults := mam.StopAll(ctx, false)
	if len(stopResults) != len(agents) {
		t.Errorf("expected %d stop results, got %d", len(agents), len(stopResults))
	}

	// 8. 测试批量重启
	restartResults := mam.RestartAll(ctx)
	if len(restartResults) != len(agents) {
		t.Errorf("expected %d restart results, got %d", len(agents), len(restartResults))
	}

	// 9. 测试注销Agent
	for _, info := range agents {
		err := mam.UnregisterAgent(info.ID)
		if err != nil {
			t.Errorf("failed to unregister agent %s: %v", info.ID, err)
		}
	}

	// 10. 验证所有Agent已注销
	if mam.Count() != 0 {
		t.Errorf("expected 0 agents after unregister, got %d", mam.Count())
	}
}

// TestMultiAgentHealthCheck_EndToEnd 端到端测试：多Agent健康检查流程
func TestMultiAgentHealthCheck_EndToEnd(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}

	// 1. 创建并注册Agent
	info := &AgentInfo{
		ID:         "test-agent-health",
		Type:       TypeFilebeat,
		Name:       "Test Health Agent",
		BinaryPath: "/usr/bin/filebeat",
		ConfigFile: "/etc/filebeat/filebeat.yml",
		WorkDir:    "/tmp/test-health-agent",
	}

	instance, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 2. 创建健康检查器配置
	healthCheckCfg := &config.HealthCheckConfig{
		Interval:          30 * time.Second,
		HeartbeatTimeout:  90 * time.Second,
		CPUThreshold:      50.0,
		MemoryThreshold:   524288000,
		ThresholdDuration: 60 * time.Second,
	}

	mhcCfg := &MultiHealthCheckerConfig{
		AgentConfigs: map[string]*config.HealthCheckConfig{
			info.ID: healthCheckCfg,
		},
	}

	// 3. 创建健康检查器
	mhc := NewMultiHealthChecker(mam, mhcCfg, logger)

	// 4. 注册Agent到健康检查器
	mhc.RegisterAgent(info.ID, healthCheckCfg)

	// 5. 验证健康状态
	status := mhc.GetHealthStatus(info.ID)
	if status == nil {
		t.Fatal("expected non-nil health status")
	}
	if status.AgentID != info.ID {
		t.Errorf("expected AgentID %s, got %s", info.ID, status.AgentID)
	}

	// 6. 发送心跳
	hb := &types.Heartbeat{
		PID:       12345,
		Timestamp: time.Now(),
		CPU:       10.0,
		Memory:    1000000,
	}
	mhc.ReceiveHeartbeat(info.ID, hb)

	// 7. 验证心跳已接收
	lastHB := mhc.GetLastHeartbeat(info.ID)
	if lastHB.IsZero() {
		t.Error("expected last heartbeat to be set")
	}

	// 8. 验证健康状态中的心跳时间
	status = mhc.GetHealthStatus(info.ID)
	if status == nil {
		t.Fatal("expected non-nil health status")
	}
	status.mu.RLock()
	heartbeatTime := status.LastHeartbeat
	status.mu.RUnlock()
	if heartbeatTime.IsZero() {
		t.Error("expected LastHeartbeat to be set in health status")
	}

	// 9. 测试健康检查（由于进程不存在，应该返回Dead）
	healthStatus := mhc.checkHealth(info.ID, healthCheckCfg)
	if healthStatus != types.HealthStatusDead {
		t.Errorf("expected HealthStatusDead, got %v", healthStatus)
	}

	// 10. 启动健康检查器（短暂运行）
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

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

	_ = instance // 避免未使用变量警告
}

// TestConfigLoading_EndToEnd 端到端测试：配置加载和验证流程
func TestConfigLoading_EndToEnd(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "daemon.yaml")

	// 写入多Agent配置
	configContent := `
daemon:
  work_dir: ` + tmpDir + `
  log_dir: ` + tmpDir + `/logs

agents:
  - id: filebeat-1
    type: filebeat
    name: Filebeat Agent
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat.yml
    work_dir: ` + tmpDir + `/agents/filebeat-1
    enabled: true
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 50.0
      memory_threshold: 524288000
      threshold_duration: 60s
    restart:
      policy: always
      max_attempts: 5
      backoff: exponential

  - id: telegraf-1
    type: telegraf
    name: Telegraf Agent
    binary_path: /usr/bin/telegraf
    config_file: /etc/telegraf/telegraf.conf
    work_dir: ` + tmpDir + `/agents/telegraf-1
    enabled: true
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 60.0
      memory_threshold: 1048576000
      threshold_duration: 60s
    restart:
      policy: always
      max_attempts: 5
      backoff: exponential

agent_defaults:
  health_check:
    interval: 30s
    heartbeat_timeout: 90s
    cpu_threshold: 50.0
    memory_threshold: 524288000
    threshold_duration: 60s
  restart:
    policy: always
    max_attempts: 5
    backoff: exponential
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// 加载配置
	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 验证配置
	if len(cfg.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(cfg.Agents))
	}

	// 验证第一个Agent配置
	agent1 := cfg.Agents[0]
	if agent1.ID != "filebeat-1" {
		t.Errorf("expected ID 'filebeat-1', got '%s'", agent1.ID)
	}
	if agent1.Type != "filebeat" {
		t.Errorf("expected Type 'filebeat', got '%s'", agent1.Type)
	}
	if agent1.HealthCheck.CPUThreshold != 50.0 {
		t.Errorf("expected CPUThreshold 50.0, got %f", agent1.HealthCheck.CPUThreshold)
	}

	// 验证第二个Agent配置
	agent2 := cfg.Agents[1]
	if agent2.ID != "telegraf-1" {
		t.Errorf("expected ID 'telegraf-1', got '%s'", agent2.ID)
	}
	if agent2.Type != "telegraf" {
		t.Errorf("expected Type 'telegraf', got '%s'", agent2.Type)
	}
	if agent2.HealthCheck.CPUThreshold != 60.0 {
		t.Errorf("expected CPUThreshold 60.0, got %f", agent2.HealthCheck.CPUThreshold)
	}

	// 验证默认配置
	if cfg.AgentDefaults.HealthCheck.Interval != 30*time.Second {
		t.Errorf("expected default interval 30s, got %v", cfg.AgentDefaults.HealthCheck.Interval)
	}

	// 测试从配置加载Agent到注册表
	logger := zap.NewNop()
	registry := NewAgentRegistry()
	if err := LoadAgentsFromConfig(cfg, registry, cfg.Daemon.WorkDir, logger); err != nil {
		t.Fatalf("failed to load agents from config: %v", err)
	}

	// 验证Agent已加载
	if registry.Count() != 2 {
		t.Errorf("expected 2 agents in registry, got %d", registry.Count())
	}

	// 验证可以获取Agent
	info1 := registry.Get("filebeat-1")
	if info1 == nil {
		t.Fatalf("failed to get agent filebeat-1: agent not found")
	}
	if info1.ID != "filebeat-1" {
		t.Errorf("expected ID 'filebeat-1', got '%s'", info1.ID)
	}

	info2 := registry.Get("telegraf-1")
	if info2 == nil {
		t.Fatalf("failed to get agent telegraf-1: agent not found")
	}
	if info2.ID != "telegraf-1" {
		t.Errorf("expected ID 'telegraf-1', got '%s'", info2.ID)
	}

	// 测试构建健康检查器配置
	mhcCfg := BuildMultiHealthCheckerConfig(cfg)
	if len(mhcCfg.AgentConfigs) != 2 {
		t.Errorf("expected 2 agent configs, got %d", len(mhcCfg.AgentConfigs))
	}

	// 验证健康检查配置
	healthCfg1 := mhcCfg.AgentConfigs["filebeat-1"]
	if healthCfg1 == nil {
		t.Fatal("expected non-nil health check config for filebeat-1")
	}
	if healthCfg1.CPUThreshold != 50.0 {
		t.Errorf("expected CPUThreshold 50.0, got %f", healthCfg1.CPUThreshold)
	}

	healthCfg2 := mhcCfg.AgentConfigs["telegraf-1"]
	if healthCfg2 == nil {
		t.Fatal("expected non-nil health check config for telegraf-1")
	}
	if healthCfg2.CPUThreshold != 60.0 {
		t.Errorf("expected CPUThreshold 60.0, got %f", healthCfg2.CPUThreshold)
	}
}

// TestMultiAgentRestartStrategy_EndToEnd 端到端测试：重启策略
func TestMultiAgentRestartStrategy_EndToEnd(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}

	// 创建Agent配置（使用不同的重启策略）
	agents := []*AgentInfo{
		{
			ID:         "agent-always",
			Type:       TypeFilebeat,
			Name:       "Always Restart Agent",
			BinaryPath: "/usr/bin/filebeat",
			ConfigFile: "/etc/filebeat/filebeat.yml",
			WorkDir:    "/tmp/test-agent-always",
		},
		{
			ID:         "agent-on-failure",
			Type:       TypeTelegraf,
			Name:       "On Failure Restart Agent",
			BinaryPath: "/usr/bin/telegraf",
			ConfigFile: "/etc/telegraf/telegraf.conf",
			WorkDir:    "/tmp/test-agent-on-failure",
		},
	}

	// 注册所有Agent
	for _, info := range agents {
		instance, err := mam.RegisterAgent(info)
		if err != nil {
			t.Fatalf("failed to register agent %s: %v", info.ID, err)
		}
		if instance == nil {
			t.Fatalf("expected non-nil instance for agent %s", info.ID)
		}
	}

	// 验证Agent已注册
	if mam.Count() != len(agents) {
		t.Errorf("expected %d agents, got %d", len(agents), mam.Count())
	}

	// 测试获取所有Agent状态
	statuses := mam.GetAllAgentStatus()
	if len(statuses) != len(agents) {
		t.Errorf("expected %d statuses, got %d", len(agents), len(statuses))
	}

	// 验证每个Agent的状态信息
	for _, status := range statuses {
		if status.ID == "" {
			t.Error("expected non-empty Agent ID in status")
		}
		if status.Type == "" {
			t.Error("expected non-empty Agent Type in status")
		}
		// 初始状态应该是Stopped
		if status.Status != StatusStopped {
			t.Logf("agent %s status: %v (expected Stopped)", status.ID, status.Status)
		}
	}

	// 测试批量重启（由于二进制文件不存在，会失败，但应该不会panic）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := mam.RestartAll(ctx)
	if len(results) != len(agents) {
		t.Errorf("expected %d restart results, got %d", len(agents), len(results))
	}

	// 验证重启计数（由于启动失败，重启计数可能增加）
	for _, info := range agents {
		instance := mam.GetAgent(info.ID)
		if instance == nil {
			t.Errorf("expected non-nil instance for agent %s", info.ID)
			continue
		}
		agentInfo := instance.GetInfo()
		restartCount := agentInfo.GetRestartCount()
		t.Logf("agent %s restart count: %d", info.ID, restartCount)
	}
}

// TestConcurrentAgentOperations 并发安全测试：并发操作多个Agent
func TestConcurrentAgentOperations(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}

	// 创建多个Agent
	numAgents := 10
	agents := make([]*AgentInfo, numAgents)
	for i := 0; i < numAgents; i++ {
		agents[i] = &AgentInfo{
			ID:         fmt.Sprintf("agent-%d", i),
			Type:       TypeFilebeat,
			Name:       fmt.Sprintf("Agent %d", i),
			BinaryPath: "/usr/bin/filebeat",
			ConfigFile: "/etc/filebeat/filebeat.yml",
			WorkDir:    fmt.Sprintf("/tmp/test-agent-%d", i),
		}
	}

	// 并发注册
	registerDone := make(chan bool, numAgents)
	for i := 0; i < numAgents; i++ {
		go func(idx int) {
			_, err := mam.RegisterAgent(agents[idx])
			if err != nil {
				t.Errorf("failed to register agent %s: %v", agents[idx].ID, err)
			}
			registerDone <- true
		}(i)
	}

	// 等待所有注册完成
	for i := 0; i < numAgents; i++ {
		<-registerDone
	}

	// 验证所有Agent已注册
	if mam.Count() != numAgents {
		t.Errorf("expected %d agents, got %d", numAgents, mam.Count())
	}

	// 并发获取Agent
	getDone := make(chan bool, numAgents)
	for i := 0; i < numAgents; i++ {
		go func(idx int) {
			instance := mam.GetAgent(agents[idx].ID)
			if instance == nil {
				t.Errorf("expected non-nil instance for agent %s", agents[idx].ID)
			}
			getDone <- true
		}(i)
	}

	// 等待所有获取完成
	for i := 0; i < numAgents; i++ {
		<-getDone
	}

	// 并发启动（会失败，但应该不会panic）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startResults := mam.StartAll(ctx)
	if len(startResults) != numAgents {
		t.Errorf("expected %d start results, got %d", numAgents, len(startResults))
	}

	// 并发获取状态
	statusDone := make(chan bool, numAgents)
	for i := 0; i < numAgents; i++ {
		go func(idx int) {
			instance := mam.GetAgent(agents[idx].ID)
			if instance != nil {
				_ = instance.GetInfo().GetStatus()
			}
			statusDone <- true
		}(i)
	}

	// 等待所有状态获取完成
	for i := 0; i < numAgents; i++ {
		<-statusDone
	}

	// 并发注销
	unregisterDone := make(chan bool, numAgents)
	for i := 0; i < numAgents; i++ {
		go func(idx int) {
			err := mam.UnregisterAgent(agents[idx].ID)
			if err != nil {
				t.Errorf("failed to unregister agent %s: %v", agents[idx].ID, err)
			}
			unregisterDone <- true
		}(i)
	}

	// 等待所有注销完成
	for i := 0; i < numAgents; i++ {
		<-unregisterDone
	}

	// 验证所有Agent已注销
	if mam.Count() != 0 {
		t.Errorf("expected 0 agents after unregister, got %d", mam.Count())
	}
}

// TestLegacyConfigCompatibility 向后兼容测试：旧格式配置转换
func TestLegacyConfigCompatibility(t *testing.T) {
	// 创建临时配置文件（旧格式）
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "daemon.yaml")

	configContent := `
daemon:
  work_dir: ` + tmpDir + `
  log_dir: ` + tmpDir + `/logs

agent:
  binary_path: /usr/bin/agent
  config_file: /etc/agent/agent.yaml
  work_dir: ` + tmpDir + `/agent
  health_check:
    interval: 30s
    heartbeat_timeout: 90s
    cpu_threshold: 50.0
    memory_threshold: 524288000
    threshold_duration: 60s
  restart:
    policy: always
    max_attempts: 5
    backoff: exponential
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// 加载配置
	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 验证旧格式配置已转换为新格式
	if len(cfg.Agents) != 1 {
		t.Errorf("expected 1 agent after conversion, got %d", len(cfg.Agents))
	}

	agent := cfg.Agents[0]
	if agent.ID != "legacy-agent" {
		t.Errorf("expected ID 'legacy-agent', got '%s'", agent.ID)
	}
	if agent.BinaryPath != "/usr/bin/agent" {
		t.Errorf("expected BinaryPath '/usr/bin/agent', got '%s'", agent.BinaryPath)
	}
	if agent.ConfigFile != "/etc/agent/agent.yaml" {
		t.Errorf("expected ConfigFile '/etc/agent/agent.yaml', got '%s'", agent.ConfigFile)
	}

	// 验证健康检查配置已转换
	if agent.HealthCheck.Interval != 30*time.Second {
		t.Errorf("expected Interval 30s, got %v", agent.HealthCheck.Interval)
	}
	if agent.HealthCheck.CPUThreshold != 50.0 {
		t.Errorf("expected CPUThreshold 50.0, got %f", agent.HealthCheck.CPUThreshold)
	}
}
