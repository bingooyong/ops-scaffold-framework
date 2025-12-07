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

# 解析参数
RUN_GRPC=false
RUN_E2E=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -grpc)
            RUN_GRPC=true
            shift
            ;;
        -e2e)
            RUN_E2E=true
            shift
            ;;
        *)
            # 忽略未知参数
            shift
            ;;
    esac
done

# 3. 运行集成测试
echo -e "${YELLOW}[3/5] 运行集成测试...${NC}"
cd /Users/bingooyong/Code/01Code/github.com/bingooyong/ops-scaffold-framework/manager/test/integration

# 运行 Manager 集成测试
echo "Running Manager integration tests..."
go test -v -timeout 5m -run TestManagerIntegration 2>&1 | tee -a /tmp/manager_test_output.log
MANAGER_TEST_RESULT=${PIPESTATUS[0]}

# 运行 Metrics API 集成测试
echo ""
echo "Running Metrics API tests..."
go test -v -timeout 5m -run TestMetricsAPI 2>&1 | tee -a /tmp/manager_test_output.log
METRICS_TEST_RESULT=${PIPESTATUS[0]}

# 运行 gRPC 客户端测试
if [ "$RUN_GRPC" = true ]; then
    echo ""
    echo -e "${YELLOW}[4/5] 运行gRPC客户端测试...${NC}"
    cd /Users/bingooyong/Code/01Code/github.com/bingooyong/ops-scaffold-framework/manager
    go test -v ./internal/grpc/... 2>&1 | tee -a /tmp/manager_test_output.log
    GRPC_TEST_RESULT=${PIPESTATUS[0]}
else
    echo ""
    echo -e "${YELLOW}[4/5] 跳过gRPC客户端测试 (使用 -grpc 参数运行)${NC}"
    GRPC_TEST_RESULT=0
fi

# 运行 gRPC 端到端测试
if [ "$RUN_E2E" = true ]; then
    echo ""
    echo -e "${YELLOW}[5/5] 运行gRPC端到端测试...${NC}"
    echo -e "${YELLOW}注意: 端到端测试需要构建标签 e2e${NC}"
    cd /Users/bingooyong/Code/01Code/github.com/bingooyong/ops-scaffold-framework/manager
    go test -v -tags=e2e ./internal/grpc/daemon_client_e2e_test.go 2>&1 | tee -a /tmp/manager_test_output.log
    E2E_TEST_RESULT=${PIPESTATUS[0]}
else
    echo ""
    echo -e "${YELLOW}[5/5] 跳过gRPC端到端测试 (使用 -e2e 参数运行)${NC}"
    E2E_TEST_RESULT=0
fi

# 合并测试结果
if [ $MANAGER_TEST_RESULT -eq 0 ] && [ $METRICS_TEST_RESULT -eq 0 ] && [ $GRPC_TEST_RESULT -eq 0 ] && [ $E2E_TEST_RESULT -eq 0 ]; then
	TEST_RESULT=0
else
	TEST_RESULT=1
fi
echo ""

# 6. 统计测试结果
echo -e "${YELLOW}[6/6] 统计测试结果...${NC}"
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

echo ""
echo "提示:"
echo "  - 运行gRPC测试: $0 -grpc"
echo "  - 运行端到端测试: $0 -e2e"
echo "  - 运行所有测试: $0 -grpc -e2e"
