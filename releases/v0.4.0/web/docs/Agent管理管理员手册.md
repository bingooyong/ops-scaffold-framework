# Agent 管理管理员手册

**版本**: v1.0  
**更新日期**: 2025-01-27  
**适用版本**: Ops Scaffold Framework v0.3.0+

---

## 目录

1. [Daemon 配置说明](#1-daemon-配置说明)
2. [Agent 注册流程](#2-agent-注册流程)
3. [Agent 生命周期管理](#3-agent-生命周期管理)
4. [监控和告警](#4-监控和告警)
5. [故障排查指南](#5-故障排查指南)
6. [维护操作](#6-维护操作)

---

## 1. Daemon 配置说明

### 1.1 配置文件结构

Daemon 配置文件采用 YAML 格式，默认位置：
- **生产环境**: `/etc/daemon/daemon.yaml`
- **开发环境**: `daemon/configs/daemon.yaml`

**完整配置结构**：

```yaml
# 基础配置
daemon:
  id: ""  # 留空则自动生成 UUID
  log_level: info  # debug, info, warn, error
  log_file: /var/log/daemon/daemon.log
  pid_file: /var/run/daemon.pid
  work_dir: /var/lib/daemon

# Manager 连接配置
manager:
  address: "manager.example.com:9090"
  tls:
    cert_file: /etc/daemon/certs/client.crt
    key_file: /etc/daemon/certs/client.key
    ca_file: /etc/daemon/certs/ca.crt
  heartbeat_interval: 60s
  reconnect_interval: 10s
  timeout: 30s

# 多 Agent 管理配置
agents:
  - id: filebeat-logs
    type: filebeat
    # ... Agent 配置
  - id: telegraf-metrics
    type: telegraf
    # ... Agent 配置

# 全局 Agent 默认配置
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

### 1.2 agents 数组配置格式

`agents` 是一个数组，每个元素表示一个要管理的 Agent 实例。

#### 1.2.1 必需字段

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `id` | string | Agent 唯一标识符 | `"filebeat-logs"` |
| `type` | string | Agent 类型 | `"filebeat"`, `"telegraf"`, `"node_exporter"`, `"custom"` |
| `binary_path` | string | Agent 可执行文件绝对路径 | `"/usr/bin/filebeat"` |

#### 1.2.2 可选字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `name` | string | `type` | Agent 显示名称 |
| `config_file` | string | `""` | Agent 配置文件路径（某些 Agent 如 node_exporter 不使用配置文件） |
| `work_dir` | string | `{daemon.work_dir}/agents/{id}` | Agent 工作目录 |
| `socket_path` | string | `""` | Unix Socket 路径（如果使用） |
| `enabled` | bool | `true` | 是否启用此 Agent |
| `args` | []string | 根据类型自动生成 | 启动参数列表（覆盖默认参数） |
| `health_check` | object | 继承全局默认值 | 健康检查配置 |
| `restart` | object | 继承全局默认值 | 重启策略配置 |

### 1.3 配置字段详细说明

#### 1.3.1 Agent 基本信息

**id（必需）**
- Agent 唯一标识符，用于在注册表中索引
- 格式建议：`{type}-{purpose}`，如 `filebeat-logs`、`telegraf-metrics`
- 不能包含空格和特殊字符
- 在同一 Daemon 实例中必须唯一

**type（必需）**
- Agent 类型，用于区分不同的 Agent 实现
- 可选值：`filebeat`、`telegraf`、`node_exporter`、`custom`
- 系统根据类型自动生成默认启动参数

**name（可选）**
- Agent 显示名称，用于日志和 UI 展示
- 默认值：使用 `type` 值
- 示例：`"Filebeat Log Collector"`、`"Telegraf Metrics Collector"`

**binary_path（必需）**
- Agent 可执行文件的绝对路径
- 必须确保文件存在且具有执行权限
- 示例：`/usr/bin/filebeat`、`/usr/local/bin/node_exporter`

#### 1.3.2 配置文件和工作目录

**config_file（可选）**
- Agent 配置文件的绝对路径
- 某些 Agent（如 node_exporter）不使用配置文件，此字段可为空
- 示例：`/etc/filebeat/filebeat.yml`、`/etc/telegraf/telegraf.conf`

**work_dir（可选）**
- Agent 工作目录，用于存储日志、临时文件等
- 默认值：`{daemon.work_dir}/agents/{id}`
- 示例：`/var/lib/daemon/agents/filebeat-logs`
- 系统会自动创建目录（如果不存在）

**socket_path（可选）**
- Agent Unix Domain Socket 路径（如果使用）
- 用于 Daemon 与 Agent 之间的本地通信
- 示例：`/var/run/daemon/agents/filebeat-logs.sock`
- 如果 Agent 不使用 Socket 通信，此字段可为空

#### 1.3.3 启动参数

**args（可选）**
- Agent 启动参数列表
- 如果未指定，系统根据 Agent 类型自动生成默认参数
- 可以覆盖默认参数

**默认启动参数**：

| Agent 类型 | 默认参数 |
|-----------|----------|
| `filebeat` | `["-c", "{config_file}", "-path.home", "{work_dir}"]` |
| `telegraf` | `["-config", "{config_file}"]` |
| `node_exporter` | `["--web.listen-address=:9100", "--path.procfs=/proc", "--path.sysfs=/sys"]` |
| `custom` | `[]`（空数组，需要手动指定） |

**示例**：
```yaml
args:
  - "-c"
  - "/etc/filebeat/filebeat.yml"
  - "-path.home"
  - "/var/lib/daemon/agents/filebeat-logs"
```

#### 1.3.4 健康检查配置

**health_check（可选）**
- 健康检查配置，如果未指定，继承 `agent_defaults.health_check`

**配置项**：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `interval` | duration | `30s` | 健康检查间隔 |
| `heartbeat_timeout` | duration | `90s` | 心跳超时时间（0 表示不使用心跳检查） |
| `cpu_threshold` | float | `50.0` | CPU 使用率阈值（%） |
| `memory_threshold` | int | `524288000` | 内存使用阈值（字节） |
| `threshold_duration` | duration | `60s` | 超过阈值持续时间 |
| `http_endpoint` | string | `""` | HTTP 健康检查端点（可选） |

**示例**：
```yaml
health_check:
  interval: 30s
  heartbeat_timeout: 90s
  cpu_threshold: 50.0
  memory_threshold: 524288000
  threshold_duration: 60s
  http_endpoint: "http://localhost:9100/metrics"  # 仅 node_exporter 使用
```

#### 1.3.5 重启策略配置

**restart（可选）**
- 重启策略配置，如果未指定，继承 `agent_defaults.restart`

**配置项**：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `max_retries` | int | `10` | 最大重启次数（超过后不再自动重启） |
| `backoff_base` | duration | `10s` | 退避基础时间 |
| `backoff_max` | duration | `60s` | 最大退避时间 |
| `policy` | string | `always` | 重启策略：`always`（总是重启）、`never`（不重启）、`on-failure`（失败时重启） |

**退避策略**：
- 第 1 次重启：立即执行
- 第 2-3 次重启：等待 `backoff_base`（10秒）
- 第 4-5 次重启：等待 `backoff_base * 2`（20秒）
- 第 6-10 次重启：等待 `backoff_max`（60秒）
- 超过 `max_retries`：不再自动重启，上报告警

**示例**：
```yaml
restart:
  max_retries: 10
  backoff_base: 10s
  backoff_max: 60s
  policy: always
```

### 1.4 配置示例

#### 1.4.1 Filebeat 配置示例

```yaml
agents:
  - id: filebeat-logs
    type: filebeat
    name: "Filebeat Log Collector"
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat.yml
    work_dir: /var/lib/daemon/agents/filebeat-logs
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
```

#### 1.4.2 Telegraf 配置示例

```yaml
agents:
  - id: telegraf-metrics
    type: telegraf
    name: "Telegraf Metrics Collector"
    binary_path: /usr/bin/telegraf
    config_file: /etc/telegraf/telegraf.conf
    work_dir: /var/lib/daemon/agents/telegraf-metrics
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
```

#### 1.4.3 Node Exporter 配置示例

```yaml
agents:
  - id: node-exporter
    type: node_exporter
    name: "Node Exporter Metrics"
    binary_path: /usr/local/bin/node_exporter
    config_file: ""  # Node Exporter 不使用配置文件
    work_dir: /var/lib/daemon/agents/node-exporter
    enabled: true
    args:
      - "--web.listen-address=:9100"
      - "--path.procfs=/proc"
      - "--path.sysfs=/sys"
      - "--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($|/)"
    health_check:
      interval: 30s
      heartbeat_timeout: 0s  # 不使用心跳检查
      cpu_threshold: 30.0
      memory_threshold: 104857600
      threshold_duration: 60s
      http_endpoint: "http://localhost:9100/metrics"  # HTTP 健康检查
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

#### 1.4.4 自定义 Agent 配置示例

```yaml
agents:
  - id: custom-agent
    type: custom
    name: "Custom Application Agent"
    binary_path: /usr/local/bin/custom-agent
    config_file: /etc/custom-agent/config.json
    work_dir: /var/lib/daemon/agents/custom-agent
    socket_path: /var/run/custom-agent.sock
    enabled: true
    args:
      - "--config"
      - "/etc/custom-agent/config.json"
      - "--log-level"
      - "info"
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

---

## 2. Agent 注册流程

### 2.1 自动注册流程

Agent 注册通常在 Daemon 启动时自动完成：

1. **Daemon 启动**
   ```bash
   sudo systemctl start daemon
   ```

2. **读取配置文件**
   - Daemon 读取 `daemon.yaml` 配置文件
   - 解析 `agents` 数组配置

3. **创建 AgentInfo**
   - 为每个 Agent 配置创建 `AgentInfo` 对象
   - 注册到 `AgentRegistry`

4. **创建 AgentInstance**
   - 为每个 Agent 创建 `AgentInstance` 管理器
   - 添加到 `MultiAgentManager`

5. **同步到 Manager**
   - 通过 gRPC 将 Agent 信息同步到 Manager
   - Manager 存储到数据库（`agents` 表）

6. **自动启动（可选）**
   - 如果 Agent 配置中 `enabled: true`，Daemon 会自动启动 Agent
   - 如果 `enabled: false`，Agent 仅注册但不启动

### 2.2 注册验证

**验证 Agent 是否注册成功**：

1. **查看 Daemon 日志**
   ```bash
   tail -f /var/log/daemon/daemon.log | grep "agent registered"
   ```

   预期输出：
   ```
   2025-01-27T10:00:00Z [INFO] agent registered agent_id=filebeat-logs agent_type=filebeat
   ```

2. **通过 API 查询**
   ```bash
   curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents" \
     -H "Authorization: Bearer <token>"
   ```

3. **检查注册表**
   ```bash
   # 如果 Daemon 提供 CLI 工具
   daemon agent list
   ```

### 2.3 注册错误处理

**常见错误**：

| 错误 | 原因 | 解决方法 |
|------|------|----------|
| `agent already exists` | Agent ID 重复 | 修改 Agent ID 或删除旧配置 |
| `binary_path not found` | 二进制文件不存在 | 安装 Agent 或检查路径配置 |
| `config_file not found` | 配置文件不存在 | 创建配置文件或检查路径配置 |
| `work_dir permission denied` | 工作目录权限不足 | 修改目录权限：`chmod 755 <work_dir>` |

**查看错误日志**：
```bash
tail -100 /var/log/daemon/daemon.log | grep -i "error\|failed"
```

---

## 3. Agent 生命周期管理

### 3.1 启动 Agent

#### 3.1.1 启动步骤

1. **检查前置条件**
   - 二进制文件存在且可执行
   - 配置文件存在（如果使用）
   - 工作目录存在且有写权限

2. **生成启动命令**
   - 根据 Agent 类型和配置生成启动命令
   - 格式：`{binary_path} {args...}`

3. **启动进程**
   - 使用 `os/exec` 启动 Agent 进程
   - 设置工作目录和环境变量
   - 重定向标准输出和错误到日志文件

4. **更新状态**
   - 记录进程 PID
   - 更新状态为 `running`
   - 更新启动时间

5. **健康检查**
   - 等待几秒钟后执行健康检查
   - 如果健康检查失败，标记为 `failed`

#### 3.1.2 启动方式

**方式 1：通过 Web 界面启动**
1. 登录 Web 管理界面
2. 进入节点详情页 → Agent 标签
3. 找到目标 Agent，点击"启动"按钮

**方式 2：通过 API 启动**
```bash
curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id/operate" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"operation": "start"}'
```

**方式 3：通过 Daemon 配置自动启动**
- 在 `daemon.yaml` 中设置 `enabled: true`
- 重启 Daemon 服务

#### 3.1.3 启动注意事项

1. **资源检查**
   - 确保系统有足够的 CPU 和内存资源
   - 检查端口是否被占用（如果 Agent 需要监听端口）

2. **配置验证**
   - 某些 Agent 支持配置验证命令，建议先验证
   - Filebeat: `filebeat test config -c /etc/filebeat/filebeat.yml`
   - Telegraf: `telegraf --config /etc/telegraf/telegraf.conf --test`

3. **日志监控**
   - 启动后立即查看日志，确认启动成功
   - 监控前几分钟的运行情况

### 3.2 停止 Agent

#### 3.2.1 停止步骤

1. **发送停止信号**
   - **优雅停止**：发送 `SIGTERM` 信号，等待进程自行退出
   - **强制停止**：如果优雅停止失败，发送 `SIGKILL` 信号

2. **等待进程退出**
   - 优雅停止：等待最多 30 秒
   - 强制停止：立即 kill 进程

3. **更新状态**
   - 清除进程 PID（设为 0）
   - 更新状态为 `stopped`
   - 更新停止时间

#### 3.2.2 停止方式

**方式 1：通过 Web 界面停止**
1. 在 Agent 列表中找到目标 Agent
2. 点击"停止"按钮
3. 选择停止方式（优雅停止/强制停止）

**方式 2：通过 API 停止**
```bash
curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id/operate" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"operation": "stop"}'
```

**方式 3：通过 Daemon 配置禁用**
- 在 `daemon.yaml` 中设置 `enabled: false`
- 重启 Daemon 服务（会停止 Agent）

#### 3.2.3 停止注意事项

1. **数据保存**
   - 某些 Agent（如 Filebeat）需要保存状态数据
   - 优雅停止可以确保数据正确保存

2. **正在执行的任务**
   - 如果 Agent 正在执行任务，优雅停止会等待任务完成
   - 强制停止可能导致任务中断

3. **依赖服务**
   - 如果其他服务依赖此 Agent，停止前需要通知

### 3.3 重启 Agent

#### 3.3.1 重启步骤

1. **停止 Agent**
   - 执行停止流程（优雅停止）

2. **等待退避时间**（如果是自动重启）
   - 根据重启次数计算退避时间
   - 手动重启跳过退避时间

3. **启动 Agent**
   - 执行启动流程

4. **更新状态**
   - 增加重启计数
   - 更新最后重启时间

#### 3.2.2 重启方式

**方式 1：通过 Web 界面重启**
1. 在 Agent 列表中找到目标 Agent
2. 点击"重启"按钮

**方式 2：通过 API 重启**
```bash
curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id/operate" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"operation": "restart"}'
```

**方式 3：自动重启**（异常恢复）
- 当 Agent 异常退出时，系统自动重启
- 遵循重启策略配置（退避时间、最大重试次数）

#### 3.3.3 重启策略

**重启策略类型**：

| 策略 | 说明 | 适用场景 |
|------|------|----------|
| `always` | 总是重启（无论退出原因） | 生产环境，确保服务持续运行 |
| `never` | 不自动重启 | 开发测试环境，手动控制 |
| `on-failure` | 仅在失败时重启 | 区分正常停止和异常退出 |

**退避策略**：
- 避免频繁重启导致系统负载过高
- 重启次数越多，等待时间越长
- 超过最大重试次数后不再自动重启，上报告警

---

## 4. 监控和告警

### 4.1 健康检查机制

#### 4.1.1 检查方式

**1. 进程检查**
- 检查 Agent 进程是否存在
- 检查进程状态（运行中/僵尸进程）

**2. HTTP 端点检查**（如果 Agent 提供）
- 定期请求 Agent 的 HTTP 健康检查端点
- 检查响应状态码和内容
- 示例：Node Exporter 的 `/metrics` 端点

**3. 心跳检查**（如果 Agent 支持）
- 检查 Agent 心跳是否超时
- 心跳超时时间：`heartbeat_timeout`（默认 90 秒）

**4. 资源检查**
- 检查 CPU 使用率是否超过阈值
- 检查内存使用是否超过阈值
- 超过阈值持续时间：`threshold_duration`（默认 60 秒）

#### 4.1.2 健康检查配置

```yaml
health_check:
  interval: 30s                    # 检查间隔
  heartbeat_timeout: 90s           # 心跳超时（0 表示不使用）
  cpu_threshold: 50.0              # CPU 阈值（%）
  memory_threshold: 524288000      # 内存阈值（字节）
  threshold_duration: 60s          # 阈值持续时间
  http_endpoint: ""                 # HTTP 端点（可选）
```

#### 4.1.3 健康状态

| 状态 | 说明 | 触发条件 |
|------|------|----------|
| `healthy` | 健康 | 所有检查通过 |
| `unhealthy` | 不健康 | 进程不存在、心跳超时、资源超限 |
| `degraded` | 降级 | 部分检查失败但未达到不健康阈值 |

### 4.2 自动重启策略配置

#### 4.2.1 重启策略配置

```yaml
restart:
  max_retries: 10        # 最大重启次数
  backoff_base: 10s      # 退避基础时间
  backoff_max: 60s       # 最大退避时间
  policy: always         # 重启策略
```

#### 4.2.2 退避时间计算

| 重启次数 | 等待时间 | 说明 |
|---------|---------|------|
| 1 | 立即 | 第一次重启立即执行 |
| 2-3 | `backoff_base` (10s) | 基础退避时间 |
| 4-5 | `backoff_base * 2` (20s) | 双倍退避时间 |
| 6-10 | `backoff_max` (60s) | 最大退避时间 |
| > 10 | 不再重启 | 超过最大重试次数，上报告警 |

### 4.3 资源监控阈值设置

#### 4.3.1 CPU 使用率阈值

```yaml
health_check:
  cpu_threshold: 50.0              # CPU 使用率阈值（%）
  threshold_duration: 60s          # 超过阈值持续时间
```

**建议值**：
- Filebeat: 50%
- Telegraf: 40%
- Node Exporter: 30%
- Custom Agent: 根据实际情况设置

#### 4.3.2 内存使用阈值

```yaml
health_check:
  memory_threshold: 524288000      # 内存使用阈值（字节，500MB）
  threshold_duration: 60s
```

**建议值**：
- Filebeat: 500MB (524288000 字节)
- Telegraf: 250MB (262144000 字节)
- Node Exporter: 100MB (104857600 字节)
- Custom Agent: 根据实际情况设置

#### 4.3.3 阈值持续时间

- 避免短暂峰值触发告警
- 建议设置：60 秒
- 只有在持续超过阈值时才触发告警和自动重启

### 4.4 告警配置建议

#### 4.4.1 告警规则

在 Manager 或监控系统中配置以下告警规则：

1. **Agent 状态异常**
   - 条件：Agent 状态 = `error` 或 `failed`
   - 级别：严重（Critical）
   - 动作：立即通知管理员

2. **心跳超时**
   - 条件：最后心跳时间 > 90 秒
   - 级别：警告（Warning）
   - 动作：通知管理员，自动重启

3. **资源使用过高**
   - 条件：CPU > 80% 或内存 > 90% 持续 5 分钟
   - 级别：警告（Warning）
   - 动作：通知管理员，考虑扩容或优化

4. **频繁重启**
   - 条件：1 小时内重启次数 > 10 次
   - 级别：严重（Critical）
   - 动作：立即通知管理员，停止自动重启

5. **Agent 停止**
   - 条件：Agent 状态 = `stopped` 且 `enabled = true`
   - 级别：警告（Warning）
   - 动作：通知管理员，检查原因

#### 4.4.2 告警通知方式

- **邮件通知**：发送到管理员邮箱
- **短信通知**：严重告警发送短信
- **企业微信/钉钉**：集成企业 IM 工具
- **PagerDuty/OpsGenie**：集成专业告警平台

---

## 5. 故障排查指南

### 5.1 常见问题诊断步骤

#### 5.1.1 Agent 无法启动

**步骤 1：检查二进制文件**
```bash
# 检查文件是否存在
ls -l /usr/bin/filebeat

# 检查文件权限
ls -l /usr/bin/filebeat | grep -E "^-r.x"

# 检查文件类型
file /usr/bin/filebeat
```

**步骤 2：检查配置文件**
```bash
# 检查配置文件是否存在
ls -l /etc/filebeat/filebeat.yml

# 检查配置文件语法（如果支持）
/usr/bin/filebeat test config -c /etc/filebeat/filebeat.yml
```

**步骤 3：检查工作目录**
```bash
# 检查目录是否存在
ls -ld /var/lib/daemon/agents/filebeat-logs

# 检查目录权限
ls -ld /var/lib/daemon/agents/filebeat-logs | grep -E "^drwx"

# 检查磁盘空间
df -h /var/lib/daemon
```

**步骤 4：查看日志**
```bash
# Daemon 日志
tail -100 /var/log/daemon/daemon.log | grep -i "filebeat"

# Agent 日志
tail -100 /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log
```

**步骤 5：手动测试启动**
```bash
# 使用相同参数手动启动，查看错误信息
/usr/bin/filebeat -c /etc/filebeat/filebeat.yml \
  -path.home /var/lib/daemon/agents/filebeat-logs
```

#### 5.1.2 Agent 频繁重启

**步骤 1：查看重启历史**
```bash
# 通过 API 查询 Agent 状态
curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id" \
  -H "Authorization: Bearer <token>" | jq '.data.agent.restart_count'
```

**步骤 2：查看日志**
```bash
# 查看 Agent 日志，查找错误信息
tail -200 /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log | grep -i "error\|fatal\|panic"
```

**步骤 3：检查资源使用**
```bash
# 查看系统资源
top
htop

# 查看 Agent 进程资源使用
ps aux | grep filebeat
```

**步骤 4：检查配置文件**
```bash
# 检查配置文件是否有语法错误
/usr/bin/filebeat test config -c /etc/filebeat/filebeat.yml
```

**步骤 5：临时禁用自动重启**
```yaml
# 修改 daemon.yaml，设置重启策略为 never
restart:
  policy: never
```

#### 5.1.3 Agent 状态不同步

**步骤 1：检查 Daemon 与 Manager 连接**
```bash
# 查看 Daemon 日志
tail -f /var/log/daemon/daemon.log | grep -i "grpc\|sync\|manager"
```

**步骤 2：检查网络连接**
```bash
# 测试 gRPC 连接
telnet manager.example.com 9090

# 检查防火墙规则
iptables -L -n | grep 9090
```

**步骤 3：手动触发同步**
```bash
# 通过 API 查询，会触发同步
curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents" \
  -H "Authorization: Bearer <token>"
```

**步骤 4：重启 Daemon**
```bash
sudo systemctl restart daemon
```

### 5.2 日志文件位置和查看方法

#### 5.2.1 日志文件位置

| 日志类型 | 位置 | 说明 |
|---------|------|------|
| Daemon 日志 | `/var/log/daemon/daemon.log` | Daemon 主日志 |
| Agent 日志 | `/var/lib/daemon/agents/{agent_id}/{agent_id}.log` | Agent 运行日志 |
| Manager 日志 | `manager/logs/manager.log` | Manager 服务日志 |

#### 5.2.2 日志查看方法

**方法 1：直接查看日志文件**
```bash
# 查看 Daemon 日志
tail -f /var/log/daemon/daemon.log

# 查看 Agent 日志
tail -f /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log
```

**方法 2：通过 Web 界面查看**
1. 登录 Web 管理界面
2. 进入节点详情页 → Agent 标签
3. 点击"查看日志"按钮

**方法 3：通过 API 查看**
```bash
curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id/logs?lines=100" \
  -H "Authorization: Bearer <token>"
```

#### 5.2.3 日志分析技巧

**查找错误信息**：
```bash
grep -i "error\|fatal\|panic" /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log
```

**查找特定时间段的日志**：
```bash
# 使用 journalctl（如果使用 systemd）
journalctl -u daemon --since "2025-01-27 10:00:00" --until "2025-01-27 11:00:00"
```

**统计错误次数**：
```bash
grep -c "error" /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log
```

### 5.3 性能问题排查方法

#### 5.3.1 CPU 使用率过高

**排查步骤**：

1. **查看 Agent 进程 CPU 使用**
   ```bash
   top -p $(pgrep -f filebeat)
   ```

2. **查看系统整体 CPU 使用**
   ```bash
   top
   ```

3. **分析 Agent 配置**
   - 检查采集频率是否过高
   - 检查并发数设置
   - 检查处理器配置

4. **优化建议**
   - 降低采集频率
   - 减少并发数
   - 优化处理器规则

#### 5.3.2 内存使用过高

**排查步骤**：

1. **查看 Agent 进程内存使用**
   ```bash
   ps aux | grep filebeat | awk '{print $6/1024 " MB"}'
   ```

2. **查看系统内存使用**
   ```bash
   free -h
   ```

3. **分析内存泄漏**
   - 查看内存使用趋势
   - 检查是否有内存泄漏

4. **优化建议**
   - 调整缓冲区大小
   - 减少批处理大小
   - 重启 Agent（临时解决）

#### 5.3.3 磁盘 I/O 过高

**排查步骤**：

1. **查看磁盘 I/O**
   ```bash
   iostat -x 1
   ```

2. **查看 Agent 日志写入**
   ```bash
   lsof -p $(pgrep -f filebeat) | grep log
   ```

3. **优化建议**
   - 减少日志级别
   - 启用日志轮转
   - 使用更快的存储

### 5.4 联系支持的方式

如果问题无法自行解决，请按以下方式联系技术支持：

1. **收集信息**
   - Agent 配置（`daemon.yaml` 相关部分）
   - 日志文件（Daemon 日志和 Agent 日志）
   - 系统信息（OS 版本、资源使用情况）
   - 错误截图或错误信息

2. **提交问题**
   - 通过工单系统提交
   - 或发送邮件到技术支持邮箱
   - 或通过企业 IM 联系技术支持

3. **问题描述**
   - 问题现象
   - 复现步骤
   - 预期行为
   - 实际行为
   - 已尝试的解决方法

---

## 6. 维护操作

### 6.1 Agent 配置更新流程

#### 6.1.1 更新 Agent 配置文件

**步骤 1：备份当前配置**
```bash
cp /etc/filebeat/filebeat.yml /etc/filebeat/filebeat.yml.bak.$(date +%Y%m%d)
```

**步骤 2：更新配置文件**
```bash
sudo vim /etc/filebeat/filebeat.yml
```

**步骤 3：验证配置**
```bash
/usr/bin/filebeat test config -c /etc/filebeat/filebeat.yml
```

**步骤 4：重载配置**（如果 Agent 支持）
```bash
# 通过 API 触发重载（如果支持）
curl -X POST "http://localhost:5060/reload" \
  -H "Authorization: Bearer <token>"
```

**步骤 5：重启 Agent**（如果重载不支持）
```bash
# 通过 API 重启
curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/filebeat-logs/operate" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"operation": "restart"}'
```

#### 6.1.2 更新 Daemon 配置

**步骤 1：备份配置**
```bash
sudo cp /etc/daemon/daemon.yaml /etc/daemon/daemon.yaml.bak.$(date +%Y%m%d)
```

**步骤 2：更新配置**
```bash
sudo vim /etc/daemon/daemon.yaml
```

**步骤 3：验证配置语法**
```bash
# 使用 YAML 验证工具
yamllint /etc/daemon/daemon.yaml
```

**步骤 4：重启 Daemon**
```bash
sudo systemctl restart daemon
```

**步骤 5：验证更新**
```bash
# 查看 Daemon 日志
tail -f /var/log/daemon/daemon.log | grep "agent registered"
```

### 6.2 Agent 版本升级流程

#### 6.2.1 升级前准备

1. **备份当前版本**
   ```bash
   # 备份二进制文件
   sudo cp /usr/bin/filebeat /usr/bin/filebeat.backup
   
   # 备份配置
   sudo cp /etc/filebeat/filebeat.yml /etc/filebeat/filebeat.yml.backup
   ```

2. **查看当前版本**
   ```bash
   /usr/bin/filebeat version
   ```

3. **下载新版本**
   ```bash
   wget https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-8.12.0-linux-x86_64.tar.gz
   tar -xzf filebeat-8.12.0-linux-x86_64.tar.gz
   ```

#### 6.2.2 执行升级

**方法 1：手动升级**

1. **停止 Agent**
   ```bash
   curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/filebeat-logs/operate" \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"operation": "stop"}'
   ```

2. **替换二进制文件**
   ```bash
   sudo mv filebeat-8.12.0-linux-x86_64/filebeat /usr/bin/filebeat
   sudo chmod +x /usr/bin/filebeat
   ```

3. **更新配置**（如果需要）
   ```bash
   # 检查配置兼容性
   /usr/bin/filebeat test config -c /etc/filebeat/filebeat.yml
   ```

4. **启动 Agent**
   ```bash
   curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/filebeat-logs/operate" \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"operation": "start"}'
   ```

**方法 2：通过 Manager 版本管理功能**（如果支持）

1. 在 Manager Web 界面上传新版本
2. 选择目标节点和 Agent
3. 执行版本更新
4. 系统自动完成升级流程

#### 6.2.3 验证升级

1. **检查版本**
   ```bash
   /usr/bin/filebeat version
   ```

2. **检查运行状态**
   ```bash
   curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents/filebeat-logs" \
     -H "Authorization: Bearer <token>"
   ```

3. **查看日志**
   ```bash
   tail -f /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log
   ```

#### 6.2.4 回滚（如果升级失败）

1. **停止 Agent**
   ```bash
   curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/filebeat-logs/operate" \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"operation": "stop"}'
   ```

2. **恢复旧版本**
   ```bash
   sudo mv /usr/bin/filebeat.backup /usr/bin/filebeat
   ```

3. **恢复配置**（如果需要）
   ```bash
   sudo cp /etc/filebeat/filebeat.yml.backup /etc/filebeat/filebeat.yml
   ```

4. **启动 Agent**
   ```bash
   curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/filebeat-logs/operate" \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"operation": "start"}'
   ```

### 6.3 数据备份和恢复

#### 6.3.1 备份内容

1. **配置文件**
   - `/etc/daemon/daemon.yaml`
   - Agent 配置文件（如 `/etc/filebeat/filebeat.yml`）

2. **工作目录数据**
   - `/var/lib/daemon/agents/{agent_id}/`（包含状态文件、日志等）

3. **日志文件**
   - `/var/log/daemon/daemon.log`
   - Agent 日志文件

#### 6.3.2 备份脚本示例

```bash
#!/bin/bash
# backup_agent_data.sh

BACKUP_DIR="/backup/daemon/$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR

# 备份 Daemon 配置
cp /etc/daemon/daemon.yaml $BACKUP_DIR/

# 备份 Agent 配置
cp /etc/filebeat/filebeat.yml $BACKUP_DIR/
cp /etc/telegraf/telegraf.conf $BACKUP_DIR/

# 备份工作目录
tar -czf $BACKUP_DIR/agents_workdir.tar.gz /var/lib/daemon/agents/

# 备份日志
tar -czf $BACKUP_DIR/logs.tar.gz /var/log/daemon/

echo "Backup completed: $BACKUP_DIR"
```

#### 6.3.3 恢复流程

1. **停止相关服务**
   ```bash
   sudo systemctl stop daemon
   ```

2. **恢复配置文件**
   ```bash
   cp /backup/daemon/20250127/daemon.yaml /etc/daemon/
   cp /backup/daemon/20250127/filebeat.yml /etc/filebeat/
   ```

3. **恢复工作目录**
   ```bash
   tar -xzf /backup/daemon/20250127/agents_workdir.tar.gz -C /
   ```

4. **启动服务**
   ```bash
   sudo systemctl start daemon
   ```

### 6.4 系统清理和维护

#### 6.4.1 日志清理

**自动日志轮转**（推荐）

在 `daemon.yaml` 中配置日志轮转：

```yaml
daemon:
  log_file: /var/log/daemon/daemon.log
  log_rotation:
    max_size: 100  # MB
    max_backups: 10
    max_age: 30  # days
    compress: true
```

**手动清理旧日志**

```bash
# 清理 30 天前的日志
find /var/log/daemon -name "*.log.*" -mtime +30 -delete

# 清理 Agent 日志
find /var/lib/daemon/agents -name "*.log.*" -mtime +30 -delete
```

#### 6.4.2 工作目录清理

```bash
# 清理临时文件
find /var/lib/daemon/agents -name "*.tmp" -mtime +7 -delete

# 清理旧的状态文件
find /var/lib/daemon/agents -name "*.state.old" -mtime +30 -delete
```

#### 6.4.3 定期维护任务

**创建维护脚本**：

```bash
#!/bin/bash
# maintenance.sh

# 1. 清理旧日志
find /var/log/daemon -name "*.log.*" -mtime +30 -delete
find /var/lib/daemon/agents -name "*.log.*" -mtime +30 -delete

# 2. 清理临时文件
find /var/lib/daemon/agents -name "*.tmp" -mtime +7 -delete

# 3. 检查磁盘空间
df -h /var/lib/daemon | awk 'NR==2 {if ($5+0 > 80) print "Warning: Disk usage > 80%"}'

# 4. 检查 Agent 状态
# 通过 API 检查所有 Agent 状态，记录异常

echo "Maintenance completed at $(date)"
```

**设置定时任务**：

```bash
# 添加到 crontab
0 2 * * 0 /usr/local/bin/maintenance.sh >> /var/log/maintenance.log 2>&1
```

---

## 附录

### A. 相关文档

- [Agent 管理功能使用指南](./Agent管理功能使用指南.md)
- [Agent 管理开发者文档](./Agent管理开发者文档.md)
- [Agent 管理配置示例](./Agent管理配置示例.md)
- [Daemon 多 Agent 管理架构设计](./设计文档_04_Daemon多Agent管理架构.md)

### B. 配置文件模板

完整的配置文件模板请参考：
- `daemon/configs/daemon.multi-agent.example.yaml`

---

**文档版本**: v1.0  
**最后更新**: 2025-01-27  
**维护者**: Ops Scaffold Framework Team
