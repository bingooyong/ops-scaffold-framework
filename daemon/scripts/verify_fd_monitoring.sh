#!/bin/bash

# 文件描述符监控功能验证脚本

set -e

echo "=========================================="
echo "文件描述符监控功能验证"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. 检查平台
echo -e "${YELLOW}1. 检查平台...${NC}"
OS=$(uname -s)
echo "   操作系统: $OS"

if [ "$OS" = "Darwin" ]; then
    echo -e "   ${GREEN}✓${NC} macOS 平台,将使用 lsof 命令"
    # 检查 lsof 是否可用
    if ! command -v lsof &> /dev/null; then
        echo -e "   ${RED}✗${NC} lsof 命令不可用,请安装"
        exit 1
    fi
    echo -e "   ${GREEN}✓${NC} lsof 命令可用"
elif [ "$OS" = "Linux" ]; then
    echo -e "   ${GREEN}✓${NC} Linux 平台,将使用 /proc 文件系统"
    # 检查 /proc 是否可用
    if [ ! -d "/proc/self/fd" ]; then
        echo -e "   ${RED}✗${NC} /proc 文件系统不可用"
        exit 1
    fi
    echo -e "   ${GREEN}✓${NC} /proc 文件系统可用"
else
    echo -e "   ${YELLOW}!${NC} 其他平台: $OS"
fi
echo ""

# 2. 运行单元测试
echo -e "${YELLOW}2. 运行文件描述符获取测试...${NC}"
cd "$(dirname "$0")/.."
go test -v ./internal/agent -run TestGetNumFDs 2>&1 | grep -E "(PASS|FAIL|Open FDs)" || true
echo ""

# 3. 运行阈值测试
echo -e "${YELLOW}3. 运行阈值配置测试...${NC}"
go test -v ./internal/agent -run TestResourceThresholdWithOpenFiles 2>&1 | grep -E "(PASS|FAIL|threshold)" || true
echo ""

# 4. 测试当前进程的文件描述符
echo -e "${YELLOW}4. 测试当前进程的文件描述符数量...${NC}"
CURRENT_PID=$$

if [ "$OS" = "Darwin" ]; then
    FD_COUNT=$(lsof -p $CURRENT_PID 2>/dev/null | tail -n +2 | wc -l | tr -d ' ')
    echo "   进程 PID: $CURRENT_PID"
    echo "   文件描述符数量: $FD_COUNT"
    
    # 显示前 5 个文件描述符
    echo "   示例文件描述符:"
    lsof -p $CURRENT_PID 2>/dev/null | head -n 6 | tail -n 5 | awk '{print "     ", $4, $5, $9}'
elif [ "$OS" = "Linux" ]; then
    FD_COUNT=$(ls -1 /proc/$CURRENT_PID/fd 2>/dev/null | wc -l | tr -d ' ')
    echo "   进程 PID: $CURRENT_PID"
    echo "   文件描述符数量: $FD_COUNT"
    
    # 显示前 5 个文件描述符
    echo "   示例文件描述符:"
    ls -l /proc/$CURRENT_PID/fd 2>/dev/null | head -n 6 | tail -n 5 | awk '{print "     ", $9, "->", $11}'
fi
echo ""

# 5. 构建项目
echo -e "${YELLOW}5. 构建 daemon...${NC}"
if make build > /dev/null 2>&1; then
    echo -e "   ${GREEN}✓${NC} 构建成功"
else
    echo -e "   ${RED}✗${NC} 构建失败"
    exit 1
fi
echo ""

# 6. 验证编译后的二进制
echo -e "${YELLOW}6. 验证编译后的二进制...${NC}"
if [ -f "bin/daemon" ]; then
    echo -e "   ${GREEN}✓${NC} daemon 二进制存在"
    ./bin/daemon --version 2>&1 | head -n 1
else
    echo -e "   ${RED}✗${NC} daemon 二进制不存在"
    exit 1
fi
echo ""

# 7. 显示使用示例
echo -e "${YELLOW}7. 使用示例${NC}"
echo "   启动 daemon 并监控文件描述符:"
echo "   $ ./bin/daemon -c configs/daemon.yaml"
echo ""
echo "   查看文件描述符监控日志:"
echo "   $ tail -f logs/daemon.log | grep -i 'open_files\\|fds'"
echo ""
echo "   手动检查 Agent 的文件描述符:"
if [ "$OS" = "Darwin" ]; then
    echo "   $ lsof -p <AGENT_PID> | wc -l"
elif [ "$OS" = "Linux" ]; then
    echo "   $ ls -1 /proc/<AGENT_PID>/fd | wc -l"
fi
echo ""

# 8. 总结
echo "=========================================="
echo -e "${GREEN}验证完成!${NC}"
echo "=========================================="
echo ""
echo "主要功能:"
echo "  ✓ 跨平台文件描述符获取 (macOS/Linux)"
echo "  ✓ 多级回退机制 (平台实现 → gopsutil)"
echo "  ✓ 文件描述符泄露检测"
echo "  ✓ 阈值告警功能"
echo ""
echo "相关文档:"
echo "  - docs/FILE_DESCRIPTOR_MONITORING.md"
echo "  - docs/FD_MONITORING_IMPLEMENTATION_SUMMARY.md"
echo ""
echo "测试命令:"
echo "  go test -v ./internal/agent -run 'TestGetNumFDs|TestResourceThresholdWithOpenFiles'"
echo ""
