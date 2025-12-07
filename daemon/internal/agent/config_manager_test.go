package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// createTestConfigManager 创建测试用的ConfigManager
func createTestConfigManager(t *testing.T) (*ConfigManager, *AgentRegistry, string) {
	logger := zaptest.NewLogger(t, zaptest.Level(zap.InfoLevel))
	registry := NewAgentRegistry()

	// 创建临时目录
	tmpDir := t.TempDir()

	cm := NewConfigManager(registry, logger)
	return cm, registry, tmpDir
}

// createTestConfigFile 创建测试配置文件
func createTestConfigFile(t *testing.T, dir, filename string, content string) string {
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}
	return filePath
}

func TestReadConfig_YAML(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 创建测试配置文件
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
	configFile := createTestConfigFile(t, tmpDir, "filebeat.yaml", configContent)

	// 注册Agent
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
		t.Fatalf("failed to register agent: %v", err)
	}

	// 读取配置
	config, err := cm.ReadConfig("test-filebeat")
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	// 验证配置内容
	if config == nil {
		t.Fatal("config is nil")
	}

	filebeat, ok := config["filebeat"].(map[string]interface{})
	if !ok {
		t.Fatal("filebeat section not found or invalid")
	}

	inputs, ok := filebeat["inputs"].([]interface{})
	if !ok || len(inputs) == 0 {
		t.Fatal("filebeat.inputs not found or empty")
	}
}

func TestReadConfig_JSON(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 创建测试配置文件
	configContent := `{
  "agent": {
    "interval": "10s"
  },
  "inputs": {
    "cpu": {
      "percpu": true
    }
  },
  "outputs": {
    "influxdb": {
      "urls": ["http://localhost:8086"]
    }
  }
}`
	configFile := createTestConfigFile(t, tmpDir, "telegraf.json", configContent)

	// 注册Agent
	_, err := registry.Register(
		"test-telegraf",
		TypeTelegraf,
		"Test Telegraf",
		"/usr/bin/telegraf",
		configFile,
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 读取配置
	config, err := cm.ReadConfig("test-telegraf")
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	// 验证配置内容
	if config == nil {
		t.Fatal("config is nil")
	}

	agent, ok := config["agent"].(map[string]interface{})
	if !ok {
		t.Fatal("agent section not found or invalid")
	}

	interval, ok := agent["interval"].(string)
	if !ok || interval != "10s" {
		t.Fatalf("expected interval '10s', got %v", interval)
	}
}

func TestReadConfig_NotFound(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 注册Agent（配置文件不存在）
	_, err := registry.Register(
		"test-agent",
		TypeFilebeat,
		"Test Agent",
		"/usr/bin/filebeat",
		filepath.Join(tmpDir, "nonexistent.yaml"),
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 尝试读取配置
	_, err = cm.ReadConfig("test-agent")
	if err == nil {
		t.Fatal("expected error for nonexistent config file")
	}

	if !os.IsNotExist(err) && !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}

	// 测试Agent不存在的情况
	_, err = cm.ReadConfig("nonexistent-agent")
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}

	if !strings.Contains(err.Error(), "agent not found") {
		t.Fatalf("expected 'agent not found' error, got: %v", err)
	}
}

func TestValidateConfig_Filebeat(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 注册Agent
	_, err := registry.Register(
		"test-filebeat",
		TypeFilebeat,
		"Test Filebeat",
		"/usr/bin/filebeat",
		filepath.Join(tmpDir, "filebeat.yaml"),
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 测试有效配置
	validConfig := map[string]interface{}{
		"filebeat.inputs": []interface{}{
			map[string]interface{}{
				"type":  "log",
				"paths": []interface{}{"/var/log/*.log"},
			},
		},
		"output": map[string]interface{}{
			"elasticsearch": map[string]interface{}{
				"hosts": []interface{}{"localhost:9200"},
			},
		},
	}

	if err := cm.ValidateConfig("test-filebeat", validConfig); err != nil {
		t.Fatalf("valid config should pass validation: %v", err)
	}

	// 测试缺少 filebeat.inputs
	invalidConfig1 := map[string]interface{}{
		"output": map[string]interface{}{
			"elasticsearch": map[string]interface{}{
				"hosts": []interface{}{"localhost:9200"},
			},
		},
	}

	if err := cm.ValidateConfig("test-filebeat", invalidConfig1); err == nil {
		t.Fatal("expected validation error for missing filebeat.inputs")
	}

	// 测试缺少 output
	invalidConfig2 := map[string]interface{}{
		"filebeat.inputs": []interface{}{
			map[string]interface{}{
				"type":  "log",
				"paths": []interface{}{"/var/log/*.log"},
			},
		},
	}

	if err := cm.ValidateConfig("test-filebeat", invalidConfig2); err == nil {
		t.Fatal("expected validation error for missing output")
	}
}

func TestValidateConfig_Telegraf(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 注册Agent
	_, err := registry.Register(
		"test-telegraf",
		TypeTelegraf,
		"Test Telegraf",
		"/usr/bin/telegraf",
		filepath.Join(tmpDir, "telegraf.conf"),
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 测试有效配置（有inputs）
	validConfig1 := map[string]interface{}{
		"agent": map[string]interface{}{
			"interval": "10s",
		},
		"inputs": map[string]interface{}{
			"cpu": map[string]interface{}{
				"percpu": true,
			},
		},
	}

	if err := cm.ValidateConfig("test-telegraf", validConfig1); err != nil {
		t.Fatalf("valid config should pass validation: %v", err)
	}

	// 测试有效配置（有outputs）
	validConfig2 := map[string]interface{}{
		"agent": map[string]interface{}{
			"interval": "10s",
		},
		"outputs": map[string]interface{}{
			"influxdb": map[string]interface{}{
				"urls": []interface{}{"http://localhost:8086"},
			},
		},
	}

	if err := cm.ValidateConfig("test-telegraf", validConfig2); err != nil {
		t.Fatalf("valid config should pass validation: %v", err)
	}

	// 测试缺少 agent
	invalidConfig1 := map[string]interface{}{
		"inputs": map[string]interface{}{
			"cpu": map[string]interface{}{
				"percpu": true,
			},
		},
	}

	if err := cm.ValidateConfig("test-telegraf", invalidConfig1); err == nil {
		t.Fatal("expected validation error for missing agent")
	}

	// 测试缺少 inputs 和 outputs
	invalidConfig2 := map[string]interface{}{
		"agent": map[string]interface{}{
			"interval": "10s",
		},
	}

	if err := cm.ValidateConfig("test-telegraf", invalidConfig2); err == nil {
		t.Fatal("expected validation error for missing inputs and outputs")
	}
}

func TestValidateConfig_NodeExporter(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 注册Agent
	_, err := registry.Register(
		"test-node-exporter",
		TypeNodeExporter,
		"Test Node Exporter",
		"/usr/bin/node_exporter",
		"", // Node Exporter 可能不使用配置文件
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 空配置应该通过验证
	emptyConfig := map[string]interface{}{}
	if err := cm.ValidateConfig("test-node-exporter", emptyConfig); err != nil {
		t.Fatalf("empty config should pass validation for node_exporter: %v", err)
	}

	// 非空配置也应该通过（仅做基本检查）
	nonEmptyConfig := map[string]interface{}{
		"some": "config",
	}
	if err := cm.ValidateConfig("test-node-exporter", nonEmptyConfig); err != nil {
		t.Fatalf("non-empty config should pass validation for node_exporter: %v", err)
	}
}

func TestUpdateConfig_Success(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 创建初始配置文件
	initialConfig := `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/app.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`
	configFile := createTestConfigFile(t, tmpDir, "filebeat.yaml", initialConfig)

	// 注册Agent
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
		t.Fatalf("failed to register agent: %v", err)
	}

	// 更新配置
	updates := map[string]interface{}{
		"output": map[string]interface{}{
			"elasticsearch": map[string]interface{}{
				"hosts": []interface{}{"localhost:9200", "localhost:9201"},
			},
		},
	}

	if err := cm.UpdateConfig("test-filebeat", updates); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	// 验证配置已更新
	updatedConfig, err := cm.ReadConfig("test-filebeat")
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
		t.Fatalf("expected 2 hosts, got %v", hosts)
	}
}

func TestUpdateConfig_AtomicWrite(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 创建初始配置文件
	initialConfig := `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/app.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`
	configFile := createTestConfigFile(t, tmpDir, "filebeat.yaml", initialConfig)

	// 注册Agent
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
		t.Fatalf("failed to register agent: %v", err)
	}

	// 读取原始文件内容
	originalData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("failed to read original config: %v", err)
	}

	// 更新配置
	updates := map[string]interface{}{
		"output": map[string]interface{}{
			"console": map[string]interface{}{
				"pretty": true,
			},
		},
	}

	if err := cm.UpdateConfig("test-filebeat", updates); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	// 验证原始文件已被替换（内容不同）
	updatedData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("failed to read updated config: %v", err)
	}

	if string(originalData) == string(updatedData) {
		t.Fatal("config file should have been updated")
	}

	// 验证临时文件不存在
	tmpFile := configFile + ".tmp"
	if _, err := os.Stat(tmpFile); err == nil {
		t.Fatal("temp file should not exist after update")
	}
}

func TestUpdateConfig_ValidationFail(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 创建初始配置文件
	initialConfig := `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/app.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`
	configFile := createTestConfigFile(t, tmpDir, "filebeat.yaml", initialConfig)

	// 注册Agent
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
		t.Fatalf("failed to register agent: %v", err)
	}

	// 读取原始文件内容
	originalData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("failed to read original config: %v", err)
	}

	// 尝试更新为无效配置（缺少必需字段）
	invalidUpdates := map[string]interface{}{
		"filebeat.inputs": nil, // 删除必需字段
	}

	err = cm.UpdateConfig("test-filebeat", invalidUpdates)
	if err == nil {
		t.Fatal("expected validation error")
	}

	// 验证配置文件未被修改
	currentData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("failed to read current config: %v", err)
	}

	if string(originalData) != string(currentData) {
		t.Fatal("config file should not be modified when validation fails")
	}
}

func TestStartWatching(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 创建配置文件
	configFile := createTestConfigFile(t, tmpDir, "filebeat.yaml", `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/app.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`)

	// 注册Agent
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
		t.Fatalf("failed to register agent: %v", err)
	}

	// 启动监听
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cm.StartWatching(ctx); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}

	// 等待一小段时间确保watcher已启动
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件
	newContent := `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/app.log
        - /var/log/error.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`
	if err := os.WriteFile(configFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// 等待文件变化被检测到
	time.Sleep(200 * time.Millisecond)

	// 停止监听
	if err := cm.StopWatching(); err != nil {
		t.Fatalf("failed to stop watching: %v", err)
	}
}

func TestStopWatching(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 创建配置文件
	configFile := createTestConfigFile(t, tmpDir, "filebeat.yaml", `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/app.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`)

	// 注册Agent
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
		t.Fatalf("failed to register agent: %v", err)
	}

	// 启动监听
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cm.StartWatching(ctx); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}

	// 停止监听
	if err := cm.StopWatching(); err != nil {
		t.Fatalf("failed to stop watching: %v", err)
	}

	// 再次停止应该不报错
	if err := cm.StopWatching(); err != nil {
		t.Fatalf("stopping twice should not error: %v", err)
	}
}

// TestReadConfig_ConcurrentRead 测试并发读取配置
func TestReadConfig_ConcurrentRead(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	configFile := createTestConfigFile(t, tmpDir, "filebeat.yaml", `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/*.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`)

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
		t.Fatalf("failed to register agent: %v", err)
	}

	// 并发读取配置
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			config, err := cm.ReadConfig("test-filebeat")
			if err != nil {
				errors <- err
				return
			}
			if config == nil {
				errors <- fmt.Errorf("config is nil")
				return
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// 成功
		case err := <-errors:
			t.Errorf("concurrent read failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent read timeout")
		}
	}
}

// TestUpdateConfig_ConcurrentUpdate 测试并发更新配置（验证原子性）
func TestUpdateConfig_ConcurrentUpdate(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	configFile := createTestConfigFile(t, tmpDir, "filebeat.yaml", `
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/*.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`)

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
		t.Fatalf("failed to register agent: %v", err)
	}

	// 并发更新配置
	const numGoroutines = 5
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			updates := map[string]interface{}{
				"output": map[string]interface{}{
					"elasticsearch": map[string]interface{}{
						"hosts": []interface{}{fmt.Sprintf("localhost:%d", 9200+id)},
					},
				},
			}
			err := cm.UpdateConfig("test-filebeat", updates)
			if err != nil {
				errors <- err
				return
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			successCount++
		case err := <-errors:
			t.Logf("concurrent update error (may be expected): %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent update timeout")
		}
	}

	// 验证最终配置的一致性（应该能读取到某个更新后的配置）
	finalConfig, err := cm.ReadConfig("test-filebeat")
	if err != nil {
		t.Fatalf("failed to read final config: %v", err)
	}
	if finalConfig == nil {
		t.Fatal("final config is nil")
	}

	t.Logf("concurrent updates completed: %d successful", successCount)
}

// TestValidateConfig_EdgeCases 测试边界值验证（空配置、超大配置等）
func TestValidateConfig_EdgeCases(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	_, err := registry.Register(
		"test-filebeat",
		TypeFilebeat,
		"Test Filebeat",
		"/usr/bin/filebeat",
		filepath.Join(tmpDir, "filebeat.yaml"),
		tmpDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 测试空配置
	emptyConfig := map[string]interface{}{}
	err = cm.ValidateConfig("test-filebeat", emptyConfig)
	if err == nil {
		t.Error("expected validation error for empty config")
	}

	// 测试超大配置（模拟）
	largeConfig := map[string]interface{}{
		"filebeat.inputs": make([]interface{}, 10000),
		"output": map[string]interface{}{
			"elasticsearch": map[string]interface{}{
				"hosts": make([]interface{}, 1000),
			},
		},
	}
	// 验证应该能处理（即使很大）
	err = cm.ValidateConfig("test-filebeat", largeConfig)
	// 允许验证通过或失败，只要不panic
	if err != nil {
		t.Logf("large config validation error (may be expected): %v", err)
	}

	// 测试nil值
	nilConfig := map[string]interface{}{
		"filebeat": nil,
		"output":   nil,
	}
	err = cm.ValidateConfig("test-filebeat", nilConfig)
	if err == nil {
		t.Error("expected validation error for nil config values")
	}
}

// TestStartWatching_MultipleAgents 测试监听多个Agent配置文件
func TestStartWatching_MultipleAgents(t *testing.T) {
	cm, registry, tmpDir := createTestConfigManager(t)

	// 创建多个配置文件
	configFiles := []string{"filebeat1.yaml", "filebeat2.yaml", "filebeat3.yaml"}
	agentIDs := []string{"filebeat-1", "filebeat-2", "filebeat-3"}

	for i, filename := range configFiles {
		configFile := createTestConfigFile(t, tmpDir, filename, fmt.Sprintf(`
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/app%d.log
output:
  elasticsearch:
    hosts: ["localhost:9200"]
`, i+1))

		_, err := registry.Register(
			agentIDs[i],
			TypeFilebeat,
			fmt.Sprintf("Test Filebeat %d", i+1),
			"/usr/bin/filebeat",
			configFile,
			tmpDir,
			"",
		)
		if err != nil {
			t.Fatalf("failed to register agent %s: %v", agentIDs[i], err)
		}
	}

	// 启动监听
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cm.StartWatching(ctx); err != nil {
		t.Fatalf("failed to start watching: %v", err)
	}

	// 等待一下确保监听已启动
	time.Sleep(100 * time.Millisecond)

	// 停止监听
	if err := cm.StopWatching(); err != nil {
		t.Fatalf("failed to stop watching: %v", err)
	}

	// 验证所有Agent配置都能读取
	for _, agentID := range agentIDs {
		config, err := cm.ReadConfig(agentID)
		if err != nil {
			t.Errorf("failed to read config for %s: %v", agentID, err)
		}
		if config == nil {
			t.Errorf("config is nil for %s", agentID)
		}
	}
}
