#!/bin/bash
# 运行多Agent管理架构的集成测试

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}多Agent管理架构集成测试${NC}"
echo -e "${GREEN}========================================${NC}"

cd "$(dirname "$0")/.."

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
            echo "Unknown option: $1"
            echo "Usage: $0 [-grpc] [-e2e]"
            exit 1
            ;;
    esac
done

echo ""
echo -e "${YELLOW}[1/5] 运行单元测试...${NC}"
go test -v ./internal/agent -run "TestNewAgentRegistry|TestAgentRegistry_|TestNewAgentInstance|TestAgentInstance_|TestNewMultiAgentManager|TestMultiAgentManager_" 2>&1 | tee /tmp/unit_tests.log

echo ""
echo -e "${YELLOW}[2/5] 运行Agent集成测试...${NC}"
go test -v ./internal/agent -run "TestMultiAgentManagement_EndToEnd|TestMultiAgentHealthCheck_EndToEnd|TestConfigLoading_EndToEnd|TestMultiAgentRestartStrategy_EndToEnd|TestConcurrentAgentOperations|TestLegacyConfigCompatibility" 2>&1 | tee /tmp/integration_tests.log

if [ "$RUN_GRPC" = true ]; then
    echo ""
    echo -e "${YELLOW}[3/5] 运行gRPC服务端测试...${NC}"
    echo -e "${YELLOW}注意: 使用 grpc_test 构建标签避免 protobuf 命名空间冲突${NC}"
    go test -v -tags=grpc_test ./internal/grpc/... 2>&1 | tee /tmp/grpc_server_tests.log
fi

if [ "$RUN_E2E" = true ]; then
    echo ""
    echo -e "${YELLOW}[4/5] 运行gRPC端到端集成测试...${NC}"
    echo -e "${YELLOW}注意: 端到端测试需要构建标签 e2e${NC}"
    go test -v -tags=e2e ./test/integration/grpc_integration_test.go ./test/integration/grpc_test_helpers.go 2>&1 | tee /tmp/grpc_e2e_tests.log
else
    echo ""
    echo -e "${YELLOW}[3/5] 跳过gRPC端到端测试 (使用 -e2e 参数运行)${NC}"
fi

echo ""
echo -e "${YELLOW}[4/5] 生成测试覆盖率报告...${NC}"
go test -coverprofile=coverage.out ./internal/agent/...
if [ "$RUN_GRPC" = true ]; then
    go test -coverprofile=grpc_coverage.out ./internal/grpc/...
    echo "gRPC测试覆盖率:"
    go tool cover -func=grpc_coverage.out | tail -1
fi
echo "Agent测试覆盖率:"
go tool cover -func=coverage.out | tail -1

echo ""
echo -e "${YELLOW}[5/5] 生成HTML覆盖率报告...${NC}"
go tool cover -html=coverage.out -o coverage.html
echo "覆盖率报告已生成: coverage.html"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}测试完成${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "提示:"
echo "  - 运行gRPC测试: $0 -grpc"
echo "  - 运行端到端测试: $0 -e2e"
echo "  - 运行所有测试: $0 -grpc -e2e"
