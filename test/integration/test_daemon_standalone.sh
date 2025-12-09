#!/bin/bash
# Daemon Agent管理功能独立验证脚本
# 此脚本独立测试Daemon的Agent启停管理功能，不依赖Manager

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
INTEGRATION_DIR="$PROJECT_ROOT/test/integration"
CONFIG_DIR="$INTEGRATION_DIR/config"
LOGS_DIR="$INTEGRATION_DIR/logs"
TMP_DIR="$INTEGRATION_DIR/tmp"
PIDS_DIR="$INTEGRATION_DIR/pids"

# 确保必要目录存在
mkdir -p "$LOGS_DIR" "$PIDS_DIR" "$TMP_DIR/daemon/metadata"

# 清理函数
cleanup() {
    echo -e "${YELLOW}Cleaning up...${NC}"
    
    # 停止Daemon
    if [ -f "$PIDS_DIR/daemon.pid" ]; then
        DAEMON_PID=$(cat "$PIDS_DIR/daemon.pid" 2>/dev/null || echo "")
        if [ -n "$DAEMON_PID" ] && kill -0 "$DAEMON_PID" 2>/dev/null; then
            echo "Stopping Daemon (PID: $DAEMON_PID)..."
            kill -TERM "$DAEMON_PID" 2>/dev/null || true
            sleep 2
            # 如果还在运行，强制杀死
            if kill -0 "$DAEMON_PID" 2>/dev/null; then
                kill -9 "$DAEMON_PID" 2>/dev/null || true
            fi
        fi
        rm -f "$PIDS_DIR/daemon.pid"
    fi
    
    # 清理Agent进程
    echo "Cleaning up Agent processes..."
    pkill -f "agent/bin/agent" 2>/dev/null || true
    
    # 清理Unix Socket
    rm -f /tmp/daemon.sock
    
    echo -e "${GREEN}Cleanup completed${NC}"
}

# 设置trap
trap cleanup EXIT INT TERM

# 检查Agent二进制是否存在
check_agent_binary() {
    local agent_bin="$PROJECT_ROOT/agent/bin/agent"
    if [ ! -f "$agent_bin" ]; then
        echo -e "${RED}✗ Agent binary not found: $agent_bin${NC}"
        echo "Building agent binary..."
        cd "$PROJECT_ROOT/agent" && make build
        if [ ! -f "$agent_bin" ]; then
            echo -e "${RED}Failed to build agent binary${NC}"
            exit 1
        fi
    fi
    echo -e "${GREEN}✓ Agent binary exists: $agent_bin${NC}"
}

# 检查Daemon二进制是否存在
check_daemon_binary() {
    local daemon_bin="$PROJECT_ROOT/daemon/daemon"
    if [ ! -f "$daemon_bin" ]; then
        echo -e "${RED}✗ Daemon binary not found: $daemon_bin${NC}"
        echo "Building daemon binary..."
        cd "$PROJECT_ROOT/daemon" && make build
        if [ ! -f "$daemon_bin" ]; then
            echo -e "${RED}Failed to build daemon binary${NC}"
            exit 1
        fi
    fi
    echo -e "${GREEN}✓ Daemon binary exists: $daemon_bin${NC}"
}

# 启动Daemon
start_daemon() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Starting Daemon${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    cd "$PROJECT_ROOT/daemon"
    
    # 清理旧日志
    > "$LOGS_DIR/daemon.log"
    
    # 启动Daemon（不连接Manager）
    ./daemon -config "$CONFIG_DIR/daemon.test.yaml" \
        > "$LOGS_DIR/daemon.log" 2>&1 &
    
    DAEMON_PID=$!
    echo $DAEMON_PID > "$PIDS_DIR/daemon.pid"
    
    echo -e "${GREEN}✓ Daemon started (PID: $DAEMON_PID)${NC}"
    
    # 等待Daemon启动
    echo "Waiting for Daemon to start..."
    sleep 3
    
    # 检查Daemon是否运行
    if ! kill -0 "$DAEMON_PID" 2>/dev/null; then
        echo -e "${RED}✗ Daemon failed to start${NC}"
        echo "Last 20 lines of daemon.log:"
        tail -20 "$LOGS_DIR/daemon.log"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Daemon is running${NC}"
}

# 检查Agent进程
check_agent_process() {
    local agent_id=$1
    local expected_status=$2  # "running" or "stopped"
    
    # 从metadata文件获取PID
    local metadata_file="$TMP_DIR/daemon/metadata/${agent_id}.json"
    
    if [ ! -f "$metadata_file" ]; then
        if [ "$expected_status" = "stopped" ]; then
            return 0
        else
            echo -e "${RED}✗ Metadata file not found for $agent_id${NC}"
            return 1
        fi
    fi
    
    local pid=$(jq -r '.pid // 0' "$metadata_file" 2>/dev/null || echo "0")
    local status=$(jq -r '.status // "unknown"' "$metadata_file" 2>/dev/null || echo "unknown")
    
    echo "Agent $agent_id: PID=$pid, Status=$status"
    
    if [ "$expected_status" = "running" ]; then
        if [ "$pid" -gt 0 ] && kill -0 "$pid" 2>/dev/null; then
            echo -e "${GREEN}✓ Agent $agent_id is running (PID: $pid)${NC}"
            return 0
        else
            echo -e "${RED}✗ Agent $agent_id is NOT running${NC}"
            return 1
        fi
    else
        if [ "$pid" -eq 0 ] || ! kill -0 "$pid" 2>/dev/null; then
            echo -e "${GREEN}✓ Agent $agent_id is stopped${NC}"
            return 0
        else
            echo -e "${RED}✗ Agent $agent_id is STILL running (PID: $pid)${NC}"
            return 1
        fi
    fi
}

# 测试Agent启动
test_agent_start() {
    local agent_id=$1
    echo -e "\n${BLUE}=== Test: Start Agent $agent_id ===${NC}"
    
    # 使用gRPC调用启动Agent
    echo "Sending start request via gRPC..."
    
    # 使用grpcurl调用（如果安装了）
    if command -v grpcurl &> /dev/null; then
        grpcurl -plaintext -d "{\"agent_id\": \"$agent_id\", \"operation\": \"start\"}" \
            localhost:9091 proto.DaemonService/OperateAgent
    else
        echo -e "${YELLOW}⚠ grpcurl not installed, checking logs instead${NC}"
    fi
    
    # 等待Agent启动
    sleep 2
    
    # 检查Agent进程
    if check_agent_process "$agent_id" "running"; then
        echo -e "${GREEN}✓ Test passed: Agent $agent_id started successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ Test failed: Agent $agent_id did not start${NC}"
        return 1
    fi
}

# 测试Agent停止
test_agent_stop() {
    local agent_id=$1
    echo -e "\n${BLUE}=== Test: Stop Agent $agent_id ===${NC}"
    
    # 使用gRPC调用停止Agent
    echo "Sending stop request via gRPC..."
    
    if command -v grpcurl &> /dev/null; then
        grpcurl -plaintext -d "{\"agent_id\": \"$agent_id\", \"operation\": \"stop\"}" \
            localhost:9091 proto.DaemonService/OperateAgent
    else
        echo -e "${YELLOW}⚠ grpcurl not installed, checking logs instead${NC}"
    fi
    
    # 等待Agent停止
    sleep 2
    
    # 检查Agent进程
    if check_agent_process "$agent_id" "stopped"; then
        echo -e "${GREEN}✓ Test passed: Agent $agent_id stopped successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ Test failed: Agent $agent_id did not stop${NC}"
        return 1
    fi
}

# 测试Agent重启
test_agent_restart() {
    local agent_id=$1
    echo -e "\n${BLUE}=== Test: Restart Agent $agent_id ===${NC}"
    
    # 获取当前PID
    local metadata_file="$TMP_DIR/daemon/metadata/${agent_id}.json"
    local old_pid=$(jq -r '.pid // 0' "$metadata_file" 2>/dev/null || echo "0")
    echo "Old PID: $old_pid"
    
    # 使用gRPC调用重启Agent
    echo "Sending restart request via gRPC..."
    
    if command -v grpcurl &> /dev/null; then
        grpcurl -plaintext -d "{\"agent_id\": \"$agent_id\", \"operation\": \"restart\"}" \
            localhost:9091 proto.DaemonService/OperateAgent
    else
        echo -e "${YELLOW}⚠ grpcurl not installed, checking logs instead${NC}"
    fi
    
    # 等待Agent重启
    sleep 3
    
    # 检查Agent进程
    local new_pid=$(jq -r '.pid // 0' "$metadata_file" 2>/dev/null || echo "0")
    echo "New PID: $new_pid"
    
    if [ "$new_pid" -gt 0 ] && [ "$new_pid" -ne "$old_pid" ] && kill -0 "$new_pid" 2>/dev/null; then
        echo -e "${GREEN}✓ Test passed: Agent $agent_id restarted successfully (Old PID: $old_pid, New PID: $new_pid)${NC}"
        return 0
    else
        echo -e "${RED}✗ Test failed: Agent $agent_id did not restart properly${NC}"
        return 1
    fi
}

# 查看Daemon日志摘要
show_daemon_logs() {
    echo -e "\n${BLUE}=== Daemon Logs Summary ===${NC}"
    echo "Last 30 lines of daemon.log:"
    tail -30 "$LOGS_DIR/daemon.log" | grep -E "(agent|OperateAgent|starting|stopped|failed)" || echo "(No relevant log entries)"
}

# 主测试流程
main() {
    echo -e "${BLUE}========================================"
    echo "Daemon Agent Management Standalone Test"
    echo -e "========================================${NC}\n"
    
    # 步骤1: 检查二进制文件
    echo -e "${BLUE}[1/6] Checking binaries...${NC}"
    check_daemon_binary
    check_agent_binary
    echo ""
    
    # 步骤2: 清理环境
    echo -e "${BLUE}[2/6] Cleaning environment...${NC}"
    cleanup
    sleep 1
    echo ""
    
    # 步骤3: 启动Daemon
    echo -e "${BLUE}[3/6] Starting Daemon...${NC}"
    start_daemon
    echo ""
    
    # 步骤4: 等待Agents自动启动（根据配置文件）
    echo -e "${BLUE}[4/6] Waiting for Agents to auto-start...${NC}"
    sleep 5
    
    # 检查初始状态
    echo "Checking initial Agent states..."
    check_agent_process "agent-001" "running" || true
    check_agent_process "agent-002" "running" || true
    echo ""
    
    # 步骤5: 运行测试
    echo -e "${BLUE}[5/6] Running Agent lifecycle tests...${NC}"
    
    local test_failed=0
    
    # 测试agent-002的停止、启动、重启
    test_agent_stop "agent-002" || test_failed=1
    sleep 1
    
    test_agent_start "agent-002" || test_failed=1
    sleep 1
    
    test_agent_restart "agent-002" || test_failed=1
    sleep 1
    
    # 测试agent-001的重启
    test_agent_restart "agent-001" || test_failed=1
    sleep 1
    
    echo ""
    
    # 步骤6: 显示日志摘要
    echo -e "${BLUE}[6/6] Test Summary${NC}"
    show_daemon_logs
    
    # 最终结果
    echo -e "\n${BLUE}========================================${NC}"
    if [ $test_failed -eq 0 ]; then
        echo -e "${GREEN}✅ All tests PASSED!${NC}"
        echo -e "${BLUE}========================================${NC}"
        return 0
    else
        echo -e "${RED}❌ Some tests FAILED!${NC}"
        echo -e "${BLUE}========================================${NC}"
        
        echo -e "\n${YELLOW}Troubleshooting:${NC}"
        echo "1. Check daemon logs: cat $LOGS_DIR/daemon.log"
        echo "2. Check agent logs: cat $LOGS_DIR/agent-*.log"
        echo "3. Check metadata: cat $TMP_DIR/daemon/metadata/*.json"
        echo "4. Check if grpcurl is installed: brew install grpcurl (macOS)"
        
        return 1
    fi
}

# 运行主程序
main
exit_code=$?

# 保持Daemon运行一段时间以便检查
if [ $exit_code -eq 0 ]; then
    echo -e "\n${YELLOW}Daemon is still running. Press Ctrl+C to stop, or wait 10 seconds...${NC}"
    sleep 10
fi

exit $exit_code
