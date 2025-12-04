# Web 前端快速启动指南

## 前置条件

1. **后端 Manager 服务已启动**：
   ```bash
   cd ../manager
   ./bin/manager -config configs/manager.dev.yaml
   ```

   Manager 服务应在 `http://127.0.0.1:8080` 运行。

2. **MySQL 数据库已启动**：
   确保 MySQL 服务正在运行，数据库 `ops_manager_dev` 已创建。

## 启动步骤

### 1. 安装依赖（首次运行）

```bash
npm install
```

### 2. 启动开发服务器

```bash
npm run dev
```

前端服务将在 `http://localhost:5173` 启动。

### 3. 访问应用

在浏览器中打开 `http://localhost:5173`，你将看到登录页面。

## 测试账号

### 方式1: 使用注册功能

1. 在登录页面，先注册一个新账号
2. 使用 Manager API 的注册接口：
   ```bash
   curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
     -H "Content-Type: application/json" \
     -d '{
       "username": "admin",
       "password": "admin123",
       "email": "admin@example.com"
     }'
   ```

### 方式2: 使用集成测试中的账号

运行 Manager 集成测试后，会自动创建测试账号：
- 用户名：`testuser_<timestamp>`
- 密码：测试过程中修改为 `newpassword456`

查看测试日志获取具体的用户名。

## 主要功能

登录后，你可以：

1. **仪表盘 (Dashboard)**
   - 查看节点统计信息（总数、在线、离线）

2. **节点管理 (Nodes)**
   - 查看节点列表
   - 分页浏览节点
   - 删除节点
   - 查看节点详细信息（主机名、IP、操作系统等）

## 开发说明

### 目录结构

- `src/api/` - API 接口定义
- `src/pages/` - 页面组件
- `src/components/` - 可复用组件
- `src/stores/` - 全局状态管理
- `src/hooks/` - 自定义 React Hooks
- `src/types/` - TypeScript 类型定义

### 添加新页面

1. 在 `src/pages/` 创建新的页面组件
2. 在 `src/App.tsx` 添加路由配置
3. 在 `src/components/Layout/MainLayout.tsx` 添加菜单项

### API 代理配置

开发环境下，所有 `/api/*` 请求会自动代理到 `http://127.0.0.1:8080`。

配置文件：`vite.config.ts`

### 环境变量

- `.env.development` - 开发环境配置
- `.env.production` - 生产环境配置

## 常见问题

### 1. 无法连接到后端

确认：
- Manager 服务正在运行
- 端口 8080 未被占用
- 检查 `vite.config.ts` 中的代理配置

### 2. 登录后跳转失败

检查：
- 浏览器控制台是否有错误
- Token 是否正确保存到 localStorage
- API 响应格式是否正确

### 3. 页面样式异常

尝试：
- 清除浏览器缓存
- 重启开发服务器
- 检查 MUI 主题配置

## 生产环境部署

### 1. 构建

```bash
npm run build
```

### 2. 部署

将 `dist/` 目录部署到 Web 服务器（Nginx、Apache 等）。

### 3. Nginx 配置示例

```nginx
server {
    listen 80;
    server_name your-domain.com;

    root /path/to/web/dist;
    index index.html;

    # 前端路由
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 技术支持

如遇问题，请查看：
- 浏览器开发者控制台
- Manager 服务日志
- Vite 开发服务器输出
