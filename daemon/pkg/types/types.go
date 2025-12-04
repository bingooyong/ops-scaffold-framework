package types

import "time"

// NodeInfo 节点信息
type NodeInfo struct {
	NodeID     string            `json:"node_id"`
	Hostname   string            `json:"hostname"`
	IP         string            `json:"ip"`
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Labels     map[string]string `json:"labels"`
	DaemonVer  string            `json:"daemon_version"`
	AgentVer   string            `json:"agent_version"`
	RegisterAt time.Time         `json:"register_at"`
}

// Metrics 指标数据结构
type Metrics struct {
	Name      string                 `json:"name"`
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"`
}

// AgentStatus Agent状态
type AgentStatus struct {
	PID         int       `json:"pid"`
	Status      string    `json:"status"` // running, stopped, unknown
	Version     string    `json:"version"`
	StartTime   time.Time `json:"start_time"`
	CPU         float64   `json:"cpu_percent"`
	Memory      uint64    `json:"memory_bytes"`
	Restarts    int       `json:"restart_count"`
	LastRestart time.Time `json:"last_restart"`
}

// Heartbeat 心跳数据
type Heartbeat struct {
	PID       int       `json:"pid"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	CPU       float64   `json:"cpu"`
	Memory    uint64    `json:"memory"`
}

// ReportData 上报数据结构
type ReportData struct {
	NodeID      string             `json:"node_id"`
	Timestamp   time.Time          `json:"timestamp"`
	Metrics     map[string]*Metrics `json:"metrics"`
	AgentStatus *AgentStatus       `json:"agent_status"`
	Logs        []LogEntry         `json:"logs,omitempty"`
}

// LogEntry 日志条目
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// UpdateRequest 更新请求
type UpdateRequest struct {
	Component   string `json:"component"`   // "agent" or "daemon"
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Hash        string `json:"hash"`      // SHA-256
	Signature   string `json:"signature"` // Base64 encoded
}

// UpdateResult 更新结果
type UpdateResult struct {
	Success    bool   `json:"success"`
	OldVersion string `json:"old_version"`
	NewVersion string `json:"new_version"`
	Error      string `json:"error,omitempty"`
	RolledBack bool   `json:"rolled_back"`
}

// HealthStatus 健康状态
type HealthStatus int

const (
	// HealthStatusHealthy 健康
	HealthStatusHealthy HealthStatus = iota
	// HealthStatusDead 进程已死亡
	HealthStatusDead
	// HealthStatusNoHeartbeat 无心跳
	HealthStatusNoHeartbeat
	// HealthStatusOverThreshold 资源超限
	HealthStatusOverThreshold
)

// String 返回健康状态的字符串表示
func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusDead:
		return "dead"
	case HealthStatusNoHeartbeat:
		return "no_heartbeat"
	case HealthStatusOverThreshold:
		return "over_threshold"
	default:
		return "unknown"
	}
}
