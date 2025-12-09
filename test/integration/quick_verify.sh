#!/bin/bash
# 一键运行Daemon Agent管理功能验证
# 自动构建、清理、测试并生成报告

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Daemon Agent管理功能一键验证脚本             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""

# 步骤1: 构建二进制
echo -e "${BLUE}[步骤 1/4] 构建二进制文件...${NC}"
echo "----------------------------------------"

if [ ! -f "$PROJECT_ROOT/daemon/daemon" ]; then
    echo "构建 Daemon..."
    cd "$PROJECT_ROOT/daemon" && make build
    echo -e "${GREEN}✓ Daemon构建完成${NC}"
else
    echo -e "${GREEN}✓ Daemon二进制已存在${NC}"
fi

if [ ! -f "$PROJECT_ROOT/agent/bin/agent" ]; then
    echo "构建 Agent..."
    cd "$PROJECT_ROOT/agent" && make build
    echo -e "${GREEN}✓ Agent构建完成${NC}"
else
    echo -e "${GREEN}✓ Agent二进制已存在${NC}"
fi

echo ""

# 步骤2: 清理环境
echo -e "${BLUE}[步骤 2/4] 清理测试环境...${NC}"
echo "----------------------------------------"

# 停止可能运行的进程
pkill -f "daemon/daemon" 2>/dev/null || true
pkill -f "agent/bin/agent" 2>/dev/null || true
rm -f /tmp/daemon.sock

# 清理临时文件（保留logs以便查看历史）
rm -rf "$PROJECT_ROOT/test/integration/tmp"
rm -rf "$PROJECT_ROOT/test/integration/pids"
mkdir -p "$PROJECT_ROOT/test/integration/logs"
mkdir -p "$PROJECT_ROOT/test/integration/reports"

echo -e "${GREEN}✓ 环境已清理${NC}"
echo ""

# 步骤3: 运行测试
echo -e "${BLUE}[步骤 3/4] 运行Daemon独立测试...${NC}"
echo "----------------------------------------"

cd "$PROJECT_ROOT/test/integration"
./test_daemon_simple.sh

test_exit_code=$?

echo ""

# 步骤4: 显示结果
echo -e "${BLUE}[步骤 4/4] 测试结果${NC}"
echo "----------------------------------------"

if [ $test_exit_code -eq 0 ]; then
    echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           ✅ 所有测试通过！                    ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BLUE}Daemon的Agent管理功能验证成功！${NC}"
    echo ""
    echo "✓ Agent自动加载和注册"
    echo "✓ Agent进程启动和管理"
    echo "✓ 元数据持久化"
    echo "✓ 日志记录完整"
    echo ""
else
    echo -e "${RED}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║           ❌ 测试失败！                        ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${YELLOW}请检查以下内容：${NC}"
    echo "1. 测试报告: test/integration/reports/daemon_standalone_test_report.md"
    echo "2. Daemon日志: test/integration/logs/daemon.log"
    echo "3. Agent日志: test/integration/logs/agent-*.log"
    echo ""
fi

# 查看测试报告
if [ -f "$PROJECT_ROOT/test/integration/reports/daemon_standalone_test_report.md" ]; then
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}测试报告位置:${NC}"
    echo "  $PROJECT_ROOT/test/integration/reports/daemon_standalone_test_report.md"
    echo ""
    echo -e "${BLUE}快速查看报告:${NC}"
    echo "  cat test/integration/reports/daemon_standalone_test_report.md"
    echo ""
    if command -v open &> /dev/null; then
        echo "  open test/integration/reports/daemon_standalone_test_report.md"
    elif command -v xdg-open &> /dev/null; then
        echo "  xdg-open test/integration/reports/daemon_standalone_test_report.md"
    fi
    echo -e "${BLUE}========================================${NC}"
fi

# 提供下一步建议
echo ""
echo -e "${YELLOW}📋 下一步建议：${NC}"
if [ $test_exit_code -eq 0 ]; then
    echo ""
    echo "Daemon的基础Agent管理功能已验证，现在可以："
    echo ""
    echo "1. 测试Manager与Daemon的通信："
    echo "   cd test/integration"
    echo "   ./start_test_env.sh"
    echo "   ./test_business_flows.sh"
    echo ""
    echo "2. 查看详细的Agent操作日志："
    echo "   tail -f test/integration/logs/daemon.log | grep -i agent"
    echo ""
    echo "3. 手动测试Agent操作（需要安装grpcurl）："
    echo "   grpcurl -plaintext -d '{\"agent_id\": \"agent-001\", \"operation\": \"stop\"}' \\"
    echo "     localhost:9091 proto.DaemonService/OperateAgent"
    echo ""
else
    echo ""
    echo "请先解决Daemon独立测试中的问题，然后再进行集成测试。"
    echo ""
    echo "常见问题排查："
    echo "  - 检查Agent二进制: ls -lh agent/bin/agent"
    echo "  - 检查Daemon二进制: ls -lh daemon/daemon"
    echo "  - 查看错误日志: grep -i error test/integration/logs/daemon.log"
    echo "  - 检查端口占用: lsof -i :9091"
    echo ""
fi

exit $test_exit_code
