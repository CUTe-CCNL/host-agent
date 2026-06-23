package pluginipc_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"host-agent/pkg/pluginipc"
)

type testHandler struct {
	healthErr error
	httpErr   error
	lastReq   pluginipc.HTTPRequest
}

func (h *testHandler) Health(ctx context.Context) error {
	return h.healthErr
}

func (h *testHandler) HandleHTTP(ctx context.Context, req pluginipc.HTTPRequest) (pluginipc.HTTPResponse, error) {
	h.lastReq = req
	if h.httpErr != nil {
		return pluginipc.HTTPResponse{}, h.httpErr
	}
	return pluginipc.HTTPResponse{
		StatusCode: http.StatusCreated,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
			"X-Plugin":     {"example"},
		},
		Body: []byte(`{"ok":true}`),
	}, nil
}

func TestServeStreamHandlesHealth(t *testing.T) {
	output := serveOne(t, `{"jsonrpc":"2.0","id":1,"method":"plugin.health","params":{}}`, &testHandler{})

	if output.JSONRPC != "2.0" {
		t.Fatalf("JSONRPC = %q, want 2.0", output.JSONRPC)
	}
	if string(output.ID) != "1" {
		t.Fatalf("ID = %s, want 1", output.ID)
	}

	result := decodeResult[map[string]bool](t, output)
	if !result["ok"] {
		t.Fatalf("result[ok] = false, want true")
	}
}

func TestServeStreamHandlesHTTPRequest(t *testing.T) {
	handler := &testHandler{}
	bodyBase64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	output := serveOne(t, `{"jsonrpc":"2.0","id":2,"method":"plugin.handle_http","params":{"method":"POST","path":"/echo","raw_query":"dry_run=true","headers":{"Content-Type":["text/plain"]},"body_base64":"`+bodyBase64+`"}}`, handler)

	if handler.lastReq.Method != http.MethodPost {
		t.Fatalf("Method = %s, want POST", handler.lastReq.Method)
	}
	if handler.lastReq.Path != "/echo" {
		t.Fatalf("Path = %s, want /echo", handler.lastReq.Path)
	}
	if handler.lastReq.RawQuery != "dry_run=true" {
		t.Fatalf("RawQuery = %s, want dry_run=true", handler.lastReq.RawQuery)
	}
	if got := handler.lastReq.Headers["Content-Type"][0]; got != "text/plain" {
		t.Fatalf("Content-Type = %s, want text/plain", got)
	}
	if string(handler.lastReq.Body) != "hello" {
		t.Fatalf("Body = %q, want hello", handler.lastReq.Body)
	}

	result := decodeResult[struct {
		StatusCode int                 `json:"status_code"`
		Headers    map[string][]string `json:"headers"`
		BodyBase64 string              `json:"body_base64"`
	}](t, output)

	if result.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", result.StatusCode, http.StatusCreated)
	}
	if result.Headers["X-Plugin"][0] != "example" {
		t.Fatalf("X-Plugin = %s, want example", result.Headers["X-Plugin"][0])
	}
	body, err := base64.StdEncoding.DecodeString(result.BodyBase64)
	if err != nil {
		t.Fatalf("decode body_base64: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %s, want {\"ok\":true}", body)
	}
}

func TestServeStreamMapsHandlerErrorToJSONRPCError(t *testing.T) {
	output := serveOne(t, `{"jsonrpc":"2.0","id":3,"method":"plugin.handle_http","params":{"method":"GET","path":"/echo","raw_query":"","headers":{},"body_base64":""}}`, &testHandler{httpErr: errors.New("boom")})

	if output.Error == nil {
		t.Fatal("Error = nil, want JSON-RPC error")
	}
	if output.Error.Code != -32000 {
		t.Fatalf("Error.Code = %d, want -32000", output.Error.Code)
	}
	if !strings.Contains(output.Error.Message, "boom") {
		t.Fatalf("Error.Message = %q, want boom", output.Error.Message)
	}
}

func TestServeStreamRejectsUnknownMethod(t *testing.T) {
	output := serveOne(t, `{"jsonrpc":"2.0","id":4,"method":"plugin.missing","params":{}}`, &testHandler{})

	if output.Error == nil {
		t.Fatal("Error = nil, want JSON-RPC error")
	}
	if output.Error.Code != -32601 {
		t.Fatalf("Error.Code = %d, want -32601", output.Error.Code)
	}
}

type rpcOutput struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func serveOne(t *testing.T, input string, handler pluginipc.Handler) rpcOutput {
	t.Helper()

	var output bytes.Buffer
	err := pluginipc.ServeStream(context.Background(), strings.NewReader(input+"\n"), &output, handler)
	if err != nil {
		t.Fatalf("ServeStream() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("output lines = %d, want 1: %q", len(lines), output.String())
	}

	var response rpcOutput
	if err := json.Unmarshal([]byte(lines[0]), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}

func decodeResult[T any](t *testing.T, output rpcOutput) T {
	t.Helper()

	if output.Error != nil {
		t.Fatalf("Error = %+v, want nil", output.Error)
	}

	var result T
	if err := json.Unmarshal(output.Result, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	return result
}
