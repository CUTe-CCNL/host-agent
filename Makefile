.PHONY: build run docker test clean build-all install uninstall

# 變數
BINARY_NAME=host-agent
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -s -w"

# 編譯當前平台
build:
	go build ${LDFLAGS} -o ${BINARY_NAME} .

# 編譯所有平台
build-all:
	@echo "Building for all platforms..."
	@chmod +x scripts/build.sh
	@./scripts/build.sh

# 編譯 Windows (無視窗)
build-windows:
	@chmod +x scripts/build-windows.sh
	@./scripts/build-windows.sh

# 執行
run:
	go run main.go

# 執行（指定配置）
run-config:
	go run main.go -config config.yaml

# Docker 映像
docker:
	docker build -t host-agent:${VERSION} .

# 測試
test:
	go test -v ./...

# 清理
clean:
	go clean
	rm -f ${BINARY_NAME}*
	rm -rf dist/

# 打包
package:
	@chmod +x scripts/package.sh
	@./scripts/package.sh

# 安裝依賴
deps:
	go mod download
	go mod tidy

# 安裝服務（需 sudo）
install:
	@if [ "$(shell uname)" = "Linux" ]; then \
		chmod +x installer/install.sh; \
		sudo ./installer/install.sh; \
	else \
		echo "請在 Windows 上執行 installer/install.bat"; \
	fi

# 卸載服務（需 sudo）
uninstall:
	@if [ "$(shell uname)" = "Linux" ]; then \
		chmod +x installer/uninstall.sh; \
		sudo ./installer/uninstall.sh; \
	else \
		echo "請在 Windows 上執行 installer/uninstall.bat"; \
	fi
