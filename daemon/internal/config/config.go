package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config Daemon配置结构
type Config struct {
	Daemon     DaemonConfig     `mapstructure:"daemon"`
	Manager    ManagerConfig    `mapstructure:"manager"`
	Agent      AgentConfig      `mapstructure:"agent"`
	Collectors CollectorConfigs `mapstructure:"collectors"`
	Update     UpdateConfig     `mapstructure:"update"`
}

// DaemonConfig Daemon基础配置
type DaemonConfig struct {
	ID       string `mapstructure:"id"`
	LogLevel string `mapstructure:"log_level"`
	LogFile  string `mapstructure:"log_file"`
	PIDFile  string `mapstructure:"pid_file"`
	WorkDir  string `mapstructure:"work_dir"`
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

	// Agent重启默认值
	if config.Agent.Restart.MaxRetries == 0 {
		config.Agent.Restart.MaxRetries = 10
	}
	if config.Agent.Restart.BackoffBase == 0 {
		config.Agent.Restart.BackoffBase = 10 * time.Second
	}
	if config.Agent.Restart.BackoffMax == 0 {
		config.Agent.Restart.BackoffMax = 60 * time.Second
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

	return nil
}
