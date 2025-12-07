#!/bin/bash
# 集成测试脚本 - 测试 Agent 与 Daemon 的集成

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_test() { echo -e "${BLUE}[TEST]${NC} $1"; }

# 测试计数
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 测试函数
run_test() {
    local test_name=$1
    local test_cmd=$2
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    print_test "Running: $test_name"
    
    if eval "$test_cmd"; then
        print_info "✓ PASSED: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        print_error "✗ FAILED: $test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 测试 HTTP 端点
test_http_endpoint() {
    local port=$1
    local endpoint=$2
    local expected_status=${3:-200}
    
    response=$(curl -s -w "\n%{http_code}" "http://localhost:${port}${endpoint}" || echo "000")
    status_code=$(echo "$response" | tail -n1)
    
    if [ "$status_code" = "$expected_status" ]; then
        return 0
    else
        print_error "Expected status $expected_status, got $status_code"
        return 1
    fi
}

# 主测试流程
main() {
    print_info "========================================="
    print_info "  Agent Integration Test Suite"
    print_info "========================================="
    echo ""
    
    # 检查 Daemon 是否运行
    print_info "Checking prerequisites..."
    if [ ! -S "/tmp/daemon.sock" ]; then
        print_warn "Daemon socket not found at /tmp/daemon.sock"
        print_warn "Please start Daemon first!"
        print_info "This test will check Agent HTTP endpoints only"
        echo ""
    fi
    
    # 检查 Agent 是否运行
    if ! pgrep -f "bin/agent" > /dev/null; then
        print_error "No agent processes found"
        print_info "Please start agents using: ./scripts/agent.sh start"
        exit 1
    fi
    
    print_info "Found running agent processes:"
    pgrep -f "bin/agent" -l
    echo ""
    
    # 测试 agent-001 (端口 8081)
    print_info "Testing agent-001 (port 8081)..."
    run_test "agent-001: GET /health" "test_http_endpoint 8081 /health 200"
    run_test "agent-001: GET /metrics" "test_http_endpoint 8081 /metrics 200"
    run_test "agent-001: POST /reload" "curl -s -X POST http://localhost:8081/reload | grep -q 'success'"
    echo ""
    
    # 测试 agent-002 (端口 8082)
    print_info "Testing agent-002 (port 8082)..."
    run_test "agent-002: GET /health" "test_http_endpoint 8082 /health 200"
    run_test "agent-002: GET /metrics" "test_http_endpoint 8082 /metrics 200"
    run_test "agent-002: POST /reload" "curl -s -X POST http://localhost:8082/reload | grep -q 'success'"
    echo ""
    
    # 测试 agent-003 (端口 8083)
    print_info "Testing agent-003 (port 8083)..."
    run_test "agent-003: GET /health" "test_http_endpoint 8083 /health 200"
    run_test "agent-003: GET /metrics" "test_http_endpoint 8083 /metrics 200"
    run_test "agent-003: POST /reload" "curl -s -X POST http://localhost:8083/reload | grep -q 'success'"
    echo ""
    
    # 测试指标数据内容
    print_info "Testing metrics content..."
    run_test "agent-001: metrics contains agent_id" "curl -s http://localhost:8081/metrics | grep -q 'agent-001'"
    run_test "agent-002: metrics contains agent_id" "curl -s http://localhost:8082/metrics | grep -q 'agent-002'"
    run_test "agent-003: metrics contains agent_id" "curl -s http://localhost:8083/metrics | grep -q 'agent-003'"
    run_test "agent-001: metrics contains cpu_percent" "curl -s http://localhost:8081/metrics | grep -q 'cpu_percent'"
    run_test "agent-001: metrics contains memory_bytes" "curl -s http://localhost:8081/metrics | grep -q 'memory_bytes'"
    echo ""
    
    # 显示测试结果
    print_info "========================================="
    print_info "  Test Results Summary"
    print_info "========================================="
    echo -e "${BLUE}Total Tests:${NC}  $TOTAL_TESTS"
    echo -e "${GREEN}Passed:${NC}       $PASSED_TESTS"
    echo -e "${RED}Failed:${NC}       $FAILED_TESTS"
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        print_info "✓ All tests passed!"
        exit 0
    else
        print_error "✗ Some tests failed!"
        exit 1
    fi
}

main "$@"
