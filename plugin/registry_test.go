package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"host-agent/config"
)

func TestPluginHelperProcess(t *testing.T) {
	mode := os.Getenv("HOST_AGENT_PLUGIN_HELPER")
	if mode == "" {
		return
	}

	addr := ""
	for i, arg := range os.Args {
		if arg == "--addr" && i+1 < len(os.Args) {
			addr = os.Args[i+1]
			break
		}
	}
	if addr == "" {
		fmt.Fprintln(os.Stderr, "missing --addr")
		os.Exit(2)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(2)
	}

	server := &http.Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/echo/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Upstream", "plugin")
		_, _ = fmt.Fprintf(w, "%s %s %s %s", r.Method, r.URL.Path, r.URL.RawQuery, string(body))
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte("slow"))
	})
	mux.HandleFunc("/close", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("closing"))
		go func() {
			time.Sleep(10 * time.Millisecond)
			_ = server.Close()
		}()
	})
	mux.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("bye"))
		go func() {
			time.Sleep(10 * time.Millisecond)
			os.Exit(0)
		}()
	})
	server.Handler = mux

	if mode == "crash" {
		os.Exit(0)
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(2)
	}
	select {}
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on free port: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	return listener.Addr().String()
}

func testRegistryConfig() config.PluginConfig {
	return config.PluginConfig{
		StartupTimeout: 2 * time.Second,
		HealthInterval: time.Hour,
		RequestTimeout: 100 * time.Millisecond,
	}
}

func testManifest(id, addr string, methods []string) Manifest {
	return Manifest{
		ID:          id,
		Name:        "Test Plugin",
		Version:     "0.1.0",
		Enabled:     true,
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestPluginHelperProcess", "--", "--addr", addr},
		Env:         map[string]string{"HOST_AGENT_PLUGIN_HELPER": "serve"},
		UpstreamURL: "http://" + addr,
		HealthPath:  "/health",
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

	deadline := time.Now().Add(2 * time.Second)
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

func TestRegistryLifecycleStartCrashAndRestart(t *testing.T) {
	addr := freeTCPAddr(t)
	registry := startTestRegistry(t, testManifest("echo", addr, []string{http.MethodGet}))

	info, ok := registry.Get("echo")
	if !ok {
		t.Fatal("Get(echo) ok = false, want true")
	}
	if info.Status != StatusRunning {
		t.Fatalf("Status = %s, want %s", info.Status, StatusRunning)
	}

	resp, err := http.Get("http://" + addr + "/exit")
	if err != nil {
		t.Fatalf("exit request error = %v", err)
	}
	_ = resp.Body.Close()

	eventuallyStatus(t, registry, "echo", StatusFailed)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := registry.Restart(ctx, "echo"); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}

	eventuallyStatus(t, registry, "echo", StatusRunning)
}

func TestProxyPreservesRequestAndResponse(t *testing.T) {
	addr := freeTCPAddr(t)
	registry := startTestRegistry(t, testManifest("echo", addr, []string{http.MethodPatch}))

	req := httptestRequest(http.MethodPatch, "/plugin-api/echo/echo/path?debug=true", "payload")
	w := httptestRecorder()

	registry.ProxyHTTP(w, req, "echo", "/echo/path")

	resp := w.Result()
	defer func() {
		_ = resp.Body.Close()
	}()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d: %s", resp.StatusCode, http.StatusOK, body)
	}
	if resp.Header.Get("X-Upstream") != "plugin" {
		t.Errorf("X-Upstream = %s, want plugin", resp.Header.Get("X-Upstream"))
	}
	if got, want := strings.TrimSpace(string(body)), "PATCH /echo/path debug=true payload"; got != want {
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

	addr := freeTCPAddr(t)
	if err := registry.Register([]Manifest{testManifest("echo", addr, []string{http.MethodGet})}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	req = httptestRequest(http.MethodGet, "/plugin-api/echo/echo", "")
	w = httptestRecorder()
	registry.ProxyHTTP(w, req, "echo", "/echo")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("stopped plugin status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	registry = startTestRegistry(t, testManifest("echo", freeTCPAddr(t), []string{http.MethodGet}))
	req = httptestRequest(http.MethodPost, "/plugin-api/echo/echo", "")
	w = httptestRecorder()
	registry.ProxyHTTP(w, req, "echo", "/echo")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("disallowed method status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestProxyMapsUpstreamErrors(t *testing.T) {
	addr := freeTCPAddr(t)
	registry := startTestRegistry(t, testManifest("echo", addr, []string{http.MethodGet}))

	resp, err := http.Get("http://" + addr + "/close")
	if err != nil {
		t.Fatalf("close request error = %v", err)
	}
	_ = resp.Body.Close()
	time.Sleep(50 * time.Millisecond)

	req := httptestRequest(http.MethodGet, "/plugin-api/echo/echo", "")
	w := httptestRecorder()
	registry.ProxyHTTP(w, req, "echo", "/echo")
	if w.Code != http.StatusBadGateway {
		t.Fatalf("closed upstream status = %d, want %d", w.Code, http.StatusBadGateway)
	}

	addr = freeTCPAddr(t)
	registry = startTestRegistry(t, testManifest("slow", addr, []string{http.MethodGet}))
	req = httptestRequest(http.MethodGet, "/plugin-api/slow/slow", "")
	w = httptestRecorder()
	registry.ProxyHTTP(w, req, "slow", "/slow")
	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("slow upstream status = %d, want %d", w.Code, http.StatusGatewayTimeout)
	}
}

func httptestRequest(method, target, body string) *http.Request {
	return httptest.NewRequest(method, target, bytes.NewBufferString(body))
}

func httptestRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}
