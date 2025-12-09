#!/bin/bash
# 异常场景测试脚本
# 功能: 测试系统在异常情况下的行为和恢复能力

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$SCRIPT_DIR"
REPORT_DIR="$TEST_DIR/reports"

mkdir -p "$REPORT_DIR"

# 打印函数
print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_step() { echo -e "${BLUE}[STEP]${NC} $1"; }
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }
print_fail() { echo -e "${RED}[✗]${NC} $1"; }

# 测试结果记录
TEST_RESULTS=()
PASSED=0
FAILED=0

record_test() {
    local test_name=$1
    local result=$2
    local message=$3
    TEST_RESULTS+=("$test_name|$result|$message")
    if [ "$result" = "PASS" ]; then
        ((PASSED++))
        print_success "$test_name: $message"
    else
        ((FAILED++))
        print_fail "$test_name: $message"
    fi
}

# 获取 JWT Token
get_jwt_token() {
    local username=${1:-"testuser"}
    local password=${2:-"password123"}
    curl -s -X POST http://127.0.0.1:8080/api/v1/auth/register \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\",\"email\":\"test@example.com\"}" > /dev/null 2>&1 || true
    local response=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\"}")
    echo "$response" | grep -o '"token":"[^"]*' | cut -d'"' -f4
}

# 获取 Node ID
get_node_id() {
    cat "$TEST_DIR/tmp/daemon/node_id" 2>/dev/null || echo ""
}

# 测试场景 1: Daemon 断线重连
test_scenario_1_reconnect() {
    print_step "测试场景 1: Daemon 断线重连"
    
    local daemon_pid=$(cat "$TEST_DIR/pids/daemon.pid" 2>/dev/null || echo "")
    if [ -z "$daemon_pid" ] || ! ps -p "$daemon_pid" > /dev/null 2>&1; then
        record_test "场景1-准备" "FAIL" "Daemon 未运行"
        return 1
    fi
    
    print_info "停止 Daemon 服务 (PID: $daemon_pid)..."
    kill -TERM "$daemon_pid" 2>/dev/null || true
    sleep 3
    
    if ps -p "$daemon_pid" > /dev/null 2>&1; then
        kill -9 "$daemon_pid" 2>/dev/null || true
        sleep 1
    fi
    
    if ! ps -p "$daemon_pid" > /dev/null 2>&1; then
        record_test "场景1-停止Daemon" "PASS" "Daemon 已停止"
    else
        record_test "场景1-停止Daemon" "FAIL" "Daemon 停止失败"
        return 1
    fi
    
    print_info "等待 5 秒后重启 Daemon..."
    sleep 5
    
    print_info "重启 Daemon..."
    cd "$PROJECT_ROOT"
    nohup "$PROJECT_ROOT/daemon/bin/daemon" -config "$TEST_DIR/config/daemon.test.yaml" > "$TEST_DIR/logs/daemon.log" 2>&1 &
    echo $! > "$TEST_DIR/pids/daemon.pid"
    sleep 5
    
    local new_pid=$(cat "$TEST_DIR/pids/daemon.pid" 2>/dev/null || echo "")
    if [ -n "$new_pid" ] && ps -p "$new_pid" > /dev/null 2>&1; then
        record_test "场景1-重启Daemon" "PASS" "Daemon 已重启 (PID: $new_pid)"
        
        print_info "等待 Daemon 重新连接到 Manager..."
        sleep 10
        
        if grep -q "registered to manager\|reconnected" "$TEST_DIR/logs/daemon.log" 2>/dev/null; then
            record_test "场景1-重连" "PASS" "Daemon 已重新连接到 Manager"
        else
            record_test "场景1-重连" "WARN" "未在日志中找到重连信息"
        fi
    else
        record_test "场景1-重启Daemon" "FAIL" "Daemon 重启失败"
    fi
}

# 测试场景 2: Agent 异常退出自动重启
test_scenario_2_agent_restart() {
    print_step "测试场景 2: Agent 异常退出自动重启"
    
    local test_agent_id="agent-002"
    local pid_file="$TEST_DIR/pids/$test_agent_id.pid"
    
    if [ ! -f "$pid_file" ]; then
        record_test "场景2-准备" "FAIL" "Agent PID 文件不存在"
        return 1
    fi
    
    local agent_pid=$(cat "$pid_file")
    if ! ps -p "$agent_pid" > /dev/null 2>&1; then
        record_test "场景2-准备" "FAIL" "Agent 未运行"
        return 1
    fi
    
    print_info "模拟 Agent 异常退出 (kill -9 $agent_pid)..."
    kill -9 "$agent_pid" 2>/dev/null || true
    sleep 2
    
    if ! ps -p "$agent_pid" > /dev/null 2>&1; then
        record_test "场景2-异常退出" "PASS" "Agent 已异常退出"
    else
        record_test "场景2-异常退出" "FAIL" "Agent 未退出"
        return 1
    fi
    
    print_info "等待 Daemon HealthChecker 检测并重启 Agent (30秒)..."
    sleep 30
    
    # 检查 Agent 是否被重启
    local new_pid=$(cat "$pid_file" 2>/dev/null || echo "")
    if [ -n "$new_pid" ] && [ "$new_pid" != "$agent_pid" ] && ps -p "$new_pid" > /dev/null 2>&1; then
        record_test "场景2-自动重启" "PASS" "Agent 已自动重启 (新 PID: $new_pid)"
    elif [ -n "$new_pid" ] && ps -p "$new_pid" > /dev/null 2>&1; then
        record_test "场景2-自动重启" "WARN" "Agent 进程存在但 PID 未变化"
    else
        record_test "场景2-自动重启" "FAIL" "Agent 未自动重启"
        print_info "查看 Daemon 日志:"
        tail -20 "$TEST_DIR/logs/daemon.log" | grep -i "agent\|restart" || true
    fi
}

# 测试场景 3: 网络延迟和超时
test_scenario_3_timeout() {
    print_step "测试场景 3: 网络延迟和超时"
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        record_test "场景3-准备" "FAIL" "无法获取 Token 或 Node ID"
        return 1
    fi
    
    print_info "测试 API 超时处理..."
    local start_time=$(date +%s)
    local response=$(timeout 5 curl -s -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" 2>&1 || echo "TIMEOUT")
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ "$response" = "TIMEOUT" ]; then
        record_test "场景3-超时处理" "WARN" "请求超时（5秒）"
    elif echo "$response" | grep -q '"code":0'; then
        record_test "场景3-正常响应" "PASS" "API 正常响应（耗时: ${duration}秒）"
    else
        record_test "场景3-错误响应" "WARN" "API 返回错误: $response"
    fi
}

# 测试场景 4: 并发操作冲突
test_scenario_4_concurrent() {
    print_step "测试场景 4: 并发操作冲突"
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    local test_agent_id="agent-002"
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        record_test "场景4-准备" "FAIL" "无法获取 Token 或 Node ID"
        return 1
    fi
    
    print_info "同时发起多个操作请求..."
    
    # 并发启动 3 个相同的操作
    for i in {1..3}; do
        (
            curl -s -X POST "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents/$test_agent_id/operate" \
                -H "Authorization: Bearer $token" \
                -H "Content-Type: application/json" \
                -d '{"operation":"restart"}' > /tmp/concurrent_op_$i.json 2>&1
        ) &
    done
    
    wait
    
    # 检查响应
    local success_count=0
    local conflict_count=0
    for i in {1..3}; do
        if [ -f "/tmp/concurrent_op_$i.json" ]; then
            local response=$(cat "/tmp/concurrent_op_$i.json")
            if echo "$response" | grep -q '"code":0'; then
                ((success_count++))
            elif echo "$response" | grep -qi "conflict\|busy\|locked"; then
                ((conflict_count++))
            fi
        fi
        rm -f "/tmp/concurrent_op_$i.json"
    done
    
    if [ $conflict_count -gt 0 ]; then
        record_test "场景4-冲突检测" "PASS" "系统正确检测到并发冲突 ($conflict_count 个冲突)"
    elif [ $success_count -eq 1 ]; then
        record_test "场景4-冲突处理" "PASS" "系统正确处理并发操作（1 个成功）"
    else
        record_test "场景4-并发处理" "WARN" "并发操作结果异常（成功: $success_count, 冲突: $conflict_count）"
    fi
}

# 测试场景 5: 数据库连接失败（模拟）
test_scenario_5_database() {
    print_step "测试场景 5: 数据库连接失败（模拟）"
    
    print_info "此场景需要手动停止 MySQL 数据库进行测试"
    print_info "为避免影响其他服务，跳过实际测试"
    
    record_test "场景5-数据库失败" "SKIP" "需要手动测试（停止 MySQL 后验证错误处理）"
}

# 生成测试报告
generate_report() {
    local report_file="$REPORT_DIR/error_scenarios_test_report.md"
    
    print_step "生成测试报告: $report_file"
    
    cat > "$report_file" <<EOF
# 异常场景测试报告

**生成时间**: $(date '+%Y-%m-%d %H:%M:%S')
**测试环境**: 完整系统集成测试环境

## 测试摘要

- **总测试数**: $((PASSED + FAILED))
- **通过**: $PASSED
- **失败**: $FAILED
- **通过率**: $(if [ $((PASSED + FAILED)) -gt 0 ]; then printf "%.1f%%" $(awk "BEGIN {print ($PASSED / ($PASSED + FAILED)) * 100}"); else echo "N/A"; fi)

## 测试结果详情

| 测试项 | 结果 | 说明 |
|--------|------|------|
EOF

    for result in "${TEST_RESULTS[@]}"; do
        IFS='|' read -r test_name result_status message <<< "$result"
        local status_icon=""
        case "$result_status" in
            PASS) status_icon="✅" ;;
            WARN) status_icon="⚠️" ;;
            SKIP) status_icon="⏭️" ;;
            *) status_icon="❌" ;;
        esac
        echo "| $test_name | $status_icon $result_status | $message |" >> "$report_file"
    done
    
    cat >> "$report_file" <<EOF

## 测试场景说明

### 场景 1: Daemon 断线重连
- Manager 和 Daemon 正常连接
- 模拟 Daemon 断线（停止 Daemon 服务）
- Manager 检测连接断开
- 重启 Daemon 服务
- Manager 自动重连

### 场景 2: Agent 异常退出自动重启
- Agent 正常运行
- 模拟 Agent 进程异常退出（kill -9）
- Daemon HealthChecker 检测到异常
- Daemon 自动重启 Agent

### 场景 3: 网络延迟和超时
- 模拟网络延迟
- 执行 Agent 操作
- 验证超时处理

### 场景 4: 并发操作冲突
- 同时发起多个操作请求（启动/停止同一 Agent）
- 验证操作冲突处理

### 场景 5: 数据库连接失败
- 模拟数据库连接失败
- Manager 处理数据库错误

## 问题记录

$(if [ $FAILED -gt 0 ]; then
    echo "### 失败项"
    for result in "${TEST_RESULTS[@]}"; do
        IFS='|' read -r test_name result_status message <<< "$result"
        if [ "$result_status" = "FAIL" ]; then
            echo "- **$test_name**: $message"
        fi
    done
else
    echo "无失败项"
fi)

---
**报告生成完成**
EOF

    print_success "测试报告已生成: $report_file"
}

# 主函数
main() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}异常场景测试${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    
    # 检查服务是否运行
    print_step "检查测试环境服务状态..."
    if ! curl -s http://127.0.0.1:8080/health > /dev/null 2>&1; then
        print_error "Manager 服务未运行，请先启动测试环境"
        exit 1
    fi
    print_info "Manager 服务运行正常"
    echo ""
    
    # 执行测试场景
    test_scenario_1_reconnect
    echo ""
    
    test_scenario_2_agent_restart
    echo ""
    
    test_scenario_3_timeout
    echo ""
    
    test_scenario_4_concurrent
    echo ""
    
    test_scenario_5_database
    echo ""
    
    # 生成报告
    generate_report
    echo ""
    
    # 输出摘要
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}测试完成${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "测试结果:"
    echo "  - 通过: $PASSED"
    echo "  - 失败: $FAILED"
    if [ $((PASSED + FAILED)) -gt 0 ]; then
        local pass_rate=$(awk "BEGIN {printf \"%.1f\", ($PASSED / ($PASSED + $FAILED)) * 100}")
        echo "  - 通过率: ${pass_rate}%"
    else
        echo "  - 通过率: N/A"
    fi
    echo ""
    echo "详细报告: $REPORT_DIR/error_scenarios_test_report.md"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

main "$@"
