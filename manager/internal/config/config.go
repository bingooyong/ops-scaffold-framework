package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config Manager配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig HTTP服务配置
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"` // debug, release
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"` // mysql
	DSN             string        `mapstructure:"dsn"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	LogLevel        string        `mapstructure:"log_level"` // silent, error, warn, info
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// GRPCConfig gRPC服务配置
type GRPCConfig struct {
	Host string    `mapstructure:"host"`
	Port int       `mapstructure:"port"`
	TLS  TLSConfig `mapstructure:"tls"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
	CAFile   string `mapstructure:"ca_file"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string        `mapstructure:"secret"`
	ExpireTime time.Duration `mapstructure:"expire_time"`
	Issuer     string        `mapstructure:"issuer"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"` // debug, info, warn, error
	OutputPath string `mapstructure:"output_path"`
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件数
	MaxAge     int    `mapstructure:"max_age"`     // 天
	Compress   bool   `mapstructure:"compress"`
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
	// Server默认值
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Mode == "" {
		config.Server.Mode = "release"
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}

	// Database默认值
	if config.Database.Driver == "" {
		config.Database.Driver = "mysql"
	}
	if config.Database.MaxIdleConns == 0 {
		config.Database.MaxIdleConns = 10
	}
	if config.Database.MaxOpenConns == 0 {
		config.Database.MaxOpenConns = 100
	}
	if config.Database.ConnMaxLifetime == 0 {
		config.Database.ConnMaxLifetime = time.Hour
	}
	if config.Database.LogLevel == "" {
		config.Database.LogLevel = "warn"
	}

	// Redis默认值
	if config.Redis.Host == "" {
		config.Redis.Host = "localhost"
	}
	if config.Redis.Port == 0 {
		config.Redis.Port = 6379
	}
	if config.Redis.PoolSize == 0 {
		config.Redis.PoolSize = 10
	}

	// GRPC默认值
	if config.GRPC.Host == "" {
		config.GRPC.Host = "0.0.0.0"
	}
	if config.GRPC.Port == 0 {
		config.GRPC.Port = 9090
	}

	// JWT默认值
	if config.JWT.ExpireTime == 0 {
		config.JWT.ExpireTime = 24 * time.Hour
	}
	if config.JWT.Issuer == "" {
		config.JWT.Issuer = "ops-manager"
	}

	// Log默认值
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}
	if config.Log.OutputPath == "" {
		config.Log.OutputPath = "logs/manager.log"
	}
	if config.Log.MaxSize == 0 {
		config.Log.MaxSize = 100
	}
	if config.Log.MaxBackups == 0 {
		config.Log.MaxBackups = 10
	}
	if config.Log.MaxAge == 0 {
		config.Log.MaxAge = 30
	}
}

// validate 验证配置
func validate(config *Config) error {
	// 验证服务模式
	validModes := map[string]bool{
		"debug":   true,
		"release": true,
	}
	if !validModes[config.Server.Mode] {
		return fmt.Errorf("invalid server mode: %s", config.Server.Mode)
	}

	// 验证数据库DSN
	if config.Database.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	// 验证JWT密钥
	if config.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[config.Log.Level] {
		return fmt.Errorf("invalid log level: %s", config.Log.Level)
	}

	// 验证TLS配置
	if config.GRPC.TLS.Enabled {
		if config.GRPC.TLS.CertFile == "" || config.GRPC.TLS.KeyFile == "" {
			return fmt.Errorf("TLS cert_file and key_file are required when TLS is enabled")
		}
	}

	return nil
}

// Address 返回HTTP服务监听地址
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Address 返回gRPC服务监听地址
func (c *GRPCConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Address 返回Redis连接地址
func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
