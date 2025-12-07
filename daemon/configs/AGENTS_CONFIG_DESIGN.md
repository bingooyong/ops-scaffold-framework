# 多 Agent 配置文件扩展方案设计文档

## 概述

本文档描述 `daemon.yaml` 配置文件中 `agents` 数组配置段的设计方案，用于支持管理多个第三方 Agent（如 filebeat、telegraf、node_exporter）。

## 配置结构设计

### 顶层配置段

在 `daemon.yaml` 中新增顶层 `agents` 数组配置段：

```yaml
agents:
  - id: agent-id-1
    type: filebeat
    # ... 其他配置
  - id: agent-id-2
    type: telegraf
    # ... 其他配置
```

### Agent 配置项说明

#### 必需字段

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `id` | string | Agent 唯一标识符，用于在注册表中索引 | `"filebeat-logs"` |
| `type` | string | Agent 类型，用于区分不同的 Agent 实现 | `"filebeat"`、`"telegraf"`、`"node_exporter"`、`"custom"` |
| `binary_path` | string | Agent 可执行文件的绝对路径 | `"/usr/bin/filebeat"` |

#### 可选字段

| 字段 | 类型 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `name` | string | `{type}` | Agent 显示名称，用于日志和 UI 展示 | `"Filebeat Log Collector"` |
| `config_file` | string | `""` | Agent 配置文件路径（某些 Agent 可能不使用） | `"/etc/filebeat/filebeat.yml"` |
| `work_dir` | string | `{daemon.work_dir}/agents/{id}` | Agent 工作目录，用于存储日志、临时文件 | `"/var/lib/daemon/agents/filebeat-logs"` |
| `socket_path` | string | `""` | Agent Unix Domain Socket 路径（如果使用） | `"/var/run/daemon/agents/filebeat-logs.sock"` |
| `enabled` | bool | `true` | 是否启用此 Agent | `true` |
| `args` | []string | 根据 `type` 自动生成 | Agent 启动参数列表（覆盖默认参数） | `["-c", "/etc/filebeat/filebeat.yml"]` |
| `health_check` | object | 继承 `agent_defaults` | Agent 特定的健康检查配置 | 见下方说明 |
| `restart` | object | 继承 `agent_defaults` | Agent 特定的重启策略配置 | 见下方说明 |

### 健康检查配置（health_check）

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `interval` | duration | `30s` | 健康检查间隔 |
| `heartbeat_timeout` | duration | `90s` | 心跳超时时间（如果 Agent 支持心跳） |
| `cpu_threshold` | float | `50.0` | CPU 使用率阈值（%） |
| `memory_threshold` | uint64 | `524288000` | 内存使用阈值（字节，500MB） |
| `threshold_duration` | duration | `60s` | 阈值持续时间 |
| `http_endpoint` | string | `""` | HTTP 健康检查端点（可选，如 node_exporter 的 `/metrics`） |

### 重启策略配置（restart）

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `max_retries` | int | `10` | 最大重启次数 |
| `backoff_base` | duration | `10s` | 退避基础时间 |
| `backoff_max` | duration | `60s` | 最大退避时间 |
| `policy` | string | `"always"` | 重启策略：`always`（总是重启）、`never`（不重启）、`on-failure`（失败时重启） |

### 全局默认配置（agent_defaults）

新增 `agent_defaults` 配置段，用于定义全局默认值：

```yaml
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
```

如果 `agents` 数组中某个 Agent 未指定某些配置项，将使用 `agent_defaults` 中的默认值。

## 第三方 Agent 兼容性分析

### 1. Filebeat

**配置文件格式**: YAML  
**启动参数**: `-c {config_file} -path.home {work_dir}`  
**工作目录要求**: 需要工作目录存储 registry 文件  
**健康检查**: 通过进程状态和资源使用情况检查  
**配置示例**:
```yaml
- id: filebeat-logs
  type: filebeat
  binary_path: /usr/bin/filebeat
  config_file: /etc/filebeat/filebeat.yml
  args:
    - "-c"
    - "/etc/filebeat/filebeat.yml"
    - "-path.home"
    - "/var/lib/daemon/agents/filebeat-logs"
```

### 2. Telegraf

**配置文件格式**: TOML  
**启动参数**: `-config {config_file}`  
**工作目录要求**: 可选，用于存储状态文件  
**健康检查**: 通过进程状态和资源使用情况检查  
**配置示例**:
```yaml
- id: telegraf-metrics
  type: telegraf
  binary_path: /usr/bin/telegraf
  config_file: /etc/telegraf/telegraf.conf
  args:
    - "-config"
    - "/etc/telegraf/telegraf.conf"
```

### 3. Node Exporter

**配置文件格式**: 无（使用命令行参数）  
**启动参数**: `--web.listen-address=:9100 --path.procfs=/proc --path.sysfs=/sys`  
**工作目录要求**: 不需要特定工作目录  
**健康检查**: 通过 HTTP 端点 `/metrics` 检查（可选）  
**配置示例**:
```yaml
- id: node-exporter
  type: node_exporter
  binary_path: /usr/local/bin/node_exporter
  config_file: ""  # 不使用配置文件
  args:
    - "--web.listen-address=:9100"
    - "--path.procfs=/proc"
    - "--path.sysfs=/sys"
  health_check:
    http_endpoint: "http://localhost:9100/metrics"
```

## 配置项命名规范

### 命名原则

1. **直观性**: 配置项名称清晰表达其用途
   - `binary_path` 而非 `bin` 或 `executable`
   - `config_file` 而非 `config` 或 `conf`
   - `work_dir` 而非 `workdir` 或 `working_directory`

2. **一致性**: 遵循现有配置命名风格
   - 使用下划线分隔（`snake_case`）
   - 与 `daemon.yaml` 中其他配置段保持一致（如 `log_file`、`pid_file`）

3. **可读性**: 使用完整的单词而非缩写
   - `health_check` 而非 `health` 或 `hc`
   - `heartbeat_timeout` 而非 `hb_timeout`

4. **语义清晰**: 配置项名称与功能对应
   - `enabled` 明确表示启用/禁用状态
   - `max_retries` 明确表示最大重试次数

### 字段命名对照表

| 功能 | 配置字段 | 说明 |
|------|----------|------|
| 标识 | `id` | Agent 唯一标识符 |
| 类型 | `type` | Agent 类型 |
| 显示名称 | `name` | Agent 显示名称 |
| 可执行文件 | `binary_path` | 二进制文件路径 |
| 配置文件 | `config_file` | 配置文件路径 |
| 工作目录 | `work_dir` | 工作目录路径 |
| Socket 路径 | `socket_path` | Unix Socket 路径 |
| 启用状态 | `enabled` | 是否启用 |
| 启动参数 | `args` | 命令行参数列表 |

## 未来扩展支持

### 1. 资源限制（resource_limits）

未来可以添加资源限制配置：

```yaml
resource_limits:
  cpu_quota: "1.5"        # CPU 配额（如 Docker 的 --cpu-quota）
  memory_limit: 1073741824  # 内存限制（字节，1GB）
  cgroup_path: "/sys/fs/cgroup/daemon/agents/{id}"  # Cgroup 路径
```

### 2. 环境变量（env）

支持为每个 Agent 设置环境变量：

```yaml
env:
  - name: "LOG_LEVEL"
    value: "info"
  - name: "DATA_PATH"
    value: "/var/lib/agent/data"
```

### 3. 日志配置（logging）

支持独立的日志配置：

```yaml
logging:
  level: "info"           # 日志级别
  file: "/var/log/agents/{id}.log"  # 日志文件路径
  max_size: 10485760      # 最大文件大小（10MB）
  max_backups: 5          # 保留的备份文件数
```

### 4. 依赖关系（dependencies）

支持定义 Agent 之间的依赖关系：

```yaml
dependencies:
  - id: "filebeat-logs"    # 依赖的 Agent ID
    required: true         # 是否必需（如果必需且未运行，则此 Agent 不启动）
```

### 5. 启动顺序（start_order）

支持定义 Agent 启动顺序：

```yaml
start_order: 1            # 启动顺序（数字越小越先启动）
```

### 6. 标签和元数据（labels/metadata）

支持为 Agent 添加标签和元数据：

```yaml
labels:
  environment: "production"
  team: "ops"
metadata:
  description: "Filebeat for application logs"
  version: "7.17.0"
```

## 配置验证规则

### 必需字段验证

- `id`: 非空字符串，在 `agents` 数组中唯一
- `type`: 必须是有效的 Agent 类型（filebeat、telegraf、node_exporter、custom）
- `binary_path`: 非空字符串，文件必须存在且可执行

### 可选字段验证

- `work_dir`: 如果指定，目录必须存在或可创建
- `config_file`: 如果指定且非空，文件必须存在
- `socket_path`: 如果指定，父目录必须存在
- `args`: 必须是字符串数组

### 配置合并规则

1. Agent 特定配置优先于全局默认配置
2. 如果 `work_dir` 为空，使用默认值：`{daemon.work_dir}/agents/{id}`
3. 如果 `args` 为空，根据 `type` 自动生成默认启动参数
4. 如果 `health_check` 或 `restart` 配置项缺失，使用 `agent_defaults` 中的对应值

## 向后兼容性

### 旧配置格式支持

为了保持向后兼容，系统应同时支持：

1. **旧格式**（单 Agent）:
```yaml
agent:
  binary_path: /usr/bin/agent
  config_file: /etc/agent/agent.yaml
  # ...
```

2. **新格式**（多 Agent）:
```yaml
agents:
  - id: agent-1
    type: custom
    binary_path: /usr/bin/agent
    # ...
```

如果同时存在 `agent` 和 `agents` 配置，优先使用 `agents` 配置，并发出警告。

## 配置示例文件

完整配置示例请参考：`daemon/configs/daemon.multi-agent.example.yaml`
