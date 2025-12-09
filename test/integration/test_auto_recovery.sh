#!/bin/bash
# 测试节点和Agent数据自动恢复功能
# 当数据库中的 nodes 或 agents 表数据被删除后，系统会在下一个心跳周期自动恢复

set -e

# 数据库配置
DB_HOST="127.0.0.1"
DB_PORT="3306"
DB_USER="root"
DB_PASS="rootpassword"
DB_NAME="ops_manager_dev"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== 节点和Agent数据自动恢复测试 ===${NC}\n"

# 1. 查看当前数据
echo -e "${GREEN}步骤 1: 查看当前数据${NC}"
echo "Nodes 表："
mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -e "SELECT node_id, hostname, status, last_seen_at FROM nodes;" 2>/dev/null | grep -v "Warning"
echo ""
echo "Agents 表："
mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -e "SELECT node_id, agent_id, type, status FROM agents ORDER BY agent_id;" 2>/dev/null | grep -v "Warning"
echo ""

# 2. 删除所有数据
echo -e "${YELLOW}步骤 2: 删除所有节点和Agent数据${NC}"
mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -e "DELETE FROM agents; DELETE FROM nodes;" 2>/dev/null
echo "数据已清空"
echo ""

# 3. 确认数据已删除
echo -e "${GREEN}步骤 3: 确认数据已删除${NC}"
NODE_COUNT=$(mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -se "SELECT COUNT(*) FROM nodes;" 2>/dev/null)
AGENT_COUNT=$(mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -se "SELECT COUNT(*) FROM agents;" 2>/dev/null)
echo "Nodes 表记录数: ${NODE_COUNT}"
echo "Agents 表记录数: ${AGENT_COUNT}"
echo ""

# 4. 等待心跳周期（30秒 + 5秒缓冲）
echo -e "${YELLOW}步骤 4: 等待心跳周期（35秒）...${NC}"
for i in {35..1}; do
    echo -ne "\r剩余 ${i} 秒...  "
    sleep 1
done
echo ""
echo ""

# 5. 检查数据是否恢复
echo -e "${GREEN}步骤 5: 检查数据是否自动恢复${NC}"
NODE_COUNT=$(mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -se "SELECT COUNT(*) FROM nodes;" 2>/dev/null)
AGENT_COUNT=$(mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -se "SELECT COUNT(*) FROM agents;" 2>/dev/null)

echo "Nodes 表记录数: ${NODE_COUNT}"
echo "Agents 表记录数: ${AGENT_COUNT}"
echo ""

if [ "${NODE_COUNT}" -gt 0 ] && [ "${AGENT_COUNT}" -gt 0 ]; then
    echo -e "${GREEN}✓ 测试通过！数据已自动恢复${NC}"
    echo ""
    echo "恢复的节点数据："
    mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -e "SELECT node_id, hostname, status, last_seen_at FROM nodes;" 2>/dev/null | grep -v "Warning"
    echo ""
    echo "恢复的Agent数据："
    mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_NAME} -e "SELECT node_id, agent_id, type, status FROM agents ORDER BY agent_id;" 2>/dev/null | grep -v "Warning"
    exit 0
else
    echo -e "${RED}✗ 测试失败！数据未能自动恢复${NC}"
    echo "请检查："
    echo "  1. Daemon 是否正在运行"
    echo "  2. Manager 是否正在运行"
    echo "  3. Daemon 配置中的 heartbeat_interval 是否正确"
    echo "  4. 查看 Manager 日志: tail -f test/integration/logs/manager.log"
    exit 1
fi

