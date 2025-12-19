#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SERVICE_NAME="host-agent"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/host-agent"
LOG_DIR="/var/log/host-agent"
BINARY_NAME="host-agent"

echo -e "${YELLOW}=== Host Agent 卸載程式 ===${NC}"

# 檢查是否為 root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}請使用 sudo 執行此腳本${NC}"
    exit 1
fi

# 停止服務
echo "停止服務..."
systemctl stop $SERVICE_NAME 2>/dev/null || true

# 禁用服務
echo "禁用服務..."
systemctl disable $SERVICE_NAME 2>/dev/null || true

# 刪除 systemd 服務檔
echo "刪除 systemd 服務檔..."
rm -f /etc/systemd/system/$SERVICE_NAME.service

# 重新載入 systemd
systemctl daemon-reload

# 刪除執行檔
echo "刪除執行檔..."
rm -f $INSTALL_DIR/$BINARY_NAME

# 詢問是否刪除配置和日誌
read -p "是否刪除配置檔和日誌? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "刪除配置和日誌..."
    rm -rf $CONFIG_DIR
    rm -rf $LOG_DIR
fi

echo -e "${GREEN}✓ 卸載完成${NC}"
