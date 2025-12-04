# 快速启动指南

本指南将帮助您快速启动 Ops Scaffold Framework 的各个组件。

## 前置要求

1. **Go 1.21+** - Manager 和 Daemon 需要
2. **Node.js 18+** - Web 前端需要
3. **MySQL 8.0+** - 数据库
4. **Redis**（可选）- 缓存

## 启动步骤

### 1. 启动 Manager（后端服务）

Manager 是中心管理节点，提供 HTTP API 和 gRPC 服务。

```bash
cd manager

# 1. 安装依赖
make deps

# 2. 确保数据库已创建并配置正确
# 编辑 configs/manager.dev.yaml，修改数据库连接信息：
# database:
#   dsn: "root:your_password@tcp(127.0.0.1:3306)/ops_manager_dev?charset=utf8mb4&parseTime=True&loc=Local"

# 3. 启动 Manager（开发模式）
make run-dev
```

Manager 将在以下地址启动：
- HTTP API: `http://127.0.0.1:8080`
- gRPC: `127.0.0.1:9090`

### 2. 创建第一个用户

Manager 启动后，数据库会自动迁移。但需要创建第一个用户才能登录。

#### 方法 1: 使用注册接口（推荐）

```bash
curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123456",
    "email": "admin@example.com"
  }'
```

**注意**: 默认注册的用户角色是 `user`，如果需要管理员权限，需要手动修改数据库或使用管理员接口。

#### 方法 2: 直接修改数据库

```sql
-- 连接到数据库
mysql -u root -p ops_manager_dev

-- 插入管理员用户（密码: admin123456）
INSERT INTO users (username, password, email, role, status, created_at, updated_at)
VALUES (
  'admin',
  '$2a$10$YourHashedPasswordHere',  -- 使用 bcrypt 加密后的密码
  'admin@example.com',
  'admin',
  'active',
  NOW(),
  NOW()
);
```

**生成 bcrypt 密码**（使用 Go）:
```go
package main
import "golang.org/x/crypto/bcrypt"
func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
    println(string(hash))
}
```

### 3. 启动 Web 前端

```bash
cd web

# 1. 安装依赖（首次运行）
npm install

# 2. 启动开发服务器
npm run dev
```

前端将在 `http://localhost:5173` 启动。

### 4. 登录

1. 打开浏览器访问 `http://localhost:5173`
2. 使用步骤 2 中创建的用户名和密码登录

## 常见问题

### 问题 1: 前端显示"网络连接失败"

**可能原因**:
1. Manager 服务未启动
2. API 地址配置错误
3. 端口被占用

**解决方法**:
1. 检查 Manager 是否正在运行：
   ```bash
   curl http://127.0.0.1:8080/health
   ```
   应该返回 `{"status":"ok","time":"..."}`

2. 检查前端环境变量配置：
   - 确保 `web/.env.development` 文件存在
   - 检查 `VITE_API_BASE_URL` 是否正确

3. 检查端口占用：
   ```bash
   # macOS/Linux
   lsof -i :8080
   
   # 如果端口被占用，修改 manager/configs/manager.dev.yaml 中的端口
   ```

### 问题 2: 数据库连接失败

**可能原因**:
1. MySQL 服务未启动
2. 数据库不存在
3. 用户名/密码错误

**解决方法**:
1. 启动 MySQL 服务
2. 创建数据库：
   ```sql
   CREATE DATABASE ops_manager_dev CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
   ```
3. 检查 `manager/configs/manager.dev.yaml` 中的数据库配置

### 问题 3: 登录失败 - "用户名或密码错误"

**可能原因**:
1. 用户不存在
2. 密码错误
3. 用户被禁用

**解决方法**:
1. 使用注册接口创建新用户
2. 检查数据库中的用户状态：
   ```sql
   SELECT username, role, status FROM users;
   ```

## 验证安装

### 1. 检查 Manager 健康状态

```bash
curl http://127.0.0.1:8080/health
```

### 2. 测试登录接口

```bash
curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123456"
  }'
```

应该返回包含 `token` 和 `user` 信息的 JSON 响应。

### 3. 检查前端 API 连接

打开浏览器开发者工具（F12），查看 Network 标签页，确认 API 请求是否成功。

## 下一步

- 查看 [Manager README](manager/README.md) 了解后端 API
- 查看 [Web README](web/README.md) 了解前端开发
- 查看 [设计文档](docs/) 了解系统架构

