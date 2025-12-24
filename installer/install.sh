#!/bin/bash

set -e

# 顏色輸出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 配置
SERVICE_NAME="host-agent"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/host-agent"
LOG_DIR="/var/log/host-agent"
BINARY_NAME="host-agent"

echo -e "${GREEN}=== Host Agent 安裝程式 ===${NC}"

# 檢查是否為 root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}請使用 sudo 執行此腳本${NC}"
    exit 1
fi

# 檢查系統架構
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        BINARY_FILE="${BINARY_NAME}"
        ;;
    aarch64|arm64)
        BINARY_FILE="${BINARY_NAME}"
        ;;
    *)
        echo -e "${RED}不支援的架構: $ARCH${NC}"
        exit 1
        ;;
esac

# 檢查執行檔是否存在
if [ ! -f "$BINARY_FILE" ]; then
    echo -e "${RED}找不到執行檔: $BINARY_FILE${NC}"
    echo "請先執行 make build-all 編譯"
    exit 1
fi

# 檢測 init 系統
INIT_SYSTEM=""
if [ -d /run/systemd/system ] || systemctl --version >/dev/null 2>&1; then
    INIT_SYSTEM="systemd"
elif [ -f /etc/alpine-release ] || command -v rc-status >/dev/null 2>&1 || [ -f /sbin/openrc ]; then
    INIT_SYSTEM="openrc"
else
    echo -e "${YELLOW}警告: 無法檢測 init 系統，預設使用 systemd${NC}"
    INIT_SYSTEM="systemd"
fi

echo -e "${GREEN}檢測到 init 系統: $INIT_SYSTEM${NC}"

# 停止舊服務
echo -e "${YELLOW}停止舊服務...${NC}"
if [ "$INIT_SYSTEM" = "systemd" ]; then
    systemctl stop $SERVICE_NAME 2>/dev/null || true
elif [ "$INIT_SYSTEM" = "openrc" ]; then
    rc-service $SERVICE_NAME stop 2>/dev/null || true
fi

# 建立目錄
echo "建立目錄..."
mkdir -p $CONFIG_DIR
mkdir -p $LOG_DIR

# 複製執行檔
echo "安裝執行檔..."
cp $BINARY_FILE $INSTALL_DIR/$BINARY_NAME
chmod +x $INSTALL_DIR/$BINARY_NAME

# 複製配置檔（如果不存在）
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    echo "安裝預設配置檔..."
    cp config.yaml $CONFIG_DIR/
else
    echo -e "${YELLOW}配置檔已存在，跳過${NC}"
fi

# 建立服務檔
if [ "$INIT_SYSTEM" = "systemd" ]; then
    echo "建立 systemd 服務檔..."
    cat > /etc/systemd/system/$SERVICE_NAME.service << EOF
[Unit]
Description=Host Monitoring Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=$INSTALL_DIR/$BINARY_NAME -service -config $CONFIG_DIR/config.yaml
Restart=on-failure
RestartSec=5s
StandardOutput=append:$LOG_DIR/agent.log
StandardError=append:$LOG_DIR/agent.log

[Install]
WantedBy=multi-user.target
EOF

    # 重新載入 systemd
    echo "重新載入 systemd..."
    systemctl daemon-reload

    # 啟用服務
    echo "啟用服務..."
    systemctl enable $SERVICE_NAME

    # 啟動服務
    echo "啟動服務..."
    systemctl start $SERVICE_NAME

    # 檢查狀態
    sleep 2
    if systemctl is-active --quiet $SERVICE_NAME; then
        echo -e "${GREEN}✓ 安裝成功！${NC}"
        echo ""
        echo "服務狀態: $(systemctl is-active $SERVICE_NAME)"
        echo "配置檔案: $CONFIG_DIR/config.yaml"
        echo "日誌檔案: $LOG_DIR/agent.log"
        echo ""
        echo "常用命令:"
        echo "  查看狀態: systemctl status $SERVICE_NAME"
        echo "  查看日誌: journalctl -u $SERVICE_NAME -f"
        echo "  重啟服務: systemctl restart $SERVICE_NAME"
        echo "  停止服務: systemctl stop $SERVICE_NAME"
        echo "  卸載服務: sudo ./uninstall.sh"
        echo ""
        echo "測試 API:"
        echo "  curl http://localhost:9100/health"
    else
        echo -e "${RED}✗ 服務啟動失敗${NC}"
        echo "查看日誌: journalctl -u $SERVICE_NAME -n 50"
        exit 1
    fi

elif [ "$INIT_SYSTEM" = "openrc" ]; then
    echo "建立 OpenRC 服務檔..."
    cat > /etc/init.d/$SERVICE_NAME << EOF
#!/sbin/openrc-run

name="\$RC_SVCNAME"
description="Host Monitoring Agent"
command="$INSTALL_DIR/$BINARY_NAME"
command_args="-service -config $CONFIG_DIR/config.yaml"
command_user="root"
pidfile="/var/run/\${name}.pid"
start_stop_daemon_args="--background --make-pidfile --stdout $LOG_DIR/agent.log --stderr $LOG_DIR/agent.log"

depend() {
    need net
    after net
}
EOF

    # 設置執行權限
    chmod +x /etc/init.d/$SERVICE_NAME

    # 啟用服務
    echo "啟用服務..."
    rc-update add $SERVICE_NAME default 2>/dev/null || true

    # 啟動服務
    echo "啟動服務..."
    rc-service $SERVICE_NAME start

    # 檢查狀態
    sleep 2
    if rc-service $SERVICE_NAME status >/dev/null 2>&1; then
        echo -e "${GREEN}✓ 安裝成功！${NC}"
        echo ""
        echo "服務狀態: $(rc-service $SERVICE_NAME status 2>&1 | head -1 || echo 'running')"
        echo "配置檔案: $CONFIG_DIR/config.yaml"
        echo "日誌檔案: $LOG_DIR/agent.log"
        echo ""
        echo "常用命令:"
        echo "  查看狀態: rc-service $SERVICE_NAME status"
        echo "  查看日誌: tail -f $LOG_DIR/agent.log"
        echo "  重啟服務: rc-service $SERVICE_NAME restart"
        echo "  停止服務: rc-service $SERVICE_NAME stop"
        echo "  卸載服務: sudo ./uninstall.sh"
        echo ""
        echo "測試 API:"
        echo "  curl http://localhost:9100/health"
    else
        echo -e "${RED}✗ 服務啟動失敗${NC}"
        echo "查看日誌: tail -n 50 $LOG_DIR/agent.log"
        exit 1
    fi
fi
