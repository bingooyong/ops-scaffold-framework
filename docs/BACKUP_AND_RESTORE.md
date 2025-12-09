# 备份和恢复指南

本文档说明如何备份和恢复 Ops Scaffold Framework 的数据和配置。

## 📋 备份内容

### 1. 配置文件

- `/etc/ops-scaffold/manager.yaml` - Manager 配置
- `/etc/ops-scaffold/daemon.yaml` - Daemon 配置
- `/etc/systemd/system/ops-manager.service` - Manager systemd 服务
- `/etc/systemd/system/ops-daemon.service` - Daemon systemd 服务
- `/etc/nginx/sites-available/ops-scaffold` - Nginx 配置（如果使用）

### 2. 数据库

- MySQL 数据库 `ops_scaffold`（包含所有业务数据）

### 3. 二进制文件

- `/usr/local/bin/ops-manager` - Manager 二进制
- `/usr/local/bin/ops-daemon` - Daemon 二进制

### 4. 日志文件

- `/var/log/ops-scaffold/manager/` - Manager 日志
- `/var/log/ops-scaffold/daemon/` - Daemon 日志

### 5. 工作目录

- `/var/lib/ops-scaffold/daemon/` - Daemon 工作目录（Agent 状态、临时文件等）

## 🚀 备份方法

### 方法 1: 使用备份脚本（推荐）

#### 完整备份

```bash
# 执行完整备份（包括所有内容）
sudo ./scripts/backup.sh full
```

#### 仅备份配置

```bash
# 仅备份配置文件
sudo ./scripts/backup.sh config
```

#### 仅备份数据库

```bash
# 仅备份数据库（需要设置 MySQL 密码）
sudo MYSQL_PASSWORD=your_password ./scripts/backup.sh database
```

备份文件将保存在 `/var/backups/ops-scaffold/YYYYMMDD_HHMMSS/` 目录。

### 方法 2: 手动备份

#### 2.1 备份配置文件

```bash
BACKUP_DIR="/var/backups/ops-scaffold/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# 备份配置文件
cp /etc/ops-scaffold/manager.yaml "$BACKUP_DIR/"
cp /etc/ops-scaffold/daemon.yaml "$BACKUP_DIR/"
cp /etc/systemd/system/ops-manager.service "$BACKUP_DIR/" 2>/dev/null || true
cp /etc/systemd/system/ops-daemon.service "$BACKUP_DIR/" 2>/dev/null || true
```

#### 2.2 备份数据库

```bash
# 备份数据库
mysqldump -u root -p ops_scaffold > "$BACKUP_DIR/database.sql"

# 或使用压缩
mysqldump -u root -p ops_scaffold | gzip > "$BACKUP_DIR/database.sql.gz"
```

#### 2.3 备份二进制文件

```bash
# 备份二进制文件
cp /usr/local/bin/ops-manager "$BACKUP_DIR/ops-manager"
cp /usr/local/bin/ops-daemon "$BACKUP_DIR/ops-daemon"
```

#### 2.4 备份日志和工作目录

```bash
# 备份日志（压缩）
tar -czf "$BACKUP_DIR/logs.tar.gz" /var/log/ops-scaffold/

# 备份工作目录（压缩）
tar -czf "$BACKUP_DIR/workdir.tar.gz" /var/lib/ops-scaffold/
```

## 📅 定期备份

### 使用 cron 定时备份

#### 1. 创建备份脚本

```bash
sudo tee /usr/local/bin/ops-backup-daily.sh > /dev/null <<'EOF'
#!/bin/bash
BACKUP_SCRIPT="/path/to/ops-scaffold-framework/scripts/backup.sh"
cd "$(dirname "$BACKUP_SCRIPT")"
sudo MYSQL_PASSWORD="${MYSQL_PASSWORD}" "$BACKUP_SCRIPT" full
EOF

sudo chmod +x /usr/local/bin/ops-backup-daily.sh
```

#### 2. 配置 cron 任务

```bash
# 编辑 crontab
sudo crontab -e

# 添加以下行（每天凌晨 2 点执行备份）
0 2 * * * /usr/local/bin/ops-backup-daily.sh >> /var/log/ops-backup.log 2>&1
```

### 备份保留策略

备份脚本会自动清理 7 天前的备份。如需修改保留策略，编辑 `scripts/backup.sh` 中的清理逻辑：

```bash
# 修改保留天数（例如保留 30 天）
find "${BACKUP_BASE_DIR}" -maxdepth 1 -type d -mtime +30 -exec rm -rf {} \;
```

## 🔄 恢复方法

### 恢复配置文件

```bash
BACKUP_DIR="/var/backups/ops-scaffold/YYYYMMDD_HHMMSS"

# 停止服务
sudo systemctl stop ops-manager
sudo systemctl stop ops-daemon

# 恢复配置文件
sudo cp "$BACKUP_DIR/manager.yaml" /etc/ops-scaffold/manager.yaml
sudo cp "$BACKUP_DIR/daemon.yaml" /etc/ops-scaffold/daemon.yaml

# 恢复 systemd 服务文件
sudo cp "$BACKUP_DIR/ops-manager.service" /etc/systemd/system/ops-manager.service
sudo cp "$BACKUP_DIR/ops-daemon.service" /etc/systemd/system/ops-daemon.service

# 重载 systemd
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start ops-manager
sudo systemctl start ops-daemon
```

### 恢复数据库

#### 方法 1: 使用 mysqldump 备份文件

```bash
BACKUP_DIR="/var/backups/ops-scaffold/YYYYMMDD_HHMMSS"

# 停止服务（可选，但推荐）
sudo systemctl stop ops-manager

# 恢复数据库
mysql -u root -p ops_scaffold < "$BACKUP_DIR/database.sql"

# 如果备份是压缩的
gunzip < "$BACKUP_DIR/database.sql.gz" | mysql -u root -p ops_scaffold

# 启动服务
sudo systemctl start ops-manager
```

#### 方法 2: 使用二进制日志（如果启用）

如果启用了 MySQL 二进制日志，可以使用 point-in-time recovery:

```bash
# 1. 恢复完整备份
mysql -u root -p ops_scaffold < /var/backups/ops-scaffold/YYYYMMDD_HHMMSS/database.sql

# 2. 应用二进制日志到指定时间点
mysqlbinlog --stop-datetime="2025-12-07 10:00:00" /var/log/mysql/mysql-bin.* | mysql -u root -p ops_scaffold
```

### 恢复二进制文件

```bash
BACKUP_DIR="/var/backups/ops-scaffold/YYYYMMDD_HHMMSS"

# 停止服务
sudo systemctl stop ops-manager
sudo systemctl stop ops-daemon

# 恢复二进制文件
sudo cp "$BACKUP_DIR/ops-manager" /usr/local/bin/ops-manager
sudo cp "$BACKUP_DIR/ops-daemon" /usr/local/bin/ops-daemon
sudo chmod +x /usr/local/bin/ops-manager
sudo chmod +x /usr/local/bin/ops-daemon

# 启动服务
sudo systemctl start ops-manager
sudo systemctl start ops-daemon
```

### 恢复日志和工作目录

```bash
BACKUP_DIR="/var/backups/ops-scaffold/YYYYMMDD_HHMMSS"

# 恢复日志（可选）
sudo tar -xzf "$BACKUP_DIR/logs.tar.gz" -C /

# 恢复工作目录
sudo systemctl stop ops-daemon
sudo tar -xzf "$BACKUP_DIR/workdir.tar.gz" -C /
sudo systemctl start ops-daemon
```

## ✅ 备份验证

### 验证备份完整性

```bash
BACKUP_DIR="/var/backups/ops-scaffold/YYYYMMDD_HHMMSS"

# 检查备份文件
ls -lh "$BACKUP_DIR"

# 验证数据库备份
head -20 "$BACKUP_DIR/database.sql"

# 验证压缩文件
tar -tzf "$BACKUP_DIR/logs.tar.gz" | head -10
```

### 测试恢复

在测试环境中测试恢复流程：

```bash
# 1. 在测试环境恢复数据库
mysql -u root -p ops_scaffold_test < "$BACKUP_DIR/database.sql"

# 2. 验证数据完整性
mysql -u root -p ops_scaffold_test -e "SELECT COUNT(*) FROM nodes;"
mysql -u root -p ops_scaffold_test -e "SELECT COUNT(*) FROM agents;"
```

## 🔒 备份安全

### 1. 加密备份

```bash
# 使用 GPG 加密备份
tar -czf - /var/backups/ops-scaffold/YYYYMMDD_HHMMSS | \
  gpg --symmetric --cipher-algo AES256 \
  --output /var/backups/ops-scaffold/YYYYMMDD_HHMMSS.tar.gz.gpg
```

### 2. 远程备份

#### 使用 rsync

```bash
# 同步到远程服务器
rsync -avz /var/backups/ops-scaffold/ user@backup-server:/backups/ops-scaffold/
```

#### 使用 S3 或其他对象存储

```bash
# 使用 AWS CLI 上传到 S3
aws s3 sync /var/backups/ops-scaffold/ s3://your-bucket/ops-scaffold-backups/
```

### 3. 备份权限

```bash
# 设置备份目录权限
sudo chmod 700 /var/backups/ops-scaffold
sudo chown root:root /var/backups/ops-scaffold
```

## 📊 备份监控

### 检查备份状态

```bash
# 检查最近的备份
ls -lht /var/backups/ops-scaffold/ | head -10

# 检查备份大小
du -sh /var/backups/ops-scaffold/*

# 检查备份日志
tail -f /var/log/ops-backup.log
```

### 备份告警

创建监控脚本检查备份是否成功：

```bash
#!/bin/bash
# check-backup.sh

LAST_BACKUP=$(find /var/backups/ops-scaffold -maxdepth 1 -type d -mtime -1 | head -1)

if [ -z "$LAST_BACKUP" ]; then
    echo "警告: 24 小时内未发现备份"
    # 发送告警（邮件、Webhook 等）
    exit 1
fi

echo "最新备份: $LAST_BACKUP"
exit 0
```

## 📚 相关文档

- [回滚指南](ROLLBACK.md)
- [部署指南](DEPLOYMENT.md)
- [Agent 管理管理员手册](Agent管理管理员手册.md)
