# Manager API 文档

**版本**: v1.0
**基础 URL**: `http://localhost:8080`
**协议**: HTTP/HTTPS

---

## 目录

1. [认证](#1-认证)
2. [响应格式](#2-响应格式)
3. [错误码](#3-错误码)
4. [API 接口](#4-api-接口)
   - [认证接口](#41-认证接口)
   - [节点管理接口](#42-节点管理接口)
   - [用户管理接口](#43-用户管理接口)
   - [健康检查](#44-健康检查)
5. [gRPC 接口](#5-grpc-接口)

---

## 1. 认证

Manager 使用 JWT (JSON Web Token) 进行身份验证。

### 认证流程

1. 用户通过 `/api/v1/auth/login` 接口登录,获取 JWT Token
2. 后续请求在 HTTP Header 中携带 Token:
   ```
   Authorization: Bearer <token>
   ```
3. Token 有效期为配置文件中指定的时间(默认 24 小时)
4. Token 过期后需要重新登录

### 权限级别

- **普通用户 (user)**: 可以查看节点信息、执行任务
- **管理员 (admin)**: 拥有所有权限,包括用户管理、节点删除等

---

## 2. 响应格式

所有 API 接口统一返回以下格式的 JSON 响应:

### 成功响应

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

### 错误响应

```json
{
  "code": 1001,
  "message": "invalid parameter",
  "details": "username is required",
  "timestamp": "2025-12-03T10:00:00Z"
}
```

### 分页数据响应

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

---

## 3. 错误码

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
| 5001 | 服务器内部错误 | 500 |
| 5002 | 数据库错误 | 500 |

---

## 4. API 接口

### 4.1 认证接口

#### 4.1.1 用户注册

**接口**: `POST /api/v1/auth/register`

**描述**: 注册新用户

**权限**: 无需认证

**请求体**:
```json
{
  "username": "testuser",
  "password": "password123",
  "email": "test@example.com"
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名,3-20个字符 |
| password | string | 是 | 密码,至少6个字符 |
| email | string | 是 | 邮箱地址 |

**成功响应** (201):
```json
{
  "code": 0,
  "message": "user registered successfully",
  "data": {
    "user": {
      "id": 1,
      "username": "testuser",
      "email": "test@example.com",
      "role": "user",
      "status": "active",
      "created_at": "2025-12-03T10:00:00Z"
    }
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 400: 参数错误
- 409: 用户已存在 (错误码 2009)
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123",
    "email": "test@example.com"
  }'
```

---

#### 4.1.2 用户登录

**接口**: `POST /api/v1/auth/login`

**描述**: 用户登录获取 JWT Token

**权限**: 无需认证

**请求体**:
```json
{
  "username": "testuser",
  "password": "password123"
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "testuser",
      "email": "test@example.com",
      "role": "user",
      "status": "active"
    }
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: 用户名或密码错误 (错误码 1009)
- 403: 用户已被禁用 (错误码 2003)
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

---

#### 4.1.3 获取当前用户信息

**接口**: `GET /api/v1/auth/profile`

**描述**: 获取当前登录用户的个人信息

**权限**: 需要认证

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": 1,
      "username": "testuser",
      "email": "test@example.com",
      "role": "user",
      "status": "active",
      "created_at": "2025-12-03T10:00:00Z",
      "updated_at": "2025-12-03T10:00:00Z",
      "last_login_at": "2025-12-03T10:00:00Z",
      "last_login_ip": "192.168.1.100"
    }
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: Token 无效或已过期
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X GET http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer <token>"
```

---

#### 4.1.4 修改密码

**接口**: `POST /api/v1/auth/change-password`

**描述**: 修改当前用户密码

**权限**: 需要认证

**请求头**:
```
Authorization: Bearer <token>
```

**请求体**:
```json
{
  "old_password": "password123",
  "new_password": "newpassword456"
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | string | 是 | 原密码 |
| new_password | string | 是 | 新密码,至少6个字符 |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "password changed successfully",
  "data": null,
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 400: 参数错误
- 401: 原密码错误或 Token 无效
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "password123",
    "new_password": "newpassword456"
  }'
```

---

### 4.2 节点管理接口

#### 4.2.1 获取节点列表

**接口**: `GET /api/v1/nodes`

**描述**: 获取节点列表(支持分页和筛选)

**权限**: 需要认证

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量(最大100) |
| status | string | 否 | - | 状态筛选: online, offline |
| search | string | 否 | - | 搜索关键字(节点名称或IP) |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "node_id": "47f3e1bd-e200-400f-bb9f-0b5330d98f5d",
        "hostname": "node-001",
        "ip": "192.168.1.100",
        "os": "Linux",
        "arch": "amd64",
        "status": "online",
        "labels": {
          "env": "production",
          "region": "us-west"
        },
        "daemon_version": "0.1.0",
        "agent_version": "0.1.0",
        "last_heartbeat_at": "2025-12-03T10:00:00Z",
        "created_at": "2025-12-03T09:00:00Z",
        "updated_at": "2025-12-03T10:00:00Z"
      }
    ],
    "page_info": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_pages": 1
    }
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: Token 无效或已过期
- 500: 服务器内部错误

**示例 curl**:
```bash
# 获取第1页,每页20条
curl -X GET "http://localhost:8080/api/v1/nodes?page=1&page_size=20" \
  -H "Authorization: Bearer <token>"

# 筛选在线节点
curl -X GET "http://localhost:8080/api/v1/nodes?status=online" \
  -H "Authorization: Bearer <token>"

# 搜索节点
curl -X GET "http://localhost:8080/api/v1/nodes?search=node-001" \
  -H "Authorization: Bearer <token>"
```

---

#### 4.2.2 获取节点详情

**接口**: `GET /api/v1/nodes/:id`

**描述**: 获取指定节点的详细信息

**权限**: 需要认证

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 节点数据库ID |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "node": {
      "id": 1,
      "node_id": "47f3e1bd-e200-400f-bb9f-0b5330d98f5d",
      "hostname": "node-001",
      "ip": "192.168.1.100",
      "os": "Linux",
      "arch": "amd64",
      "status": "online",
      "labels": {
        "env": "production",
        "region": "us-west"
      },
      "daemon_version": "0.1.0",
      "agent_version": "0.1.0",
      "last_heartbeat_at": "2025-12-03T10:00:00Z",
      "created_at": "2025-12-03T09:00:00Z",
      "updated_at": "2025-12-03T10:00:00Z"
    }
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: Token 无效或已过期
- 404: 节点不存在
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X GET http://localhost:8080/api/v1/nodes/1 \
  -H "Authorization: Bearer <token>"
```

---

#### 4.2.3 获取节点统计信息

**接口**: `GET /api/v1/nodes/statistics`

**描述**: 获取节点统计信息(总数、在线数、离线数)

**权限**: 需要认证

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "statistics": {
      "total": 100,
      "online": 85,
      "offline": 15
    }
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: Token 无效或已过期
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X GET http://localhost:8080/api/v1/nodes/statistics \
  -H "Authorization: Bearer <token>"
```

---

#### 4.2.4 删除节点

**接口**: `DELETE /api/v1/admin/nodes/:id`

**描述**: 删除指定节点(仅管理员)

**权限**: 需要认证 + 管理员权限

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 节点数据库ID |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "node deleted successfully",
  "data": null,
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: Token 无效或已过期
- 403: 无管理员权限
- 404: 节点不存在
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X DELETE http://localhost:8080/api/v1/admin/nodes/1 \
  -H "Authorization: Bearer <token>"
```

---

### 4.3 用户管理接口

#### 4.3.1 获取用户列表

**接口**: `GET /api/v1/admin/users`

**描述**: 获取用户列表(仅管理员)

**权限**: 需要认证 + 管理员权限

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量 |
| role | string | 否 | - | 角色筛选: admin, user |
| status | string | 否 | - | 状态筛选: active, disabled |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "username": "admin",
        "email": "admin@example.com",
        "role": "admin",
        "status": "active",
        "created_at": "2025-12-03T09:00:00Z",
        "updated_at": "2025-12-03T10:00:00Z",
        "last_login_at": "2025-12-03T10:00:00Z",
        "last_login_ip": "192.168.1.100"
      }
    ],
    "page_info": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_pages": 1
    }
  },
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: Token 无效或已过期
- 403: 无管理员权限
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X GET "http://localhost:8080/api/v1/admin/users?page=1&page_size=20" \
  -H "Authorization: Bearer <token>"
```

---

#### 4.3.2 禁用用户

**接口**: `POST /api/v1/admin/users/:id/disable`

**描述**: 禁用指定用户(仅管理员)

**权限**: 需要认证 + 管理员权限

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 用户ID |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "user disabled successfully",
  "data": null,
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: Token 无效或已过期
- 403: 无管理员权限
- 404: 用户不存在
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X POST http://localhost:8080/api/v1/admin/users/2/disable \
  -H "Authorization: Bearer <token>"
```

---

#### 4.3.3 启用用户

**接口**: `POST /api/v1/admin/users/:id/enable`

**描述**: 启用指定用户(仅管理员)

**权限**: 需要认证 + 管理员权限

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 用户ID |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "user enabled successfully",
  "data": null,
  "timestamp": "2025-12-03T10:00:00Z"
}
```

**错误响应**:
- 401: Token 无效或已过期
- 403: 无管理员权限
- 404: 用户不存在
- 500: 服务器内部错误

**示例 curl**:
```bash
curl -X POST http://localhost:8080/api/v1/admin/users/2/enable \
  -H "Authorization: Bearer <token>"
```

---

### 4.4 健康检查

#### 4.4.1 健康检查

**接口**: `GET /health`

**描述**: 检查服务健康状态

**权限**: 无需认证

**成功响应** (200):
```json
{
  "status": "ok",
  "time": "2025-12-03T10:00:00Z"
}
```

**示例 curl**:
```bash
curl -X GET http://localhost:8080/health
```

---

### 4.5 监控指标接口

监控指标接口提供节点资源使用情况的查询功能，支持实时指标、历史趋势和统计聚合查询。所有接口需要 JWT Token 认证，使用 `Authorization: Bearer <token>` header。

**通用参数说明**:
- 时间参数使用 ISO8601 格式 `YYYY-MM-DDTHH:MM:SSZ`，支持 UTC 或带时区偏移
- 最大查询时间范围：30 天
- 指标类型支持：`cpu`、`memory`、`disk`、`network`

**通用错误处理**:
- 401: Token 无效或过期
- 404: 节点不存在
- 400: 参数错误（时间格式错误、时间范围超限、无效的指标类型）
- 500: 服务器内部错误

---

#### 4.5.1 获取节点最新指标

**接口**: `GET /api/v1/metrics/nodes/:node_id/latest`

**描述**: 获取指定节点所有类型（cpu/memory/disk/network）的最新指标数据，用于实时监控卡片展示。

**权限**: 需要认证

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符，例如 `daemon-001` |

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "cpu": {
      "id": 123,
      "node_id": "daemon-001",
      "type": "cpu",
      "timestamp": "2025-12-04T12:00:00Z",
      "values": {
        "usage_percent": 45.2,
        "cores": 8,
        "model": "Intel Core i7"
      }
    },
    "memory": {
      "id": 124,
      "node_id": "daemon-001",
      "type": "memory",
      "timestamp": "2025-12-04T12:00:00Z",
      "values": {
        "usage_percent": 60.5,
        "used_bytes": 8589934592,
        "total_bytes": 17179869184
      }
    },
    "disk": {
      "id": 125,
      "node_id": "daemon-001",
      "type": "disk",
      "timestamp": "2025-12-04T12:00:00Z",
      "values": {
        "usage_percent": 75.3,
        "used_bytes": 161061273600,
        "total_bytes": 214748364800
      }
    },
    "network": {
      "id": 126,
      "node_id": "daemon-001",
      "type": "network",
      "timestamp": "2025-12-04T12:00:00Z",
      "values": {
        "rx_bytes": 1024000,
        "tx_bytes": 2048000,
        "rx_packets": 1000,
        "tx_packets": 2000
      }
    }
  },
  "timestamp": "2025-12-04T12:00:00Z"
}
```

**说明**:
- 如果某个指标类型无数据，对应的 key 值为 `null`
- `values` 字段包含该指标类型的具体数值，不同指标类型包含不同的字段

**错误响应**:
- 401: Token 无效或过期 (错误码 1002)
- 404: 节点不存在 (错误码 2003)
- 500: 服务器内部错误

**示例 curl**:
```bash
# 设置 Token 环境变量
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 获取节点最新指标
curl -X GET "http://localhost:8080/api/v1/metrics/nodes/daemon-001/latest" \
  -H "Authorization: Bearer $TOKEN"
```

---

#### 4.5.2 获取历史指标数据

**接口**: `GET /api/v1/metrics/nodes/:node_id/:type/history`

**描述**: 查询指定节点和指标类型的历史数据，支持自定义时间范围，根据查询范围自动应用采样策略以优化返回数据量。

**权限**: 需要认证

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |
| type | string | 是 | 指标类型，可选值：`cpu`、`memory`、`disk`、`network` |

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_time | string | 是 | 开始时间，ISO8601 格式，例如 `2025-12-01T00:00:00Z` |
| end_time | string | 是 | 结束时间，ISO8601 格式 |

**采样策略**:
系统根据查询时间范围自动选择采样间隔，确保返回数据点数量适中（约 300-400 点）：

| 时间范围 | 采样间隔 | 说明 |
|---------|---------|------|
| ≤ 15 分钟 | 60 秒（原始数据） | 返回原始数据，保证实时性 |
| ≤ 1 小时 | 60 秒（原始数据） | 返回原始数据，保证实时性 |
| ≤ 1 天 | 5 分钟 | 聚合为 5 分钟桶，约 288 个数据点 |
| ≤ 7 天 | 30 分钟 | 聚合为 30 分钟桶，约 336 个数据点 |
| ≤ 30 天 | 2 小时 | 聚合为 2 小时桶，约 360 个数据点 |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 100,
      "node_id": "daemon-001",
      "type": "cpu",
      "timestamp": "2025-12-04T00:00:00Z",
      "values": {
        "usage_percent": 45.2
      }
    },
    {
      "id": 101,
      "node_id": "daemon-001",
      "type": "cpu",
      "timestamp": "2025-12-04T00:05:00Z",
      "values": {
        "usage_percent": 46.8
      }
    }
  ],
  "timestamp": "2025-12-04T12:00:00Z"
}
```

**错误响应**:
- 400: 时间范围超过 30 天 (错误码 1001)
- 400: 参数格式错误，时间格式不正确 (错误码 1001)
- 400: 无效的指标类型 (错误码 1001)
- 401: Token 无效或过期 (错误码 1002)
- 404: 节点不存在 (错误码 2003)
- 500: 服务器内部错误

**示例 curl**:
```bash
# 查询 1 天历史数据（自动使用 5 分钟聚合）
curl -X GET "http://localhost:8080/api/v1/metrics/nodes/daemon-001/cpu/history?start_time=2025-12-03T00:00:00Z&end_time=2025-12-04T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN"

# 查询 7 天历史数据（自动使用 30 分钟聚合）
curl -X GET "http://localhost:8080/api/v1/metrics/nodes/daemon-001/memory/history?start_time=2025-11-27T00:00:00Z&end_time=2025-12-04T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN"

# 查询 30 天历史数据（自动使用 2 小时聚合）
curl -X GET "http://localhost:8080/api/v1/metrics/nodes/daemon-001/disk/history?start_time=2025-11-04T00:00:00Z&end_time=2025-12-04T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN"
```

---

#### 4.5.3 获取指标统计摘要

**接口**: `GET /api/v1/metrics/nodes/:node_id/summary`

**描述**: 获取指定节点在时间范围内的资源使用统计（min/max/avg/latest），用于快速了解资源使用趋势。

**权限**: 需要认证

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_time | string | 否 | 开始时间，ISO8601 格式，默认 24 小时前 |
| end_time | string | 否 | 结束时间，ISO8601 格式，默认当前时间 |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "cpu": {
      "min": 10.5,
      "max": 85.2,
      "avg": 45.3,
      "latest": 50.1
    },
    "memory": {
      "min": 20.0,
      "max": 75.8,
      "avg": 55.2,
      "latest": 60.5
    },
    "disk": {
      "min": 30.0,
      "max": 80.5,
      "avg": 65.3,
      "latest": 75.3
    },
    "network": {
      "min": null,
      "max": null,
      "avg": null,
      "latest": null
    }
  },
  "timestamp": "2025-12-04T12:00:00Z"
}
```

**说明**:
- `min`、`max`、`avg` 为时间范围内的统计值（浮点数，单位 % 或字节）
- `latest` 为最新指标值
- 如果某个指标类型无数据，对应的 key 值为 `null`
- 如果时间范围内无数据但存在最新值，统计字段为 `null`，`latest` 字段有值

**错误响应**:
- 400: 时间范围超过 30 天 (错误码 1001)
- 400: 参数格式错误 (错误码 1001)
- 401: Token 无效或过期 (错误码 1002)
- 404: 节点不存在 (错误码 2003)
- 500: 服务器内部错误

**示例 curl**:
```bash
# 默认查询（最近 24 小时）
curl -X GET "http://localhost:8080/api/v1/metrics/nodes/daemon-001/summary" \
  -H "Authorization: Bearer $TOKEN"

# 自定义时间范围
curl -X GET "http://localhost:8080/api/v1/metrics/nodes/daemon-001/summary?start_time=2025-12-01T00:00:00Z&end_time=2025-12-04T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 5. gRPC 接口

Manager 提供 gRPC 服务供 Daemon 调用,默认监听端口 `9090`。

### 5.1 节点注册

**方法**: `RegisterNode`

**描述**: Daemon 启动时调用此接口向 Manager 注册节点信息

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

**示例**:
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

---

### 5.2 心跳上报

**方法**: `Heartbeat`

**描述**: Daemon 定期调用此接口上报心跳,维持在线状态

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

**示例**:
```go
req := &pb.HeartbeatRequest{
    NodeId:    "47f3e1bd-e200-400f-bb9f-0b5330d98f5d",
    Timestamp: time.Now().Unix(),
}

resp, err := client.Heartbeat(ctx, req)
if err != nil {
    log.Fatal(err)
}

if resp.Success {
    fmt.Println("心跳上报成功")
}
```

---

### 5.3 指标上报

**方法**: `ReportMetrics`

**描述**: Daemon 定期调用此接口上报系统资源指标

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

**示例**:
```go
metrics := []*pb.MetricData{
    {
        Type:      "cpu",
        Timestamp: time.Now().Unix(),
        Values: map[string]float64{
            "usage_percent": 45.5,
            "cores":         4,
        },
    },
    {
        Type:      "memory",
        Timestamp: time.Now().Unix(),
        Values: map[string]float64{
            "total_mb":     16384,
            "used_mb":      10240,
            "used_percent": 62.5,
        },
    },
}

req := &pb.ReportMetricsRequest{
    NodeId:  "47f3e1bd-e200-400f-bb9f-0b5330d98f5d",
    Metrics: metrics,
}

resp, err := client.ReportMetrics(ctx, req)
if err != nil {
    log.Fatal(err)
}

if resp.Success {
    fmt.Println("指标上报成功")
}
```

---

## 附录

### A. 数据模型

#### User (用户)
```json
{
  "id": 1,
  "username": "admin",
  "email": "admin@example.com",
  "role": "admin",           // admin 或 user
  "status": "active",        // active 或 disabled
  "created_at": "2025-12-03T09:00:00Z",
  "updated_at": "2025-12-03T10:00:00Z",
  "last_login_at": "2025-12-03T10:00:00Z",
  "last_login_ip": "192.168.1.100"
}
```

#### Node (节点)
```json
{
  "id": 1,
  "node_id": "47f3e1bd-e200-400f-bb9f-0b5330d98f5d",
  "hostname": "node-001",
  "ip": "192.168.1.100",
  "os": "Linux",
  "arch": "amd64",
  "status": "online",        // online 或 offline
  "labels": {
    "env": "production",
    "region": "us-west"
  },
  "daemon_version": "0.1.0",
  "agent_version": "0.1.0",
  "last_heartbeat_at": "2025-12-03T10:00:00Z",
  "created_at": "2025-12-03T09:00:00Z",
  "updated_at": "2025-12-03T10:00:00Z"
}
```

### B. 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| MANAGER_HOST | Manager HTTP 服务监听地址 | 0.0.0.0 |
| MANAGER_PORT | Manager HTTP 服务监听端口 | 8080 |
| MANAGER_GRPC_PORT | Manager gRPC 服务监听端口 | 9090 |
| MANAGER_DB_DSN | MySQL 数据库连接字符串 | - |
| MANAGER_JWT_SECRET | JWT 签名密钥 | - |

### C. 配置文件示例

```yaml
# configs/manager.dev.yaml
server:
  host: 127.0.0.1
  port: 8080
  mode: debug

database:
  dsn: "root:root@tcp(127.0.0.1:3306)/ops_manager_dev?charset=utf8mb4&parseTime=True&loc=Local"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

grpc:
  host: 0.0.0.0
  port: 9090
  tls:
    enabled: false

jwt:
  secret: "your-secret-key-change-in-production"
  expire_time: 24h
  issuer: "ops-manager"

log:
  level: debug
  output_path: "logs/manager.log"
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true
```

---

**文档更新日期**: 2025-12-03
**维护者**: Ops Scaffold Framework Team
