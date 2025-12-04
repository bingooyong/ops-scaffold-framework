#!/bin/bash

set -e

# 安装Daemon守护进程

echo "Installing Daemon..."

# 检查是否为root用户
if [ "$(id -u)" != "0" ]; then
   echo "This script must be run as root"
   exit 1
fi

# 定义变量
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/daemon"
WORK_DIR="/var/lib/daemon"
LOG_DIR="/var/log/daemon"
SYSTEMD_DIR="/etc/systemd/system"

# 创建必要的目录
echo "Creating directories..."
mkdir -p "$CONFIG_DIR"
mkdir -p "$WORK_DIR"
mkdir -p "$LOG_DIR"
mkdir -p "$CONFIG_DIR/certs"
mkdir -p "$WORK_DIR/downloads"
mkdir -p "$WORK_DIR/backups"

# 复制二进制文件
echo "Installing binary..."
if [ -f "bin/daemon" ]; then
    install -m 755 bin/daemon "$BIN_DIR/daemon"
else
    echo "Error: bin/daemon not found. Please build first: make build"
    exit 1
fi

# 复制配置文件
echo "Installing config..."
if [ ! -f "$CONFIG_DIR/daemon.yaml" ]; then
    cp configs/daemon.yaml "$CONFIG_DIR/daemon.yaml"
    echo "Config file installed to $CONFIG_DIR/daemon.yaml"
    echo "Please edit this file before starting the daemon"
else
    echo "Config file already exists, skipping"
fi

# 安装systemd服务文件
echo "Installing systemd service..."
cp scripts/systemd/daemon.service "$SYSTEMD_DIR/daemon.service"
systemctl daemon-reload

# 设置权限
echo "Setting permissions..."
chown -R root:root "$CONFIG_DIR"
chown -R root:root "$WORK_DIR"
chown -R root:root "$LOG_DIR"
chmod 755 "$WORK_DIR"
chmod 755 "$LOG_DIR"

echo "Installation complete!"
echo ""
echo "Next steps:"
echo "1. Edit configuration file: $CONFIG_DIR/daemon.yaml"
echo "2. Start the service: systemctl start daemon"
echo "3. Enable autostart: systemctl enable daemon"
echo "4. Check status: systemctl status daemon"
