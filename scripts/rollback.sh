#!/bin/bash

# 回滚脚本
# 用法: ./scripts/rollback.sh [target_version]
# 示例: ./scripts/rollback.sh v0.3.0

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

TARGET_VERSION=${1:-"v0.3.0"}
BACKUP_DIR="/var/backups/ops-scaffold"
RELEASE_DIR="releases/${TARGET_VERSION}"

echo -e "${GREEN}开始回滚到版本: ${TARGET_VERSION}${NC}"

# 检查是否在项目根目录
if [ ! -f "README.md" ]; then
    echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 检查目标版本是否存在
if [ ! -d "$RELEASE_DIR" ]; then
    echo -e "${RED}错误: 目标版本 ${TARGET_VERSION} 的发布包不存在${NC}"
    echo -e "${YELLOW}请确保 releases/${TARGET_VERSION} 目录存在${NC}"
    exit 1
fi

# 创建备份目录
echo -e "${YELLOW}创建备份目录...${NC}"
sudo mkdir -p "${BACKUP_DIR}/$(date +%Y%m%d_%H%M%S)"
BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
CURRENT_BACKUP_DIR="${BACKUP_DIR}/${BACKUP_TIMESTAMP}"
sudo mkdir -p "${CURRENT_BACKUP_DIR}"

# 1. 停止当前版本服务
echo -e "${YELLOW}停止当前版本服务...${NC}"
sudo systemctl stop ops-manager 2>/dev/null || echo "Manager 服务未运行"
sudo systemctl stop ops-daemon 2>/dev/null || echo "Daemon 服务未运行"

# 2. 备份当前版本文件
echo -e "${YELLOW}备份当前版本文件...${NC}"

# 备份 Manager
if [ -f "/usr/local/bin/ops-manager" ]; then
    echo "备份 Manager 二进制文件..."
    sudo cp /usr/local/bin/ops-manager "${CURRENT_BACKUP_DIR}/ops-manager"
fi

if [ -f "/etc/ops-scaffold/manager.yaml" ]; then
    echo "备份 Manager 配置文件..."
    sudo cp /etc/ops-scaffold/manager.yaml "${CURRENT_BACKUP_DIR}/manager.yaml"
fi

# 备份 Daemon
if [ -f "/usr/local/bin/ops-daemon" ]; then
    echo "备份 Daemon 二进制文件..."
    sudo cp /usr/local/bin/ops-daemon "${CURRENT_BACKUP_DIR}/ops-daemon"
fi

if [ -f "/etc/ops-scaffold/daemon.yaml" ]; then
    echo "备份 Daemon 配置文件..."
    sudo cp /etc/ops-scaffold/daemon.yaml "${CURRENT_BACKUP_DIR}/daemon.yaml"
fi

# 备份 Web 前端
if [ -d "/var/www/ops-scaffold" ]; then
    echo "备份 Web 前端文件..."
    sudo tar -czf "${CURRENT_BACKUP_DIR}/web-frontend.tar.gz" -C /var/www ops-scaffold
fi

# 备份数据库（如果可能）
echo -e "${YELLOW}备份数据库...${NC}"
if command -v mysql &> /dev/null; then
    DB_NAME=$(grep -A 5 "^database:" /etc/ops-scaffold/manager.yaml 2>/dev/null | grep "database:" | awk '{print $2}' | tr -d '"' || echo "ops_scaffold")
    if [ -n "$DB_NAME" ] && [ "$DB_NAME" != "null" ]; then
        echo "备份数据库: ${DB_NAME}"
        sudo mysqldump -u root -p"${MYSQL_ROOT_PASSWORD:-}" "${DB_NAME}" > "${CURRENT_BACKUP_DIR}/database.sql" 2>/dev/null || \
        echo -e "${YELLOW}警告: 无法自动备份数据库，请手动备份${NC}"
    fi
fi

echo -e "${GREEN}备份完成: ${CURRENT_BACKUP_DIR}${NC}"

# 3. 恢复上一版本文件
echo -e "${YELLOW}恢复版本 ${TARGET_VERSION} 文件...${NC}"

# 恢复 Manager
if [ -f "${RELEASE_DIR}/manager/manager-linux-amd64" ]; then
    echo "恢复 Manager 二进制文件..."
    sudo cp "${RELEASE_DIR}/manager/manager-linux-amd64" /usr/local/bin/ops-manager
    sudo chmod +x /usr/local/bin/ops-manager
fi

# 恢复 Daemon
if [ -f "${RELEASE_DIR}/daemon/daemon-linux-amd64" ]; then
    echo "恢复 Daemon 二进制文件..."
    sudo cp "${RELEASE_DIR}/daemon/daemon-linux-amd64" /usr/local/bin/ops-daemon
    sudo chmod +x /usr/local/bin/ops-daemon
fi

# 恢复 Web 前端
if [ -d "${RELEASE_DIR}/web/dist" ]; then
    echo "恢复 Web 前端文件..."
    sudo rm -rf /var/www/ops-scaffold/*
    sudo cp -r "${RELEASE_DIR}/web/dist"/* /var/www/ops-scaffold/
    sudo chown -R www-data:www-data /var/www/ops-scaffold
fi

# 4. 启动上一版本服务
echo -e "${YELLOW}启动服务...${NC}"

# 启动 Manager
echo "启动 Manager..."
sudo systemctl daemon-reload
sudo systemctl start ops-manager
sleep 2

# 检查 Manager 状态
if sudo systemctl is-active --quiet ops-manager; then
    echo -e "${GREEN}Manager 启动成功${NC}"
else
    echo -e "${RED}Manager 启动失败，请检查日志: sudo journalctl -u ops-manager${NC}"
    exit 1
fi

# 启动 Daemon
echo "启动 Daemon..."
sudo systemctl start ops-daemon
sleep 2

# 检查 Daemon 状态
if sudo systemctl is-active --quiet ops-daemon; then
    echo -e "${GREEN}Daemon 启动成功${NC}"
else
    echo -e "${RED}Daemon 启动失败，请检查日志: sudo journalctl -u ops-daemon${NC}"
    exit 1
fi

# 5. 验证服务正常
echo -e "${YELLOW}验证服务...${NC}"

# 等待服务就绪
sleep 5

# 检查 Manager 健康状态
echo "检查 Manager 健康状态..."
if curl -s http://localhost:8080/api/v1/health | grep -q "healthy"; then
    echo -e "${GREEN}Manager 健康检查通过${NC}"
else
    echo -e "${RED}Manager 健康检查失败${NC}"
    exit 1
fi

# 检查 Daemon 日志
echo "检查 Daemon 连接状态..."
if sudo journalctl -u ops-daemon --since "1 minute ago" | grep -q "connected\|registered"; then
    echo -e "${GREEN}Daemon 连接正常${NC}"
else
    echo -e "${YELLOW}警告: Daemon 可能未连接到 Manager，请检查日志${NC}"
fi

# 总结
echo ""
echo -e "${GREEN}=== 回滚完成 ===${NC}"
echo -e "${GREEN}当前版本: ${TARGET_VERSION}${NC}"
echo -e "${GREEN}备份位置: ${CURRENT_BACKUP_DIR}${NC}"
echo ""
echo -e "${YELLOW}下一步:${NC}"
echo "1. 运行冒烟测试验证功能: ./scripts/smoke-test.sh"
echo "2. 检查服务日志: sudo journalctl -u ops-manager -u ops-daemon"
echo "3. 如果回滚失败，可以从备份恢复: ${CURRENT_BACKUP_DIR}"
