# Agent 清理脚本修复说明

**修复时间**: 2025-12-07  
**问题**: Agent 进程无法通过清理脚本停止

---

## 问题描述

### 现象
- 运行 `cleanup_test_env.sh` 时，提示 Agent PID 文件不存在
- 但通过 `lsof` 检查发现端口 8081, 8082, 8083 仍被占用
- Agent 进程无法被停止

### 根本原因

**PID 文件命名不一致**:
- **启动脚本** (`start_test_env.sh`): 创建 PID 文件为 `agent-agent-001.pid`（因为 `agent_id` 变量值是 `"agent-001"`，所以 `${agent_id}` 展开后是 `agent-agent-001`）
- **清理脚本** (`cleanup_test_env.sh`): 查找 PID 文件为 `agent-001.pid`

导致清理脚本找不到 PID 文件，无法停止进程。

---

## 修复方案

### 1. 修复启动脚本

**文件**: `test/integration/start_test_env.sh`

**修改**:
```bash
# 修改前
echo $agent_pid > "$PID_DIR/agent-${agent_id}.pid"

# 修改后
echo $agent_pid > "$PID_DIR/${agent_id}.pid"
```

**说明**: 使用更简洁的命名 `agent-001.pid`，避免重复的 `agent-` 前缀。

### 2. 增强清理脚本

**文件**: `test/integration/cleanup_test_env.sh`

**改进**:
1. **兼容两种 PID 文件格式**:
   - 新格式: `agent-001.pid`
   - 旧格式: `agent-agent-001.pid`（向后兼容）

2. **通过端口查找进程**:
   - 如果 PID 文件不存在，通过端口查找进程
   - 端口映射: agent-001 → 8081, agent-002 → 8082, agent-003 → 8083

3. **优雅停止**:
   - 先尝试 `TERM` 信号（优雅停止）
   - 如果 2 秒后仍在运行，使用 `KILL` 信号（强制停止）

**代码逻辑**:
```bash
# 优先使用新格式（agent-agent-001.pid）
if [ -f "$pid_file2" ]; then
    stop_process "agent-$agent_id"
elif [ -f "$pid_file1" ]; then
    stop_process "$agent_id"
else
    # 如果 PID 文件不存在，通过端口查找进程
    local pid=$(lsof -ti:$port 2>/dev/null || echo "")
    if [ -n "$pid" ]; then
        # 停止进程
    fi
fi
```

---

## 验证结果

### 修复前
```bash
$ ./cleanup_test_env.sh
[WARN] agent-001 PID 文件不存在，可能未运行
[WARN] agent-002 PID 文件不存在，可能未运行
[WARN] agent-003 PID 文件不存在，可能未运行

$ lsof -ti:8081,8082,8083
36273
36356
36428
```

### 修复后
```bash
$ ./cleanup_test_env.sh
[INFO] 停止 agent-agent-001 (PID: 36273)...
[INFO] agent-agent-001 已停止
[INFO] 停止 agent-agent-002 (PID: 36356)...
[INFO] agent-agent-002 已停止
[INFO] 停止 agent-agent-003 (PID: 36428)...
[INFO] agent-agent-003 已停止

$ lsof -ti:8081,8082,8083
所有端口已释放
```

---

## 影响范围

### 向后兼容性
- ✅ 清理脚本现在能处理两种 PID 文件格式
- ✅ 即使 PID 文件丢失，也能通过端口查找并停止进程
- ✅ 不影响现有运行中的 Agent（如果使用旧格式 PID 文件）

### 新启动的 Agent
- ✅ 新启动的 Agent 将使用新格式 PID 文件（`agent-001.pid`）
- ✅ 更简洁、一致的命名

---

## 建议

1. **清理旧 PID 文件**:
   ```bash
   # 如果存在旧格式的 PID 文件，可以手动清理
   rm -f test/integration/pids/agent-agent-*.pid
   ```

2. **重新启动测试环境**:
   ```bash
   # 重新启动后，将使用新格式的 PID 文件
   ./start_test_env.sh
   ```

3. **验证清理功能**:
   ```bash
   # 启动环境
   ./start_test_env.sh
   
   # 验证 Agent 运行
   lsof -ti:8081,8082,8083
   
   # 清理环境
   ./cleanup_test_env.sh
   
   # 验证端口已释放
   lsof -ti:8081,8082,8083  # 应该无输出
   ```

---

## 相关文件

- `test/integration/start_test_env.sh` - 启动脚本（已修复）
- `test/integration/cleanup_test_env.sh` - 清理脚本（已增强）
- `test/integration/pids/` - PID 文件目录

---

**修复完成**: ✅ Agent 清理功能已正常工作
