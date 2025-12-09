#!/bin/bash

# 备份脚本
# 用法: ./scripts/backup.sh [backup_type]
# 示例: ./scripts/backup.sh full
# backup_type: full (完整备份), config (仅配置), database (仅数据库)

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BACKUP_TYPE=${1:-"full"}
BACKUP_BASE_DIR="/var/backups/ops-scaffold"
BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="${BACKUP_BASE_DIR}/${BACKUP_DATE}"

echo -e "${GREEN}开始备份 (类型: ${BACKUP_TYPE})...${NC}"

# 检查是否为 root 用户
if [ "$(id -u)" != "0" ]; then
    echo -e "${RED}错误: 此脚本需要 root 权限${NC}"
    exit 1
fi

# 创建备份目录
mkdir -p "${BACKUP_DIR}"
cd "${BACKUP_DIR}"

# 完整备份
if [ "$BACKUP_TYPE" = "full" ] || [ "$BACKUP_TYPE" = "config" ]; then
    echo -e "${YELLOW}备份配置文件...${NC}"
    
    # Manager 配置
    if [ -f "/etc/ops-scaffold/manager.yaml" ]; then
        cp /etc/ops-scaffold/manager.yaml manager.yaml
        echo "✓ Manager 配置已备份"
    fi
    
    # Daemon 配置
    if [ -f "/etc/ops-scaffold/daemon.yaml" ]; then
        cp /etc/ops-scaffold/daemon.yaml daemon.yaml
        echo "✓ Daemon 配置已备份"
    fi
    
    # systemd 服务文件
    if [ -f "/etc/systemd/system/ops-manager.service" ]; then
        cp /etc/systemd/system/ops-manager.service ops-manager.service
        echo "✓ Manager systemd 服务文件已备份"
    fi
    
    if [ -f "/etc/systemd/system/ops-daemon.service" ]; then
        cp /etc/systemd/system/ops-daemon.service ops-daemon.service
        echo "✓ Daemon systemd 服务文件已备份"
    fi
    
    # Nginx 配置
    if [ -f "/etc/nginx/sites-available/ops-scaffold" ]; then
        cp /etc/nginx/sites-available/ops-scaffold nginx-ops-scaffold.conf
        echo "✓ Nginx 配置已备份"
    fi
fi

# 完整备份
if [ "$BACKUP_TYPE" = "full" ] || [ "$BACKUP_TYPE" = "database" ]; then
    echo -e "${YELLOW}备份数据库...${NC}"
    
    # 从配置文件读取数据库信息
    if [ -f "/etc/ops-scaffold/manager.yaml" ]; then
        DB_HOST=$(grep -A 5 "^database:" /etc/ops-scaffold/manager.yaml | grep "host:" | awk '{print $2}' | tr -d '"' || echo "localhost")
        DB_PORT=$(grep -A 5 "^database:" /etc/ops-scaffold/manager.yaml | grep "port:" | awk '{print $2}' | tr -d '"' || echo "3306")
        DB_NAME=$(grep -A 5 "^database:" /etc/ops-scaffold/manager.yaml | grep "database:" | awk '{print $2}' | tr -d '"' || echo "ops_scaffold")
        DB_USER=$(grep -A 5 "^database:" /etc/ops-scaffold/manager.yaml | grep "username:" | awk '{print $2}' | tr -d '"' || echo "root")
        
        if [ -n "$DB_NAME" ] && [ "$DB_NAME" != "null" ]; then
            echo "备份数据库: ${DB_NAME}"
            
            # 尝试使用配置文件中的用户名，如果没有密码则提示
            if [ -n "${MYSQL_PASSWORD}" ]; then
                mysqldump -h "${DB_HOST}" -P "${DB_PORT}" -u "${DB_USER}" -p"${MYSQL_PASSWORD}" "${DB_NAME}" > database.sql 2>/dev/null && \
                echo "✓ 数据库已备份" || echo -e "${YELLOW}警告: 数据库备份失败，请手动备份${NC}"
            else
                echo -e "${YELLOW}提示: 设置 MYSQL_PASSWORD 环境变量以自动备份数据库${NC}"
                echo -e "${YELLOW}或手动执行: mysqldump -u ${DB_USER} -p ${DB_NAME} > ${BACKUP_DIR}/database.sql${NC}"
            fi
        fi
    fi
fi

# 完整备份
if [ "$BACKUP_TYPE" = "full" ]; then
    echo -e "${YELLOW}备份二进制文件...${NC}"
    
    # Manager 二进制
    if [ -f "/usr/local/bin/ops-manager" ]; then
        cp /usr/local/bin/ops-manager ops-manager.bin
        echo "✓ Manager 二进制文件已备份"
    fi
    
    # Daemon 二进制
    if [ -f "/usr/local/bin/ops-daemon" ]; then
        cp /usr/local/bin/ops-daemon ops-daemon.bin
        echo "✓ Daemon 二进制文件已备份"
    fi
    
    echo -e "${YELLOW}备份日志文件...${NC}"
    
    # Manager 日志
    if [ -d "/var/log/ops-scaffold/manager" ]; then
        tar -czf manager-logs.tar.gz -C /var/log/ops-scaffold manager 2>/dev/null && \
        echo "✓ Manager 日志已备份" || echo -e "${YELLOW}警告: Manager 日志备份失败${NC}"
    fi
    
    # Daemon 日志
    if [ -d "/var/log/ops-scaffold/daemon" ]; then
        tar -czf daemon-logs.tar.gz -C /var/log/ops-scaffold daemon 2>/dev/null && \
        echo "✓ Daemon 日志已备份" || echo -e "${YELLOW}警告: Daemon 日志备份失败${NC}"
    fi
    
    echo -e "${YELLOW}备份工作目录...${NC}"
    
    # Daemon 工作目录
    if [ -d "/var/lib/ops-scaffold/daemon" ]; then
        tar -czf daemon-workdir.tar.gz -C /var/lib/ops-scaffold daemon 2>/dev/null && \
        echo "✓ Daemon 工作目录已备份" || echo -e "${YELLOW}警告: Daemon 工作目录备份失败${NC}"
    fi
fi

# 创建备份清单
echo -e "${YELLOW}创建备份清单...${NC}"
cat > backup-manifest.txt <<EOF
备份时间: $(date)
备份类型: ${BACKUP_TYPE}
备份目录: ${BACKUP_DIR}

备份内容:
$(ls -lh)

系统信息:
$(uname -a)

版本信息:
$(/usr/local/bin/ops-manager --version 2>/dev/null || echo "无法获取版本信息")
EOF

# 计算备份大小
BACKUP_SIZE=$(du -sh "${BACKUP_DIR}" | awk '{print $1}')

# 总结
echo ""
echo -e "${GREEN}=== 备份完成 ===${NC}"
echo -e "${GREEN}备份位置: ${BACKUP_DIR}${NC}"
echo -e "${GREEN}备份大小: ${BACKUP_SIZE}${NC}"
echo ""
echo -e "${YELLOW}备份清单:${NC}"
cat backup-manifest.txt

# 清理旧备份（保留最近 7 天）
echo ""
echo -e "${YELLOW}清理旧备份（保留最近 7 天）...${NC}"
find "${BACKUP_BASE_DIR}" -maxdepth 1 -type d -mtime +7 -exec rm -rf {} \; 2>/dev/null || true
echo "✓ 旧备份已清理"
