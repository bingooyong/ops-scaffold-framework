# 回滚操作指南

本文档说明如何在出现问题时将 Ops Scaffold Framework 回滚到上一版本。

## 📋 回滚前提条件

### 1. 备份要求

在回滚之前，**必须**先备份当前版本：

- ✅ 当前版本二进制文件
- ✅ 当前版本配置文件
- ✅ 数据库数据
- ✅ 日志文件（可选）

### 2. 目标版本要求

- ✅ 目标版本的发布包必须存在于 `releases/` 目录
- ✅ 目标版本的配置文件格式兼容（如果配置格式有变化）

### 3. 权限要求

回滚操作需要 **root 权限**，确保有足够的权限执行以下操作：

- 停止/启动 systemd 服务
- 复制二进制文件和配置文件
- 备份和恢复文件

## 🚀 回滚步骤

### 方法 1: 使用回滚脚本（推荐）

#### 1.1 执行备份

```bash
# 执行完整备份
sudo ./scripts/backup.sh full

# 或仅备份配置和数据库
sudo ./scripts/backup.sh config
sudo MYSQL_PASSWORD=your_password ./scripts/backup.sh database
```

#### 1.2 执行回滚

```bash
# 回滚到指定版本（例如 v0.3.0）
sudo ./scripts/rollback.sh v0.3.0
```

脚本会自动执行以下操作：

1. 停止当前版本服务
2. 备份当前版本文件
3. 恢复目标版本文件
4. 启动目标版本服务
5. 验证服务正常

#### 1.3 验证回滚

```bash
# 运行冒烟测试
./scripts/smoke-test.sh http://localhost:8080 <token>

# 检查服务状态
sudo systemctl status ops-manager
sudo systemctl status ops-daemon

# 检查健康状态
curl http://localhost:8080/api/v1/health
```

### 方法 2: 手动回滚

如果自动回滚脚本失败，可以手动执行以下步骤：

#### 2.1 停止服务

```bash
sudo systemctl stop ops-manager
sudo systemctl stop ops-daemon
```

#### 2.2 备份当前版本

```bash
# 创建备份目录
BACKUP_DIR="/var/backups/ops-scaffold/$(date +%Y%m%d_%H%M%S)"
sudo mkdir -p "$BACKUP_DIR"

# 备份二进制文件
sudo cp /usr/local/bin/ops-manager "$BACKUP_DIR/ops-manager"
sudo cp /usr/local/bin/ops-daemon "$BACKUP_DIR/ops-daemon"

# 备份配置文件
sudo cp /etc/ops-scaffold/manager.yaml "$BACKUP_DIR/manager.yaml"
sudo cp /etc/ops-scaffold/daemon.yaml "$BACKUP_DIR/daemon.yaml"

# 备份数据库
sudo mysqldump -u root -p ops_scaffold > "$BACKUP_DIR/database.sql"
```

#### 2.3 恢复目标版本

```bash
# 设置目标版本
TARGET_VERSION="v0.3.0"
RELEASE_DIR="releases/${TARGET_VERSION}"

# 恢复 Manager
sudo cp "${RELEASE_DIR}/manager/manager-linux-amd64" /usr/local/bin/ops-manager
sudo chmod +x /usr/local/bin/ops-manager

# 恢复 Daemon
sudo cp "${RELEASE_DIR}/daemon/daemon-linux-amd64" /usr/local/bin/ops-daemon
sudo chmod +x /usr/local/bin/ops-daemon

# 恢复 Web 前端
sudo rm -rf /var/www/ops-scaffold/*
sudo cp -r "${RELEASE_DIR}/web/dist"/* /var/www/ops-scaffold/
sudo chown -R www-data:www-data /var/www/ops-scaffold
```

#### 2.4 启动服务

```bash
sudo systemctl daemon-reload
sudo systemctl start ops-manager
sudo systemctl start ops-daemon

# 验证服务状态
sudo systemctl status ops-manager
sudo systemctl status ops-daemon
```

## ✅ 回滚验证

### 1. 服务状态检查

```bash
# 检查服务是否运行
sudo systemctl is-active ops-manager
sudo systemctl is-active ops-daemon

# 检查服务日志
sudo journalctl -u ops-manager -n 50
sudo journalctl -u ops-daemon -n 50
```

### 2. 功能验证

#### 2.1 健康检查

```bash
curl http://localhost:8080/api/v1/health
# 预期输出: {"code":0,"message":"success","data":{"status":"healthy"}}
```

#### 2.2 用户认证

```bash
# 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123456"}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

# 验证 Token
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/nodes
```

#### 2.3 节点和 Agent 管理

```bash
# 查看节点列表
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/nodes

# 查看 Agent 列表（需要节点 ID）
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/nodes/{node_id}/agents
```

### 3. 运行冒烟测试

```bash
# 运行完整的冒烟测试
./scripts/smoke-test.sh http://localhost:8080 "$TOKEN"
```

## 📋 回滚后检查清单

- [ ] Manager 服务正常运行
- [ ] Daemon 服务正常运行
- [ ] Manager 健康检查通过
- [ ] Daemon 已连接到 Manager
- [ ] 节点已注册到 Manager
- [ ] 用户认证功能正常
- [ ] Agent 管理功能正常（如果适用）
- [ ] 监控功能正常
- [ ] Web 前端可正常访问
- [ ] 所有冒烟测试通过

## 🔧 常见问题处理

### 问题 1: 回滚后服务无法启动

**症状**: systemctl start 失败

**排查步骤**:

1. 检查服务日志:
   ```bash
   sudo journalctl -u ops-manager -n 100
   sudo journalctl -u ops-daemon -n 100
   ```

2. 检查配置文件:
   ```bash
   sudo cat /etc/ops-scaffold/manager.yaml
   sudo cat /etc/ops-scaffold/daemon.yaml
   ```

3. 检查二进制文件权限:
   ```bash
   ls -l /usr/local/bin/ops-manager
   ls -l /usr/local/bin/ops-daemon
   ```

4. 手动运行测试:
   ```bash
   sudo /usr/local/bin/ops-manager -config /etc/ops-scaffold/manager.yaml
   ```

### 问题 2: 数据库版本不兼容

**症状**: 启动后数据库错误

**解决方法**:

1. 恢复数据库备份:
   ```bash
   mysql -u root -p ops_scaffold < /var/backups/ops-scaffold/YYYYMMDD_HHMMSS/database.sql
   ```

2. 或执行数据库迁移回滚（如果支持）:
   ```bash
   cd manager
   make migrate-down
   ```

### 问题 3: 配置文件格式不兼容

**症状**: 启动时配置解析错误

**解决方法**:

1. 检查配置文件格式:
   ```bash
   # 查看配置文件示例
   cat releases/v0.3.0/manager/manager.yaml.example
   ```

2. 手动调整配置文件以匹配目标版本格式

3. 或从备份恢复配置文件:
   ```bash
   sudo cp /var/backups/ops-scaffold/YYYYMMDD_HHMMSS/manager.yaml /etc/ops-scaffold/manager.yaml
   ```

### 问题 4: Web 前端无法访问

**症状**: 浏览器无法加载页面

**排查步骤**:

1. 检查 Nginx 服务:
   ```bash
   sudo systemctl status nginx
   ```

2. 检查 Nginx 配置:
   ```bash
   sudo nginx -t
   ```

3. 检查静态文件:
   ```bash
   ls -la /var/www/ops-scaffold/
   ```

4. 检查 Nginx 日志:
   ```bash
   sudo tail -f /var/log/nginx/error.log
   ```

### 问题 5: 回滚失败需要恢复

如果回滚失败，可以从备份恢复:

```bash
BACKUP_DIR="/var/backups/ops-scaffold/YYYYMMDD_HHMMSS"

# 恢复二进制文件
sudo cp "$BACKUP_DIR/ops-manager" /usr/local/bin/ops-manager
sudo cp "$BACKUP_DIR/ops-daemon" /usr/local/bin/ops-daemon

# 恢复配置文件
sudo cp "$BACKUP_DIR/manager.yaml" /etc/ops-scaffold/manager.yaml
sudo cp "$BACKUP_DIR/daemon.yaml" /etc/ops-scaffold/daemon.yaml

# 恢复数据库
mysql -u root -p ops_scaffold < "$BACKUP_DIR/database.sql"

# 启动服务
sudo systemctl start ops-manager
sudo systemctl start ops-daemon
```

## 📝 回滚记录

建议记录每次回滚操作：

| 项目 | 值 |
|------|-----|
| 回滚时间 | [填写] |
| 回滚原因 | [填写] |
| 从版本 | [填写] |
| 回滚到版本 | [填写] |
| 回滚方式 | 自动/手动 |
| 回滚结果 | 成功/失败 |
| 备注 | [填写] |

## 🔒 安全注意事项

1. **备份敏感信息**: 回滚前确保备份包含所有敏感配置（密码、密钥等）
2. **权限控制**: 回滚脚本需要 root 权限，确保在安全环境中执行
3. **网络隔离**: 如果可能，在回滚期间临时隔离系统，避免影响生产环境
4. **通知相关人员**: 回滚前通知相关团队成员

## 📚 相关文档

- [部署指南](DEPLOYMENT.md)
- [备份和恢复指南](BACKUP_AND_RESTORE.md)
- [部署验证报告](DEPLOYMENT_VERIFICATION_REPORT.md)
- [CHANGELOG](../CHANGELOG.md)
