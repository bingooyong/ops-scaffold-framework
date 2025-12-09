#!/bin/bash
# 测试环境验证脚本
# 功能: 验证所有服务是否正常运行并记录环境信息

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$SCRIPT_DIR"
PID_DIR="$TEST_DIR/pids"
LOG_DIR="$TEST_DIR/logs"
REPORT_FILE="$TEST_DIR/test_env_verification_report.md"

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

# 检查进程
check_process() {
    local name=$1
    local pid_file="$PID_DIR/$name.pid"
    
    if [ ! -f "$pid_file" ]; then
        echo "❌ $name: PID 文件不存在"
        return 1
    fi
    
    local pid=$(cat "$pid_file")
    if ps -p "$pid" > /dev/null 2>&1; then
        echo "✅ $name: 进程运行中 (PID: $pid)"
        return 0
    else
        echo "❌ $name: 进程不存在 (PID: $pid)"
        return 1
    fi
}

# 检查 HTTP 端点
check_http_endpoint() {
    local url=$1
    local name=$2
    
    if curl -s "$url" > /dev/null 2>&1; then
        echo "✅ $name: HTTP 端点可访问 ($url)"
        return 0
    else
        echo "❌ $name: HTTP 端点不可访问 ($url)"
        return 1
    fi
}

# 检查端口
check_port_listening() {
    local port=$1
    local name=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "✅ $name: 端口 $port 正在监听"
        return 0
    else
        echo "❌ $name: 端口 $port 未监听"
        return 1
    fi
}

# 生成验证报告
generate_report() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    cat > "$REPORT_FILE" <<EOF
# 测试环境验证报告

**生成时间**: $timestamp

## 测试环境信息

### 服务端口配置
- Manager HTTP API: http://127.0.0.1:8080
- Manager gRPC: 127.0.0.1:9090
- Daemon gRPC: 127.0.0.1:9091 (如果启用)
- Agent-001 HTTP: http://127.0.0.1:8081
- Agent-002 HTTP: http://127.0.0.1:8082
- Agent-003 HTTP: http://127.0.0.1:8083

### 配置文件路径
- Manager 配置: \`test/integration/config/manager.test.yaml\`
- Daemon 配置: \`test/integration/config/daemon.test.yaml\`
- Agent-001 配置: \`test/integration/config/agent-001.test.yaml\`
- Agent-002 配置: \`test/integration/config/agent-002.test.yaml\`
- Agent-003 配置: \`test/integration/config/agent-003.test.yaml\`

### 日志文件路径
- Manager 日志: \`test/integration/logs/manager.log\`
- Daemon 日志: \`test/integration/logs/daemon.log\`
- Agent-001 日志: \`test/integration/logs/agent-001.log\`
- Agent-002 日志: \`test/integration/logs/agent-002.log\`
- Agent-003 日志: \`test/integration/logs/agent-003.log\`

### PID 文件路径
- PID 文件目录: \`test/integration/pids/\`

## 验证结果

EOF

    # 验证 Manager
    echo "### Manager 服务" >> "$REPORT_FILE"
    if check_process "manager"; then
        echo "- ✅ Manager 进程运行正常" >> "$REPORT_FILE"
    else
        echo "- ❌ Manager 进程异常" >> "$REPORT_FILE"
    fi
    
    if check_port_listening 8080 "Manager HTTP"; then
        echo "- ✅ Manager HTTP 端口 (8080) 正常" >> "$REPORT_FILE"
    else
        echo "- ❌ Manager HTTP 端口 (8080) 异常" >> "$REPORT_FILE"
    fi
    
    if check_http_endpoint "http://127.0.0.1:8080/health" "Manager Health"; then
        echo "- ✅ Manager 健康检查通过" >> "$REPORT_FILE"
    else
        echo "- ❌ Manager 健康检查失败" >> "$REPORT_FILE"
    fi
    echo "" >> "$REPORT_FILE"
    
    # 验证 Daemon
    echo "### Daemon 服务" >> "$REPORT_FILE"
    if check_process "daemon"; then
        echo "- ✅ Daemon 进程运行正常" >> "$REPORT_FILE"
    else
        echo "- ❌ Daemon 进程异常" >> "$REPORT_FILE"
    fi
    echo "" >> "$REPORT_FILE"
    
    # 验证 Agent
    echo "### Agent 服务" >> "$REPORT_FILE"
    for agent_id in agent-001 agent-002 agent-003; do
        if check_process "$agent_id"; then
            echo "- ✅ $agent_id 进程运行正常" >> "$REPORT_FILE"
        else
            echo "- ⚠️  $agent_id 进程未运行（可能由 Daemon 管理）" >> "$REPORT_FILE"
        fi
    done
    echo "" >> "$REPORT_FILE"
    
    # 验证服务间连接
    echo "### 服务间连接" >> "$REPORT_FILE"
    echo "- Manager ↔ Daemon: gRPC 连接（需要运行时验证）" >> "$REPORT_FILE"
    echo "- Daemon ↔ Agent: Unix Socket 或 HTTP（需要运行时验证）" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # 环境要求
    echo "## 环境要求" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "- MySQL 8.0+: 运行在 127.0.0.1:3306" >> "$REPORT_FILE"
    echo "- Go 1.24.0+: 用于构建服务" >> "$REPORT_FILE"
    echo "- 端口可用: 8080, 8081, 8082, 8083, 9090" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    print_info "验证报告已生成: $REPORT_FILE"
}

# 主函数
main() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}测试环境验证${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    
    print_step "验证服务进程..."
    check_process "manager"
    check_process "daemon"
    for agent_id in agent-001 agent-002 agent-003; do
        check_process "$agent_id"
    done
    echo ""
    
    print_step "验证服务端口..."
    check_port_listening 8080 "Manager HTTP"
    check_port_listening 8081 "Agent-001 HTTP"
    check_port_listening 8082 "Agent-002 HTTP"
    check_port_listening 8083 "Agent-003 HTTP"
    echo ""
    
    print_step "验证 HTTP 端点..."
    check_http_endpoint "http://127.0.0.1:8080/health" "Manager Health"
    check_http_endpoint "http://127.0.0.1:8081/health" "Agent-001 Health" || true
    check_http_endpoint "http://127.0.0.1:8082/health" "Agent-002 Health" || true
    check_http_endpoint "http://127.0.0.1:8083/health" "Agent-003 Health" || true
    echo ""
    
    print_step "生成验证报告..."
    generate_report
    echo ""
    
    print_info "验证完成！查看报告: $REPORT_FILE"
}

# 执行主函数
main "$@"
