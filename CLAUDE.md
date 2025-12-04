# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## 项目概述

**Ops Scaffold Framework** 是一个轻量级、高可用的分布式运维管理平台,用于实现对分布式主机的集中管理、监控和自动化运维。

### 核心组件

系统由三个核心组件组成:

1. **Manager (中心管理节点)**: Web 服务,提供全局管理、监控和版本更新功能
2. **Daemon (守护进程)**: 运行在每台被管主机上,负责资源采集、Agent 管理和状态上报
3. **Agent (执行进程)**: 运行在每台被管主机上,负责具体任务执行并提供 HTTP/HTTPS API

### 技术栈

| 组件 | 技术选型 | 实际版本 |
|------|----------|----------|
| 编程语言 | Go 1.21+ | Go 1.24.0 |
| Manager 后端 | Gin + GORM | Gin 1.10.0 + GORM 1.25.5 |
| Manager 前端 | React + Vite + MUI | React 18.2 + Vite 7.2 + MUI 7.3 |
| 前端状态管理 | Zustand + React Query | Zustand 5.0 + React Query 5.90 |
| 前端路由 | React Router | v7.10 |
| HTTP 客户端 | Axios | v1.13 |
| 数据库 | MySQL | MySQL 8.0+ |
| JWT 认证 | golang-jwt | v5.2.0 |
| 密码加密 | bcrypt | golang.org/x/crypto v0.43.0 |
| 配置管理 | Viper | v1.18.2 |
| 日志 | zap | go.uber.org/zap v1.26.0 |
| gRPC | gRPC + Protobuf | grpc v1.77.0 + protobuf v1.36.10 |
| 测试框架 | testify | v1.9.0 |
| 系统资源采集 | gopsutil | shirou/gopsutil v3 |
| 通信协议 | gRPC + HTTP | Manager ↔ Daemon (gRPC), API (HTTP/HTTPS) |

---

## 架构说明

### 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                     【呈现层 Presentation】                      │
│              React + MUI (Material UI) 前端应用                  │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP/HTTPS (REST API)
┌────────────────────────────▼────────────────────────────────────┐
│                     【应用层 Application】                       │
│              Manager后端 (Go + Gin + GORM)                       │
│   ┌──────────┬──────────┬──────────┬──────────┬──────────┐      │
│   │节点管理  │监控服务  │版本管理  │任务调度  │认证授权  │      │
│   └──────────┴──────────┴──────────┴──────────┴──────────┘      │
└────────────────────────────┬────────────────────────────────────┘
                             │ gRPC/HTTPS
┌────────────────────────────▼────────────────────────────────────┐
│                      【主机层 Host Layer】                       │
│  ┌─────────Host 1─────────┐    ┌─────────Host N─────────┐       │
│  │ ┌───────────────────┐  │    │ ┌───────────────────┐  │       │
│  │ │      Daemon       │  │    │ │      Daemon       │  │       │
│  │ │  (守护进程管理)   │  │    │ │  (守护进程管理)   │  │       │
│  │ └─────────┬─────────┘  │    │ └─────────┬─────────┘  │       │
│  │           │ 心跳/管理  │    │           │ 心跳/管理  │       │
│  │ ┌─────────▼─────────┐  │    │ ┌─────────▼─────────┐  │       │
│  │ │      Agent        │  │    │ │      Agent        │  │       │
│  │ │   (任务执行)      │  │    │ │   (任务执行)      │  │       │
│  │ └───────────────────┘  │    │ └───────────────────┘  │       │
│  └────────────────────────┘    └────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

### 通信协议

- **Manager ↔ Daemon**: gRPC over TLS (mTLS 双向认证)
- **Manager/Daemon ↔ Agent**: HTTP/HTTPS RESTful API (请求签名认证)
- **Daemon ↔ Agent (本地)**: Unix Domain Socket / Named Pipe (JSON-RPC 2.0)

---

## 完成状态

| 组件 | 状态 | 说明 |
|------|------|------|
| **Manager 后端** | ✅ 完成 | HTTP API + gRPC + 认证授权 + 数据库 + 自动化测试 |
| **Daemon 守护进程** | ✅ 完成 | 资源采集 + Agent管理 + gRPC通信 + 心跳上报 |
| **Web 前端** | ✅ 完成 | React + Vite + MUI + 认证 + Dashboard + 节点管理 |
| **Agent 执行进程** | ⏳ 待开发 | 任务执行 + HTTP API + 插件系统 |

**当前版本**: v0.3.0 (Manager + Daemon + Web 前端基础功能完成)

---

## 项目目录结构

### 实际目录结构

项目采用**多模块独立管理**方式，Manager 和 Daemon 各有独立的 go.mod：

```
ops-scaffold-framework/
├── manager/                  # Manager 模块(独立模块)
│   ├── cmd/manager/
│   │   └── main.go          # Manager 入口
│   ├── internal/
│   │   ├── handler/         # HTTP 处理器
│   │   ├── service/         # 业务逻辑层
│   │   ├── repository/      # 数据访问层
│   │   ├── model/           # 数据模型
│   │   ├── middleware/      # 中间件(JWT认证等)
│   │   ├── grpc/           # gRPC 服务端
│   │   ├── router/         # 路由配置
│   │   └── database/       # 数据库初始化
│   ├── pkg/
│   │   ├── proto/          # Protobuf 定义和生成代码
│   │   ├── response/       # 统一响应格式
│   │   └── logger/         # 日志工具
│   ├── configs/
│   │   ├── manager.dev.yaml    # 开发环境配置
│   │   └── manager.prod.yaml   # 生产环境配置
│   ├── test/
│   │   ├── integration/    # 集成测试
│   │   │   ├── manager_integration_test.go  # 完整测试套件
│   │   │   └── README.md   # 测试文档
│   │   └── run_tests.sh    # 自动化测试脚本
│   ├── bin/
│   │   └── manager         # 编译后的二进制文件
│   ├── go.mod              # Manager 独立依赖
│   ├── go.sum
│   ├── Makefile
│   └── .golangci.yml       # Lint 配置
│
├── daemon/                   # Daemon 模块(独立模块)
│   ├── cmd/daemon/
│   │   └── main.go          # Daemon 入口
│   ├── internal/
│   │   ├── collector/       # 资源采集器
│   │   │   ├── manager.go   # 采集器管理
│   │   │   ├── cpu.go       # CPU 采集
│   │   │   ├── memory.go    # 内存采集
│   │   │   ├── disk.go      # 磁盘采集
│   │   │   └── network.go   # 网络采集
│   │   ├── agent/          # Agent 进程管理
│   │   │   ├── manager.go   # Agent 生命周期管理
│   │   │   ├── health.go    # 健康检查
│   │   │   └── heartbeat.go # 心跳监控
│   │   ├── comm/           # 通信层
│   │   │   └── grpc_client.go  # gRPC 客户端
│   │   ├── daemon/         # Daemon 核心逻辑
│   │   │   ├── daemon.go    # 主流程
│   │   │   └── signal.go    # 信号处理
│   │   ├── config/         # 配置管理
│   │   │   └── config.go
│   │   └── logger/         # 日志
│   │       └── logger.go
│   ├── pkg/
│   │   ├── proto/          # Protobuf 定义和生成代码
│   │   └── types/          # 公共类型定义
│   ├── configs/
│   │   └── daemon.yaml     # Daemon 配置
│   ├── scripts/
│   │   ├── install.sh      # 安装脚本
│   │   └── systemd/
│   │       └── daemon.service  # Systemd 服务文件
│   ├── bin/
│   │   └── daemon          # 编译后的二进制文件
│   ├── go.mod              # Daemon 独立依赖
│   ├── go.sum
│   ├── Makefile
│   └── .golangci.yml
│
├── docs/                    # 项目文档
│   ├── api/                # API 文档
│   │   ├── README.md       # API 文档索引
│   │   └── Manager_API.md  # Manager API 完整文档
│   ├── 运维工具框架需求文档.md
│   ├── 设计文档_01_Daemon模块.md
│   ├── 设计文档_02_Agent模块.md
│   ├── 设计文档_03_Manager后端模块.md
│   └── 代码生成计划.md
│
├── .gitignore
└── README.md
```

### 模块说明

1. **Manager 模块**: 独立 Go 模块，module 名称 `github.com/bingooyong/ops-scaffold-framework/manager`
2. **Daemon 模块**: 独立 Go 模块，module 名称 `github.com/bingooyong/ops-scaffold-framework/daemon`
3. **根目录**: 不包含 go.mod，仅作为项目容器

---

## 核心功能需求

### Daemon 进程

- **资源监控**: 采集 CPU、内存、磁盘、网络等系统资源指标(默认 60 秒间隔)
- **Agent 管理**: 启动、健康检查、异常重启、资源限制监控
- **版本更新**: 下载、签名验证、备份、回滚机制
- **自更新**: Daemon 自身更新机制
- **通信**: 与 Manager 的 gRPC 连接,与 Agent 的本地 Socket 通信

### Agent 进程

- **任务执行**:
  - 脚本执行(Shell、Python、PowerShell)
  - 文件操作(上传、下载、复制、删除)
  - 服务管理(启动、停止、重启)
- **HTTP API**: 提供 RESTful API 接口(支持 HTTPS)
- **心跳上报**: 每 30 秒向 Daemon 上报状态
- **插件系统**: 支持 Go 插件动态加载

### Manager 后端

- **节点管理**: 节点注册、分组、标签、状态监控
- **监控仪表盘**: 实时指标展示、历史数据查询
- **版本发布**: 上传版本、签名、灰度发布、批量更新、回滚
- **任务调度**: 即时任务、定时任务(Cron)、任务模板
- **用户认证**: OAuth2/JWT 认证、RBAC 权限控制
- **审计日志**: 记录所有管理操作

---

## 安全设计要点

### 通信安全

- Manager ↔ Daemon: TLS 1.3 + mTLS 双向认证
- Agent API: HMAC-SHA256 请求签名 + IP 白名单
- 所有敏感通信使用加密传输

### 版本更新安全

更新包必须经过以下验证:
1. 来源校验: Manager IP 白名单或证书验证
2. 签名验证: RSA-2048/ECDSA 签名
3. 完整性校验: SHA-256 哈希验证
4. 版本检查: 新版本号 > 当前版本

### 敏感数据保护

- 配置中的敏感信息加密存储
- 日志脱敏处理
- 内存中密钥及时清除

---

## 开发指南

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Redis (Manager 组件需要)

### 构建命令

由于项目采用多模块结构，需要分别构建各组件：

```bash
# 构建 Manager
cd manager
make build
# 输出: manager/bin/manager

# 构建 Daemon
cd daemon
make build
# 输出: daemon/bin/daemon

# 或使用 go build 直接构建
cd manager
go build -o bin/manager ./cmd/manager/

cd daemon
go build -o bin/daemon ./cmd/daemon/
```

### 运行组件

```bash
# 启动 Manager (开发环境)
cd manager
./bin/manager -config configs/manager.dev.yaml
# HTTP API: http://127.0.0.1:8080
# gRPC: 127.0.0.1:9090

# 启动 Daemon (开发环境)
cd daemon
./bin/daemon -config configs/daemon.dev.yaml

# 查看帮助
./bin/manager -h
./bin/daemon -h
```

### 数据库迁移

```bash
# 初始化数据库
cd migrations
mysql -u root -p < init.sql

# 使用 GORM AutoMigrate
# 在 Manager 启动时会自动创建/更新表结构
```

---

## 性能指标

### 资源占用

- Daemon: CPU < 1% (空闲时), 内存 < 30MB
- Agent: CPU < 1% (空闲时), 内存 < 50MB
- Manager API 响应: < 200ms (P95)

### 容量要求

- 单 Manager 支持管理 1000+ 节点
- Agent 支持 10 并发任务
- 任务响应延迟 < 100ms

---

## 测试要求

### Manager 集成测试 ✅

Manager 已实现完整的自动化集成测试套件，基于 Go testify/suite 框架。

**测试覆盖**:
- Phase 0: 健康检查
- Phase 1: 认证模块 (注册、登录、获取资料、修改密码)
- Phase 2: 节点管理 (列表、统计)
- Phase 3: 错误场景 (无效凭证、无Token、重复注册)

**运行测试**:
```bash
# 方式1: 使用自动化脚本(推荐)
cd manager
./test/run_tests.sh

# 方式2: 直接运行 Go 测试
cd manager
go test -v ./test/integration/

# 方式3: 运行特定测试阶段
go test -v ./test/integration/ -run "Phase1"
```

**测试文档**: `manager/test/integration/README.md`

### 测试覆盖率目标

- Service 层: > 80% 覆盖率
- Repository 层: > 70% 覆盖率
- Handler 层: > 70% 覆盖率
- 执行器模块(Agent): > 80% 覆盖率

### 测试命令

```bash
# Manager 集成测试
cd manager
go test -v ./test/integration/

# Daemon 单元测试(待补充)
cd daemon
go test ./...

# 测试覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 测试原则

1. **自动化优先**: 所有测试必须可自动化执行，避免手动测试
2. **可重复性**: 测试脚本必须可重复运行，每次结果一致
3. **测试隔离**: 测试之间互不影响，可独立执行
4. **文档同步**: 测试代码与 API 文档保持严格一致
5. **CI/CD 就绪**: 测试脚本可直接集成到 CI/CD 流程

---

## 关键设计模式

### 分层架构 (Manager)

```
Handler Layer → Service Layer → Repository Layer → Model Layer
```

### 插件系统 (Agent)

- 支持 Go 插件 (.so) 动态加载
- 插件实现 `Plugin` 和 `Executor` 接口
- 示例: 自定义数据库执行器、监控采集器

### 进程隔离 (Daemon/Agent)

- Agent 作为独立进程运行,不是 Daemon 子进程
- Daemon 退出不影响 Agent 运行
- Agent 异常时 Daemon 自动重启,支持退避策略

---

## 故障处理

### Daemon 管理 Agent 重启策略

| 重启次数 | 等待时间 | 备注 |
|----------|----------|------|
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

## API 设计规范

### RESTful API 响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "timestamp": "2025-12-01T10:00:00Z"
}
```

### 错误码规范

- 0: 成功
- 1xxx: 客户端错误(参数、认证、权限)
- 2xxx: 业务错误(节点、任务)
- 3xxx: 版本管理错误
- 5xxx: 服务器内部错误

---

## 配置管理

### 配置文件格式

所有组件使用 YAML 格式配置文件,位于 `configs/` 目录。

### 配置优先级

命令行参数 > 环境变量 > 配置文件 > 默认值

---

## 日志规范

### 日志级别

- DEBUG: 详细调试信息
- INFO: 一般信息(默认)
- WARN: 警告信息
- ERROR: 错误信息

### 日志格式

使用 zap 结构化日志,包含时间戳、级别、调用者、消息、字段等信息。

---

## 注意事项

1. **进程独立性**: Agent 必须作为独立进程运行,不能是 Daemon 的子进程
2. **版本兼容性**: 确保 Daemon 和 Agent 版本兼容性
3. **资源限制**: Daemon 监控 Agent 资源占用,超限自动重启
4. **安全第一**: 所有更新包必须经过完整的安全验证流程
5. **优雅退出**: 所有组件支持优雅退出,确保数据完整性
6. **错误恢复**: 实现自动重试和故障恢复机制
7. **监控告警**: 关键操作失败要及时告警

---

## 文档组织规范

项目文档统一存放在 `docs/` 目录，采用以下结构：

```
docs/
├── api/                          # API 文档目录
│   ├── README.md                 # API 文档索引
│   ├── Manager_API.md            # Manager API 完整文档 ✅
│   ├── Daemon_gRPC_API.md        # Daemon gRPC API (待补充)
│   └── Agent_HTTP_API.md         # Agent HTTP API (待补充)
├── 运维工具框架需求文档.md       # 完整需求规格
├── 设计文档_01_Daemon模块.md     # Daemon 详细设计
├── 设计文档_02_Agent模块.md      # Agent 详细设计
├── 设计文档_03_Manager后端模块.md # Manager 后端详细设计
└── 代码生成计划.md               # 开发计划
```

### 文档规范

1. **API 文档**:
   - 位置: `docs/api/` 目录
   - 格式: Markdown
   - 内容: 完整的接口定义、请求/响应示例、错误码说明
   - 更新: 代码修改后必须同步更新文档

2. **测试文档**:
   - 位置: 各模块的 `test/` 目录
   - 示例: `manager/test/integration/README.md`
   - 内容: 测试用例、运行方法、环境要求

3. **配置文档**:
   - 位置: 各模块的 `configs/` 目录
   - 格式: YAML + 注释说明
   - 示例: `manager/configs/manager.dev.yaml`

### 文档一致性要求

**关键原则**: 文档与代码必须保持严格一致

- API 文档的 HTTP 状态码必须与代码一致
- 响应格式必须与 `pkg/response/response.go` 一致
- 错误码必须与代码中的定义一致
- 所有 curl 示例必须可直接运行

**验证方法**: 所有 API 文档的一致性必须通过自动化集成测试验证

```bash
cd manager
go test -v ./test/integration/
```

### 已完成文档

- ✅ Manager API 文档: `docs/api/Manager_API.md` (12个HTTP接口 + 3个gRPC接口)
  - ✅ 已通过 11 个集成测试用例验证文档一致性
  - ✅ 错误码: 0, 1001-1009, 2001-2009, 2003-2004, 2101-2102, 5001-5002
  - ✅ HTTP 状态码: 200, 201, 400, 401, 403, 404, 409, 500
- ✅ Manager 测试文档: `manager/test/integration/README.md`
- ✅ API 文档索引: `docs/api/README.md`
- ⏳ Daemon API 文档: 待补充
- ⏳ Agent API 文档: 待补充

---

## 参考文档

详细设计文档位于 `docs/` 目录:

- **需求文档**: `docs/运维工具框架需求文档.md` - 完整需求规格
- **设计文档**:
  - `docs/设计文档_01_Daemon模块.md` - Daemon 详细设计
  - `docs/设计文档_02_Agent模块.md` - Agent 详细设计
  - `docs/设计文档_03_Manager后端模块.md` - Manager 后端详细设计
- **API 文档**: `docs/api/README.md` - API 文档索引入口
