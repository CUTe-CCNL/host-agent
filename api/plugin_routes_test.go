package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CUTe-CCNL/host-agent/config"
	agentplugin "github.com/CUTe-CCNL/host-agent/plugin"

	"github.com/gorilla/mux"
)

type fakePluginRegistry struct {
	infos          []agentplugin.Info
	restartID      string
	proxyID        string
	proxyPath      string
	proxyWasCalled bool
}

func (f *fakePluginRegistry) List() []agentplugin.Info {
	return f.infos
}

func (f *fakePluginRegistry) Get(id string) (agentplugin.Info, bool) {
	for _, info := range f.infos {
		if info.ID == id {
			return info, true
		}
	}
	return agentplugin.Info{}, false
}

func (f *fakePluginRegistry) Restart(ctx context.Context, id string) error {
	f.restartID = id
	return nil
}

func (f *fakePluginRegistry) ProxyHTTP(w http.ResponseWriter, r *http.Request, id, path string) {
	f.proxyWasCalled = true
	f.proxyID = id
	f.proxyPath = path
	w.WriteHeader(http.StatusAccepted)
}

func TestPluginRoutesRequireAuthWhenEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.Server.EnableAuth = true
	cfg.Server.AuthToken = "secret"

	registry := &fakePluginRegistry{
		infos: []agentplugin.Info{{ID: "firewall", Name: "Firewall", Status: agentplugin.StatusRunning, Transport: "stdio-jsonrpc"}},
	}
	router := mux.NewRouter()
	SetupRoutesWithPlugins(router, cfg, registry)

	req := httptest.NewRequest(http.MethodGet, "/plugins", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /plugins status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/plugins", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("authenticated /plugins status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPluginRoutesAllowRequestsWhenAuthDisabled(t *testing.T) {
	cfg := config.Default()
	registry := &fakePluginRegistry{
		infos: []agentplugin.Info{{ID: "firewall", Name: "Firewall", Status: agentplugin.StatusRunning, Transport: "stdio-jsonrpc"}},
	}
	router := mux.NewRouter()
	SetupRoutesWithPlugins(router, cfg, registry)

	req := httptest.NewRequest(http.MethodGet, "/plugins/firewall", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var info agentplugin.Info
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("decode info: %v", err)
	}
	if info.ID != "firewall" {
		t.Errorf("info.ID = %s, want firewall", info.ID)
	}
	if info.Transport != "stdio-jsonrpc" {
		t.Errorf("info.Transport = %s, want stdio-jsonrpc", info.Transport)
	}
}

func TestPluginRestartAndProxyRoutes(t *testing.T) {
	cfg := config.Default()
	registry := &fakePluginRegistry{
		infos: []agentplugin.Info{{ID: "firewall", Name: "Firewall", Status: agentplugin.StatusRunning, Transport: "stdio-jsonrpc"}},
	}
	router := mux.NewRouter()
	SetupRoutesWithPlugins(router, cfg, registry)

	req := httptest.NewRequest(http.MethodPost, "/plugins/firewall/restart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("restart status = %d, want %d", w.Code, http.StatusOK)
	}
	if registry.restartID != "firewall" {
		t.Errorf("restartID = %s, want firewall", registry.restartID)
	}

	req = httptest.NewRequest(http.MethodPatch, "/plugin-api/firewall/rules/ssh?dry_run=true", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("proxy status = %d, want %d", w.Code, http.StatusAccepted)
	}
	if !registry.proxyWasCalled {
		t.Fatal("proxy was not called")
	}
	if registry.proxyID != "firewall" {
		t.Errorf("proxyID = %s, want firewall", registry.proxyID)
	}
	if registry.proxyPath != "/rules/ssh" {
		t.Errorf("proxyPath = %s, want /rules/ssh", registry.proxyPath)
	}
}
