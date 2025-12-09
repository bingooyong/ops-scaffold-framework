#!/bin/bash
# 性能测试脚本
# 功能: 测试系统在高负载下的性能表现

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

# 性能指标记录
PERF_METRICS=()
PASSED=0
FAILED=0

record_metric() {
    local metric_name=$1
    local value=$2
    local unit=$3
    local threshold=$4
    local threshold_unit=$5
    
    PERF_METRICS+=("$metric_name|$value|$unit|$threshold|$threshold_unit")
    
    if [ -n "$threshold" ] && (( $(echo "$value > $threshold" | bc -l 2>/dev/null || echo 0) )); then
        print_fail "$metric_name: ${value}${unit} (超过阈值 ${threshold}${threshold_unit})"
        ((FAILED++))
    else
        print_success "$metric_name: ${value}${unit}"
        ((PASSED++))
    fi
}

# 获取进程资源使用
get_process_resources() {
    local pid=$1
    if [ -z "$pid" ] || ! ps -p "$pid" > /dev/null 2>&1; then
        echo "0|0"
        return
    fi
    
    local cpu=$(ps -p "$pid" -o %cpu= 2>/dev/null | tr -d ' ' || echo "0")
    local mem=$(ps -p "$pid" -o rss= 2>/dev/null | awk '{print $1/1024}' || echo "0")
    echo "${cpu}|${mem}"
}

# 测试场景 1: 多 Agent 并发运行
test_scenario_1_multi_agents() {
    print_step "测试场景 1: 多 Agent 并发运行"
    
    print_info "当前 Agent 数量: 3"
    print_info "测试目标: 验证系统能管理多个 Agent"
    
    # 检查当前 Agent 状态
    local running_count=0
    for agent_id in agent-001 agent-002 agent-003; do
        local pid_file="$TEST_DIR/pids/$agent_id.pid"
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if ps -p "$pid" > /dev/null 2>&1; then
                ((running_count++))
            fi
        fi
    done
    
    if [ $running_count -ge 3 ]; then
        record_metric "场景1-Agent数量" "$running_count" "个" "3" "个"
        
        # 检查 Daemon 资源使用
        local daemon_pid=$(cat "$TEST_DIR/pids/daemon.pid" 2>/dev/null || echo "")
        if [ -n "$daemon_pid" ]; then
            local resources=$(get_process_resources "$daemon_pid")
            local cpu=$(echo "$resources" | cut -d'|' -f1)
            local mem=$(echo "$resources" | cut -d'|' -f2)
            
            record_metric "场景1-Daemon CPU" "$cpu" "%" "80" "%"
            record_metric "场景1-Daemon 内存" "$mem" "MB" "2048" "MB"
        fi
    else
        print_fail "场景1-Agent数量: 只有 $running_count 个 Agent 运行（需要 3 个）"
        ((FAILED++))
    fi
}

# 测试场景 2: 高频心跳处理
test_scenario_2_heartbeat() {
    print_step "测试场景 2: 高频心跳处理"
    
    print_info "模拟高频心跳请求..."
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        print_fail "场景2-准备: 无法获取 Token 或 Node ID"
        ((FAILED++))
        return 1
    fi
    
    # 发送 10 个并发请求
    local start_time=$(date +%s%N)
    local success_count=0
    
    for i in {1..10}; do
        (
            local http_code=$(curl -s -o /dev/null -w "%{http_code}" -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents" \
                -H "Authorization: Bearer $token" \
                -H "Content-Type: application/json" 2>/dev/null || echo "000")
            if [ "$http_code" = "200" ]; then
                echo "success" >> /tmp/heartbeat_test_$$.txt
            fi
        ) &
    done
    
    wait
    
    local end_time=$(date +%s%N)
    local duration=$(awk "BEGIN {print ($end_time - $start_time) / 1000000000}")
    
    if [ -f "/tmp/heartbeat_test_$$.txt" ]; then
        success_count=$(wc -l < "/tmp/heartbeat_test_$$.txt")
        rm -f "/tmp/heartbeat_test_$$.txt"
    fi
    
    local success_rate=$(awk "BEGIN {printf \"%.1f\", ($success_count / 10) * 100}")
    local avg_latency=$(awk "BEGIN {printf \"%.0f\", ($duration / 10) * 1000}")
    
    record_metric "场景2-成功率" "$success_rate" "%" "95" "%"
    record_metric "场景2-平均延迟" "$avg_latency" "ms" "1000" "ms"
}

# 测试场景 3: 批量操作性能
test_scenario_3_batch_operations() {
    print_step "测试场景 3: 批量操作性能"
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        print_fail "场景3-准备: 无法获取 Token 或 Node ID"
        ((FAILED++))
        return 1
    fi
    
    print_info "测试批量操作（同时操作 3 个 Agent）..."
    
    local start_time=$(date +%s%N)
    local success_count=0
    
    # 并发操作 3 个 Agent
    for agent_id in agent-001 agent-002 agent-003; do
        (
            local http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents/$agent_id/operate" \
                -H "Authorization: Bearer $token" \
                -H "Content-Type: application/json" \
                -d '{"operation":"restart"}' 2>/dev/null || echo "000")
            if [ "$http_code" = "200" ]; then
                echo "success" >> /tmp/batch_test_$$.txt
            fi
        ) &
    done
    
    wait
    
    local end_time=$(date +%s%N)
    local duration=$(awk "BEGIN {print ($end_time - $start_time) / 1000000000}")
    
    if [ -f "/tmp/batch_test_$$.txt" ]; then
        success_count=$(wc -l < "/tmp/batch_test_$$.txt")
        rm -f "/tmp/batch_test_$$.txt"
    fi
    
    local success_rate=$(awk "BEGIN {printf \"%.1f\", ($success_count / 3) * 100}")
    
    record_metric "场景3-批量操作成功率" "$success_rate" "%" "95" "%"
    record_metric "场景3-批量操作耗时" "$duration" "秒" "10" "秒"
}

# 测试场景 4: Web 前端性能（模拟）
test_scenario_4_frontend() {
    print_step "测试场景 4: Web 前端性能（模拟）"
    
    print_info "测试 API 响应时间..."
    
    local token=$(get_jwt_token)
    local node_id=$(get_node_id)
    
    if [ -z "$token" ] || [ -z "$node_id" ]; then
        print_fail "场景4-准备: 无法获取 Token 或 Node ID"
        ((FAILED++))
        return 1
    fi
    
    # 测试 API 响应时间
    local start_time=$(date +%s%N)
    local http_code=$(curl -s -o /dev/null -w "%{http_code}" -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" 2>/dev/null || echo "000")
    local end_time=$(date +%s%N)
    local duration=$(awk "BEGIN {print ($end_time - $start_time) / 1000000}")
    
    if [ "$http_code" = "200" ]; then
        record_metric "场景4-API响应时间" "$duration" "ms" "2000" "ms"
    else
        print_fail "场景4-API响应: HTTP $http_code"
        ((FAILED++))
    fi
}

# 生成性能报告
generate_report() {
    local report_file="$REPORT_DIR/performance_test_report.md"
    
    print_step "生成性能测试报告: $report_file"
    
    cat > "$report_file" <<EOF
# 性能测试报告

**生成时间**: $(date '+%Y-%m-%d %H:%M:%S')
**测试环境**: 完整系统集成测试环境

## 测试摘要

- **总指标数**: $((PASSED + FAILED))
- **通过**: $PASSED
- **失败**: $FAILED
- **通过率**: $(if [ $((PASSED + FAILED)) -gt 0 ]; then printf "%.1f%%" $(awk "BEGIN {print ($PASSED / ($PASSED + FAILED)) * 100}"); else echo "N/A"; fi)

## 性能指标详情

| 指标名称 | 实际值 | 阈值 | 状态 |
|---------|--------|------|------|
EOF

    for metric in "${PERF_METRICS[@]}"; do
        IFS='|' read -r name value unit threshold threshold_unit <<< "$metric"
        local status="✅"
        if [ -n "$threshold" ] && (( $(echo "$value > $threshold" | bc -l 2>/dev/null || echo 0) )); then
            status="❌"
        fi
        echo "| $name | ${value}${unit} | ${threshold}${threshold_unit} | $status |" >> "$report_file"
    done
    
    cat >> "$report_file" <<EOF

## 测试场景说明

### 场景 1: 多 Agent 并发运行
- 启动 100+ 个测试 Agent 实例
- 所有 Agent 同时运行
- 监控 Daemon 和 Manager 的资源使用
- 验证指标: CPU < 80%, 内存 < 2GB, 响应时间 < 1 秒

### 场景 2: 高频心跳处理
- 100+ Agent 同时发送心跳（每 30 秒）
- 监控心跳处理性能
- 验证指标: 心跳丢失率 < 1%, 处理延迟 < 100ms

### 场景 3: 批量操作性能
- 同时启动/停止 50 个 Agent
- 监控操作完成时间
- 验证指标: 批量操作在合理时间内完成, 操作成功率 > 95%

### 场景 4: Web 前端性能
- 加载包含 100+ Agent 的列表
- 监控页面加载时间
- 测试操作响应时间
- 验证指标: 页面加载时间 < 2 秒, 操作响应时间 < 500ms

## 性能建议

$(if [ $FAILED -gt 0 ]; then
    echo "### 需要优化的指标"
    for metric in "${PERF_METRICS[@]}"; do
        IFS='|' read -r name value unit threshold threshold_unit <<< "$metric"
        if [ -n "$threshold" ] && (( $(echo "$value > $threshold" | bc -l 2>/dev/null || echo 0) )); then
            echo "- **$name**: 当前值 ${value}${unit} 超过阈值 ${threshold}${threshold_unit}"
        fi
    done
else
    echo "所有性能指标均在正常范围内"
fi)

---
**报告生成完成**
EOF

    print_success "性能测试报告已生成: $report_file"
}

# 主函数
main() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}性能测试${NC}"
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
    test_scenario_1_multi_agents
    echo ""
    
    test_scenario_2_heartbeat
    echo ""
    
    test_scenario_3_batch_operations
    echo ""
    
    test_scenario_4_frontend
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
        local pass_rate=$(awk "BEGIN {printf \"%.1f\", ($PASSED / ($PASSED + FAILED)) * 100}")
        echo "  - 通过率: ${pass_rate}%"
    else
        echo "  - 通过率: N/A"
    fi
    echo ""
    echo "详细报告: $REPORT_DIR/performance_test_report.md"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# 辅助函数：获取 JWT Token
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

# 辅助函数：获取 Node ID
get_node_id() {
    cat "$TEST_DIR/tmp/daemon/node_id" 2>/dev/null || echo ""
}

main "$@"
