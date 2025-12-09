#!/bin/bash
# 完整的gRPC Agent操作测试脚本
# 测试Stop、Start、Restart等所有操作

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LOGS_DIR="$PROJECT_ROOT/test/integration/logs"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Daemon gRPC Agent操作完整测试${NC}"
echo -e "${BLUE}========================================${NC}\n"

# 清空日志
echo -e "${YELLOW}[1/7] 清空日志...${NC}"
> "$LOGS_DIR/daemon.log"
> "$LOGS_DIR/manager.log"
echo -e "${GREEN}✓ 日志已清空${NC}\n"

# 运行Go测试程序
echo -e "${YELLOW}[2/7] 运行gRPC测试程序...${NC}"
cd "$PROJECT_ROOT/daemon"

# 运行测试并捕获输出
TEST_OUTPUT=$(timeout 180 go run ../test/integration/test_grpc_agent_ops.go 2>&1)
echo "$TEST_OUTPUT"

# 检查测试结果
if echo "$TEST_OUTPUT" | grep -q "Test Complete"; then
    echo -e "${GREEN}✓ 测试程序执行完成${NC}\n"
else
    echo -e "${RED}✗ 测试程序未正常完成${NC}\n"
fi

# 分析测试结果
echo -e "${YELLOW}[3/7] 分析测试结果...${NC}"

# 检查ListAgents
if echo "$TEST_OUTPUT" | grep -q "List agents succeeded"; then
    LIST_TIME=$(echo "$TEST_OUTPUT" | grep "List agents succeeded" | grep -oP 'took \K[0-9.]+ms')
    echo -e "${GREEN}✓ ListAgents成功 (耗时: ${LIST_TIME})${NC}"
else
    echo -e "${RED}✗ ListAgents失败${NC}"
fi

# 检查Stop操作
if echo "$TEST_OUTPUT" | grep -q "Stop result: success=true"; then
    echo -e "${GREEN}✓ Stop操作成功${NC}"
    if echo "$TEST_OUTPUT" | grep -A 5 "Listing agents after stop" | grep -q "agent-002.*status=stopped"; then
        echo -e "${GREEN}  ✓ Agent状态正确更新为stopped${NC}"
    else
        echo -e "${YELLOW}  ⚠ Agent状态未正确更新${NC}"
    fi
else
    echo -e "${RED}✗ Stop操作失败${NC}"
fi

# 检查Start操作
if echo "$TEST_OUTPUT" | grep -q "Start result: success=true"; then
    echo -e "${GREEN}✓ Start操作成功${NC}"
    if echo "$TEST_OUTPUT" | grep -A 3 "Verifying agent-002 after start" | grep -q "status=running.*pid=[0-9]"; then
        echo -e "${GREEN}  ✓ Agent状态正确更新为running${NC}"
    else
        echo -e "${YELLOW}  ⚠ Agent状态未正确更新（可能需要更多时间）${NC}"
    fi
else
    echo -e "${RED}✗ Start操作失败${NC}"
fi

# 检查Restart操作
if echo "$TEST_OUTPUT" | grep -q "Restart result: success=true"; then
    echo -e "${GREEN}✓ Restart操作成功${NC}"
else
    echo -e "${RED}✗ Restart操作失败${NC}"
fi

echo ""

# 检查Daemon日志
echo -e "${YELLOW}[4/7] 检查Daemon日志...${NC}"
DAEMON_LOG="$LOGS_DIR/daemon.log"

# 检查操作完成日志
OPERATION_COMPLETED=$(cat "$DAEMON_LOG" | strings | grep -c "agent operation completed" || echo "0")
echo "  - agent operation completed日志: $OPERATION_COMPLETED 条"

# 检查错误日志
ERROR_COUNT=$(cat "$DAEMON_LOG" | strings | grep -c '"level":"error"' || echo "0")
if [ "$ERROR_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}  ⚠ 发现 $ERROR_COUNT 条错误日志${NC}"
    echo "  最近的错误:"
    cat "$DAEMON_LOG" | strings | grep '"level":"error"' | tail -3 | sed 's/^/    /'
else
    echo -e "${GREEN}  ✓ 无错误日志${NC}"
fi

echo ""

# 检查Agent进程
echo -e "${YELLOW}[5/7] 检查Agent进程状态...${NC}"
AGENT_COUNT=$(ps aux | grep "[a]gent/bin/agent" | wc -l | tr -d ' ')
echo "  - 运行中的Agent进程数: $AGENT_COUNT"

if [ "$AGENT_COUNT" -eq 3 ]; then
    echo -e "${GREEN}  ✓ Agent进程数量正确 (3个)${NC}"
elif [ "$AGENT_COUNT" -gt 3 ]; then
    echo -e "${YELLOW}  ⚠ Agent进程数量异常 ($AGENT_COUNT个，可能有残留进程)${NC}"
else
    echo -e "${RED}  ✗ Agent进程数量不足 ($AGENT_COUNT个)${NC}"
fi

echo ""

# 检查gRPC响应时间
echo -e "${YELLOW}[6/7] 检查gRPC操作响应时间...${NC}"
cat "$DAEMON_LOG" | strings | grep "agent operation completed" | grep -oP 'duration":\K[0-9.]+' | head -5 | while read duration; do
    if (( $(echo "$duration < 1.0" | bc -l) )); then
        echo -e "${GREEN}  ✓ 操作耗时: ${duration}s (正常)${NC}"
    elif (( $(echo "$duration < 5.0" | bc -l) )); then
        echo -e "${YELLOW}  ⚠ 操作耗时: ${duration}s (较慢)${NC}"
    else
        echo -e "${RED}  ✗ 操作耗时: ${duration}s (很慢)${NC}"
    fi
done

echo ""

# 生成测试报告
echo -e "${YELLOW}[7/7] 生成测试报告...${NC}"
REPORT_FILE="$PROJECT_ROOT/test/integration/reports/grpc_agent_ops_test_report.md"
mkdir -p "$(dirname "$REPORT_FILE")"

cat > "$REPORT_FILE" << EOF
# Daemon gRPC Agent操作测试报告

**测试时间**: $(date '+%Y-%m-%d %H:%M:%S')
**测试类型**: gRPC接口完整测试

## 测试结果

\`\`\`
$TEST_OUTPUT
\`\`\`

## 关键指标

- **ListAgents响应时间**: $(echo "$TEST_OUTPUT" | grep "List agents succeeded" | grep -oP 'took \K[0-9.]+ms' || echo "N/A")
- **Stop操作**: $(echo "$TEST_OUTPUT" | grep -q "Stop result: success=true" && echo "✅ 成功" || echo "❌ 失败")
- **Start操作**: $(echo "$TEST_OUTPUT" | grep -q "Start result: success=true" && echo "✅ 成功" || echo "❌ 失败")
- **Restart操作**: $(echo "$TEST_OUTPUT" | grep -q "Restart result: success=true" && echo "✅ 成功" || echo "❌ 失败")
- **运行中的Agent进程**: $AGENT_COUNT
- **Daemon错误日志数**: $ERROR_COUNT

## Daemon日志摘要

\`\`\`
$(cat "$DAEMON_LOG" | strings | tail -50)
\`\`\`

EOF

echo -e "${GREEN}✓ 测试报告已生成: $REPORT_FILE${NC}\n"

# 总结
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}测试完成${NC}"
echo -e "${BLUE}========================================${NC}\n"

# 判断整体结果
if echo "$TEST_OUTPUT" | grep -q "Stop result: success=true" && \
   echo "$TEST_OUTPUT" | grep -q "Start result: success=true" && \
   echo "$TEST_OUTPUT" | grep -q "Restart result: success=true"; then
    echo -e "${GREEN}✅ 所有gRPC Agent操作测试通过！${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ 部分操作可能存在问题，请查看详细日志${NC}"
    exit 1
fi
