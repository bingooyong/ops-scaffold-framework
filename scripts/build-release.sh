#!/bin/bash

# 发布构建脚本
# 用法: ./scripts/build-release.sh [version]
# 示例: ./scripts/build-release.sh v0.4.0

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取版本号
VERSION=${1:-$(git describe --tags --always 2>/dev/null || echo "dev")}
RELEASE_DIR="releases/${VERSION}"

echo -e "${GREEN}开始构建发布包: ${VERSION}${NC}"

# 检查是否在项目根目录
if [ ! -f "README.md" ]; then
    echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 创建发布目录
echo -e "${YELLOW}创建发布目录...${NC}"
mkdir -p "${RELEASE_DIR}"/{manager,daemon,web/docs}

# 构建 Manager
echo -e "${YELLOW}构建 Manager...${NC}"
cd manager
make build-linux
cd ..
cp manager/bin/manager-linux-amd64 "${RELEASE_DIR}/manager/"
cp manager/configs/manager.yaml "${RELEASE_DIR}/manager/manager.yaml.example"
cp manager/README.md "${RELEASE_DIR}/manager/"

# 构建 Daemon
echo -e "${YELLOW}构建 Daemon...${NC}"
cd daemon
make build-linux
cd ..
cp daemon/bin/daemon-linux-amd64 "${RELEASE_DIR}/daemon/"
cp daemon/configs/daemon.multi-agent.example.yaml "${RELEASE_DIR}/daemon/daemon.yaml.example"
cp daemon/README.md "${RELEASE_DIR}/daemon/"

# 构建 Web 前端
echo -e "${YELLOW}构建 Web 前端...${NC}"
cd web
npm run build
cd ..
cp -r web/dist "${RELEASE_DIR}/web/"

# 复制文档
echo -e "${YELLOW}复制文档...${NC}"
cp CHANGELOG.md README.md "${RELEASE_DIR}/"
cp docs/Agent管理*.md "${RELEASE_DIR}/web/docs/" 2>/dev/null || true

# 生成校验和
echo -e "${YELLOW}生成校验和...${NC}"
cd "${RELEASE_DIR}"
sha256sum manager/manager-linux-amd64 > manager/manager-linux-amd64.sha256
sha256sum daemon/daemon-linux-amd64 > daemon/daemon-linux-amd64.sha256

# 显示发布包内容
echo -e "${GREEN}发布包构建完成!${NC}"
echo -e "${GREEN}发布包位置: ${RELEASE_DIR}${NC}"
echo ""
echo -e "${YELLOW}发布包内容:${NC}"
find . -type f | sort

echo ""
echo -e "${GREEN}校验和:${NC}"
cat manager/manager-linux-amd64.sha256
cat daemon/daemon-linux-amd64.sha256
