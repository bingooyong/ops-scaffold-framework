#!/bin/bash
# 诊断和修复脚本
# 功能: 诊断测试环境问题并自动修复

set +e  # 允许函数返回非零值

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

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

# 诊断问题
diagnose() {
    print_step "诊断测试环境问题..."
    
    local issues=0
    
    # 1. 检查 Daemon 状态同步错误
    print_info "检查 Daemon 状态同步..."
    if grep -q "unknown service proto.DaemonService" "$SCRIPT_DIR/logs/daemon.log" 2>/dev/null; then
        print_warn "发现: Daemon 状态同步失败 - unknown service proto.DaemonService"
        print_info "原因: Manager gRPC 服务器可能未正确注册 DaemonService"
        print_info "解决: 需要重启 Manager 服务"
        ((issues++))
    fi
    
    # 2. 检查数据库字段缺失
    print_info "检查数据库错误..."
    if grep -q "Unknown column 'last_heartbeat_at'" "$SCRIPT_DIR/logs/manager.log" 2>/dev/null; then
        print_warn "发现: 数据库字段缺失 - last_heartbeat_at"
        print_info "原因: 数据库表结构可能不完整"
        print_info "解决: 需要运行数据库迁移或手动添加字段"
        ((issues++))
    fi
    
    # 3. 检查节点注册
    print_info "检查节点注册..."
    local node_id=$(cat "$SCRIPT_DIR/tmp/daemon/node_id" 2>/dev/null || echo "")
    if [ -z "$node_id" ]; then
        print_warn "发现: Node ID 文件不存在"
        ((issues++))
    else
        print_info "Node ID: $node_id"
    fi
    
    # 4. 检查服务运行状态
    print_info "检查服务运行状态..."
    if ! curl -s http://127.0.0.1:8080/health > /dev/null 2>&1; then
        print_warn "发现: Manager 服务未运行"
        ((issues++))
    else
        print_info "Manager 服务运行正常"
    fi
    
    if [ ! -f "$SCRIPT_DIR/pids/daemon.pid" ] || ! ps -p $(cat "$SCRIPT_DIR/pids/daemon.pid") > /dev/null 2>&1; then
        print_warn "发现: Daemon 服务未运行"
        ((issues++))
    else
        print_info "Daemon 服务运行正常"
    fi
    
    if [ $issues -eq 0 ]; then
        print_info "未发现明显问题"
    else
        print_warn "发现 $issues 个问题"
    fi
    
    return $issues
}

# 修复问题
fix() {
    print_step "开始修复问题..."
    
    # 1. 重启 Manager（修复 gRPC 服务注册问题）
    if grep -q "unknown service proto.DaemonService" "$SCRIPT_DIR/logs/daemon.log" 2>/dev/null || [ "${FORCE_RESTART_MANAGER:-}" = "yes" ]; then
        print_info "重启 Manager 服务以修复 gRPC 服务注册..."
        if [ -f "$SCRIPT_DIR/pids/manager.pid" ]; then
            local pid=$(cat "$SCRIPT_DIR/pids/manager.pid")
            if ps -p "$pid" > /dev/null 2>&1; then
                print_info "停止 Manager (PID: $pid)..."
                kill -TERM "$pid" 2>/dev/null || true
                sleep 3
                if ps -p "$pid" > /dev/null 2>&1; then
                    kill -9 "$pid" 2>/dev/null || true
                fi
            fi
        fi
        
        # 重新启动 Manager
        print_info "启动 Manager..."
        cd "$PROJECT_ROOT"
        nohup "$PROJECT_ROOT/manager/bin/manager" -config "$SCRIPT_DIR/config/manager.test.yaml" > "$SCRIPT_DIR/logs/manager.log" 2>&1 &
        echo $! > "$SCRIPT_DIR/pids/manager.pid"
        sleep 5
        
        # 验证 Manager 启动
        if curl -s http://127.0.0.1:8080/health > /dev/null 2>&1; then
            print_info "Manager 重启成功"
        else
            print_error "Manager 重启失败，查看日志: $SCRIPT_DIR/logs/manager.log"
            return 1
        fi
    fi
    
    # 2. 等待 Daemon 重新连接
    print_info "等待 Daemon 重新连接到 Manager..."
    sleep 5
    
    # 3. 检查状态同步
    print_info "检查状态同步..."
    sleep 5
    if ! grep -q "unknown service proto.DaemonService" "$SCRIPT_DIR/logs/daemon.log" 2>/dev/null; then
        print_info "状态同步错误已解决"
    else
        print_warn "状态同步错误仍然存在，可能需要检查代码"
    fi
    
    return 0
}

# 主函数
main() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}测试环境诊断和修复${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    
    # 诊断
    local issues=0
    diagnose; issues=$?  # 获取返回值，但不因非零退出
    echo ""
    
    print_info "诊断完成，发现 $issues 个问题"
    
    if [ $issues -gt 0 ]; then
        # 支持非交互式执行（通过环境变量或参数）
        local auto_fix=false
        if [ "${AUTO_FIX:-}" = "yes" ] || [ "${1:-}" = "--auto-fix" ] || [ "${1:-}" = "-y" ]; then
            auto_fix=true
        fi
        
        if [ "$auto_fix" = "true" ]; then
            print_info "自动修复模式：开始修复..."
            fix || true
            echo ""
            print_step "重新诊断..."
            diagnose || true
        else
            read -p "是否自动修复这些问题? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                fix || true
                echo ""
                print_step "重新诊断..."
                diagnose || true
            else
                print_info "跳过自动修复"
            fi
        fi
    fi
    
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}诊断完成${NC}"
    echo -e "${GREEN}========================================${NC}"
}

main "$@"
