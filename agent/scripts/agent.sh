#!/bin/bash
# Agent 启动脚本 - 用于快速启动多个 Agent 实例

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_PATH="$PROJECT_DIR/bin/agent"
CONFIGS_DIR="$PROJECT_DIR/configs"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# 检查二进制文件
check_binary() {
    if [ ! -f "$BIN_PATH" ]; then
        print_error "Agent binary not found: $BIN_PATH"
        print_info "Please run 'make build' first"
        exit 1
    fi
}

# 启动单个 Agent
start_agent() {
    local config_file=$1
    local agent_name=$(basename "$config_file" .yaml)
    
    print_info "Starting $agent_name..."
    
    # 启动 Agent（后台运行）
    nohup "$BIN_PATH" -config "$config_file" > "/tmp/${agent_name}.out" 2>&1 &
    local pid=$!
    
    # 等待一下确保启动
    sleep 1
    
    # 检查进程是否还在运行
    if ps -p $pid > /dev/null; then
        print_info "$agent_name started successfully (PID: $pid)"
        echo "$pid" > "/tmp/${agent_name}.pid"
    else
        print_error "$agent_name failed to start"
        print_info "Check log: /tmp/${agent_name}.out"
        return 1
    fi
}

# 停止单个 Agent
stop_agent() {
    local agent_name=$1
    local pid_file="/tmp/${agent_name}.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p $pid > /dev/null; then
            print_info "Stopping $agent_name (PID: $pid)..."
            kill -TERM $pid
            sleep 1
            if ps -p $pid > /dev/null; then
                print_warn "Force killing $agent_name..."
                kill -9 $pid
            fi
            print_info "$agent_name stopped"
        else
            print_warn "$agent_name is not running"
        fi
        rm -f "$pid_file"
    else
        print_warn "PID file not found for $agent_name"
    fi
}

# 检查 Agent 状态
check_agent() {
    local agent_name=$1
    local pid_file="/tmp/${agent_name}.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p $pid > /dev/null; then
            print_info "$agent_name is running (PID: $pid)"
            return 0
        else
            print_warn "$agent_name is not running (stale PID file)"
            return 1
        fi
    else
        print_warn "$agent_name is not running"
        return 1
    fi
}

# 显示使用帮助
usage() {
    cat <<EOF
Usage: $0 [start|stop|restart|status] [agent-name]

Commands:
    start [agent-name]    Start agent(s)
    stop [agent-name]     Stop agent(s)
    restart [agent-name]  Restart agent(s)
    status [agent-name]   Check agent(s) status

Agent names:
    agent-001             Agent with config agent.yaml
    agent-002             Agent with config agent-002.yaml
    agent-003             Agent with config agent-003.yaml
    all                   All agents (default)

Examples:
    $0 start              # Start all agents
    $0 start agent-001    # Start only agent-001
    $0 stop all           # Stop all agents
    $0 status             # Check status of all agents

EOF
}

# 主函数
main() {
    local command=${1:-status}
    local agent_name=${2:-all}
    
    check_binary
    
    case "$command" in
        start)
            if [ "$agent_name" = "all" ]; then
                start_agent "$CONFIGS_DIR/agent.yaml"
                start_agent "$CONFIGS_DIR/agent-002.yaml"
                start_agent "$CONFIGS_DIR/agent-003.yaml"
            elif [ "$agent_name" = "agent-001" ]; then
                start_agent "$CONFIGS_DIR/agent.yaml"
            else
                start_agent "$CONFIGS_DIR/${agent_name}.yaml"
            fi
            ;;
        stop)
            if [ "$agent_name" = "all" ]; then
                stop_agent "agent"
                stop_agent "agent-002"
                stop_agent "agent-003"
            elif [ "$agent_name" = "agent-001" ]; then
                stop_agent "agent"
            else
                stop_agent "$agent_name"
            fi
            ;;
        restart)
            if [ "$agent_name" = "all" ]; then
                $0 stop all
                sleep 2
                $0 start all
            else
                $0 stop "$agent_name"
                sleep 2
                $0 start "$agent_name"
            fi
            ;;
        status)
            if [ "$agent_name" = "all" ]; then
                check_agent "agent"
                check_agent "agent-002"
                check_agent "agent-003"
            elif [ "$agent_name" = "agent-001" ]; then
                check_agent "agent"
            else
                check_agent "$agent_name"
            fi
            ;;
        *)
            usage
            exit 1
            ;;
    esac
}

main "$@"
