package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config Agent 配置结构
type Config struct {
	AgentID   string          `mapstructure:"agent_id"`
	Version   string          `mapstructure:"version"`
	Heartbeat HeartbeatConfig `mapstructure:"heartbeat"`
	HTTP      HTTPConfig      `mapstructure:"http"`
	Log       LogConfig       `mapstructure:"log"`
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	SocketPath string        `mapstructure:"socket_path"`
	Interval   time.Duration `mapstructure:"interval"`
}

// HTTPConfig HTTP 配置
type HTTPConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	File   string `mapstructure:"file"`
	Format string `mapstructure:"format"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("agent")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 启用环境变量支持
	v.AutomaticEnv()
	v.SetEnvPrefix("AGENT")

	// 绑定关键配置项到环境变量
	v.BindEnv("agent_id", "AGENT_AGENT_ID")
	v.BindEnv("heartbeat.socket_path", "AGENT_HEARTBEAT_SOCKET_PATH")
	v.BindEnv("heartbeat.interval", "AGENT_HEARTBEAT_INTERVAL")
	v.BindEnv("http.port", "AGENT_HTTP_PORT")
	v.BindEnv("log.level", "AGENT_LOG_LEVEL")

	// 设置默认值
	v.SetDefault("version", "1.0.0")
	v.SetDefault("heartbeat.socket_path", "/tmp/daemon.sock")
	v.SetDefault("heartbeat.interval", "30s")
	v.SetDefault("http.port", 8081)
	v.SetDefault("http.host", "0.0.0.0")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 解析配置
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证必需字段
	if config.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	return &config, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}

	if c.Heartbeat.SocketPath == "" {
		return fmt.Errorf("heartbeat.socket_path is required")
	}

	if c.Heartbeat.Interval <= 0 {
		return fmt.Errorf("heartbeat.interval must be greater than 0")
	}

	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		return fmt.Errorf("http.port must be between 1 and 65535")
	}

	return nil
}
