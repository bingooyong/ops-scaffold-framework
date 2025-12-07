package agent

import (
	"fmt"
	"path/filepath"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"go.uber.org/zap"
)

// LoadAgentsFromConfig 从配置加载所有Agent到注册表
func LoadAgentsFromConfig(cfg *config.Config, registry *AgentRegistry, daemonWorkDir string, logger *zap.Logger) error {
	// 如果新格式agents配置为空，不加载
	if len(cfg.Agents) == 0 {
		logger.Info("no agents configured")
		return nil
	}

	// 遍历所有Agent配置
	for _, agentCfg := range cfg.Agents {
		// 跳过未启用的Agent
		if !agentCfg.Enabled {
			logger.Info("agent disabled, skipping",
				zap.String("agent_id", agentCfg.ID),
				zap.String("agent_type", agentCfg.Type))
			continue
		}

		// 确定工作目录
		workDir := agentCfg.WorkDir
		if workDir == "" {
			// 使用默认工作目录: {daemon.work_dir}/agents/{id}
			workDir = filepath.Join(daemonWorkDir, "agents", agentCfg.ID)
		}

		// 确定名称
		name := agentCfg.Name
		if name == "" {
			name = agentCfg.Type
		}

		// 转换Agent类型
		agentType := parseAgentType(agentCfg.Type)
		if agentType == "" {
			return fmt.Errorf("invalid agent type: %s (agent: %s)", agentCfg.Type, agentCfg.ID)
		}

		// 注册Agent到注册表
		info, err := registry.Register(
			agentCfg.ID,
			agentType,
			name,
			agentCfg.BinaryPath,
			agentCfg.ConfigFile,
			workDir,
			agentCfg.SocketPath,
		)
		if err != nil {
			return fmt.Errorf("failed to register agent %s: %w", agentCfg.ID, err)
		}

		logger.Info("agent loaded from config",
			zap.String("agent_id", info.ID),
			zap.String("agent_type", string(info.Type)),
			zap.String("agent_name", info.Name),
			zap.String("binary_path", info.BinaryPath),
			zap.String("work_dir", info.WorkDir))
	}

	return nil
}

// parseAgentType 解析Agent类型字符串为AgentType
func parseAgentType(typeStr string) AgentType {
	switch typeStr {
	case "filebeat":
		return TypeFilebeat
	case "telegraf":
		return TypeTelegraf
	case "node_exporter":
		return TypeNodeExporter
	case "custom":
		return TypeCustom
	default:
		return ""
	}
}

// GetAgentConfig 从配置中获取指定Agent的配置
func GetAgentConfig(cfg *config.Config, agentID string) *config.AgentItemConfig {
	for i := range cfg.Agents {
		if cfg.Agents[i].ID == agentID {
			return &cfg.Agents[i]
		}
	}
	return nil
}

// BuildMultiHealthCheckerConfig 从配置构建MultiHealthChecker配置
func BuildMultiHealthCheckerConfig(cfg *config.Config) *MultiHealthCheckerConfig {
	agentConfigs := make(map[string]*config.HealthCheckConfig)

	for _, agentCfg := range cfg.Agents {
		// 使用Agent特定的健康检查配置，如果没有则使用全局默认值
		healthCheckCfg := agentCfg.HealthCheck
		if healthCheckCfg.Interval == 0 {
			healthCheckCfg.Interval = cfg.AgentDefaults.HealthCheck.Interval
		}
		if healthCheckCfg.HeartbeatTimeout == 0 {
			healthCheckCfg.HeartbeatTimeout = cfg.AgentDefaults.HealthCheck.HeartbeatTimeout
		}
		if healthCheckCfg.CPUThreshold == 0 {
			healthCheckCfg.CPUThreshold = cfg.AgentDefaults.HealthCheck.CPUThreshold
		}
		if healthCheckCfg.MemoryThreshold == 0 {
			healthCheckCfg.MemoryThreshold = cfg.AgentDefaults.HealthCheck.MemoryThreshold
		}
		if healthCheckCfg.ThresholdDuration == 0 {
			healthCheckCfg.ThresholdDuration = cfg.AgentDefaults.HealthCheck.ThresholdDuration
		}

		agentConfigs[agentCfg.ID] = &healthCheckCfg
	}

	return &MultiHealthCheckerConfig{
		AgentConfigs: agentConfigs,
	}
}
