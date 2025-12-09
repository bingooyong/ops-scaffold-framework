# Ops Scaffold Framework

[![Go Version](https://img.shields.io/badge/Go-1.24.0-blue.svg)](https://golang.org)
[![React Version](https://img.shields.io/badge/React-18.2-blue.svg)](https://reactjs.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Active-success.svg)]()

一个轻量级、高可用的分布式运维管理平台,用于实现对分布式主机的集中管理、监控和自动化运维。

---

## 📖 项目简介

**Ops Scaffold Framework** 是一个现代化的分布式运维管理平台,旨在解决企业在管理大规模服务器集群时面临的挑战:

- **统一管理**: 通过中心化的 Manager 节点,统一管理所有主机上的 Daemon 和 Agent
- **多 Agent 支持**: 在单个主机上同时管理多个第三方 Agent(如 Filebeat、Telegraf、Node Exporter 等)
- **自动化运维**: 支持自动化部署、配置管理、版本更新、故障恢复
- **实时监控**: 实时采集和展示主机资源指标、Agent 运行状态
- **可扩展性**: 插件化设计,易于扩展新功能和集成第三方工具

**核心解决的问题**:
- 多主机管理复杂,需要逐台登录操作
- 第三方 Agent 管理分散,缺乏统一的监控和控制
- 配置管理混乱,缺少版本控制和自动化手段
- 故障发现和恢复不及时,缺少自动化运维能力

**目标用户群体**:
- 运维工程师: 需要管理大规模服务器集群
- DevOps 团队: 需要自动化运维和持续交付能力
- 系统管理员: 需要集中化的监控和管理工具
- 中小型企业: 需要轻量级、易于部署的运维平台

---

## ✨ 核心功能

### 节点管理
- ✅ **节点注册**: 自动注册和发现新节点
- ✅ **心跳监控**: 实时监控节点在线状态(60秒心跳间隔)
- ✅ **状态同步**: 自动同步节点状态到 Manager
- ✅ **节点分组**: 支持按标签、区域等维度对节点分组

### Agent 管理
- ✅ **多 Agent 并发管理**: 支持在单个节点上同时管理多个 Agent 实例(Filebeat、Telegraf、Node Exporter 等)
- ✅ **生命周期管理**: 启动、停止、重启 Agent,支持批量操作
- ✅ **健康检查**: 自动健康检查(进程检查、HTTP 端点检查、心跳检查)
- ✅ **自动恢复**: Agent 异常退出或心跳超时时自动重启
- ✅ **资源监控**: 监控 Agent 的 CPU、内存使用情况,超过阈值时告警
- ✅ **配置管理**: 集中管理 Agent 配置,支持配置热更新
- ✅ **日志查看**: 通过 Web 界面或 API 查看 Agent 运行日志

### 监控告警
- ✅ **资源指标采集**: 采集 CPU、内存、磁盘、网络等系统资源指标(60秒间隔)
- ✅ **实时监控**: 实时展示指标数据,支持历史数据查询
- ✅ **告警规则**: 配置告警规则,触发条件时自动告警
- ⏳ **通知集成**: 支持邮件、Webhook 等告警通知方式(待开发)

### 任务调度
- ⏳ **任务创建**: 创建即时任务和定时任务(Cron)(待开发)
- ⏳ **任务执行**: 在指定节点上执行脚本、命令等(待开发)
- ⏳ **历史记录**: 查看任务执行历史和结果(待开发)

### 版本管理
- ⏳ **版本发布**: 上传新版本,支持 Agent 和 Daemon 更新(待开发)
- ⏳ **签名验证**: 更新包签名验证,确保安全性(待开发)
- ⏳ **灰度发布**: 支持灰度发布和批量更新(待开发)
- ⏳ **回滚机制**: 更新失败时自动回滚到旧版本(待开发)

### 用户认证
- ✅ **JWT 认证**: 基于 JWT 的用户认证机制
- ✅ **用户管理**: 用户注册、登录、密码修改
- ⏳ **权限管理**: RBAC 权限控制(待完善)
- ⏳ **审计日志**: 记录所有管理操作(待完善)

---

## 🏗️ 系统架构

### 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                     【呈现层 Presentation】                      │
│              React + MUI (Material UI) 前端应用                  │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP/HTTPS (REST API + JWT Auth)
┌────────────────────────────▼────────────────────────────────────┐
│                     【应用层 Application】                       │
│              Manager后端 (Go + Gin + GORM)                       │
│   ┌──────────┬──────────┬──────────┬──────────┬──────────┐      │
│   │节点管理  │Agent管理 │监控服务  │任务调度  │认证授权  │      │
│   └──────────┴──────────┴──────────┴──────────┴──────────┘      │
└────────────────────────────┬────────────────────────────────────┘
                             │ gRPC (mTLS 双向认证)
┌────────────────────────────▼────────────────────────────────────┐
│                      【主机层 Host Layer】                       │
│  ┌─────────Host 1─────────┐    ┌─────────Host N─────────┐       │
│  │ ┌───────────────────┐  │    │ ┌───────────────────┐  │       │
│  │ │      Daemon       │  │    │ │      Daemon       │  │       │
│  │ │  (守护进程管理)   │  │    │ │  (守护进程管理)   │  │       │
│  │ └─────────┬─────────┘  │    │ └─────────┬─────────┘  │       │
│  │ ┌─────────▼─────────┐  │    │ ┌─────────▼─────────┐  │       │
│  │ │   AgentRegistry   │  │    │ │   AgentRegistry   │  │       │
│  │ │  (多Agent管理)    │  │    │ │  (多Agent管理)    │  │       │
│  │ └───────┬───────────┘  │    │ └───────┬───────────┘  │       │
│  │         │               │    │         │               │       │
│  │ ┌───────▼───────┐      │    │ ┌───────▼───────┐      │       │
│  │ │ Filebeat      │      │    │ │ Filebeat      │      │       │
│  │ │ Telegraf      │      │    │ │ Telegraf      │      │       │
│  │ │ Node Exporter │      │    │ │ Node Exporter │      │       │
│  │ └───────────────┘      │    │ └───────────────┘      │       │
│  └────────────────────────┘    └────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

### 三层架构说明

1. **Manager (中心管理节点)**:
   - 提供 Web UI 和 RESTful API
   - 管理所有节点和 Agent
   - 存储监控数据和配置信息
   - 提供用户认证和权限管理

2. **Daemon (守护进程)**:
   - 运行在每台被管主机上
   - 管理多个第三方 Agent 实例(通过 AgentRegistry 和 MultiAgentManager)
   - 采集系统资源指标并上报到 Manager
   - 接收 Manager 的管理指令并执行

3. **Agent (执行进程)**:
   - 第三方 Agent 进程(如 Filebeat、Telegraf、Node Exporter)
   - 由 Daemon 统一管理其生命周期
   - 执行具体的采集、监控、日志收集等任务
   - 通过 Unix Socket 或 HTTP 与 Daemon 通信

### 通信协议

- **Manager ↔ Daemon**: gRPC over TLS (mTLS 双向认证)
- **Manager ↔ Web Frontend**: HTTP/HTTPS RESTful API (JWT 认证)
- **Daemon ↔ Agent**: Unix Domain Socket / HTTP (心跳上报)

---

## 🚀 快速开始

### 环境要求

| 组件 | 版本要求 |
|------|----------|
| Go | 1.24.0+ |
| MySQL | 8.0+ |
| Node.js | 18+ |
| npm | 9+ |

### 安装步骤

#### 1. 安装 Manager

```bash
# 克隆项目
git clone https://github.com/your-org/ops-scaffold-framework.git
cd ops-scaffold-framework/manager

# 编译
make build

# 配置数据库
mysql -u root -p < ../config/mysql/schema.sql

# 编辑配置文件
cp configs/manager.yaml.example configs/manager.yaml
vim configs/manager.yaml

# 启动服务
./bin/manager -config configs/manager.yaml
```

Manager 服务将在以下端口运行:
- HTTP API: `http://127.0.0.1:8080`
- gRPC Server: `127.0.0.1:9090`

#### 2. 安装 Daemon

```bash
cd daemon

# 编译
make build

# 编辑配置文件
cp configs/daemon.yaml.example configs/daemon.yaml
vim configs/daemon.yaml

# 配置 Manager 地址和证书
# manager:
#   address: "manager.example.com:9090"
#   tls:
#     cert_file: /etc/daemon/certs/client.crt
#     key_file: /etc/daemon/certs/client.key
#     ca_file: /etc/daemon/certs/ca.crt

# 启动服务
./bin/daemon -config configs/daemon.yaml
```

#### 3. 安装 Web 前端

```bash
cd web

# 安装依赖
npm install

# 配置 API 地址
cp .env.development.example .env.development
vim .env.development

# 启动开发服务器
npm run dev
```

Web 前端将在 `http://localhost:5173` 运行。

### 配置说明

#### Manager 配置 (`manager/configs/manager.yaml`)

```yaml
server:
  port: 8080
  grpc_port: 9090

database:
  host: localhost
  port: 3306
  database: ops_scaffold
  username: root
  password: your_password

jwt:
  secret: your_jwt_secret
  expire_hours: 24
```

#### Daemon 配置 (`daemon/configs/daemon.yaml`)

```yaml
manager:
  address: "manager.example.com:9090"
  heartbeat_interval: 60s

agents:
  - id: filebeat-logs
    type: filebeat
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat.yml
    enabled: true
```

### 启动服务

**推荐使用 systemd 管理服务** (Linux):

```bash
# Manager
sudo systemctl start manager
sudo systemctl enable manager

# Daemon
sudo systemctl start daemon
sudo systemctl enable daemon
```

**或直接运行**:

```bash
# Manager
cd manager
./bin/manager -config configs/manager.yaml

# Daemon
cd daemon
./bin/daemon -config configs/daemon.yaml

# Web Frontend
cd web
npm run dev
```

### 验证安装

#### 1. 检查 Manager 健康状态

```bash
curl http://localhost:8080/api/v1/health
# 预期输出: {"code":0,"message":"success","data":{"status":"healthy"}}
```

#### 2. 注册用户并登录

```bash
# 注册用户
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","email":"admin@example.com"}'

# 登录获取 Token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

#### 3. 查看节点列表

```bash
curl http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer <your-token>"
```

#### 4. 访问 Web 界面

打开浏览器访问 `http://localhost:5173`,使用注册的用户名和密码登录。

---

## 🎯 功能特性

### 已实现功能

#### Manager 后端
- ✅ RESTful API (12个HTTP接口)
- ✅ gRPC Server (3个gRPC接口)
- ✅ JWT 认证和用户管理
- ✅ 节点管理(注册、心跳、状态同步)
- ✅ Agent 管理 API (列表、操作、日志查看)
- ✅ 监控指标存储和查询
- ✅ 数据库自动迁移(GORM AutoMigrate)
- ✅ 完整的集成测试套件

#### Daemon 守护进程
- ✅ 资源采集(CPU、内存、磁盘、网络)
- ✅ 多 Agent 管理(AgentRegistry、MultiAgentManager)
- ✅ Agent 生命周期管理(启动、停止、重启)
- ✅ 健康检查和自动恢复(MultiHealthChecker)
- ✅ 资源监控和告警(ResourceMonitor)
- ✅ 日志管理(LogManager)
- ✅ gRPC 通信(连接 Manager)
- ✅ 心跳上报(60秒间隔)
- ✅ 配置热加载

#### Web 前端
- ✅ React + Vite + MUI 技术栈
- ✅ 用户认证(登录、注册)
- ✅ Dashboard (节点统计、监控概览)
- ✅ 节点管理(列表、详情、状态监控)
- ✅ Agent 管理界面(列表、操作、日志查看)
- ✅ 响应式设计,支持移动端
- ✅ 完整的单元测试和集成测试

### 待开发功能

- ⏳ Agent 任务执行引擎
- ⏳ 版本管理和自动更新
- ⏳ 告警通知(邮件、Webhook)
- ⏳ 任务调度(Cron、即时任务)
- ⏳ 审计日志
- ⏳ RBAC 权限管理

---

## 🛠️ 技术栈

### 后端技术

| 组件 | 技术选型 | 版本 | 说明 |
|------|----------|------|------|
| 编程语言 | Go | 1.24.0 | 高性能、并发友好 |
| Web 框架 | Gin | 1.10.0 | 轻量级、高性能 HTTP 框架 |
| ORM | GORM | 1.25.5 | 强大的 ORM 库 |
| 数据库 | MySQL | 8.0+ | 关系型数据库 |
| JWT 认证 | golang-jwt | 5.2.0 | JSON Web Token 库 |
| 密码加密 | bcrypt | - | 安全的密码哈希 |
| 配置管理 | Viper | 1.18.2 | 配置文件管理 |
| 日志 | zap | 1.26.0 | 高性能日志库 |
| gRPC | grpc-go | 1.77.0 | RPC 通信框架 |
| Protobuf | protobuf-go | 1.36.10 | 协议缓冲区 |
| 测试框架 | testify | 1.9.0 | 单元测试和断言 |
| 系统监控 | gopsutil | v3 | 系统资源采集 |

### 前端技术

| 组件 | 技术选型 | 版本 | 说明 |
|------|----------|------|------|
| 前端框架 | React | 18.2 | 组件化 UI 框架 |
| 构建工具 | Vite | 7.2 | 现代化构建工具 |
| UI 框架 | Material-UI | 7.3 | React UI 组件库 |
| 状态管理 | Zustand | 5.0 | 轻量级状态管理 |
| 数据请求 | React Query | 5.90 | 服务端状态管理 |
| 路由 | React Router | 7.10 | 前端路由 |
| HTTP 客户端 | Axios | 1.13 | HTTP 请求库 |
| TypeScript | TypeScript | 5.x | 类型安全 |

### 通信协议

- **gRPC**: Manager ↔ Daemon (mTLS 双向认证)
- **HTTP/HTTPS**: Manager ↔ Web Frontend (JWT 认证)
- **Unix Socket**: Daemon ↔ Agent (本地通信)

---

## 📁 项目结构

```
ops-scaffold-framework/
├── manager/                  # Manager 模块(独立 Go 模块)
│   ├── cmd/manager/          # Manager 入口
│   ├── internal/             # 内部包
│   │   ├── handler/          # HTTP 处理器
│   │   ├── service/          # 业务逻辑层
│   │   ├── repository/       # 数据访问层
│   │   ├── model/            # 数据模型
│   │   ├── middleware/       # 中间件
│   │   └── grpc/             # gRPC 服务端
│   ├── pkg/                  # 公共包
│   │   ├── proto/            # Protobuf 定义
│   │   ├── database/         # 数据库初始化
│   │   └── response/         # 统一响应格式
│   ├── configs/              # 配置文件
│   ├── test/                 # 测试代码
│   └── go.mod                # Go 模块定义
│
├── daemon/                   # Daemon 模块(独立 Go 模块)
│   ├── cmd/daemon/           # Daemon 入口
│   ├── internal/             # 内部包
│   │   ├── collector/        # 资源采集器
│   │   ├── agent/            # Agent 管理
│   │   │   ├── registry.go   # Agent 注册表
│   │   │   ├── multi_manager.go  # 多 Agent 管理器
│   │   │   ├── multi_health_checker.go  # 健康检查器
│   │   │   └── instance.go   # Agent 实例管理
│   │   ├── grpc/             # gRPC 客户端
│   │   └── daemon/           # Daemon 核心逻辑
│   ├── configs/              # 配置文件
│   └── go.mod                # Go 模块定义
│
├── web/                      # Web 前端模块
│   ├── src/
│   │   ├── components/       # React 组件
│   │   ├── pages/            # 页面组件
│   │   ├── api/              # API 客户端
│   │   ├── stores/           # 状态管理
│   │   └── router/           # 路由配置
│   ├── package.json          # npm 依赖
│   └── vite.config.ts        # Vite 配置
│
├── docs/                     # 项目文档
│   ├── api/                  # API 文档
│   ├── Agent管理功能使用指南.md
│   ├── Agent管理管理员手册.md
│   ├── Agent管理开发者文档.md
│   └── 设计文档_*.md         # 设计文档
│
├── config/mysql/             # MySQL 配置
│   ├── schema.sql            # 数据库 schema
│   └── my.cnf                # MySQL 配置文件
│
└── README.md                 # 本文件
```

### 模块说明

1. **Manager 模块**: 独立 Go 模块,module 名称 `github.com/bingooyong/ops-scaffold-framework/manager`
2. **Daemon 模块**: 独立 Go 模块,module 名称 `github.com/bingooyong/ops-scaffold-framework/daemon`
3. **Web 模块**: 前端项目,使用 npm 管理依赖
4. **根目录**: 项目容器,不包含 go.mod

---

## 📚 文档链接

### 用户文档
- [Agent 管理功能使用指南](docs/Agent管理功能使用指南.md) - 用户操作手册
- [Agent 管理管理员手册](docs/Agent管理管理员手册.md) - 系统管理员指南
- [快速入门指南](QUICKSTART.md) - 快速开始使用

### 开发者文档
- [Agent 管理开发者文档](docs/Agent管理开发者文档.md) - 开发者指南
- [Agent 管理配置示例](docs/Agent管理配置示例.md) - 配置示例
- [gRPC 最佳实践](docs/grpc-best-practices.md) - gRPC 开发规范
- [前端开发规范](docs/前端开发规范.md) - 前端开发指南

### API 文档
- [Manager API 文档](docs/api/Manager_API.md) - Manager HTTP API 完整文档
- [API 文档索引](docs/api/README.md) - API 文档入口

### 设计文档
- [Daemon 模块设计](docs/设计文档_01_Daemon模块.md) - Daemon 详细设计
- [Agent 模块设计](docs/设计文档_02_Agent模块.md) - Agent 详细设计
- [Manager 模块设计](docs/设计文档_03_Manager模块.md) - Manager 详细设计
- [Daemon 多 Agent 管理架构](docs/设计文档_04_Daemon多Agent管理架构.md) - 多 Agent 架构设计
- [Web 前端模块设计](docs/设计文档_04_Web前端模块.md) - Web 前端设计
- [运维工具框架需求文档](docs/运维工具框架需求文档.md) - 完整需求规格

---

## 💻 开发指南

### 开发环境搭建

#### 1. 安装依赖

```bash
# Go 环境
go version  # 确保 Go 1.24.0+

# Node.js 环境
node --version  # 确保 Node.js 18+
npm --version   # 确保 npm 9+

# MySQL
mysql --version  # 确保 MySQL 8.0+
```

#### 2. 克隆项目

```bash
git clone https://github.com/your-org/ops-scaffold-framework.git
cd ops-scaffold-framework
```

#### 3. 启动 MySQL

```bash
# 使用 Docker 启动 MySQL(可选)
docker run -d \
  --name mysql \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=ops_scaffold \
  -p 3306:3306 \
  mysql:8.0

# 或使用本地 MySQL
sudo systemctl start mysql
```

#### 4. 初始化数据库

```bash
mysql -u root -p < config/mysql/schema.sql
```

### 代码规范

#### Go 代码规范
- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行代码检查
- 遵循 Go 官方代码规范
- 使用有意义的变量名和函数名
- 添加完整的注释,尤其是公共函数和结构体

```bash
# 格式化代码
gofmt -w .

# 代码检查
golangci-lint run
```

#### TypeScript/React 代码规范
- 使用 ESLint 和 Prettier 格式化代码
- 遵循 React 官方最佳实践
- 使用 TypeScript 类型注解
- 组件命名使用 PascalCase
- 文件名使用 kebab-case

```bash
# 格式化代码
npm run lint
npm run format
```

### 测试指南

#### 运行 Manager 测试

```bash
cd manager

# 运行所有测试
go test ./...

# 运行集成测试
./test/run_tests.sh

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### 运行 Daemon 测试

```bash
cd daemon

# 运行所有测试
go test ./...

# 运行特定测试
go test -v ./internal/agent/
```

#### 运行 Web 前端测试

```bash
cd web

# 运行单元测试
npm run test

# 运行测试覆盖率
npm run test:coverage
```

### 贡献流程

1. **Fork 项目**
2. **创建功能分支**: `git checkout -b feature/your-feature`
3. **提交代码**: `git commit -am 'Add some feature'`
4. **推送分支**: `git push origin feature/your-feature`
5. **创建 Pull Request**

#### Pull Request 要求
- 代码通过所有测试
- 代码符合代码规范
- 添加必要的测试用例
- 更新相关文档
- 提供清晰的 PR 描述

---

## 🤝 贡献指南

我们欢迎所有形式的贡献,包括但不限于:

- 🐛 提交 Bug 报告
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 修复 Bug
- ✨ 实现新功能

### 如何贡献

1. 查看 [Issues](https://github.com/your-org/ops-scaffold-framework/issues) 列表
2. 选择感兴趣的 Issue 或创建新 Issue
3. Fork 项目并创建功能分支
4. 实现功能并添加测试
5. 提交 Pull Request

### 贡献者

感谢所有贡献者的付出!

---

## 📄 许可证

MIT License

Copyright (c) 2025 Ops Scaffold Framework Team

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

---

## 📞 联系我们

- **项目主页**: https://github.com/your-org/ops-scaffold-framework
- **问题反馈**: https://github.com/your-org/ops-scaffold-framework/issues
- **邮件**: support@example.com

---

**当前版本**: v0.4.0  
**最后更新**: 2025-01-27  
**维护者**: Ops Scaffold Framework Team
