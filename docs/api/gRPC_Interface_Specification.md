# gRPC 接口规范文档

本文档定义了 Ops Scaffold Framework 中 Manager 和 Daemon 之间的 gRPC 通信协议规范。

> **版本**: v1.1.0  
> **最后更新**: 2025-12-09  
> **状态**: 已验证一致性

## 1. 概述

### 1.1 通信架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          gRPC 通信架构                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────┐                          ┌─────────────────┐       │
│  │    Manager      │                          │     Daemon      │       │
│  │                 │                          │                 │       │
│  │  ┌───────────┐  │                          │  ┌───────────┐  │       │
│  │  │ Manager   │  │◄──── gRPC (9090) ────────│  │ GRPC      │  │       │
│  │  │ Service   │  │  Daemon → Manager        │  │ Client    │  │       │
│  │  │ (Server)  │  │  - RegisterNode          │  │           │  │       │
│  │  │           │  │  - Heartbeat             │  │           │  │       │
│  │  │           │  │  - ReportMetrics         │  │           │  │       │
│  │  └───────────┘  │                          │  └───────────┘  │       │
│  │                 │                          │                 │       │
│  │  ┌───────────┐  │                          │  ┌───────────┐  │       │
│  │  │ Daemon    │  │────── gRPC (9091) ──────►│  │ Daemon    │  │       │
│  │  │ Client    │  │  Manager → Daemon        │  │ Service   │  │       │
│  │  │ Pool      │  │  - ListAgents            │  │ (Server)  │  │       │
│  │  │           │  │  - OperateAgent          │  │           │  │       │
│  │  │           │  │  - GetAgentMetrics       │  │           │  │       │
│  │  │           │  │  - SyncAgentStates       │  │           │  │       │
│  │  └───────────┘  │                          │  └───────────┘  │       │
│  │                 │                          │                 │       │
│  │  ┌───────────┐  │                          │  ┌───────────┐  │       │
│  │  │ Manager   │  │◄──── gRPC (9090) ────────│  │ Manager   │  │       │
│  │  │ Service   │  │  Daemon → Manager        │  │ Client    │  │       │
│  │  │ (Server)  │  │  - SyncAgentStates       │  │           │  │       │
│  │  └───────────┘  │  (Agent状态同步)          │  └───────────┘  │       │
│  │                 │                          │                 │       │
│  └─────────────────┘                          └─────────────────┘       │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 服务定义

| 服务 | 端口 | 方向 | Proto 文件 | 描述 |
|-----|------|-----|-----------|------|
| ManagerService | 9090 | Daemon → Manager | `manager.proto` | Daemon 向 Manager 上报数据 |
| DaemonService | 9091 | Manager → Daemon | `daemon.proto` | Manager 调用 Daemon 执行操作 |

### 1.3 Proto 文件位置

| 模块 | 文件路径 | 用途 |
|-----|---------|------|
| Manager | `manager/pkg/proto/manager.proto` | Manager 作为 ManagerService 服务端 |
| Manager | `manager/pkg/proto/daemon/daemon.proto` | Manager 作为 DaemonService 客户端 |
| Daemon | `daemon/pkg/proto/manager.proto` | Daemon 作为 ManagerService 客户端 |
| Daemon | `daemon/pkg/proto/manager/manager.proto` | Daemon 作为 ManagerService 客户端 (备用) |
| Daemon | `daemon/pkg/proto/daemon.proto` | Daemon 作为 DaemonService 服务端 |

---

## 2. ManagerService (Daemon → Manager)

**Proto 文件**: `manager/pkg/proto/manager.proto` 和 `daemon/pkg/proto/manager.proto`

**端口**: 9090

### 2.1 RegisterNode - 节点注册

Daemon 启动时向 Manager 注册节点。

#### 请求 (RegisterNodeRequest)

| 字段 | 类型 | 字段号 | 必填 | 描述 |
|-----|------|-------|-----|------|
| node_id | string | 1 | 是 | 节点唯一标识符 |
| hostname | string | 2 | 是 | 主机名 |
| ip | string | 3 | 是 | IP 地址 |
| os | string | 4 | 是 | 操作系统 |
| arch | string | 5 | 是 | CPU 架构 |
| labels | map<string, string> | 6 | 否 | 标签 |
| daemon_version | string | 7 | 否 | Daemon 版本 |
| agent_version | string | 8 | 否 | Agent 版本 |

#### 响应 (RegisterNodeResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| success | bool | 1 | 是否成功 |
| message | string | 2 | 响应消息 |

---

### 2.2 Heartbeat - 心跳上报

Daemon 定期向 Manager 发送心跳。

#### 请求 (HeartbeatRequest)

| 字段 | 类型 | 字段号 | 必填 | 描述 |
|-----|------|-------|-----|------|
| node_id | string | 1 | 是 | 节点 ID |
| timestamp | int64 | 2 | 是 | 时间戳 (Unix) |

#### 响应 (HeartbeatResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| success | bool | 1 | 是否成功 |
| message | string | 2 | 响应消息 |

---

### 2.3 ReportMetrics - 指标上报

Daemon 向 Manager 上报系统指标。

#### 请求 (ReportMetricsRequest)

| 字段 | 类型 | 字段号 | 必填 | 描述 |
|-----|------|-------|-----|------|
| node_id | string | 1 | 是 | 节点 ID |
| metrics | repeated MetricData | 2 | 是 | 指标数据列表 |

#### MetricData

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| type | string | 1 | 指标类型 (cpu/memory/disk/network) |
| timestamp | int64 | 2 | 时间戳 (Unix) |
| values | map<string, double> | 3 | 指标值 |

#### 响应 (ReportMetricsResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| success | bool | 1 | 是否成功 |
| message | string | 2 | 响应消息 |

---

## 3. DaemonService (Manager → Daemon)

**Proto 文件**: `manager/pkg/proto/daemon/daemon.proto` 和 `daemon/pkg/proto/daemon.proto`

**端口**: 9091

### 3.1 ListAgents - 列举 Agent

Manager 获取 Daemon 上的所有 Agent 列表。

#### 请求 (ListAgentsRequest)

空消息，无字段。

#### 响应 (ListAgentsResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| agents | repeated AgentInfo | 1 | Agent 列表 |

#### AgentInfo

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| id | string | 1 | Agent 唯一标识符 |
| type | string | 2 | Agent 类型 (filebeat/telegraf/node_exporter/custom) |
| version | string | 3 | Agent 版本号 (可选) |
| status | string | 4 | 运行状态 (running/stopped/error/starting/stopping) |
| pid | int32 | 5 | 进程 ID (0 表示未运行) |
| start_time | int64 | 6 | 启动时间 (Unix 时间戳) |
| restart_count | int32 | 7 | 重启次数 |
| last_heartbeat | int64 | 8 | 最后心跳时间 (Unix 时间戳) |

> ⚠️ **重要**: 字段顺序必须严格一致，version=3, status=4, pid=5

---

### 3.2 OperateAgent - 操作 Agent

Manager 控制 Daemon 上的 Agent (启动/停止/重启)。

#### 请求 (AgentOperationRequest)

| 字段 | 类型 | 字段号 | 必填 | 描述 |
|-----|------|-------|-----|------|
| agent_id | string | 1 | 是 | Agent ID |
| operation | string | 2 | 是 | 操作类型: start/stop/restart |

#### 响应 (AgentOperationResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| success | bool | 1 | 是否成功 |
| error_message | string | 2 | 错误消息 (失败时) |

---

### 3.3 GetAgentMetrics - 获取 Agent 指标

Manager 获取指定 Agent 的资源使用指标。

#### 请求 (AgentMetricsRequest)

| 字段 | 类型 | 字段号 | 必填 | 描述 |
|-----|------|-------|-----|------|
| agent_id | string | 1 | 是 | Agent ID |
| duration_seconds | int64 | 2 | 否 | 查询时间范围 (秒，默认 3600) |

#### 响应 (AgentMetricsResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| agent_id | string | 1 | Agent ID |
| data_points | repeated ResourceDataPoint | 2 | 资源数据点列表 |

#### ResourceDataPoint

| 字段 | 类型 (规范) | 字段号 | 描述 |
|-----|-----------|-------|------|
| timestamp | int64 | 1 | 时间戳 (Unix) |
| cpu | double | 2 | CPU 使用率 (%) |
| memory_rss | uint64 | 3 | 内存 RSS (字节) |
| memory_vms | uint64 | 4 | 内存 VMS (字节) |
| disk_read_bytes | uint64 | 5 | 磁盘读取字节数 |
| disk_write_bytes | uint64 | 6 | 磁盘写入字节数 |
| open_files | int32 | 7 | 打开文件数 |

> ✅ **已统一**: memory_rss/memory_vms/disk_read_bytes/disk_write_bytes 现已统一使用 `uint64` 类型。

---

### 3.4 SyncAgentStates - 同步 Agent 状态

Daemon 向 Manager 同步所有 Agent 的状态。

#### 请求 (SyncAgentStatesRequest)

| 字段 | 类型 | 字段号 | 必填 | 描述 |
|-----|------|-------|-----|------|
| node_id | string | 1 | 是 | 节点 ID |
| states | repeated AgentState | 2 | 是 | Agent 状态列表 |

#### AgentState

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| agent_id | string | 1 | Agent ID |
| status | string | 2 | 运行状态 (running/stopped/error) |
| pid | int32 | 3 | 进程 ID |
| last_heartbeat | int64 | 4 | 最后心跳时间 (Unix 时间戳) |
| type | string | 5 | Agent 类型 |
| version | string | 6 | Agent 版本号 |

#### 响应 (SyncAgentStatesResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| success | bool | 1 | 是否成功 |
| message | string | 2 | 响应消息 |

---

### 3.5 GetConfig - 获取配置 (预留)

Manager 向 Daemon 下发配置。

#### 请求 (ConfigRequest)

| 字段 | 类型 | 字段号 | 必填 | 描述 |
|-----|------|-------|-----|------|
| node_id | string | 1 | 是 | 节点 ID |
| config_type | string | 2 | 是 | 配置类型: "daemon" 或 "agent" |

#### 响应 (ConfigResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| success | bool | 1 | 是否成功 |
| message | string | 2 | 响应消息 |
| config_data | bytes | 3 | JSON或YAML格式的配置数据 |

---

### 3.6 PushUpdate - 推送更新 (预留)

Manager 向 Daemon 推送更新。

#### 请求 (UpdateRequest)

| 字段 | 类型 | 字段号 | 必填 | 描述 |
|-----|------|-------|-----|------|
| node_id | string | 1 | 是 | 节点 ID |
| component | string | 2 | 是 | 组件类型: "daemon" 或 "agent" |
| version | string | 3 | 是 | 目标版本号 |
| download_url | string | 4 | 否 | 更新包下载 URL |
| hash | string | 5 | 否 | 文件哈希 (SHA256) |
| signature | string | 6 | 否 | 数字签名 (用于验证) |
| update_data | bytes | 7 | 否 | 更新包数据 (可选，用于小文件直接传输) |

#### 响应 (UpdateResponse)

| 字段 | 类型 | 字段号 | 描述 |
|-----|------|-------|------|
| success | bool | 1 | 是否成功 |
| message | string | 2 | 响应消息 |

---

## 4. 一致性检查结果

### 4.1 ✅ 已一致的接口

| 接口 | Manager Proto | Daemon Proto | 状态 | 备注 |
|-----|--------------|--------------|------|------|
| ManagerService.RegisterNode | ✅ | ✅ | 一致 | 字段完全匹配 |
| ManagerService.Heartbeat | ✅ | ✅ | 一致 | 字段完全匹配 |
| ManagerService.ReportMetrics | ✅ | ✅ | 一致 | 字段完全匹配 |
| DaemonService.ListAgents | ✅ | ✅ | 一致 | AgentInfo 字段顺序已修复 |
| DaemonService.OperateAgent | ✅ | ✅ | 一致 | 字段完全匹配 |
| DaemonService.GetAgentMetrics | ✅ | ✅ | 一致 | ResourceDataPoint 类型已统一 |
| DaemonService.SyncAgentStates | ✅ | ✅ | 一致 | 字段完全匹配 |
| DaemonService.GetConfig | ✅ | ✅ | 一致 | ConfigRequest/ConfigResponse 已统一 |
| DaemonService.PushUpdate | ✅ | ✅ | 一致 | UpdateRequest 已统一 |

### 4.2 🔧 已修复的问题

| 问题 | 修复前 | 修复后 | 修复日期 |
|-----|-------|-------|---------|
| AgentInfo 字段顺序 | Manager: status=3, pid=4, version=5 | 统一为: version=3, status=4, pid=5 | 2025-12-09 |
| ConfigRequest 缺少 config_type | Daemon 端缺少 config_type 字段 | 添加 config_type 字段 | 2025-12-09 |
| ConfigResponse 结构不同 | Daemon 端只有 config 字段 | 统一为 success/message/config_data | 2025-12-09 |
| UpdateRequest 结构不同 | Manager 端只有 update_data | 合并两端字段: download_url/hash/signature/update_data | 2025-12-09 |
| ResourceDataPoint 类型不一致 | Manager 端使用 int64 | 统一为 uint64 | 2025-12-09 |

### 4.3 📋 待修复问题清单

当前所有接口已统一，无待修复问题。

---

## 5. 状态值规范

### 5.1 Agent 状态 (status)

| 状态值 | 描述 |
|-------|------|
| stopped | Agent 已停止 |
| starting | Agent 正在启动 |
| running | Agent 正在运行 |
| stopping | Agent 正在停止 |
| restarting | Agent 正在重启 |
| error | Agent 启动失败或运行异常 |
| failed | Agent 启动失败或运行异常 (等同于 error) |

### 5.2 操作类型 (operation)

| 操作值 | 描述 |
|-------|------|
| start | 启动 Agent |
| stop | 停止 Agent |
| restart | 重启 Agent |

### 5.3 Agent 类型 (type)

| 类型值 | 描述 |
|-------|------|
| filebeat | Filebeat 日志采集 |
| telegraf | Telegraf 指标采集 |
| node_exporter | Node Exporter 指标采集 |
| custom | 自定义 Agent |

---

## 6. 错误处理

### 6.1 gRPC 错误码

| 错误码 | 描述 | 场景 |
|-------|------|------|
| OK (0) | 成功 | 操作成功 |
| INVALID_ARGUMENT (3) | 参数无效 | 缺少必填字段 |
| NOT_FOUND (5) | 未找到 | Agent 不存在 |
| INTERNAL (13) | 内部错误 | 服务端异常 |
| UNAVAILABLE (14) | 服务不可用 | 连接失败 |
| DEADLINE_EXCEEDED (4) | 超时 | 操作超时 |

### 6.2 超时配置

| 操作 | 超时时间 | 说明 |
|-----|---------|------|
| 心跳 | 10s | 心跳请求超时 |
| 指标上报 | 30s | 指标上报超时 |
| 列举 Agent | 10s | ListAgents 超时 |
| 操作 Agent | 60s | OperateAgent 超时 (包含优雅停止时间) |
| 获取指标 | 30s | GetAgentMetrics 超时 |
| 状态同步 | 10s | SyncAgentStates 超时 |

---

## 7. 版本历史

| 版本 | 日期 | 变更内容 |
|-----|------|---------|
| v1.0.0 | 2025-12-09 | 初始版本，统一 AgentInfo 字段顺序 |
| v1.1.0 | 2025-12-09 | 统一 ConfigRequest/ConfigResponse/UpdateRequest/ResourceDataPoint 定义 |
| v1.2.0 | 2025-12-09 | 优化架构：移除操作后的 ListAgents 调用，添加手动同步接口 |

---

## 8. 实现验证

### 8.1 Manager 端实现

#### 8.1.1 ManagerService 服务端 (`manager/internal/grpc/server.go`)

| 方法 | 实现状态 | 请求/响应匹配 |
|-----|---------|-------------|
| RegisterNode | ✅ 已实现 | ✅ 匹配 |
| Heartbeat | ✅ 已实现 | ✅ 匹配 |
| ReportMetrics | ✅ 已实现 | ✅ 匹配 |

#### 8.1.2 DaemonService 客户端 (`manager/internal/grpc/daemon_client.go`)

| 方法 | 实现状态 | 请求/响应匹配 | 超时设置 |
|-----|---------|-------------|---------|
| ListAgents | ✅ 已实现 | ✅ 匹配 | 30s |
| OperateAgent | ✅ 已实现 | ✅ 匹配 | 90s |
| GetAgentMetrics | ✅ 已实现 | ✅ 匹配 | 30s |

### 8.2 Daemon 端实现

#### 8.2.1 DaemonService 服务端 (`daemon/internal/grpc/server.go`)

| 方法 | 实现状态 | 请求/响应匹配 |
|-----|---------|-------------|
| ListAgents | ✅ 已实现 | ✅ 匹配 |
| OperateAgent | ✅ 已实现 | ✅ 匹配 |
| GetAgentMetrics | ✅ 已实现 | ✅ 匹配 |
| SyncAgentStates | ✅ 已实现 | ✅ 匹配 |

#### 8.2.2 ManagerService 客户端 (`daemon/internal/comm/grpc_client.go`)

| 方法 | 实现状态 | 请求/响应匹配 |
|-----|---------|-------------|
| RegisterNode | ✅ 已实现 | ✅ 匹配 |
| Heartbeat | ✅ 已实现 | ✅ 匹配 |
| ReportMetrics | ✅ 已实现 | ✅ 匹配 |

#### 8.2.3 DaemonService 客户端 (`daemon/internal/grpc/manager_client.go`)

| 方法 | 实现状态 | 请求/响应匹配 |
|-----|---------|-------------|
| SyncAgentStates | ✅ 已实现 | ✅ 匹配 |

---

## 9. 附录

### 9.1 Proto 文件位置

| 模块 | 文件路径 | 用途 |
|-----|---------|------|
| Manager | `manager/pkg/proto/manager.proto` | Manager 作为 ManagerService 服务端 |
| Manager | `manager/pkg/proto/daemon/daemon.proto` | Manager 作为 DaemonService 客户端 |
| Daemon | `daemon/pkg/proto/manager.proto` | Daemon 作为 ManagerService 客户端 |
| Daemon | `daemon/pkg/proto/manager/manager.proto` | Daemon 作为 ManagerService 客户端 (备用) |
| Daemon | `daemon/pkg/proto/daemon.proto` | Daemon 作为 DaemonService 服务端 |

### 9.2 代码生成命令

```bash
# Manager 端
cd manager
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       pkg/proto/manager.proto
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       pkg/proto/daemon/daemon.proto

# Daemon 端
cd daemon
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       pkg/proto/manager.proto
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       pkg/proto/daemon.proto
```

### 9.3 连接参数配置

#### Manager 端 DaemonClient 配置

| 参数 | 值 | 说明 |
|-----|---|------|
| defaultTimeout | 30s | 默认操作超时 |
| operateAgentTimeout | 90s | Agent 操作超时 (需大于优雅停止 30s) |
| keepaliveTime | 45s | Keepalive 间隔 |
| keepaliveTimeout | 15s | Keepalive 超时 |
| maxMsgSize | 10MB | 最大消息大小 |
| initialWindowSize | 1MB | 初始窗口大小 |

#### Daemon 端 GRPCClient 配置

| 参数 | 值 | 说明 |
|-----|---|------|
| keepaliveTime | 30s | Keepalive 间隔 (需 > Manager MinTime 20s) |
| keepaliveTimeout | 10s | Keepalive 超时 |
| maxMsgSize | 10MB | 最大消息大小 |
| reconnectInterval | 5s | 重连间隔 |

### 9.4 重试策略

```json
{
  "methodConfig": [{
    "name": [{"service": "proto.DaemonService"}],
    "waitForReady": true,
    "retryPolicy": {
      "MaxAttempts": 3,
      "InitialBackoff": "0.1s",
      "MaxBackoff": "1s",
      "BackoffMultiplier": 2.0,
      "RetryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
    }
  }]
}
```

