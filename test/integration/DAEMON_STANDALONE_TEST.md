# Daemon Agent管理功能独立验证

本目录包含用于独立验证Daemon的Multi-Agent管理功能的测试脚本，**不依赖Manager**。

## 🎯 测试目标

验证Daemon的以下核心功能：
1. ✅ 从配置文件加载多个Agent
2. ✅ 自动启动已配置的Agents
3. ✅ Agent进程管理（启动/停止/重启）
4. ✅ 元数据持久化（状态、PID、启动时间等）
5. ✅ 日志记录和错误处理

## 📋 前置条件

### 1. 构建必要的二进制文件

```bash
# 构建Daemon
cd daemon && make build

# 构建测试Agent
cd agent && make build
```

### 2. 检查配置文件

确保以下配置文件存在：
- `test/integration/config/daemon.test.yaml` - Daemon配置
- `test/integration/config/agent-001.test.yaml` - Agent-001配置
- `test/integration/config/agent-002.test.yaml` - Agent-002配置
- `test/integration/config/agent-003.test.yaml` - Agent-003配置

### 3. 可选：安装grpcurl（用于高级测试）

```bash
# macOS
brew install grpcurl

# Linux
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

## 🚀 运行测试

### 方式1: 简化验证（推荐）

**最简单的验证方式**，通过检查Daemon启动后的日志、进程和元数据来验证Agent管理功能：

```bash
cd test/integration
./test_daemon_simple.sh
```

**验证内容**：
- ✓ 检查Daemon和Agent二进制是否存在
- ✓ 启动Daemon并等待Agents自动启动
- ✓ 检查所有Agent进程是否运行
- ✓ 验证Daemon日志中的Agent相关记录
- ✓ 验证元数据文件是否正确生成

**输出**：
- 终端：彩色测试结果
- 报告：`reports/daemon_standalone_test_report.md`

### 方式2: 完整测试（需要grpcurl）

**完整的Agent生命周期测试**，包括通过gRPC接口进行启动/停止/重启操作：

```bash
cd test/integration
./test_daemon_standalone.sh
```

**测试场景**：
- ✓ Agent停止（agent-002）
- ✓ Agent启动（agent-002）
- ✓ Agent重启（agent-002）
- ✓ Agent重启（agent-001）

### 方式3: Go程序测试（独立）

**纯Go测试程序**，不依赖现有配置，创建独立的测试环境：

```bash
cd daemon/test/standalone
go run test_agent_lifecycle.go -verbose -workdir /tmp/daemon-test-$(date +%s)
```

**可选参数**：
- `-workdir`: 工作目录（默认：/tmp/daemon-test）
- `-type`: 测试类型（start|stop|restart|status|all，默认：all）
- `-agent`: Agent ID（默认：test-agent-001）
- `-verbose`: 详细日志
- `-cleanup`: 测试后清理（默认：true）

## 📊 查看测试结果

### 查看测试报告

```bash
# 简化测试报告
cat test/integration/reports/daemon_standalone_test_report.md

# 或在浏览器中打开
open test/integration/reports/daemon_standalone_test_report.md  # macOS
xdg-open test/integration/reports/daemon_standalone_test_report.md  # Linux
```

### 查看日志

```bash
# Daemon日志
tail -f test/integration/logs/daemon.log

# 过滤Agent相关日志
tail -f test/integration/logs/daemon.log | grep -i agent

# Agent日志（如果存在）
tail -f test/integration/logs/agent-*.log
```

### 查看元数据

```bash
# 查看所有Agent元数据
cat test/integration/tmp/daemon/metadata/*.json | jq '.'

# 查看特定Agent
cat test/integration/tmp/daemon/metadata/agent-001.json | jq '.'
```

### 检查进程

```bash
# 查看Daemon进程
ps aux | grep daemon | grep -v grep

# 查看Agent进程
ps aux | grep "agent/bin/agent" | grep -v grep

# 查看进程树
pstree -p $(cat test/integration/pids/daemon.pid) 2>/dev/null || ps -ef | grep daemon
```

## 🔍 故障排查

### 问题1: Agent进程未启动

**症状**：测试显示Agent未运行

**检查步骤**：
```bash
# 1. 检查Agent二进制是否存在
ls -lh agent/bin/agent

# 2. 检查Agent二进制是否可执行
./agent/bin/agent -version

# 3. 查看Daemon日志中的错误
grep -i "error" test/integration/logs/daemon.log | grep -i agent

# 4. 检查配置文件路径
cat test/integration/config/daemon.test.yaml | grep binary_path
```

**常见原因**：
- Agent二进制未构建或路径不正确
- 配置文件中的binary_path指向错误位置
- Agent工作目录权限问题

### 问题2: Daemon无法启动

**症状**：Daemon进程立即退出

**检查步骤**：
```bash
# 1. 查看Daemon日志
cat test/integration/logs/daemon.log

# 2. 手动启动Daemon查看错误
cd daemon
./daemon -config ../test/integration/config/daemon.test.yaml

# 3. 检查端口占用
lsof -i :9091  # gRPC端口
```

**常见原因**：
- 端口被占用（9091）
- 配置文件格式错误
- 必要目录权限不足

### 问题3: 元数据文件未生成

**症状**：`test/integration/tmp/daemon/metadata/` 目录为空

**检查步骤**：
```bash
# 1. 确认目录存在
ls -la test/integration/tmp/daemon/

# 2. 检查Daemon是否有写权限
touch test/integration/tmp/daemon/metadata/test.txt && rm test/integration/tmp/daemon/metadata/test.txt

# 3. 查看Daemon日志中的元数据相关错误
grep -i "metadata" test/integration/logs/daemon.log
```

### 问题4: grpcurl命令失败

**症状**：完整测试无法通过gRPC调用Agent操作

**解决方案**：
1. 使用简化测试（不需要grpcurl）：`./test_daemon_simple.sh`
2. 或安装grpcurl：
   ```bash
   # macOS
   brew install grpcurl
   
   # Linux/macOS (Go方式)
   go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
   ```

## 📝 测试输出示例

### 成功输出

```
========================================
Daemon Agent Management Standalone Test
简化版本 - 通过日志和进程验证
========================================

[1/6] Checking Agent binary...
✓ Agent binary exists: /path/to/agent/bin/agent

[2/6] Checking Daemon binary...
✓ Daemon binary exists: /path/to/daemon/daemon

[3/6] Starting Daemon...
✓ Daemon started (PID: 12345)
✓ Daemon is running

[4/6] Checking Agent processes...
Agent agent-001:
  - PID: 12346
  - Status: running
  ✓ Process is running

Agent agent-002:
  - PID: 12347
  - Status: running
  ✓ Process is running

Agent agent-003:
  - PID: 12348
  - Status: running
  ✓ Process is running

[5/6] Checking Daemon logs...
  - Agents registered: 3
  - Agents started: 3
  - MultiAgentManager mentions: 5
  - Error logs: 0
✓ Daemon logs look good

[6/6] Checking metadata files...
✓ Metadata exists: agent-001
✓ Metadata exists: agent-002
✓ Metadata exists: agent-003

✅ All tests PASSED!
Daemon的Agent管理功能验证成功！

========================================
测试报告: test/integration/reports/daemon_standalone_test_report.md
========================================
```

## 🎓 下一步

测试通过后，可以进行以下工作：

1. **集成Manager测试**：运行完整的Manager-Daemon集成测试
   ```bash
   cd test/integration
   ./test_business_flows.sh
   ```

2. **性能测试**：验证大量Agent场景下的性能
   ```bash
   cd test/integration
   ./test_performance.sh
   ```

3. **错误场景测试**：测试异常情况处理
   ```bash
   cd test/integration
   ./test_error_scenarios.sh
   ```

## 🔧 清理环境

测试完成后清理环境：

```bash
# 停止所有进程
pkill -f "daemon/daemon"
pkill -f "agent/bin/agent"

# 清理临时文件
rm -rf test/integration/tmp/*
rm -rf test/integration/logs/*
rm -rf test/integration/pids/*

# 清理Unix Socket
rm -f /tmp/daemon.sock
```

## 📚 相关文档

- [Daemon设计文档](../../docs/设计文档_01_Daemon模块.md)
- [Multi-Agent架构设计](../../docs/设计文档_04_Daemon多Agent管理架构.md)
- [集成测试README](./README.md)
- [Agent管理使用指南](../../docs/Agent管理功能使用指南.md)

---

**最后更新**: 2025-12-07
**维护者**: Development Team
