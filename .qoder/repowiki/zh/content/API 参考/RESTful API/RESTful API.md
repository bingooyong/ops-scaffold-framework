# RESTful API

<cite>
**本文档引用文件**   
- [Manager_API.md](file://docs/api/Manager_API.md)
- [auth.go](file://manager/internal/handler/auth.go)
- [node.go](file://manager/internal/handler/node.go)
- [agent.go](file://manager/internal/handler/agent.go)
- [metrics.go](file://manager/internal/handler/metrics.go)
- [user.go](file://manager/internal/model/user.go)
- [node.go](file://manager/internal/model/node.go)
- [agent.go](file://manager/internal/model/agent.go)
- [metrics.go](file://manager/internal/model/metrics.go)
- [auth.ts](file://web/src/api/auth.ts)
- [nodes.ts](file://web/src/api/nodes.ts)
- [agents.ts](file://web/src/api/agents.ts)
- [metrics.ts](file://web/src/api/metrics.ts)
- [response.go](file://manager/pkg/response/response.go)
</cite>

## 目录
1. [认证](#1-认证)
2. [响应格式](#2-响应格式)
3. [错误码](#3-错误码)
4. [API 接口](#4-api-接口)
   - [认证接口](#41-认证接口)
   - [节点管理接口](#42-节点管理接口)
   - [用户管理接口](#43-用户管理接口)
   - [健康检查](#44-健康检查)
   - [Agent 管理接口](#45-agent-管理接口)
   - [监控指标接口](#46-监控指标接口)
5. [API版本控制与分页机制](#5-api版本控制与分页机制)

## 1. 认证

Manager 模块使用 JWT (JSON Web Token) 进行身份验证。

### 认证流程

1. 用户通过 `/api/v1/auth/login` 接口登录，获取 JWT Token
2. 后续请求在 HTTP Header 中携带 Token:
   ```
   Authorization: Bearer <token>
   ```
3. Token 有效期为配置文件中指定的时间（默认 24 小时）
4. Token 过期后需要重新登录

### 权限级别

- **普通用户 (user)**: 可以查看节点信息、执行任务
- **管理员 (admin)**: 拥有所有权限，包括用户管理、节点删除等

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#1-认证)
- [auth.go](file://manager/internal/handler/auth.go#L12-L25)

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

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#2-响应格式)
- [response.go](file://manager/pkg/response/response.go#L11-L31)

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
| 3001 | Agent不存在 | 404 |
| 3002 | 操作失败 | 500 |
| 3003 | 无效的操作类型 | 400 |
| 3004 | 功能未实现 | 501 |
| 5001 | 服务器内部错误 | 500 |
| 5002 | 数据库错误 | 500 |

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#3-错误码)

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

**TypeScript前端调用示例**:
```typescript
import { register } from '@/api/auth';

const data = {
  username: 'testuser',
  password: 'password123',
  email: 'test@example.com'
};

register(data).then(response => {
  console.log('注册成功:', response.data.user);
}).catch(error => {
  console.error('注册失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#411-用户注册)
- [auth.go](file://manager/internal/handler/auth.go#L26-L69)
- [auth.ts](file://web/src/api/auth.ts#L26-L28)

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

**TypeScript前端调用示例**:
```typescript
import { login } from '@/api/auth';

const data = {
  username: 'testuser',
  password: 'password123'
};

login(data).then(response => {
  const { token, user } = response.data;
  // 存储token
  localStorage.setItem('token', token);
  console.log('登录成功:', user);
}).catch(error => {
  console.error('登录失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#412-用户登录)
- [auth.go](file://manager/internal/handler/auth.go#L33-L96)
- [auth.ts](file://web/src/api/auth.ts#L19-L21)

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

**TypeScript前端调用示例**:
```typescript
import { getProfile } from '@/api/auth';

getProfile().then(response => {
  console.log('用户信息:', response.data.user);
}).catch(error => {
  console.error('获取用户信息失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#413-获取当前用户信息)
- [auth.go](file://manager/internal/handler/auth.go#L98-L122)
- [auth.ts](file://web/src/api/auth.ts#L33-L35)

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

**TypeScript前端调用示例**:
```typescript
import { changePassword } from '@/api/auth';

const data = {
  old_password: 'password123',
  new_password: 'newpassword456'
};

changePassword(data).then(response => {
  console.log('密码修改成功');
}).catch(error => {
  console.error('密码修改失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#414-修改密码)
- [auth.go](file://manager/internal/handler/auth.go#L124-L150)
- [auth.ts](file://web/src/api/auth.ts#L40-L42)

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

**TypeScript前端调用示例**:
```typescript
import { getNodes } from '@/api/nodes';

// 获取所有节点
getNodes({}).then(response => {
  console.log('节点列表:', response.data.list);
});

// 获取在线节点
getNodes({ status: 'online' }).then(response => {
  console.log('在线节点:', response.data.list);
});

// 分页查询
getNodes({ page: 1, page_size: 10 }).then(response => {
  console.log('分页节点:', response.data);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#421-获取节点列表)
- [node.go](file://manager/internal/handler/node.go#L36-L68)
- [nodes.ts](file://web/src/api/nodes.ts#L11-L18)

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

**TypeScript前端调用示例**:
```typescript
import { getNode } from '@/api/nodes';

getNode('1').then(response => {
  console.log('节点详情:', response.data.node);
}).catch(error => {
  console.error('获取节点详情失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#422-获取节点详情)
- [node.go](file://manager/internal/handler/node.go#L71-L92)
- [nodes.ts](file://web/src/api/nodes.ts#L24-L27)

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

**TypeScript前端调用示例**:
```typescript
import { getNodeStatistics } from '@/api/nodes';

getNodeStatistics().then(response => {
  const { total, online, offline } = response.data.statistics;
  console.log(`节点统计: 总数${total}, 在线${online}, 离线${offline}`);
}).catch(error => {
  console.error('获取节点统计失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#423-获取节点统计信息)
- [node.go](file://manager/internal/handler/node.go#L116-L157)
- [nodes.ts](file://web/src/api/nodes.ts#L42-L46)

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

**TypeScript前端调用示例**:
```typescript
import { deleteNode } from '@/api/nodes';

deleteNode(1).then(response => {
  console.log('节点删除成功');
}).catch(error => {
  console.error('删除节点失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#424-删除节点)
- [node.go](file://manager/internal/handler/node.go#L94-L114)
- [nodes.ts](file://web/src/api/nodes.ts#L33-L36)

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

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#431-获取用户列表)
- [auth.go](file://manager/internal/handler/auth.go#L152-L180)

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

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#432-禁用用户)
- [auth.go](file://manager/internal/handler/auth.go#L182-L202)

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

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#433-启用用户)
- [auth.go](file://manager/internal/handler/auth.go#L204-L224)

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

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#441-健康检查)

### 4.5 Agent 管理接口

Agent 管理接口提供节点下 Agent 的查询、操作和日志查看功能。所有接口需要 JWT Token 认证，使用 `Authorization: Bearer <token>` header。

#### 4.5.1 获取节点下的所有Agent

**接口**: `GET /api/v1/nodes/:node_id/agents`

**描述**: 获取指定节点下的所有 Agent 列表

**权限**: 需要认证

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "agents": [
      {
        "id": 1,
        "node_id": "node-1",
        "agent_id": "agent-1",
        "type": "filebeat",
        "version": "1.0.0",
        "status": "running",
        "config": "{}",
        "pid": 12345,
        "last_heartbeat": "2025-01-27T10:00:00Z",
        "last_sync_time": "2025-01-27T10:00:00Z",
        "created_at": "2025-01-27T09:00:00Z",
        "updated_at": "2025-01-27T10:00:00Z"
      }
    ],
    "count": 1
  },
  "timestamp": "2025-01-27T10:00:00Z"
}
```

**响应字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| agents | array | Agent 列表 |
| agents[].id | int | Agent 数据库ID |
| agents[].node_id | string | 节点ID |
| agents[].agent_id | string | Agent 唯一标识符 |
| agents[].type | string | Agent 类型(filebeat/telegraf等) |
| agents[].version | string | Agent 版本号 |
| agents[].status | string | 运行状态(running/stopped/error/starting/stopping) |
| agents[].config | string | Agent 配置(JSON格式) |
| agents[].pid | int | 进程ID(0表示未运行) |
| agents[].last_heartbeat | string | 最后心跳时间(ISO8601格式) |
| agents[].last_sync_time | string | 最后同步时间(ISO8601格式) |
| agents[].created_at | string | 创建时间(ISO8601格式) |
| agents[].updated_at | string | 更新时间(ISO8601格式) |
| count | int | Agent 数量 |

**错误响应**:
- 400: 节点ID为空 (错误码 1001)
- 401: Token 无效或过期 (错误码 1002)
- 404: 节点不存在 (错误码 2001)
- 500: 数据库错误 (错误码 5002)

**示例 curl**:
```bash
curl -X GET "http://localhost:8080/api/v1/nodes/node-1/agents" \
  -H "Authorization: Bearer <token>"
```

**TypeScript前端调用示例**:
```typescript
import { listAgents } from '@/api/agents';

listAgents('node-1').then(response => {
  console.log('Agent列表:', response.data.agents);
  console.log('Agent数量:', response.data.count);
}).catch(error => {
  console.error('获取Agent列表失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#451-获取节点下的所有agent)
- [agent.go](file://manager/internal/handler/agent.go#L32-L60)
- [agents.ts](file://web/src/api/agents.ts#L17-L21)

#### 4.5.2 操作Agent(启动/停止/重启)

**接口**: `POST /api/v1/nodes/:node_id/agents/:agent_id/operate`

**描述**: 操作指定节点下的 Agent(启动/停止/重启)

**权限**: 需要认证

**请求头**:
```
Authorization: Bearer <token>
Content-Type: application/json
```

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |
| agent_id | string | 是 | Agent 唯一标识符 |

**请求体**:
```json
{
  "operation": "start"
}
```

**请求参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| operation | string | 是 | 操作类型，可选值：`start`(启动)、`stop`(停止)、`restart`(重启) |

**成功响应** (200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "操作成功"
  },
  "timestamp": "2025-01-27T10:00:00Z"
}
```

**错误响应**:
- 400: 参数错误(节点ID/Agent ID为空,或无效的操作类型) (错误码 1001)
- 401: Token 无效或过期 (错误码 1002)
- 404: 节点不存在 (错误码 2001)
- 404: Agent不存在 (错误码 3001)
- 500: gRPC连接错误或操作失败 (错误码 5001)

**示例 curl**:
```bash
# 启动Agent
curl -X POST "http://localhost:8080/api/v1/nodes/node-1/agents/agent-1/operate" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "start"
  }'

# 停止Agent
curl -X POST "http://localhost:8080/api/v1/nodes/node-1/agents/agent-1/operate" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "stop"
  }'

# 重启Agent
curl -X POST "http://localhost:8080/api/v1/nodes/node-1/agents/agent-1/operate" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "restart"
  }'
```

**TypeScript前端调用示例**:
```typescript
import { operateAgent } from '@/api/agents';

// 启动Agent
operateAgent('node-1', 'agent-1', 'start').then(response => {
  console.log('Agent启动成功');
}).catch(error => {
  console.error('Agent启动失败:', error);
});

// 停止Agent
operateAgent('node-1', 'agent-1', 'stop').then(response => {
  console.log('Agent停止成功');
}).catch(error => {
  console.error('Agent停止失败:', error);
});

// 重启Agent
operateAgent('node-1', 'agent-1', 'restart').then(response => {
  console.log('Agent重启成功');
}).catch(error => {
  console.error('Agent重启失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#452-操作agent启动停止重启)
- [agent.go](file://manager/internal/handler/agent.go#L62-L103)
- [agents.ts](file://web/src/api/agents.ts#L29-L39)

#### 4.5.3 获取Agent日志

**接口**: `GET /api/v1/nodes/:node_id/agents/:agent_id/logs`

**描述**: 获取指定 Agent 的日志

**权限**: 需要认证

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | 节点唯一标识符 |
| agent_id | string | 是 | Agent 唯一标识符 |

**查询参数**:
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| lines | int | 否 | 100 | 日志行数，最大 1000 |

**成功响应** (200):
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
  },
  "timestamp": "2025-01-27T10:00:00Z"
}
```

**响应字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| logs | array | 日志行列表(字符串数组) |
| count | int | 日志行数 |

**错误响应**:
- 400: 参数错误(节点ID/Agent ID为空,或无效的行数) (错误码 1001)
- 401: Token 无效或过期 (错误码 1002)
- 404: 节点不存在 (错误码 2001)
- 404: Agent不存在 (错误码 3001)
- 501: 功能未实现(当前状态) (错误码 3004)

**说明**:
- 当前此功能尚未实现，调用时会返回 501 错误码
- 未来实现后，将返回实际的日志内容

**示例 curl**:
```bash
# 获取默认100行日志
curl -X GET "http://localhost:8080/api/v1/nodes/node-1/agents/agent-1/logs" \
  -H "Authorization: Bearer <token>"

# 获取指定行数日志(最多1000行)
curl -X GET "http://localhost:8080/api/v1/nodes/node-1/agents/agent-1/logs?lines=200" \
  -H "Authorization: Bearer <token>"
```

**TypeScript前端调用示例**:
```typescript
import { getAgentLogs } from '@/api/agents';

// 获取默认100行日志
getAgentLogs('node-1', 'agent-1').then(response => {
  console.log('Agent日志:', response.data.logs);
  console.log('日志行数:', response.data.count);
}).catch(error => {
  console.error('获取Agent日志失败:', error);
});

// 获取200行日志
getAgentLogs('node-1', 'agent-1', 200).then(response => {
  console.log('Agent日志(200行):', response.data.logs);
}).catch(error => {
  console.error('获取Agent日志失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#453-获取agent日志)
- [agent.go](file://manager/internal/handler/agent.go#L105-L161)
- [agents.ts](file://web/src/api/agents.ts#L47-L57)

### 4.6 监控指标接口

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

**TypeScript前端调用示例**:
```typescript
import { getLatestMetrics } from '@/api/metrics';

getLatestMetrics('daemon-001').then(response => {
  const metrics = response.data;
  
  if (metrics.cpu) {
    console.log('CPU使用率:', metrics.cpu.values.usage_percent);
  }
  
  if (metrics.memory) {
    console.log('内存使用率:', metrics.memory.values.usage_percent);
  }
  
  if (metrics.disk) {
    console.log('磁盘使用率:', metrics.disk.values.usage_percent);
  }
  
  if (metrics.network) {
    console.log('网络接收字节:', metrics.network.values.rx_bytes);
  }
}).catch(error => {
  console.error('获取最新指标失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#451-获取节点最新指标)
- [metrics.go](file://manager/internal/handler/metrics.go#L28-L54)
- [metrics.ts](file://web/src/api/metrics.ts#L11-L15)

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

**TypeScript前端调用示例**:
```typescript
import { getMetricsHistory } from '@/api/metrics';

// 查询CPU历史数据
getMetricsHistory('daemon-001', 'cpu', {
  start_time: '2025-12-03T00:00:00Z',
  end_time: '2025-12-04T00:00:00Z'
}).then(response => {
  console.log('CPU历史数据:', response.data);
}).catch(error => {
  console.error('获取历史指标失败:', error);
});

// 查询内存历史数据
getMetricsHistory('daemon-001', 'memory', {
  start_time: '2025-11-27T00:00:00Z',
  end_time: '2025-12-04T00:00:00Z'
}).then(response => {
  console.log('内存历史数据:', response.data);
}).catch(error => {
  console.error('获取历史指标失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#452-获取历史指标数据)
- [metrics.go](file://manager/internal/handler/metrics.go#L56-L129)
- [metrics.ts](file://web/src/api/metrics.ts#L20-L28)

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

**TypeScript前端调用示例**:
```typescript
import { getMetricsSummary } from '@/api/metrics';
import { TimeRange } from '@/types';

// 默认查询（最近24小时）
getMetricsSummary('daemon-001').then(response => {
  const summary = response.data;
  
  if (summary.cpu) {
    console.log(`CPU: 最小${summary.cpu.min}%, 最大${summary.cpu.max}%, 平均${summary.cpu.avg}%, 最新${summary.cpu.latest}%`);
  }
  
  if (summary.memory) {
    console.log(`内存: 最小${summary.memory.min}%, 最大${summary.memory.max}%, 平均${summary.memory.avg}%, 最新${summary.memory.latest}%`);
  }
  
  if (summary.disk) {
    console.log(`磁盘: 最小${summary.disk.min}%, 最大${summary.disk.max}%, 平均${summary.disk.avg}%, 最新${summary.disk.latest}%`);
  }
}).catch(error => {
  console.error('获取指标统计摘要失败:', error);
});

// 自定义时间范围查询
const timeRange: TimeRange = {
  startTime: new Date('2025-12-01T00:00:00Z'),
  endTime: new Date('2025-12-04T00:00:00Z')
};

getMetricsSummary('daemon-001', timeRange).then(response => {
  console.log('自定义时间范围统计:', response.data);
}).catch(error => {
  console.error('获取指标统计摘要失败:', error);
});
```

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#453-获取指标统计摘要)
- [metrics.go](file://manager/internal/handler/metrics.go#L131-L191)
- [metrics.ts](file://web/src/api/metrics.ts#L33-L47)

## 5. API版本控制与分页机制

### API版本控制策略

Manager API 采用 URL 路径版本控制策略，所有接口均以 `/api/v1/` 作为前缀。这种策略具有以下优点：

1. **清晰明确**: 版本信息直接体现在URL中，便于识别和管理
2. **向后兼容**: 不同版本可以同时存在，确保旧客户端的兼容性
3. **易于部署**: 可以独立部署不同版本的API

当需要进行不兼容的API变更时，将创建新的版本（如 `/api/v2/`），同时保持旧版本的可用性，给予客户端充分的迁移时间。

### 分页机制

所有返回列表数据的接口均支持分页机制，采用标准的分页参数和响应格式：

**分页参数**:
- `page`: 页码，从1开始，默认为1
- `page_size`: 每页数量，范围1-100，默认为20

**分页响应格式**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [...],
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

**分页字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| page | int | 当前页码 |
| page_size | int | 每页数量 |
| total | int64 | 总记录数 |
| total_pages | int | 总页数 |

**最佳实践建议**:
1. 客户端应始终处理分页响应，即使当前数据量较少
2. 对于大数据集，建议使用较小的 page_size 以提高响应速度
3. 可以结合筛选参数（如 status、search）来减少返回数据量
4. 当 total_pages 较大时，考虑实现滚动加载而非页码导航

**Section sources**
- [Manager_API.md](file://docs/api/Manager_API.md#2-响应格式)
- [response.go](file://manager/pkg/response/response.go#L19-L31)
- [node.go](file://manager/internal/handler/node.go#L38-L47)