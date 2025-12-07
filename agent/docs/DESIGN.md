# Agent 架构设计文档

## 文档信息

- **文档版本**: v1.0
- **创建日期**: 2025-01-XX
- **任务引用**: Task 5.1 - 设计测试 Agent 架构和接口规范
- **状态**: 待用户审查批准

---

## 目录

1. [背景和目标](#第-1-章-背景和目标)
2. [第三方 Agent 特征分析](#第-2-章-第三方-agent-特征分析)
3. [Agent 功能设计](#第-3-章-agent-功能设计)
4. [通信协议和接口规范](#第-4-章-通信协议和接口规范)
5. [实施计划](#第-5-章-实施计划)

---

## 第 1 章: 背景和目标

### 1.1 设计背景

Ops Scaffold Framework 是一个轻量级、高可用的分布式运维管理平台，由三个核心组件组成：

1. **Manager (中心管理节点)**: Web 服务，提供全局管理、监控和版本更新功能
2. **Daemon (守护进程)**: 运行在每台被管主机上，负责资源采集、Agent 管理和状态上报
3. **Agent (执行进程)**: 运行在每台被管主机上，负责具体任务执行并提供 HTTP/HTTPS API

当前 Daemon 已实现多 Agent 管理架构（MultiAgentManager），支持同时管理多个 Agent 实例。为了演示和验证 Daemon 的多 Agent 管理功能，需要实现一个轻量级的测试 Agent。

### 1.2 设计目标

Agent 的设计目标包括：

1. **演示 Daemon 多 Agent 管理能力**: Agent 能够与 Daemon 正常通信，展示 Daemon 的 Agent 注册、心跳监控、健康检查等功能
2. **模拟真实 Agent 行为**: Agent 具备真实 Agent（如 filebeat、telegraf）的核心特征，包括独立进程运行、配置文件驱动、HTTP API、心跳机制等
3. **轻量级设计**: Agent 功能简化，仅实现核心管理接口，不进行真实数据采集，资源占用极低
4. **易于部署和测试**: Agent 易于编译、部署和测试，便于快速验证 Daemon 功能

### 1.3 Agent 特点

- **轻量级**: 功能精简，仅实现核心管理接口，资源占用极低
- **易部署**: 单一二进制文件，配置简单，易于部署
- **模拟真实行为**: 模拟真实 Agent 的核心特征，包括心跳上报、健康检查等
- **功能简化**: 不做真实数据采集，仅用于演示和测试

### 1.4 与生产 Agent 的区别

Agent 与生产 Agent（filebeat、telegraf）的主要区别：

| 对比项 | 生产 Agent | Agent |
|-------|-----------|-------|
| **数据采集** | 真实采集日志/指标数据 | 不采集真实数据，仅模拟行为 |
| **输出目标** | 向 Elasticsearch/InfluxDB 等输出 | 无外部输出，仅心跳上报 |
| **功能复杂度** | 功能完整，支持多种插件 | 功能简化，仅实现核心管理接口 |
| **配置复杂度** | 配置项丰富，支持多种场景 | 配置项精简，仅包含必需项 |
| **资源占用** | 根据采集量变化 | 资源占用极低，仅用于演示 |

---

## 第 2 章: 第三方 Agent 特征分析

### 2.1 分析目标

分析常见第三方 Agent（filebeat、telegraf、node_exporter）的核心特征，为 Agent 设计提供参考。

### 2.2 核心 Agent 特征对比

#### 2.2.1 Filebeat（Elastic Stack 日志采集 Agent）

| 特征类别 | 具体特征 |
|---------|---------|
| **进程特性** | 独立进程，长期运行，资源占用：CPU 1-5%，内存 50-200MB |
| **配置方式** | YAML 配置文件，支持热重载 |
| **通信机制** | 无主动心跳上报（被动监控），提供 HTTP API |
| **运维特性** | 支持 SIGTERM/SIGINT 优雅退出，JSON 格式日志输出 |
| **健康检查** | 进程存在性检查，状态文件时间戳检查，HTTP API 健康检查 |

#### 2.2.2 Telegraf（InfluxData 指标采集 Agent）

| 特征类别 | 具体特征 |
|---------|---------|
| **进程特性** | 独立进程，长期运行，资源占用：CPU 2-10%，内存 50-300MB |
| **配置方式** | TOML 配置文件，支持热重载 |
| **通信机制** | 无主动心跳上报（被动监控），提供 HTTP API |
| **运维特性** | 支持 SIGTERM/SIGINT 优雅退出，可配置日志格式 |
| **健康检查** | 进程存在性检查，HTTP API 健康检查 |

#### 2.2.3 Node Exporter（Prometheus 节点监控 Agent）

| 特征类别 | 具体特征 |
|---------|---------|
| **进程特性** | 独立进程，长期运行，资源占用：CPU 1-3%，内存 20-50MB |
| **配置方式** | 命令行参数（无配置文件） |
| **通信机制** | 无主动心跳上报（被动监控），提供 HTTP API |
| **运维特性** | 支持 SIGTERM/SIGINT 优雅退出，文本格式日志输出 |
| **健康检查** | 进程存在性检查，HTTP API 健康检查 |

### 2.3 通用特征总结

基于以上三个 Agent 的分析，总结出以下 **5 个核心通用特征**：

1. **进程特性**: 独立进程运行、长期运行、资源占用可控、多实例支持
2. **配置驱动**: 配置文件格式（YAML/TOML/JSON）、配置热重载、配置验证、配置隔离
3. **通信机制**: HTTP API 暴露、被动监控模式、状态暴露、日志输出
4. **运维特性**: 优雅退出、启动参数、日志管理、状态持久化
5. **健康检查**: 进程存在性、HTTP 健康检查、状态文件检查、资源监控

### 2.4 Agent 必须实现的最小功能集

基于通用特征分析，Agent 需要实现以下 **最小功能集**：

**必需功能（P0）**:
1. 独立进程运行
2. 配置文件读取（YAML，包含 agent_id、心跳配置）
3. 定时心跳上报（主动上报，区别于被动监控模式）
4. HTTP Health Check 接口（`GET /health`）
5. 优雅退出（SIGTERM/SIGINT 信号处理）

**推荐功能（P1）**:
6. 结构化日志输出（JSON 格式）
7. 配置热重载（`POST /reload`）
8. 指标暴露（`GET /metrics`）

**可选功能（P2）**:
9. 资源使用采集（gopsutil）
10. 状态文件（可选的状态文件记录）

---

## 第 3 章: Agent 功能设计

### 3.1 功能清单

Agent 的核心功能清单详见 [功能设计文档.md](./功能设计文档.md)，包括：

- **P0 功能**: 独立进程启动/停止、配置文件读取、定时心跳上报、HTTP Health Check、优雅退出
- **P1 功能**: 结构化日志输出、配置热重载、指标暴露
- **P2 功能**: 资源使用采集、状态文件

### 3.2 技术实现方案

| 功能模块 | 技术选型 | 理由 |
|---------|---------|------|
| **配置管理** | Viper | Go 生态最流行的配置库，支持多种格式 |
| **HTTP 框架** | Gin | 轻量级、高性能，API 简洁 |
| **日志库** | zap | 高性能结构化日志，JSON 输出 |
| **资源采集** | gopsutil | 跨平台进程和系统信息采集 |
| **信号处理** | os/signal | Go 标准库，无需额外依赖 |

### 3.3 目录结构

```
agent/
├── cmd/
│   └── agent/
│       └── main.go              # 入口文件
├── internal/
│   ├── config/                 # 配置管理
│   ├── heartbeat/              # 心跳机制
│   ├── api/                    # HTTP API
│   └── logger/                 # 日志
├── configs/
│   └── agent.yaml              # 示例配置
├── go.mod
├── Makefile
├── README.md
└── docs/
    ├── 第三方Agent特征分析.md
    ├── 功能设计文档.md
    ├── 接口规范文档.md
    └── DESIGN.md               # 本文档
```

### 3.4 实现优先级和依赖关系

**实现顺序建议**:

**第一阶段（P0 功能）**:
1. 配置读取（基础）
2. 结构化日志（基础，其他功能需要）
3. 启动/停止（基础）
4. 资源采集（心跳需要）
5. 心跳上报（核心功能）
6. HTTP Health Check（健康检查）
7. 优雅退出（完善）

**第二阶段（P1 功能）**:
8. 配置热重载
9. 指标暴露

**第三阶段（P2 功能，可选）**:
10. 状态文件

---

## 第 4 章: 通信协议和接口规范

### 4.1 心跳数据格式

Agent 向 Daemon 发送的心跳数据采用 JSON 格式：

```json
{
  "agent_id": "agent-001",
  "pid": 12345,
  "status": "running",
  "cpu": 5.0,
  "memory": 102400,
  "timestamp": "2025-12-05T10:00:00Z"
}
```

**字段说明**:
- `agent_id` (string, 必填): Agent 唯一标识符
- `pid` (int, 必填): Agent 进程 PID
- `status` (string, 必填): 运行状态，枚举值：`"running"`/`"stopping"`/`"error"`
- `cpu` (float, 可选): CPU 使用率（%），范围 0-100
- `memory` (int, 可选): 内存占用（字节）
- `timestamp` (string, 必填): 心跳时间戳，ISO 8601 格式（UTC 时区）

### 4.2 心跳 API 规范

**端点**: `POST http://daemon-host:port/heartbeat`

**请求格式**:
- 请求方法: `POST`
- 请求头: `Content-Type: application/json`
- 请求体: 心跳数据 JSON

**响应格式**:
- 成功: `200 OK`，响应体 `{"message": "heartbeat received"}`
- 失败: `400 Bad Request`，响应体 `{"error": "invalid agent_id"}`

**心跳发送频率**: 默认 30 秒间隔（可通过配置文件 `heartbeat_interval` 调整）

> **注意**: 当前 Daemon 实现使用 Unix Domain Socket 接收心跳。如果 Daemon 尚未提供 HTTP 心跳端点，Agent 可以通过 Unix Socket 发送心跳（使用 JSON-RPC 2.0 格式），或等待 Daemon 添加 HTTP 端点支持。

### 4.3 Agent HTTP API 规范

Agent 提供以下 HTTP API 端点：

#### 4.3.1 健康检查接口

**端点**: `GET /health`

**响应格式**:
```json
{
  "status": "healthy",
  "uptime": 3600,
  "last_heartbeat": "2025-12-05T10:00:00Z",
  "agent_id": "agent-001"
}
```

**HTTP 状态码**:
- `200 OK`: Agent 正常运行
- `503 Service Unavailable`: Agent 不健康（心跳失败超过阈值）

#### 4.3.2 配置重载接口

**端点**: `POST /reload`

**响应格式**:
```json
{
  "success": true,
  "message": "config reloaded successfully",
  "reloaded_at": "2025-12-05T10:00:00Z"
}
```

**HTTP 状态码**:
- `200 OK`: 重载成功
- `400 Bad Request`: 配置文件无效
- `500 Internal Server Error`: 重载失败

#### 4.3.3 指标暴露接口

**端点**: `GET /metrics`

**响应格式**:
```json
{
  "agent_id": "agent-001",
  "uptime": 3600,
  "heartbeat_count": 120,
  "heartbeat_failures": 2,
  "last_heartbeat": "2025-12-05T10:00:00Z",
  "cpu_percent": 2.5,
  "memory_bytes": 10240000,
  "status": "running"
}
```

**HTTP 状态码**: `200 OK`

### 4.4 通信协议总结

| 方向 | 协议 | 端点 | 方法 | 说明 |
|------|------|------|------|------|
| Agent → Daemon | HTTP | `/heartbeat` | POST | 心跳上报（如果 Daemon 支持 HTTP） |
| Agent → Daemon | Unix Socket | `socket_path` | JSON-RPC | 心跳上报（如果 Daemon 仅支持 Socket） |
| 外部 → Agent | HTTP | `/health` | GET | 健康检查 |
| 外部 → Agent | HTTP | `/reload` | POST | 配置重载 |
| 外部 → Agent | HTTP | `/metrics` | GET | 指标查询 |

详细的接口规范详见 [接口规范文档.md](./接口规范文档.md)。

---

## 第 5 章: 实施计划

### 5.1 后续任务概览

基于本设计文档，后续实施任务包括：

- **Task 5.2**: 实现 Agent 核心功能（配置读取、心跳机制、优雅退出）
- **Task 5.3**: 实现 Agent HTTP API（health check、reload、metrics）
- **Task 5.4**: 编写部署文件和文档（配置文件、Makefile、README）
- **Task 5.5**: 端到端验证测试（与 Daemon 集成测试）

### 5.2 Task 5.2: 实现 Agent 核心功能

**目标**: 实现 Agent 核心功能，包括配置读取、心跳机制、优雅退出。

**实施内容**:
1. 在 `agent/` 目录创建 Go 项目，实现 `main.go` 入口
2. 实现配置文件读取：使用 Viper 读取 YAML 配置
3. 实现定时心跳：使用 time.Ticker 每 30 秒向 Daemon 发送心跳
4. 实现优雅退出：监听 SIGTERM/SIGINT 信号，发送最后一次心跳后退出

**输出**: 完成的代码文件，通过单元测试验证

### 5.3 Task 5.3: 实现 Agent HTTP API

**目标**: 实现 Agent HTTP API，包括健康检查、配置重载、指标暴露。

**实施内容**:
1. 使用 Gin 框架创建 HTTP 服务器，监听配置的端口（默认 8081）
2. 实现 `GET /health` 接口：返回 Agent 健康状态
3. 实现 `POST /reload` 接口：触发配置文件重新加载
4. 实现 `GET /metrics` 接口：返回 Agent 自身的指标数据

**输出**: 完成的代码文件，通过单元测试和集成测试验证

### 5.4 Task 5.4: 编写部署文件和文档

**目标**: 编写 Agent 部署文件和文档，便于部署和使用。

**实施内容**:
1. 创建示例配置文件 `agent/configs/agent.yaml`
2. 编写 Makefile：提供 `make build`、`make run`、`make install` 等命令
3. 编写 README.md：说明 Agent 的用途、编译方法、配置说明、使用示例
4. 创建 Systemd service 文件（可选）：用于生产环境部署

**输出**: 配置文件、Makefile、README.md、Systemd service 文件

### 5.5 Task 5.5: 端到端验证测试

**目标**: 端到端验证测试，确保 Agent 与 Daemon 正常集成。

**实施内容**:
1. 编写集成测试：验证 Agent 启动、心跳上报、健康检查等功能
2. 编写端到端测试：验证 Agent 与 Daemon 的完整交互流程
3. 测试多 Agent 场景：验证 Daemon 同时管理多个 Agent 实例
4. 测试异常场景：验证 Agent 异常退出、心跳失败等场景的处理

**输出**: 集成测试文件，测试报告

---

## 架构图

### Agent 与 Daemon 交互架构

```
┌─────────────────┐         ┌─────────────────┐
│     Agent       │◄───────►│     Daemon      │
│                 │ Heartbeat│                 │
│  - Config       │         │  - Registry     │
│  - Heartbeat    │         │  - MultiManager │
│  - HTTP API     │         │  - HealthCheck  │
│  - Logger       │         │                 │
└─────────────────┘         └─────────────────┘
       │
       │ HTTP API
       │ (GET /health, /metrics)
       │ (POST /reload)
       ▼
┌─────────────────┐
│  External       │
│  Monitor/Client │
└─────────────────┘
```

### Agent 内部架构

```
┌─────────────────────────────────────────┐
│              Agent Process              │
│                                         │
│  ┌──────────┐  ┌──────────┐           │
│  │  Config  │  │  Logger  │           │
│  │  (Viper) │  │  (zap)   │           │
│  └────┬─────┘  └────┬─────┘           │
│       │             │                  │
│  ┌────▼────────────▼─────┐           │
│  │   Heartbeat Manager   │           │
│  │  - Ticker (30s)       │           │
│  │  - HTTP Client        │           │
│  │  - Resource Collector│           │
│  └────┬──────────────────┘           │
│       │                               │
│  ┌────▼──────────────┐              │
│  │   HTTP Server      │              │
│  │   (Gin)            │              │
│  │  - /health         │              │
│  │  - /reload         │              │
│  │  - /metrics        │              │
│  └────────────────────┘              │
│                                       │
│  ┌──────────┐                        │
│  │  Signal  │                        │
│  │  Handler │                        │
│  └──────────┘                        │
└─────────────────────────────────────────┘
```

---

## 总结

本文档定义了 Agent 的完整架构设计，包括：

1. **背景和目标**: 明确了 Agent 的设计目标和与生产 Agent 的区别
2. **第三方 Agent 特征分析**: 分析了 filebeat、telegraf、node_exporter 的通用特征
3. **功能设计**: 定义了 Agent 的最小功能集、技术实现方案和目录结构
4. **通信协议和接口规范**: 定义了心跳数据格式、心跳 API 规范和 Agent HTTP API 规范
5. **实施计划**: 列出了后续任务（Task 5.2-5.5）的简要说明

Agent 设计遵循"最小功能集"原则，确保能够演示 Daemon 的多 Agent 管理能力，同时保持轻量级和易部署的特点。

---

## 用户审查与批准

**此设计文档需要用户审查批准。**

请用户回复 **"批准"**、**"确认"** 或提供修改意见。获得批准后，方可进入 Task 5.2 实现阶段。

如果用户提供修改意见而非批准，将根据意见修改文档并重新提交审查。

---

**文档版本**: v1.0  
**创建日期**: 2025-01-XX  
**最后更新**: 2025-01-XX
