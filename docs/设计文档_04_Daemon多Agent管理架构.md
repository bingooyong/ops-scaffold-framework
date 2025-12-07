# Daemon 多 Agent 管理架构设计文档

## 文档信息

- **文档版本**: v1.0
- **创建日期**: 2024-12-19
- **任务引用**: Task 1.1 - 设计多 Agent 管理架构
- **状态**: 待用户审查批准

---

## 目录

1. [架构现状分析总结](#第-1-章-架构现状分析总结)
2. [AgentRegistry 和 AgentInfo 数据结构定义](#第-2-章-agentregistry-和-agentinfo-数据结构定义)
3. [daemon.yaml 配置格式示例](#第-3-章-daemonyaml-配置格式示例)
4. [重构步骤清单](#第-4-章-重构步骤清单)
5. [第三方 Agent 兼容性分析](#第-5-章-第三方-agent-兼容性分析)

---

## 第 1 章: 架构现状分析总结

### 1.1 当前架构组件

#### 1.1.1 Manager 结构体

**位置**: `daemon/internal/agent/manager.go`

**设计特点**: 单实例设计，只能管理一个 Agent 进程

**核心字段**:
- `config *config.AgentConfig`: 单个 Agent 的配置引用
- `process *os.Process`: 单个进程实例
- `pid int`: 单个进程 PID
- `restartCount int`: 单个重启计数
- `lastRestart time.Time`: 单个重启时间
- `mu sync.Mutex`: 并发安全锁

**关键方法**:
- `Start(ctx)`: 启动单个 Agent 进程
- `Stop(ctx, graceful)`: 停止单个 Agent 进程
- `Restart(ctx)`: 重启单个 Agent 进程
- `IsRunning()`: 检查单个进程状态

#### 1.1.2 AgentConfig 配置结构

**位置**: `daemon/internal/config/config.go`

**设计特点**: 单 Agent 配置结构，作为 Daemon 配置的顶层字段

**核心字段**:
- `BinaryPath string`: 单个二进制文件路径
- `WorkDir string`: 单个工作目录
- `ConfigFile string`: 单个配置文件路径
- `SocketPath string`: 单个 Socket 路径
- `HealthCheck HealthCheckConfig`: 单个健康检查配置
- `Restart RestartConfig`: 单个重启策略配置

#### 1.1.3 Daemon 集成方式

**位置**: `daemon/internal/daemon/daemon.go`

**初始化方式**:
```go
agentMgr := agent.NewManager(&cfg.Agent, logger)
```

**使用方式**:
```go
d.agentManager.Start(d.ctx)
```

**健康检查绑定**:
```go
healthChecker := agent.NewHealthChecker(&cfg.Agent.HealthCheck, agentMgr, logger)
```

### 1.2 架构局限性分析

#### 1.2.1 单实例设计限制

**问题**: Manager 结构体只能管理一个 Agent 进程实例

**原因**:
- 所有状态字段（process、pid、restartCount）都是单一值，不是集合
- 没有 Agent 注册表或索引机制
- 方法实现直接操作单个 process 对象

**影响**: 无法同时管理多个 Agent（如 filebeat、telegraf、node_exporter）

#### 1.2.2 缺少 Agent 注册机制

**问题**: 没有 Agent ID、类型标识或注册表

**原因**:
- Manager 结构体没有 Agent 标识字段（ID、Type、Name）
- 没有 AgentRegistry 或类似的注册表数据结构
- 无法区分和管理不同类型的 Agent

**影响**: 无法支持动态添加、查询、列举多个 Agent

#### 1.2.3 配置耦合在 Daemon 主配置中

**问题**: Agent 配置作为 Daemon 配置的顶层字段，不是数组结构

**原因**:
- `config.AgentConfig` 是单一结构体，不是数组
- `daemon.yaml` 中 `agent:` 段是单对象，不是数组
- 配置加载和验证逻辑只处理单个 Agent

**影响**:
- 无法在配置文件中定义多个 Agent
- 无法为不同 Agent 设置不同的配置（工作目录、健康检查策略等）

#### 1.2.4 硬编码的进程管理逻辑

**问题**: Start 方法中启动命令构建是硬编码的

**原因**:
- `args := []string{"-config", m.config.ConfigFile}` - 假设所有 Agent 都使用 `-config` 参数
- 日志文件路径硬编码: `fmt.Sprintf("%s/agent.log", m.config.WorkDir)` - 无法区分不同 Agent

**影响**: 无法适配不同 Agent 的启动参数差异（如 node_exporter 使用命令行参数而非配置文件）

#### 1.2.5 状态管理无法扩展

**问题**: 所有状态（PID、重启计数、进程对象）都是单一值

**原因**:
- 没有使用 map 或 slice 存储多个 Agent 的状态
- 没有 Agent ID 作为键来索引状态

**影响**: 无法跟踪和管理多个 Agent 的独立状态

### 1.3 需要重构的原因总结

1. **功能需求**: 需要支持管理多个第三方 Agent（filebeat、telegraf、node_exporter 等）
2. **架构扩展性**: 当前单实例设计无法扩展为多实例管理
3. **配置灵活性**: 需要为不同 Agent 配置不同的参数、工作目录、健康检查策略
4. **代码可维护性**: 硬编码逻辑需要抽象为可配置的 Agent 类型系统
5. **状态隔离**: 需要为每个 Agent 维护独立的状态（PID、重启计数、健康状态）

---

## 第 2 章: AgentRegistry 和 AgentInfo 数据结构定义

### 2.1 AgentInfo 结构体

**完整定义**:

```go
package agent

import (
	"sync"
	"time"
)

// AgentStatus Agent运行状态常量
type AgentStatus string

const (
	StatusStopped    AgentStatus = "stopped"     // Agent已停止
	StatusStarting   AgentStatus = "starting"   // Agent正在启动中
	StatusRunning    AgentStatus = "running"    // Agent正在运行
	StatusStopping   AgentStatus = "stopping"   // Agent正在停止中
	StatusRestarting AgentStatus = "restarting" // Agent正在重启中
	StatusFailed     AgentStatus = "failed"     // Agent启动失败或运行异常
)

// AgentType Agent类型常量
type AgentType string

const (
	TypeFilebeat     AgentType = "filebeat"      // Filebeat日志采集Agent
	TypeTelegraf     AgentType = "telegraf"      // Telegraf指标采集Agent
	TypeNodeExporter AgentType = "node_exporter" // Node Exporter指标采集Agent
	TypeCustom       AgentType = "custom"        // 自定义Agent类型
)

// AgentInfo Agent信息结构体
// 存储单个Agent实例的完整信息，包括配置、状态和运行时数据
type AgentInfo struct {
	// ID Agent唯一标识符，用于在注册表中索引
	// 格式建议: {type}-{name} 或 UUID，如 "filebeat-logs"、"telegraf-metrics"
	ID string

	// Type Agent类型，用于区分不同的Agent实现
	// 常见类型: filebeat、telegraf、node_exporter、custom
	Type AgentType

	// Name Agent显示名称，用于日志和UI展示
	// 示例: "Filebeat Log Collector"、"Telegraf Metrics Collector"
	Name string

	// BinaryPath Agent可执行文件的绝对路径
	// 示例: "/usr/bin/filebeat"、"/opt/telegraf/bin/telegraf"
	BinaryPath string

	// ConfigFile Agent配置文件的绝对路径
	// 示例: "/etc/filebeat/filebeat.yml"、"/etc/telegraf/telegraf.conf"
	// 注意: 某些Agent（如node_exporter）可能不使用配置文件，此字段可为空
	ConfigFile string

	// WorkDir Agent工作目录，用于存储日志、临时文件等
	// 默认值: {daemon.work_dir}/agents/{id}
	// 示例: "/var/lib/daemon/agents/filebeat-logs"
	WorkDir string

	// SocketPath Agent Unix Domain Socket路径（如果使用）
	// 用于Daemon与Agent之间的本地通信
	// 示例: "/var/run/daemon/agents/filebeat-logs.sock"
	// 注意: 某些Agent可能不使用Socket通信，此字段可为空
	SocketPath string

	// PID Agent进程ID，0表示未运行
	PID int

	// Status Agent当前运行状态
	// 可能值: stopped、starting、running、stopping、restarting、failed
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
	// 注意: 此锁仅保护AgentInfo自身字段，不保护进程对象
	mu sync.RWMutex
}
```

### 2.2 AgentRegistry 结构体

**完整定义**:

```go
// AgentRegistry Agent注册表
// 管理多个Agent实例的注册、查询和列举，提供并发安全的访问接口
type AgentRegistry struct {
	// agents 存储所有已注册的Agent实例
	// key: Agent ID (string)
	// value: AgentInfo指针
	agents map[string]*AgentInfo

	// mu 保护注册表并发访问的读写锁
	// 使用RWMutex支持并发读取，写入时独占
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

// 错误类型定义
type AgentExistsError struct {
	ID string
}

func (e *AgentExistsError) Error() string {
	return "agent already exists: " + e.ID
}

type AgentNotFoundError struct {
	ID string
}

func (e *AgentNotFoundError) Error() string {
	return "agent not found: " + e.ID
}

type AgentRunningError struct {
	ID string
}

func (e *AgentRunningError) Error() string {
	return "agent is running, cannot unregister: " + e.ID
}
```

### 2.3 并发安全保证机制

1. **注册表级别**: 使用 `sync.RWMutex` 实现读写锁
   - 读操作（Get、List、ListByType、Count、Exists）使用读锁（RLock），支持并发读取
   - 写操作（Register、Unregister）使用写锁（Lock），独占访问

2. **AgentInfo级别**: 每个AgentInfo内部使用 `sync.RWMutex`
   - 保护AgentInfo自身字段的并发访问
   - 状态更新、PID更新等操作需要持锁

3. **错误处理**: 定义了三种错误类型
   - `AgentExistsError`: Agent已存在
   - `AgentNotFoundError`: Agent不存在
   - `AgentRunningError`: Agent正在运行，无法注销

### 2.4 可扩展性设计

1. **Agent类型扩展**: 通过 `AgentType` 常量定义支持的类型
   - 内置类型: filebeat、telegraf、node_exporter
   - 自定义类型: custom（支持任意第三方Agent）

2. **动态添加新Agent类型**: 
   - 新增Agent类型只需添加新的 `AgentType` 常量
   - 注册时使用 `TypeCustom` 或新增类型常量
   - 无需修改注册表核心逻辑

3. **配置灵活性**:
   - `ConfigFile` 和 `SocketPath` 字段可为空，适配不同Agent的配置方式
   - `WorkDir` 支持自定义，也可使用默认值模板

4. **状态管理扩展**:
   - `Status` 字段使用字符串常量，易于扩展新状态
   - `RestartCount` 和 `LastRestart` 支持重启策略扩展

---

## 第 3 章: daemon.yaml 配置格式示例

### 3.1 配置结构设计

在 `daemon.yaml` 中新增顶层 `agents` 数组配置段，支持配置多个 Agent 实例。

**必需字段**:
- `id` (string): Agent 唯一标识符
- `type` (string): Agent 类型（filebeat、telegraf、node_exporter、custom）
- `binary_path` (string): Agent 可执行文件绝对路径

**可选字段**:
- `name` (string): Agent 显示名称（默认使用 type）
- `config_file` (string): Agent 配置文件路径（可为空）
- `work_dir` (string): 工作目录（默认：`{daemon.work_dir}/agents/{id}`）
- `socket_path` (string): Unix Socket 路径（可为空）
- `enabled` (bool): 是否启用（默认 true）
- `args` ([]string): 启动参数列表（可选，覆盖默认参数）
- `health_check` (object): 健康检查配置（可选，继承全局默认值）
- `restart` (object): 重启策略配置（可选，继承全局默认值）

### 3.2 完整配置示例

```yaml
# Daemon 多 Agent 管理配置文件示例

# 基础配置
daemon:
  id: ""
  log_level: info
  log_file: /var/log/daemon/daemon.log
  pid_file: /var/run/daemon.pid
  work_dir: /var/lib/daemon

# Manager连接配置
manager:
  address: "manager.example.com:9090"
  tls:
    cert_file: /etc/daemon/certs/client.crt
    key_file: /etc/daemon/certs/client.key
    ca_file: /etc/daemon/certs/ca.crt
  heartbeat_interval: 60s
  reconnect_interval: 10s
  timeout: 30s

# 多 Agent 管理配置（新设计）
agents:
  # 示例 1: Filebeat 日志采集 Agent
  - id: filebeat-logs
    type: filebeat
    name: "Filebeat Log Collector"
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat.yml
    work_dir: /var/lib/daemon/agents/filebeat-logs
    socket_path: ""
    enabled: true
    args:
      - "-c"
      - "/etc/filebeat/filebeat.yml"
      - "-path.home"
      - "/var/lib/daemon/agents/filebeat-logs"
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 50.0
      memory_threshold: 524288000
      threshold_duration: 60s
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always

  # 示例 2: Telegraf 指标采集 Agent
  - id: telegraf-metrics
    type: telegraf
    name: "Telegraf Metrics Collector"
    binary_path: /usr/bin/telegraf
    config_file: /etc/telegraf/telegraf.conf
    work_dir: /var/lib/daemon/agents/telegraf-metrics
    socket_path: ""
    enabled: true
    args:
      - "-config"
      - "/etc/telegraf/telegraf.conf"
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 40.0
      memory_threshold: 262144000
      threshold_duration: 60s
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always

  # 示例 3: Node Exporter 指标采集 Agent
  - id: node-exporter
    type: node_exporter
    name: "Node Exporter Metrics"
    binary_path: /usr/local/bin/node_exporter
    config_file: ""
    work_dir: /var/lib/daemon/agents/node-exporter
    socket_path: ""
    enabled: true
    args:
      - "--web.listen-address=:9100"
      - "--path.procfs=/proc"
      - "--path.sysfs=/sys"
      - "--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($|/)"
    health_check:
      interval: 30s
      heartbeat_timeout: 0s
      cpu_threshold: 30.0
      memory_threshold: 104857600
      threshold_duration: 60s
      http_endpoint: "http://localhost:9100/metrics"
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always

# 全局 Agent 默认配置（可选）
agent_defaults:
  health_check:
    interval: 30s
    heartbeat_timeout: 90s
    cpu_threshold: 50.0
    memory_threshold: 524288000
    threshold_duration: 60s
  restart:
    max_retries: 10
    backoff_base: 10s
    backoff_max: 60s
    policy: always

# 采集器配置（Daemon 自身的资源采集）
collectors:
  cpu:
    enabled: true
    interval: 60s
  memory:
    enabled: true
    interval: 60s
  disk:
    enabled: true
    interval: 60s
    mount_points: []
  network:
    enabled: true
    interval: 60s
    interfaces: []

# 更新配置
update:
  download_dir: /var/lib/daemon/downloads
  backup_dir: /var/lib/daemon/backups
  max_backups: 5
  verify_timeout: 300s
  public_key_file: /etc/daemon/keys/update.pub
```

### 3.3 配置项命名规范

1. **直观性**: 配置项名称清晰表达其用途（如 `binary_path`、`config_file`）
2. **一致性**: 遵循现有配置命名风格（下划线分隔，`snake_case`）
3. **可读性**: 使用完整的单词而非缩写（如 `health_check` 而非 `hc`）
4. **语义清晰**: 配置项名称与功能对应（如 `enabled`、`max_retries`）

### 3.4 向后兼容性

系统同时支持旧格式（单 Agent）和新格式（多 Agent），如果同时存在，优先使用新格式并发出警告。

---

## 第 4 章: 重构步骤清单

### 4.1 重构任务概览

以下任务按实施顺序列出，每个任务都有明确的输入、输出和验收标准。

| 任务ID | 任务名称 | 优先级 | 预估工作量 |
|--------|----------|--------|------------|
| Task 1.2 | 实现 Agent 注册与发现机制 | P0 | 2-3 天 |
| Task 1.3 | 重构 Agent Manager 为 AgentInstance | P0 | 2-3 天 |
| Task 1.4 | 创建 MultiAgentManager 管理多实例 | P0 | 3-4 天 |
| Task 1.5 | 实现 Agent 配置隔离机制 | P0 | 2-3 天 |
| Task 1.6 | 更新 HealthChecker 支持多 Agent 监控 | P1 | 2-3 天 |
| Task 1.7 | 编写架构重构测试套件 | P1 | 2-3 天 |

### 4.2 Task 1.2: 实现 Agent 注册与发现机制

**目标**: 实现 `AgentRegistry` 和 `AgentInfo` 数据结构，提供注册、查询、列举功能。

**实施内容**:
1. 创建 `daemon/internal/agent/registry.go`，实现 `AgentRegistry` 结构体
2. 实现 `AgentInfo` 结构体及其字段访问方法
3. 实现注册表核心方法：`Register()`、`Get()`、`Unregister()`、`List()`、`ListByType()`
4. 实现错误类型：`AgentExistsError`、`AgentNotFoundError`、`AgentRunningError`
5. 编写单元测试，覆盖所有注册表方法

**验收标准**:
- [ ] 注册表支持并发安全的注册和查询操作
- [ ] 所有方法通过单元测试，覆盖率 > 90%
- [ ] 错误处理正确，返回适当的错误类型

**依赖**: 无

### 4.3 Task 1.3: 重构 Agent Manager 为 AgentInstance

**目标**: 将现有的单实例 `Manager` 重构为 `AgentInstance`，支持管理单个 Agent 实例。

**实施内容**:
1. 重命名 `Manager` 为 `AgentInstance`（或创建新结构体）
2. 将 `AgentInstance` 与 `AgentInfo` 关联，使用 `AgentInfo` 存储配置和状态
3. 重构 `Start()`、`Stop()`、`Restart()` 方法，使用 `AgentInfo` 中的配置
4. 实现基于 `AgentType` 的启动参数生成逻辑（支持 filebeat、telegraf、node_exporter）
5. 更新日志输出，包含 Agent ID 和类型信息

**验收标准**:
- [ ] `AgentInstance` 可以独立管理单个 Agent 进程
- [ ] 启动参数根据 Agent 类型正确生成
- [ ] 日志文件路径包含 Agent ID，避免冲突
- [ ] 所有现有功能保持不变，向后兼容

**依赖**: Task 1.2

### 4.4 Task 1.4: 创建 MultiAgentManager 管理多实例

**目标**: 创建 `MultiAgentManager`，使用 `AgentRegistry` 管理多个 `AgentInstance`。

**实施内容**:
1. 创建 `daemon/internal/agent/multi_manager.go`，实现 `MultiAgentManager` 结构体
2. `MultiAgentManager` 内部使用 `AgentRegistry` 存储所有 Agent 实例
3. 实现批量操作：`StartAll()`、`StopAll()`、`RestartAll()`
4. 实现单个 Agent 操作：`StartAgent(id)`、`StopAgent(id)`、`RestartAgent(id)`
5. 实现从配置文件加载多个 Agent 的逻辑
6. 更新 `daemon.go`，使用 `MultiAgentManager` 替代单实例 `Manager`

**验收标准**:
- [ ] `MultiAgentManager` 可以同时管理多个 Agent 实例
- [ ] 支持从 `agents` 数组配置加载多个 Agent
- [ ] 批量操作和单个操作都正常工作
- [ ] 与现有 Daemon 启动流程集成

**依赖**: Task 1.2, Task 1.3

### 4.5 Task 1.5: 实现 Agent 配置隔离机制

**目标**: 实现配置加载、验证和隔离机制，支持为每个 Agent 设置独立配置。

**实施内容**:
1. 更新 `daemon/internal/config/config.go`，添加 `AgentsConfig` 结构体
2. 实现配置加载逻辑，支持 `agents` 数组和 `agent_defaults` 全局默认值
3. 实现配置合并逻辑：Agent 特定配置优先于全局默认配置
4. 实现配置验证：检查必需字段、文件路径、ID 唯一性
5. 实现向后兼容：如果存在旧格式 `agent:` 配置，自动转换为新格式

**验收标准**:
- [ ] 配置加载正确解析 `agents` 数组
- [ ] 配置合并逻辑正确，默认值生效
- [ ] 配置验证捕获所有无效配置
- [ ] 向后兼容性测试通过

**依赖**: Task 1.2

### 4.6 Task 1.6: 更新 HealthChecker 支持多 Agent 监控

**目标**: 更新健康检查器，支持监控多个 Agent 的健康状态。

**实施内容**:
1. 重构 `HealthChecker`，支持从 `AgentRegistry` 获取所有 Agent 实例
2. 为每个 Agent 实例创建独立的健康检查 goroutine
3. 实现基于 Agent 类型的健康检查策略（进程检查、HTTP 端点检查等）
4. 更新健康检查结果存储，使用 Agent ID 作为键
5. 实现健康检查结果聚合和告警

**验收标准**:
- [ ] 健康检查器可以监控所有已注册的 Agent
- [ ] 每个 Agent 的健康检查独立运行，互不干扰
- [ ] 支持不同 Agent 类型的健康检查策略
- [ ] 健康检查结果正确存储和聚合

**依赖**: Task 1.2, Task 1.4

### 4.7 Task 1.7: 编写架构重构测试套件

**目标**: 编写完整的测试套件，验证多 Agent 管理架构的正确性。

**实施内容**:
1. 编写 `AgentRegistry` 单元测试（注册、查询、列举、并发安全）
2. 编写 `AgentInstance` 单元测试（启动、停止、重启、状态管理）
3. 编写 `MultiAgentManager` 集成测试（多实例管理、批量操作）
4. 编写配置加载和验证测试
5. 编写端到端测试：启动多个 Agent、健康检查、重启策略

**验收标准**:
- [ ] 所有单元测试通过，覆盖率 > 85%
- [ ] 集成测试覆盖主要使用场景
- [ ] 端到端测试验证完整流程
- [ ] 并发安全测试验证无竞态条件

**依赖**: Task 1.2, Task 1.3, Task 1.4, Task 1.5, Task 1.6

---

## 第 5 章: 第三方 Agent 兼容性分析

### 5.1 Filebeat 兼容性

#### 5.1.1 配置文件格式

- **格式**: YAML
- **默认路径**: `/etc/filebeat/filebeat.yml`
- **配置特点**: 支持多输入源、输出目标、处理器配置

#### 5.1.2 启动参数

**默认启动参数**:
```bash
filebeat -c /etc/filebeat/filebeat.yml -path.home /var/lib/daemon/agents/filebeat-logs
```

**参数说明**:
- `-c`: 指定配置文件路径
- `-path.home`: 指定工作目录（用于存储 registry 文件）

#### 5.1.3 工作目录要求

- **必需**: 是
- **用途**: 存储 registry 文件（记录已采集文件的偏移量）
- **默认路径**: `/var/lib/filebeat`

#### 5.1.4 健康检查

- **方式**: 进程状态检查 + 资源使用检查
- **HTTP 端点**: 无（Filebeat 不提供 HTTP 健康检查端点）
- **心跳机制**: 无

#### 5.1.5 配置模板

```yaml
- id: filebeat-logs
  type: filebeat
  name: "Filebeat Log Collector"
  binary_path: /usr/bin/filebeat
  config_file: /etc/filebeat/filebeat.yml
  work_dir: /var/lib/daemon/agents/filebeat-logs
  args:
    - "-c"
    - "/etc/filebeat/filebeat.yml"
    - "-path.home"
    - "/var/lib/daemon/agents/filebeat-logs"
  health_check:
    interval: 30s
    cpu_threshold: 50.0
    memory_threshold: 524288000
  restart:
    policy: always
```

### 5.2 Telegraf 兼容性

#### 5.2.1 配置文件格式

- **格式**: TOML
- **默认路径**: `/etc/telegraf/telegraf.conf`
- **配置特点**: 支持插件配置、输入/输出/处理器插件

#### 5.2.2 启动参数

**默认启动参数**:
```bash
telegraf -config /etc/telegraf/telegraf.conf
```

**参数说明**:
- `-config`: 指定配置文件路径

#### 5.2.3 工作目录要求

- **必需**: 否（可选）
- **用途**: 存储状态文件（如聚合器状态）
- **默认路径**: `/var/lib/telegraf`

#### 5.2.4 健康检查

- **方式**: 进程状态检查 + 资源使用检查
- **HTTP 端点**: 可选（如果启用 `http_listener` 插件）
- **心跳机制**: 无

#### 5.2.5 配置模板

```yaml
- id: telegraf-metrics
  type: telegraf
  name: "Telegraf Metrics Collector"
  binary_path: /usr/bin/telegraf
  config_file: /etc/telegraf/telegraf.conf
  work_dir: /var/lib/daemon/agents/telegraf-metrics
  args:
    - "-config"
    - "/etc/telegraf/telegraf.conf"
  health_check:
    interval: 30s
    cpu_threshold: 40.0
    memory_threshold: 262144000
  restart:
    policy: always
```

### 5.3 Node Exporter 兼容性

#### 5.3.1 配置文件格式

- **格式**: 无（使用命令行参数）
- **配置特点**: 所有配置通过命令行参数传递

#### 5.3.2 启动参数

**默认启动参数**:
```bash
node_exporter --web.listen-address=:9100 --path.procfs=/proc --path.sysfs=/sys
```

**常用参数**:
- `--web.listen-address`: HTTP 监听地址（默认 `:9100`）
- `--path.procfs`: procfs 路径（默认 `/proc`）
- `--path.sysfs`: sysfs 路径（默认 `/sys`）
- `--collector.*`: 启用/禁用特定采集器

#### 5.3.3 工作目录要求

- **必需**: 否
- **用途**: 无特定要求

#### 5.3.4 健康检查

- **方式**: HTTP 端点检查 + 进程状态检查
- **HTTP 端点**: `http://localhost:9100/metrics`
- **心跳机制**: 无

#### 5.3.5 配置模板

```yaml
- id: node-exporter
  type: node_exporter
  name: "Node Exporter Metrics"
  binary_path: /usr/local/bin/node_exporter
  config_file: ""
  args:
    - "--web.listen-address=:9100"
    - "--path.procfs=/proc"
    - "--path.sysfs=/sys"
    - "--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($|/)"
  health_check:
    interval: 30s
    http_endpoint: "http://localhost:9100/metrics"
    cpu_threshold: 30.0
    memory_threshold: 104857600
  restart:
    policy: always
```

### 5.4 配置模板设计建议

#### 5.4.1 启动参数生成策略

**策略**: 根据 `AgentType` 自动生成默认启动参数，允许通过 `args` 字段覆盖。

**实现方式**:
```go
func generateDefaultArgs(agentType AgentType, configFile string, workDir string) []string {
	switch agentType {
	case TypeFilebeat:
		return []string{"-c", configFile, "-path.home", workDir}
	case TypeTelegraf:
		return []string{"-config", configFile}
	case TypeNodeExporter:
		return []string{"--web.listen-address=:9100", "--path.procfs=/proc", "--path.sysfs=/sys"}
	default:
		return []string{}
	}
}
```

#### 5.4.2 健康检查策略

**策略**: 根据 Agent 类型和配置选择健康检查方式。

**实现方式**:
1. **进程检查**: 所有 Agent 都支持（检查进程是否存在、资源使用情况）
2. **HTTP 端点检查**: 如果配置了 `health_check.http_endpoint`，定期请求该端点
3. **心跳检查**: 如果 Agent 支持心跳机制，检查心跳超时

#### 5.4.3 日志文件路径

**策略**: 为每个 Agent 使用独立的日志文件，避免冲突。

**路径模板**: `{work_dir}/{agent_id}.log`

**示例**:
- Filebeat: `/var/lib/daemon/agents/filebeat-logs/filebeat-logs.log`
- Telegraf: `/var/lib/daemon/agents/telegraf-metrics/telegraf-metrics.log`
- Node Exporter: `/var/lib/daemon/agents/node-exporter/node-exporter.log`

### 5.5 其他常见 Agent 支持建议

#### 5.5.1 Prometheus Node Exporter

- **类型**: `node_exporter`（已支持）
- **特点**: 无配置文件，使用命令行参数

#### 5.5.2 Logstash

- **类型**: `custom`
- **配置文件格式**: YAML
- **启动参数**: `-f /etc/logstash/conf.d/`
- **工作目录**: 必需（用于存储队列数据）

#### 5.5.3 Fluentd

- **类型**: `custom`
- **配置文件格式**: Ruby DSL
- **启动参数**: `-c /etc/fluent/fluent.conf`
- **工作目录**: 可选

#### 5.5.4 Promtail

- **类型**: `custom`
- **配置文件格式**: YAML
- **启动参数**: `-config.file=/etc/promtail/config.yml`
- **工作目录**: 可选

---

## 总结

本文档完整描述了 Daemon 多 Agent 管理架构的设计方案，包括：

1. **架构现状分析**: 深入分析了当前单 Agent 架构的局限性
2. **数据结构设计**: 设计了 `AgentRegistry` 和 `AgentInfo` 核心数据结构
3. **配置格式设计**: 设计了支持多 Agent 的配置文件格式
4. **重构步骤清单**: 列出了 Task 1.2-1.7 的详细实施计划
5. **兼容性分析**: 分析了 filebeat、telegraf、node_exporter 等常见 Agent 的兼容性

### 关键设计决策

1. **注册表模式**: 使用 `AgentRegistry` 管理多个 Agent 实例，提供并发安全的访问接口
2. **配置隔离**: 每个 Agent 拥有独立的配置、工作目录、日志文件，避免冲突
3. **类型系统**: 通过 `AgentType` 常量支持不同类型的 Agent，易于扩展
4. **向后兼容**: 同时支持旧格式（单 Agent）和新格式（多 Agent）

### 下一步行动

1. **用户审查**: 本文档需要用户审查批准
2. **实施准备**: 获得批准后，按照重构步骤清单（Task 1.2-1.7）逐步实施
3. **测试验证**: 每个任务完成后进行测试验证，确保功能正确

---

## 用户审查与批准

**此架构设计文档需要用户审查批准。请用户回复"批准"、"确认"或提供修改意见。获得批准后，方可进入 Task 1.2 实现阶段。**

**审查要点**:
- [ ] 架构设计是否满足多 Agent 管理需求
- [ ] 数据结构设计是否合理
- [ ] 配置格式是否直观易用
- [ ] 重构步骤是否完整可行
- [ ] 第三方 Agent 兼容性分析是否充分

**修改意见**: 如果用户提供修改意见，将根据意见修改文档并重新提交审查。

---

**文档状态**: 待审查  
**最后更新**: 2024-12-19
