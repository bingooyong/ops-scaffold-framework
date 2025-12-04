# Ops Scaffold Framework - Web Frontend

基于 React + TypeScript + Vite + Material-UI 的运维管理平台前端应用。

## 技术栈

- **框架**: React 18.2+ with TypeScript 5.0+
- **构建工具**: Vite 5.0+
- **UI 组件库**: Material-UI (MUI) 5.14+
- **路由**: React Router 6.18+
- **状态管理**: Zustand 4.4+
- **数据请求**: TanStack React Query 5.0+ + Axios 1.6+
- **表单处理**: React Hook Form 7.48+
- **日期处理**: dayjs 1.11+

## 快速开始

### 环境要求

- Node.js 18+
- npm 或 yarn
- Manager 服务已启动（后端 API）

### 配置环境变量

首次运行前，需要配置 API 地址：

```bash
# 运行配置脚本（自动创建 .env.development 和 .env.production）
bash scripts/setup-env.sh

# 或手动创建 .env.development 文件
cat > .env.development << EOF
VITE_API_BASE_URL=http://127.0.0.1:8080
VITE_API_TIMEOUT=30000
EOF
```

**重要**: 确保 Manager 服务已启动（参考 [QUICKSTART.md](../QUICKSTART.md)）

### 安装依赖

```bash
npm install
```

### 开发环境运行

```bash
npm run dev
```

前端服务将在 `http://localhost:5173` 启动。

### 生产环境构建

```bash
npm run build
```

构建产物将输出到 `dist/` 目录。

## 项目结构

```
web/
├── src/
│   ├── api/            # API 接口层
│   ├── components/     # 组件目录
│   ├── pages/          # 页面组件
│   ├── stores/         # 状态管理
│   ├── hooks/          # 自定义 Hooks
│   ├── types/          # TypeScript 类型
│   ├── utils/          # 工具函数
│   └── theme/          # MUI 主题
├── .env.development    # 开发环境配置
└── .env.production     # 生产环境配置
```

## 功能特性

- ✅ 用户认证（登录/登出）
- ✅ JWT Token 管理
- ✅ 仪表盘（节点统计）
- ✅ 节点列表（分页、查询、删除）
- ✅ 响应式布局
- ✅ API 请求拦截器

## 故障排除

### 问题：登录时显示"网络连接失败"

**可能原因**:
1. Manager 服务未启动
2. API 地址配置错误
3. 端口被占用

**解决方法**:

1. **检查 Manager 服务是否运行**:
   ```bash
   curl http://127.0.0.1:8080/health
   ```
   应该返回 `{"status":"ok","time":"..."}`

2. **检查环境变量配置**:
   ```bash
   # 查看当前配置
   cat .env.development
   
   # 如果不存在，运行配置脚本
   bash scripts/setup-env.sh
   ```

3. **检查端口占用**:
   ```bash
   # macOS/Linux
   lsof -i :8080
   ```

4. **启动 Manager 服务**:
   ```bash
   cd ../manager
   make run-dev
   ```

### 问题：登录失败 - "用户名或密码错误"

**解决方法**:

1. **创建第一个用户**（如果还没有）:
   ```bash
   curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
     -H "Content-Type: application/json" \
     -d '{
       "username": "admin",
       "password": "admin123456",
       "email": "admin@example.com"
     }'
   ```

2. 使用注册的用户名和密码登录

详细说明请参考 [QUICKSTART.md](../QUICKSTART.md)

## License

MIT
