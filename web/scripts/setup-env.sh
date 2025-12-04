#!/bin/bash

# 前端环境变量设置脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

ENV_DEV_FILE="$WEB_DIR/.env.development"
ENV_PROD_FILE="$WEB_DIR/.env.production"

# 默认配置
DEFAULT_API_BASE_URL="http://127.0.0.1:8080"
DEFAULT_API_TIMEOUT="30000"

echo "=== Ops Scaffold Framework - Web 前端环境配置 ==="
echo ""

# 检查 .env.development
if [ ! -f "$ENV_DEV_FILE" ]; then
  echo "创建开发环境配置文件: .env.development"
  cat > "$ENV_DEV_FILE" << EOF
# 开发环境配置
# Vite 会自动加载此文件

# API 基础地址
VITE_API_BASE_URL=${DEFAULT_API_BASE_URL}

# API 请求超时时间（毫秒）
VITE_API_TIMEOUT=${DEFAULT_API_TIMEOUT}
EOF
  echo "✓ 已创建 .env.development"
else
  echo "✓ .env.development 已存在"
fi

# 检查 .env.production
if [ ! -f "$ENV_PROD_FILE" ]; then
  echo "创建生产环境配置文件: .env.production"
  cat > "$ENV_PROD_FILE" << EOF
# 生产环境配置
# Vite 会自动加载此文件

# API 基础地址（生产环境需要根据实际情况修改）
VITE_API_BASE_URL=${DEFAULT_API_BASE_URL}

# API 请求超时时间（毫秒）
VITE_API_TIMEOUT=${DEFAULT_API_TIMEOUT}
EOF
  echo "✓ 已创建 .env.production"
else
  echo "✓ .env.production 已存在"
fi

echo ""
echo "配置完成！"
echo ""
echo "当前配置："
echo "  API 地址: ${DEFAULT_API_BASE_URL}"
echo "  超时时间: ${DEFAULT_API_TIMEOUT}ms"
echo ""
echo "如需修改，请编辑以下文件："
echo "  - 开发环境: $ENV_DEV_FILE"
echo "  - 生产环境: $ENV_PROD_FILE"
echo ""
echo "提示：修改配置后需要重启开发服务器（npm run dev）"

