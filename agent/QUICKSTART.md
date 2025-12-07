# Agent 快速启动指南

本指南帮助您快速启动和测试 Agent。

## 前置条件

1. **确保 Daemon 已启动**:
   - Daemon 需要创建 Unix Socket: `/tmp/daemon.sock`
   - 启动 Daemon: `cd ../daemon && make run`

2. **构建 Agent**:
   ```bash
   make build
   ```

## 启动 Agent

### 方式 1: 使用启动脚本(推荐)

```bash
# 启动所有 Agent (agent-001, agent-002, agent-003)
./scripts/agent.sh start

# 启动单个 Agent
./scripts/agent.sh start agent-001
./scripts/agent.sh start agent-002

# 查看状态
./scripts/agent.sh status

# 停止所有 Agent
./scripts/agent.sh stop

# 重启
./scripts/agent.sh restart
```

### 方式 2: 手动启动

```bash
# 启动 agent-001 (前台运行,用于调试)
./bin/agent -config ./configs/agent.yaml

# 后台运行
nohup ./bin/agent -config ./configs/agent.yaml > /tmp/agent-001.out 2>&1 &

# 启动 agent-002
nohup ./bin/agent -config ./configs/agent-002.yaml > /tmp/agent-002.out 2>&1 &

# 启动 agent-003
nohup ./bin/agent -config ./configs/agent-003.yaml > /tmp/agent-003.out 2>&1 &
```

### 方式 3: 使用环境变量

```bash
# 覆盖配置
AGENT_AGENT_ID=agent-004 AGENT_HTTP_PORT=8084 ./bin/agent -config ./configs/agent.yaml
```

## 验证 Agent 运行

### 检查进程

```bash
# 查看 Agent 进程
ps aux | grep agent

# 查看 PID 文件
cat /tmp/agent.pid
cat /tmp/agent-002.pid
cat /tmp/agent-003.pid
```

### 测试 HTTP API

```bash
# agent-001 (端口 8081)
curl http://localhost:8081/health
curl http://localhost:8081/metrics
curl -X POST http://localhost:8081/reload

# agent-002 (端口 8082)
curl http://localhost:8082/health
curl http://localhost:8082/metrics

# agent-003 (端口 8083)
curl http://localhost:8083/health
curl http://localhost:8083/metrics
```

### 查看日志

```bash
# 查看输出日志
tail -f /tmp/agent.out
tail -f /tmp/agent-002.out
tail -f /tmp/agent-003.out

# agent-003 有单独的日志文件
tail -f /tmp/agent-003.log
```

## 运行集成测试

```bash
# 确保所有 Agent 已启动
./scripts/agent.sh start

# 运行集成测试
./scripts/test-integration.sh
```

**预期输出**:
```
[INFO] =========================================
[INFO]   Agent Integration Test Suite
[INFO] =========================================

[TEST] Running: agent-001: GET /health
[INFO] ✓ PASSED: agent-001: GET /health

[TEST] Running: agent-001: GET /metrics
[INFO] ✓ PASSED: agent-001: GET /metrics

...

[INFO] =========================================
[INFO]   Test Results Summary
[INFO] =========================================
Total Tests:  15
Passed:       15
Failed:       0

[INFO] ✓ All tests passed!
```

## 多 Agent 配置说明

| Agent ID | 配置文件 | HTTP 端口 | 日志输出 |
|---------|---------|----------|---------|
| agent-001 | `configs/agent.yaml` | 8081 | stdout |
| agent-002 | `configs/agent-002.yaml` | 8082 | stdout |
| agent-003 | `configs/agent-003.yaml` | 8083 | `/tmp/agent-003.log` |

**注意**:
- 所有 Agent 共享同一个 Daemon Socket: `/tmp/daemon.sock`
- 每个 Agent 使用不同的 HTTP 端口避免冲突
- agent-003 使用文件日志,其他使用 stdout

## 常见问题

### 1. Agent 启动失败

**问题**: `failed to connect to daemon socket`

**解决**:
- 检查 Daemon 是否运行: `ls -l /tmp/daemon.sock`
- 启动 Daemon: `cd ../daemon && make run`

### 2. HTTP 端口冲突

**问题**: `bind: address already in use`

**解决**:
- 检查端口占用: `lsof -i :8081`
- 修改配置文件中的 `http.port`
- 或停止占用端口的进程

### 3. 心跳发送失败

**问题**: 日志显示 `failed to send heartbeat`

**解决**:
- 检查 Daemon Socket 权限
- 检查网络连接
- 查看 Daemon 日志

## 停止 Agent

```bash
# 使用脚本停止
./scripts/agent.sh stop

# 手动停止
kill -TERM $(cat /tmp/agent.pid)
kill -TERM $(cat /tmp/agent-002.pid)
kill -TERM $(cat /tmp/agent-003.pid)

# 强制停止
pkill -9 -f "bin/agent"
```

## 下一步

- 查看完整文档: [README.md](../README.md)
- 查看设计文档: [docs/DESIGN.md](../docs/DESIGN.md)
- 运行单元测试: `make test`
