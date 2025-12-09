#!/bin/bash
# Agent操作超时问题验证脚本

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Agent操作超时问题修复验证${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 1. 检查文件修改
echo -e "${GREEN}[1/5] 检查修改的文件...${NC}"

check_file() {
    local file=$1
    local pattern=$2
    if grep -q "$pattern" "$file"; then
        echo -e "  ${GREEN}✓${NC} $file - 已更新"
    else
        echo -e "  ${RED}✗${NC} $file - 未找到预期修改"
        return 1
    fi
}

MANAGER_CLIENT="manager/internal/grpc/daemon_client.go"
DAEMON_CONFIG="daemon/internal/daemon/daemon.go"
DAEMON_SERVER="daemon/internal/grpc/server.go"
MANAGER_SERVICE="manager/internal/service/agent.go"

check_file "$MANAGER_CLIENT" "operateAgentTimeout = 90"
check_file "$MANAGER_CLIENT" "keepaliveTime = 45"
check_file "$DAEMON_CONFIG" "Time:.*120.*time.Second"
check_file "$DAEMON_CONFIG" "MaxConnectionAgeGrace:.*10.*time.Second"
check_file "$DAEMON_SERVER" "received OperateAgent request"
check_file "$MANAGER_SERVICE" "isConnectionError"

echo ""

# 2. 编译检查
echo -e "${GREEN}[2/5] 编译检查...${NC}"

echo "  编译 Manager..."
if cd manager && go build -o /dev/null ./cmd/manager 2>&1; then
    echo -e "  ${GREEN}✓${NC} Manager 编译成功"
else
    echo -e "  ${RED}✗${NC} Manager 编译失败"
    exit 1
fi

cd ..

echo "  编译 Daemon..."
if cd daemon && go build -o /dev/null ./cmd/daemon 2>&1; then
    echo -e "  ${GREEN}✓${NC} Daemon 编译成功"
else
    echo -e "  ${RED}✗${NC} Daemon 编译失败"
    exit 1
fi

cd ..

echo ""

# 3. 生成二进制
echo -e "${GREEN}[3/5] 构建二进制文件...${NC}"

echo "  构建 Manager..."
if cd manager && make build > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓${NC} Manager 构建成功"
    ls -lh manager 2>/dev/null || echo "  (二进制文件: manager/manager)"
else
    echo -e "  ${YELLOW}⚠${NC}  Manager make build 失败，尝试直接构建..."
    go build -o manager ./cmd/manager
fi

cd ..

echo "  构建 Daemon..."
if cd daemon && make build > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓${NC} Daemon 构建成功"
    ls -lh daemon 2>/dev/null || echo "  (二进制文件: daemon/daemon)"
else
    echo -e "  ${YELLOW}⚠${NC}  Daemon make build 失败，尝试直接构建..."
    go build -o daemon ./cmd/daemon
fi

cd ..

echo ""

# 4. 检查测试环境
echo -e "${GREEN}[4/5] 检查测试环境...${NC}"

if [ ! -d "test/integration" ]; then
    echo -e "  ${RED}✗${NC} 测试目录不存在"
    exit 1
fi

if [ ! -f "test/integration/test_business_flows.sh" ]; then
    echo -e "  ${RED}✗${NC} 测试脚本不存在"
    exit 1
fi

echo -e "  ${GREEN}✓${NC} 测试环境就绪"
echo ""

# 5. 提供运行指南
echo -e "${GREEN}[5/5] 测试运行指南${NC}"
echo ""
echo -e "${YELLOW}接下来的步骤：${NC}"
echo ""
echo "1. 清理并重启测试环境："
echo -e "   ${BLUE}cd test/integration${NC}"
echo -e "   ${BLUE}./cleanup_test_env.sh${NC}"
echo -e "   ${BLUE}./start_test_env.sh${NC}"
echo ""
echo "2. 运行集成测试："
echo -e "   ${BLUE}./test_business_flows.sh${NC}"
echo ""
echo "3. 查看测试报告："
echo -e "   ${BLUE}cat reports/business_flows_test_report.md${NC}"
echo ""
echo "4. 监控日志（在另一个终端）："
echo -e "   ${BLUE}# Manager日志${NC}"
echo -e "   ${BLUE}tail -f test/integration/logs/manager.log | grep -E 'OperateAgent|connection|timeout'${NC}"
echo ""
echo -e "   ${BLUE}# Daemon日志${NC}"
echo -e "   ${BLUE}tail -f test/integration/logs/daemon.log | grep -E 'OperateAgent|received|completed'${NC}"
echo ""

# 6. 预期结果
echo -e "${YELLOW}预期测试结果：${NC}"
echo ""
echo -e "  ${GREEN}✓${NC} 场景2-停止Agent: 成功停止 Agent"
echo -e "  ${GREEN}✓${NC} 场景2-验证停止: Agent 进程已停止"
echo -e "  ${GREEN}✓${NC} 场景2-启动Agent: 成功启动 Agent"
echo -e "  ${GREEN}✓${NC} 场景2-验证启动: Agent 进程已启动"
echo -e "  ${GREEN}✓${NC} 场景2-重启Agent: 成功重启 Agent"
echo ""

# 7. 文档参考
echo -e "${YELLOW}相关文档：${NC}"
echo ""
echo "  - docs/bugfix-agent-operation-timeout.md (问题详细分析)"
echo "  - docs/bugfix-agent-operation-timeout-implementation.md (实施记录)"
echo "  - docs/bugfix-agent-operation-timeout-summary.md (完整解决方案)"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}验证完成！所有检查通过 ✓${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}提示：现在可以运行集成测试来验证修复效果${NC}"
echo ""
