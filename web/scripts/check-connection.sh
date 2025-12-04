#!/bin/bash

# 前端连接检查脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

ENV_DEV_FILE="$WEB_DIR/.env.development"

echo "=== Ops Scaffold Framework - 连接诊断 ==="
echo ""

# 1. 检查环境变量文件
echo "1. 检查环境变量配置..."
if [ -f "$ENV_DEV_FILE" ]; then
  echo "   ✓ .env.development 文件存在"
  # 读取环境变量（避免 source 执行注释行）
  export $(grep -v '^#' "$ENV_DEV_FILE" | xargs)
  echo "   API 地址: ${VITE_API_BASE_URL:-未设置}"
  echo "   超时时间: ${VITE_API_TIMEOUT:-未设置}ms"
else
  echo "   ✗ .env.development 文件不存在"
  echo "   运行: bash scripts/setup-env.sh"
  exit 1
fi

echo ""

# 2. 检查 Manager 服务
echo "2. 检查 Manager 服务连接..."
API_URL="${VITE_API_BASE_URL:-http://127.0.0.1:8080}"
HEALTH_URL="${API_URL}/health"

if command -v curl &> /dev/null; then
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$HEALTH_URL" 2>/dev/null)
  if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✓ Manager 服务运行正常"
    RESPONSE=$(curl -s --max-time 5 "$HEALTH_URL" 2>/dev/null)
    echo "   响应: $RESPONSE"
  else
    echo "   ✗ Manager 服务连接失败 (HTTP $HTTP_CODE)"
    echo "   请确保 Manager 服务已启动:"
    echo "   cd ../manager && make run-dev"
  fi
else
  echo "   ⚠ curl 命令未找到，跳过连接检查"
fi

echo ""

# 3. 检查端口占用
echo "3. 检查端口占用..."
if command -v lsof &> /dev/null; then
  PORT=$(echo "$API_URL" | sed -E 's|.*:([0-9]+).*|\1|')
  if lsof -i :"$PORT" &> /dev/null; then
    echo "   ✓ 端口 $PORT 已被占用（可能是 Manager 服务）"
    lsof -i :"$PORT" | head -2
  else
    echo "   ✗ 端口 $PORT 未被占用（Manager 服务可能未启动）"
  fi
else
  echo "   ⚠ lsof 命令未找到，跳过端口检查"
fi

echo ""
echo "=== 诊断完成 ==="
echo ""
echo "如果 Manager 服务未运行，请执行："
echo "  cd ../manager"
echo "  make run-dev"
echo ""
echo "如果这是首次运行，请先创建用户："
echo "  curl -X POST http://127.0.0.1:8080/api/v1/auth/register \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"admin\",\"password\":\"admin123456\",\"email\":\"admin@example.com\"}'"

