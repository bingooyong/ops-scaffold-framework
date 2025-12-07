package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// BenchmarkConfigManager_ReadConfig 配置读取性能基准测试
func BenchmarkConfigManager_ReadConfig(b *testing.B) {
	tmpDir := b.TempDir()

	cm, registry, _ := createTestConfigManager(&testing.T{})

	configFile := filepath.Join(tmpDir, "filebeat.yaml")
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
		b.Fatalf("failed to create config file: %v", err)
	}

	_, err := registry.Register(
		"test-filebeat",
		TypeFilebeat,
		"Test Filebeat",
		"/usr/bin/filebeat",
		configFile,
		tmpDir,
		"",
	)
	if err != nil {
		b.Fatalf("failed to register agent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cm.ReadConfig("test-filebeat")
		if err != nil {
			b.Fatalf("failed to read config: %v", err)
		}
	}
}

// BenchmarkConfigManager_UpdateConfig 配置更新性能基准测试
func BenchmarkConfigManager_UpdateConfig(b *testing.B) {
	tmpDir := b.TempDir()

	cm, registry, _ := createTestConfigManager(&testing.T{})

	configFile := filepath.Join(tmpDir, "filebeat.yaml")
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
		b.Fatalf("failed to create config file: %v", err)
	}

	_, err := registry.Register(
		"test-filebeat",
		TypeFilebeat,
		"Test Filebeat",
		"/usr/bin/filebeat",
		configFile,
		tmpDir,
		"",
	)
	if err != nil {
		b.Fatalf("failed to register agent: %v", err)
	}

	updates := map[string]interface{}{
		"output": map[string]interface{}{
			"elasticsearch": map[string]interface{}{
				"hosts": []interface{}{"localhost:9200", "localhost:9201"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cm.UpdateConfig("test-filebeat", updates)
		if err != nil {
			b.Fatalf("failed to update config: %v", err)
		}
	}
}

// BenchmarkConfigManager_ConcurrentRead 并发读取配置性能基准测试
func BenchmarkConfigManager_ConcurrentRead(b *testing.B) {
	tmpDir := b.TempDir()

	cm, registry, _ := createTestConfigManager(&testing.T{})

	configFile := filepath.Join(tmpDir, "filebeat.yaml")
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
		b.Fatalf("failed to create config file: %v", err)
	}

	_, err := registry.Register(
		"test-filebeat",
		TypeFilebeat,
		"Test Filebeat",
		"/usr/bin/filebeat",
		configFile,
		tmpDir,
		"",
	)
	if err != nil {
		b.Fatalf("failed to register agent: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := cm.ReadConfig("test-filebeat")
			if err != nil {
				b.Fatalf("failed to read config: %v", err)
			}
		}
	})
}

// BenchmarkMetadataStore_SaveMetadata 元数据保存性能基准测试
func BenchmarkMetadataStore_SaveMetadata(b *testing.B) {
	tmpDir := b.TempDir()
	logger := zaptest.NewLogger(b)

	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		b.Fatalf("failed to create metadata store: %v", err)
	}

	metadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Version:       "7.14.0",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		LastHeartbeat: time.Now(),
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := store.SaveMetadata("test-agent", metadata)
		if err != nil {
			b.Fatalf("failed to save metadata: %v", err)
		}
	}
}

// BenchmarkMetadataStore_ConcurrentSave 并发保存元数据性能基准测试
func BenchmarkMetadataStore_ConcurrentSave(b *testing.B) {
	tmpDir := b.TempDir()
	logger := zaptest.NewLogger(b)

	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		b.Fatalf("failed to create metadata store: %v", err)
	}

	metadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Version:       "7.14.0",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		LastHeartbeat: time.Now(),
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := store.SaveMetadata("test-agent", metadata)
			if err != nil {
				b.Fatalf("failed to save metadata: %v", err)
			}
		}
	})
}

// BenchmarkHeartbeatReceiver_ProcessHeartbeat 心跳处理性能基准测试
func BenchmarkHeartbeatReceiver_ProcessHeartbeat(b *testing.B) {
	tmpDir := b.TempDir()
	logger := zaptest.NewLogger(b)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		b.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		b.Fatalf("failed to register agent: %v", err)
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
		b.Fatalf("failed to register agent to manager: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	hb := &Heartbeat{
		AgentID:   "test-agent",
		PID:       12345,
		Status:    "running",
		CPU:       50.0,
		Memory:    1024000,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		receiver.processHeartbeat(hb)
	}
}

// BenchmarkHeartbeatReceiver_ConcurrentHeartbeat 并发心跳处理性能基准测试
func BenchmarkHeartbeatReceiver_ConcurrentHeartbeat(b *testing.B) {
	tmpDir := b.TempDir()
	logger := zaptest.NewLogger(b)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		b.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	_, err = registry.Register("test-agent", TypeCustom, "Test Agent", "/bin/test", "", "/tmp", "")
	if err != nil {
		b.Fatalf("failed to register agent: %v", err)
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
		b.Fatalf("failed to register agent to manager: %v", err)
	}

	receiver := NewHTTPHeartbeatReceiver(multiManager, registry, logger)
	defer receiver.Stop()

	hb := &Heartbeat{
		AgentID:   "test-agent",
		PID:       12345,
		Status:    "running",
		CPU:       50.0,
		Memory:    1024000,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			receiver.processHeartbeat(hb)
		}
	})
}

// BenchmarkResourceMonitor_CollectResources 资源采集性能基准测试
func BenchmarkResourceMonitor_CollectResources(b *testing.B) {
	logger := zaptest.NewLogger(b)
	workDir := b.TempDir()

	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		b.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 注册一个Agent（不启动，因为需要真实进程）
	agentID := "test-agent"
	_, err = registry.Register(
		agentID,
		TypeCustom,
		"Test Agent",
		"/bin/echo",
		"",
		workDir,
		"",
	)
	if err != nil {
		b.Fatalf("failed to register agent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 由于Agent未运行，会失败，但可以测试性能
		_, _ = monitor.collectAgentResources(agentID)
	}
}

// BenchmarkLogManager_GetAgentLogs 日志查询性能基准测试
func BenchmarkLogManager_GetAgentLogs(b *testing.B) {
	tmpDir := b.TempDir()
	logger := zaptest.NewLogger(b)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		b.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")

	// 创建中等大小的日志文件（1MB）
	const fileSize = 1024 * 1024
	logContent := make([]byte, fileSize)
	for i := range logContent {
		logContent[i] = byte('A' + (i % 26))
		if i > 0 && i%100 == 0 {
			logContent[i] = '\n'
		}
	}
	logContent[len(logContent)-1] = '\n'

	if err := os.WriteFile(logPath, logContent, 0644); err != nil {
		b.Fatalf("failed to write log file: %v", err)
	}

	lm := NewLogManager(tmpDir, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := lm.GetAgentLogs(agentID, 100)
		if err != nil {
			b.Fatalf("failed to get logs: %v", err)
		}
	}
}

// BenchmarkResourceUsageHistory_AddResourceData 资源使用历史添加数据性能基准测试
func BenchmarkResourceUsageHistory_AddResourceData(b *testing.B) {
	history := NewResourceUsageHistory(1440)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cpu := float64(i % 100)
		memory := uint64(1024*1024*100 + i*1024)
		history.AddResourceData(cpu, memory)
	}
}

// BenchmarkResourceUsageHistory_GetRecent 资源使用历史查询性能基准测试
func BenchmarkResourceUsageHistory_GetRecent(b *testing.B) {
	history := NewResourceUsageHistory(1440)

	// 预填充数据
	for i := 0; i < 1440; i++ {
		cpu := float64(i % 100)
		memory := uint64(1024*1024*100 + i*1024)
		history.AddResourceData(cpu, memory)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = history.GetRecentCPU(1 * time.Hour)
		_ = history.GetRecentMemory(1 * time.Hour)
	}
}

// BenchmarkMultiAgentManager_RegisterAgent 注册Agent性能基准测试
func BenchmarkMultiAgentManager_RegisterAgent(b *testing.B) {
	tmpDir := b.TempDir()
	logger := zaptest.NewLogger(b)

	multiManager, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		b.Fatalf("failed to create multi agent manager: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentID := "test-agent-" + string(rune(i))
		agentInfo := &AgentInfo{
			ID:         agentID,
			Type:       TypeCustom,
			Name:       "Test Agent",
			BinaryPath: "/bin/test",
			WorkDir:    "/tmp",
		}
		_, err := multiManager.RegisterAgent(agentInfo)
		if err != nil {
			b.Fatalf("failed to register agent: %v", err)
		}
	}
}

// BenchmarkMetadataStore_GetMetadata 获取元数据性能基准测试
func BenchmarkMetadataStore_GetMetadata(b *testing.B) {
	tmpDir := b.TempDir()
	logger := zaptest.NewLogger(b)

	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		b.Fatalf("failed to create metadata store: %v", err)
	}

	// 预创建元数据
	metadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}
	if err := store.SaveMetadata("test-agent", metadata); err != nil {
		b.Fatalf("failed to save metadata: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetMetadata("test-agent")
		if err != nil {
			b.Fatalf("failed to get metadata: %v", err)
		}
	}
}
