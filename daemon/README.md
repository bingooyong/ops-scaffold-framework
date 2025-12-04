# Daemon - 运维工具框架守护进程

Daemon是运维工具框架(Ops Scaffold Framework)的守护进程组件，运行在每台被管主机上，负责系统资源采集、Agent进程管理和与Manager的通信。

## 功能特性

- ✅ **系统资源采集**: 采集CPU、内存、磁盘、网络等系统资源指标（覆盖率82.7%）
- ✅ **Agent进程管理**: 自动启动、健康检查、异常重启Agent进程
- ✅ **高可用设计**: Daemon退出不影响Agent运行，支持进程隔离
- ✅ **安全通信**: 支持TLS/mTLS加密与Manager通信
- ✅ **独立运行模式**: 支持不依赖Manager和Agent的独立运行（适合开发调试）
- ✅ **轻量级**: CPU占用<1%(空闲),内存占用<30MB
- ✅ **可配置**: 灵活的YAML配置文件，支持配置验证和默认值

## 系统要求

- Go 1.21+
- Linux (CentOS 7+, Ubuntu 18.04+, Debian 10+)
- 最低配置: 1核CPU, 512MB内存

## 快速开始

### 1. 构建

```bash
# 克隆仓库
git clone https://github.com/bingooyong/ops-scaffold-framework.git
cd ops-scaffold-framework/daemon

# 安装依赖
make deps

# 构建
make build

# 查看二进制文件
ls -lh bin/daemon
```

### 2. 配置

编辑配置文件 `configs/daemon.yaml`:

```yaml
daemon:
  log_level: info
  log_file: /var/log/daemon/daemon.log

manager:
  address: "manager.example.com:9090"
  heartbeat_interval: 60s

agent:
  binary_path: /usr/local/bin/agent
  socket_path: /var/run/agent.sock

collectors:
  cpu:
    enabled: true
    interval: 60s
  memory:
    enabled: true
    interval: 60s
```

### 3. 运行

#### 开发环境（独立运行模式）

```bash
# 使用开发配置（不依赖Manager和Agent）
./bin/daemon -config configs/daemon.dev.yaml

# 或直接运行
./bin/daemon -config configs/daemon.yaml

# 查看版本
./bin/daemon -version

# 生成测试TLS证书（可选）
./scripts/generate-test-certs.sh
```

开发配置特点：
- Manager地址留空：独立运行不连接Manager
- Agent路径留空：不管理Agent进程
- 仅运行资源采集器，每10秒采集一次
- 日志输出到`./logs/daemon.log`

#### 生产环境

```bash
# 安装
sudo make install

# 或使用安装脚本
sudo ./scripts/install.sh

# 启动服务
sudo systemctl start daemon

# 设置开机自启
sudo systemctl enable daemon

# 查看状态
sudo systemctl status daemon

# 查看日志
sudo journalctl -u daemon -f
```

## 架构设计

```
┌─────────────────────────────────────┐
│           Daemon Process             │
├─────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐         │
│  │Collector │  │  Agent   │         │
│  │ Manager  │  │ Manager  │         │
│  └─────┬────┘  └────┬─────┘         │
│        │            │               │
│  ┌─────▼────────────▼─────┐         │
│  │     gRPC Client        │         │
│  │   (to Manager)         │         │
│  └────────────────────────┘         │
└─────────────────────────────────────┘
           │
           │ gRPC/TLS
           ▼
    ┌─────────────┐
    │   Manager   │
    └─────────────┘
```

## 目录结构

```
daemon/
├── cmd/daemon/          # 主程序入口
├── internal/            # 内部模块
│   ├── agent/           # Agent管理
│   ├── collector/       # 资源采集器
│   ├── comm/            # 通信层
│   ├── config/          # 配置管理
│   ├── daemon/          # 核心引擎
│   └── logger/          # 日志系统
├── pkg/                 # 公共包
│   ├── proto/           # Protobuf定义
│   └── types/           # 类型定义
├── configs/             # 配置文件
├── scripts/             # 脚本
└── test/                # 测试
```

## 配置说明

### Daemon配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `daemon.log_level` | 日志级别 (debug/info/warn/error) | info |
| `daemon.log_file` | 日志文件路径 | /var/log/daemon/daemon.log |
| `daemon.pid_file` | PID文件路径 | /var/run/daemon.pid |
| `daemon.work_dir` | 工作目录 | /var/lib/daemon |

### Manager连接配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `manager.address` | Manager地址 | - (必填) |
| `manager.heartbeat_interval` | 心跳间隔 | 60s |
| `manager.reconnect_interval` | 重连间隔 | 10s |
| `manager.timeout` | 请求超时 | 30s |

### Agent管理配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `agent.binary_path` | Agent二进制路径 | - (必填) |
| `agent.work_dir` | Agent工作目录 | /var/lib/agent |
| `agent.socket_path` | Unix Socket路径 | /var/run/agent.sock |
| `agent.health_check.interval` | 健康检查间隔 | 30s |
| `agent.health_check.heartbeat_timeout` | 心跳超时 | 90s |

### 采集器配置

每个采集器支持以下参数:

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `enabled` | 是否启用 | true |
| `interval` | 采集间隔 | 60s |

额外参数:
- `disk.mount_points`: 监控的挂载点列表(空=全部)
- `network.interfaces`: 监控的网卡列表(空=全部)

## 开发指南

### 运行测试

```bash
# 单元测试
make test

# 测试覆盖率
make test-coverage

# 代码检查
make lint
```

### 代码格式化

```bash
make fmt
```

### 构建不同平台

```bash
# Linux AMD64
make build-linux
```

## 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| CPU占用 | < 1% | 空闲时 |
| 内存占用 | < 30MB | 不含缓存 |
| 采集耗时 | < 500ms | 单次采集 |

## 故障排查

### Daemon无法启动

1. 检查配置文件是否正确
2. 检查日志文件权限
3. 查看系统日志: `journalctl -xe`

### Agent频繁重启

1. 检查Agent日志
2. 检查资源限制配置
3. 确认Agent二进制文件正常

### 无法连接Manager

1. 检查网络连通性
2. 确认Manager地址正确
3. 检查TLS证书配置

## License

Apache License 2.0

## 贡献

欢迎提交Issue和Pull Request!
