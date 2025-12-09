# 完整系统集成测试环境

本目录包含完整系统集成测试的环境搭建脚本和配置文件。

## 目录结构

```
test/integration/
├── start_test_env.sh          # 测试环境启动脚本
├── cleanup_test_env.sh        # 测试环境清理脚本
├── verify_test_env.sh         # 测试环境验证脚本
├── config/                     # 测试配置文件目录
│   ├── manager.test.yaml      # Manager 测试配置
│   ├── daemon.test.yaml       # Daemon 测试配置
│   ├── agent-001.test.yaml    # Agent-001 测试配置
│   ├── agent-002.test.yaml    # Agent-002 测试配置
│   └── agent-003.test.yaml    # Agent-003 测试配置
├── logs/                      # 日志文件目录（自动创建）
├── pids/                      # PID 文件目录（自动创建）
└── tmp/                       # 临时文件目录（自动创建）
```

## 快速开始

### 1. 启动测试环境

```bash
cd test/integration
./start_test_env.sh
```

此脚本会：
- 检查 MySQL 数据库连接
- 构建并启动 Manager 服务（HTTP: 8080, gRPC: 9090）
- 构建并启动 Daemon 服务（gRPC: 9091）
- 构建并启动 3 个测试 Agent 实例（HTTP: 8081, 8082, 8083）
- 验证所有服务健康状态

### 2. 验证测试环境

```bash
./verify_test_env.sh
```

此脚本会：
- 检查所有服务进程状态
- 检查服务端口监听状态
- 检查 HTTP 健康检查端点
- 生成验证报告：`test_env_verification_report.md`

### 3. 清理测试环境

```bash
# 停止所有服务
./cleanup_test_env.sh

# 停止服务并清理日志
./cleanup_test_env.sh --clean-logs

# 停止服务并清理所有临时文件
./cleanup_test_env.sh --clean-all
```

## 服务配置

### Manager 服务
- **HTTP API**: http://127.0.0.1:8080
- **gRPC**: 127.0.0.1:9090
- **配置文件**: `config/manager.test.yaml`
- **日志文件**: `logs/manager.log`

### Daemon 服务
- **gRPC**: 127.0.0.1:9091
- **配置文件**: `config/daemon.test.yaml`
- **日志文件**: `logs/daemon.log`
- **管理的 Agent**: agent-001, agent-002, agent-003

### Agent 服务
- **Agent-001**: http://127.0.0.1:8081
- **Agent-002**: http://127.0.0.1:8082
- **Agent-003**: http://127.0.0.1:8083
- **配置文件**: `config/agent-*.test.yaml`
- **日志文件**: `logs/agent-*.log`

## 前置要求

1. **MySQL 8.0+**
   - 运行在 127.0.0.1:3306
   - 用户: root
   - 密码: rootpassword
   - 数据库: ops_manager_dev

2. **Go 1.24.0+**
   - 用于构建服务

3. **必要命令**
   - `mysql`: MySQL 客户端
   - `curl`: HTTP 客户端
   - `lsof`: 端口检查工具

## 注意事项

1. **端口占用**: 确保以下端口未被占用：
   - 8080 (Manager HTTP)
   - 8081, 8082, 8083 (Agent HTTP)
   - 9090 (Manager gRPC)
   - 9091 (Daemon gRPC)

2. **配置文件路径**: 所有配置文件路径都是相对于项目根目录的，确保在项目根目录下运行服务。

3. **Agent 管理**: Agent 可以由 Daemon 自动管理（根据 `daemon.test.yaml` 配置），也可以独立启动（用于某些测试场景）。

4. **日志查看**: 所有服务日志保存在 `logs/` 目录下，可用于调试。

## 故障排查

### Manager 启动失败
- 检查 MySQL 是否运行
- 检查端口 8080 是否被占用
- 查看日志: `logs/manager.log`

### Daemon 启动失败
- 检查 Manager gRPC 是否可访问
- 检查端口 9091 是否被占用
- 查看日志: `logs/daemon.log`

### Agent 启动失败
- 检查 Agent 二进制文件是否存在: `agent/bin/agent`
- 检查配置文件路径是否正确
- 检查端口是否被占用
- 查看日志: `logs/agent-*.log`

## 测试脚本

### 业务流程测试

**文件**: `test/integration/test_business_flows.sh`

**功能**: 测试从 Daemon 启动到 Web 前端显示的完整业务流程

**测试场景**:
1. Agent 注册和发现
2. Agent 操作流程（启动/停止/重启）
3. 状态同步流程
4. 日志查看流程
5. 监控图表流程

**使用方法**:
```bash
# 确保测试环境已启动
./start_test_env.sh

# 运行业务流程测试
./test_business_flows.sh

# 查看测试报告
cat reports/business_flows_test_report.md
```

**前置条件**:
- 测试环境已启动（Manager、Daemon、Agents）
- Manager HTTP API 可访问（http://127.0.0.1:8080）
- 必要工具：`curl`、`python3`（可选）

## 测试脚本

### 业务流程测试

**文件**: `test/integration/test_business_flows.sh`

**功能**: 测试从 Daemon 启动到 Web 前端显示的完整业务流程

**测试场景**:
1. Agent 注册和发现
2. Agent 操作流程（启动/停止/重启）
3. 状态同步流程
4. 日志查看流程
5. 监控图表流程

**使用方法**:
```bash
./test_business_flows.sh
```

### 异常场景测试

**文件**: `test/integration/test_error_scenarios.sh`

**功能**: 测试系统在异常情况下的行为和恢复能力

**测试场景**:
1. Daemon 断线重连
2. Agent 异常退出自动重启
3. 网络延迟和超时
4. 并发操作冲突
5. 数据库连接失败（模拟）

**使用方法**:
```bash
./test_error_scenarios.sh
```

### 性能测试

**文件**: `test/integration/test_performance.sh`

**功能**: 测试系统在高负载下的性能表现

**测试场景**:
1. 多 Agent 并发运行
2. 高频心跳处理
3. 批量操作性能
4. Web 前端性能

**使用方法**:
```bash
./test_performance.sh
```

### 集成测试报告

**文件**: `test/integration/generate_integration_report.sh`

**功能**: 汇总所有测试结果，生成完整的集成测试报告

**使用方法**:
```bash
# 先运行所有测试
./test_business_flows.sh
./test_error_scenarios.sh
./test_performance.sh

# 生成集成报告
./generate_integration_report.sh
```

### 诊断和修复

**文件**: `test/integration/diagnose_and_fix.sh`

**功能**: 诊断测试环境问题并自动修复

**使用方法**:
```bash
./diagnose_and_fix.sh
```

## 测试报告

所有测试报告保存在 `reports/` 目录下：

- `business_flows_test_report.md` - 业务流程测试报告
- `error_scenarios_test_report.md` - 异常场景测试报告
- `performance_test_report.md` - 性能测试报告
- `integration_test_report.md` - 完整集成测试报告

## 下一步

测试环境搭建完成后，可以继续执行：
- ✅ Step 1: 搭建完整测试环境（已完成）
- ✅ Step 2: 执行完整业务流程测试（已完成）
- ✅ Step 3: 测试异常场景（已完成）
- ✅ Step 4: 性能测试（已完成）
- ✅ Step 5: 生成集成测试报告（已完成）

**所有测试脚本已创建完成！**
