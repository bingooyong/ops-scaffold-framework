# 部署指南

本文档说明如何在生产环境部署 Ops Scaffold Framework v0.4.0。

## 📋 前置要求

### 系统要求

- **操作系统**: Linux (推荐 Ubuntu 20.04+, CentOS 7+, Debian 10+)
- **CPU**: 2 核以上
- **内存**: 4GB 以上
- **磁盘**: 20GB 以上可用空间

### 软件依赖

- **MySQL**: 8.0+ (用于 Manager 数据存储)
- **Nginx**: 1.18+ (用于 Web 前端部署，可选)
- **systemd**: 用于服务管理 (Linux)

### 网络要求

- Manager HTTP API 端口: 8080
- Manager gRPC 端口: 9090
- Web 前端端口: 80/443 (通过 Nginx)
- 确保 Manager 和 Daemon 之间网络互通

## 🚀 部署步骤

### 1. 准备发布包

从发布包目录 `releases/v0.4.0/` 获取以下文件：

- `manager/manager-linux-amd64` - Manager 二进制文件
- `daemon/daemon-linux-amd64` - Daemon 二进制文件
- `web/dist/` - Web 前端静态文件
- `manager/manager.yaml.example` - Manager 配置示例
- `daemon/daemon.yaml.example` - Daemon 配置示例

### 2. 部署 Manager

#### 2.1 安装二进制文件

```bash
# 复制二进制文件
sudo cp releases/v0.4.0/manager/manager-linux-amd64 /usr/local/bin/ops-manager
sudo chmod +x /usr/local/bin/ops-manager

# 创建配置目录
sudo mkdir -p /etc/ops-scaffold
sudo mkdir -p /var/log/ops-scaffold/manager
```

#### 2.2 配置数据库

```bash
# 创建数据库
mysql -u root -p <<EOF
CREATE DATABASE IF NOT EXISTS ops_scaffold CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'ops_user'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON ops_scaffold.* TO 'ops_user'@'localhost';
FLUSH PRIVILEGES;
EOF
```

#### 2.3 配置 Manager

```bash
# 复制配置文件
sudo cp releases/v0.4.0/manager/manager.yaml.example /etc/ops-scaffold/manager.yaml

# 编辑配置文件
sudo vim /etc/ops-scaffold/manager.yaml
```

关键配置项：

```yaml
server:
  port: 8080
  grpc_port: 9090

database:
  host: localhost
  port: 3306
  database: ops_scaffold
  username: ops_user
  password: your_password

jwt:
  secret: your_jwt_secret_key_here  # 请使用强密钥
  expire_hours: 24

log:
  level: info
  output: /var/log/ops-scaffold/manager/manager.log
```

#### 2.4 创建 systemd 服务

```bash
sudo tee /etc/systemd/system/ops-manager.service > /dev/null <<EOF
[Unit]
Description=Ops Scaffold Framework Manager
After=network.target mysql.service

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/ops-manager -config /etc/ops-scaffold/manager.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 重载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start ops-manager
sudo systemctl enable ops-manager

# 检查状态
sudo systemctl status ops-manager
```

#### 2.5 验证 Manager 部署

```bash
# 检查健康状态
curl http://localhost:8080/api/v1/health

# 预期输出: {"code":0,"message":"success","data":{"status":"healthy"}}
```

### 3. 部署 Daemon

#### 3.1 安装二进制文件

```bash
# 复制二进制文件
sudo cp releases/v0.4.0/daemon/daemon-linux-amd64 /usr/local/bin/ops-daemon
sudo chmod +x /usr/local/bin/ops-daemon

# 创建配置目录
sudo mkdir -p /etc/ops-scaffold
sudo mkdir -p /var/log/ops-scaffold/daemon
sudo mkdir -p /var/lib/ops-scaffold/daemon
```

#### 3.2 配置 Daemon

```bash
# 复制配置文件
sudo cp releases/v0.4.0/daemon/daemon.yaml.example /etc/ops-scaffold/daemon.yaml

# 编辑配置文件
sudo vim /etc/ops-scaffold/daemon.yaml
```

关键配置项：

```yaml
manager:
  address: "manager.example.com:9090"  # Manager 的 gRPC 地址
  heartbeat_interval: 60s

agents:
  - id: filebeat-logs
    type: filebeat
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat.yml
    enabled: true
    restart_policy:
      policy: always
      max_retries: 3

log:
  level: info
  output: /var/log/ops-scaffold/daemon/daemon.log
```

#### 3.3 创建 systemd 服务

```bash
sudo tee /etc/systemd/system/ops-daemon.service > /dev/null <<EOF
[Unit]
Description=Ops Scaffold Framework Daemon
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/ops-daemon -config /etc/ops-scaffold/daemon.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 重载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start ops-daemon
sudo systemctl enable ops-daemon

# 检查状态
sudo systemctl status ops-daemon
```

#### 3.4 验证 Daemon 部署

```bash
# 查看日志，确认已连接到 Manager
sudo journalctl -u ops-daemon -f

# 应该看到类似以下日志：
# INFO  daemon started successfully
# INFO  connected to manager at manager.example.com:9090
# INFO  node registered successfully
```

### 4. 部署 Web 前端

#### 4.1 安装 Nginx

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install nginx

# CentOS/RHEL
sudo yum install nginx
```

#### 4.2 配置 Nginx

```bash
# 复制静态文件
sudo mkdir -p /var/www/ops-scaffold
sudo cp -r releases/v0.4.0/web/dist/* /var/www/ops-scaffold/
sudo chown -R www-data:www-data /var/www/ops-scaffold

# 创建 Nginx 配置
sudo tee /etc/nginx/sites-available/ops-scaffold > /dev/null <<EOF
server {
    listen 80;
    server_name your-domain.com;  # 替换为您的域名

    root /var/www/ops-scaffold;
    index index.html;

    # 前端静态文件
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # API 反向代理
    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
EOF

# 启用站点
sudo ln -s /etc/nginx/sites-available/ops-scaffold /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重启 Nginx
sudo systemctl restart nginx
sudo systemctl enable nginx
```

#### 4.3 配置 HTTPS (可选，推荐)

使用 Let's Encrypt 配置 HTTPS：

```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo certbot renew --dry-run
```

### 5. 验证部署

#### 5.1 运行冒烟测试

```bash
# 注册用户并获取 Token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123456","email":"admin@example.com"}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

# 运行冒烟测试
./scripts/smoke-test.sh http://localhost:8080 "$TOKEN"
```

#### 5.2 手动验证

1. **访问 Web 界面**: 打开浏览器访问 `http://your-domain.com`
2. **登录系统**: 使用注册的用户名和密码登录
3. **查看节点**: 在节点列表中应该能看到已注册的 Daemon 节点
4. **查看 Agent**: 在节点详情中应该能看到配置的 Agent

## 🔧 常见问题

### Manager 无法连接数据库

- 检查数据库服务是否运行: `sudo systemctl status mysql`
- 检查数据库连接配置是否正确
- 检查防火墙是否允许连接: `sudo ufw allow 3306`

### Daemon 无法连接 Manager

- 检查 Manager gRPC 服务是否运行: `netstat -tlnp | grep 9090`
- 检查网络连通性: `telnet manager.example.com 9090`
- 检查防火墙是否允许连接: `sudo ufw allow 9090`

### Web 前端无法访问 API

- 检查 Nginx 配置中的 proxy_pass 地址是否正确
- 检查 Manager HTTP 服务是否运行: `curl http://localhost:8080/api/v1/health`
- 查看 Nginx 错误日志: `sudo tail -f /var/log/nginx/error.log`

## 📝 维护

### 查看日志

```bash
# Manager 日志
sudo journalctl -u ops-manager -f

# Daemon 日志
sudo journalctl -u ops-daemon -f

# Nginx 日志
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log
```

### 更新版本

1. 停止服务: `sudo systemctl stop ops-manager ops-daemon`
2. 备份当前版本: `sudo cp /usr/local/bin/ops-manager /usr/local/bin/ops-manager.backup`
3. 复制新版本: `sudo cp releases/v0.4.0/manager/manager-linux-amd64 /usr/local/bin/ops-manager`
4. 启动服务: `sudo systemctl start ops-manager ops-daemon`
5. 验证: 运行冒烟测试确认功能正常

### 回滚

参考 [回滚指南](ROLLBACK.md)

## 🔒 安全建议

1. **使用强密码**: 数据库密码、JWT secret 等应使用强随机密码
2. **启用 HTTPS**: 生产环境必须使用 HTTPS
3. **配置防火墙**: 只开放必要的端口
4. **定期更新**: 及时更新系统和依赖包
5. **监控告警**: 配置监控和告警，及时发现问题

## 📚 相关文档

- [Agent 管理功能使用指南](Agent管理功能使用指南.md)
- [Agent 管理管理员手册](Agent管理管理员手册.md)
- [回滚指南](ROLLBACK.md)
- [CHANGELOG](../CHANGELOG.md)
