#!/bin/bash
# 完整系统集成测试环境清理脚本
# 功能: 停止所有测试服务并清理临时文件

# 不使用 set -e，因为某些操作失败是正常的（如进程不存在）
# 我们会在关键位置手动检查错误
set +e

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

# 打印函数
print_info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

print_warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

print_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
}

print_step() {
    printf "${BLUE}[STEP]${NC} %s\n" "$1"
}

# 通过端口停止进程
stop_process_by_port() {
    local name=$1
    local port=$2
    
    if [ -z "$port" ]; then
        return 1
    fi
    
    local pid=$(lsof -ti:$port 2>/dev/null || echo "")
    if [ -z "$pid" ]; then
        return 1
    fi
    
    if ! ps -p "$pid" > /dev/null 2>&1; then
        return 1
    fi
    
    print_info "通过端口 $port 找到 $name 进程 (PID: $pid)，正在停止..."
    
    # 尝试优雅停止
    kill -TERM "$pid" 2>/dev/null || true
    sleep 2
    
    # 如果还在运行，强制停止
    if ps -p "$pid" > /dev/null 2>&1; then
        print_warn "强制停止 $name..."
        kill -9 "$pid" 2>/dev/null || true
        sleep 1
    fi
    
    # 确认已停止
    if ! ps -p "$pid" > /dev/null 2>&1; then
        print_info "$name 已停止"
        return 0
    else
        print_error "$name 停止失败"
        return 1
    fi
}

# 通过进程名停止进程
stop_process_by_name() {
    local name=$1
    local pattern=$2  # 进程匹配模式
    
    if [ -z "$pattern" ]; then
        return 1
    fi
    
    # 查找所有匹配的进程
    local pids=$(pgrep -f "$pattern" 2>/dev/null || echo "")
    if [ -z "$pids" ]; then
        return 1
    fi
    
    for pid in $pids; do
        if ! ps -p "$pid" > /dev/null 2>&1; then
            continue
        fi
        
        print_info "通过进程名找到 $name 进程 (PID: $pid)，正在停止..."
        
        # 尝试优雅停止
        kill -TERM "$pid" 2>/dev/null || true
        sleep 2
        
        # 如果还在运行，强制停止
        if ps -p "$pid" > /dev/null 2>&1; then
            print_warn "强制停止 $name (PID: $pid)..."
            kill -9 "$pid" 2>/dev/null || true
            sleep 1
        fi
        
        # 确认已停止
        if ! ps -p "$pid" > /dev/null 2>&1; then
            print_info "$name (PID: $pid) 已停止"
        else
            print_error "$name (PID: $pid) 停止失败"
        fi
    done
    
    return 0
}

# 停止进程
stop_process() {
    local name=$1
    local port=$2  # 可选：如果提供端口，PID 文件不存在时通过端口查找
    local pattern=$3  # 可选：进程匹配模式
    local pid_file="$PID_DIR/$name.pid"
    
    if [ ! -f "$pid_file" ]; then
        # 如果提供了进程模式，优先通过进程名查找（更可靠）
        if [ -n "$pattern" ]; then
            if stop_process_by_name "$name" "$pattern"; then
                return 0
            fi
        fi
        # 如果提供了端口，尝试通过端口查找
        if [ -n "$port" ]; then
            if stop_process_by_port "$name" "$port"; then
                return 0
            fi
        fi
        print_warn "$name PID 文件不存在，可能未运行"
        return 0
    fi
    
    local pid=$(cat "$pid_file")
    
    if ! ps -p "$pid" > /dev/null 2>&1; then
        print_warn "$name 进程不存在 (PID: $pid)"
        rm -f "$pid_file"
        
        # 如果提供了进程模式，优先通过进程名查找（更可靠）
        if [ -n "$pattern" ]; then
            if stop_process_by_name "$name" "$pattern"; then
                return 0
            fi
        fi
        # 如果提供了端口，尝试通过端口查找
        if [ -n "$port" ]; then
            if stop_process_by_port "$name" "$port"; then
                return 0
            fi
        fi
        return 0
    fi
    
    print_info "停止 $name (PID: $pid)..."
    
    # 尝试优雅停止
    kill -TERM "$pid" 2>/dev/null || true
    sleep 2
    
    # 如果还在运行，强制停止
    if ps -p "$pid" > /dev/null 2>&1; then
        print_warn "强制停止 $name..."
        kill -9 "$pid" 2>/dev/null || true
        sleep 1
    fi
    
    # 确认已停止
    if ! ps -p "$pid" > /dev/null 2>&1; then
        print_info "$name 已停止"
        rm -f "$pid_file"
        return 0
    else
        print_error "$name 停止失败"
        # 如果停止失败，尝试通过进程名查找并停止
        if [ -n "$pattern" ]; then
            stop_process_by_name "$name" "$pattern"
        fi
        return 1
    fi
}

# 停止所有 Agent
stop_all_agents() {
    print_step "停止所有测试 Agent..."
    
    # 首先通过进程名停止所有 agent 进程（最可靠的方法）
    print_info "通过进程名查找并停止所有 Agent 进程..."
    local agent_pattern="agent/bin/agent.*test/integration"
    local agent_pids=$(pgrep -f "$agent_pattern" 2>/dev/null || echo "")
    
    if [ -n "$agent_pids" ]; then
        for pid in $agent_pids; do
            if ps -p "$pid" > /dev/null 2>&1; then
                print_info "找到 Agent 进程 (PID: $pid)，正在停止..."
                kill -TERM "$pid" 2>/dev/null || true
                sleep 1
                if ps -p "$pid" > /dev/null 2>&1; then
                    kill -9 "$pid" 2>/dev/null || true
                    sleep 1
                fi
                if ! ps -p "$pid" > /dev/null 2>&1; then
                    print_info "Agent 进程 (PID: $pid) 已停止"
                fi
            fi
        done
    fi
    
    # 然后尝试通过 PID 文件和端口停止（兼容旧方式）
    for agent_id in agent-001 agent-002 agent-003; do
        # 尝试两种 PID 文件命名格式（兼容旧版本和新版本）
        local pid_file1="$PID_DIR/$agent_id.pid"
        local pid_file2="$PID_DIR/agent-$agent_id.pid"
        
        # 优先使用新格式（agent-agent-001.pid）
        if [ -f "$pid_file2" ]; then
            stop_process "agent-$agent_id" "" "agent/bin/agent.*$agent_id"
        elif [ -f "$pid_file1" ]; then
            stop_process "$agent_id" "" "agent/bin/agent.*$agent_id"
        else
            # 如果 PID 文件不存在，尝试通过端口查找进程
            local port=""
            case "$agent_id" in
                agent-001) port=8081 ;;
                agent-002) port=8082 ;;
                agent-003) port=8083 ;;
            esac
            
            if [ -n "$port" ]; then
                stop_process_by_port "agent-$agent_id" "$port" || true
            fi
        fi
    done
    
    # 最后再次检查是否还有残留的 Agent 进程
    sleep 1
    local remaining_pids=$(pgrep -f "$agent_pattern" 2>/dev/null || echo "")
    if [ -n "$remaining_pids" ]; then
        print_warn "发现残留的 Agent 进程，强制停止..."
        for pid in $remaining_pids; do
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
            fi
        done
        sleep 1
    fi
}

# 主函数
main() {
    printf "${GREEN}========================================${NC}\n"
    printf "${GREEN}完整系统集成测试环境清理${NC}\n"
    printf "${GREEN}========================================${NC}\n"
    echo ""
    
    # 1. 先停止 Daemon（防止自动重启 Agent）
    print_step "停止 Daemon 服务..."
    stop_process "daemon" "9091" "daemon/bin/daemon.*test/integration" || true  # Daemon gRPC 端口和进程模式
    echo ""
    
    # 2. 等待 Daemon 完全停止
    print_info "等待 Daemon 完全停止（2秒）..."
    sleep 2
    echo ""
    
    # 3. 停止所有 Agent
    stop_all_agents || true
    echo ""
    
    # 4. 停止 Manager
    print_step "停止 Manager 服务..."
    # 优先通过进程名查找（更可靠），尝试多种模式
    local manager_stopped=false
    # 尝试精确匹配
    if stop_process_by_name "manager" "manager/bin/manager.*test/integration"; then
        manager_stopped=true
    # 尝试更通用的模式
    elif stop_process_by_name "manager" "manager/bin/manager.*test"; then
        manager_stopped=true
    # 尝试最通用的模式（匹配所有 test 相关的 manager）
    elif stop_process_by_name "manager" "manager/bin/manager.*test"; then
        manager_stopped=true
    fi
    
    # 如果进程名查找失败，尝试通过端口查找（但要验证确实是 manager 进程）
    if [ "$manager_stopped" = false ]; then
        local port_pid=$(lsof -ti:8080 2>/dev/null | head -1)
        if [ -n "$port_pid" ]; then
            # 验证进程确实是 manager
            local cmd=$(ps -p "$port_pid" -o command= 2>/dev/null || echo "")
            if echo "$cmd" | grep -q "manager/bin/manager.*test"; then
                print_info "通过端口 8080 找到 manager 进程 (PID: $port_pid)，正在停止..."
                kill -TERM "$port_pid" 2>/dev/null || true
                sleep 2
                if ps -p "$port_pid" > /dev/null 2>&1; then
                    kill -9 "$port_pid" 2>/dev/null || true
                    sleep 1
                fi
                if ! ps -p "$port_pid" > /dev/null 2>&1; then
                    print_info "manager (PID: $port_pid) 已停止"
                    manager_stopped=true
                fi
            fi
        fi
    fi
    
    if [ "$manager_stopped" = false ]; then
        print_warn "manager 进程未找到，可能未运行"
    fi
    echo ""
    
    # 5. 默认清理日志文件
    print_step "清理日志文件..."
    if [ -d "$LOG_DIR" ]; then
        rm -rf "$LOG_DIR"/*
        print_info "日志文件已清理"
    else
        print_warn "日志目录不存在: $LOG_DIR"
    fi
    echo ""
    
    # 清理选项
    if [ "${1:-}" = "--clean-all" ]; then
        print_step "清理所有临时文件..."
        if [ -d "$PID_DIR" ]; then
            rm -rf "$PID_DIR"/*
            print_info "PID 文件已清理"
        fi
        echo ""
    fi
    
    echo ""
    printf "${GREEN}========================================${NC}\n"
    printf "${GREEN}测试环境清理完成${NC}\n"
    printf "${GREEN}========================================${NC}\n"
    echo ""
    echo "提示:"
    echo "  - 清理所有（包括 PID 文件）: $0 --clean-all"
    echo ""
}

# 执行主函数
main "$@"
