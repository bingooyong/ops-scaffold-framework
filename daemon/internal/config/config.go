package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config Daemon配置结构
type Config struct {
	Daemon        DaemonConfig        `mapstructure:"daemon"`
	Manager       ManagerConfig       `mapstructure:"manager"`
	Agent         AgentConfig         `mapstructure:"agent"`          // 旧格式，向后兼容
	Agents        AgentsConfig        `mapstructure:"agents"`         // 新格式，多Agent配置
	AgentDefaults AgentDefaultsConfig `mapstructure:"agent_defaults"` // 全局默认配置
	Collectors    CollectorConfigs    `mapstructure:"collectors"`
	Update        UpdateConfig        `mapstructure:"update"`
}

// DaemonConfig Daemon基础配置
type DaemonConfig struct {
	ID       string `mapstructure:"id"`
	LogLevel string `mapstructure:"log_level"`
	LogFile  string `mapstructure:"log_file"`
	PIDFile  string `mapstructure:"pid_file"`
	WorkDir  string `mapstructure:"work_dir"`
	GRPCPort int    `mapstructure:"grpc_port"` // gRPC服务器端口，默认9091
}

// ManagerConfig Manager连接配置
type ManagerConfig struct {
	Address           string        `mapstructure:"address"`
	TLS               TLSConfig     `mapstructure:"tls"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	ReconnectInterval time.Duration `mapstructure:"reconnect_interval"`
	Timeout           time.Duration `mapstructure:"timeout"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
	CAFile   string `mapstructure:"ca_file"`
}

// AgentConfig Agent管理配置
type AgentConfig struct {
	BinaryPath  string            `mapstructure:"binary_path"`
	WorkDir     string            `mapstructure:"work_dir"`
	ConfigFile  string            `mapstructure:"config_file"`
	SocketPath  string            `mapstructure:"socket_path"`
	HealthCheck HealthCheckConfig `mapstructure:"health_check"`
	Restart     RestartConfig     `mapstructure:"restart"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Interval          time.Duration `mapstructure:"interval"`
	HeartbeatTimeout  time.Duration `mapstructure:"heartbeat_timeout"`
	CPUThreshold      float64       `mapstructure:"cpu_threshold"`
	MemoryThreshold   uint64        `mapstructure:"memory_threshold"`
	ThresholdDuration time.Duration `mapstructure:"threshold_duration"`
}

// RestartConfig 重启配置
type RestartConfig struct {
	MaxRetries  int           `mapstructure:"max_retries"`
	BackoffBase time.Duration `mapstructure:"backoff_base"`
	BackoffMax  time.Duration `mapstructure:"backoff_max"`
	Policy      string        `mapstructure:"policy"` // always, never, on-failure
}

// AgentsConfig 多Agent配置（新格式）
type AgentsConfig []AgentItemConfig

// AgentItemConfig 单个Agent配置项
type AgentItemConfig struct {
	ID          string            `mapstructure:"id"`
	Type        string            `mapstructure:"type"`
	Name        string            `mapstructure:"name"`
	BinaryPath  string            `mapstructure:"binary_path"`
	ConfigFile  string            `mapstructure:"config_file"`
	WorkDir     string            `mapstructure:"work_dir"`
	SocketPath  string            `mapstructure:"socket_path"`
	Enabled     bool              `mapstructure:"enabled"`
	Args        []string          `mapstructure:"args"`
	HealthCheck HealthCheckConfig `mapstructure:"health_check"`
	Restart     RestartConfig     `mapstructure:"restart"`
}

// AgentDefaultsConfig 全局Agent默认配置
type AgentDefaultsConfig struct {
	HealthCheck HealthCheckConfig `mapstructure:"health_check"`
	Restart     RestartConfig     `mapstructure:"restart"`
}

// CollectorConfigs 采集器配置
type CollectorConfigs struct {
	CPU     CollectorConfig `mapstructure:"cpu"`
	Memory  CollectorConfig `mapstructure:"memory"`
	Disk    DiskConfig      `mapstructure:"disk"`
	Network NetworkConfig   `mapstructure:"network"`
}

// CollectorConfig 通用采集器配置
type CollectorConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
}

// DiskConfig 磁盘采集器配置
type DiskConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	Interval    time.Duration `mapstructure:"interval"`
	MountPoints []string      `mapstructure:"mount_points"`
}

// NetworkConfig 网络采集器配置
type NetworkConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Interval   time.Duration `mapstructure:"interval"`
	Interfaces []string      `mapstructure:"interfaces"`
}

// UpdateConfig 更新配置
type UpdateConfig struct {
	DownloadDir   string        `mapstructure:"download_dir"`
	BackupDir     string        `mapstructure:"backup_dir"`
	MaxBackups    int           `mapstructure:"max_backups"`
	VerifyTimeout time.Duration `mapstructure:"verify_timeout"`
	PublicKeyFile string        `mapstructure:"public_key_file"`
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 启用环境变量支持
	v.AutomaticEnv()
	// 设置环境变量前缀（可选）
	// v.SetEnvPrefix("DAEMON")
	// 环境变量中的下划线映射到配置中的点号
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 绑定特定的环境变量到配置项
	v.BindEnv("manager.address", "MANAGER_ADDRESS")
	v.BindEnv("daemon.log_level", "LOG_LEVEL")
	v.BindEnv("daemon.id", "NODE_NAME")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 设置默认值
	setDefaults(config)

	// 处理向后兼容：如果存在旧格式agent配置，转换为新格式
	if err := convertLegacyAgentConfig(config); err != nil {
		return nil, fmt.Errorf("failed to convert legacy agent config: %w", err)
	}

	// 合并Agent配置（应用默认值）
	if err := mergeAgentConfigs(config); err != nil {
		return nil, fmt.Errorf("failed to merge agent configs: %w", err)
	}

	// 验证配置
	if err := validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// setDefaults 设置默认值
func setDefaults(config *Config) {
	// Daemon默认值
	if config.Daemon.LogLevel == "" {
		config.Daemon.LogLevel = "info"
	}
	if config.Daemon.LogFile == "" {
		config.Daemon.LogFile = "/var/log/daemon/daemon.log"
	}
	if config.Daemon.PIDFile == "" {
		config.Daemon.PIDFile = "/var/run/daemon.pid"
	}
	if config.Daemon.WorkDir == "" {
		config.Daemon.WorkDir = "/var/lib/daemon"
	}

	// Manager默认值
	if config.Manager.HeartbeatInterval == 0 {
		config.Manager.HeartbeatInterval = 60 * time.Second
	}
	if config.Manager.ReconnectInterval == 0 {
		config.Manager.ReconnectInterval = 10 * time.Second
	}
	if config.Manager.Timeout == 0 {
		config.Manager.Timeout = 30 * time.Second
	}

	// Agent健康检查默认值
	if config.Agent.HealthCheck.Interval == 0 {
		config.Agent.HealthCheck.Interval = 30 * time.Second
	}
	if config.Agent.HealthCheck.HeartbeatTimeout == 0 {
		config.Agent.HealthCheck.HeartbeatTimeout = 90 * time.Second
	}
	if config.Agent.HealthCheck.CPUThreshold == 0 {
		config.Agent.HealthCheck.CPUThreshold = 50.0
	}
	if config.Agent.HealthCheck.MemoryThreshold == 0 {
		config.Agent.HealthCheck.MemoryThreshold = 524288000 // 500MB
	}
	if config.Agent.HealthCheck.ThresholdDuration == 0 {
		config.Agent.HealthCheck.ThresholdDuration = 60 * time.Second
	}

	// Agent重启默认值（旧格式）
	if config.Agent.Restart.MaxRetries == 0 {
		config.Agent.Restart.MaxRetries = 10
	}
	if config.Agent.Restart.BackoffBase == 0 {
		config.Agent.Restart.BackoffBase = 10 * time.Second
	}
	if config.Agent.Restart.BackoffMax == 0 {
		config.Agent.Restart.BackoffMax = 60 * time.Second
	}
	if config.Agent.Restart.Policy == "" {
		config.Agent.Restart.Policy = "always"
	}

	// Agent默认值（新格式）
	if config.AgentDefaults.HealthCheck.Interval == 0 {
		config.AgentDefaults.HealthCheck.Interval = 30 * time.Second
	}
	if config.AgentDefaults.HealthCheck.HeartbeatTimeout == 0 {
		config.AgentDefaults.HealthCheck.HeartbeatTimeout = 90 * time.Second
	}
	if config.AgentDefaults.HealthCheck.CPUThreshold == 0 {
		config.AgentDefaults.HealthCheck.CPUThreshold = 50.0
	}
	if config.AgentDefaults.HealthCheck.MemoryThreshold == 0 {
		config.AgentDefaults.HealthCheck.MemoryThreshold = 524288000 // 500MB
	}
	if config.AgentDefaults.HealthCheck.ThresholdDuration == 0 {
		config.AgentDefaults.HealthCheck.ThresholdDuration = 60 * time.Second
	}
	if config.AgentDefaults.Restart.MaxRetries == 0 {
		config.AgentDefaults.Restart.MaxRetries = 10
	}
	if config.AgentDefaults.Restart.BackoffBase == 0 {
		config.AgentDefaults.Restart.BackoffBase = 10 * time.Second
	}
	if config.AgentDefaults.Restart.BackoffMax == 0 {
		config.AgentDefaults.Restart.BackoffMax = 60 * time.Second
	}
	if config.AgentDefaults.Restart.Policy == "" {
		config.AgentDefaults.Restart.Policy = "always"
	}

	// 采集器默认值
	if config.Collectors.CPU.Interval == 0 {
		config.Collectors.CPU.Interval = 60 * time.Second
	}
	if config.Collectors.Memory.Interval == 0 {
		config.Collectors.Memory.Interval = 60 * time.Second
	}
	if config.Collectors.Disk.Interval == 0 {
		config.Collectors.Disk.Interval = 60 * time.Second
	}
	if config.Collectors.Network.Interval == 0 {
		config.Collectors.Network.Interval = 60 * time.Second
	}

	// 更新默认值
	if config.Update.MaxBackups == 0 {
		config.Update.MaxBackups = 5
	}
	if config.Update.VerifyTimeout == 0 {
		config.Update.VerifyTimeout = 300 * time.Second
	}
}

// validate 验证配置
func validate(config *Config) error {
	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[config.Daemon.LogLevel] {
		return fmt.Errorf("invalid log level: %s", config.Daemon.LogLevel)
	}

	// 验证Manager地址（开发环境可以为空）
	if config.Manager.Address == "" {
		fmt.Println("Warning: manager.address is empty, running in standalone mode")
	}

	// 验证TLS证书文件（如果配置了才检查，文件不存在只警告）
	if config.Manager.TLS.CertFile != "" {
		if _, err := os.Stat(config.Manager.TLS.CertFile); os.IsNotExist(err) {
			fmt.Printf("Warning: TLS cert file not found: %s, TLS disabled\n", config.Manager.TLS.CertFile)
			config.Manager.TLS.CertFile = ""
		}
	}
	if config.Manager.TLS.KeyFile != "" {
		if _, err := os.Stat(config.Manager.TLS.KeyFile); os.IsNotExist(err) {
			fmt.Printf("Warning: TLS key file not found: %s, TLS disabled\n", config.Manager.TLS.KeyFile)
			config.Manager.TLS.KeyFile = ""
		}
	}
	if config.Manager.TLS.CAFile != "" {
		if _, err := os.Stat(config.Manager.TLS.CAFile); os.IsNotExist(err) {
			fmt.Printf("Warning: TLS CA file not found: %s, TLS disabled\n", config.Manager.TLS.CAFile)
			config.Manager.TLS.CAFile = ""
		}
	}

	// 验证Agent配置（开发环境可以不配置）
	if config.Agent.BinaryPath == "" {
		fmt.Println("Warning: agent.binary_path is empty, agent management disabled")
	} else {
		// 检查Agent二进制是否存在
		if _, err := os.Stat(config.Agent.BinaryPath); os.IsNotExist(err) {
			fmt.Printf("Warning: agent binary not found: %s, agent management disabled\n", config.Agent.BinaryPath)
			config.Agent.BinaryPath = ""
		}
	}

	// 验证Agents配置（新格式）
	if err := validateAgentsConfig(config); err != nil {
		return err
	}

	return nil
}

// convertLegacyAgentConfig 将旧格式agent配置转换为新格式agents数组
func convertLegacyAgentConfig(config *Config) error {
	// 如果新格式agents已配置，不进行转换
	if len(config.Agents) > 0 {
		return nil
	}

	// 如果旧格式agent未配置，不进行转换
	if config.Agent.BinaryPath == "" {
		return nil
	}

	// 转换旧格式为新格式
	fmt.Println("Warning: using legacy agent config format, converting to new format")
	agentItem := AgentItemConfig{
		ID:          "legacy-agent",
		Type:        "custom",
		Name:        "Legacy Agent",
		BinaryPath:  config.Agent.BinaryPath,
		ConfigFile:  config.Agent.ConfigFile,
		WorkDir:     config.Agent.WorkDir,
		SocketPath:  config.Agent.SocketPath,
		Enabled:     true,
		HealthCheck: config.Agent.HealthCheck,
		Restart:     config.Agent.Restart,
	}

	config.Agents = AgentsConfig{agentItem}
	return nil
}

// mergeAgentConfigs 合并Agent配置，应用全局默认值
func mergeAgentConfigs(config *Config) error {
	defaults := config.AgentDefaults

	for i := range config.Agents {
		agent := &config.Agents[i]

		// 合并健康检查配置
		if agent.HealthCheck.Interval == 0 {
			agent.HealthCheck.Interval = defaults.HealthCheck.Interval
		}
		if agent.HealthCheck.HeartbeatTimeout == 0 {
			agent.HealthCheck.HeartbeatTimeout = defaults.HealthCheck.HeartbeatTimeout
		}
		if agent.HealthCheck.CPUThreshold == 0 {
			agent.HealthCheck.CPUThreshold = defaults.HealthCheck.CPUThreshold
		}
		if agent.HealthCheck.MemoryThreshold == 0 {
			agent.HealthCheck.MemoryThreshold = defaults.HealthCheck.MemoryThreshold
		}
		if agent.HealthCheck.ThresholdDuration == 0 {
			agent.HealthCheck.ThresholdDuration = defaults.HealthCheck.ThresholdDuration
		}

		// 合并重启配置
		if agent.Restart.MaxRetries == 0 {
			agent.Restart.MaxRetries = defaults.Restart.MaxRetries
		}
		if agent.Restart.BackoffBase == 0 {
			agent.Restart.BackoffBase = defaults.Restart.BackoffBase
		}
		if agent.Restart.BackoffMax == 0 {
			agent.Restart.BackoffMax = defaults.Restart.BackoffMax
		}
		if agent.Restart.Policy == "" {
			agent.Restart.Policy = defaults.Restart.Policy
		}

		// 设置默认Name
		if agent.Name == "" {
			agent.Name = agent.Type
		}

		// 设置默认Enabled
		// enabled字段默认为true，如果未设置则保持true
	}

	return nil
}

// validateAgentsConfig 验证Agents配置
func validateAgentsConfig(config *Config) error {
	// 检查ID唯一性
	ids := make(map[string]bool)
	for i, agent := range config.Agents {
		// 验证必需字段
		if agent.ID == "" {
			return fmt.Errorf("agents[%d].id is required", i)
		}
		if agent.Type == "" {
			return fmt.Errorf("agents[%d].type is required", i)
		}
		if agent.BinaryPath == "" {
			return fmt.Errorf("agents[%d].binary_path is required", i)
		}

		// 检查ID唯一性
		if ids[agent.ID] {
			return fmt.Errorf("duplicate agent id: %s", agent.ID)
		}
		ids[agent.ID] = true

		// 验证Agent类型
		validTypes := map[string]bool{
			"filebeat":      true,
			"telegraf":      true,
			"node_exporter": true,
			"custom":        true,
		}
		if !validTypes[agent.Type] {
			return fmt.Errorf("invalid agent type: %s (valid types: filebeat, telegraf, node_exporter, custom)", agent.Type)
		}

		// 验证二进制文件路径（如果配置了）
		if agent.BinaryPath != "" {
			if _, err := os.Stat(agent.BinaryPath); os.IsNotExist(err) {
				fmt.Printf("Warning: agent binary not found: %s (agent: %s)\n", agent.BinaryPath, agent.ID)
			}
		}

		// 验证配置文件路径（如果配置了且非空）
		if agent.ConfigFile != "" {
			if _, err := os.Stat(agent.ConfigFile); os.IsNotExist(err) {
				fmt.Printf("Warning: agent config file not found: %s (agent: %s)\n", agent.ConfigFile, agent.ID)
			}
		}

		// 验证重启策略
		if agent.Restart.Policy != "" {
			validPolicies := map[string]bool{
				"always":     true,
				"never":      true,
				"on-failure": true,
			}
			if !validPolicies[agent.Restart.Policy] {
				return fmt.Errorf("invalid restart policy: %s (valid policies: always, never, on-failure)", agent.Restart.Policy)
			}
		}
	}

	return nil
}
