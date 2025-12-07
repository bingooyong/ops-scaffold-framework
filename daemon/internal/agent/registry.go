package agent

import (
	"sync"
	"time"
)

// AgentStatus Agent运行状态常量
type AgentStatus string

const (
	// StatusStopped Agent已停止
	StatusStopped AgentStatus = "stopped"
	// StatusStarting Agent正在启动中
	StatusStarting AgentStatus = "starting"
	// StatusRunning Agent正在运行
	StatusRunning AgentStatus = "running"
	// StatusStopping Agent正在停止中
	StatusStopping AgentStatus = "stopping"
	// StatusRestarting Agent正在重启中
	StatusRestarting AgentStatus = "restarting"
	// StatusFailed Agent启动失败或运行异常
	StatusFailed AgentStatus = "failed"
)

// AgentType Agent类型常量
type AgentType string

const (
	// TypeFilebeat Filebeat日志采集Agent
	TypeFilebeat AgentType = "filebeat"
	// TypeTelegraf Telegraf指标采集Agent
	TypeTelegraf AgentType = "telegraf"
	// TypeNodeExporter Node Exporter指标采集Agent
	TypeNodeExporter AgentType = "node_exporter"
	// TypeCustom 自定义Agent类型
	TypeCustom AgentType = "custom"
)

// AgentInfo Agent信息结构体
// 存储单个Agent实例的完整信息，包括配置、状态和运行时数据
type AgentInfo struct {
	// ID Agent唯一标识符，用于在注册表中索引
	ID string

	// Type Agent类型，用于区分不同的Agent实现
	Type AgentType

	// Name Agent显示名称，用于日志和UI展示
	Name string

	// BinaryPath Agent可执行文件的绝对路径
	BinaryPath string

	// ConfigFile Agent配置文件的绝对路径
	// 注意: 某些Agent（如node_exporter）可能不使用配置文件，此字段可为空
	ConfigFile string

	// WorkDir Agent工作目录，用于存储日志、临时文件等
	WorkDir string

	// SocketPath Agent Unix Domain Socket路径（如果使用）
	// 注意: 某些Agent可能不使用Socket通信，此字段可为空
	SocketPath string

	// PID Agent进程ID，0表示未运行
	PID int

	// Status Agent当前运行状态
	Status AgentStatus

	// RestartCount Agent重启次数，用于重启策略和告警
	RestartCount int

	// LastRestart 上次重启时间，用于计算重启退避时间
	LastRestart time.Time

	// CreatedAt Agent注册到注册表的时间
	CreatedAt time.Time

	// UpdatedAt Agent信息最后更新时间
	UpdatedAt time.Time

	// mu 保护AgentInfo字段的并发访问锁
	mu sync.RWMutex
}

// GetPID 获取Agent进程ID（线程安全）
func (a *AgentInfo) GetPID() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.PID
}

// SetPID 设置Agent进程ID（线程安全）
func (a *AgentInfo) SetPID(pid int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.PID = pid
	a.UpdatedAt = time.Now()
}

// GetStatus 获取Agent运行状态（线程安全）
func (a *AgentInfo) GetStatus() AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Status
}

// SetStatus 设置Agent运行状态（线程安全）
func (a *AgentInfo) SetStatus(status AgentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Status = status
	a.UpdatedAt = time.Now()
}

// GetRestartCount 获取Agent重启次数（线程安全）
func (a *AgentInfo) GetRestartCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.RestartCount
}

// IncrementRestartCount 增加Agent重启次数（线程安全）
func (a *AgentInfo) IncrementRestartCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.RestartCount++
	a.LastRestart = time.Now()
	a.UpdatedAt = time.Now()
}

// ResetRestartCount 重置Agent重启次数（线程安全）
func (a *AgentInfo) ResetRestartCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.RestartCount = 0
	a.UpdatedAt = time.Now()
}

// GetLastRestart 获取上次重启时间（线程安全）
func (a *AgentInfo) GetLastRestart() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.LastRestart
}

// UpdateTimestamp 更新最后更新时间（线程安全）
func (a *AgentInfo) UpdateTimestamp() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.UpdatedAt = time.Now()
}

// AgentRegistry Agent注册表
// 管理多个Agent实例的注册、查询和列举，提供并发安全的访问接口
type AgentRegistry struct {
	// agents 存储所有已注册的Agent实例
	agents map[string]*AgentInfo

	// mu 保护注册表并发访问的读写锁
	mu sync.RWMutex
}

// NewAgentRegistry 创建新的Agent注册表
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*AgentInfo),
	}
}

// Register 注册一个新的Agent到注册表
func (r *AgentRegistry) Register(
	id string,
	agentType AgentType,
	name string,
	binaryPath string,
	configFile string,
	workDir string,
	socketPath string,
) (*AgentInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查ID是否已存在
	if _, exists := r.agents[id]; exists {
		return nil, &AgentExistsError{ID: id}
	}

	// 创建AgentInfo
	now := time.Now()
	info := &AgentInfo{
		ID:           id,
		Type:         agentType,
		Name:         name,
		BinaryPath:   binaryPath,
		ConfigFile:   configFile,
		WorkDir:      workDir,
		SocketPath:   socketPath,
		PID:          0,
		Status:       StatusStopped,
		RestartCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 注册到map
	r.agents[id] = info

	return info, nil
}

// Get 根据ID获取Agent信息
func (r *AgentRegistry) Get(id string) *AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[id]
}

// Unregister 从注册表中移除Agent
func (r *AgentRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.agents[id]
	if !exists {
		return &AgentNotFoundError{ID: id}
	}

	// 检查Agent是否正在运行
	info.mu.RLock()
	isRunning := info.Status == StatusRunning || info.Status == StatusStarting
	info.mu.RUnlock()

	if isRunning {
		return &AgentRunningError{ID: id}
	}

	// 从map中删除
	delete(r.agents, id)
	return nil
}

// List 列举所有已注册的Agent
func (r *AgentRegistry) List() []*AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*AgentInfo, 0, len(r.agents))
	for _, info := range r.agents {
		result = append(result, info)
	}

	return result
}

// ListByType 根据类型列举Agent
func (r *AgentRegistry) ListByType(agentType AgentType) []*AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*AgentInfo, 0)
	for _, info := range r.agents {
		if info.Type == agentType {
			result = append(result, info)
		}
	}

	return result
}

// Count 返回注册表中Agent的数量
func (r *AgentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// Exists 检查指定ID的Agent是否存在
func (r *AgentRegistry) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.agents[id]
	return exists
}

// AgentExistsError Agent已存在错误
type AgentExistsError struct {
	ID string
}

func (e *AgentExistsError) Error() string {
	return "agent already exists: " + e.ID
}

// AgentNotFoundError Agent不存在错误
type AgentNotFoundError struct {
	ID string
}

func (e *AgentNotFoundError) Error() string {
	return "agent not found: " + e.ID
}

// AgentRunningError Agent正在运行错误
type AgentRunningError struct {
	ID string
}

func (e *AgentRunningError) Error() string {
	return "agent is running, cannot unregister: " + e.ID
}
