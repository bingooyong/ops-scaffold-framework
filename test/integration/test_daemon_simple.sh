#!/bin/bash
# Daemon Agent管理功能简化验证脚本
# 直接检查Daemon启动时的Agent管理功能（通过日志和进程验证）

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

# 测试报告文件
REPORT_FILE="$INTEGRATION_DIR/reports/daemon_standalone_test_report.md"

# 初始化报告
init_report() {
    mkdir -p "$(dirname "$REPORT_FILE")"
    cat > "$REPORT_FILE" << EOF
# Daemon Agent管理独立测试报告

**测试时间**: $(date '+%Y-%m-%d %H:%M:%S')
**测试环境**: 独立Daemon测试（不依赖Manager）
**配置文件**: $CONFIG_DIR/daemon.test.yaml

---

## 测试目标

验证Daemon的Multi-Agent管理功能是否正常工作：
1. ✓ Agent自动启动
2. ✓ Agent进程管理
3. ✓ 元数据持久化
4. ✓ 日志记录

---

## 测试步骤

EOF
}

# 添加测试步骤到报告
add_step() {
    local step_num=$1
    local step_name=$2
    local status=$3  # PASS/FAIL/SKIP
    local details=$4
    
    local icon
    case $status in
        PASS) icon="✅" ;;
        FAIL) icon="❌" ;;
        SKIP) icon="⏭️" ;;
        *) icon="📝" ;;
    esac
    
    cat >> "$REPORT_FILE" << EOF
### $step_num. $step_name - $icon $status

$details

EOF
}

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
            sleep 3
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
            add_step "1" "检查Agent二进制" "FAIL" "Agent二进制构建失败"
            exit 1
        fi
    fi
    echo -e "${GREEN}✓ Agent binary exists: $agent_bin${NC}"
    add_step "1" "检查Agent二进制" "PASS" "Agent二进制存在: \`$agent_bin\`"
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
            add_step "2" "检查Daemon二进制" "FAIL" "Daemon二进制构建失败"
            exit 1
        fi
    fi
    echo -e "${GREEN}✓ Daemon binary exists: $daemon_bin${NC}"
    add_step "2" "检查Daemon二进制" "PASS" "Daemon二进制存在: \`$daemon_bin\`"
}

# 启动Daemon
start_daemon() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Starting Daemon${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    cd "$PROJECT_ROOT/daemon"
    
    # 清理旧日志
    > "$LOGS_DIR/daemon.log"
    
    # 启动Daemon
    ./daemon -config "$CONFIG_DIR/daemon.test.yaml" \
        > "$LOGS_DIR/daemon.log" 2>&1 &
    
    DAEMON_PID=$!
    echo $DAEMON_PID > "$PIDS_DIR/daemon.pid"
    
    echo -e "${GREEN}✓ Daemon started (PID: $DAEMON_PID)${NC}"
    
    # 等待Daemon启动并加载Agents
    echo "Waiting for Daemon to initialize Agents..."
    sleep 5
    
    # 检查Daemon是否运行
    if ! kill -0 "$DAEMON_PID" 2>/dev/null; then
        echo -e "${RED}✗ Daemon failed to start${NC}"
        echo "Last 20 lines of daemon.log:"
        tail -20 "$LOGS_DIR/daemon.log"
        add_step "3" "启动Daemon" "FAIL" "Daemon进程启动失败\n\n\`\`\`\n$(tail -20 "$LOGS_DIR/daemon.log")\n\`\`\`"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Daemon is running${NC}"
    add_step "3" "启动Daemon" "PASS" "Daemon成功启动 (PID: $DAEMON_PID)"
}

# 检查Agent进程状态
check_agent_processes() {
    echo -e "\n${BLUE}=== Checking Agent Processes ===${NC}"
    
    local all_passed=true
    local details=""
    
    for agent_id in agent-001 agent-002 agent-003; do
        local metadata_file="$TMP_DIR/daemon/metadata/${agent_id}.json"
        
        if [ ! -f "$metadata_file" ]; then
            echo -e "${RED}✗ No metadata file for $agent_id${NC}"
            details+="- **$agent_id**: ❌ 元数据文件不存在\n"
            all_passed=false
            continue
        fi
        
        local pid=$(jq -r '.pid // 0' "$metadata_file" 2>/dev/null || echo "0")
        local status=$(jq -r '.status // "unknown"' "$metadata_file" 2>/dev/null || echo "unknown")
        local start_time=$(jq -r '.start_time // "N/A"' "$metadata_file" 2>/dev/null || echo "N/A")
        
        echo "Agent $agent_id:"
        echo "  - PID: $pid"
        echo "  - Status: $status"
        echo "  - Start Time: $start_time"
        
        if [ "$pid" -gt 0 ] && kill -0 "$pid" 2>/dev/null; then
            echo -e "${GREEN}  ✓ Process is running${NC}"
            details+="- **$agent_id**: ✅ 运行中 (PID: $pid, Status: $status)\n"
        else
            echo -e "${RED}  ✗ Process is NOT running${NC}"
            details+="- **$agent_id**: ❌ 未运行 (PID: $pid, Status: $status)\n"
            all_passed=false
        fi
        
        echo ""
    done
    
    if $all_passed; then
        add_step "4" "检查Agent进程" "PASS" "$details"
        return 0
    else
        add_step "4" "检查Agent进程" "FAIL" "$details"
        return 1
    fi
}

# 检查Daemon日志
check_daemon_logs() {
    echo -e "\n${BLUE}=== Checking Daemon Logs ===${NC}"
    
    local log_file="$LOGS_DIR/daemon.log"
    local details=""
    
    # 检查关键日志条目
    echo "Checking for key log entries..."
    
    # 1. Agent注册日志
    local agent_registered=$(grep -c "agent registered" "$log_file" || echo "0")
    echo "  - Agents registered: $agent_registered"
    details+="**Agent注册日志**: 发现 $agent_registered 条\n\n"
    
    # 2. Agent启动日志
    local agent_started=$(grep -c "agent started" "$log_file" || echo "0")
    echo "  - Agents started: $agent_started"
    details+="**Agent启动日志**: 发现 $agent_started 条\n\n"
    
    # 3. MultiAgentManager初始化
    local manager_init=$(grep -c "MultiAgentManager" "$log_file" || echo "0")
    echo "  - MultiAgentManager mentions: $manager_init"
    details+="**MultiAgentManager日志**: 发现 $manager_init 条\n\n"
    
    # 4. 错误日志
    local errors=$(grep -c '"level":"error"' "$log_file" || echo "0")
    echo "  - Error logs: $errors"
    if [ "$errors" -gt 0 ]; then
        details+="**错误日志**: ⚠️ 发现 $errors 条错误\n\n"
        details+="\`\`\`\n$(grep '"level":"error"' "$log_file" | tail -5)\n\`\`\`\n\n"
    else
        details+="**错误日志**: ✅ 无错误\n\n"
    fi
    
    # 提取最近的Agent相关日志
    details+="**最近的Agent相关日志** (最后10条):\n\n\`\`\`\n"
    details+="$(grep -i "agent" "$log_file" | tail -10)\n\`\`\`\n"
    
    if [ "$agent_registered" -ge 3 ] && [ "$agent_started" -ge 3 ] && [ "$errors" -eq 0 ]; then
        echo -e "${GREEN}✓ Daemon logs look good${NC}"
        add_step "5" "检查Daemon日志" "PASS" "$details"
        return 0
    else
        echo -e "${YELLOW}⚠ Daemon logs show potential issues${NC}"
        add_step "5" "检查Daemon日志" "FAIL" "$details"
        return 1
    fi
}

# 检查元数据文件
check_metadata_files() {
    echo -e "\n${BLUE}=== Checking Metadata Files ===${NC}"
    
    local all_exist=true
    local details=""
    
    for agent_id in agent-001 agent-002 agent-003; do
        local metadata_file="$TMP_DIR/daemon/metadata/${agent_id}.json"
        
        if [ -f "$metadata_file" ]; then
            echo -e "${GREEN}✓ Metadata exists: $agent_id${NC}"
            details+="**$agent_id**: ✅ 元数据文件存在\n\n"
            details+="\`\`\`json\n$(cat "$metadata_file" | jq '.' 2>/dev/null || cat "$metadata_file")\n\`\`\`\n\n"
        else
            echo -e "${RED}✗ Metadata missing: $agent_id${NC}"
            details+="**$agent_id**: ❌ 元数据文件不存在\n\n"
            all_exist=false
        fi
    done
    
    if $all_exist; then
        add_step "6" "检查元数据文件" "PASS" "$details"
        return 0
    else
        add_step "6" "检查元数据文件" "FAIL" "$details"
        return 1
    fi
}

# 生成最终报告
finalize_report() {
    local overall_status=$1
    
    cat >> "$REPORT_FILE" << EOF

---

## 测试结果

EOF
    
    if [ "$overall_status" = "PASS" ]; then
        cat >> "$REPORT_FILE" << EOF
### ✅ 测试通过

所有Daemon的Multi-Agent管理功能测试均通过：
- ✓ Agent自动注册和启动
- ✓ 进程管理正常
- ✓ 元数据持久化工作正常
- ✓ 日志记录完整

**结论**: Daemon的Agent管理功能基本实现正常。

EOF
    else
        cat >> "$REPORT_FILE" << EOF
### ❌ 测试失败

部分测试未通过，请检查：
1. Daemon日志: \`$LOGS_DIR/daemon.log\`
2. Agent日志: \`$LOGS_DIR/agent-*.log\`
3. 元数据文件: \`$TMP_DIR/daemon/metadata/*.json\`

**下一步**:
- 检查Agent二进制是否正确构建
- 检查配置文件路径是否正确
- 查看详细的错误日志

EOF
    fi
    
    cat >> "$REPORT_FILE" << EOF

---

## 附录

### Daemon配置
\`\`\`yaml
$(cat "$CONFIG_DIR/daemon.test.yaml")
\`\`\`

### 环境信息
- 操作系统: $(uname -s)
- 架构: $(uname -m)
- Go版本: $(go version 2>/dev/null || echo "N/A")

EOF
    
    echo -e "\n${BLUE}测试报告已生成: $REPORT_FILE${NC}"
}

# 主测试流程
main() {
    echo -e "${BLUE}========================================"
    echo "Daemon Agent Management Standalone Test"
    echo "简化版本 - 通过日志和进程验证"
    echo -e "========================================${NC}\n"
    
    # 初始化报告
    init_report
    
    # 测试步骤
    local test_passed=true
    
    echo -e "${BLUE}[1/6] Checking Agent binary...${NC}"
    check_agent_binary || test_passed=false
    echo ""
    
    echo -e "${BLUE}[2/6] Checking Daemon binary...${NC}"
    check_daemon_binary || test_passed=false
    echo ""
    
    echo -e "${BLUE}[3/6] Cleaning environment and starting Daemon...${NC}"
    cleanup
    sleep 1
    start_daemon || test_passed=false
    echo ""
    
    echo -e "${BLUE}[4/6] Checking Agent processes...${NC}"
    check_agent_processes || test_passed=false
    echo ""
    
    echo -e "${BLUE}[5/6] Checking Daemon logs...${NC}"
    check_daemon_logs || test_passed=false
    echo ""
    
    echo -e "${BLUE}[6/6] Checking metadata files...${NC}"
    check_metadata_files || test_passed=false
    echo ""
    
    # 生成最终报告
    if $test_passed; then
        finalize_report "PASS"
        echo -e "\n${GREEN}✅ All tests PASSED!${NC}"
        echo -e "${GREEN}Daemon的Agent管理功能验证成功！${NC}"
        return 0
    else
        finalize_report "FAIL"
        echo -e "\n${RED}❌ Some tests FAILED!${NC}"
        echo -e "${YELLOW}请查看测试报告了解详情: $REPORT_FILE${NC}"
        return 1
    fi
}

# 运行主程序
main
exit_code=$?

# 显示报告路径
echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}测试报告: $REPORT_FILE${NC}"
echo -e "${BLUE}========================================${NC}"

# 保持Daemon运行一段时间以便检查
if [ $exit_code -eq 0 ]; then
    echo -e "\n${YELLOW}Daemon is still running. Press Ctrl+C to stop, or wait 5 seconds...${NC}"
    sleep 5
fi

exit $exit_code
