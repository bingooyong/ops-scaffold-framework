package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConvertLegacyAgentConfig(t *testing.T) {
	// 测试旧格式转换为新格式
	config := &Config{
		Agent: AgentConfig{
			BinaryPath: "/usr/bin/agent",
			ConfigFile: "/etc/agent/agent.yaml",
			WorkDir:    "/var/lib/agent",
		},
		Agents: AgentsConfig{},
	}

	err := convertLegacyAgentConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证转换结果
	if len(config.Agents) != 1 {
		t.Fatalf("expected 1 agent after conversion, got %d", len(config.Agents))
	}

	agent := config.Agents[0]
	if agent.ID != "legacy-agent" {
		t.Errorf("expected ID 'legacy-agent', got '%s'", agent.ID)
	}
	if agent.BinaryPath != "/usr/bin/agent" {
		t.Errorf("expected BinaryPath '/usr/bin/agent', got '%s'", agent.BinaryPath)
	}
	if agent.ConfigFile != "/etc/agent/agent.yaml" {
		t.Errorf("expected ConfigFile '/etc/agent/agent.yaml', got '%s'", agent.ConfigFile)
	}
}

func TestConvertLegacyAgentConfig_NewFormatExists(t *testing.T) {
	// 测试新格式已存在时不转换
	config := &Config{
		Agent: AgentConfig{
			BinaryPath: "/usr/bin/agent",
		},
		Agents: AgentsConfig{
			AgentItemConfig{
				ID:         "new-agent",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
			},
		},
	}

	err := convertLegacyAgentConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证新格式保持不变
	if len(config.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(config.Agents))
	}
	if config.Agents[0].ID != "new-agent" {
		t.Errorf("expected ID 'new-agent', got '%s'", config.Agents[0].ID)
	}
}

func TestConvertLegacyAgentConfig_NoLegacyConfig(t *testing.T) {
	// 测试没有旧格式配置时不转换
	config := &Config{
		Agent: AgentConfig{
			BinaryPath: "", // 空，表示未配置
		},
		Agents: AgentsConfig{},
	}

	err := convertLegacyAgentConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证Agents仍为空
	if len(config.Agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(config.Agents))
	}
}

func TestMergeAgentConfigs(t *testing.T) {
	// 测试配置合并
	config := &Config{
		AgentDefaults: AgentDefaultsConfig{
			HealthCheck: HealthCheckConfig{
				Interval:          30 * time.Second,
				HeartbeatTimeout:  90 * time.Second,
				CPUThreshold:      50.0,
				MemoryThreshold:   524288000,
				ThresholdDuration: 60 * time.Second,
			},
			Restart: RestartConfig{
				MaxRetries:  10,
				BackoffBase: 10 * time.Second,
				BackoffMax:  60 * time.Second,
				Policy:      "always",
			},
		},
		Agents: AgentsConfig{
			// Agent 1: 使用默认值
			AgentItemConfig{
				ID:         "agent-1",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
			},
			// Agent 2: 覆盖部分默认值
			AgentItemConfig{
				ID:         "agent-2",
				Type:       "telegraf",
				BinaryPath: "/usr/bin/telegraf",
				HealthCheck: HealthCheckConfig{
					CPUThreshold: 40.0, // 覆盖默认值
				},
				Restart: RestartConfig{
					Policy: "never", // 覆盖默认值
				},
			},
		},
	}

	err := mergeAgentConfigs(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证Agent 1使用默认值
	agent1 := config.Agents[0]
	if agent1.HealthCheck.Interval != 30*time.Second {
		t.Errorf("expected Interval 30s, got %v", agent1.HealthCheck.Interval)
	}
	if agent1.HealthCheck.CPUThreshold != 50.0 {
		t.Errorf("expected CPUThreshold 50.0, got %f", agent1.HealthCheck.CPUThreshold)
	}
	if agent1.Restart.Policy != "always" {
		t.Errorf("expected Restart Policy 'always', got '%s'", agent1.Restart.Policy)
	}
	if agent1.Name != "filebeat" {
		t.Errorf("expected Name 'filebeat', got '%s'", agent1.Name)
	}

	// 验证Agent 2覆盖了默认值
	agent2 := config.Agents[1]
	if agent2.HealthCheck.CPUThreshold != 40.0 {
		t.Errorf("expected CPUThreshold 40.0, got %f", agent2.HealthCheck.CPUThreshold)
	}
	if agent2.Restart.Policy != "never" {
		t.Errorf("expected Restart Policy 'never', got '%s'", agent2.Restart.Policy)
	}
	// 验证其他字段使用默认值
	if agent2.HealthCheck.Interval != 30*time.Second {
		t.Errorf("expected Interval 30s, got %v", agent2.HealthCheck.Interval)
	}
}

func TestValidateAgentsConfig(t *testing.T) {
	// 测试配置验证
	config := &Config{
		Agents: AgentsConfig{
			AgentItemConfig{
				ID:         "agent-1",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
			},
			AgentItemConfig{
				ID:         "agent-2",
				Type:       "telegraf",
				BinaryPath: "/usr/bin/telegraf",
			},
		},
	}

	err := validateAgentsConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAgentsConfig_MissingID(t *testing.T) {
	config := &Config{
		Agents: AgentsConfig{
			AgentItemConfig{
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
			},
		},
	}

	err := validateAgentsConfig(config)
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
	if err.Error() != "agents[0].id is required" {
		t.Errorf("expected error message about missing id, got: %v", err)
	}
}

func TestValidateAgentsConfig_MissingType(t *testing.T) {
	config := &Config{
		Agents: AgentsConfig{
			AgentItemConfig{
				ID:         "agent-1",
				BinaryPath: "/usr/bin/filebeat",
			},
		},
	}

	err := validateAgentsConfig(config)
	if err == nil {
		t.Fatal("expected error for missing type")
	}
	if err.Error() != "agents[0].type is required" {
		t.Errorf("expected error message about missing type, got: %v", err)
	}
}

func TestValidateAgentsConfig_MissingBinaryPath(t *testing.T) {
	config := &Config{
		Agents: AgentsConfig{
			AgentItemConfig{
				ID:   "agent-1",
				Type: "filebeat",
			},
		},
	}

	err := validateAgentsConfig(config)
	if err == nil {
		t.Fatal("expected error for missing binary_path")
	}
	if err.Error() != "agents[0].binary_path is required" {
		t.Errorf("expected error message about missing binary_path, got: %v", err)
	}
}

func TestValidateAgentsConfig_DuplicateID(t *testing.T) {
	config := &Config{
		Agents: AgentsConfig{
			AgentItemConfig{
				ID:         "agent-1",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
			},
			AgentItemConfig{
				ID:         "agent-1", // 重复ID
				Type:       "telegraf",
				BinaryPath: "/usr/bin/telegraf",
			},
		},
	}

	err := validateAgentsConfig(config)
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
	if err.Error() != "duplicate agent id: agent-1" {
		t.Errorf("expected error message about duplicate id, got: %v", err)
	}
}

func TestValidateAgentsConfig_InvalidType(t *testing.T) {
	config := &Config{
		Agents: AgentsConfig{
			AgentItemConfig{
				ID:         "agent-1",
				Type:       "invalid-type",
				BinaryPath: "/usr/bin/agent",
			},
		},
	}

	err := validateAgentsConfig(config)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestValidateAgentsConfig_InvalidRestartPolicy(t *testing.T) {
	config := &Config{
		Agents: AgentsConfig{
			AgentItemConfig{
				ID:         "agent-1",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
				Restart: RestartConfig{
					Policy: "invalid-policy",
				},
			},
		},
	}

	err := validateAgentsConfig(config)
	if err == nil {
		t.Fatal("expected error for invalid restart policy")
	}
}

func TestValidateAgentsConfig_ValidTypes(t *testing.T) {
	validTypes := []string{"filebeat", "telegraf", "node_exporter", "custom"}

	for _, agentType := range validTypes {
		config := &Config{
			Agents: AgentsConfig{
				AgentItemConfig{
					ID:         "agent-1",
					Type:       agentType,
					BinaryPath: "/usr/bin/agent",
				},
			},
		}

		err := validateAgentsConfig(config)
		if err != nil {
			t.Errorf("expected no error for valid type %s, got: %v", agentType, err)
		}
	}
}

func TestValidateAgentsConfig_ValidRestartPolicies(t *testing.T) {
	validPolicies := []string{"always", "never", "on-failure"}

	for _, policy := range validPolicies {
		config := &Config{
			Agents: AgentsConfig{
				AgentItemConfig{
					ID:         "agent-1",
					Type:       "filebeat",
					BinaryPath: "/usr/bin/filebeat",
					Restart: RestartConfig{
						Policy: policy,
					},
				},
			},
		}

		err := validateAgentsConfig(config)
		if err != nil {
			t.Errorf("expected no error for valid policy %s, got: %v", policy, err)
		}
	}
}

// 测试完整配置加载流程
func TestLoadConfig_WithAgents(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
daemon:
  work_dir: /var/lib/daemon
  log_level: info

agent_defaults:
  health_check:
    interval: 30s
    cpu_threshold: 50.0
  restart:
    max_retries: 10
    policy: always

agents:
  - id: filebeat-1
    type: filebeat
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat.yml
  - id: telegraf-1
    type: telegraf
    binary_path: /usr/bin/telegraf
    config_file: /etc/telegraf/telegraf.conf
    health_check:
      cpu_threshold: 40.0
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 验证Agents配置
	if len(cfg.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(cfg.Agents))
	}

	// 验证第一个Agent
	agent1 := cfg.Agents[0]
	if agent1.ID != "filebeat-1" {
		t.Errorf("expected ID 'filebeat-1', got '%s'", agent1.ID)
	}
	if agent1.HealthCheck.Interval != 30*time.Second {
		t.Errorf("expected Interval 30s, got %v", agent1.HealthCheck.Interval)
	}
	if agent1.HealthCheck.CPUThreshold != 50.0 {
		t.Errorf("expected CPUThreshold 50.0, got %f", agent1.HealthCheck.CPUThreshold)
	}

	// 验证第二个Agent（覆盖了默认值）
	agent2 := cfg.Agents[1]
	if agent2.ID != "telegraf-1" {
		t.Errorf("expected ID 'telegraf-1', got '%s'", agent2.ID)
	}
	if agent2.HealthCheck.CPUThreshold != 40.0 {
		t.Errorf("expected CPUThreshold 40.0, got %f", agent2.HealthCheck.CPUThreshold)
	}
	// 其他字段应该使用默认值
	if agent2.HealthCheck.Interval != 30*time.Second {
		t.Errorf("expected Interval 30s, got %v", agent2.HealthCheck.Interval)
	}
}

func TestLoadConfig_WithLegacyAgent(t *testing.T) {
	// 创建临时配置文件（旧格式）
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
daemon:
  work_dir: /var/lib/daemon
  log_level: info

agent:
  binary_path: /usr/bin/agent
  config_file: /etc/agent/agent.yaml
  work_dir: /var/lib/agent
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 验证旧格式已转换为新格式
	if len(cfg.Agents) != 1 {
		t.Fatalf("expected 1 agent after conversion, got %d", len(cfg.Agents))
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
}
