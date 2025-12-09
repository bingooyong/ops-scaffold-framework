#!/bin/bash

# 冒烟测试脚本
# 用法: ./scripts/smoke-test.sh [manager_url] [token]
# 示例: ./scripts/smoke-test.sh http://localhost:8080 <token>

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

MANAGER_URL=${1:-"http://localhost:8080"}
TOKEN=${2:-""}

echo -e "${GREEN}开始执行冒烟测试...${NC}"
echo -e "${YELLOW}Manager URL: ${MANAGER_URL}${NC}"
echo ""

# 测试计数器
PASSED=0
FAILED=0

# 测试函数
test_case() {
    local name=$1
    local command=$2
    local expected=$3
    
    echo -e "${YELLOW}测试: ${name}${NC}"
    result=$(eval "$command" 2>&1) || true
    
    if echo "$result" | grep -q "$expected"; then
        echo -e "${GREEN}✓ 通过${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ 失败${NC}"
        echo "  输出: $result"
        ((FAILED++))
        return 1
    fi
}

# 测试场景 1: 健康检查
echo -e "${GREEN}=== 测试场景 1: 健康检查 ===${NC}"
test_case "Manager 健康检查" \
    "curl -s ${MANAGER_URL}/api/v1/health" \
    "healthy"

# 测试场景 2: 用户认证（如果提供了 token）
if [ -n "$TOKEN" ]; then
    echo ""
    echo -e "${GREEN}=== 测试场景 2: 用户认证 ===${NC}"
    test_case "验证 Token 有效性" \
        "curl -s -H \"Authorization: Bearer ${TOKEN}\" ${MANAGER_URL}/api/v1/nodes" \
        "code"
    
    # 测试获取节点列表
    test_case "获取节点列表" \
        "curl -s -H \"Authorization: Bearer ${TOKEN}\" ${MANAGER_URL}/api/v1/nodes | grep -q 'data' || echo 'no data'" \
        "data"
else
    echo ""
    echo -e "${YELLOW}=== 测试场景 2: 用户认证（跳过，未提供 Token） ===${NC}"
    echo -e "${YELLOW}提示: 提供 Token 参数以测试认证功能${NC}"
    echo -e "${YELLOW}示例: ./scripts/smoke-test.sh ${MANAGER_URL} <your-token>${NC}"
fi

# 测试场景 3: 节点注册（需要 Daemon 已启动并注册）
if [ -n "$TOKEN" ]; then
    echo ""
    echo -e "${GREEN}=== 测试场景 3: 节点注册 ===${NC}"
    nodes_result=$(curl -s -H "Authorization: Bearer ${TOKEN}" "${MANAGER_URL}/api/v1/nodes")
    if echo "$nodes_result" | grep -q "node_id"; then
        echo -e "${GREEN}✓ 节点已注册${NC}"
        ((PASSED++))
        
        # 检查节点状态
        test_case "节点状态为 online" \
            "echo '$nodes_result' | grep -q 'online' || echo 'not online'" \
            "online"
    else
        echo -e "${YELLOW}⚠ 未发现已注册的节点（Daemon 可能未启动）${NC}"
    fi
fi

# 测试场景 4: Agent 管理（需要节点和 Agent）
if [ -n "$TOKEN" ]; then
    echo ""
    echo -e "${GREEN}=== 测试场景 4: Agent 管理 ===${NC}"
    # 获取第一个节点 ID
    node_id=$(curl -s -H "Authorization: Bearer ${TOKEN}" "${MANAGER_URL}/api/v1/nodes" | grep -o '"node_id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [ -n "$node_id" ]; then
        echo -e "${YELLOW}测试节点: ${node_id}${NC}"
        
        # 获取 Agent 列表
        test_case "获取 Agent 列表" \
            "curl -s -H \"Authorization: Bearer ${TOKEN}\" ${MANAGER_URL}/api/v1/nodes/${node_id}/agents | grep -q 'data' || echo 'no data'" \
            "data"
    else
        echo -e "${YELLOW}⚠ 未找到节点，跳过 Agent 管理测试${NC}"
    fi
fi

# 测试场景 5: 监控功能
if [ -n "$TOKEN" ]; then
    echo ""
    echo -e "${GREEN}=== 测试场景 5: 监控功能 ===${NC}"
    if [ -n "$node_id" ]; then
        # 测试获取指标
        test_case "获取节点指标" \
            "curl -s -H \"Authorization: Bearer ${TOKEN}\" ${MANAGER_URL}/api/v1/nodes/${node_id}/metrics/latest | grep -q 'data' || echo 'no data'" \
            "data"
    else
        echo -e "${YELLOW}⚠ 未找到节点，跳过监控功能测试${NC}"
    fi
fi

# 总结
echo ""
echo -e "${GREEN}=== 测试总结 ===${NC}"
echo -e "${GREEN}通过: ${PASSED}${NC}"
echo -e "${RED}失败: ${FAILED}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}部分测试失败，请检查系统状态${NC}"
    exit 1
fi
