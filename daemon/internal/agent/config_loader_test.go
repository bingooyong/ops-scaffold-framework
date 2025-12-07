package agent

import (
	"path/filepath"
	"testing"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"go.uber.org/zap"
)

func TestLoadAgentsFromConfig(t *testing.T) {
	logger := zap.NewNop()
	registry := NewAgentRegistry()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			config.AgentItemConfig{
				ID:         "filebeat-1",
				Type:       "filebeat",
				Name:       "Filebeat Agent",
				BinaryPath: "/usr/bin/filebeat",
				ConfigFile: "/etc/filebeat/filebeat.yml",
				WorkDir:    "/var/lib/daemon/agents/filebeat-1",
				Enabled:    true,
			},
			config.AgentItemConfig{
				ID:         "telegraf-1",
				Type:       "telegraf",
				BinaryPath: "/usr/bin/telegraf",
				ConfigFile: "/etc/telegraf/telegraf.conf",
				Enabled:    true,
			},
			config.AgentItemConfig{
				ID:         "disabled-agent",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
				Enabled:    false, // 未启用
			},
		},
	}

	err := LoadAgentsFromConfig(cfg, registry, "/var/lib/daemon", logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证已注册的Agent（应该只有2个，disabled-agent被跳过）
	if registry.Count() != 2 {
		t.Errorf("expected 2 agents registered, got %d", registry.Count())
	}

	// 验证filebeat-1
	info1 := registry.Get("filebeat-1")
	if info1 == nil {
		t.Fatal("expected filebeat-1 to be registered")
	}
	if info1.Type != TypeFilebeat {
		t.Errorf("expected Type TypeFilebeat, got %s", info1.Type)
	}
	if info1.Name != "Filebeat Agent" {
		t.Errorf("expected Name 'Filebeat Agent', got '%s'", info1.Name)
	}
	if info1.WorkDir != "/var/lib/daemon/agents/filebeat-1" {
		t.Errorf("expected WorkDir '/var/lib/daemon/agents/filebeat-1', got '%s'", info1.WorkDir)
	}

	// 验证telegraf-1（使用默认工作目录）
	info2 := registry.Get("telegraf-1")
	if info2 == nil {
		t.Fatal("expected telegraf-1 to be registered")
	}
	expectedWorkDir := filepath.Join("/var/lib/daemon", "agents", "telegraf-1")
	if info2.WorkDir != expectedWorkDir {
		t.Errorf("expected WorkDir '%s', got '%s'", expectedWorkDir, info2.WorkDir)
	}

	// 验证disabled-agent未被注册
	if registry.Exists("disabled-agent") {
		t.Error("expected disabled-agent not to be registered")
	}
}

func TestLoadAgentsFromConfig_EmptyConfig(t *testing.T) {
	logger := zap.NewNop()
	registry := NewAgentRegistry()

	cfg := &config.Config{
		Agents: config.AgentsConfig{},
	}

	err := LoadAgentsFromConfig(cfg, registry, "/var/lib/daemon", logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("expected 0 agents, got %d", registry.Count())
	}
}

func TestLoadAgentsFromConfig_DefaultName(t *testing.T) {
	logger := zap.NewNop()
	registry := NewAgentRegistry()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			config.AgentItemConfig{
				ID:         "agent-1",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
				Enabled:    true,
				// Name未设置，应该使用Type作为默认值
			},
		},
	}

	err := LoadAgentsFromConfig(cfg, registry, "/var/lib/daemon", logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info := registry.Get("agent-1")
	if info == nil {
		t.Fatal("expected agent-1 to be registered")
	}
	// 注意：Name的默认值设置是在mergeAgentConfigs中完成的，这里我们测试的是config_loader
	// 如果Name为空，config_loader会使用Type作为默认值
	if info.Name == "" {
		t.Error("expected Name to be set")
	}
}

func TestParseAgentType(t *testing.T) {
	tests := []struct {
		input    string
		expected AgentType
	}{
		{"filebeat", TypeFilebeat},
		{"telegraf", TypeTelegraf},
		{"node_exporter", TypeNodeExporter},
		{"custom", TypeCustom},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := parseAgentType(tt.input)
		if result != tt.expected {
			t.Errorf("parseAgentType(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestGetAgentConfig(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			config.AgentItemConfig{
				ID:         "agent-1",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
			},
			config.AgentItemConfig{
				ID:         "agent-2",
				Type:       "telegraf",
				BinaryPath: "/usr/bin/telegraf",
			},
		},
	}

	// 测试获取存在的Agent配置
	agentCfg := GetAgentConfig(cfg, "agent-1")
	if agentCfg == nil {
		t.Fatal("expected non-nil config for agent-1")
	}
	if agentCfg.ID != "agent-1" {
		t.Errorf("expected ID 'agent-1', got '%s'", agentCfg.ID)
	}

	// 测试获取不存在的Agent配置
	agentCfg = GetAgentConfig(cfg, "non-existent")
	if agentCfg != nil {
		t.Error("expected nil config for non-existent agent")
	}
}

func TestLoadAgentsFromConfig_InvalidType(t *testing.T) {
	logger := zap.NewNop()
	registry := NewAgentRegistry()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			config.AgentItemConfig{
				ID:         "agent-1",
				Type:       "invalid-type",
				BinaryPath: "/usr/bin/agent",
				Enabled:    true,
			},
		},
	}

	err := LoadAgentsFromConfig(cfg, registry, "/var/lib/daemon", logger)
	if err == nil {
		t.Fatal("expected error for invalid agent type")
	}
}

func TestLoadAgentsFromConfig_DuplicateID(t *testing.T) {
	logger := zap.NewNop()
	registry := NewAgentRegistry()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			config.AgentItemConfig{
				ID:         "agent-1",
				Type:       "filebeat",
				BinaryPath: "/usr/bin/filebeat",
				Enabled:    true,
			},
		},
	}

	// 第一次加载
	err := LoadAgentsFromConfig(cfg, registry, "/var/lib/daemon", logger)
	if err != nil {
		t.Fatalf("unexpected error on first load: %v", err)
	}

	// 第二次加载相同配置（应该失败，因为ID已存在）
	err = LoadAgentsFromConfig(cfg, registry, "/var/lib/daemon", logger)
	if err == nil {
		t.Fatal("expected error for duplicate agent ID")
	}
}
