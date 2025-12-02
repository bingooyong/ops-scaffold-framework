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

| 组件 | 技术选型 |
|------|----------|
| 编程语言 | Go 1.21+ |
| Manager 后端 | Gin 1.10.1 + GORM 1.31.0 |
| Manager 前端 | React 18+ + MUI 5+ |
| 数据库 | MySQL 8.0+ |
| 缓存 | Redis |
| 任务调度 | robfig/cron v3.0.1 |
| 系统资源采集 | shirou/gopsutil |
| 日志 | uber-go/zap |
| 通信协议 | gRPC (Manager ↔ Daemon), HTTP/HTTPS (Manager/Daemon ↔ Agent) |

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

## 项目目录结构

### 预期目录结构

```
ops-scaffold-framework/
├── cmd/                      # 各组件入口
│   ├── daemon/
│   │   └── main.go
│   ├── agent/
│   │   └── main.go
│   └── manager/
│       └── main.go
├── internal/                 # 内部代码(不对外暴露)
│   ├── daemon/              # Daemon 模块
│   │   ├── collector/       # 资源采集器
│   │   ├── agent/          # Agent 进程管理
│   │   ├── updater/        # 版本更新
│   │   └── comm/           # 通信层
│   ├── agent/              # Agent 模块
│   │   ├── server/         # HTTP 服务
│   │   ├── executor/       # 任务执行器
│   │   └── heartbeat/      # 心跳模块
│   └── manager/            # Manager 模块
│       ├── handler/        # HTTP 处理器
│       ├── service/        # 业务逻辑
│       ├── repository/     # 数据访问层
│       ├── model/          # 数据模型
│       ├── middleware/     # 中间件
│       └── grpc/          # gRPC 服务
├── pkg/                     # 可对外暴露的包
│   ├── proto/              # Protobuf 定义
│   ├── utils/              # 工具函数
│   └── errors/             # 错误定义
├── configs/                 # 配置文件
│   ├── daemon.yaml
│   ├── agent.yaml
│   └── manager.yaml
├── migrations/              # 数据库迁移脚本
├── scripts/                 # 脚本文件
│   ├── install.sh
│   └── build.sh
├── docs/                    # 文档
│   ├── 运维工具框架需求文档.md
│   ├── 设计文档_01_Daemon模块.md
│   ├── 设计文档_02_Agent模块.md
│   └── 设计文档_03_Manager后端模块.md
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

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

```bash
# 构建所有组件
make build

# 构建特定组件
make build-daemon
make build-agent
make build-manager

# 运行测试
make test

# 代码检查
make lint
```

### 运行组件

```bash
# 启动 Daemon
./bin/daemon -config configs/daemon.yaml

# 启动 Agent
./bin/agent -config configs/agent.yaml

# 启动 Manager
./bin/manager -config configs/manager.yaml
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

### 单元测试

- Service 层: > 80% 覆盖率
- Repository 层: > 70% 覆盖率
- 执行器模块: > 80% 覆盖率

### 测试命令

```bash
# 运行所有测试
go test ./...

# 测试覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

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

## 参考文档

详细设计文档位于 `docs/` 目录:

- `运维工具框架需求文档.md`: 完整需求规格
- `设计文档_01_Daemon模块.md`: Daemon 详细设计
- `设计文档_02_Agent模块.md`: Agent 详细设计
- `设计文档_03_Manager后端模块.md`: Manager 后端详细设计
