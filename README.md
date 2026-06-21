# Host Agent

輕量級主機監控 Agent，支援 Linux 和 Windows，可安裝為系統服務。

## 功能特性

- 收集 CPU、記憶體、磁碟、網路指標
- RESTful API 查詢
- 自動回報到後端系統
- 跨平台支援（Linux/Windows/macOS）
- 系統服務安裝（開機自啟動）
- 無視窗背景執行
- 輕量級（單一執行檔）

## 快速開始

### 1. 編譯

```bash
# 安裝依賴
go mod tidy

# 編譯當前平台
make build

# 編譯所有平台
make build-all
```

### 2. Linux 安裝

```bash
# 複製檔案
scp host-agent-linux-amd64 root@server:/tmp/
scp config.yaml root@server:/tmp/
scp installer/install.sh root@server:/tmp/

# SSH 登入並安裝
ssh root@server
cd /tmp
chmod +x install.sh
sudo ./install.sh

# 檢查狀態
systemctl status host-agent
curl http://localhost:9100/health
```

### 3. Windows 安裝

```batch
REM 以管理員身份執行
install.bat

REM 檢查服務
sc query HostAgent
curl http://localhost:9100/health
```

## API 端點

```bash
# 健康檢查
GET /health

# 完整指標
GET /metrics

# 個別指標
GET /metrics/cpu
GET /metrics/memory
GET /metrics/disk
GET /metrics/network
GET /metrics/process

# 插件管理
GET  /plugins
GET  /plugins/{id}
POST /plugins/{id}/restart
ANY  /plugin-api/{id}/{path}
```

## 配置說明

編輯 `config.yaml`:

```yaml
server:
  port: 9100              # API 端口
  enable_auth: false      # 是否啟用認證

collector:
  enable_cpu: true        # 啟用 CPU 監控
  enable_memory: true     # 啟用記憶體監控
  enable_disk: true       # 啟用磁碟監控
  enable_network: true    # 啟用網路監控

report:
  enabled: true           # 啟用自動回報
  mode: "rabbitmq"        # http, rabbitmq, both
  interval: 30s           # 回報間隔
  http:
    endpoint: "http://localhost:8080/api/metrics/collect"
  rabbitmq:
    url: "amqp://guest:guest@localhost:5672/"
    exchange: "host-metrics"
    exchange_type: "topic"
    routing_key_template: "host.metrics"
    durable: true
    auto_delete: false

plugins:
  enabled: false
  directory: "/etc/host-agent/plugins.d"
  startup_timeout: 10s
  health_interval: 15s
  request_timeout: 30s
```

插件 manifest 放在 `plugins.directory` 中，一個插件一個 YAML 檔。插件必須只在本機 loopback HTTP 監聽，host-agent 會統一代理到 `/plugin-api/{id}/...`。

## 服務管理

### Linux

```bash
# 查看狀態
systemctl status host-agent

# 啟動/停止/重啟
systemctl start host-agent
systemctl stop host-agent
systemctl restart host-agent

# 查看日誌
journalctl -u host-agent -f

# 卸載
sudo ./uninstall.sh
```

### Windows

```batch
# 查看狀態
sc query HostAgent

# 啟動/停止
net start HostAgent
net stop HostAgent

# 卸載
uninstall.bat
```

## 開發

```bash
# 前台運行（測試）
go run main.go -config config.yaml

# 或
./host-agent -config config.yaml
```

## 授權

Apache License 2.0
