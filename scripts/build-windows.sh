#!/bin/bash

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -s -w"

echo "Building Host Agent for Windows v${VERSION}"

# 1. 控制台版本（除錯用）
echo "Building console version..."
GOOS=windows GOARCH=amd64 go build \
  -ldflags "${LDFLAGS}" \
  -o host-agent-windows-amd64-console.exe

# 2. 無視窗版本（生產用）
echo "Building GUI version (no console)..."
GOOS=windows GOARCH=amd64 go build \
  -ldflags "${LDFLAGS} -H windowsgui" \
  -o host-agent-windows-amd64.exe

# 3. 打包
echo "Packaging..."
mkdir -p dist/windows
cp host-agent-windows-amd64.exe dist/windows/
cp host-agent-windows-amd64-console.exe dist/windows/
cp config.yaml dist/windows/
cp installer/install.bat dist/windows/
cp installer/uninstall.bat dist/windows/

# 建立 README
cat > dist/windows/README.txt << 'EOF'
Host Agent 安裝說明
==================

檔案說明:
---------
- host-agent-windows-amd64.exe        # 無視窗版本（推薦）
- host-agent-windows-amd64-console.exe # 控制台版本（除錯用）
- install.bat                          # 安裝腳本
- uninstall.bat                        # 卸載腳本
- config.yaml                          # 配置檔

安裝步驟:
---------
1. 以管理員身份執行 install.bat
2. 服務會自動啟動，完全在背景執行
3. 訪問 http://localhost:9100/health 測試

測試模式:
---------
如果需要在前台測試並查看輸出:
  host-agent-windows-amd64-console.exe -config config.yaml

服務管理:
---------
啟動: net start HostAgent
停止: net stop HostAgent
狀態: sc query HostAgent
卸載: uninstall.bat
EOF

cd dist && zip -r host-agent-${VERSION}-windows-amd64.zip windows/ && cd ..

echo ""
echo "✓ Build complete!"
echo "  Console version: host-agent-windows-amd64-console.exe (有視窗)"
echo "  GUI version:     host-agent-windows-amd64.exe (無視窗)"
echo "  Package:         dist/host-agent-${VERSION}-windows-amd64.zip"
