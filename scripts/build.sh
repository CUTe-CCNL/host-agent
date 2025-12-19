#!/bin/bash

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
BINARY_NAME="host-agent"

LDFLAGS="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -s -w"

echo "Building Host Agent v${VERSION}"
echo "================================"

# Linux AMD64
echo "Building for Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o ${BINARY_NAME}-linux-amd64

# Linux ARM64
echo "Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o ${BINARY_NAME}-linux-arm64

# Windows AMD64
echo "Building for Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o ${BINARY_NAME}-windows-amd64.exe

# macOS AMD64
echo "Building for macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o ${BINARY_NAME}-darwin-amd64

# macOS ARM64 (Apple Silicon)
echo "Building for macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o ${BINARY_NAME}-darwin-arm64

echo ""
echo "✓ Build complete!"
ls -lh ${BINARY_NAME}-*
