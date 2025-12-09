# Qoder Rules - Ops Scaffold Framework

## 项目概述

**Ops Scaffold Framework** 是一个轻量级、高可用的分布式运维管理平台，用于实现对分布式主机的集中管理、监控和自动化运维。

### 核心组件

1. **Manager (中心管理节点)**: Web 服务，提供全局管理、监控和版本更新功能
2. **Daemon (守护进程)**: 运行在每台被管主机上，负责资源采集、Agent 管理和状态上报
3. **Agent (执行进程)**: 运行在每台被管主机上，负责具体任务执行并提供 HTTP/HTTPS API
4. **Web (前端界面)**: React + Vite + MUI 实现的管理界面

---

## 技术栈规范

### 后端技术栈

| 技术领域 | 选型 | 版本要求 |
|---------|------|----------|
| 编程语言 | Go | 1.21+ (实际使用 1.24.0) |
| Web 框架 | Gin | 1.10.0+ |
| ORM | GORM | 1.25.5+ |
| 数据库 | MySQL | 8.0+ |
| 通信协议 | gRPC + Protobuf | grpc v1.77.0, protobuf v1.36.10 |
| 配置管理 | Viper | v1.18.2 |
| 日志 | zap | v1.26.0 |
| JWT 认证 | golang-jwt | v5.2.0 |
| 密码加密 | bcrypt | golang.org/x/crypto v0.43.0 |
| 系统监控 | gopsutil | shirou/gopsutil v3 |
| 测试框架 | testify | v1.9.0 |

### 前端技术栈

| 技术领域 | 选型 | 版本要求 |
|---------|------|----------|
| 框架 | React | 18.2+ |
| 构建工具 | Vite | 7.2+ |
| UI 组件库 | Material-UI (MUI) | 7.3+ |
| 状态管理 | Zustand | 5.0+ |
| 服务端状态 | React Query (TanStack Query) | 5.90+ |
| 路由 | React Router | v7.10+ |
| HTTP 客户端 | Axios | v1.13+ |
| TypeScript | TypeScript | 最新稳定版 |

---

## 项目结构规范

### 多模块独立管理

项目采用 **多模块独立管理** 方式，每个核心组件都有独立的 go.mod 或 package.json：

```
ops-scaffold-framework/
├── manager/              # Manager 模块 (独立 Go 模块)
│   ├── go.mod           # module: github.com/bingooyong/ops-scaffold-framework/manager
│   └── ...
├── daemon/              # Daemon 模块 (独立 Go 模块)
│   ├── go.mod           # module: github.com/bingooyong/ops-scaffold-framework/daemon
│   └── ...
├── agent/               # Agent 模块 (独立 Go 模块)
│   ├── go.mod           # module: github.com/bingooyong/ops-scaffold-framework/agent
│   └── ...
└── web/                 # Web 前端 (独立 npm 项目)
    ├── package.json
    └── ...
```

### Go 模块目录结构

每个 Go 模块遵循标准项目布局：

```
<module>/
├── cmd/<module>/        # 主程序入口
│   └── main.go
├── internal/            # 内部包 (不对外暴露)
│   ├── handler/         # HTTP 处理器
│   ├── service/         # 业务逻辑层
│   ├── repository/      # 数据访问层
│   ├── model/           # 数据模型
│   ├── middleware/      # 中间件
│   └── config/          # 配置管理
├── pkg/                 # 公共包 (可对外暴露)
│   ├── proto/           # Protobuf 定义
│   ├── types/           # 公共类型
│   └── utils/           # 工具函数
├── configs/             # 配置文件
│   ├── <module>.dev.yaml
│   └── <module>.yaml
├── test/                # 测试文件
│   ├── integration/     # 集成测试
│   └── unit/            # 单元测试
├── scripts/             # 脚本文件
├── bin/                 # 编译输出目录
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 前端目录结构

```
web/
├── src/
│   ├── api/             # API 客户端
│   ├── components/      # 可复用组件
│   ├── pages/           # 页面组件
│   ├── hooks/           # 自定义 Hooks
│   ├── stores/          # Zustand 状态管理
│   ├── types/           # TypeScript 类型定义
│   ├── utils/           # 工具函数
│   ├── router/          # 路由配置
│   └── theme/           # 主题配置
├── public/              # 静态资源
├── configs/             # 配置文件
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

---

## 代码规范

### Go 代码规范

#### 1. 包命名
- 使用小写单词，避免下划线和驼峰
- 简短且有意义，如 `handler`, `service`, `repository`

#### 2. 文件命名
- 使用小写 + 下划线，如 `user_service.go`, `node_handler.go`
- 测试文件以 `_test.go` 结尾

#### 3. 分层架构
严格遵循三层架构：

```
Handler Layer (HTTP) → Service Layer (业务逻辑) → Repository Layer (数据访问)
```

- **Handler**: 处理 HTTP 请求/响应，参数验证，调用 Service
- **Service**: 业务逻辑处理，事务管理，调用 Repository
- **Repository**: 数据库操作，只负责 CRUD

#### 4. 错误处理
- 使用自定义错误类型 (`pkg/errors`)
- 错误必须包含错误码和消息
- 不要忽略错误，必须处理或向上传递

```go
if err != nil {
    return nil, errors.NewInternalError("操作失败", err)
}
```

#### 5. 日志规范
- 使用 zap 结构化日志
- 日志级别: DEBUG, INFO, WARN, ERROR
- 关键操作必须记录日志

```go
logger.Info("用户登录成功", zap.String("username", username))
```

#### 6. API 响应格式
统一使用以下响应格式：

```go
type Response struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data,omitempty"`
    Timestamp string      `json:"timestamp"`
}
```

#### 7. 错误码规范
- 0: 成功
- 1xxx: 客户端错误 (参数、认证、权限)
- 2xxx: 业务错误 (节点、任务)
- 3xxx: 版本管理错误
- 5xxx: 服务器内部错误

### TypeScript/React 代码规范

#### 1. 组件规范
- 使用函数组件 + Hooks
- 组件文件名使用 PascalCase，如 `UserList.tsx`
- 每个组件一个文件

#### 2. 类型定义
- 所有 API 响应必须定义 TypeScript 接口
- 使用 `types/` 目录统一管理类型
- 避免使用 `any`

#### 3. 状态管理
- 客户端状态使用 Zustand
- 服务端状态使用 React Query
- 避免 prop drilling，使用状态管理

#### 4. API 调用
- 使用 Axios 封装的 API 客户端
- 统一的错误处理和 token 注入
- 使用 React Query 管理请求状态

---

## 开发工作流

### 1. 功能开发流程

```
需求分析 → 接口设计 → 数据库设计 → 后端开发 → 前端开发 → 测试 → 文档
```

### 2. 代码提交规范

使用 Conventional Commits：

- `feat:` 新功能
- `fix:` 修复 bug
- `docs:` 文档更新
- `style:` 代码格式调整
- `refactor:` 重构
- `test:` 测试相关
- `chore:` 构建/工具链相关

示例：
```
feat(manager): 添加节点分组管理功能
fix(daemon): 修复资源采集内存泄漏问题
docs(api): 更新 Manager API 文档
```

### 3. 分支管理

- `main`: 生产环境分支
- `develop`: 开发分支
- `feature/*`: 功能分支
- `bugfix/*`: 修复分支
- `release/*`: 发布分支

---

## 测试规范

### 测试覆盖率目标

- Service 层: > 80%
- Repository 层: > 70%
- Handler 层: > 70%
- 关键业务逻辑: 100%

### 测试类型

#### 1. 单元测试
- 文件命名: `*_test.go`
- 使用 testify/assert 断言
- Mock 外部依赖

#### 2. 集成测试
- 位置: `test/integration/`
- 测试真实数据库连接
- 测试 HTTP API 端到端流程

#### 3. 测试原则
- **自动化优先**: 所有测试必须可自动化执行
- **可重复性**: 测试结果一致，避免随机失败
- **测试隔离**: 测试之间互不影响
- **文档同步**: 测试用例与 API 文档保持一致

### 运行测试

```bash
# Manager 集成测试
cd manager
./test/run_tests.sh

# 单元测试
go test ./...

# 测试覆盖率
go test -cover ./...
```

---

## 配置管理规范

### 配置文件格式

- 统一使用 YAML 格式
- 位置: 各模块的 `configs/` 目录
- 命名: `<module>.<env>.yaml`

### 配置优先级

```
命令行参数 > 环境变量 > 配置文件 > 默认值
```

### 敏感信息处理

- 不要将敏感信息提交到代码库
- 使用环境变量或密钥管理系统
- 配置文件中的密码必须加密存储

---

## 安全规范

### 1. 通信安全

- Manager ↔ Daemon: TLS 1.3 + mTLS 双向认证
- Agent API: HMAC-SHA256 请求签名 + IP 白名单
- 所有敏感通信使用加密传输

### 2. 认证授权

- 使用 JWT Token 认证
- Token 有效期不超过 24 小时
- 实现 RBAC 权限控制

### 3. 密码安全

- 使用 bcrypt 加密存储
- 密码强度验证 (最少 8 位，包含字母数字)
- 密码错误次数限制

### 4. 输入验证

- 所有用户输入必须验证
- 防止 SQL 注入、XSS 攻击
- 文件上传必须检查类型和大小

### 5. 日志脱敏

- 不记录敏感信息 (密码、token)
- 必要时对敏感字段脱敏处理

---

## 性能规范

### 资源占用限制

- Daemon: CPU < 1% (空闲), 内存 < 30MB
- Agent: CPU < 1% (空闲), 内存 < 50MB
- Manager: 根据节点数量动态调整

### API 性能要求

- API 响应时间: < 200ms (P95)
- 任务响应延迟: < 100ms
- 数据库查询: < 50ms

### 容量要求

- 单 Manager 支持管理 1000+ 节点
- Agent 支持 10 并发任务
- 监控数据保留 30 天

---

## 文档规范

### 文档组织

```
docs/
├── api/                 # API 文档
│   ├── README.md        # API 文档索引
│   ├── Manager_API.md   # Manager API 完整文档
│   ├── Daemon_gRPC_API.md
│   └── Agent_HTTP_API.md
├── 运维工具框架需求文档.md
├── 设计文档_01_Daemon模块.md
├── 设计文档_02_Agent模块.md
├── 设计文档_03_Manager模块.md
└── 设计文档_04_Web前端模块.md
```

### 文档一致性要求

**关键原则**: 文档与代码必须保持严格一致

- API 文档的 HTTP 状态码必须与代码一致
- 响应格式必须与 `pkg/response/response.go` 一致
- 错误码必须与代码中的定义一致
- 所有 curl 示例必须可直接运行
- **验证方法**: 通过自动化集成测试验证文档一致性

### 文档更新规则

- 代码修改后必须同步更新文档
- API 变更必须更新 API 文档
- 新增功能必须更新 README
- 配置变更必须更新配置说明

---

## 数据库规范

### 表命名规范

- 使用小写 + 下划线
- 复数形式，如 `users`, `nodes`, `agents`

### 字段命名规范

- 使用小写 + 下划线
- 主键统一命名为 `id`
- 时间字段: `created_at`, `updated_at`, `deleted_at`

### GORM 模型规范

```go
type User struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
    Username  string         `gorm:"uniqueIndex;not null" json:"username"`
}
```

### 索引规范

- 频繁查询的字段添加索引
- 唯一性约束使用 `uniqueIndex`
- 软删除字段添加索引

---

## 构建与部署

### 构建命令

```bash
# Manager
cd manager
make build              # 输出: bin/manager

# Daemon
cd daemon
make build              # 输出: bin/daemon

# Agent
cd agent
make build              # 输出: bin/agent

# Web
cd web
npm run build           # 输出: dist/
```

### 运行环境

- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- Redis 6.0+ (Manager 组件需要)

---

## 故障处理规范

### Agent 重启策略

| 重启次数 | 等待时间 | 操作 |
|---------|---------|------|
| 第1次 | 立即执行 | - |
| 第2-3次 | 10秒 | - |
| 第4-5次 | 30秒 | - |
| 超过5次 | 60秒 | 上报告警 |

### 版本更新流程

```
接收更新指令 → 下载更新包 → 校验签名 → 校验哈希 →
备份当前版本 → 停止服务 → 替换文件 → 启动新版本 →
健康检查(30秒) → 成功则清理备份/失败则回滚
```

---

## 关键设计原则

### 1. 进程独立性
- Agent 必须作为独立进程运行，不能是 Daemon 的子进程
- Daemon 退出不影响 Agent 运行

### 2. 优雅退出
- 所有组件支持优雅退出
- 确保数据完整性和资源释放

### 3. 自动恢复
- 实现自动重试和故障恢复机制
- Agent 异常时 Daemon 自动重启

### 4. 监控告警
- 关键操作失败要及时告警
- 资源超限要及时通知

### 5. 版本兼容性
- 确保 Daemon 和 Agent 版本兼容
- 实现向后兼容的 API 设计

---

## 开发注意事项

1. **多模块开发**: 注意各模块独立的 go.mod，不要跨模块直接引用
2. **配置管理**: 开发环境使用 `*.dev.yaml`，生产环境使用 `*.yaml`
3. **日志记录**: 关键操作必须记录日志，便于问题排查
4. **错误处理**: 不要忽略错误，必须妥善处理
5. **测试先行**: 新功能开发前先编写测试用例
6. **文档同步**: 代码修改后立即更新相关文档
7. **安全第一**: 所有涉及安全的操作必须经过严格验证

---

## 参考文档

- **需求文档**: `docs/运维工具框架需求文档.md`
- **设计文档**: `docs/设计文档_*.md`
- **API 文档**: `docs/api/README.md`
- **测试文档**: `<module>/test/integration/README.md`

---

## 版本历史

- v0.3.0: Manager + Daemon + Web 前端基础功能完成
- v0.2.0: Daemon 多 Agent 管理完成
- v0.1.0: Manager 后端基础功能完成
