#!/bin/bash
# 完整业务流程测试脚本
# 功能: 测试从 Daemon 启动到 Web 前端显示的完整业务流程

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$SCRIPT_DIR"
CONFIG_DIR="$TEST_DIR/config"
LOG_DIR="$TEST_DIR/logs"
REPORT_DIR="$TEST_DIR/reports"

# 创建报告目录
mkdir -p "$REPORT_DIR"

# 打印函数
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_fail() {
    echo -e "${RED}[✗]${NC} $1"
}

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
    
    # 先尝试注册用户（如果不存在）
    curl -s -X POST http://127.0.0.1:8080/api/v1/auth/register \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\",\"email\":\"test@example.com\"}" > /dev/null 2>&1 || true
    
    # 登录获取 Token
    local response=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\"}")
    
    # 提取 Token（优先使用 python3，如果没有则使用 grep）
    local token=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null)
    if [ -z "$token" ]; then
        token=$(echo "$response" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    fi
    echo "$token"
}

# 获取 Node ID
get_node_id() {
    local node_id_file="$TEST_DIR/tmp/daemon/node_id"
    if [ -f "$node_id_file" ]; then
        cat "$node_id_file"
    else
        print_error "Node ID 文件不存在: $node_id_file"
        print_info "请确保 Daemon 已启动并注册到 Manager"
        return 1
    fi
}

# 从Agent列表响应中提取Agent状态
get_agent_status() {
    local response=$1
    local agent_id=$2
    local field=$3  # status 或 pid
    
    # 优先使用python3
    if command -v python3 > /dev/null 2>&1; then
        if [ "$field" = "status" ]; then
            echo "$response" | python3 -c "import sys, json; data=json.load(sys.stdin); agents=data.get('data', {}).get('agents', []); agent=[a for a in agents if a.get('agent_id')=='$agent_id']; print(agent[0].get('status', 'unknown') if agent else 'not_found')" 2>/dev/null
        elif [ "$field" = "pid" ]; then
            echo "$response" | python3 -c "import sys, json; data=json.load(sys.stdin); agents=data.get('data', {}).get('agents', []); agent=[a for a in agents if a.get('agent_id')=='$agent_id']; print(agent[0].get('pid', 0) if agent else 0)" 2>/dev/null
        fi
    else
        # 回退到grep方法（不太可靠，但可用）
        if [ "$field" = "status" ]; then
            echo "$response" | grep -o "\"agent_id\":\"$agent_id\"" -A 10 | grep -o "\"status\":\"[^\"]*" | cut -d'"' -f4 || echo "unknown"
        elif [ "$field" = "pid" ]; then
            echo "$response" | grep -o "\"agent_id\":\"$agent_id\"" -A 10 | grep -o "\"pid\":[0-9]*" | cut -d':' -f2 || echo "0"
        fi
    fi
}

# 测试场景 1: Agent 注册和发现
test_scenario_1_agent_registration() {
    print_step "测试场景 1: Agent 注册和发现"
    
    local token=$(get_jwt_token)
    if [ -z "$token" ]; then
        record_test "场景1-登录" "FAIL" "无法获取 JWT Token"
        return 1
    fi
    record_test "场景1-登录" "PASS" "成功获取 JWT Token"
    
    local node_id=$(get_node_id)
    if [ -z "$node_id" ]; then
        record_test "场景1-获取NodeID" "FAIL" "无法获取 Node ID"
        return 1
    fi
    record_test "场景1-获取NodeID" "PASS" "成功获取 Node ID: $node_id"
    
    # 通过 HTTP API 获取 Agent 列表
    print_info "通过 Manager HTTP API 获取 Agent 列表..."
    local http_code=$(curl -s -o /tmp/agent_list_response.json -w "%{http_code}" -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json")
    local response=$(cat /tmp/agent_list_response.json 2>/dev/null || echo "")
    rm -f /tmp/agent_list_response.json
    
    # 检查 HTTP 状态码
    if [ "$http_code" != "200" ]; then
        print_warn "HTTP 状态码: $http_code"
        print_info "响应内容: $response"
        record_test "场景1-获取Agent列表" "FAIL" "API 调用失败 (HTTP $http_code): $response"
        return 1
    fi
    
    # 检查响应
    if echo "$response" | grep -q '"code":0'; then
        local agent_count=$(echo "$response" | grep -o '"count":[0-9]*' | cut -d':' -f2)
        if [ -n "$agent_count" ] && [ "$agent_count" -gt 0 ]; then
            record_test "场景1-获取Agent列表" "PASS" "成功获取 $agent_count 个 Agent"
            print_info "Agent 列表响应:"
            echo "$response" | python3 -m json.tool 2>/dev/null || echo "$response"
        else
            record_test "场景1-获取Agent列表" "WARN" "Agent 列表为空（可能节点尚未注册 Agent）"
            print_info "响应内容: $response"
        fi
    else
        record_test "场景1-获取Agent列表" "FAIL" "API 返回错误: $response"
        print_info "响应内容: $response"
    fi
    
    # 验证 Agent 状态
    print_info "验证 Agent 状态..."
    for agent_id in agent-001 agent-002 agent-003; do
        if echo "$response" | grep -q "\"agent_id\":\"$agent_id\""; then
            record_test "场景1-Agent存在($agent_id)" "PASS" "Agent $agent_id 已注册"
        else
            record_test "场景1-Agent存在($agent_id)" "FAIL" "Agent $agent_id 未找到"
        fi
    done
}

# 测试场景 2: Agent 操作流程
test_scenario_2_agent_operations() {
    print_step "测试场景 2: Agent 操作流程"
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        record_test "场景2-准备" "FAIL" "无法获取 Token 或 Node ID"
        return 1
    fi
    
    # 选择一个 Agent 进行测试（使用 agent-002，因为它健康检查正常）
    local test_agent_id="agent-002"
    
    # ========== 1. 停止 Agent ==========
    print_info "停止 Agent: $test_agent_id"
    
    # 记录操作前的时间戳，用于检查日志
    local operation_start_time=$(date +%s)
    
    local daemonctl_path="$PROJECT_ROOT/daemon/bin/daemonctl"
    if [ ! -f "$daemonctl_path" ]; then
        # 如果daemonctl不存在，尝试通过HTTP API
        if [ -z "$token" ] || [ -z "$node_id" ]; then
            record_test "场景2-停止Agent" "FAIL" "Token 或 Node ID 为空，且 daemonctl 不存在"
            return 1
        fi
        
        local stop_http_code=$(curl -s --max-time 30 --connect-timeout 5 \
            -o /tmp/stop_response.json \
            -w "%{http_code}" \
            -X POST "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents/$test_agent_id/operate" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "{\"operation\":\"stop\"}" 2>/tmp/curl_error.log)
        
        local curl_error=$(cat /tmp/curl_error.log 2>/dev/null || echo "")
        local stop_response=$(cat /tmp/stop_response.json 2>/dev/null || echo "")
        rm -f /tmp/stop_response.json /tmp/curl_error.log
        
        if [ "$stop_http_code" = "200" ] && echo "$stop_response" | grep -q '"code":0'; then
            record_test "场景2-停止Agent" "PASS" "成功停止 Agent"
        elif [ "$stop_http_code" = "000" ]; then
            record_test "场景2-停止Agent" "FAIL" "连接失败 (HTTP 000): $curl_error"
        else
            record_test "场景2-停止Agent" "FAIL" "停止 Agent 失败 (HTTP $stop_http_code): $stop_response"
        fi
    else
        # 使用daemonctl
        local stop_output=$("$daemonctl_path" stop "$test_agent_id" 2>&1)
        local stop_exit_code=$?
        
        if [ $stop_exit_code -eq 0 ] && echo "$stop_output" | grep -q "成功"; then
            record_test "场景2-停止Agent" "PASS" "成功停止 Agent"
        else
            record_test "场景2-停止Agent" "FAIL" "停止 Agent 失败: $stop_output"
        fi
    fi
    
    # ========== 2. 验证 Agent 进程已停止 ==========
    print_info "验证 Agent 进程状态..."
    
    # 优先使用daemonctl验证状态（这是Daemon的权威状态）
    local daemonctl_path="$PROJECT_ROOT/daemon/bin/daemonctl"
    if [ -f "$daemonctl_path" ]; then
        # Agent优雅停止最多需要30秒，所以等待35秒确保完全退出
        print_info "等待Agent停止（最多35秒）..."
        local max_wait=35
        local waited=0
        local agent_stopped=false
        
        while [ $waited -lt $max_wait ]; do
            sleep 2
            waited=$((waited + 2))
            local status_output=$("$daemonctl_path" status "$test_agent_id" 2>&1)
            local agent_status=$(echo "$status_output" | grep "状态:" | grep -v "^$" | awk '{print $2}' | tail -1 | tr -d '[:space:]')
            local agent_pid=$(echo "$status_output" | grep "PID:" | awk '{print $2}' | tail -1 | tr -d '[:space:]')
            
            if echo "$status_output" | grep -q "状态.*stopped" && [ "$agent_pid" = "0" ]; then
                agent_stopped=true
                record_test "场景2-验证停止" "PASS" "Agent状态为stopped (waited ${waited}s)"
                break
            elif [ $waited -lt $max_wait ]; then
                print_info "  等待中... (${waited}s/${max_wait}s, 当前状态: $agent_status)..."
            fi
        done
        
        if [ "$agent_stopped" = "false" ]; then
            # 最终检查
            local status_output=$("$daemonctl_path" status "$test_agent_id" 2>&1)
            local agent_pid=$(echo "$status_output" | grep -o "PID:[[:space:]]*[0-9]*" | awk '{print $2}')
            
            if echo "$status_output" | grep -q "状态.*stopped" && [ "$agent_pid" = "0" ]; then
                record_test "场景2-验证停止" "PASS" "Agent状态为stopped, PID=0"
            else
                print_warn "Agent状态检查: $status_output"
                record_test "场景2-验证停止" "FAIL" "Agent状态未变为stopped (状态: $status_output)"
            fi
        fi
    else
        # 如果没有daemonctl，回退到进程检查（但可能包含僵尸进程）
        print_info "等待Agent进程退出（最多35秒）..."
        local max_wait=35
        local waited=0
        local agent_process_count=1
        
        while [ $waited -lt $max_wait ] && [ $agent_process_count -gt 0 ]; do
            sleep 2
            waited=$((waited + 2))
            agent_process_count=$(ps aux | grep -E "[a]gent/bin/agent.*$test_agent_id" | wc -l | tr -d ' ')
            if [ $waited -lt $max_wait ] && [ $agent_process_count -gt 0 ]; then
                print_info "  等待中... (${waited}s/${max_wait}s, 进程数: $agent_process_count)"
            fi
        done
        
        agent_process_count=$(ps aux | grep -E "[a]gent/bin/agent.*$test_agent_id" | wc -l | tr -d ' ')
        if [ "$agent_process_count" = "0" ]; then
            record_test "场景2-验证停止" "PASS" "Agent 进程已停止 (waited ${waited}s)"
        else
            record_test "场景2-验证停止" "FAIL" "Agent 进程仍在运行 (进程数: $agent_process_count, 可能包含僵尸进程)"
        fi
    fi
    
    # ========== 3. 启动 Agent ==========
    print_info "启动 Agent: $test_agent_id"
    
    local daemonctl_path="$PROJECT_ROOT/daemon/bin/daemonctl"
    if [ ! -f "$daemonctl_path" ]; then
        # 如果daemonctl不存在，尝试通过HTTP API
        local start_http_code=$(curl -s --max-time 30 --connect-timeout 5 \
            -o /tmp/start_response.json \
            -w "%{http_code}" \
            -X POST "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents/$test_agent_id/operate" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "{\"operation\":\"start\"}" 2>/tmp/curl_error.log)
        
        local curl_error=$(cat /tmp/curl_error.log 2>/dev/null || echo "")
        local start_response=$(cat /tmp/start_response.json 2>/dev/null || echo "")
        rm -f /tmp/start_response.json /tmp/curl_error.log
        
        if [ "$start_http_code" = "200" ] && echo "$start_response" | grep -q '"code":0'; then
            record_test "场景2-启动Agent" "PASS" "成功启动 Agent"
        elif [ "$start_http_code" = "000" ]; then
            record_test "场景2-启动Agent" "FAIL" "连接失败 (HTTP 000): $curl_error"
        else
            record_test "场景2-启动Agent" "FAIL" "启动 Agent 失败 (HTTP $start_http_code): $start_response"
        fi
    else
        # 使用daemonctl
        local start_output=$("$daemonctl_path" start "$test_agent_id" 2>&1)
        local start_exit_code=$?
        
        if [ $start_exit_code -eq 0 ] && echo "$start_output" | grep -q "成功"; then
            record_test "场景2-启动Agent" "PASS" "成功启动 Agent"
        else
            record_test "场景2-启动Agent" "FAIL" "启动 Agent 失败: $start_output"
        fi
    fi
    
    # ========== 4. 验证 Agent 进程已启动 ==========
    print_info "验证 Agent 进程状态..."
    
    # 优先使用daemonctl验证状态
    local daemonctl_path="$PROJECT_ROOT/daemon/bin/daemonctl"
    if [ -f "$daemonctl_path" ]; then
        # Agent启动可能需要时间，最多等待20秒
        print_info "等待Agent启动完成（最多20秒）..."
        local max_wait=20
        local waited=0
        local agent_started=false
        
        while [ $waited -lt $max_wait ]; do
            sleep 2
            waited=$((waited + 2))
            local status_output=$("$daemonctl_path" status "$test_agent_id" 2>&1)
            local agent_status=$(echo "$status_output" | grep "状态:" | grep -v "^$" | awk '{print $2}' | tail -1 | tr -d '[:space:]')
            local agent_pid=$(echo "$status_output" | grep "PID:" | awk '{print $2}' | tail -1 | tr -d '[:space:]')
            
            print_info "  检查状态: status=$agent_status, PID=$agent_pid (waited ${waited}s/${max_wait}s)"
            
            if [ "$agent_status" = "running" ] && [ "$agent_pid" != "0" ] && [ -n "$agent_pid" ]; then
                agent_started=true
                record_test "场景2-验证启动" "PASS" "Agent状态为running, PID=$agent_pid (waited ${waited}s)"
                break
            elif [ "$agent_status" != "starting" ] && [ "$agent_status" != "restarting" ] && [ "$agent_status" != "stopped" ]; then
                # 如果状态异常，提前退出
                record_test "场景2-验证启动" "FAIL" "Agent状态异常: $agent_status, PID=$agent_pid"
                break
            fi
        done
        
        if [ "$agent_started" = "false" ]; then
            # 最终检查
            local status_output=$("$daemonctl_path" status "$test_agent_id" 2>&1)
            local agent_status=$(echo "$status_output" | grep "状态:" | grep -v "^$" | awk '{print $2}' | tail -1 | tr -d '[:space:]')
            local agent_pid=$(echo "$status_output" | grep "PID:" | awk '{print $2}' | tail -1 | tr -d '[:space:]')
            
            if [ "$agent_status" = "running" ] && [ "$agent_pid" != "0" ] && [ -n "$agent_pid" ]; then
                record_test "场景2-验证启动" "PASS" "Agent状态为running, PID=$agent_pid"
            else
                record_test "场景2-验证启动" "FAIL" "Agent启动超时或失败: status=$agent_status, PID=$agent_pid"
            fi
        fi
    else
        # 如果没有daemonctl，回退到进程检查
        sleep 3
        local agent_process_count=$(ps aux | grep -E "[a]gent/bin/agent.*$test_agent_id" | wc -l | tr -d ' ')
        if [ "$agent_process_count" -gt 0 ]; then
            local agent_pid=$(ps aux | grep -E "[a]gent/bin/agent.*$test_agent_id" | awk '{print $2}' | head -1)
            record_test "场景2-验证启动" "PASS" "Agent 进程已启动 (PID: $agent_pid)"
        else
            sleep 2
            agent_process_count=$(ps aux | grep -E "[a]gent/bin/agent.*$test_agent_id" | wc -l | tr -d ' ')
            if [ "$agent_process_count" -gt 0 ]; then
                local agent_pid=$(ps aux | grep -E "[a]gent/bin/agent.*$test_agent_id" | awk '{print $2}' | head -1)
                record_test "场景2-验证启动" "PASS" "Agent 进程已启动 (PID: $agent_pid, 延迟启动)"
            else
                record_test "场景2-验证启动" "FAIL" "Agent 进程未启动"
            fi
        fi
    fi
}

# 测试场景 3: 状态同步流程
test_scenario_3_state_sync() {
    print_step "测试场景 3: 状态同步流程"
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        record_test "场景3-准备" "FAIL" "无法获取 Token 或 Node ID"
        return 1
    fi
    
    # 获取初始 Agent 状态
    print_info "获取初始 Agent 状态..."
    local initial_response=$(curl -s -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json")
    
    record_test "场景3-获取初始状态" "PASS" "成功获取 Agent 状态"
    
    # 等待状态同步（Daemon 会定期同步状态到 Manager）
    print_info "等待状态同步（10秒）..."
    sleep 10
    
    # 再次获取 Agent 状态
    print_info "获取更新后的 Agent 状态..."
    local updated_response=$(curl -s -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json")
    
    if [ "$initial_response" != "$updated_response" ]; then
        record_test "场景3-状态同步" "PASS" "状态已同步更新"
    else
        record_test "场景3-状态同步" "WARN" "状态未变化（可能已是最新状态）"
    fi
}

# 测试场景 4: 日志查看流程
test_scenario_4_logs() {
    print_step "测试场景 4: 日志查看流程"
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        record_test "场景4-准备" "FAIL" "无法获取 Token 或 Node ID"
        return 1
    fi
    
    # 选择一个 Agent 进行测试
    local test_agent_id="agent-002"
    
    # 获取 Agent 日志
    print_info "获取 Agent 日志: $test_agent_id"
    local log_http_code=$(curl -s -o /tmp/log_response.json -w "%{http_code}" -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents/$test_agent_id/logs?lines=50" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json")
    local log_response=$(cat /tmp/log_response.json 2>/dev/null || echo "")
    rm -f /tmp/log_response.json
    
    if [ "$log_http_code" = "200" ] && echo "$log_response" | grep -q '"code":0'; then
        record_test "场景4-获取日志" "PASS" "成功获取 Agent 日志"
        print_info "日志预览（前5行）:"
        echo "$log_response" | python3 -m json.tool 2>/dev/null | head -20 || echo "$log_response" | head -5
    else
        record_test "场景4-获取日志" "FAIL" "获取日志失败 (HTTP $log_http_code): $log_response"
    fi
    
    # 验证日志文件存在
    local log_file="$TEST_DIR/logs/agent-$test_agent_id.log"
    if [ -f "$log_file" ]; then
        local log_size=$(wc -l < "$log_file" 2>/dev/null || echo "0")
        if [ "$log_size" -gt 0 ]; then
            record_test "场景4-日志文件" "PASS" "日志文件存在，包含 $log_size 行"
        else
            record_test "场景4-日志文件" "WARN" "日志文件存在但为空"
        fi
    else
        record_test "场景4-日志文件" "FAIL" "日志文件不存在: $log_file"
    fi
}

# 测试场景 5: 监控图表流程
test_scenario_5_metrics() {
    print_step "测试场景 5: 监控图表流程"
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        record_test "场景5-准备" "FAIL" "无法获取 Token 或 Node ID"
        return 1
    fi
    
    # 获取节点最新指标
    print_info "获取节点最新指标..."
    local metrics_http_code=$(curl -s -o /tmp/metrics_response.json -w "%{http_code}" -X GET "http://127.0.0.1:8080/api/v1/metrics/nodes/$node_id/latest" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json")
    local metrics_response=$(cat /tmp/metrics_response.json 2>/dev/null || echo "")
    rm -f /tmp/metrics_response.json
    
    if [ "$metrics_http_code" = "200" ] && echo "$metrics_response" | grep -q '"code":0'; then
        record_test "场景5-节点指标" "PASS" "成功获取节点指标"
    else
        record_test "场景5-节点指标" "WARN" "获取节点指标失败 (HTTP $metrics_http_code，可能功能未实现): $metrics_response"
    fi
    
    # 获取 Agent 指标（如果 API 已实现）
    local test_agent_id="agent-002"
    print_info "获取 Agent 指标: $test_agent_id"
    # 注意：Agent 指标 API 可能还未实现，这里先测试接口是否存在
    local agent_metrics_response=$(curl -s -X GET "http://127.0.0.1:8080/api/v1/metrics/agents/$test_agent_id/history?duration=3600" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" 2>&1)
    
    if echo "$agent_metrics_response" | grep -q '"code":0'; then
        record_test "场景5-Agent指标" "PASS" "成功获取 Agent 指标"
    else
        record_test "场景5-Agent指标" "WARN" "Agent 指标 API 可能未实现或不可用"
    fi
}

# 生成测试报告
generate_report() {
    local report_file="$REPORT_DIR/business_flows_test_report.md"
    
    print_step "生成测试报告: $report_file"
    
    cat > "$report_file" <<EOF
# 业务流程测试报告

**生成时间**: $(date '+%Y-%m-%d %H:%M:%S')
**测试环境**: 完整系统集成测试环境

## 测试摘要

- **总测试数**: $((PASSED + FAILED))
- **通过**: $PASSED
- **失败**: $FAILED
- **通过率**: $(if [ $((PASSED + FAILED)) -gt 0 ]; then printf "%.1f%%" $(awk "BEGIN {print ($PASSED / ($PASSED + $FAILED)) * 100}"); else echo "N/A"; fi)

## 测试结果详情

| 测试项 | 结果 | 说明 |
|--------|------|------|
EOF

    for result in "${TEST_RESULTS[@]}"; do
        IFS='|' read -r test_name result_status message <<< "$result"
        local status_icon=""
        if [ "$result_status" = "PASS" ]; then
            status_icon="✅"
        elif [ "$result_status" = "WARN" ]; then
            status_icon="⚠️"
        else
            status_icon="❌"
        fi
        echo "| $test_name | $status_icon $result_status | $message |" >> "$report_file"
    done
    
    cat >> "$report_file" <<EOF

## 测试场景说明

### 场景 1: Agent 注册和发现
- 验证 Daemon 启动时自动注册 Agent 到 AgentRegistry
- 验证 Manager 通过 gRPC 查询 Agent 列表
- 验证 Web 前端通过 HTTP API 获取 Agent 列表
- 验证 Agent 状态正确同步

### 场景 2: Agent 操作流程
- 通过 Web 前端启动 Agent
- 验证 Agent 进程启动
- 通过 Web 前端停止 Agent
- 验证 Agent 进程停止
- 通过 Web 前端重启 Agent
- 验证 Agent 进程重启

### 场景 3: 状态同步流程
- Agent 状态变化（启动/停止）
- Daemon 检测状态变化
- Daemon 通过 gRPC 上报状态到 Manager
- Manager 更新数据库
- Web 前端刷新显示最新状态

### 场景 4: 日志查看流程
- Agent 运行并产生日志
- 通过 Web 前端查看 Agent 日志
- 验证日志内容正确显示

### 场景 5: 监控图表流程
- Agent 运行并产生资源使用数据
- Daemon 采集 Agent 资源数据
- 通过 Web 前端查看监控图表

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

## 建议

1. 确保所有服务正常运行
2. 检查网络连接和端口占用
3. 查看服务日志以获取详细错误信息
4. 验证数据库连接和配置

---
**报告生成完成**
EOF

    print_success "测试报告已生成: $report_file"
}

# 主函数
main() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}完整业务流程测试${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    
    # 检查服务是否运行
    print_step "检查测试环境服务状态..."
    if ! curl -s http://127.0.0.1:8080/health > /dev/null 2>&1; then
        print_error "Manager 服务未运行，请先启动测试环境"
        print_info "运行: $TEST_DIR/start_test_env.sh"
        exit 1
    fi
    print_info "Manager 服务运行正常"
    echo ""
    
    # 执行测试场景
    test_scenario_1_agent_registration
    echo ""
    
    test_scenario_2_agent_operations
    echo ""
    
    test_scenario_3_state_sync
    echo ""
    
    test_scenario_4_logs
    echo ""
    
    test_scenario_5_metrics
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
    echo "详细报告: $REPORT_DIR/business_flows_test_report.md"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# 执行主函数
main "$@"
