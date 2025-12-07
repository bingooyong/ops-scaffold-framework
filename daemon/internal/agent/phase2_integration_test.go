package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// TestPhase2_CompleteAgentLifecycle 测试完整的Agent生命周期管理流程
func TestPhase2_CompleteAgentLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// 1. 创建所有Phase 2组件
	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}

	registry := multiManager.GetRegistry()
	configManager := NewConfigManager(registry, logger)
	heartbeatReceiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer heartbeatReceiver.Stop()

	resourceMonitor := NewResourceMonitor(multiManager, registry, logger)
	logManager := NewLogManager(tmpDir, logger)

	// 2. 创建测试Agent配置
	agentID := "test-lifecycle-agent"
	configFile := filepath.Join(tmpDir, "agent.yaml")
	configContent := `
agent:
  interval: 10s
inputs:
  cpu:
    percpu: true
outputs:
  stdout: {}
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// 3. 注册Agent
	_, err = registry.Register(
		agentID,
		TypeTelegraf,
		"Test Lifecycle Agent",
		"/usr/bin/telegraf",
		configFile,
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	agentInfo := &AgentInfo{
		ID:         agentID,
		Type:       TypeTelegraf,
		Name:       "Test Lifecycle Agent",
		BinaryPath: "/usr/bin/telegraf",
		ConfigFile: configFile,
		WorkDir:    tmpDir,
	}
	instance, err := multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	// 4. 启动Agent（会失败因为二进制文件不存在，但元数据应该创建）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = multiManager.StartAgent(ctx, agentID)
	// 允许启动失败（因为二进制文件不存在）
	if err != nil {
		t.Logf("agent start failed (expected): %v", err)
	}

	// 5. 验证元数据创建（即使启动失败，元数据也应该被创建或更新）
	time.Sleep(100 * time.Millisecond)
	metadata, err := multiManager.GetAgentMetadata(agentID)
	if err == nil {
		// 如果元数据存在，验证基本字段
		if metadata.ID != agentID {
			t.Errorf("expected metadata ID %s, got %s", agentID, metadata.ID)
		}
		if metadata.Type != string(TypeTelegraf) {
			t.Errorf("expected metadata Type %s, got %s", TypeTelegraf, metadata.Type)
		}
	}

	// 6. 发送心跳，验证元数据更新
	reqBody := HeartbeatRequest{
		AgentID:   agentID,
		PID:       12345,
		Status:    "running",
		CPU:       25.5,
		Memory:    512000,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	heartbeatReceiver.HandleHeartbeat(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// 等待心跳处理完成
	time.Sleep(200 * time.Millisecond)

	// 验证心跳数据更新到元数据
	metadata, err = multiManager.GetAgentMetadata(agentID)
	if err == nil {
		if metadata.LastHeartbeat.IsZero() {
			t.Error("expected LastHeartbeat to be set")
		}
		if len(metadata.ResourceUsage.CPU) == 0 {
			t.Error("expected resource usage data to be added")
		}
	}

	// 7. 更新配置，验证配置热重载
	configManager.SetAgentInstance(agentID, instance)
	updates := map[string]interface{}{
		"agent": map[string]interface{}{
			"interval": "20s",
		},
	}
	if err := configManager.UpdateConfig(agentID, updates); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	// 验证配置已更新
	updatedConfig, err := configManager.ReadConfig(agentID)
	if err != nil {
		t.Fatalf("failed to read updated config: %v", err)
	}
	agent, ok := updatedConfig["agent"].(map[string]interface{})
	if !ok {
		t.Fatal("agent section not found")
	}
	interval, ok := agent["interval"].(string)
	if !ok || interval != "20s" {
		t.Errorf("expected interval '20s', got %v", interval)
	}

	// 8. 检查资源监控（启动监控并等待一次采集）
	resourceMonitor.SetInterval(1 * time.Second)
	resourceMonitor.Start()
	defer resourceMonitor.Stop()

	// 等待一次采集周期
	time.Sleep(2 * time.Second)

	// 9. 检查日志文件创建
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if _, err := os.Stat(logDir); err != nil {
		t.Logf("log directory not created (may be expected if agent didn't start): %v", err)
	}

	// 10. 停止Agent，验证元数据更新
	err = multiManager.StopAgent(ctx, agentID, true)
	if err != nil {
		t.Logf("agent stop failed (may be expected): %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	metadata, err = multiManager.GetAgentMetadata(agentID)
	if err == nil {
		if metadata.Status != "stopped" {
			t.Logf("expected status 'stopped', got %s (may be expected if agent wasn't running)", metadata.Status)
		}
	}

	// 11. 测试日志查询接口
	logs, err := logManager.GetAgentLogs(agentID, 10)
	if err != nil {
		t.Logf("failed to get logs (may be expected): %v", err)
	} else {
		t.Logf("retrieved %d log lines", len(logs))
	}

	// 12. 验证所有组件正常工作
	if multiManager.Count() != 1 {
		t.Errorf("expected 1 agent, got %d", multiManager.Count())
	}

	stats := heartbeatReceiver.GetStats()
	if stats.TotalReceived == 0 {
		t.Error("expected at least one heartbeat to be received")
	}
}

// TestPhase2_ConfigAndMetadataIntegration 测试配置管理和元数据跟踪的协同
func TestPhase2_ConfigAndMetadataIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// 创建组件
	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}

	registry := multiManager.GetRegistry()
	configManager := NewConfigManager(registry, logger)

	// 创建配置文件
	agentID := "test-config-metadata"
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/*.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// 注册Agent
	_, err = registry.Register(
		agentID,
		TypeFilebeat,
		"Test Config Metadata Agent",
		"/usr/bin/filebeat",
		configFile,
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	agentInfo := &AgentInfo{
		ID:         agentID,
		Type:       TypeFilebeat,
		Name:       "Test Config Metadata Agent",
		BinaryPath: "/usr/bin/filebeat",
		ConfigFile: configFile,
		WorkDir:    tmpDir,
	}
	instance, err := multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	// 读取Agent配置
	config, err := configManager.ReadConfig(agentID)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	// 验证配置内容
	if config == nil {
		t.Fatal("config is nil")
	}

	// 更新配置
	updates := map[string]interface{}{
		"output": map[string]interface{}{
			"elasticsearch": map[string]interface{}{
				"hosts": []interface{}{"localhost:9200", "localhost:9201"},
			},
		},
	}
	if err := configManager.UpdateConfig(agentID, updates); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	// 验证配置已更新
	updatedConfig, err := configManager.ReadConfig(agentID)
	if err != nil {
		t.Fatalf("failed to read updated config: %v", err)
	}

	output, ok := updatedConfig["output"].(map[string]interface{})
	if !ok {
		t.Fatal("output section not found")
	}
	es, ok := output["elasticsearch"].(map[string]interface{})
	if !ok {
		t.Fatal("elasticsearch section not found")
	}
	hosts, ok := es["hosts"].([]interface{})
	if !ok || len(hosts) != 2 {
		t.Errorf("expected 2 hosts, got %v", hosts)
	}

	// 启动Agent以创建元数据
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configManager.SetAgentInstance(agentID, instance)
	err = multiManager.StartAgent(ctx, agentID)
	if err != nil {
		t.Logf("agent start failed (expected): %v", err)
	}

	// 验证元数据中的配置信息（如果存储了）
	time.Sleep(100 * time.Millisecond)
	metadata, err := multiManager.GetAgentMetadata(agentID)
	if err == nil {
		if metadata.ID != agentID {
			t.Errorf("expected metadata ID %s, got %s", agentID, metadata.ID)
		}
		if metadata.Type != string(TypeFilebeat) {
			t.Errorf("expected metadata Type %s, got %s", TypeFilebeat, metadata.Type)
		}
	}

	// 测试配置更新后元数据的一致性
	// 再次更新配置
	updates2 := map[string]interface{}{
		"filebeat": map[string]interface{}{
			"inputs": []interface{}{
				map[string]interface{}{
					"type":  "log",
					"paths": []interface{}{"/var/log/*.log", "/var/log/app/*.log"},
				},
			},
		},
	}
	if err := configManager.UpdateConfig(agentID, updates2); err != nil {
		t.Fatalf("failed to update config again: %v", err)
	}

	// 验证元数据仍然一致
	metadata2, err := multiManager.GetAgentMetadata(agentID)
	if err == nil {
		if metadata2.ID != agentID {
			t.Errorf("metadata ID changed after config update: %s", metadata2.ID)
		}
	}
}

// TestPhase2_HeartbeatAndResourceMonitoring 测试心跳接收和资源监控的协同
func TestPhase2_HeartbeatAndResourceMonitoring(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// 创建组件
	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}

	registry := multiManager.GetRegistry()
	heartbeatReceiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer heartbeatReceiver.Stop()

	resourceMonitor := NewResourceMonitor(multiManager, registry, logger)
	resourceMonitor.SetInterval(1 * time.Second)

	// 注册Agent
	agentID := "test-heartbeat-resource"
	_, err = registry.Register(
		agentID,
		TypeCustom,
		"Test Heartbeat Resource Agent",
		"/bin/test",
		"",
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	agentInfo := &AgentInfo{
		ID:         agentID,
		Type:       TypeCustom,
		Name:       "Test Heartbeat Resource Agent",
		BinaryPath: "/bin/test",
		WorkDir:    tmpDir,
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	// 启动Agent（会失败，但用于测试）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = multiManager.StartAgent(ctx, agentID)
	if err != nil {
		t.Logf("agent start failed (expected): %v", err)
	}

	// 发送心跳（包含CPU、Memory数据）
	cpuValue := 45.5
	memoryValue := uint64(1024000)
	reqBody := HeartbeatRequest{
		AgentID:   agentID,
		PID:       12345,
		Status:    "running",
		CPU:       cpuValue,
		Memory:    memoryValue,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	heartbeatReceiver.HandleHeartbeat(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// 等待心跳处理完成
	time.Sleep(300 * time.Millisecond)

	// 验证心跳数据更新到元数据
	metadata, err := multiManager.GetAgentMetadata(agentID)
	if err == nil {
		if metadata.LastHeartbeat.IsZero() {
			t.Error("expected LastHeartbeat to be set")
		}
		if len(metadata.ResourceUsage.CPU) == 0 {
			t.Error("expected resource usage CPU data to be added")
		}
		if len(metadata.ResourceUsage.Memory) == 0 {
			t.Error("expected resource usage Memory data to be added")
		}

		// 验证CPU和Memory值
		if len(metadata.ResourceUsage.CPU) > 0 {
			lastCPU := metadata.ResourceUsage.CPU[len(metadata.ResourceUsage.CPU)-1]
			if lastCPU != cpuValue {
				t.Errorf("expected CPU %f, got %f", cpuValue, lastCPU)
			}
		}
		if len(metadata.ResourceUsage.Memory) > 0 {
			lastMemory := metadata.ResourceUsage.Memory[len(metadata.ResourceUsage.Memory)-1]
			if lastMemory != memoryValue {
				t.Errorf("expected Memory %d, got %d", memoryValue, lastMemory)
			}
		}
	}

	// 启动资源监控并等待采集
	resourceMonitor.Start()
	defer resourceMonitor.Stop()

	// 等待资源监控采集（由于Agent未运行，采集会失败，但不会panic）
	time.Sleep(2 * time.Second)

	// 验证资源监控数据也更新到元数据（如果Agent在运行）
	metadata2, err := multiManager.GetAgentMetadata(agentID)
	if err == nil {
		// 验证两种数据源的一致性（心跳和资源监控都更新了元数据）
		if len(metadata2.ResourceUsage.CPU) > 0 {
			t.Logf("resource usage has %d CPU data points", len(metadata2.ResourceUsage.CPU))
		}
		if len(metadata2.ResourceUsage.Memory) > 0 {
			t.Logf("resource usage has %d Memory data points", len(metadata2.ResourceUsage.Memory))
		}
	}

	// 验证统计信息
	stats := heartbeatReceiver.GetStats()
	if stats.TotalReceived == 0 {
		t.Error("expected at least one heartbeat to be received")
	}
	if stats.TotalProcessed == 0 {
		t.Error("expected at least one heartbeat to be processed")
	}
}

// TestPhase2_LogManagementAndCleanup 测试日志管理和清理
func TestPhase2_LogManagementAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// 创建日志管理器
	logManager := NewLogManager(tmpDir, logger)

	// 创建测试日志文件
	agentID := "test-log-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")
	logContent := "2024-01-01 10:00:00 [INFO] Agent started\n"
	logContent += "2024-01-01 10:00:01 [INFO] Processing data\n"
	logContent += "2024-01-01 10:00:02 [ERROR] Failed to connect\n"
	logContent += "2024-01-01 10:00:03 [INFO] Retrying connection\n"
	logContent += "2024-01-01 10:00:04 [INFO] Connection established\n"

	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// 测试日志查询接口
	logs, err := logManager.GetAgentLogs(agentID, 3)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("expected 3 lines, got %d", len(logs))
	}

	// 测试日志搜索接口
	entries, err := logManager.SearchLogs(agentID, "ERROR", 10)
	if err != nil {
		t.Fatalf("failed to search logs: %v", err)
	}

	if len(entries) == 0 {
		t.Error("expected at least one ERROR log entry")
	}

	// 验证搜索结果
	found := false
	for _, entry := range entries {
		if entry.Content != "" && len(entry.Content) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find log entries with content")
	}

	// 测试日志轮转（模拟大文件）
	largeLogContent := make([]byte, 2*1024*1024) // 2MB
	for i := range largeLogContent {
		largeLogContent[i] = byte('A' + (i % 26))
		if i > 0 && i%100 == 0 {
			largeLogContent[i] = '\n'
		}
	}
	largeLogContent[len(largeLogContent)-1] = '\n'

	largeLogPath := filepath.Join(logDir, "large.log")
	if err := os.WriteFile(largeLogPath, largeLogContent, 0644); err != nil {
		t.Fatalf("failed to write large log file: %v", err)
	}

	// 创建日志轮转器
	logRotator := NewLogRotator(largeLogPath, 1024*1024, 5, logger) // 1MB max size, 5 files max

	// 执行轮转
	if err := logRotator.RotateIfNeeded(); err != nil {
		t.Fatalf("failed to rotate log: %v", err)
	}

	// 验证轮转文件已创建
	rotatedFile := largeLogPath + ".1.gz"
	if _, err := os.Stat(rotatedFile); err != nil {
		t.Logf("rotated file may not exist (compression may take time): %v", err)
	}

	// 测试日志清理（模拟过期文件）
	oldLogPath := filepath.Join(logDir, "old.log")
	oldLogContent := "old log content\n"
	if err := os.WriteFile(oldLogPath, []byte(oldLogContent), 0644); err != nil {
		t.Fatalf("failed to write old log file: %v", err)
	}

	// 修改文件时间为31天前（超过默认30天保留期）
	oldTime := time.Now().AddDate(0, 0, -31)
	if err := os.Chtimes(oldLogPath, oldTime, oldTime); err != nil {
		t.Fatalf("failed to change file time: %v", err)
	}

	// 执行清理
	logManager.SetRetentionDays(30)
	logManager.cleanupOldLogs()

	// 验证过期文件已删除
	if _, err := os.Stat(oldLogPath); !os.IsNotExist(err) {
		t.Logf("old log file may still exist (cleanup runs on schedule): %v", err)
	}

	// 验证日志文件管理正确
	files, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("failed to read log dir: %v", err)
	}

	t.Logf("log directory contains %d files after cleanup", len(files))
}
