# Test Agent

轻量级测试 Agent,用于演示和验证 Daemon 的多 Agent 管理功能。

## 功能特性

### ✅ 已实现 (P0 必需功能)

- **配置文件读取**: 使用 Viper 读取 YAML 配置,支持环境变量覆盖
- **定时心跳上报**: 通过 Unix Socket 向 Daemon 发送心跳,包含 CPU/Memory 使用情况
- **HTTP Health Check**: 提供 `/health` 端点,返回 Agent 健康状态
- **优雅退出**: 正确处理 SIGTERM/SIGINT 信号,发送最后一次心跳后退出
- **结构化日志**: 使用 zap 输出 JSON 格式日志

### ✅ 已实现 (P1 推荐功能)

- **指标暴露**: 提供 `/metrics` 端点,返回 Agent 自身指标(CPU/Memory/心跳统计)
- **配置重载**: 提供 `/reload` 端点(基础实现,TODO: 完整功能)

## 快速开始

### 1. 构建

```bash
make build
```

### 2. 配置

编辑 `configs/agent.yaml`:

```yaml
agent_id: "agent-001"
heartbeat:
  socket_path: "/tmp/daemon.sock"  # Daemon 心跳接收 Socket 路径
  interval: 30s
http:
  port: 8081
  host: "0.0.0.0"
log:
  level: "info"
  format: "json"
```

### 3. 运行

**前提**: 确保 Daemon 已启动并创建了 Unix Socket (`/tmp/daemon.sock`)

```bash
# 使用默认配置
make run

# 使用自定义配置
./bin/agent -config /path/to/config.yaml

# 使用环境变量覆盖配置
AGENT_AGENT_ID=agent-002 AGENT_HTTP_PORT=8082 ./bin/agent
```

## HTTP API

### GET /health

健康检查接口

**响应示例**:
```json
{
  "status": "healthy",
  "uptime": 3600,
  "last_heartbeat": "2025-12-05T10:00:00Z",
  "agent_id": "agent-001"
}
```

**测试命令**:
```bash
curl http://localhost:8081/health
```

### GET /metrics

指标查询接口

**响应示例**:
```json
{
  "agent_id": "agent-001",
  "version": "1.0.0",
  "uptime": 3600,
  "heartbeat_count": 120,
  "heartbeat_failures": 2,
  "last_heartbeat": "2025-12-05T10:00:00Z",
  "cpu_percent": 2.5,
  "memory_bytes": 10240000,
  "status": "running"
}
```

**测试命令**:
```bash
curl http://localhost:8081/metrics
```

### POST /reload

配置重载接口

**响应示例**:
```json
{
  "success": true,
  "message": "config reloaded successfully",
  "reloaded_at": "2025-12-05T10:00:00Z"
}
```

**测试命令**:
```bash
curl -X POST http://localhost:8081/reload
```

## 心跳数据格式

Agent 通过 Unix Socket 向 Daemon 发送 JSON 格式的心跳数据:

```json
{
  "pid": 12345,
  "timestamp": "2025-12-05T10:00:00Z",
  "version": "1.0.0",
  "status": "running",
  "cpu": 2.5,
  "memory": 10240000
}
```

**字段说明**:
- `pid`: Agent 进程 PID
- `timestamp`: 心跳时间戳(ISO 8601 格式)
- `version`: Agent 版本号
- `status`: 运行状态(`running`/`stopping`/`error`)
- `cpu`: CPU 使用率(%)
- `memory`: 内存占用(字节)

## 环境变量

支持通过环境变量覆盖配置:

- `AGENT_AGENT_ID`: Agent ID
- `AGENT_HEARTBEAT_SOCKET_PATH`: Unix Socket 路径
- `AGENT_HEARTBEAT_INTERVAL`: 心跳间隔
- `AGENT_HTTP_PORT`: HTTP 端口
- `AGENT_LOG_LEVEL`: 日志级别

## 开发

### 运行测试

```bash
make test
```

### 代码格式化

```bash
make fmt
```

### 静态检查

```bash
make lint
```

### 清理

```bash
make clean
```

## 架构

```
┌─────────────────┐         ┌─────────────────┐
│  Test Agent     │◄───────►│    Daemon       │
│                 │ Heartbeat│                 │
│  - Config       │  (Unix)  │  - Registry     │
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

## 与 Daemon 集成

1. **启动 Daemon**: 确保 Daemon 已启动并监听 Unix Socket
2. **配置 Socket 路径**: Agent 配置中的 `heartbeat.socket_path` 需要与 Daemon 一致
3. **启动 Agent**: Agent 会自动连接到 Daemon 并开始发送心跳
4. **监控**: Daemon 的 MultiHealthChecker 会监控 Agent 的健康状态

## 目录结构

```
agent/
├── cmd/agent/          # 主程序入口
├── internal/
│   ├── config/        # 配置管理
│   ├── heartbeat/     # 心跳机制
│   ├── api/           # HTTP API
│   └── logger/        # 日志
├── configs/           # 配置文件
├── docs/              # 设计文档
├── go.mod             # Go 模块
├── Makefile           # 构建脚本
└── README.md          # 本文档
```

## 设计文档

详细的设计文档请参考:

- [DESIGN.md](docs/DESIGN.md) - 完整架构设计文档
- [第三方Agent特征分析.md](docs/第三方Agent特征分析.md) - 第三方 Agent 特征分析
- [功能设计文档.md](docs/功能设计文档.md) - 功能设计说明
- [接口规范文档.md](docs/接口规范文档.md) - 心跳和 API 接口规范

## 许可证

本项目遵循 MIT 许可证。
