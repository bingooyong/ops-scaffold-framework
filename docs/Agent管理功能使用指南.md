# Agent 管理功能使用指南

**版本**: v1.0  
**更新日期**: 2025-01-27  
**适用版本**: Ops Scaffold Framework v0.3.0+

---

## 目录

1. [功能概述](#1-功能概述)
2. [架构说明](#2-架构说明)
3. [使用场景](#3-使用场景)
4. [操作流程](#4-操作流程)
5. [常见问题](#5-常见问题)
6. [最佳实践](#6-最佳实践)

---

## 1. 功能概述

### 1.1 功能定位和价值

Agent 管理功能是 Ops Scaffold Framework 的核心功能之一，提供了对分布式主机上多个第三方 Agent（如 Filebeat、Telegraf、Node Exporter 等）的统一管理和监控能力。

**核心价值**：
- **统一管理**：通过 Manager 中心节点统一管理所有主机上的 Agent，无需逐台登录操作
- **自动化运维**：支持 Agent 的自动启动、停止、重启，以及异常自动恢复
- **实时监控**：实时查看 Agent 运行状态、资源使用情况、健康状态
- **配置管理**：集中管理 Agent 配置，支持配置更新和版本控制
- **故障排查**：提供日志查看、状态查询等工具，快速定位问题

### 1.2 支持的功能列表

| 功能类别 | 功能项 | 说明 |
|---------|--------|------|
| **生命周期管理** | 启动 Agent | 通过 Web 界面或 API 启动指定 Agent |
| | 停止 Agent | 支持优雅停止和强制停止 |
| | 重启 Agent | 支持手动重启和自动重启（异常恢复） |
| | 批量操作 | 支持批量启动、停止、重启多个 Agent |
| **监控功能** | 状态监控 | 实时查看 Agent 运行状态（running/stopped/error 等） |
| | 资源监控 | 监控 Agent 的 CPU、内存使用情况 |
| | 健康检查 | 自动健康检查，包括进程检查、HTTP 端点检查、心跳检查 |
| | 告警通知 | Agent 异常时自动告警（需配置告警规则） |
| **配置管理** | 配置查看 | 查看 Agent 的配置信息 |
| | 配置更新 | 更新 Agent 配置并重新加载 |
| | 配置模板 | 提供常见 Agent 类型的配置模板 |
| **日志管理** | 日志查看 | 查看 Agent 运行日志（通过 Web 界面或 API） |
| | 日志下载 | 下载 Agent 日志文件 |
| | 日志轮转 | 自动日志轮转，防止日志文件过大 |
| **故障排查** | 状态查询 | 查询 Agent 详细状态信息 |
| | 故障诊断 | 提供故障诊断工具和建议 |
| | 历史记录 | 查看 Agent 操作历史和状态变化记录 |

### 1.3 支持的 Agent 类型

| Agent 类型 | 说明 | 典型用途 |
|-----------|------|----------|
| **filebeat** | Elastic 日志采集 Agent | 日志收集和转发到 Elasticsearch |
| **telegraf** | InfluxData 指标采集 Agent | 系统指标采集和上报 |
| **node_exporter** | Prometheus 节点指标采集器 | Prometheus 监控指标暴露 |
| **custom** | 自定义 Agent | 用户自定义的第三方 Agent |

---

## 2. 架构说明

### 2.1 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    Manager (中心管理节点)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │  HTTP API    │  │  gRPC Server │  │   Database   │        │
│  │  (Port 8080) │  │  (Port 9090) │  │   (MySQL)   │        │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘        │
└─────────┼──────────────────┼─────────────────────────────────┘
          │ HTTP/HTTPS        │ gRPC (mTLS)
          │ (JWT Auth)        │
┌─────────▼───────────────────▼─────────────────────────────────┐
│                    Web Frontend (React)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Dashboard  │  │ Node Manager │  │ Agent Manager│      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└──────────────────────────────────────────────────────────────┘
          │
          │ gRPC (mTLS)
          │
┌─────────▼─────────────────────────────────────────────────────┐
│              Daemon (守护进程，运行在每台主机)                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         MultiAgentManager (多 Agent 管理器)            │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────┐  │   │
│  │  │AgentRegistry │  │AgentInstance │  │HealthCheck│  │   │
│  │  │  (注册表)    │  │  (实例管理)  │  │ (健康检查)│  │   │
│  │  └──────────────┘  └──────────────┘  └──────────┘  │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │HeartbeatRecv │  │ResourceMonitor│  │  LogManager  │      │
│  │ (心跳接收)   │  │ (资源监控)    │  │  (日志管理)  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────┬─────────────────────────────────────────────────────┘
          │ Unix Socket / HTTP
          │ (心跳上报)
┌─────────▼─────────────────────────────────────────────────────┐
│              Agent (第三方 Agent 进程)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Filebeat    │  │  Telegraf    │  │Node Exporter │      │
│  │  (日志采集)  │  │  (指标采集)  │  │  (指标暴露)  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└───────────────────────────────────────────────────────────────┘
```

### 2.2 数据流图

#### 2.2.1 Agent 注册流程

```
1. Daemon 启动
   ↓
2. 读取 daemon.yaml 配置文件
   ↓
3. 解析 agents 数组配置
   ↓
4. 创建 AgentInfo 并注册到 AgentRegistry
   ↓
5. 创建 AgentInstance 并添加到 MultiAgentManager
   ↓
6. 通过 gRPC 同步 Agent 信息到 Manager
   ↓
7. Manager 存储到数据库 (agents 表)
```

#### 2.2.2 Agent 状态同步流程

```
Agent 进程
   ↓ (心跳上报，30秒间隔)
Daemon HeartbeatReceiver
   ↓ (更新本地元数据)
MultiAgentManager.UpdateHeartbeat()
   ↓ (定期同步，60秒间隔)
Manager gRPC Client
   ↓ (gRPC 调用)
Manager gRPC Server
   ↓ (更新数据库)
Database (agents 表)
   ↓ (Web 前端轮询)
Web Frontend (实时显示)
```

#### 2.2.3 Agent 操作流程（启动/停止/重启）

```
Web Frontend / API Client
   ↓ (HTTP POST /api/v1/nodes/:node_id/agents/:agent_id/operate)
Manager HTTP API Handler
   ↓ (验证权限、查询节点状态)
Manager Service Layer
   ↓ (gRPC 调用 Daemon)
Daemon gRPC Server
   ↓ (调用 MultiAgentManager)
MultiAgentManager.StartAgent() / StopAgent() / RestartAgent()
   ↓ (调用 AgentInstance)
AgentInstance.Start() / Stop() / Restart()
   ↓ (执行系统调用)
Agent 进程启动/停止/重启
   ↓ (状态更新)
状态同步回 Manager
```

### 2.3 组件说明

#### 2.3.1 AgentRegistry（Agent 注册表）

**职责**：管理所有已注册的 Agent 实例，提供并发安全的注册、查询、列举功能。

**核心方法**：
- `Register()`: 注册新 Agent
- `Get(id)`: 根据 ID 获取 Agent 信息
- `List()`: 列举所有 Agent
- `ListByType(type)`: 根据类型列举 Agent
- `Unregister(id)`: 注销 Agent

#### 2.3.2 MultiAgentManager（多 Agent 管理器）

**职责**：管理多个 Agent 实例的生命周期，提供批量操作和单个操作接口。

**核心方法**：
- `RegisterAgent()`: 注册 Agent 并创建实例
- `StartAgent()`: 启动指定 Agent
- `StopAgent()`: 停止指定 Agent
- `RestartAgent()`: 重启指定 Agent
- `StartAll()`: 批量启动所有 Agent
- `StopAll()`: 批量停止所有 Agent

#### 2.3.3 AgentInstance（Agent 实例管理器）

**职责**：管理单个 Agent 进程的生命周期，包括启动、停止、重启、状态管理。

**核心功能**：
- 进程管理（启动、停止、重启）
- 状态跟踪（PID、运行状态、重启次数）
- 日志管理（日志文件路径、日志轮转）
- 配置管理（启动参数生成、工作目录管理）

#### 2.3.4 MultiHealthChecker（多 Agent 健康检查器）

**职责**：监控所有 Agent 的健康状态，执行健康检查并触发自动恢复。

**检查方式**：
- **进程检查**：检查 Agent 进程是否存在
- **HTTP 端点检查**：如果 Agent 提供 HTTP 健康检查端点，定期请求
- **心跳检查**：检查 Agent 心跳是否超时
- **资源检查**：检查 CPU、内存使用是否超过阈值

**自动恢复**：
- Agent 异常退出时自动重启
- 心跳超时时自动重启
- 资源超限时自动重启（可配置）

---

## 3. 使用场景

### 3.1 场景 1: 部署新的 Agent

**场景描述**：在节点上部署一个新的 Filebeat Agent 用于日志采集。

**操作步骤**：

1. **准备 Agent 二进制文件**
   ```bash
   # 下载并安装 Filebeat
   wget https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-8.11.0-linux-x86_64.tar.gz
   tar -xzf filebeat-8.11.0-linux-x86_64.tar.gz
   sudo mv filebeat-8.11.0-linux-x86_64 /usr/local/filebeat
   ```

2. **配置 Filebeat**
   ```bash
   # 创建配置文件
   sudo vim /etc/filebeat/filebeat.yml
   ```

3. **配置 Daemon**
   ```bash
   # 编辑 daemon.yaml，添加 Agent 配置
   sudo vim /etc/daemon/daemon.yaml
   ```
   
   添加配置：
   ```yaml
   agents:
     - id: filebeat-logs
       type: filebeat
       name: "Filebeat Log Collector"
       binary_path: /usr/local/filebeat/filebeat
       config_file: /etc/filebeat/filebeat.yml
       work_dir: /var/lib/daemon/agents/filebeat-logs
       enabled: true
   ```

4. **重启 Daemon**
   ```bash
   sudo systemctl restart daemon
   ```

5. **验证部署**
   - 通过 Web 界面查看 Agent 状态
   - 或通过 API 查询：`GET /api/v1/nodes/:node_id/agents`

### 3.2 场景 2: 监控 Agent 运行状态

**场景描述**：实时监控所有节点上的 Agent 运行状态，及时发现异常。

**操作方式**：

1. **通过 Web 界面监控**
   - 登录 Web 管理界面
   - 进入"节点管理" → 选择节点 → 查看"Agent 列表"
   - 查看 Agent 状态、PID、最后心跳时间、资源使用情况

2. **通过 API 监控**
   ```bash
   # 获取节点下所有 Agent
   curl -X GET "http://manager:8080/api/v1/nodes/node-001/agents" \
     -H "Authorization: Bearer <token>"
   ```

3. **设置告警规则**
   - Agent 状态变为 `error` 时告警
   - Agent 心跳超时（90秒）时告警
   - Agent CPU 使用率超过 80% 持续 5 分钟时告警

### 3.3 场景 3: 批量管理多个 Agent

**场景描述**：在多个节点上批量启动/停止/重启相同类型的 Agent。

**操作步骤**：

1. **通过 Web 界面批量操作**
   - 进入"节点管理"
   - 使用筛选功能选择多个节点
   - 选择要操作的 Agent 类型（如所有 Filebeat）
   - 点击"批量启动"或"批量停止"

2. **通过 API 批量操作**
   ```bash
   # 遍历节点列表，对每个节点执行操作
   for node_id in node-001 node-002 node-003; do
     curl -X POST "http://manager:8080/api/v1/nodes/$node_id/agents/filebeat-logs/operate" \
       -H "Authorization: Bearer <token>" \
       -H "Content-Type: application/json" \
       -d '{"operation": "start"}'
   done
   ```

3. **使用脚本自动化**
   ```bash
   #!/bin/bash
   # 批量重启所有节点的 Filebeat
   NODES=$(curl -s "http://manager:8080/api/v1/nodes" \
     -H "Authorization: Bearer $TOKEN" | jq -r '.data.list[].node_id')
   
   for node in $NODES; do
     echo "Restarting filebeat on $node..."
     curl -X POST "http://manager:8080/api/v1/nodes/$node/agents/filebeat-logs/operate" \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"operation": "restart"}'
   done
   ```

### 3.4 场景 4: 故障排查和恢复

**场景描述**：Agent 运行异常，需要排查问题并恢复服务。

**排查步骤**：

1. **查看 Agent 状态**
   ```bash
   # 通过 API 查询 Agent 详细状态
   curl -X GET "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs" \
     -H "Authorization: Bearer <token>"
   ```

2. **查看 Agent 日志**
   ```bash
   # 通过 API 获取日志
   curl -X GET "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs/logs?lines=100" \
     -H "Authorization: Bearer <token>"
   ```

3. **检查配置文件**
   ```bash
   # 登录到节点，检查 Agent 配置文件
   ssh node-001
   cat /etc/filebeat/filebeat.yml
   ```

4. **检查进程状态**
   ```bash
   # 检查 Agent 进程
   ps aux | grep filebeat
   ```

5. **尝试重启 Agent**
   ```bash
   # 通过 API 重启
   curl -X POST "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs/operate" \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"operation": "restart"}'
   ```

6. **如果自动重启失败，手动排查**
   - 检查二进制文件是否存在：`ls -l /usr/local/filebeat/filebeat`
   - 检查配置文件语法：`/usr/local/filebeat/filebeat test config`
   - 检查工作目录权限：`ls -ld /var/lib/daemon/agents/filebeat-logs`
   - 检查日志文件：`tail -f /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log`

---

## 4. 操作流程

### 4.1 通过 Web 前端管理 Agent

#### 4.1.1 查看 Agent 列表

1. 登录 Web 管理界面
2. 点击左侧菜单"节点管理"
3. 在节点列表中选择目标节点
4. 进入节点详情页，切换到"Agent"标签页
5. 查看该节点下的所有 Agent 列表

**显示信息**：
- Agent ID 和名称
- Agent 类型（filebeat/telegraf/node_exporter/custom）
- 运行状态（running/stopped/error/starting/stopping）
- 进程 ID (PID)
- 最后心跳时间
- CPU 和内存使用率

#### 4.1.2 启动 Agent

1. 在 Agent 列表中找到目标 Agent
2. 如果 Agent 状态为 `stopped`，点击"启动"按钮
3. 系统会发送启动指令到 Daemon
4. 等待几秒钟，刷新页面查看状态变化
5. 如果状态变为 `running`，表示启动成功

#### 4.1.3 停止 Agent

1. 在 Agent 列表中找到目标 Agent
2. 如果 Agent 状态为 `running`，点击"停止"按钮
3. 选择停止方式：
   - **优雅停止**：发送停止信号，等待 Agent 自行退出（推荐）
   - **强制停止**：直接 kill 进程（仅在优雅停止失败时使用）
4. 等待几秒钟，刷新页面查看状态变化

#### 4.1.4 重启 Agent

1. 在 Agent 列表中找到目标 Agent
2. 点击"重启"按钮
3. 系统会先停止 Agent，然后重新启动
4. 等待几秒钟，刷新页面查看状态变化

#### 4.1.5 查看 Agent 日志

1. 在 Agent 列表中找到目标 Agent
2. 点击"查看日志"按钮
3. 在弹出的日志查看器中查看日志
4. 可以设置日志行数（默认 100 行，最多 1000 行）
5. 支持实时刷新和日志下载

### 4.2 通过 API 管理 Agent

#### 4.2.1 获取 Agent 列表

```bash
# 请求示例
curl -X GET "http://manager:8080/api/v1/nodes/node-001/agents" \
  -H "Authorization: Bearer <your-jwt-token>"

# 响应示例
{
  "code": 0,
  "message": "success",
  "data": {
    "agents": [
      {
        "id": 1,
        "node_id": "node-001",
        "agent_id": "filebeat-logs",
        "type": "filebeat",
        "version": "8.11.0",
        "status": "running",
        "pid": 12345,
        "last_heartbeat": "2025-01-27T10:00:00Z",
        "created_at": "2025-01-27T09:00:00Z",
        "updated_at": "2025-01-27T10:00:00Z"
      }
    ],
    "count": 1
  }
}
```

#### 4.2.2 启动 Agent

```bash
# 请求示例
curl -X POST "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs/operate" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "start"
  }'

# 响应示例
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "操作成功"
  }
}
```

#### 4.2.3 停止 Agent

```bash
# 请求示例
curl -X POST "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs/operate" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "stop"
  }'
```

#### 4.2.4 重启 Agent

```bash
# 请求示例
curl -X POST "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs/operate" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "restart"
  }'
```

#### 4.2.5 获取 Agent 日志

```bash
# 请求示例（获取最近 200 行日志）
curl -X GET "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs/logs?lines=200" \
  -H "Authorization: Bearer <your-jwt-token>"

# 响应示例
{
  "code": 0,
  "message": "success",
  "data": {
    "logs": [
      "2025-01-27T10:00:00Z [INFO] Agent started",
      "2025-01-27T10:00:01Z [INFO] Configuration loaded",
      "2025-01-27T10:00:02Z [INFO] Heartbeat sent"
    ],
    "count": 3
  }
}
```

### 4.3 通过 Daemon 配置管理 Agent

#### 4.3.1 配置文件位置

Daemon 配置文件通常位于：
- Linux: `/etc/daemon/daemon.yaml` 或 `~/.daemon/daemon.yaml`
- 开发环境: `daemon/configs/daemon.yaml`

#### 4.3.2 添加 Agent 配置

编辑 `daemon.yaml`，在 `agents` 数组中添加新配置：

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
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

#### 4.3.3 重新加载配置

**方法 1：重启 Daemon 服务**
```bash
sudo systemctl restart daemon
```

**方法 2：发送重载信号（如果支持）**
```bash
sudo kill -HUP $(cat /var/run/daemon.pid)
```

#### 4.3.4 验证配置

1. 检查 Daemon 日志，确认配置加载成功
   ```bash
   tail -f /var/log/daemon/daemon.log | grep "agent registered"
   ```

2. 通过 API 查询 Agent 列表，确认新 Agent 已注册
   ```bash
   curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents" \
     -H "Authorization: Bearer <token>"
   ```

---

## 5. 常见问题

### 5.1 Agent 无法启动

**问题现象**：Agent 状态一直为 `starting` 或 `failed`，无法正常启动。

**排查步骤**：

1. **检查二进制文件**
   ```bash
   # 检查文件是否存在
   ls -l /usr/bin/filebeat
   
   # 检查文件权限
   ls -l /usr/bin/filebeat | grep -E "^-r.x"
   ```

2. **检查配置文件**
   ```bash
   # 检查配置文件是否存在
   ls -l /etc/filebeat/filebeat.yml
   
   # 检查配置文件语法（如果 Agent 支持）
   /usr/bin/filebeat test config -c /etc/filebeat/filebeat.yml
   ```

3. **检查工作目录**
   ```bash
   # 检查工作目录是否存在
   ls -ld /var/lib/daemon/agents/filebeat-logs
   
   # 检查目录权限
   ls -ld /var/lib/daemon/agents/filebeat-logs | grep -E "^drwx"
   ```

4. **查看 Daemon 日志**
   ```bash
   tail -100 /var/log/daemon/daemon.log | grep -i "filebeat"
   ```

5. **查看 Agent 日志**
   ```bash
   # Agent 日志通常在工作目录下
   tail -100 /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log
   ```

**常见原因和解决方法**：

| 原因 | 解决方法 |
|------|----------|
| 二进制文件不存在 | 安装 Agent 或检查 `binary_path` 配置 |
| 配置文件不存在 | 创建配置文件或检查 `config_file` 配置 |
| 工作目录权限不足 | 修改目录权限：`chmod 755 /var/lib/daemon/agents/filebeat-logs` |
| 配置文件语法错误 | 修复配置文件语法错误 |
| 端口被占用 | 检查端口占用：`netstat -tlnp | grep <port>` |

### 5.2 Agent 状态不同步

**问题现象**：Web 界面显示的 Agent 状态与实际运行状态不一致。

**排查步骤**：

1. **检查 Daemon 与 Manager 的连接**
   ```bash
   # 查看 Daemon 日志
   tail -f /var/log/daemon/daemon.log | grep -i "grpc\|sync"
   ```

2. **手动触发状态同步**
   ```bash
   # 通过 API 查询 Agent 状态（会触发同步）
   curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id" \
     -H "Authorization: Bearer <token>"
   ```

3. **检查心跳上报**
   ```bash
   # 查看 Daemon 心跳接收日志
   tail -f /var/log/daemon/daemon.log | grep -i "heartbeat"
   ```

**解决方法**：

- **重启 Daemon**：`sudo systemctl restart daemon`
- **检查网络连接**：确保 Daemon 可以连接到 Manager
- **检查 gRPC 配置**：验证 TLS 证书配置正确

### 5.3 日志查看问题

**问题现象**：无法通过 Web 界面或 API 查看 Agent 日志。

**排查步骤**：

1. **检查日志文件是否存在**
   ```bash
   ls -l /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log
   ```

2. **检查日志文件权限**
   ```bash
   ls -l /var/lib/daemon/agents/filebeat-logs/filebeat-logs.log | grep -E "^-r"
   ```

3. **检查 Daemon 日志功能**
   ```bash
   # 查看 Daemon 日志，确认日志管理功能正常
   tail -f /var/log/daemon/daemon.log | grep -i "log"
   ```

**解决方法**：

- **直接查看日志文件**：`tail -f /var/lib/daemon/agents/:agent_id/:agent_id.log`
- **检查 LogManager 配置**：确保日志管理功能已启用
- **检查文件权限**：确保 Daemon 进程有读取日志文件的权限

### 5.4 性能问题排查

**问题现象**：Agent CPU 或内存使用率过高。

**排查步骤**：

1. **查看 Agent 资源使用情况**
   ```bash
   # 通过 API 查询
   curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id" \
     -H "Authorization: Bearer <token>" | jq '.data.agent.cpu_usage'
   ```

2. **检查 Agent 配置**
   ```bash
   # 查看 Agent 配置文件，检查是否有资源限制配置
   cat /etc/filebeat/filebeat.yml | grep -i "cpu\|memory\|limit"
   ```

3. **检查系统资源**
   ```bash
   # 查看系统整体资源使用
   top
   htop
   ```

**解决方法**：

- **调整 Agent 配置**：减少采集频率、限制并发数
- **设置资源限制**：在 `health_check` 中设置合理的阈值
- **升级硬件**：如果资源确实不足，考虑升级服务器硬件
- **重启 Agent**：临时解决资源泄漏问题

---

## 6. 最佳实践

### 6.1 Agent 配置建议

1. **使用有意义的 Agent ID**
   - 格式：`{type}-{purpose}`，如 `filebeat-logs`、`telegraf-metrics`
   - 避免使用特殊字符和空格

2. **合理设置工作目录**
   - 每个 Agent 使用独立的工作目录
   - 工作目录路径：`{daemon.work_dir}/agents/{agent_id}`
   - 确保目录权限正确：`chmod 755 <work_dir>`

3. **配置启动参数**
   - 根据 Agent 类型设置正确的启动参数
   - Filebeat: `["-c", "{config_file}", "-path.home", "{work_dir}"]`
   - Telegraf: `["-config", "{config_file}"]`
   - Node Exporter: `["--web.listen-address=:9100", ...]`

### 6.2 资源限制设置

1. **CPU 使用率限制**
   ```yaml
   health_check:
     cpu_threshold: 50.0  # CPU 使用率阈值（%）
     threshold_duration: 60s  # 超过阈值持续时间
   ```

2. **内存使用限制**
   ```yaml
   health_check:
     memory_threshold: 524288000  # 内存使用阈值（字节，500MB）
     threshold_duration: 60s
   ```

3. **建议值**：
   - Filebeat: CPU 50%, Memory 500MB
   - Telegraf: CPU 40%, Memory 250MB
   - Node Exporter: CPU 30%, Memory 100MB

### 6.3 监控告警配置

1. **设置健康检查间隔**
   ```yaml
   health_check:
     interval: 30s  # 健康检查间隔
   ```

2. **设置心跳超时**
   ```yaml
   health_check:
     heartbeat_timeout: 90s  # 心跳超时时间（3个心跳周期）
   ```

3. **配置告警规则**（在 Manager 或监控系统中）：
   - Agent 状态变为 `error` → 立即告警
   - 心跳超时（90秒） → 告警
   - CPU 使用率 > 80% 持续 5 分钟 → 告警
   - 内存使用率 > 90% 持续 5 分钟 → 告警
   - 重启次数 > 10 次/小时 → 告警

### 6.4 安全建议

1. **文件权限设置**
   ```bash
   # Agent 二进制文件
   chmod 755 /usr/bin/filebeat
   chown root:root /usr/bin/filebeat
   
   # 配置文件
   chmod 644 /etc/filebeat/filebeat.yml
   chown root:root /etc/filebeat/filebeat.yml
   
   # 工作目录
   chmod 755 /var/lib/daemon/agents/filebeat-logs
   chown daemon:daemon /var/lib/daemon/agents/filebeat-logs
   ```

2. **网络访问控制**
   - 限制 Agent HTTP API 访问（如果 Agent 提供）
   - 使用防火墙规则限制端口访问
   - 使用 TLS/HTTPS 加密通信

3. **日志脱敏**
   - 配置文件中的敏感信息（密码、密钥）不要记录到日志
   - 使用环境变量或密钥管理服务存储敏感信息

4. **认证和授权**
   - Manager API 使用 JWT 认证
   - 限制 API 访问权限（RBAC）
   - 定期轮换 JWT 密钥

---

## 附录

### A. 相关文档

- [Agent 管理管理员手册](./Agent管理管理员手册.md)
- [Agent 管理开发者文档](./Agent管理开发者文档.md)
- [Agent 管理配置示例](./Agent管理配置示例.md)
- [Manager API 文档](./api/Manager_API.md)
- [Daemon 多 Agent 管理架构设计](./设计文档_04_Daemon多Agent管理架构.md)

### B. 技术支持

如遇到问题，请：
1. 查阅本文档的"常见问题"章节
2. 查看系统日志文件
3. 联系技术支持团队

---

**文档版本**: v1.0  
**最后更新**: 2025-01-27  
**维护者**: Ops Scaffold Framework Team
