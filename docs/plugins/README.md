# 插件建立指南

host-agent plugin 是由 host-agent 啟動的子進程。插件不需要自己監聽 HTTP port；host-agent 會對外提供 `/plugin-api/{id}/...`，再透過 stdio newline-delimited JSON-RPC 2.0 把請求交給插件。

## 建立流程

1. 寫一個 Go 可執行程式。
2. 程式從 stdin 逐行讀取 JSON-RPC request。
3. 程式把 JSON-RPC response 逐行寫到 stdout。
4. 程式把日誌、診斷訊息、錯誤堆疊寫到 stderr。
5. 在 `plugins.directory` 建立一個 YAML manifest。
6. 啟動 host-agent，確認 `/plugins/{id}` 顯示 `status: "running"`。
7. 透過 `/plugin-api/{id}/...` 測試 routes 是否能正常代理。

## Manifest

manifest 檔案放在 `plugins.directory`，副檔名必須是 `.yaml` 或 `.yml`。每個插件一個 manifest。Go 範例的完整 manifest 檔在 `docs/plugins/example-go/manifest.yaml`。

```yaml
id: "example-go"
name: "Example Go Plugin"
version: "0.1.0"
enabled: true
description: "Example stdio JSON-RPC plugin written in Go"
command: "/opt/host-agent/plugins/example-go/example-go-plugin"
working_dir: "/opt/host-agent/plugins/example-go"
env:
  LOG_LEVEL: "info"
routes:
  - path_prefix: "/status"
    methods: ["GET"]
  - path_prefix: "/echo"
    methods: ["GET", "POST"]
  - path_prefix: "/items"
    methods: ["GET", "POST", "DELETE"]
```

欄位說明:

- `id`: 插件識別碼，只能使用小寫英文字母、數字、hyphen，且不能以 hyphen 開頭或結尾。
- `name`: 顯示名稱。
- `version`: 插件版本。
- `enabled`: 是否由 host-agent 啟動。
- `description`: 選填，插件描述。
- `command`: 必填，host-agent 要啟動的可執行檔。
- `args`: 選填，傳給 `command` 的參數。
- `working_dir`: 選填，插件工作目錄。
- `env`: 選填，額外環境變數。
- `routes`: 必填，對外 REST route 宣告。

`routes.methods` 只支援 `GET`、`POST`、`PUT`、`PATCH`、`DELETE`。`routes.path_prefix` 必須以 `/` 開頭。host-agent 會先檢查 manifest routes，允許的 request 才會轉交給插件。

## IPC 規則

- stdin/stdout 只用於 JSON-RPC，不可混入一般文字。
- 每一行是一個完整 JSON-RPC 物件，以 newline 作為 framing。
- 插件必須依照 request `id` 回傳 response。
- host-agent 可能同時送出多個 request，插件應用 `id` 對應回應。
- 日誌必須寫到 stderr。

## 必要 Methods

插件必須實作 `plugin.health` 和 `plugin.handle_http`。

Go plugin 可以直接使用 `host-agent/pkg/pluginipc`，不用手寫 JSON-RPC framing。若 plugin 在另一個 Go module 中開發，可以先在該專案的 `go.mod` 使用 replace 指向 host-agent repo:

```go
require host-agent v0.0.0

replace host-agent => /path/to/host-agent
```

然後在 plugin 程式中引用:

```go
import "github.com/CUTe-CCNL/host-agent/pkg/pluginipc"
```

### plugin.health

host-agent 會在啟動時和健康檢查週期呼叫此 method。

request:

```json
{"jsonrpc":"2.0","id":1,"method":"plugin.health","params":{}}
```

成功 response:

```json
{"jsonrpc":"2.0","id":1,"result":{"ok":true}}
```

如果 `ok` 不是 `true`，或插件沒有在 timeout 內回應，host-agent 會把插件標成 failed 或 unhealthy。

### plugin.handle_http

host-agent 收到 `/plugin-api/{id}/...` 後，會把 HTTP-like request 包成 JSON-RPC。

request:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "plugin.handle_http",
  "params": {
    "method": "POST",
    "path": "/echo",
    "raw_query": "",
    "headers": {
      "Content-Type": ["application/json"]
    },
    "body_base64": "aGVsbG8="
  }
}
```

response:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "status_code": 201,
    "headers": {
      "Content-Type": ["application/json"]
    },
    "body_base64": "eyJvayI6dHJ1ZX0="
  }
}
```

`body_base64` 是原始 HTTP body 的 base64 字串。插件回應 body 時也必須使用 base64。

host-agent 在轉交 request 給插件前，會拒絕超過 6 MiB 的 HTTP request body，並回傳 `413 Request Entity Too Large`。

## Go 範例

範例程式在 `docs/plugins/example-go/main.go`。它實作:

- `plugin.health`
- `GET /status`
- `GET /echo`
- `POST /echo`
- `GET /items`
- `POST /items`
- `GET /items/{id}`
- `DELETE /items/{id}`

範例使用 `host-agent/pkg/pluginipc`，plugin 作者只需要實作:

```go
type Handler interface {
    Health(context.Context) error
    HandleHTTP(context.Context, pluginipc.HTTPRequest) (pluginipc.HTTPResponse, error)
}
```

然後在 `main` 呼叫:

```go
pluginipc.Serve(context.Background(), myHandler{})
```

`pluginipc` 會負責 stdin/stdout JSON-RPC framing、`body_base64` decode/encode，以及 `plugin.health` / `plugin.handle_http` 的協定細節。

編譯範例:

```bash
go build -o /opt/host-agent/plugins/example-go/example-go-plugin ./docs/plugins/example-go
```

manifest 範例可直接使用 `docs/plugins/example-go/manifest.yaml`，或複製下列內容到 `plugins.directory/example-go.yaml`:

```yaml
id: "example-go"
name: "Example Go Plugin"
version: "0.1.0"
enabled: true
description: "Example stdio JSON-RPC plugin written in Go"
command: "/opt/host-agent/plugins/example-go/example-go-plugin"
working_dir: "/opt/host-agent/plugins/example-go"
routes:
  - path_prefix: "/status"
    methods: ["GET"]
  - path_prefix: "/echo"
    methods: ["GET", "POST"]
  - path_prefix: "/items"
    methods: ["GET", "POST", "DELETE"]
```

呼叫範例:

```bash
curl http://localhost:9100/plugin-api/example-go/status
curl http://localhost:9100/plugin-api/example-go/echo
curl -X POST http://localhost:9100/plugin-api/example-go/echo -d 'hello'
curl http://localhost:9100/plugin-api/example-go/items
curl -X POST http://localhost:9100/plugin-api/example-go/items -d '{"name":"Gamma"}'
curl http://localhost:9100/plugin-api/example-go/items/alpha
curl -X DELETE http://localhost:9100/plugin-api/example-go/items/alpha
```

## 錯誤語意

JSON-RPC `error` 代表 IPC 或插件基礎設施失敗，例如 request 無法解析、內部 panic、必要資源不存在。host-agent 會把這類錯誤轉成 `502 Bad Gateway`。

HTTP 應用層錯誤不要使用 JSON-RPC `error`。例如使用者送錯參數時，插件應回傳:

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "status_code": 400,
    "headers": {
      "Content-Type": ["application/json"]
    },
    "body_base64": "eyJlcnJvciI6ImludmFsaWQgcmVxdWVzdCJ9"
  }
}
```

## 開發檢查清單

- stdout 沒有日誌，只輸出 JSON-RPC response。
- stderr 可輸出日誌。
- `plugin.health` 能快速回應 `{"ok":true}`。
- `plugin.handle_http` 正確處理 method、path、raw query、headers、body。
- manifest routes 只宣告插件實際支援的 path 和 method。
- 不再使用 `upstream_url` 或 `health_path`。
- 插件不需要監聽本機 HTTP port。
