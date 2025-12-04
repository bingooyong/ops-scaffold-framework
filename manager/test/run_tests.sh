#!/bin/bash

# Manager 自动化测试脚本
# 用法: ./run_tests.sh

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Manager 自动化测试${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 1. 检查Manager服务是否运行
echo -e "${YELLOW}[1/4] 检查Manager服务...${NC}"
if curl -s http://127.0.0.1:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Manager服务运行正常${NC}"
else
    echo -e "${RED}❌ Manager服务未运行${NC}"
    echo -e "${YELLOW}请先启动Manager:${NC}"
    echo -e "  cd manager"
    echo -e "  ./bin/manager -config configs/manager.dev.yaml"
    exit 1
fi
echo ""

# 2. 检查数据库连接
echo -e "${YELLOW}[2/4] 检查数据库连接...${NC}"
if mysql -h 127.0.0.1 -P 3306 -uroot -prootpassword -e "USE ops_manager_dev;" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 数据库连接正常${NC}"
else
    echo -e "${RED}❌ 数据库连接失败${NC}"
    echo -e "${YELLOW}请检查MySQL是否运行${NC}"
    exit 1
fi
echo ""

# 3. 运行集成测试
echo -e "${YELLOW}[3/4] 运行集成测试...${NC}"
cd /Users/bingooyong/Code/01Code/github.com/bingooyong/ops-scaffold-framework/manager/test/integration
go test -v -timeout 5m -run TestManagerIntegration 2>&1 | tee /tmp/manager_test_output.log
TEST_RESULT=${PIPESTATUS[0]}
echo ""

# 4. 统计测试结果
echo -e "${YELLOW}[4/4] 统计测试结果...${NC}"
TOTAL=$(grep -c "=== RUN" /tmp/manager_test_output.log || echo "0")
PASSED=$(grep -c "--- PASS" /tmp/manager_test_output.log || echo "0")
FAILED=$(grep -c "--- FAIL" /tmp/manager_test_output.log || echo "0")

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}测试结果摘要${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "总测试数: ${TOTAL}"
echo -e "${GREEN}通过: ${PASSED}${NC}"
if [ "$FAILED" -gt 0 ]; then
    echo -e "${RED}失败: ${FAILED}${NC}"
else
    echo -e "${GREEN}失败: ${FAILED}${NC}"
fi

if [ "$TEST_RESULT" -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ 所有测试通过!${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}❌ 部分测试失败${NC}"
    echo -e "${YELLOW}详细日志: /tmp/manager_test_output.log${NC}"
    exit 1
fi
