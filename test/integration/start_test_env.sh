#!/bin/bash
# 完整系统集成测试环境启动脚本
# 功能: 启动 MySQL、Manager、Daemon 和多个测试 Agent 实例

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
PID_DIR="$TEST_DIR/pids"

# 创建必要的目录
mkdir -p "$LOG_DIR" "$PID_DIR" "$CONFIG_DIR"

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

# 检查命令是否存在
check_command() {
    if ! command -v "$1" &> /dev/null; then
        print_error "$1 未安装，请先安装"
        return 1
    fi
    return 0
}

# 检查端口是否被占用
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0  # 端口被占用
    else
        return 1  # 端口空闲
    fi
}

# 等待服务就绪
wait_for_service() {
    local url=$1
    local max_attempts=${2:-30}
    local attempt=0
    
    print_info "等待服务就绪: $url"
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            print_info "服务已就绪: $url"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    print_error "服务启动超时: $url"
    return 1
}

# 检查 MySQL
check_mysql() {
    print_step "检查 MySQL 数据库..."
    if mysql -h 127.0.0.1 -P 3306 -uroot -prootpassword -e "SELECT 1;" > /dev/null 2>&1; then
        print_info "MySQL 数据库运行正常"
        return 0
    else
        print_warn "MySQL 数据库未运行或连接失败"
        print_info "请确保 MySQL 已启动并配置正确"
        print_info "默认配置: host=127.0.0.1, port=3306, user=root, password=rootpassword"
        return 1
    fi
}

# 停止占用端口的进程
stop_process_by_port() {
    local port=$1
    local name=$2
    
    local pid=$(lsof -ti:$port 2>/dev/null || echo "")
    if [ -n "$pid" ] && ps -p "$pid" > /dev/null 2>&1; then
        print_warn "端口 $port 被进程 $pid 占用，正在停止..."
        kill -TERM "$pid" 2>/dev/null || true
        sleep 2
        if ps -p "$pid" > /dev/null 2>&1; then
            kill -9 "$pid" 2>/dev/null || true
            sleep 1
        fi
        if ! ps -p "$pid" > /dev/null 2>&1; then
            print_info "$name 进程已停止"
        else
            print_error "$name 进程停止失败"
            return 1
        fi
    fi
    return 0
}


# 启动 Manager 服务
start_manager() {
    print_step "启动 Manager 服务..."
    
    if check_port 8080; then
        print_warn "Manager HTTP 端口 8080 已被占用"
        if ! stop_process_by_port 8080 "Manager"; then
            print_error "无法停止占用端口的进程，请手动处理"
            return 1
        fi
        sleep 1
    fi
    
    cd "$PROJECT_ROOT/manager"
    
    # 删除旧的二进制文件，确保每次都是全新构建
    if [ -f "bin/manager" ]; then
        print_info "删除旧的 Manager 二进制文件..."
        rm -f "bin/manager"
    fi
    
    # 构建 Manager
    print_info "构建 Manager..."
    make build
    
    # 启动 Manager（在项目根目录下运行，以便配置文件路径正确）
    print_info "启动 Manager (HTTP: 8080, gRPC: 9090)..."
    cd "$PROJECT_ROOT"
    nohup "$PROJECT_ROOT/manager/bin/manager" -config "$CONFIG_DIR/manager.test.yaml" > "$LOG_DIR/manager.log" 2>&1 &
    echo $! > "$PID_DIR/manager.pid"
    
    # 等待 Manager 就绪
    sleep 3
    if wait_for_service "http://127.0.0.1:8080/health" 30; then
        print_info "Manager 服务启动成功 (PID: $(cat $PID_DIR/manager.pid))"
        return 0
    else
        print_error "Manager 服务启动失败，查看日志: $LOG_DIR/manager.log"
        return 1
    fi
}

# 启动 Daemon 服务
start_daemon() {
    print_step "启动 Daemon 服务..."
    
    if check_port 9091; then
        print_warn "Daemon gRPC 端口 9091 已被占用"
        if ! stop_process_by_port 9091 "Daemon"; then
            print_error "无法停止占用端口的进程，请手动处理"
            return 1
        fi
        sleep 1
    fi
    
    cd "$PROJECT_ROOT/daemon"
    
    # 删除旧的二进制文件，确保每次都是全新构建
    if [ -f "bin/daemon" ]; then
        print_info "删除旧的 Daemon 二进制文件..."
        rm -f "bin/daemon"
    fi
    
    # 构建 Daemon
    print_info "构建 Daemon..."
    make build
    
    # 启动 Daemon（在项目根目录下运行，以便配置文件路径正确）
    print_info "启动 Daemon (gRPC: 9091)..."
    cd "$PROJECT_ROOT"
    nohup "$PROJECT_ROOT/daemon/bin/daemon" -config "$CONFIG_DIR/daemon.test.yaml" > "$LOG_DIR/daemon.log" 2>&1 &
    echo $! > "$PID_DIR/daemon.pid"
    
    # 等待 Daemon 就绪（检查进程）
    sleep 3
    if ps -p $(cat "$PID_DIR/daemon.pid") > /dev/null 2>&1; then
        print_info "Daemon 服务启动成功 (PID: $(cat $PID_DIR/daemon.pid))"
        return 0
    else
        print_error "Daemon 服务启动失败，查看日志: $LOG_DIR/daemon.log"
        return 1
    fi
}

# 启动测试 Agent
start_agent() {
    local agent_id=$1
    local config_file=$2
    local port=$3
    
    print_step "启动测试 Agent: $agent_id"
    
    if check_port $port; then
        print_warn "Agent $agent_id HTTP 端口 $port 已被占用"
        local pid=$(lsof -ti:$port 2>/dev/null || echo "")
        if [ -n "$pid" ] && ps -p "$pid" > /dev/null 2>&1; then
            print_info "停止占用端口的进程 (PID: $pid)..."
            kill -TERM "$pid" 2>/dev/null || true
            sleep 2
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
                sleep 1
            fi
        fi
        sleep 1
    fi
    
    cd "$PROJECT_ROOT/agent"
    
    # 删除旧的二进制文件，确保每次都是全新构建
    if [ -f "bin/agent" ]; then
        print_info "删除旧的 Agent 二进制文件..."
        rm -f "bin/agent"
    fi
    
    # 构建 Agent
    print_info "构建 Agent..."
    if ! make build; then
        print_error "Agent 构建失败"
        return 1
    fi
    
    # 验证二进制文件存在
    if [ ! -f "$PROJECT_ROOT/agent/bin/agent" ]; then
        print_error "Agent 二进制文件不存在: $PROJECT_ROOT/agent/bin/agent"
        return 1
    fi
    
    # 使用绝对路径
    local abs_config_file="$CONFIG_DIR/$(basename $config_file)"
    if [ ! -f "$abs_config_file" ]; then
        print_error "Agent 配置文件不存在: $abs_config_file"
        return 1
    fi
    
    # 启动 Agent（在项目根目录下运行，以便配置文件路径正确）
    print_info "启动 Agent $agent_id (HTTP: $port, Config: $abs_config_file)..."
    cd "$PROJECT_ROOT"
    
    # 确保日志目录存在
    mkdir -p "$LOG_DIR"
    
    # 启动 Agent，将 stderr 和 stdout 都重定向到日志文件
    nohup "$PROJECT_ROOT/agent/bin/agent" -config "$abs_config_file" > "$LOG_DIR/agent-${agent_id}.log" 2>&1 &
    local agent_pid=$!
    echo $agent_pid > "$PID_DIR/${agent_id}.pid"
    
    # 等待一下确保进程启动
    sleep 2
    
    # 检查进程是否还在运行
    if ! ps -p $agent_pid > /dev/null 2>&1; then
        print_error "Agent $agent_id 进程启动后立即退出"
        print_info "查看日志: $LOG_DIR/agent-${agent_id}.log"
        if [ -f "$LOG_DIR/agent-${agent_id}.log" ]; then
            print_info "最后 20 行日志:"
            tail -20 "$LOG_DIR/agent-${agent_id}.log" || true
        fi
        return 1
    fi
    
    # 等待 Agent 就绪（健康检查）
    print_info "等待 Agent $agent_id 健康检查端点就绪..."
    if wait_for_service "http://127.0.0.1:$port/health" 30; then
        print_info "Agent $agent_id 启动成功 (PID: $agent_pid)"
        return 0
    else
        # 即使健康检查失败，如果进程还在运行，也认为启动成功（可能是健康检查端点未实现）
        if ps -p $agent_pid > /dev/null 2>&1; then
            print_warn "Agent $agent_id 进程运行中，但健康检查端点未响应"
            print_info "Agent $agent_id 进程运行中 (PID: $agent_pid)"
            print_info "查看日志: $LOG_DIR/agent-${agent_id}.log"
            return 0
        else
            print_error "Agent $agent_id 启动失败，进程已退出"
            print_info "查看日志: $LOG_DIR/agent-${agent_id}.log"
            if [ -f "$LOG_DIR/agent-${agent_id}.log" ]; then
                print_info "最后 20 行日志:"
                tail -20 "$LOG_DIR/agent-${agent_id}.log" || true
            fi
            return 1
        fi
    fi
}

# 验证服务健康状态
verify_services() {
    print_step "验证服务健康状态..."
    
    local all_ok=true
    
    # 检查 Manager
    if curl -s "http://127.0.0.1:8080/health" > /dev/null 2>&1; then
        print_info "✅ Manager HTTP API 正常"
    else
        print_error "❌ Manager HTTP API 异常"
        all_ok=false
    fi
    
    # 检查 Daemon 进程
    if [ -f "$PID_DIR/daemon.pid" ] && ps -p $(cat "$PID_DIR/daemon.pid") > /dev/null 2>&1; then
        print_info "✅ Daemon 进程运行正常"
    else
        print_error "❌ Daemon 进程异常"
        all_ok=false
    fi
    
    # 检查 Agent 进程
    for agent_id in agent-001 agent-002 agent-003; do
        if [ -f "$PID_DIR/$agent_id.pid" ] && ps -p $(cat "$PID_DIR/$agent_id.pid") > /dev/null 2>&1; then
            print_info "✅ Agent $agent_id 进程运行正常"
        else
            print_warn "⚠️  Agent $agent_id 进程未运行"
        fi
    done
    
    if [ "$all_ok" = true ]; then
        print_info "所有核心服务运行正常"
        return 0
    else
        print_error "部分服务异常，请检查日志"
        return 1
    fi
}

# 主函数
main() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}完整系统集成测试环境启动${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    
    # 检查必要命令
    print_step "检查必要命令..."
    check_command "mysql" || exit 1
    check_command "curl" || exit 1
    check_command "lsof" || exit 1
    check_command "go" || exit 1
    print_info "所有必要命令已安装"
    echo ""
    
    # 检查 MySQL
    if ! check_mysql; then
        print_error "MySQL 数据库不可用，请先启动 MySQL"
        exit 1
    fi
    echo ""
    
    # 启动 Manager
    if ! start_manager; then
        print_error "Manager 启动失败"
        exit 1
    fi
    echo ""
    
    # 启动 Daemon
    if ! start_daemon; then
        print_error "Daemon 启动失败"
        exit 1
    fi
    echo ""
    
    # 注意：不再手动启动 Agent，因为 Daemon 会根据配置自动启动所有 enabled 的 Agent
    # 这样可以避免重复启动，确保 Agent 由 Daemon 统一管理
    print_step "等待 Daemon 自动启动 Agent 实例..."
    print_info "Daemon 会根据配置文件自动启动所有 enabled 的 Agent"
    print_info "等待 Agent 启动（最多30秒）..."
    
    local max_wait=30
    local waited=0
    local all_agents_started=false
    
    while [ $waited -lt $max_wait ]; do
        sleep 2
        waited=$((waited + 2))
        
        # 检查 Agent 进程（由 Daemon 启动的，父进程应该是 Daemon）
        local agent_count=$(ps aux | grep -E "[a]gent/bin/agent.*test/integration/config" | wc -l | tr -d ' ')
        if [ "$agent_count" -ge 3 ]; then
            all_agents_started=true
            print_info "检测到 $agent_count 个 Agent 进程（由 Daemon 启动）"
            break
        fi
        
        if [ $waited -lt $max_wait ]; then
            print_info "  等待中... (${waited}s/${max_wait}s, 当前进程数: $agent_count)"
        fi
    done
    
    if [ "$all_agents_started" = "false" ]; then
        print_warn "部分 Agent 可能未启动，请检查 Daemon 日志"
        print_info "Daemon 日志: $LOG_DIR/daemon.log"
    else
        print_info "Agent 启动完成（由 Daemon 管理）"
    fi
    echo ""
    
    # 验证服务
    sleep 5  # 等待所有服务完全启动
    verify_services
    echo ""
    
    # 输出测试环境信息
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}测试环境启动完成${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "服务信息:"
    echo "  - Manager HTTP API: http://127.0.0.1:8080"
    echo "  - Manager gRPC: 127.0.0.1:9090"
    echo "  - Daemon gRPC: 127.0.0.1:9091"
    echo "  - Agent-001 HTTP: http://127.0.0.1:8081"
    echo "  - Agent-002 HTTP: http://127.0.0.1:8082"
    echo "  - Agent-003 HTTP: http://127.0.0.1:8083"
    echo ""
    echo "日志目录: $LOG_DIR"
    echo "PID 文件: $PID_DIR"
    echo "配置文件: $CONFIG_DIR"
    echo ""
    echo "停止测试环境: $TEST_DIR/cleanup_test_env.sh"
    echo ""
}

# 执行主函数
main "$@"
