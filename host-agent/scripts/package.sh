#!/bin/bash

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

echo "Packaging Host Agent v${VERSION}"
echo "================================"

# 清理舊的打包
rm -rf dist/
mkdir -p dist

# Linux
echo "Packaging for Linux..."
mkdir -p dist/linux
cp host-agent-linux-amd64 dist/linux/host-agent 2>/dev/null || echo "Warning: Linux binary not found"
cp config.yaml dist/linux/
cp installer/install.sh dist/linux/
cp installer/uninstall.sh dist/linux/
chmod +x dist/linux/install.sh
chmod +x dist/linux/uninstall.sh
cd dist && tar -czf host-agent-${VERSION}-linux-amd64.tar.gz linux/ && cd ..

# Windows
echo "Packaging for Windows..."
mkdir -p dist/windows
cp host-agent-windows-amd64.exe dist/windows/host-agent.exe 2>/dev/null || echo "Warning: Windows binary not found"
cp config.yaml dist/windows/
cp installer/install.bat dist/windows/
cp installer/uninstall.bat dist/windows/
cd dist && zip -r host-agent-${VERSION}-windows-amd64.zip windows/ && cd ..

echo ""
echo "✓ Packages created in dist/"
ls -lh dist/*.tar.gz dist/*.zip 2>/dev/null || echo "No packages created"
