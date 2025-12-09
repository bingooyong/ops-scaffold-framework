# Agent 管理开发者文档

**版本**: v1.0  
**更新日期**: 2025-01-27  
**适用版本**: Ops Scaffold Framework v0.3.0+

---

## 目录

1. [如何开发自定义 Agent](#1-如何开发自定义-agent)
2. [心跳协议规范](#2-心跳协议规范)
3. [API 接口说明](#3-api-接口说明)
4. [集成指南](#4-集成指南)
5. [扩展开发](#5-扩展开发)

---

## 1. 如何开发自定义 Agent

### 1.1 Agent 开发规范和要求

#### 1.1.1 基本要求

自定义 Agent 需要满足以下基本要求：

1. **独立进程运行**
   - Agent 必须作为独立进程运行，不能是 Daemon 的子进程
   - 支持通过命令行参数或配置文件启动

2. **进程管理**
   - 支持优雅停止（响应 SIGTERM/SIGINT 信号）
   - 支持强制停止（响应 SIGKILL 信号）
   - 进程退出时返回适当的退出码

3. **心跳上报**（推荐）
   - 定期向 Daemon 发送心跳数据
   - 心跳间隔：30 秒（可配置）
   - 心跳超时：90 秒（3 个心跳周期）

4. **HTTP API**（推荐）
   - 提供 HTTP 健康检查端点（如 `/health`）
   - 提供状态查询端点（如 `/status`）
   - 支持配置重载端点（如 `/reload`）

5. **日志输出**
   - 支持结构化日志输出（JSON 格式推荐）
   - 日志级别可配置（DEBUG/INFO/WARN/ERROR）
   - 日志输出到标准输出或文件

#### 1.1.2 Agent 必须实现的功能

**1. 心跳上报功能**

Agent 需要定期向 Daemon 发送心跳数据，用于 Daemon 监控 Agent 运行状态。

**实现方式**：
- **方式 1**：通过 Unix Domain Socket 发送心跳（推荐，本地通信）
- **方式 2**：通过 HTTP API 发送心跳（如果 Daemon 提供 HTTP 端点）

**心跳数据格式**：JSON 格式，包含以下字段：
```json
{
  "agent_id": "custom-agent-001",
  "pid": 12345,
  "status": "running",
  "cpu": 5.0,
  "memory": 10240000,
  "timestamp": "2025-01-27T10:00:00Z"
}
```

**2. HTTP API 功能**

Agent 需要提供 HTTP API，用于健康检查和状态查询。

**必需端点**：
- `GET /health`: 健康检查端点
- `GET /status`: 状态查询端点

**可选端点**：
- `POST /reload`: 配置重载端点
- `GET /metrics`: 指标暴露端点（Prometheus 格式）

#### 1.1.3 Agent 配置格式要求

Agent 配置文件可以是任意格式（YAML、JSON、TOML、INI 等），但需要包含以下信息：

**必需配置项**：
- `agent_id`: Agent 唯一标识符
- `heartbeat_interval`: 心跳间隔（秒）
- `socket_path` 或 `heartbeat_url`: 心跳上报地址

**推荐配置项**：
- `log_level`: 日志级别
- `log_file`: 日志文件路径
- `http_listen`: HTTP API 监听地址

**配置示例（YAML 格式）**：
```yaml
agent:
  id: "custom-agent-001"
  name: "Custom Application Agent"
  
heartbeat:
  interval: 30s
  socket_path: "/var/run/daemon/agents/custom-agent-001.sock"
  # 或使用 HTTP 方式
  # url: "http://localhost:5060/heartbeat"

http:
  listen: ":8080"
  enable_health: true
  enable_metrics: true

logging:
  level: "info"
  file: "/var/log/custom-agent/agent.log"
```

#### 1.1.4 Agent 与 Daemon 的通信协议

**通信方式**：

1. **Unix Domain Socket**（推荐）
   - 协议：JSON-RPC 2.0 或纯 JSON
   - 路径：`/var/run/daemon/agents/{agent_id}.sock`
   - 用途：心跳上报、状态查询

2. **HTTP API**（可选）
   - 协议：HTTP/HTTPS
   - 端点：`POST /heartbeat`（心跳上报）
   - 用途：心跳上报、健康检查

**通信流程**：

```
Agent 启动
   ↓
连接到 Daemon Socket（或 HTTP 端点）
   ↓
定期发送心跳（30 秒间隔）
   ↓
Daemon 接收心跳并更新状态
   ↓
Agent 停止时发送最后心跳（status="stopping"）
```

### 1.2 Agent 开发示例（Go 语言）

#### 1.2.1 项目结构

```
custom-agent/
├── cmd/
│   └── agent/
│       └── main.go          # 主程序入口
├── internal/
│   ├── agent/
│   │   ├── agent.go         # Agent 核心逻辑
│   │   └── config.go        # 配置管理
│   ├── heartbeat/
│   │   └── heartbeat.go    # 心跳模块
│   └── http/
│       └── server.go        # HTTP API 服务器
├── configs/
│   └── agent.yaml           # 配置文件示例
├── go.mod
└── README.md
```

#### 1.2.2 心跳模块实现

```go
package heartbeat

import (
    "encoding/json"
    "net"
    "os"
    "time"
    "github.com/shirou/gopsutil/v3/process"
)

// Heartbeat 心跳数据结构
type Heartbeat struct {
    AgentID   string    `json:"agent_id"`
    PID       int       `json:"pid"`
    Status    string    `json:"status"`
    CPU       float64   `json:"cpu"`
    Memory    uint64    `json:"memory"`
    Timestamp time.Time `json:"timestamp"`
}

// Manager 心跳管理器
type Manager struct {
    agentID     string
    socketPath  string
    interval    time.Duration
    conn        net.Conn
    logger      Logger
}

// NewManager 创建心跳管理器
func NewManager(agentID, socketPath string, interval time.Duration, logger Logger) *Manager {
    return &Manager{
        agentID:    agentID,
        socketPath: socketPath,
        interval:   interval,
        logger:     logger,
    }
}

// Start 启动心跳发送
func (m *Manager) Start() error {
    // 连接到 Daemon Socket
    conn, err := net.Dial("unix", m.socketPath)
    if err != nil {
        return err
    }
    m.conn = conn

    // 启动心跳循环
    go m.heartbeatLoop()
    return nil
}

// heartbeatLoop 心跳发送循环
func (m *Manager) heartbeatLoop() {
    ticker := time.NewTicker(m.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if err := m.sendHeartbeat("running"); err != nil {
                m.logger.Error("failed to send heartbeat", err)
                // 尝试重连
                if err := m.reconnect(); err != nil {
                    m.logger.Error("failed to reconnect", err)
                }
            }
        }
    }
}

// sendHeartbeat 发送心跳
func (m *Manager) sendHeartbeat(status string) error {
    // 采集资源使用情况
    cpu, memory := m.collectResourceUsage()

    // 构建心跳数据
    hb := Heartbeat{
        AgentID:   m.agentID,
        PID:       os.Getpid(),
        Status:    status,
        CPU:       cpu,
        Memory:    memory,
        Timestamp: time.Now(),
    }

    // 序列化为 JSON
    data, err := json.Marshal(hb)
    if err != nil {
        return err
    }

    // 发送到 Socket
    _, err = m.conn.Write(append(data, '\n'))
    return err
}

// collectResourceUsage 采集资源使用情况
func (m *Manager) collectResourceUsage() (float64, uint64) {
    pid := os.Getpid()
    proc, err := process.NewProcess(int32(pid))
    if err != nil {
        return 0, 0
    }

    cpu, _ := proc.CPUPercent()
    memInfo, _ := proc.MemoryInfo()
    if memInfo == nil {
        return cpu, 0
    }

    return cpu, memInfo.RSS
}

// reconnect 重连到 Daemon
func (m *Manager) reconnect() error {
    if m.conn != nil {
        m.conn.Close()
    }

    conn, err := net.Dial("unix", m.socketPath)
    if err != nil {
        return err
    }
    m.conn = conn
    return nil
}

// Stop 停止心跳发送
func (m *Manager) Stop() {
    // 发送最后心跳
    m.sendHeartbeat("stopping")
    
    if m.conn != nil {
        m.conn.Close()
    }
}
```

#### 1.2.3 HTTP API 服务器实现

```go
package http

import (
    "encoding/json"
    "net/http"
    "time"
)

// Server HTTP API 服务器
type Server struct {
    agentID string
    status  string
    startTime time.Time
}

// NewServer 创建 HTTP 服务器
func NewServer(agentID string) *Server {
    return &Server{
        agentID:   agentID,
        status:    "running",
        startTime: time.Now(),
    }
}

// Start 启动 HTTP 服务器
func (s *Server) Start(addr string) error {
    http.HandleFunc("/health", s.handleHealth)
    http.HandleFunc("/status", s.handleStatus)
    http.HandleFunc("/reload", s.handleReload)
    return http.ListenAndServe(addr, nil)
}

// handleHealth 处理健康检查请求
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    if s.status == "running" {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "healthy",
        })
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
        })
    }
}

// handleStatus 处理状态查询请求
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]interface{}{
        "agent_id":    s.agentID,
        "status":      s.status,
        "uptime":      time.Since(s.startTime).Seconds(),
        "started_at":  s.startTime.Format(time.RFC3339),
    })
}

// handleReload 处理配置重载请求
func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 实现配置重载逻辑
    // ...

    json.NewEncoder(w).Encode(map[string]interface{}{
        "success":     true,
        "message":     "config reloaded successfully",
        "reloaded_at": time.Now().Format(time.RFC3339),
    })
}
```

#### 1.2.4 主程序实现

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "custom-agent/internal/agent"
    "custom-agent/internal/heartbeat"
    "custom-agent/internal/http"
)

func main() {
    // 加载配置
    cfg := loadConfig()
    
    // 创建 Agent 实例
    a := agent.New(cfg)
    
    // 启动心跳管理器
    hbManager := heartbeat.NewManager(
        cfg.AgentID,
        cfg.SocketPath,
        30*time.Second,
        logger,
    )
    if err := hbManager.Start(); err != nil {
        logger.Fatal("failed to start heartbeat manager", err)
    }
    defer hbManager.Stop()
    
    // 启动 HTTP 服务器
    httpServer := http.NewServer(cfg.AgentID)
    go func() {
        if err := httpServer.Start(cfg.HTTPListen); err != nil {
            logger.Fatal("failed to start HTTP server", err)
        }
    }()
    
    // 启动 Agent 主逻辑
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go a.Run(ctx)
    
    // 等待停止信号
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    <-sigChan
    logger.Info("received stop signal, shutting down...")
    
    // 优雅停止
    cancel()
    hbManager.Stop()
    
    logger.Info("agent stopped")
}
```

### 1.3 其他语言实现

#### 1.3.1 Python 实现示例

```python
import json
import socket
import time
import psutil
import os
from datetime import datetime

class HeartbeatManager:
    def __init__(self, agent_id, socket_path, interval=30):
        self.agent_id = agent_id
        self.socket_path = socket_path
        self.interval = interval
        self.conn = None
    
    def connect(self):
        self.conn = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.conn.connect(self.socket_path)
    
    def send_heartbeat(self, status="running"):
        # 采集资源使用情况
        process = psutil.Process(os.getpid())
        cpu = process.cpu_percent()
        memory = process.memory_info().rss
        
        # 构建心跳数据
        heartbeat = {
            "agent_id": self.agent_id,
            "pid": os.getpid(),
            "status": status,
            "cpu": cpu,
            "memory": memory,
            "timestamp": datetime.utcnow().isoformat() + "Z"
        }
        
        # 发送到 Socket
        data = json.dumps(heartbeat) + "\n"
        self.conn.send(data.encode())
    
    def start(self):
        self.connect()
        while True:
            try:
                self.send_heartbeat()
                time.sleep(self.interval)
            except Exception as e:
                print(f"Error sending heartbeat: {e}")
                self.connect()
```

#### 1.3.2 Node.js 实现示例

```javascript
const net = require('net');
const os = require('os');
const process = require('process');

class HeartbeatManager {
    constructor(agentId, socketPath, interval = 30000) {
        this.agentId = agentId;
        this.socketPath = socketPath;
        this.interval = interval;
        this.client = null;
    }
    
    connect() {
        this.client = net.createConnection(this.socketPath);
        this.client.on('error', (err) => {
            console.error('Socket error:', err);
            setTimeout(() => this.connect(), 5000);
        });
    }
    
    sendHeartbeat(status = 'running') {
        const heartbeat = {
            agent_id: this.agentId,
            pid: process.pid,
            status: status,
            cpu: process.cpuUsage().user / 1000000, // 简化示例
            memory: process.memoryUsage().rss,
            timestamp: new Date().toISOString()
        };
        
        const data = JSON.stringify(heartbeat) + '\n';
        if (this.client && this.client.writable) {
            this.client.write(data);
        }
    }
    
    start() {
        this.connect();
        setInterval(() => {
            this.sendHeartbeat();
        }, this.interval);
    }
}
```

---

## 2. 心跳协议规范

### 2.1 心跳数据格式（JSON 结构）

Agent 向 Daemon 发送的心跳数据采用 JSON 格式：

```json
{
  "agent_id": "custom-agent-001",
  "pid": 12345,
  "status": "running",
  "cpu": 5.0,
  "memory": 10240000,
  "timestamp": "2025-01-27T10:00:00Z"
}
```

### 2.2 字段说明

| 字段名 | 类型 | 必填 | 说明 |
|-------|------|------|------|
| `agent_id` | string | 是 | Agent 唯一标识符，必须与 Daemon 配置中的 `id` 一致 |
| `pid` | int | 是 | Agent 进程 PID，用于 Daemon 验证进程存在性 |
| `status` | string | 是 | Agent 运行状态，枚举值：<br>- `"running"`: 正常运行<br>- `"stopping"`: 正在停止（优雅退出时发送）<br>- `"error"`: 错误状态（发生异常时发送） |
| `cpu` | float | 否 | CPU 使用率（百分比），范围 0-100 |
| `memory` | int | 否 | 内存占用（字节），推荐使用 RSS（Resident Set Size） |
| `timestamp` | string | 是 | 心跳时间戳，ISO 8601 格式（UTC 时区），例如：`"2025-01-27T10:00:00Z"` |

### 2.3 心跳上报频率要求

- **默认间隔**：30 秒
- **可配置范围**：10-300 秒
- **心跳超时**：90 秒（3 个心跳周期）
- **超时处理**：如果 Daemon 连续 3 次未收到心跳，判定 Agent 异常并触发自动重启

### 2.4 心跳上报方式

#### 2.4.1 Unix Domain Socket（推荐）

**连接方式**：
- 协议：Unix Domain Socket（AF_UNIX）
- 路径：`/var/run/daemon/agents/{agent_id}.sock`
- 数据格式：JSON 文本，每行一个心跳数据，以 `\n` 结尾

**实现示例**（Go）：
```go
conn, err := net.Dial("unix", "/var/run/daemon/agents/custom-agent-001.sock")
if err != nil {
    return err
}
defer conn.Close()

heartbeat := map[string]interface{}{
    "agent_id": "custom-agent-001",
    "pid": os.Getpid(),
    "status": "running",
    "cpu": 5.0,
    "memory": 10240000,
    "timestamp": time.Now().UTC().Format(time.RFC3339),
}

data, _ := json.Marshal(heartbeat)
conn.Write(append(data, '\n'))
```

#### 2.4.2 HTTP API（如果 Daemon 提供）

**端点**：`POST http://localhost:5060/heartbeat`

**请求格式**：
- 请求方法：`POST`
- 请求头：`Content-Type: application/json`
- 请求体：心跳数据 JSON

**响应格式**：
- 成功：`200 OK`，响应体 `{"success": true, "message": "heartbeat received"}`
- 失败：`400 Bad Request`，响应体 `{"success": false, "error": "invalid agent_id"}`

**实现示例**（Go）：
```go
heartbeat := map[string]interface{}{
    "agent_id": "custom-agent-001",
    "pid": os.Getpid(),
    "status": "running",
    "cpu": 5.0,
    "memory": 10240000,
    "timestamp": time.Now().UTC().Format(time.RFC3339),
}

data, _ := json.Marshal(heartbeat)
resp, err := http.Post("http://localhost:5060/heartbeat", "application/json", bytes.NewBuffer(data))
```

### 2.5 心跳数据字段说明

#### 2.5.1 agent_id

- **类型**：string
- **必填**：是
- **说明**：Agent 唯一标识符，必须与 Daemon 配置中的 `id` 字段完全一致
- **示例**：`"custom-agent-001"`

#### 2.5.2 pid

- **类型**：int
- **必填**：是
- **说明**：Agent 进程 PID，Daemon 使用此字段验证进程存在性
- **获取方式**：`os.Getpid()`（Go）、`os.getpid()`（Python）、`process.pid`（Node.js）

#### 2.5.3 status

- **类型**：string
- **必填**：是
- **枚举值**：
  - `"running"`: Agent 正常运行
  - `"stopping"`: Agent 正在停止（优雅退出时发送）
  - `"error"`: Agent 发生错误
- **说明**：Daemon 根据此字段判断 Agent 状态

#### 2.5.4 cpu

- **类型**：float
- **必填**：否
- **说明**：CPU 使用率（百分比），范围 0-100
- **获取方式**：使用系统库（如 `gopsutil`、`psutil`）获取进程 CPU 使用率

#### 2.5.5 memory

- **类型**：int
- **必填**：否
- **说明**：内存占用（字节），推荐使用 RSS（Resident Set Size）
- **获取方式**：使用系统库获取进程内存占用

#### 2.5.6 timestamp

- **类型**：string
- **必填**：是
- **格式**：ISO 8601 格式（UTC 时区），例如：`"2025-01-27T10:00:00Z"`
- **说明**：心跳时间戳，用于计算心跳延迟和超时

---

## 3. API 接口说明

### 3.1 Daemon gRPC API（Agent 管理相关）

Daemon 通过 gRPC 与 Manager 通信，提供以下 Agent 管理相关的接口：

#### 3.1.1 ListAgents - 列举所有 Agent

**方法**：`ListAgents`

**请求**：
```protobuf
message ListAgentsRequest {
  // 空消息
}
```

**响应**：
```protobuf
message ListAgentsResponse {
  repeated AgentInfo agents = 1;
}

message AgentInfo {
  string agent_id = 1;
  string type = 2;
  string status = 3;
  int32 pid = 4;
  int64 last_heartbeat = 5;
}
```

**说明**：Manager 调用此接口获取 Daemon 管理的所有 Agent 列表。

#### 3.1.2 OperateAgent - 操作 Agent

**方法**：`OperateAgent`

**请求**：
```protobuf
message AgentOperationRequest {
  string agent_id = 1;
  string operation = 2;  // "start", "stop", "restart"
}
```

**响应**：
```protobuf
message AgentOperationResponse {
  bool success = 1;
  string message = 2;
}
```

**说明**：Manager 调用此接口操作 Agent（启动/停止/重启）。

#### 3.1.3 SyncAgentStates - 同步 Agent 状态

**方法**：`SyncAgentStates`

**请求**：
```protobuf
message SyncAgentStatesRequest {
  repeated AgentState states = 1;
}

message AgentState {
  string agent_id = 1;
  string status = 2;
  int32 pid = 3;
  int64 last_heartbeat = 4;
}
```

**响应**：
```protobuf
message SyncAgentStatesResponse {
  bool success = 1;
  string message = 2;
}
```

**说明**：Daemon 定期调用此接口向 Manager 同步 Agent 状态。

### 3.2 Manager HTTP API（Agent 管理相关）

Manager 提供 HTTP API 供 Web 前端和外部系统调用，Agent 管理相关的接口如下：

#### 3.2.1 获取节点下的所有 Agent

**接口**：`GET /api/v1/nodes/:node_id/agents`

**请求示例**：
```bash
curl -X GET "http://manager:8080/api/v1/nodes/node-001/agents" \
  -H "Authorization: Bearer <token>"
```

**响应示例**：
```json
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
        "status": "running",
        "pid": 12345,
        "last_heartbeat": "2025-01-27T10:00:00Z"
      }
    ],
    "count": 1
  }
}
```

#### 3.2.2 操作 Agent

**接口**：`POST /api/v1/nodes/:node_id/agents/:agent_id/operate`

**请求示例**：
```bash
curl -X POST "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs/operate" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "start"
  }'
```

**响应示例**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "操作成功"
  }
}
```

#### 3.2.3 获取 Agent 日志

**接口**：`GET /api/v1/nodes/:node_id/agents/:agent_id/logs`

**请求示例**：
```bash
curl -X GET "http://manager:8080/api/v1/nodes/node-001/agents/filebeat-logs/logs?lines=100" \
  -H "Authorization: Bearer <token>"
```

**响应示例**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "logs": [
      "2025-01-27T10:00:00Z [INFO] Agent started",
      "2025-01-27T10:00:01Z [INFO] Configuration loaded"
    ],
    "count": 2
  }
}
```

### 3.3 API 调用示例（Go 代码示例）

#### 3.3.1 调用 Manager HTTP API

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

// AgentClient Manager API 客户端
type AgentClient struct {
    baseURL string
    token   string
    client  *http.Client
}

// NewAgentClient 创建 API 客户端
func NewAgentClient(baseURL, token string) *AgentClient {
    return &AgentClient{
        baseURL: baseURL,
        token:   token,
        client:  &http.Client{},
    }
}

// ListAgents 获取节点下的所有 Agent
func (c *AgentClient) ListAgents(nodeID string) ([]Agent, error) {
    url := fmt.Sprintf("%s/api/v1/nodes/%s/agents", c.baseURL, nodeID)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+c.token)
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Code int `json:"code"`
        Data struct {
            Agents []Agent `json:"agents"`
        } `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Data.Agents, nil
}

// OperateAgent 操作 Agent
func (c *AgentClient) OperateAgent(nodeID, agentID, operation string) error {
    url := fmt.Sprintf("%s/api/v1/nodes/%s/agents/%s/operate", c.baseURL, nodeID, agentID)
    
    body := map[string]string{
        "operation": operation,
    }
    data, _ := json.Marshal(body)
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", "Bearer "+c.token)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("operation failed: %d", resp.StatusCode)
    }
    
    return nil
}

// Agent Agent 信息
type Agent struct {
    ID           int    `json:"id"`
    NodeID       string `json:"node_id"`
    AgentID      string `json:"agent_id"`
    Type         string `json:"type"`
    Status       string `json:"status"`
    PID          int    `json:"pid"`
    LastHeartbeat string `json:"last_heartbeat"`
}

// 使用示例
func main() {
    client := NewAgentClient("http://manager:8080", "your-jwt-token")
    
    // 获取 Agent 列表
    agents, err := client.ListAgents("node-001")
    if err != nil {
        panic(err)
    }
    
    for _, agent := range agents {
        fmt.Printf("Agent: %s, Status: %s\n", agent.AgentID, agent.Status)
    }
    
    // 启动 Agent
    if err := client.OperateAgent("node-001", "filebeat-logs", "start"); err != nil {
        panic(err)
    }
}
```

---

## 4. 集成指南

### 4.1 如何将第三方 Agent 集成到系统

#### 4.1.1 集成步骤

**步骤 1：安装 Agent 二进制文件**

```bash
# 示例：安装 Filebeat
wget https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-8.11.0-linux-x86_64.tar.gz
tar -xzf filebeat-8.11.0-linux-x86_64.tar.gz
sudo mv filebeat-8.11.0-linux-x86_64 /usr/local/filebeat
sudo chmod +x /usr/local/filebeat/filebeat
```

**步骤 2：配置 Agent**

```bash
# 创建配置文件
sudo vim /etc/filebeat/filebeat.yml
```

**步骤 3：在 Daemon 配置中添加 Agent**

编辑 `/etc/daemon/daemon.yaml`：

```yaml
agents:
  - id: filebeat-logs
    type: filebeat
    name: "Filebeat Log Collector"
    binary_path: /usr/local/filebeat/filebeat
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

**步骤 4：重启 Daemon**

```bash
sudo systemctl restart daemon
```

**步骤 5：验证集成**

```bash
# 查看 Daemon 日志
tail -f /var/log/daemon/daemon.log | grep "filebeat"

# 通过 API 查询 Agent
curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents" \
  -H "Authorization: Bearer <token>"
```

#### 4.1.2 集成注意事项

1. **Agent 类型选择**
   - 如果 Agent 是 filebeat、telegraf、node_exporter，使用对应的类型
   - 如果是其他 Agent，使用 `custom` 类型

2. **启动参数配置**
   - 查看 Agent 官方文档，了解启动参数格式
   - 在 `args` 字段中配置正确的启动参数

3. **健康检查配置**
   - 如果 Agent 提供 HTTP 健康检查端点，配置 `http_endpoint`
   - 如果 Agent 支持心跳，配置 `heartbeat_timeout`
   - 根据 Agent 资源占用特点设置合理的阈值

### 4.2 配置模板创建方法

#### 4.2.1 创建配置模板

为常见 Agent 类型创建配置模板，方便快速部署：

**模板文件**：`daemon/configs/templates/filebeat.yaml.template`

```yaml
agents:
  - id: filebeat-{ENV}
    type: filebeat
    name: "Filebeat Log Collector"
    binary_path: {BINARY_PATH}
    config_file: {CONFIG_FILE}
    work_dir: {WORK_DIR}
    enabled: true
    args:
      - "-c"
      - "{CONFIG_FILE}"
      - "-path.home"
      - "{WORK_DIR}"
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

#### 4.2.2 使用配置模板

```bash
# 使用环境变量替换模板变量
export BINARY_PATH="/usr/local/filebeat/filebeat"
export CONFIG_FILE="/etc/filebeat/filebeat.yml"
export WORK_DIR="/var/lib/daemon/agents/filebeat-logs"
export ENV="production"

envsubst < daemon/configs/templates/filebeat.yaml.template >> daemon.yaml
```

### 4.3 测试和验证步骤

#### 4.3.1 功能测试

1. **启动测试**
   ```bash
   # 通过 API 启动 Agent
   curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id/operate" \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"operation": "start"}'
   
   # 验证 Agent 是否启动
   ps aux | grep filebeat
   ```

2. **状态查询测试**
   ```bash
   # 查询 Agent 状态
   curl -X GET "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id" \
     -H "Authorization: Bearer <token>"
   ```

3. **健康检查测试**
   ```bash
   # 如果 Agent 提供 HTTP 健康检查端点
   curl http://localhost:5060/health
   ```

4. **停止测试**
   ```bash
   # 停止 Agent
   curl -X POST "http://manager:8080/api/v1/nodes/:node_id/agents/:agent_id/operate" \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"operation": "stop"}'
   ```

#### 4.3.2 异常测试

1. **进程异常退出测试**
   ```bash
   # 手动 kill Agent 进程
   kill -9 $(pgrep -f filebeat)
   
   # 验证自动重启
   sleep 60
   ps aux | grep filebeat
   ```

2. **心跳超时测试**
   ```bash
   # 停止心跳发送（修改 Agent 代码或配置）
   # 验证 Daemon 是否检测到心跳超时并重启
   ```

3. **资源超限测试**
   ```bash
   # 模拟 CPU 或内存使用过高
   # 验证 Daemon 是否检测到资源超限并重启
   ```

---

## 5. 扩展开发

### 5.1 如何扩展 Agent 类型支持

#### 5.1.1 添加新的 Agent 类型常量

在 `daemon/internal/agent/registry.go` 中添加新的类型常量：

```go
const (
    TypeFilebeat     AgentType = "filebeat"
    TypeTelegraf     AgentType = "telegraf"
    TypeNodeExporter AgentType = "node_exporter"
    TypeCustom       AgentType = "custom"
    TypeNewAgent     AgentType = "new_agent"  // 新增类型
)
```

#### 5.1.2 添加默认启动参数生成逻辑

在 `daemon/internal/agent/instance.go` 中的 `generateDefaultArgs` 函数中添加新类型的处理：

```go
func generateDefaultArgs(agentType AgentType, configFile string, workDir string) []string {
    switch agentType {
    case TypeFilebeat:
        return []string{"-c", configFile, "-path.home", workDir}
    case TypeTelegraf:
        return []string{"-config", configFile}
    case TypeNodeExporter:
        return []string{"--web.listen-address=:9100", "--path.procfs=/proc", "--path.sysfs=/sys"}
    case TypeNewAgent:
        return []string{"--config", configFile, "--work-dir", workDir}  // 新增类型参数
    default:
        return []string{}
    }
}
```

#### 5.1.3 更新配置示例

在 `daemon/configs/daemon.multi-agent.example.yaml` 中添加新类型的配置示例：

```yaml
agents:
  - id: new-agent-001
    type: new_agent
    name: "New Agent"
    binary_path: /usr/local/bin/new-agent
    config_file: /etc/new-agent/config.yaml
    work_dir: /var/lib/daemon/agents/new-agent-001
    enabled: true
    args:
      - "--config"
      - "/etc/new-agent/config.yaml"
      - "--work-dir"
      - "/var/lib/daemon/agents/new-agent-001"
```

### 5.2 如何添加新的监控指标

#### 5.2.1 扩展 AgentMetadata 结构

在 `daemon/internal/agent/metadata.go` 中扩展 `AgentMetadata` 结构：

```go
type AgentMetadata struct {
    ID            string
    Type          string
    Status        string
    StartTime     time.Time
    RestartCount  int
    LastHeartbeat time.Time
    ResourceUsage ResourceUsageHistory
    
    // 新增指标
    CustomMetric1 float64
    CustomMetric2 int64
}
```

#### 5.2.2 更新心跳数据结构

在心跳数据中添加新指标：

```go
type Heartbeat struct {
    AgentID   string
    PID       int
    Status    string
    CPU       float64
    Memory    uint64
    Timestamp time.Time
    
    // 新增指标
    CustomMetric1 float64 `json:"custom_metric1"`
    CustomMetric2 int64    `json:"custom_metric2"`
}
```

#### 5.2.3 更新存储和查询逻辑

在 `MultiAgentManager.UpdateHeartbeat` 方法中处理新指标：

```go
func (mam *MultiAgentManager) UpdateHeartbeat(agentID string, timestamp time.Time, cpu float64, memory uint64, customMetric1 float64, customMetric2 int64) error {
    // ... 现有逻辑 ...
    
    // 更新新指标
    metadata.CustomMetric1 = customMetric1
    metadata.CustomMetric2 = customMetric2
    
    // ... 保存逻辑 ...
}
```

### 5.3 如何自定义健康检查规则

#### 5.3.1 实现自定义健康检查器

创建新的健康检查器实现：

```go
package agent

import (
    "context"
    "time"
)

// CustomHealthChecker 自定义健康检查器
type CustomHealthChecker struct {
    agentID string
    rules   []HealthCheckRule
}

// HealthCheckRule 健康检查规则
type HealthCheckRule interface {
    Check(ctx context.Context, agent *AgentInstance) (bool, error)
}

// NewCustomHealthChecker 创建自定义健康检查器
func NewCustomHealthChecker(agentID string, rules []HealthCheckRule) *CustomHealthChecker {
    return &CustomHealthChecker{
        agentID: agentID,
        rules:   rules,
    }
}

// Check 执行健康检查
func (c *CustomHealthChecker) Check(ctx context.Context, agent *AgentInstance) (bool, error) {
    for _, rule := range c.rules {
        healthy, err := rule.Check(ctx, agent)
        if err != nil {
            return false, err
        }
        if !healthy {
            return false, nil
        }
    }
    return true, nil
}
```

#### 5.3.2 实现自定义检查规则

```go
// DiskUsageRule 磁盘使用率检查规则
type DiskUsageRule struct {
    threshold float64
    workDir   string
}

func (r *DiskUsageRule) Check(ctx context.Context, agent *AgentInstance) (bool, error) {
    // 检查工作目录磁盘使用率
    usage, err := getDiskUsage(r.workDir)
    if err != nil {
        return false, err
    }
    return usage < r.threshold, nil
}

// NetworkConnectivityRule 网络连通性检查规则
type NetworkConnectivityRule struct {
    endpoint string
    timeout  time.Duration
}

func (r *NetworkConnectivityRule) Check(ctx context.Context, agent *AgentInstance) (bool, error) {
    // 检查网络连通性
    ctx, cancel := context.WithTimeout(ctx, r.timeout)
    defer cancel()
    
    req, _ := http.NewRequestWithContext(ctx, "GET", r.endpoint, nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return false, nil
    }
    defer resp.Body.Close()
    
    return resp.StatusCode == http.StatusOK, nil
}
```

#### 5.3.3 集成到健康检查系统

在 `MultiHealthChecker` 中注册自定义检查器：

```go
func (mhc *MultiHealthChecker) RegisterCustomChecker(agentID string, checker HealthChecker) {
    mhc.customCheckers[agentID] = checker
}
```

---

## 附录

### A. 相关文档

- [Agent 管理功能使用指南](./Agent管理功能使用指南.md)
- [Agent 管理管理员手册](./Agent管理管理员手册.md)
- [Agent 管理配置示例](./Agent管理配置示例.md)
- [Daemon 多 Agent 管理架构设计](./设计文档_04_Daemon多Agent管理架构.md)
- [Manager API 文档](./api/Manager_API.md)

### B. 代码示例

完整的代码示例请参考：
- `agent/`: 测试 Agent 实现示例
- `daemon/internal/agent/`: Daemon Agent 管理实现

### C. 技术支持

如遇到开发问题，请：
1. 查阅本文档和相关架构设计文档
2. 查看代码示例和测试用例
3. 联系技术支持团队

---

**文档版本**: v1.0  
**最后更新**: 2025-01-27  
**维护者**: Ops Scaffold Framework Team
