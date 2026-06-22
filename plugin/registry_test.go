package plugin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"host-agent/config"
)

type testRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type testRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *testRPCError   `json:"error,omitempty"`
}

type testRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type testHTTPParams struct {
	Method     string              `json:"method"`
	Path       string              `json:"path"`
	RawQuery   string              `json:"raw_query"`
	Headers    map[string][]string `json:"headers"`
	BodyBase64 string              `json:"body_base64"`
}

type testHTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	BodyBase64 string              `json:"body_base64"`
}

func TestPluginHelperProcess(t *testing.T) {
	mode := os.Getenv("HOST_AGENT_PLUGIN_HELPER")
	if mode == "" {
		return
	}

	runStdioPluginHelper(mode)
	os.Exit(0)
}

func runStdioPluginHelper(mode string) {
	if mode == "crash" {
		os.Exit(0)
	}

	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	encoder := json.NewEncoder(writer)
	writeResponse := func(response testRPCResponse) {
		if err := encoder.Encode(response); err != nil {
			os.Exit(3)
		}
		if err := writer.Flush(); err != nil {
			os.Exit(3)
		}
	}

	for scanner.Scan() {
		var request testRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(3)
		}

		switch request.Method {
		case "plugin.health":
			if mode == "hang-health" {
				select {}
			}
			writeResponse(testRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Result:  map[string]bool{"ok": true},
			})

		case "plugin.handle_http":
			var params testHTTPParams
			if err := json.Unmarshal(request.Params, &params); err != nil {
				writeResponse(testRPCResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Error:   &testRPCError{Code: -32602, Message: err.Error()},
				})
				continue
			}

			switch params.Path {
			case "/error":
				writeResponse(testRPCResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Error:   &testRPCError{Code: -32000, Message: "plugin failed"},
				})
			case "/invalid":
				_, _ = fmt.Fprintln(os.Stdout, "not json")
			case "/slow":
				time.Sleep(300 * time.Millisecond)
				writeResponse(testRPCResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Result:  echoHTTPResponse(params),
				})
			case "/exit":
				os.Exit(0)
			default:
				writeResponse(testRPCResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Result:  echoHTTPResponse(params),
				})
			}

		default:
			writeResponse(testRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error:   &testRPCError{Code: -32601, Message: "method not found"},
			})
		}
	}
}

func echoHTTPResponse(params testHTTPParams) testHTTPResponse {
	body, _ := base64.StdEncoding.DecodeString(params.BodyBase64)

	contentType := ""
	if values := params.Headers["Content-Type"]; len(values) > 0 {
		contentType = values[0]
	}
	if _, ok := params.Headers["Connection"]; ok {
		contentType += " leaked-connection"
	}

	responseBody := fmt.Sprintf("%s %s %s %s %s", params.Method, params.Path, params.RawQuery, contentType, string(body))
	return testHTTPResponse{
		StatusCode: http.StatusCreated,
		Headers: map[string][]string{
			"Content-Type": {"text/plain"},
			"Connection":   {"close"},
			"X-Plugin":     {"stdio"},
		},
		BodyBase64: base64.StdEncoding.EncodeToString([]byte(responseBody)),
	}
}

func testRegistryConfig() config.PluginConfig {
	return config.PluginConfig{
		StartupTimeout: 500 * time.Millisecond,
		HealthInterval: time.Hour,
		RequestTimeout: 100 * time.Millisecond,
	}
}

func testManifest(id string, methods []string, mode string) Manifest {
	return Manifest{
		ID:      id,
		Name:    "Test Plugin",
		Version: "0.1.0",
		Enabled: true,
		Command: os.Args[0],
		Args:    []string{"-test.run=TestPluginHelperProcess"},
		Env:     map[string]string{"HOST_AGENT_PLUGIN_HELPER": mode},
		Routes: []Route{
			{PathPrefix: "/", Methods: methods},
		},
	}
}

func startTestRegistry(t *testing.T, manifest Manifest) *Registry {
	t.Helper()

	registry := NewRegistry(testRegistryConfig())
	if err := registry.Register([]Manifest{manifest}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := registry.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = registry.Stop(ctx)
	})

	return registry
}

func eventuallyStatus(t *testing.T, registry *Registry, id string, want Status) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		info, ok := registry.Get(id)
		if ok && info.Status == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	info, ok := registry.Get(id)
	t.Fatalf("status = %v (ok=%v), want %v", info.Status, ok, want)
}

func TestRegistryLifecycleStartExitAndRestart(t *testing.T) {
	registry := startTestRegistry(t, testManifest("echo", []string{http.MethodGet}, "serve"))

	info, ok := registry.Get("echo")
	if !ok {
		t.Fatal("Get(echo) ok = false, want true")
	}
	if info.Status != StatusRunning {
		t.Fatalf("Status = %s, want %s", info.Status, StatusRunning)
	}
	if info.Transport != "stdio-jsonrpc" {
		t.Fatalf("Transport = %s, want stdio-jsonrpc", info.Transport)
	}

	req := httptestRequest(http.MethodGet, "/plugin-api/echo/exit", "")
	w := httptestRecorder()
	registry.ProxyHTTP(w, req, "echo", "/exit")
	if w.Code != http.StatusBadGateway {
		t.Fatalf("exit proxy status = %d, want %d", w.Code, http.StatusBadGateway)
	}
	eventuallyStatus(t, registry, "echo", StatusFailed)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := registry.Restart(ctx, "echo"); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}

	eventuallyStatus(t, registry, "echo", StatusRunning)
}

func TestRegistryMarksHealthTimeoutFailed(t *testing.T) {
	registry := NewRegistry(testRegistryConfig())
	if err := registry.Register([]Manifest{testManifest("hung", []string{http.MethodGet}, "hang-health")}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := registry.Start(ctx); err == nil {
		t.Fatal("Start() error = nil, want health timeout")
	}

	info, ok := registry.Get("hung")
	if !ok {
		t.Fatal("Get(hung) ok = false, want true")
	}
	if info.Status != StatusFailed {
		t.Fatalf("Status = %s, want %s", info.Status, StatusFailed)
	}
}

func TestProxyPreservesRequestAndResponse(t *testing.T) {
	registry := startTestRegistry(t, testManifest("echo", []string{http.MethodPatch}, "serve"))

	req := httptestRequest(http.MethodPatch, "/plugin-api/echo/echo/path?debug=true", "payload")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	w := httptestRecorder()

	registry.ProxyHTTP(w, req, "echo", "/echo/path")

	resp := w.Result()
	defer func() {
		_ = resp.Body.Close()
	}()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d: %s", resp.StatusCode, http.StatusCreated, body)
	}
	if resp.Header.Get("X-Plugin") != "stdio" {
		t.Errorf("X-Plugin = %s, want stdio", resp.Header.Get("X-Plugin"))
	}
	if resp.Header.Get("Connection") != "" {
		t.Errorf("Connection response header = %s, want empty", resp.Header.Get("Connection"))
	}
	if got, want := strings.TrimSpace(string(body)), "PATCH /echo/path debug=true application/json payload"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestProxyRejectsUnknownStoppedAndDisallowedMethod(t *testing.T) {
	registry := NewRegistry(testRegistryConfig())

	req := httptestRequest(http.MethodGet, "/plugin-api/missing/echo", "")
	w := httptestRecorder()
	registry.ProxyHTTP(w, req, "missing", "/echo")
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown plugin status = %d, want %d", w.Code, http.StatusNotFound)
	}

	if err := registry.Register([]Manifest{testManifest("echo", []string{http.MethodGet}, "serve")}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	req = httptestRequest(http.MethodGet, "/plugin-api/echo/echo", "")
	w = httptestRecorder()
	registry.ProxyHTTP(w, req, "echo", "/echo")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("stopped plugin status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	registry = startTestRegistry(t, testManifest("echo", []string{http.MethodGet}, "serve"))
	req = httptestRequest(http.MethodPost, "/plugin-api/echo/echo", "")
	w = httptestRecorder()
	registry.ProxyHTTP(w, req, "echo", "/echo")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("disallowed method status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestProxyMapsIPCFailures(t *testing.T) {
	registry := startTestRegistry(t, testManifest("error", []string{http.MethodGet}, "serve"))
	req := httptestRequest(http.MethodGet, "/plugin-api/error/error", "")
	w := httptestRecorder()
	registry.ProxyHTTP(w, req, "error", "/error")
	if w.Code != http.StatusBadGateway {
		t.Fatalf("json-rpc error status = %d, want %d", w.Code, http.StatusBadGateway)
	}

	registry = startTestRegistry(t, testManifest("invalid", []string{http.MethodGet}, "serve"))
	req = httptestRequest(http.MethodGet, "/plugin-api/invalid/invalid", "")
	w = httptestRecorder()
	registry.ProxyHTTP(w, req, "invalid", "/invalid")
	if w.Code != http.StatusBadGateway {
		t.Fatalf("invalid json-rpc response status = %d, want %d", w.Code, http.StatusBadGateway)
	}

	registry = startTestRegistry(t, testManifest("slow", []string{http.MethodGet}, "serve"))
	req = httptestRequest(http.MethodGet, "/plugin-api/slow/slow", "")
	w = httptestRecorder()
	registry.ProxyHTTP(w, req, "slow", "/slow")
	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("slow plugin status = %d, want %d", w.Code, http.StatusGatewayTimeout)
	}
}

func httptestRequest(method, target, body string) *http.Request {
	return httptest.NewRequest(method, target, bytes.NewBufferString(body))
}

func httptestRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}
