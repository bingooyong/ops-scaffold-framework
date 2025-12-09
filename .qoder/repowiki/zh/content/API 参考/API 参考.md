# API 参考

<cite>
**本文档中引用的文件**
- [Manager_API.md](file://docs/api/Manager_API.md)
- [README.md](file://docs/api/README.md)
- [auth.go](file://manager/internal/handler/auth.go)
- [node.go](file://manager/internal/handler/node.go)
- [agent.go](file://manager/internal/handler/agent.go)
- [metrics.go](file://manager/internal/handler/metrics.go)
- [server.go](file://manager/internal/grpc/server.go)
- [manager.proto](file://manager/pkg/proto/manager.proto)
- [daemon.proto](file://daemon/pkg/proto/daemon.proto)
- [helper.go](file://manager/internal/handler/helper.go)
- [auth.go](file://manager/internal/middleware/auth.go)
- [response.go](file://manager/pkg/response/response.go)
- [daemon_client.go](file://manager/internal/grpc/daemon_client.go)
</cite>

## 目录
1. [简介](#简介)
2. [RESTful API](#restful-api)
   - [认证](#认证)
   - [响应格式](#响应格式)
   - [错误码](#错误码)
   - [节点管理接口](#节点管理接口)
   - [用户管理接口](#用户管理接口)
   - [Agent管理接口](#agent管理接口)
   - [监控指标接口](#监控指标接口)
   - [健康检查](#健康检查)
3. [gRPC API](#grpc-api)
   - [Manager与Daemon之间的gRPC API](#manager与daemon之间的grpc-api)
   - [Daemon与Agent之间的gRPC API](#daemon与agent之间的grpc-api)
4. [API版本控制与错误码规范](#api版本控制与错误码规范)
5. [使用示例](#使用示例)

## 简介
本API参考文档详细描述了运维工具框架中所有公开的接口。文档分为两大部分：RESTful API和gRPC API。RESTful API主要由Manager模块提供，用于节点管理、任务调度、版本发布和用户认证。gRPC API定义了Manager与Daemon之间、Daemon与Agent之间的通信协议。所有API设计遵循一致的错误处理和认证机制，确保系统的安全性和可靠性。

## RESTful API

### 认证
Manager模块使用JWT（JSON Web Token）进行身份验证。认证流程如下：
1. 用户通过`/api/v1/auth/login`接口登录，获取JWT Token
2. 后续请求在HTTP Header中携带Token：`Authorization: Bearer <token>`
3. Token有效期为配置文件中指定的时间（默认24小时）
4. Token过期后需要重新登录

权限级别包括：
- **普通用户 (user)**: 可以查看节点信息、执行任务
- **管理员 (admin)**: 拥有所有权限，包括用户管理、节点删除等

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#1-认证)
- [auth.go](file://manager/internal/handler/auth.go#L71-L96)
- [auth.go](file://manager/internal/middleware/auth.go#L12-L49)

### 响应格式
所有API接口统一返回JSON格式的响应。

#### 成功响应
```json
{
  "code": 0,
  "message": "success",
  "data": {
    // 具体数据
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

#### 错误响应
```json
{
  "code": 1001,
  "message": "invalid parameter",
  "details": "username is required",
  "timestamp": "2025-12-03T10:00:00Z"
}
```

#### 分页数据响应
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      // 数据列表
    ],
    "page_info": {
      "page": 1,
      "page_size": 20,
      "total": 100,
      "total_pages": 5
    }
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#2-响应格式)
- [response.go](file://manager/pkg/response/response.go#L11-L31)

### 错误码
| 错误码 | 说明 | HTTP 状态码 |
|--------|------|-------------|
| 0 | 成功 | 200 |
| 1001 | 无效的参数 | 400 |
| 1002 | 认证失败 | 401 |
| 1003 | 无权限 | 403 |
| 1004 | 资源不存在 | 404 |
| 1005 | Token 无效或过期 | 401 |
| 1009 | 用户名或密码错误 | 401 |
| 2001 | 用户不存在 | 404 |
| 2003 | 节点不存在 | 404 |
| 2009 | 用户已存在 | 409 |
| 2004 | 密码错误 | 401 |
| 2101 | 节点不存在 | 404 |
| 2102 | 节点已存在 | 400 |
| 3001 | Agent不存在 | 404 |
| 3002 | 操作失败 | 500 |
| 3003 | 无效的操作类型 | 400 |
| 3004 | 功能未实现 | 501 |
| 5001 | 服务器内部错误 | 500 |
| 5002 | 数据库错误 | 500 |

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#3-错误码)
- [response.go](file://manager/pkg/response/response.go#L53-L101)

### 节点管理接口
节点管理接口提供节点的增删改查功能，所有接口需要JWT Token认证。

#### 获取节点列表
**接口**: `GET /api/v1/nodes`  
**描述**: 获取节点列表（支持分页和筛选）  
**权限**: 需要认证  
**请求头**: `Authorization: Bearer <token>`  

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量（最大100） |
| status | string | 否 | - | 状态筛选：online, offline |
| search | string | 否 | - | 搜索关键字（节点名称或IP） |

#### 获取节点详情
**接口**: `GET /api/v1/nodes/:id`  
**描述**: 获取指定节点的详细信息  
**权限**: 需要认证  
**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 节点数据库ID |

#### 获取节点统计信息
**接口**: `GET /api/v1/nodes/statistics`  
**描述**: 获取节点统计信息（总数、在线数、离线数）  
**权限**: 需要认证

#### 删除节点
**接口**: `DELETE /api/v1/admin/nodes/:id`  
**描述**: 删除指定节点（仅管理员）  
**权限**: 需要认证 + 管理员权限  
**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 节点数据库ID |

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#42-节点管理接口)
- [node.go](file://manager/internal/handler/node.go#L36-L157)
- [helper.go](file://manager/internal/handler/helper.go#L9-L47)

### 用户管理接口
用户管理接口提供用户信息的查询和管理功能，仅管理员可访问。

#### 获取用户列表
**接口**: `GET /api/v1/admin/users`  
**描述**: 获取用户列表（仅管理员）  
**权限**: 需要认证 + 管理员权限  
**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量 |
| role | string | 否 | - | 角色筛选：admin, user |
| status | string | 否 | - | 状态筛选：active, disabled |

#### 禁用用户
**接口**: `POST /api/v1/admin/users/:id/disable`  
**描述**: 禁用指定用户（仅管理员）  
**权限**: 需要认证 + 管理员权限  
**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 用户ID |

#### 启用用户
**接口**: `POST /api/v1/admin/users/:id/enable`  
**描述**: 启用指定用户（仅管理员）  
**权限**: 需要认证 + 管理员权限  
**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 用户ID |

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#43-用户管理接口)
- [auth.go](file://manager/internal/handler/auth.go#L152-L224)

### Agent管理接口
Agent管理接口提供节点下Agent的查询、操作和日志查看功能。

#### 获取节点下的所有Agent
**接口**: `GET /api/v1/nodes/:node_id/agents`  
**描述**: 获取指定节点下的所有Agent列表  
**权限**: 需要认证  
**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |

#### 操作Agent(启动/停止/重启)
**接口**: `POST /api/v1/nodes/:node_id/agents/:agent_id/operate`  
**描述**: 操作指定节点下的Agent（启动/停止/重启）  
**权限**: 需要认证  
**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |
| agent_id | string | 是 | Agent唯一标识符 |

**请求体**:
```json
{
  "operation": "start"
}
```

#### 获取Agent日志
**接口**: `GET /api/v1/nodes/:node_id/agents/:agent_id/logs`  
**描述**: 获取指定Agent的日志  
**权限**: 需要认证  
**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| lines | int | 否 | 100 | 日志行数，最大1000 |

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#45-agent-管理接口)
- [agent.go](file://manager/internal/handler/agent.go#L34-L161)

### 监控指标接口
监控指标接口提供节点资源使用情况的查询功能。

#### 获取节点最新指标
**接口**: `GET /api/v1/metrics/nodes/:node_id/latest`  
**描述**: 获取指定节点所有类型的最新指标数据  
**权限**: 需要认证  
**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |

#### 获取历史指标数据
**接口**: `GET /api/v1/metrics/nodes/:node_id/:type/history`  
**描述**: 查询指定节点和指标类型的历史数据  
**权限**: 需要认证  
**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |
| type | string | 是 | 指标类型：cpu, memory, disk, network |

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_time | string | 是 | 开始时间，ISO8601格式 |
| end_time | string | 是 | 结束时间，ISO8601格式 |

#### 获取指标统计摘要
**接口**: `GET /api/v1/metrics/nodes/:node_id/summary`  
**描述**: 获取指定节点在时间范围内的资源使用统计  
**权限**: 需要认证  
**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_time | string | 否 | 开始时间，ISO8601格式，默认24小时前 |
| end_time | string | 否 | 结束时间，ISO8601格式，默认当前时间 |

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#46-监控指标接口)
- [metrics.go](file://manager/internal/handler/metrics.go#L28-L210)

### 健康检查
健康检查接口用于检查服务健康状态。

#### 健康检查
**接口**: `GET /health`  
**描述**: 检查服务健康状态  
**权限**: 无需认证  
**成功响应**:
```json
{
  "status": "ok",
  "time": "2025-12-03T10:00:00Z"
}
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#44-健康检查)

## gRPC API

### Manager与Daemon之间的gRPC API
Manager提供gRPC服务供Daemon调用，默认监听端口9090。

#### 节点注册
**方法**: `RegisterNode`  
**描述**: Daemon启动时调用此接口向Manager注册节点信息

**请求消息** (RegisterNodeRequest):
```protobuf
message RegisterNodeRequest {
  string node_id = 1;           // 节点唯一ID (UUID)
  string hostname = 2;          // 主机名
  string ip = 3;                // IP地址
  string os = 4;                // 操作系统
  string arch = 5;              // CPU架构
  map<string, string> labels = 6;  // 节点标签
  string daemon_version = 7;    // Daemon版本
  string agent_version = 8;     // Agent版本
}
```

**响应消息** (RegisterNodeResponse):
```protobuf
message RegisterNodeResponse {
  bool success = 1;             // 是否成功
  string message = 2;           // 响应消息
}
```

#### 心跳上报
**方法**: `Heartbeat`  
**描述**: Daemon定期调用此接口上报心跳，维持在线状态

**请求消息** (HeartbeatRequest):
```protobuf
message HeartbeatRequest {
  string node_id = 1;           // 节点ID
  int64 timestamp = 2;          // 时间戳(Unix时间戳,秒)
}
```

**响应消息** (HeartbeatResponse):
```protobuf
message HeartbeatResponse {
  bool success = 1;             // 是否成功
  string message = 2;           // 响应消息
}
```

#### 指标上报
**方法**: `ReportMetrics`  
**描述**: Daemon定期调用此接口上报系统资源指标

**请求消息** (ReportMetricsRequest):
```protobuf
message MetricData {
  string type = 1;              // 指标类型: cpu, memory, disk, network
  int64 timestamp = 2;          // 采集时间戳
  map<string, double> values = 3;  // 指标值
}

message ReportMetricsRequest {
  string node_id = 1;           // 节点ID
  repeated MetricData metrics = 2;  // 指标数据列表
}
```

**响应消息** (ReportMetricsResponse):
```protobuf
message ReportMetricsResponse {
  bool success = 1;             // 是否成功
  string message = 2;           // 响应消息
}
```

**Diagram sources**
- [manager.proto](file://manager/pkg/proto/manager.proto#L7-L67)
- [server.go](file://manager/internal/grpc/server.go#L34-L144)

### Daemon与Agent之间的gRPC API
Daemon提供gRPC服务供Agent调用，用于Agent管理。

#### 节点注册
**方法**: `Register`  
**描述**: Agent启动时调用此接口向Daemon注册

#### 心跳上报
**方法**: `Heartbeat`  
**描述**: Agent定期调用此接口上报心跳

#### 指标上报
**方法**: `ReportMetrics`  
**描述**: Agent定期调用此接口上报指标数据

#### 获取配置
**方法**: `GetConfig`  
**描述**: Agent调用此接口获取配置

#### 推送更新
**方法**: `PushUpdate`  
**描述**: Manager通过Daemon向Agent推送更新

#### 列举所有Agent
**方法**: `ListAgents`  
**描述**: 列举所有Agent

#### 操作Agent
**方法**: `OperateAgent`  
**描述**: 操作Agent（启动/停止/重启）

**Diagram sources**
- [daemon.proto](file://daemon/pkg/proto/daemon.proto#L7-L183)

## API版本控制与错误码规范
### 版本管理
- **当前版本**: v1
- **版本路径**: `/api/v1/*`
- **向后兼容**: 同一大版本内（如v1.x），保证向后兼容
- **新功能**: 通过添加新字段或新接口实现
- **废弃警告**: 废弃的API会在下个大版本前至少保留6个月

### 错误码规范
所有API使用统一的错误码体系，确保客户端能够一致地处理错误。错误码分为以下几类：
- **1xxx**: 客户端错误（参数错误、认证失败等）
- **2xxx**: 业务逻辑错误（资源不存在、状态冲突等）
- **3xxx**: 操作相关错误（操作失败、功能未实现等）
- **5xxx**: 服务器内部错误（数据库错误、内部服务错误等）

**Section sources**
- [README.md](file://docs/api/README.md#103-版本兼容性承诺)
- [Manager_API.md](file://docs/api/Manager_API.md#3-错误码)

## 使用示例
### REST API调用示例
#### 使用curl调用登录接口
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

#### 使用curl获取节点列表
```bash
curl -X GET "http://localhost:8080/api/v1/nodes?page=1&page_size=20" \
  -H "Authorization: Bearer <token>"
```

### gRPC API调用示例
#### Go语言调用节点注册
```go
req := &pb.RegisterNodeRequest{
    NodeId:        "47f3e1bd-e200-400f-bb9f-0b5330d98f5d",
    Hostname:      "node-001",
    Ip:            "192.168.1.100",
    Os:            "Linux",
    Arch:          "amd64",
    Labels:        map[string]string{"env": "production"},
    DaemonVersion: "0.1.0",
    AgentVersion:  "0.1.0",
}

resp, err := client.RegisterNode(ctx, req)
if err != nil {
    log.Fatal(err)
}

if resp.Success {
    fmt.Println("节点注册成功")
}
```

#### TypeScript调用获取节点列表
```typescript
import axios from 'axios';

// 设置JWT Token
const token = 'your-jwt-token';

// 获取节点列表
axios.get('http://localhost:8080/api/v1/nodes', {
  headers: {
    'Authorization': `Bearer ${token}`
  },
  params: {
    page: 1,
    page_size: 20
  }
}).then(response => {
  console.log('节点列表:', response.data);
}).catch(error => {
  console.error('获取节点列表失败:', error.response?.data);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#示例-curl)
- [README.md](file://docs/api/README.md#3-集成到项目)