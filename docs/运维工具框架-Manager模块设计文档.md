# Manager管理节点模块设计文档

**Ops Scaffold Framework - Manager Module Design**

---

| 项目 | 内容 |
|------|------|
| 文档版本 | 1.0 |
| 文档日期 | 2025-12-01 |
| 模块名称 | Manager（中心管理节点） |
| 文档状态 | 设计中 |

---

## 目录

1. [模块概述](#1-模块概述)
2. [架构设计](#2-架构设计)
3. [数据库设计](#3-数据库设计)
4. [后端详细设计](#4-后端详细设计)
5. [前端详细设计](#5-前端详细设计)
6. [安全设计](#6-安全设计)
7. [部署设计](#7-部署设计)
8. [配置管理](#8-配置管理)
9. [监控与运维](#9-监控与运维)

---

## 1. 模块概述

### 1.1 模块定位

Manager是运维工具框架的中心管理节点，采用前后端分离架构，负责：
- 管理所有Daemon节点的注册、状态监控
- 聚合展示节点资源使用情况和Agent运行状态
- 提供版本发布、更新和回滚功能
- 任务调度和执行管理
- 用户认证授权和审计日志

### 1.2 技术选型

| 层次 | 技术选型 | 版本 |
|------|----------|------|
| 后端语言 | Go | 1.21+ |
| Web框架 | Gin | 1.10.1 |
| ORM框架 | GORM | 1.31.0 |
| 数据库 | MySQL | 8.0+ |
| 缓存 | Redis | 7.0+ |
| 任务调度 | robfig/cron | v3.0.1 |
| 消息队列 | Kafka (可选) | 3.0+ |
| 认证 | OAuth2 / JWT | - |
| 前端框架 | React | 18+ |
| UI组件库 | MUI (Material UI) | 5+ |
| 状态管理 | Zustand | 4+ |
| 图表库 | Recharts / MUI X Charts | - |

### 1.3 模块边界

```
┌─────────────────────────────────────────────────────────────┐
│                      Manager 模块边界                        │
├─────────────────────────────────────────────────────────────┤
│  输入:                                                       │
│  • Daemon心跳和状态上报 (gRPC)                               │
│  • 用户管理操作请求 (HTTP/HTTPS)                             │
│  • 版本发布包上传                                            │
│                                                              │
│  输出:                                                       │
│  • 节点状态和监控数据 (REST API)                             │
│  • 版本更新推送指令 (gRPC)                                   │
│  • 任务执行指令 (HTTP/HTTPS → Agent)                         │
│  • 前端页面渲染                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Manager 架构                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    前端层 (React + MUI)                      │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐│    │
│  │  │Dashboard│ │节点管理 │ │版本管理 │ │任务中心 │ │系统设置││    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └────────┘│    │
│  └──────────────────────────┬──────────────────────────────────┘    │
│                             │ REST API (HTTP/HTTPS)                  │
│  ┌──────────────────────────▼──────────────────────────────────┐    │
│  │                    API网关层 (Gin Router)                    │    │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐    │    │
│  │  │ JWT认证中间件│ │ 限流中间件  │ │ 日志/审计中间件     │    │    │
│  │  └─────────────┘ └─────────────┘ └─────────────────────┘    │    │
│  └──────────────────────────┬──────────────────────────────────┘    │
│                             │                                        │
│  ┌──────────────────────────▼──────────────────────────────────┐    │
│  │                      业务服务层                              │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │    │
│  │  │NodeService│ │UpdateSvc │ │ TaskSvc  │ │ AuthSvc  │        │    │
│  │  │节点管理   │ │版本管理   │ │任务调度  │ │认证授权  │        │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘        │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐                     │    │
│  │  │MetricSvc │ │ AlertSvc │ │ AuditSvc │                     │    │
│  │  │监控指标   │ │告警服务   │ │审计日志  │                     │    │
│  │  └──────────┘ └──────────┘ └──────────┘                     │    │
│  └──────────────────────────┬──────────────────────────────────┘    │
│                             │                                        │
│  ┌──────────────────────────▼──────────────────────────────────┐    │
│  │                      数据访问层                              │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │    │
│  │  │NodeRepo  │ │VerRepo   │ │TaskRepo  │ │UserRepo  │        │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘        │    │
│  └──────────────────────────┬──────────────────────────────────┘    │
│                             │                                        │
│  ┌──────────────────────────▼──────────────────────────────────┐    │
│  │                      基础设施层                              │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │    │
│  │  │  MySQL   │ │  Redis   │ │  Kafka   │ │ 文件存储  │        │    │
│  │  │ (主数据) │ │ (缓存)   │ │ (可选)   │ │ (版本包) │        │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘        │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    gRPC服务层 (与Daemon通信)                 │    │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐    │    │
│  │  │ 节点注册    │ │ 心跳接收    │ │ 指标上报接收         │    │    │
│  │  └─────────────┘ └─────────────┘ └─────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 目录结构

```
manager/
├── cmd/
│   └── manager/
│       └── main.go                 # 程序入口
├── internal/
│   ├── config/
│   │   ├── config.go               # 配置结构定义
│   │   └── loader.go               # 配置加载器
│   ├── server/
│   │   ├── http.go                 # HTTP服务器
│   │   └── grpc.go                 # gRPC服务器
│   ├── router/
│   │   ├── router.go               # 路由注册
│   │   └── middleware/
│   │       ├── auth.go             # JWT认证中间件
│   │       ├── ratelimit.go        # 限流中间件
│   │       ├── cors.go             # CORS中间件
│   │       └── audit.go            # 审计日志中间件
│   ├── handler/
│   │   ├── node_handler.go         # 节点管理Handler
│   │   ├── version_handler.go      # 版本管理Handler
│   │   ├── task_handler.go         # 任务管理Handler
│   │   ├── metric_handler.go       # 监控指标Handler
│   │   ├── auth_handler.go         # 认证Handler
│   │   └── system_handler.go       # 系统管理Handler
│   ├── service/
│   │   ├── node_service.go         # 节点业务逻辑
│   │   ├── version_service.go      # 版本业务逻辑
│   │   ├── task_service.go         # 任务业务逻辑
│   │   ├── metric_service.go       # 监控业务逻辑
│   │   ├── auth_service.go         # 认证业务逻辑
│   │   ├── alert_service.go        # 告警业务逻辑
│   │   └── audit_service.go        # 审计业务逻辑
│   ├── repository/
│   │   ├── node_repo.go            # 节点数据访问
│   │   ├── version_repo.go         # 版本数据访问
│   │   ├── task_repo.go            # 任务数据访问
│   │   ├── metric_repo.go          # 指标数据访问
│   │   └── user_repo.go            # 用户数据访问
│   ├── model/
│   │   ├── node.go                 # 节点模型
│   │   ├── version.go              # 版本模型
│   │   ├── task.go                 # 任务模型
│   │   ├── metric.go               # 指标模型
│   │   ├── user.go                 # 用户模型
│   │   └── audit.go                # 审计日志模型
│   ├── grpc/
│   │   ├── server.go               # gRPC服务实现
│   │   └── proto/
│   │       └── daemon.proto        # Protobuf定义
│   ├── scheduler/
│   │   ├── scheduler.go            # 任务调度器
│   │   └── jobs/
│   │       ├── metric_cleanup.go   # 指标数据清理
│   │       └── node_check.go       # 节点状态检查
│   └── pkg/
│       ├── crypto/
│       │   ├── sign.go             # 签名工具
│       │   └── hash.go             # 哈希工具
│       ├── response/
│       │   └── response.go         # 统一响应格式
│       └── validator/
│           └── validator.go        # 参数校验
├── api/
│   └── openapi.yaml                # OpenAPI规范文档
├── web/                            # 前端代码
│   ├── src/
│   │   ├── components/             # 通用组件
│   │   ├── pages/                  # 页面组件
│   │   ├── services/               # API服务
│   │   ├── stores/                 # 状态管理
│   │   ├── hooks/                  # 自定义Hooks
│   │   ├── utils/                  # 工具函数
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
├── configs/
│   ├── config.yaml                 # 配置文件模板
│   └── config.example.yaml
├── scripts/
│   ├── init_db.sql                 # 数据库初始化
│   └── migrate.sh                  # 迁移脚本
├── Dockerfile
├── docker-compose.yaml
├── Makefile
└── README.md
```

### 2.3 核心流程

#### 2.3.1 节点注册流程

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Daemon  │     │  gRPC    │     │  Node    │     │  MySQL   │
│          │     │  Server  │     │  Service │     │          │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │
     │ 1.Register()   │                │                │
     │───────────────▶│                │                │
     │                │ 2.验证证书     │                │
     │                │───────────────▶│                │
     │                │                │ 3.检查节点     │
     │                │                │───────────────▶│
     │                │                │                │
     │                │                │ 4.返回结果     │
     │                │                │◀───────────────│
     │                │                │                │
     │                │                │ 5.新建/更新    │
     │                │                │───────────────▶│
     │                │                │                │
     │                │ 6.返回节点ID   │                │
     │                │◀───────────────│                │
     │ 7.注册成功     │                │                │
     │◀───────────────│                │                │
     │                │                │                │
```

#### 2.3.2 版本更新流程

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  前端    │     │  API     │     │ Version  │     │  gRPC    │     │  Daemon  │
│          │     │ Handler  │     │ Service  │     │  Client  │     │          │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │                │
     │ 1.发起更新     │                │                │                │
     │───────────────▶│                │                │                │
     │                │ 2.校验请求     │                │                │
     │                │───────────────▶│                │                │
     │                │                │ 3.创建更新任务 │                │
     │                │                │────────────────│                │
     │                │                │                │                │
     │                │                │ 4.推送更新     │                │
     │                │                │───────────────▶│                │
     │                │                │                │ 5.PushUpdate() │
     │                │                │                │───────────────▶│
     │                │                │                │                │
     │                │                │                │ 6.下载执行     │
     │                │                │                │◀───────────────│
     │                │                │                │                │
     │                │                │                │ 7.上报结果     │
     │                │                │◀───────────────│◀───────────────│
     │                │                │                │                │
     │ 8.返回进度     │                │                │                │
     │◀───────────────│◀───────────────│                │                │
     │                │                │                │                │
```

---

## 3. 数据库设计

### 3.1 ER图

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│     users       │       │   node_groups   │       │      nodes      │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id (PK)         │       │ id (PK)         │       │ id (PK)         │
│ username        │       │ name            │◀──────│ group_id (FK)   │
│ password_hash   │       │ description     │       │ hostname        │
│ email           │       │ parent_id       │       │ ip              │
│ role            │       │ created_at      │       │ os              │
│ status          │       └─────────────────┘       │ arch            │
│ created_at      │                                 │ daemon_version  │
│ updated_at      │       ┌─────────────────┐       │ status          │
└─────────────────┘       │     agents      │       │ last_heartbeat  │
                          ├─────────────────┤       │ tags (JSON)     │
┌─────────────────┐       │ id (PK)         │       │ created_at      │
│    versions     │       │ node_id (FK)    │◀──────│ updated_at      │
├─────────────────┤       │ version         │       └─────────────────┘
│ id (PK)         │       │ status          │               │
│ component       │       │ pid             │               │
│ version         │       │ cpu_usage       │               ▼
│ file_path       │       │ memory_usage    │       ┌─────────────────┐
│ file_size       │       │ updated_at      │       │  node_metrics   │
│ hash_sha256     │       └─────────────────┘       ├─────────────────┤
│ signature       │                                 │ id (PK)         │
│ release_notes   │       ┌─────────────────┐       │ node_id (FK)    │
│ created_by      │       │update_records   │       │ cpu_usage       │
│ created_at      │       ├─────────────────┤       │ memory_usage    │
└─────────────────┘       │ id (PK)         │       │ disk_usage      │
        │                 │ node_id (FK)    │       │ network_rx      │
        │                 │ component       │       │ network_tx      │
        └────────────────▶│ from_version    │       │ load_avg        │
                          │ to_version (FK) │       │ timestamp       │
                          │ status          │       └─────────────────┘
                          │ error_message   │
                          │ started_at      │       ┌─────────────────┐
                          │ finished_at     │       │   audit_logs    │
                          └─────────────────┘       ├─────────────────┤
                                                    │ id (PK)         │
┌─────────────────┐       ┌─────────────────┐       │ user_id (FK)    │
│     tasks       │       │task_executions  │       │ action          │
├─────────────────┤       ├─────────────────┤       │ resource_type   │
│ id (PK)         │◀──────│ task_id (FK)    │       │ resource_id     │
│ name            │       │ id (PK)         │       │ detail (JSON)   │
│ type            │       │ node_id (FK)    │       │ ip              │
│ content (JSON)  │       │ status          │       │ user_agent      │
│ cron_expr       │       │ output (TEXT)   │       │ created_at      │
│ timeout         │       │ error           │       └─────────────────┘
│ retry_count     │       │ started_at      │
│ status          │       │ finished_at     │
│ created_by      │       └─────────────────┘
│ created_at      │
│ updated_at      │
└─────────────────┘
```

### 3.2 表结构定义

#### 3.2.1 用户表 (users)

```sql
CREATE TABLE `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(64) NOT NULL COMMENT '用户名',
  `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希',
  `email` VARCHAR(128) DEFAULT NULL COMMENT '邮箱',
  `phone` VARCHAR(20) DEFAULT NULL COMMENT '手机号',
  `role` ENUM('admin', 'operator', 'viewer') NOT NULL DEFAULT 'viewer' COMMENT '角色',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 0-禁用, 1-启用',
  `last_login_at` DATETIME DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` VARCHAR(45) DEFAULT NULL COMMENT '最后登录IP',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  UNIQUE KEY `uk_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';
```

#### 3.2.2 节点分组表 (node_groups)

```sql
CREATE TABLE `node_groups` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL COMMENT '分组名称',
  `description` VARCHAR(255) DEFAULT NULL COMMENT '描述',
  `parent_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '父分组ID',
  `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='节点分组表';
```

#### 3.2.3 节点表 (nodes)

```sql
CREATE TABLE `nodes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `uuid` VARCHAR(36) NOT NULL COMMENT '节点UUID',
  `hostname` VARCHAR(128) NOT NULL COMMENT '主机名',
  `ip` VARCHAR(45) NOT NULL COMMENT 'IP地址',
  `port` INT NOT NULL DEFAULT 9100 COMMENT 'Daemon端口',
  `group_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '分组ID',
  `os` VARCHAR(32) DEFAULT NULL COMMENT '操作系统',
  `arch` VARCHAR(16) DEFAULT NULL COMMENT '架构',
  `kernel_version` VARCHAR(64) DEFAULT NULL COMMENT '内核版本',
  `daemon_version` VARCHAR(32) DEFAULT NULL COMMENT 'Daemon版本',
  `cpu_cores` INT DEFAULT NULL COMMENT 'CPU核数',
  `memory_total` BIGINT DEFAULT NULL COMMENT '总内存(bytes)',
  `disk_total` BIGINT DEFAULT NULL COMMENT '总磁盘(bytes)',
  `status` ENUM('online', 'offline', 'unknown') NOT NULL DEFAULT 'unknown' COMMENT '状态',
  `last_heartbeat` DATETIME DEFAULT NULL COMMENT '最后心跳时间',
  `tags` JSON DEFAULT NULL COMMENT '标签',
  `labels` JSON DEFAULT NULL COMMENT '标签键值对',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uuid` (`uuid`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_status` (`status`),
  KEY `idx_ip` (`ip`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='节点表';
```

#### 3.2.4 Agent表 (agents)

```sql
CREATE TABLE `agents` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `node_id` BIGINT UNSIGNED NOT NULL COMMENT '节点ID',
  `version` VARCHAR(32) DEFAULT NULL COMMENT 'Agent版本',
  `status` ENUM('running', 'stopped', 'error', 'unknown') NOT NULL DEFAULT 'unknown' COMMENT '状态',
  `pid` INT DEFAULT NULL COMMENT '进程ID',
  `cpu_usage` DECIMAL(5,2) DEFAULT NULL COMMENT 'CPU使用率(%)',
  `memory_usage` BIGINT DEFAULT NULL COMMENT '内存使用(bytes)',
  `start_time` DATETIME DEFAULT NULL COMMENT '启动时间',
  `restart_count` INT NOT NULL DEFAULT 0 COMMENT '重启次数',
  `last_error` TEXT DEFAULT NULL COMMENT '最后错误信息',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_node_id` (`node_id`),
  CONSTRAINT `fk_agents_node` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Agent表';
```

#### 3.2.5 节点指标表 (node_metrics)

```sql
CREATE TABLE `node_metrics` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `node_id` BIGINT UNSIGNED NOT NULL COMMENT '节点ID',
  `cpu_usage` DECIMAL(5,2) DEFAULT NULL COMMENT 'CPU使用率(%)',
  `memory_usage` DECIMAL(5,2) DEFAULT NULL COMMENT '内存使用率(%)',
  `memory_used` BIGINT DEFAULT NULL COMMENT '已用内存(bytes)',
  `disk_usage` DECIMAL(5,2) DEFAULT NULL COMMENT '磁盘使用率(%)',
  `disk_used` BIGINT DEFAULT NULL COMMENT '已用磁盘(bytes)',
  `network_rx` BIGINT DEFAULT NULL COMMENT '网络接收(bytes/s)',
  `network_tx` BIGINT DEFAULT NULL COMMENT '网络发送(bytes/s)',
  `load_avg_1` DECIMAL(5,2) DEFAULT NULL COMMENT '1分钟负载',
  `load_avg_5` DECIMAL(5,2) DEFAULT NULL COMMENT '5分钟负载',
  `load_avg_15` DECIMAL(5,2) DEFAULT NULL COMMENT '15分钟负载',
  `process_count` INT DEFAULT NULL COMMENT '进程数',
  `tcp_connections` INT DEFAULT NULL COMMENT 'TCP连接数',
  `timestamp` DATETIME NOT NULL COMMENT '采集时间',
  PRIMARY KEY (`id`),
  KEY `idx_node_timestamp` (`node_id`, `timestamp`),
  KEY `idx_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='节点指标表';

-- 按月分区（可选）
-- ALTER TABLE node_metrics PARTITION BY RANGE (TO_DAYS(timestamp)) (
--   PARTITION p202512 VALUES LESS THAN (TO_DAYS('2026-01-01')),
--   PARTITION p202601 VALUES LESS THAN (TO_DAYS('2026-02-01'))
-- );
```

#### 3.2.6 版本表 (versions)

```sql
CREATE TABLE `versions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `component` ENUM('daemon', 'agent') NOT NULL COMMENT '组件类型',
  `version` VARCHAR(32) NOT NULL COMMENT '版本号',
  `file_path` VARCHAR(255) NOT NULL COMMENT '文件路径',
  `file_name` VARCHAR(128) NOT NULL COMMENT '文件名',
  `file_size` BIGINT NOT NULL COMMENT '文件大小(bytes)',
  `hash_sha256` VARCHAR(64) NOT NULL COMMENT 'SHA256哈希',
  `signature` TEXT NOT NULL COMMENT '数字签名',
  `os` VARCHAR(32) NOT NULL DEFAULT 'linux' COMMENT '操作系统',
  `arch` VARCHAR(16) NOT NULL DEFAULT 'amd64' COMMENT '架构',
  `release_notes` TEXT DEFAULT NULL COMMENT '发布说明',
  `is_latest` TINYINT NOT NULL DEFAULT 0 COMMENT '是否最新版本',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建人',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_component_version_os_arch` (`component`, `version`, `os`, `arch`),
  KEY `idx_component_latest` (`component`, `is_latest`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='版本表';
```

#### 3.2.7 更新记录表 (update_records)

```sql
CREATE TABLE `update_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `batch_id` VARCHAR(36) NOT NULL COMMENT '批次ID',
  `node_id` BIGINT UNSIGNED NOT NULL COMMENT '节点ID',
  `component` ENUM('daemon', 'agent') NOT NULL COMMENT '组件类型',
  `from_version` VARCHAR(32) DEFAULT NULL COMMENT '原版本',
  `to_version` VARCHAR(32) NOT NULL COMMENT '目标版本',
  `status` ENUM('pending', 'downloading', 'installing', 'verifying', 'success', 'failed', 'rollback') NOT NULL DEFAULT 'pending' COMMENT '状态',
  `progress` INT NOT NULL DEFAULT 0 COMMENT '进度(0-100)',
  `error_message` TEXT DEFAULT NULL COMMENT '错误信息',
  `started_at` DATETIME DEFAULT NULL COMMENT '开始时间',
  `finished_at` DATETIME DEFAULT NULL COMMENT '完成时间',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建人',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_batch_id` (`batch_id`),
  KEY `idx_node_id` (`node_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='更新记录表';
```

#### 3.2.8 任务表 (tasks)

```sql
CREATE TABLE `tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL COMMENT '任务名称',
  `type` ENUM('script', 'file', 'service', 'custom') NOT NULL COMMENT '任务类型',
  `content` JSON NOT NULL COMMENT '任务内容',
  `cron_expr` VARCHAR(64) DEFAULT NULL COMMENT 'Cron表达式(定时任务)',
  `timeout` INT NOT NULL DEFAULT 300 COMMENT '超时时间(秒)',
  `retry_count` INT NOT NULL DEFAULT 0 COMMENT '重试次数',
  `retry_interval` INT NOT NULL DEFAULT 60 COMMENT '重试间隔(秒)',
  `target_type` ENUM('node', 'group', 'tag', 'all') NOT NULL DEFAULT 'node' COMMENT '目标类型',
  `target_value` JSON DEFAULT NULL COMMENT '目标值',
  `status` ENUM('enabled', 'disabled') NOT NULL DEFAULT 'enabled' COMMENT '状态',
  `description` VARCHAR(255) DEFAULT NULL COMMENT '描述',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建人',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_type` (`type`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务表';
```

#### 3.2.9 任务执行表 (task_executions)

```sql
CREATE TABLE `task_executions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_id` BIGINT UNSIGNED NOT NULL COMMENT '任务ID',
  `execution_id` VARCHAR(36) NOT NULL COMMENT '执行批次ID',
  `node_id` BIGINT UNSIGNED NOT NULL COMMENT '节点ID',
  `status` ENUM('pending', 'running', 'success', 'failed', 'timeout', 'cancelled') NOT NULL DEFAULT 'pending' COMMENT '状态',
  `exit_code` INT DEFAULT NULL COMMENT '退出码',
  `output` MEDIUMTEXT DEFAULT NULL COMMENT '执行输出',
  `error` TEXT DEFAULT NULL COMMENT '错误信息',
  `started_at` DATETIME DEFAULT NULL COMMENT '开始时间',
  `finished_at` DATETIME DEFAULT NULL COMMENT '完成时间',
  `triggered_by` ENUM('manual', 'schedule', 'api') NOT NULL DEFAULT 'manual' COMMENT '触发方式',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建人',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_execution_id` (`execution_id`),
  KEY `idx_node_id` (`node_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务执行表';
```

#### 3.2.10 审计日志表 (audit_logs)

```sql
CREATE TABLE `audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '用户ID',
  `username` VARCHAR(64) DEFAULT NULL COMMENT '用户名',
  `action` VARCHAR(64) NOT NULL COMMENT '操作',
  `resource_type` VARCHAR(32) NOT NULL COMMENT '资源类型',
  `resource_id` VARCHAR(64) DEFAULT NULL COMMENT '资源ID',
  `detail` JSON DEFAULT NULL COMMENT '详细信息',
  `ip` VARCHAR(45) NOT NULL COMMENT '客户端IP',
  `user_agent` VARCHAR(255) DEFAULT NULL COMMENT 'User-Agent',
  `status` ENUM('success', 'failed') NOT NULL DEFAULT 'success' COMMENT '操作状态',
  `error_message` TEXT DEFAULT NULL COMMENT '错误信息',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_action` (`action`),
  KEY `idx_resource` (`resource_type`, `resource_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计日志表';
```

#### 3.2.11 告警规则表 (alert_rules)

```sql
CREATE TABLE `alert_rules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL COMMENT '规则名称',
  `metric` VARCHAR(64) NOT NULL COMMENT '指标名称',
  `operator` ENUM('gt', 'gte', 'lt', 'lte', 'eq', 'ne') NOT NULL COMMENT '比较运算符',
  `threshold` DECIMAL(10,2) NOT NULL COMMENT '阈值',
  `duration` INT NOT NULL DEFAULT 60 COMMENT '持续时间(秒)',
  `severity` ENUM('critical', 'warning', 'info') NOT NULL DEFAULT 'warning' COMMENT '严重级别',
  `target_type` ENUM('all', 'group', 'node') NOT NULL DEFAULT 'all' COMMENT '目标类型',
  `target_value` JSON DEFAULT NULL COMMENT '目标值',
  `notification` JSON DEFAULT NULL COMMENT '通知配置',
  `status` ENUM('enabled', 'disabled') NOT NULL DEFAULT 'enabled' COMMENT '状态',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建人',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_metric` (`metric`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警规则表';
```

#### 3.2.12 告警记录表 (alerts)

```sql
CREATE TABLE `alerts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `rule_id` BIGINT UNSIGNED NOT NULL COMMENT '规则ID',
  `node_id` BIGINT UNSIGNED NOT NULL COMMENT '节点ID',
  `metric` VARCHAR(64) NOT NULL COMMENT '指标名称',
  `current_value` DECIMAL(10,2) NOT NULL COMMENT '当前值',
  `threshold` DECIMAL(10,2) NOT NULL COMMENT '阈值',
  `severity` ENUM('critical', 'warning', 'info') NOT NULL COMMENT '严重级别',
  `status` ENUM('firing', 'resolved') NOT NULL DEFAULT 'firing' COMMENT '状态',
  `fired_at` DATETIME NOT NULL COMMENT '触发时间',
  `resolved_at` DATETIME DEFAULT NULL COMMENT '恢复时间',
  `acknowledged_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '确认人',
  `acknowledged_at` DATETIME DEFAULT NULL COMMENT '确认时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_rule_id` (`rule_id`),
  KEY `idx_node_id` (`node_id`),
  KEY `idx_status` (`status`),
  KEY `idx_fired_at` (`fired_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警记录表';
```

### 3.3 索引设计说明

| 表名 | 索引名 | 索引列 | 用途 |
|------|--------|--------|------|
| nodes | idx_status | status | 按状态筛选节点 |
| nodes | idx_group_id | group_id | 按分组查询节点 |
| node_metrics | idx_node_timestamp | node_id, timestamp | 查询节点历史指标 |
| node_metrics | idx_timestamp | timestamp | 清理过期数据 |
| task_executions | idx_execution_id | execution_id | 按批次查询执行 |
| audit_logs | idx_created_at | created_at | 按时间范围查询 |

---

## 4. 后端详细设计

### 4.1 核心模型定义

#### 4.1.1 节点模型 (model/node.go)

```go
package model

import (
    "time"
    "gorm.io/datatypes"
)

type NodeStatus string

const (
    NodeStatusOnline  NodeStatus = "online"
    NodeStatusOffline NodeStatus = "offline"
    NodeStatusUnknown NodeStatus = "unknown"
)

type Node struct {
    ID            uint64         `gorm:"primaryKey" json:"id"`
    UUID          string         `gorm:"size:36;uniqueIndex" json:"uuid"`
    Hostname      string         `gorm:"size:128;not null" json:"hostname"`
    IP            string         `gorm:"size:45;not null;index" json:"ip"`
    Port          int            `gorm:"default:9100" json:"port"`
    GroupID       *uint64        `gorm:"index" json:"group_id"`
    OS            string         `gorm:"size:32" json:"os"`
    Arch          string         `gorm:"size:16" json:"arch"`
    KernelVersion string         `gorm:"size:64" json:"kernel_version"`
    DaemonVersion string         `gorm:"size:32" json:"daemon_version"`
    CPUCores      int            `json:"cpu_cores"`
    MemoryTotal   int64          `json:"memory_total"`
    DiskTotal     int64          `json:"disk_total"`
    Status        NodeStatus     `gorm:"size:16;default:unknown;index" json:"status"`
    LastHeartbeat *time.Time     `json:"last_heartbeat"`
    Tags          datatypes.JSON `json:"tags"`
    Labels        datatypes.JSON `json:"labels"`
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
    
    // 关联
    Group  *NodeGroup `gorm:"foreignKey:GroupID" json:"group,omitempty"`
    Agent  *Agent     `gorm:"foreignKey:NodeID" json:"agent,omitempty"`
}

func (Node) TableName() string {
    return "nodes"
}
```

#### 4.1.2 版本模型 (model/version.go)

```go
package model

import "time"

type ComponentType string

const (
    ComponentDaemon ComponentType = "daemon"
    ComponentAgent  ComponentType = "agent"
)

type Version struct {
    ID           uint64        `gorm:"primaryKey" json:"id"`
    Component    ComponentType `gorm:"size:16;not null" json:"component"`
    Version      string        `gorm:"size:32;not null" json:"version"`
    FilePath     string        `gorm:"size:255;not null" json:"-"`
    FileName     string        `gorm:"size:128;not null" json:"file_name"`
    FileSize     int64         `gorm:"not null" json:"file_size"`
    HashSHA256   string        `gorm:"size:64;not null" json:"hash_sha256"`
    Signature    string        `gorm:"type:text;not null" json:"-"`
    OS           string        `gorm:"size:32;default:linux" json:"os"`
    Arch         string        `gorm:"size:16;default:amd64" json:"arch"`
    ReleaseNotes string        `gorm:"type:text" json:"release_notes"`
    IsLatest     bool          `gorm:"default:false" json:"is_latest"`
    CreatedBy    uint64        `gorm:"not null" json:"created_by"`
    CreatedAt    time.Time     `json:"created_at"`
}

func (Version) TableName() string {
    return "versions"
}
```

### 4.2 服务层设计

#### 4.2.1 节点服务 (service/node_service.go)

```go
package service

import (
    "context"
    "time"
)

type NodeService interface {
    // 节点管理
    List(ctx context.Context, req *NodeListRequest) (*NodeListResponse, error)
    Get(ctx context.Context, id uint64) (*Node, error)
    GetByUUID(ctx context.Context, uuid string) (*Node, error)
    Update(ctx context.Context, id uint64, req *NodeUpdateRequest) error
    Delete(ctx context.Context, id uint64) error
    
    // 节点注册(供Daemon调用)
    Register(ctx context.Context, req *NodeRegisterRequest) (*NodeRegisterResponse, error)
    
    // 心跳处理
    Heartbeat(ctx context.Context, req *HeartbeatRequest) error
    
    // 状态检查
    CheckOfflineNodes(ctx context.Context) error
    
    // 指标查询
    GetMetrics(ctx context.Context, nodeID uint64, start, end time.Time) ([]*NodeMetric, error)
    GetLatestMetrics(ctx context.Context, nodeID uint64) (*NodeMetric, error)
}

type NodeListRequest struct {
    Page     int    `form:"page" binding:"min=1"`
    PageSize int    `form:"page_size" binding:"min=1,max=100"`
    GroupID  uint64 `form:"group_id"`
    Status   string `form:"status"`
    Keyword  string `form:"keyword"`
    Tags     string `form:"tags"`
}

type NodeListResponse struct {
    Total int64   `json:"total"`
    Items []*Node `json:"items"`
}

type NodeRegisterRequest struct {
    UUID          string            `json:"uuid" binding:"required"`
    Hostname      string            `json:"hostname" binding:"required"`
    IP            string            `json:"ip" binding:"required"`
    Port          int               `json:"port"`
    OS            string            `json:"os"`
    Arch          string            `json:"arch"`
    KernelVersion string            `json:"kernel_version"`
    DaemonVersion string            `json:"daemon_version"`
    CPUCores      int               `json:"cpu_cores"`
    MemoryTotal   int64             `json:"memory_total"`
    DiskTotal     int64             `json:"disk_total"`
    Labels        map[string]string `json:"labels"`
}

type HeartbeatRequest struct {
    NodeUUID    string       `json:"node_uuid" binding:"required"`
    AgentStatus *AgentStatus `json:"agent_status"`
    Metrics     *NodeMetric  `json:"metrics"`
}

type AgentStatus struct {
    Version     string  `json:"version"`
    Status      string  `json:"status"`
    PID         int     `json:"pid"`
    CPUUsage    float64 `json:"cpu_usage"`
    MemoryUsage int64   `json:"memory_usage"`
}
```

#### 4.2.2 版本服务 (service/version_service.go)

```go
package service

import (
    "context"
    "io"
)

type VersionService interface {
    // 版本管理
    List(ctx context.Context, component string) ([]*Version, error)
    Get(ctx context.Context, id uint64) (*Version, error)
    GetLatest(ctx context.Context, component, os, arch string) (*Version, error)
    Upload(ctx context.Context, req *VersionUploadRequest) (*Version, error)
    Delete(ctx context.Context, id uint64) error
    
    // 版本更新
    Deploy(ctx context.Context, req *DeployRequest) (*DeployResponse, error)
    GetDeployStatus(ctx context.Context, batchID string) (*DeployStatusResponse, error)
    Rollback(ctx context.Context, req *RollbackRequest) error
    
    // 文件下载
    GetDownloadURL(ctx context.Context, id uint64) (string, error)
}

type VersionUploadRequest struct {
    Component    string    `form:"component" binding:"required,oneof=daemon agent"`
    Version      string    `form:"version" binding:"required"`
    OS           string    `form:"os" binding:"required"`
    Arch         string    `form:"arch" binding:"required"`
    ReleaseNotes string    `form:"release_notes"`
    File         io.Reader `form:"-"`
    FileName     string    `form:"-"`
    FileSize     int64     `form:"-"`
    UserID       uint64    `form:"-"`
}

type DeployRequest struct {
    Component   string   `json:"component" binding:"required,oneof=daemon agent"`
    VersionID   uint64   `json:"version_id" binding:"required"`
    TargetType  string   `json:"target_type" binding:"required,oneof=node group tag all"`
    TargetValue []string `json:"target_value"`
    Strategy    string   `json:"strategy" binding:"oneof=all rolling canary"`
    BatchSize   int      `json:"batch_size"`
    UserID      uint64   `json:"-"`
}

type DeployResponse struct {
    BatchID    string `json:"batch_id"`
    TotalNodes int    `json:"total_nodes"`
}

type DeployStatusResponse struct {
    BatchID   string              `json:"batch_id"`
    Status    string              `json:"status"`
    Total     int                 `json:"total"`
    Success   int                 `json:"success"`
    Failed    int                 `json:"failed"`
    Pending   int                 `json:"pending"`
    Records   []*UpdateRecordDTO  `json:"records"`
}
```

#### 4.2.3 任务服务 (service/task_service.go)

```go
package service

import "context"

type TaskService interface {
    // 任务管理
    List(ctx context.Context, req *TaskListRequest) (*TaskListResponse, error)
    Get(ctx context.Context, id uint64) (*Task, error)
    Create(ctx context.Context, req *TaskCreateRequest) (*Task, error)
    Update(ctx context.Context, id uint64, req *TaskUpdateRequest) error
    Delete(ctx context.Context, id uint64) error
    
    // 任务执行
    Execute(ctx context.Context, req *TaskExecuteRequest) (*TaskExecuteResponse, error)
    GetExecution(ctx context.Context, executionID string) (*ExecutionDetailResponse, error)
    CancelExecution(ctx context.Context, executionID string) error
    
    // 执行历史
    ListExecutions(ctx context.Context, taskID uint64, req *ExecutionListRequest) (*ExecutionListResponse, error)
}

type TaskCreateRequest struct {
    Name          string                 `json:"name" binding:"required,max=128"`
    Type          string                 `json:"type" binding:"required,oneof=script file service custom"`
    Content       map[string]interface{} `json:"content" binding:"required"`
    CronExpr      string                 `json:"cron_expr"`
    Timeout       int                    `json:"timeout" binding:"min=1,max=86400"`
    RetryCount    int                    `json:"retry_count" binding:"min=0,max=10"`
    RetryInterval int                    `json:"retry_interval"`
    TargetType    string                 `json:"target_type" binding:"required,oneof=node group tag all"`
    TargetValue   []string               `json:"target_value"`
    Description   string                 `json:"description"`
    UserID        uint64                 `json:"-"`
}

type TaskExecuteRequest struct {
    TaskID      uint64   `json:"task_id" binding:"required"`
    TargetNodes []uint64 `json:"target_nodes"` // 可选，覆盖默认目标
    UserID      uint64   `json:"-"`
}

type TaskExecuteResponse struct {
    ExecutionID string `json:"execution_id"`
    TotalNodes  int    `json:"total_nodes"`
}
```

### 4.3 API Handler设计

#### 4.3.1 节点Handler (handler/node_handler.go)

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "net/http"
)

type NodeHandler struct {
    nodeService NodeService
}

func NewNodeHandler(ns NodeService) *NodeHandler {
    return &NodeHandler{nodeService: ns}
}

// @Summary 获取节点列表
// @Tags 节点管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param group_id query int false "分组ID"
// @Param status query string false "状态" Enums(online, offline, unknown)
// @Param keyword query string false "关键词(hostname/ip)"
// @Success 200 {object} response.Response{data=NodeListResponse}
// @Router /api/v1/nodes [get]
func (h *NodeHandler) List(c *gin.Context) {
    var req NodeListRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "参数错误", err)
        return
    }
    
    if req.Page == 0 {
        req.Page = 1
    }
    if req.PageSize == 0 {
        req.PageSize = 20
    }
    
    result, err := h.nodeService.List(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "查询失败", err)
        return
    }
    
    response.Success(c, result)
}

// @Summary 获取节点详情
// @Tags 节点管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Success 200 {object} response.Response{data=Node}
// @Router /api/v1/nodes/{id} [get]
func (h *NodeHandler) Get(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        response.Error(c, http.StatusBadRequest, "无效的节点ID", err)
        return
    }
    
    node, err := h.nodeService.Get(c.Request.Context(), id)
    if err != nil {
        response.Error(c, http.StatusNotFound, "节点不存在", err)
        return
    }
    
    response.Success(c, node)
}

// @Summary 更新节点信息
// @Tags 节点管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Param body body NodeUpdateRequest true "更新内容"
// @Success 200 {object} response.Response
// @Router /api/v1/nodes/{id} [put]
func (h *NodeHandler) Update(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        response.Error(c, http.StatusBadRequest, "无效的节点ID", err)
        return
    }
    
    var req NodeUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "参数错误", err)
        return
    }
    
    if err := h.nodeService.Update(c.Request.Context(), id, &req); err != nil {
        response.Error(c, http.StatusInternalServerError, "更新失败", err)
        return
    }
    
    response.Success(c, nil)
}

// @Summary 获取节点指标
// @Tags 节点管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Param start query string false "开始时间(RFC3339)"
// @Param end query string false "结束时间(RFC3339)"
// @Success 200 {object} response.Response{data=[]NodeMetric}
// @Router /api/v1/nodes/{id}/metrics [get]
func (h *NodeHandler) GetMetrics(c *gin.Context) {
    // ... 实现
}
```

### 4.4 路由设计

```go
package router

import (
    "github.com/gin-gonic/gin"
)

func SetupRouter(
    nodeHandler *handler.NodeHandler,
    versionHandler *handler.VersionHandler,
    taskHandler *handler.TaskHandler,
    authHandler *handler.AuthHandler,
    systemHandler *handler.SystemHandler,
) *gin.Engine {
    r := gin.New()
    
    // 全局中间件
    r.Use(gin.Recovery())
    r.Use(middleware.Logger())
    r.Use(middleware.CORS())
    r.Use(middleware.RequestID())
    
    // 健康检查
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })
    
    // API v1
    v1 := r.Group("/api/v1")
    {
        // 认证相关（无需登录）
        auth := v1.Group("/auth")
        {
            auth.POST("/login", authHandler.Login)
            auth.POST("/refresh", authHandler.RefreshToken)
        }
        
        // 需要认证的接口
        authorized := v1.Group("")
        authorized.Use(middleware.JWTAuth())
        authorized.Use(middleware.AuditLog())
        {
            // 节点管理
            nodes := authorized.Group("/nodes")
            {
                nodes.GET("", nodeHandler.List)
                nodes.GET("/:id", nodeHandler.Get)
                nodes.PUT("/:id", nodeHandler.Update)
                nodes.DELETE("/:id", middleware.RequireRole("admin"), nodeHandler.Delete)
                nodes.GET("/:id/metrics", nodeHandler.GetMetrics)
                nodes.GET("/:id/agent", nodeHandler.GetAgent)
            }
            
            // 节点分组
            groups := authorized.Group("/groups")
            {
                groups.GET("", nodeHandler.ListGroups)
                groups.POST("", middleware.RequireRole("admin", "operator"), nodeHandler.CreateGroup)
                groups.PUT("/:id", middleware.RequireRole("admin", "operator"), nodeHandler.UpdateGroup)
                groups.DELETE("/:id", middleware.RequireRole("admin"), nodeHandler.DeleteGroup)
            }
            
            // 版本管理
            versions := authorized.Group("/versions")
            {
                versions.GET("", versionHandler.List)
                versions.GET("/:id", versionHandler.Get)
                versions.POST("", middleware.RequireRole("admin", "operator"), versionHandler.Upload)
                versions.DELETE("/:id", middleware.RequireRole("admin"), versionHandler.Delete)
            }
            
            // 版本更新
            updates := authorized.Group("/updates")
            {
                updates.POST("/deploy", middleware.RequireRole("admin", "operator"), versionHandler.Deploy)
                updates.GET("/:batch_id/status", versionHandler.GetDeployStatus)
                updates.POST("/:batch_id/rollback", middleware.RequireRole("admin", "operator"), versionHandler.Rollback)
            }
            
            // 任务管理
            tasks := authorized.Group("/tasks")
            {
                tasks.GET("", taskHandler.List)
                tasks.GET("/:id", taskHandler.Get)
                tasks.POST("", middleware.RequireRole("admin", "operator"), taskHandler.Create)
                tasks.PUT("/:id", middleware.RequireRole("admin", "operator"), taskHandler.Update)
                tasks.DELETE("/:id", middleware.RequireRole("admin"), taskHandler.Delete)
                tasks.POST("/:id/execute", middleware.RequireRole("admin", "operator"), taskHandler.Execute)
                tasks.GET("/:id/executions", taskHandler.ListExecutions)
            }
            
            // 任务执行
            executions := authorized.Group("/executions")
            {
                executions.GET("/:execution_id", taskHandler.GetExecution)
                executions.POST("/:execution_id/cancel", middleware.RequireRole("admin", "operator"), taskHandler.CancelExecution)
            }
            
            // 告警管理
            alerts := authorized.Group("/alerts")
            {
                alerts.GET("", systemHandler.ListAlerts)
                alerts.POST("/:id/acknowledge", systemHandler.AcknowledgeAlert)
                alerts.GET("/rules", systemHandler.ListAlertRules)
                alerts.POST("/rules", middleware.RequireRole("admin"), systemHandler.CreateAlertRule)
                alerts.PUT("/rules/:id", middleware.RequireRole("admin"), systemHandler.UpdateAlertRule)
                alerts.DELETE("/rules/:id", middleware.RequireRole("admin"), systemHandler.DeleteAlertRule)
            }
            
            // 系统管理
            system := authorized.Group("/system")
            {
                system.GET("/dashboard", systemHandler.GetDashboard)
                system.GET("/audit-logs", middleware.RequireRole("admin"), systemHandler.ListAuditLogs)
                system.GET("/users", middleware.RequireRole("admin"), systemHandler.ListUsers)
                system.POST("/users", middleware.RequireRole("admin"), systemHandler.CreateUser)
                system.PUT("/users/:id", middleware.RequireRole("admin"), systemHandler.UpdateUser)
                system.DELETE("/users/:id", middleware.RequireRole("admin"), systemHandler.DeleteUser)
            }
        }
    }
    
    return r
}
```

### 4.5 gRPC服务设计

#### 4.5.1 Proto定义 (proto/daemon.proto)

```protobuf
syntax = "proto3";

package ops.daemon;

option go_package = "manager/internal/grpc/proto";

service DaemonService {
  // 节点注册
  rpc Register(RegisterRequest) returns (RegisterResponse);
  
  // 心跳上报
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  
  // 指标上报
  rpc ReportMetrics(ReportMetricsRequest) returns (ReportMetricsResponse);
  
  // 获取配置
  rpc GetConfig(GetConfigRequest) returns (GetConfigResponse);
  
  // 推送更新（流式）
  rpc SubscribeUpdates(SubscribeRequest) returns (stream UpdateEvent);
}

message RegisterRequest {
  string uuid = 1;
  string hostname = 2;
  string ip = 3;
  int32 port = 4;
  string os = 5;
  string arch = 6;
  string kernel_version = 7;
  string daemon_version = 8;
  int32 cpu_cores = 9;
  int64 memory_total = 10;
  int64 disk_total = 11;
  map<string, string> labels = 12;
}

message RegisterResponse {
  int64 node_id = 1;
  int32 heartbeat_interval = 2;  // 心跳间隔(秒)
  int32 metric_interval = 3;     // 指标上报间隔(秒)
}

message HeartbeatRequest {
  string node_uuid = 1;
  AgentStatus agent_status = 2;
  int64 timestamp = 3;
}

message AgentStatus {
  string version = 1;
  string status = 2;
  int32 pid = 3;
  double cpu_usage = 4;
  int64 memory_usage = 5;
  int64 start_time = 6;
  int32 restart_count = 7;
}

message HeartbeatResponse {
  bool success = 1;
  repeated Command commands = 2;  // 待执行命令
}

message Command {
  string id = 1;
  string type = 2;  // update, restart, config
  bytes payload = 3;
}

message ReportMetricsRequest {
  string node_uuid = 1;
  NodeMetrics metrics = 2;
}

message NodeMetrics {
  double cpu_usage = 1;
  double memory_usage = 2;
  int64 memory_used = 3;
  double disk_usage = 4;
  int64 disk_used = 5;
  int64 network_rx = 6;
  int64 network_tx = 7;
  double load_avg_1 = 8;
  double load_avg_5 = 9;
  double load_avg_15 = 10;
  int32 process_count = 11;
  int32 tcp_connections = 12;
  int64 timestamp = 13;
}

message ReportMetricsResponse {
  bool success = 1;
}

message GetConfigRequest {
  string node_uuid = 1;
}

message GetConfigResponse {
  map<string, string> config = 1;
}

message SubscribeRequest {
  string node_uuid = 1;
}

message UpdateEvent {
  string event_type = 1;  // version_update, config_change
  string component = 2;   // daemon, agent
  string version = 3;
  string download_url = 4;
  string hash_sha256 = 5;
  bytes signature = 6;
}
```

---

## 5. 前端详细设计

### 5.1 页面结构

```
web/src/
├── components/                 # 通用组件
│   ├── Layout/
│   │   ├── Header.tsx         # 顶部导航
│   │   ├── Sidebar.tsx        # 侧边栏菜单
│   │   └── Layout.tsx         # 布局组件
│   ├── Table/
│   │   └── DataTable.tsx      # 数据表格
│   ├── Charts/
│   │   ├── LineChart.tsx      # 折线图
│   │   ├── PieChart.tsx       # 饼图
│   │   └── GaugeChart.tsx     # 仪表盘
│   ├── Status/
│   │   ├── NodeStatus.tsx     # 节点状态标签
│   │   └── AgentStatus.tsx    # Agent状态标签
│   └── Common/
│       ├── Loading.tsx        # 加载组件
│       ├── ErrorBoundary.tsx  # 错误边界
│       └── ConfirmDialog.tsx  # 确认对话框
├── pages/
│   ├── Dashboard/
│   │   └── index.tsx          # 仪表盘页面
│   ├── Nodes/
│   │   ├── index.tsx          # 节点列表
│   │   ├── Detail.tsx         # 节点详情
│   │   └── Groups.tsx         # 节点分组
│   ├── Versions/
│   │   ├── index.tsx          # 版本列表
│   │   ├── Upload.tsx         # 版本上传
│   │   └── Deploy.tsx         # 版本部署
│   ├── Tasks/
│   │   ├── index.tsx          # 任务列表
│   │   ├── Create.tsx         # 创建任务
│   │   └── Executions.tsx     # 执行历史
│   ├── Alerts/
│   │   ├── index.tsx          # 告警列表
│   │   └── Rules.tsx          # 告警规则
│   ├── System/
│   │   ├── Users.tsx          # 用户管理
│   │   ├── AuditLogs.tsx      # 审计日志
│   │   └── Settings.tsx       # 系统设置
│   └── Auth/
│       └── Login.tsx          # 登录页面
├── services/
│   ├── api.ts                 # API基础配置
│   ├── nodeService.ts         # 节点API
│   ├── versionService.ts      # 版本API
│   ├── taskService.ts         # 任务API
│   └── authService.ts         # 认证API
├── stores/
│   ├── authStore.ts           # 认证状态
│   ├── nodeStore.ts           # 节点状态
│   └── uiStore.ts             # UI状态
├── hooks/
│   ├── useAuth.ts             # 认证Hook
│   ├── useNodes.ts            # 节点数据Hook
│   └── useWebSocket.ts        # WebSocket Hook
├── utils/
│   ├── format.ts              # 格式化工具
│   ├── storage.ts             # 存储工具
│   └── constants.ts           # 常量定义
├── App.tsx
├── main.tsx
└── routes.tsx                 # 路由配置
```

### 5.2 核心页面设计

#### 5.2.1 Dashboard页面

```tsx
// pages/Dashboard/index.tsx
import React from 'react';
import { Grid, Paper, Typography, Box } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { getDashboardData } from '@/services/systemService';
import StatCard from '@/components/StatCard';
import NodeStatusPie from './components/NodeStatusPie';
import ResourceTrend from './components/ResourceTrend';
import RecentAlerts from './components/RecentAlerts';
import AgentVersionPie from './components/AgentVersionPie';

const Dashboard: React.FC = () => {
  const { data, isLoading } = useQuery(['dashboard'], getDashboardData, {
    refetchInterval: 30000, // 30秒刷新
  });

  if (isLoading) return <Loading />;

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        仪表盘
      </Typography>
      
      {/* 统计卡片 */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="节点总数"
            value={data?.totalNodes || 0}
            icon="computer"
            color="primary"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="在线节点"
            value={data?.onlineNodes || 0}
            total={data?.totalNodes}
            icon="check_circle"
            color="success"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Agent运行中"
            value={data?.runningAgents || 0}
            total={data?.totalNodes}
            icon="play_circle"
            color="info"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="活跃告警"
            value={data?.activeAlerts || 0}
            icon="warning"
            color="error"
          />
        </Grid>
      </Grid>

      {/* 图表区域 */}
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              节点状态分布
            </Typography>
            <NodeStatusPie data={data?.nodeStatusDistribution} />
          </Paper>
        </Grid>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Agent版本分布
            </Typography>
            <AgentVersionPie data={data?.agentVersionDistribution} />
          </Paper>
        </Grid>
        <Grid item xs={12}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              资源使用趋势（24小时）
            </Typography>
            <ResourceTrend data={data?.resourceTrend} />
          </Paper>
        </Grid>
        <Grid item xs={12}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              最近告警
            </Typography>
            <RecentAlerts data={data?.recentAlerts} />
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default Dashboard;
```

#### 5.2.2 节点列表页面

```tsx
// pages/Nodes/index.tsx
import React, { useState } from 'react';
import {
  Box, Paper, TextField, Select, MenuItem, Button,
  IconButton, Chip, Tooltip
} from '@mui/material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useQuery } from '@tanstack/react-query';
import { getNodes } from '@/services/nodeService';
import { useNavigate } from 'react-router-dom';
import RefreshIcon from '@mui/icons-material/Refresh';
import FilterListIcon from '@mui/icons-material/FilterList';
import NodeStatusChip from '@/components/Status/NodeStatusChip';
import AgentStatusChip from '@/components/Status/AgentStatusChip';

const NodeList: React.FC = () => {
  const navigate = useNavigate();
  const [filters, setFilters] = useState({
    page: 1,
    pageSize: 20,
    status: '',
    groupId: '',
    keyword: '',
  });

  const { data, isLoading, refetch } = useQuery(
    ['nodes', filters],
    () => getNodes(filters),
    { keepPreviousData: true }
  );

  const columns: GridColDef[] = [
    { field: 'hostname', headerName: '主机名', flex: 1, minWidth: 150 },
    { field: 'ip', headerName: 'IP地址', width: 140 },
    {
      field: 'status',
      headerName: '状态',
      width: 100,
      renderCell: (params) => <NodeStatusChip status={params.value} />,
    },
    {
      field: 'agent',
      headerName: 'Agent状态',
      width: 120,
      renderCell: (params) => (
        <AgentStatusChip status={params.row.agent?.status} />
      ),
    },
    { field: 'daemonVersion', headerName: 'Daemon版本', width: 120 },
    {
      field: 'agent.version',
      headerName: 'Agent版本',
      width: 120,
      valueGetter: (params) => params.row.agent?.version || '-',
    },
    {
      field: 'group',
      headerName: '分组',
      width: 120,
      valueGetter: (params) => params.row.group?.name || '-',
    },
    {
      field: 'tags',
      headerName: '标签',
      flex: 1,
      minWidth: 200,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
          {(params.value || []).slice(0, 3).map((tag: string) => (
            <Chip key={tag} label={tag} size="small" />
          ))}
          {(params.value || []).length > 3 && (
            <Chip label={`+${params.value.length - 3}`} size="small" />
          )}
        </Box>
      ),
    },
    {
      field: 'lastHeartbeat',
      headerName: '最后心跳',
      width: 160,
      valueFormatter: (params) => formatDateTime(params.value),
    },
  ];

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="h4">节点管理</Typography>
        <Box>
          <Tooltip title="刷新">
            <IconButton onClick={() => refetch()}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      {/* 筛选栏 */}
      <Paper sx={{ p: 2, mb: 2 }}>
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
          <TextField
            size="small"
            placeholder="搜索主机名/IP"
            value={filters.keyword}
            onChange={(e) => setFilters({ ...filters, keyword: e.target.value })}
          />
          <Select
            size="small"
            value={filters.status}
            onChange={(e) => setFilters({ ...filters, status: e.target.value })}
            displayEmpty
            sx={{ minWidth: 120 }}
          >
            <MenuItem value="">全部状态</MenuItem>
            <MenuItem value="online">在线</MenuItem>
            <MenuItem value="offline">离线</MenuItem>
          </Select>
          <Select
            size="small"
            value={filters.groupId}
            onChange={(e) => setFilters({ ...filters, groupId: e.target.value })}
            displayEmpty
            sx={{ minWidth: 120 }}
          >
            <MenuItem value="">全部分组</MenuItem>
            {/* 动态加载分组 */}
          </Select>
        </Box>
      </Paper>

      {/* 数据表格 */}
      <Paper sx={{ height: 600 }}>
        <DataGrid
          rows={data?.items || []}
          columns={columns}
          loading={isLoading}
          paginationMode="server"
          rowCount={data?.total || 0}
          page={filters.page - 1}
          pageSize={filters.pageSize}
          onPageChange={(page) => setFilters({ ...filters, page: page + 1 })}
          onPageSizeChange={(pageSize) => setFilters({ ...filters, pageSize })}
          onRowClick={(params) => navigate(`/nodes/${params.id}`)}
          disableSelectionOnClick
        />
      </Paper>
    </Box>
  );
};

export default NodeList;
```

### 5.3 状态管理

```typescript
// stores/authStore.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface User {
  id: number;
  username: string;
  email: string;
  role: string;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  isAuthenticated: boolean;
  
  login: (token: string, refreshToken: string, user: User) => void;
  logout: () => void;
  updateToken: (token: string) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,

      login: (token, refreshToken, user) =>
        set({
          token,
          refreshToken,
          user,
          isAuthenticated: true,
        }),

      logout: () =>
        set({
          token: null,
          refreshToken: null,
          user: null,
          isAuthenticated: false,
        }),

      updateToken: (token) => set({ token }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        token: state.token,
        refreshToken: state.refreshToken,
        user: state.user,
      }),
    }
  )
);
```

### 5.4 API服务封装

```typescript
// services/api.ts
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import { useAuthStore } from '@/stores/authStore';

const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().token;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    const { code, message, data } = response.data;
    if (code === 0) {
      return data;
    }
    return Promise.reject(new Error(message));
  },
  async (error) => {
    if (error.response?.status === 401) {
      // Token过期，尝试刷新
      const refreshToken = useAuthStore.getState().refreshToken;
      if (refreshToken) {
        try {
          const { token } = await refreshAccessToken(refreshToken);
          useAuthStore.getState().updateToken(token);
          // 重试原请求
          error.config.headers.Authorization = `Bearer ${token}`;
          return api.request(error.config);
        } catch {
          useAuthStore.getState().logout();
          window.location.href = '/login';
        }
      }
    }
    return Promise.reject(error);
  }
);

export default api;

// services/nodeService.ts
import api from './api';

export interface NodeListParams {
  page?: number;
  pageSize?: number;
  status?: string;
  groupId?: string;
  keyword?: string;
}

export const getNodes = (params: NodeListParams) =>
  api.get('/nodes', { params });

export const getNode = (id: number) =>
  api.get(`/nodes/${id}`);

export const updateNode = (id: number, data: any) =>
  api.put(`/nodes/${id}`, data);

export const deleteNode = (id: number) =>
  api.delete(`/nodes/${id}`);

export const getNodeMetrics = (id: number, params: { start?: string; end?: string }) =>
  api.get(`/nodes/${id}/metrics`, { params });
```

---

## 6. 安全设计

### 6.1 认证授权

#### 6.1.1 JWT Token设计

```go
// Token结构
type Claims struct {
    UserID   uint64 `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}

// Token配置
type TokenConfig struct {
    AccessTokenSecret  string        // 访问令牌密钥
    RefreshTokenSecret string        // 刷新令牌密钥
    AccessTokenTTL     time.Duration // 访问令牌有效期: 2小时
    RefreshTokenTTL    time.Duration // 刷新令牌有效期: 7天
}
```

#### 6.1.2 RBAC权限矩阵

| 资源 | admin | operator | viewer |
|------|-------|----------|--------|
| 节点-查看 | ✓ | ✓ | ✓ |
| 节点-编辑 | ✓ | ✓ | ✗ |
| 节点-删除 | ✓ | ✗ | ✗ |
| 版本-查看 | ✓ | ✓ | ✓ |
| 版本-上传 | ✓ | ✓ | ✗ |
| 版本-部署 | ✓ | ✓ | ✗ |
| 版本-删除 | ✓ | ✗ | ✗ |
| 任务-查看 | ✓ | ✓ | ✓ |
| 任务-创建 | ✓ | ✓ | ✗ |
| 任务-执行 | ✓ | ✓ | ✗ |
| 任务-删除 | ✓ | ✗ | ✗ |
| 用户-管理 | ✓ | ✗ | ✗ |
| 审计日志 | ✓ | ✗ | ✗ |

### 6.2 通信安全

```yaml
# TLS配置
tls:
  enabled: true
  cert_file: /etc/manager/certs/server.crt
  key_file: /etc/manager/certs/server.key
  ca_file: /etc/manager/certs/ca.crt    # 用于mTLS
  min_version: "1.2"
  cipher_suites:
    - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
    - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
```

### 6.3 版本签名

```go
// 版本签名流程
func SignVersion(filePath string, privateKey *rsa.PrivateKey) (string, string, error) {
    // 1. 计算文件SHA256
    file, _ := os.Open(filePath)
    hasher := sha256.New()
    io.Copy(hasher, file)
    hashBytes := hasher.Sum(nil)
    hashHex := hex.EncodeToString(hashBytes)
    
    // 2. 使用RSA私钥签名
    signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashBytes)
    if err != nil {
        return "", "", err
    }
    
    signatureBase64 := base64.StdEncoding.EncodeToString(signature)
    return hashHex, signatureBase64, nil
}
```

---

## 7. 部署设计

### 7.1 Docker部署

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o manager ./cmd/manager

# 前端构建
FROM node:20-alpine AS frontend

WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# 最终镜像
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/manager .
COPY --from=frontend /app/web/dist ./web/dist
COPY configs/config.yaml ./configs/

EXPOSE 8080 9090

CMD ["./manager"]
```

### 7.2 Docker Compose

```yaml
# docker-compose.yaml
version: '3.8'

services:
  manager:
    build: .
    ports:
      - "8080:8080"   # HTTP API
      - "9090:9090"   # gRPC
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=manager
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=ops_scaffold
      - REDIS_HOST=redis
      - REDIS_PORT=6379
    volumes:
      - ./configs:/app/configs
      - ./data/versions:/app/data/versions
      - ./logs:/app/logs
    depends_on:
      - mysql
      - redis
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
      - MYSQL_DATABASE=ops_scaffold
      - MYSQL_USER=manager
      - MYSQL_PASSWORD=${DB_PASSWORD}
    volumes:
      - mysql_data:/var/lib/mysql
      - ./scripts/init_db.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "3306:3306"
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./certs:/etc/nginx/certs
    depends_on:
      - manager
    restart: unless-stopped

volumes:
  mysql_data:
  redis_data:
```

### 7.3 Kubernetes部署

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: manager
  labels:
    app: ops-scaffold-manager
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ops-scaffold-manager
  template:
    metadata:
      labels:
        app: ops-scaffold-manager
    spec:
      containers:
      - name: manager
        image: ops-scaffold/manager:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: grpc
        env:
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: manager-secrets
              key: db-host
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: manager-secrets
              key: db-password
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config
          mountPath: /app/configs
        - name: versions
          mountPath: /app/data/versions
      volumes:
      - name: config
        configMap:
          name: manager-config
      - name: versions
        persistentVolumeClaim:
          claimName: manager-versions-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: manager
spec:
  selector:
    app: ops-scaffold-manager
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: grpc
    port: 9090
    targetPort: 9090
  type: ClusterIP
```

---

## 8. 配置管理

### 8.1 配置文件

```yaml
# configs/config.yaml
server:
  http:
    addr: ":8080"
    read_timeout: 30s
    write_timeout: 30s
  grpc:
    addr: ":9090"
    max_recv_msg_size: 10485760  # 10MB
    max_send_msg_size: 10485760

database:
  driver: mysql
  host: ${DB_HOST:localhost}
  port: ${DB_PORT:3306}
  user: ${DB_USER:manager}
  password: ${DB_PASSWORD}
  name: ${DB_NAME:ops_scaffold}
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h
  
redis:
  host: ${REDIS_HOST:localhost}
  port: ${REDIS_PORT:6379}
  password: ${REDIS_PASSWORD:}
  db: 0
  pool_size: 100

auth:
  jwt:
    access_secret: ${JWT_ACCESS_SECRET}
    refresh_secret: ${JWT_REFRESH_SECRET}
    access_ttl: 2h
    refresh_ttl: 168h  # 7天
  oauth2:
    enabled: false
    provider: ""
    client_id: ""
    client_secret: ""

tls:
  enabled: ${TLS_ENABLED:false}
  cert_file: ${TLS_CERT_FILE}
  key_file: ${TLS_KEY_FILE}
  ca_file: ${TLS_CA_FILE}

storage:
  type: local  # local, s3, oss
  local:
    path: ./data/versions
  s3:
    bucket: ""
    region: ""
    access_key: ""
    secret_key: ""

node:
  heartbeat_timeout: 90s      # 心跳超时时间
  offline_check_interval: 30s # 离线检查间隔

metrics:
  retention_days: 30  # 指标数据保留天数

log:
  level: info
  format: json
  output: stdout
  file:
    enabled: false
    path: ./logs/manager.log
    max_size: 100    # MB
    max_backups: 10
    max_age: 30      # days
```

---

## 9. 监控与运维

### 9.1 Prometheus指标

```go
// 暴露的指标
var (
    // HTTP请求指标
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "manager_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "manager_http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
    
    // 节点指标
    nodesTotal = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "manager_nodes_total",
            Help: "Total number of nodes",
        },
        []string{"status"},
    )
    
    // gRPC指标
    grpcRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "manager_grpc_requests_total",
            Help: "Total number of gRPC requests",
        },
        []string{"method", "status"},
    )
)
```

### 9.2 健康检查

```go
// /health 端点
func HealthCheck(c *gin.Context) {
    checks := map[string]string{
        "mysql": checkMySQL(),
        "redis": checkRedis(),
    }
    
    healthy := true
    for _, status := range checks {
        if status != "ok" {
            healthy = false
            break
        }
    }
    
    if healthy {
        c.JSON(200, gin.H{"status": "ok", "checks": checks})
    } else {
        c.JSON(503, gin.H{"status": "unhealthy", "checks": checks})
    }
}
```

### 9.3 日志规范

```json
{
  "level": "info",
  "ts": "2025-12-01T10:00:00.000Z",
  "caller": "handler/node_handler.go:45",
  "msg": "node registered",
  "request_id": "abc123",
  "node_uuid": "xxx-xxx-xxx",
  "ip": "192.168.1.100",
  "duration_ms": 15
}
```

---

## 附录

### A. API错误码

| 错误码 | 说明 | HTTP状态码 |
|--------|------|------------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 认证失败 | 401 |
| 1003 | 权限不足 | 403 |
| 1004 | 资源不存在 | 404 |
| 1005 | 请求频率超限 | 429 |
| 2001 | 节点不存在 | 404 |
| 2002 | 节点已存在 | 409 |
| 2003 | 节点离线 | 400 |
| 3001 | 版本不存在 | 404 |
| 3002 | 版本已存在 | 409 |
| 3003 | 签名验证失败 | 400 |
| 4001 | 任务不存在 | 404 |
| 4002 | 任务执行失败 | 500 |
| 5001 | 数据库错误 | 500 |
| 5002 | 缓存错误 | 500 |

### B. 参考文档

- [Gin框架文档](https://gin-gonic.com/docs/)
- [GORM文档](https://gorm.io/docs/)
- [Material UI文档](https://mui.com/)
- [gRPC Go文档](https://grpc.io/docs/languages/go/)

---

*— 文档结束 —*
