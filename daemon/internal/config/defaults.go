package config

import "time"

// setDaemonDefaults 设置 Daemon 默认值
func setDaemonDefaults(daemon *DaemonConfig) {
	if daemon.LogLevel == "" {
		daemon.LogLevel = "info"
	}
	if daemon.LogFile == "" {
		daemon.LogFile = "/var/log/daemon/daemon.log"
	}
	if daemon.PIDFile == "" {
		daemon.PIDFile = "/var/run/daemon.pid"
	}
	if daemon.WorkDir == "" {
		daemon.WorkDir = "/var/lib/daemon"
	}
	if daemon.GRPCPort == 0 {
		daemon.GRPCPort = 9091
	}
	// HTTPPort 默认为 0，表示不启动 HTTP 服务器（仅使用 Unix Socket）
	// 如果需要使用 HTTP 心跳接收，请在配置中显式设置 http_port
	if daemon.PprofAddress == "" {
		daemon.PprofAddress = "127.0.0.1" // 默认只监听本地，安全考虑
	}
	// PprofPort 默认为空（不启用），需要显式配置
}

// setManagerDefaults 设置 Manager 默认值
func setManagerDefaults(manager *ManagerConfig) {
	if manager.HeartbeatInterval == 0 {
		manager.HeartbeatInterval = 60 * time.Second
	}
	if manager.ReconnectInterval == 0 {
		manager.ReconnectInterval = 10 * time.Second
	}
	if manager.Timeout == 0 {
		manager.Timeout = 30 * time.Second
	}
}

// setAgentDefaults 设置 Agent 默认值（旧格式）
func setAgentDefaults(agent *AgentConfig) {
	setHealthCheckDefaults(&agent.HealthCheck)
	setRestartDefaults(&agent.Restart)
}

// setAgentDefaultsConfig 设置 Agent 全局默认值（新格式）
func setAgentDefaultsConfig(defaults *AgentDefaultsConfig) {
	setHealthCheckDefaults(&defaults.HealthCheck)
	setRestartDefaults(&defaults.Restart)
}

// setHealthCheckDefaults 设置健康检查默认值
func setHealthCheckDefaults(healthCheck *HealthCheckConfig) {
	if healthCheck.Interval == 0 {
		healthCheck.Interval = 30 * time.Second
	}
	if healthCheck.HeartbeatTimeout == 0 {
		healthCheck.HeartbeatTimeout = 90 * time.Second
	}
	if healthCheck.CPUThreshold == 0 {
		healthCheck.CPUThreshold = 50.0
	}
	if healthCheck.MemoryThreshold == 0 {
		healthCheck.MemoryThreshold = 524288000 // 500MB
	}
	if healthCheck.ThresholdDuration == 0 {
		healthCheck.ThresholdDuration = 60 * time.Second
	}
}

// setRestartDefaults 设置重启配置默认值
func setRestartDefaults(restart *RestartConfig) {
	if restart.MaxRetries == 0 {
		restart.MaxRetries = 10
	}
	if restart.BackoffBase == 0 {
		restart.BackoffBase = 10 * time.Second
	}
	if restart.BackoffMax == 0 {
		restart.BackoffMax = 60 * time.Second
	}
	if restart.Policy == "" {
		restart.Policy = "always"
	}
}

// setCollectorDefaults 设置采集器默认值
func setCollectorDefaults(collectors *CollectorConfigs) {
	if collectors.CPU.Interval == 0 {
		collectors.CPU.Interval = 60 * time.Second
	}
	if collectors.Memory.Interval == 0 {
		collectors.Memory.Interval = 60 * time.Second
	}
	if collectors.Disk.Interval == 0 {
		collectors.Disk.Interval = 60 * time.Second
	}
	if collectors.Network.Interval == 0 {
		collectors.Network.Interval = 60 * time.Second
	}
}

// setUpdateDefaults 设置更新配置默认值
func setUpdateDefaults(update *UpdateConfig) {
	if update.MaxBackups == 0 {
		update.MaxBackups = 5
	}
	if update.VerifyTimeout == 0 {
		update.VerifyTimeout = 300 * time.Second
	}
}
