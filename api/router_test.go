package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CUTe-CCNL/host-agent/config"

	"github.com/gorilla/mux"
)

func TestSetupRoutes(t *testing.T) {
	cfg := config.Default()
	router := mux.NewRouter()

	SetupRoutes(router, cfg)

	// 測試各個端點是否正確註冊
	routes := []struct {
		path   string
		method string
	}{
		{"/health", http.MethodGet},
		{"/metrics", http.MethodGet},
		{"/metrics/cpu", http.MethodGet},
		{"/metrics/memory", http.MethodGet},
		{"/metrics/disk", http.MethodGet},
		{"/metrics/network", http.MethodGet},
		{"/metrics/process", http.MethodGet},
	}

	for _, route := range routes {
		t.Run(route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 只要不是 404 就表示路由已註冊
			if w.Code == http.StatusNotFound {
				t.Errorf("Route %s %s not found", route.method, route.path)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	cfg := config.Default()
	router := mux.NewRouter()

	SetupRoutes(router, cfg)

	// 測試 CORS 標頭
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		_ = resp.Body.Close()
	}()

	if cors := resp.Header.Get("Access-Control-Allow-Origin"); cors != "*" {
		t.Errorf("Access-Control-Allow-Origin = %s, want *", cors)
	}

	if methods := resp.Header.Get("Access-Control-Allow-Methods"); methods != "GET, POST, PUT, PATCH, DELETE, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %s, want 'GET, POST, PUT, PATCH, DELETE, OPTIONS'", methods)
	}
}

func TestCORSPreflight(t *testing.T) {
	cfg := config.Default()
	router := mux.NewRouter()

	SetupRoutes(router, cfg)

	// 測試 OPTIONS 預檢請求
	req := httptest.NewRequest(http.MethodOptions, "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d for OPTIONS preflight", resp.StatusCode, http.StatusOK)
	}
}

func TestNotFoundRoute(t *testing.T) {
	cfg := config.Default()
	router := mux.NewRouter()

	SetupRoutes(router, cfg)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d for nonexistent route", w.Code, http.StatusNotFound)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	cfg := config.Default()
	router := mux.NewRouter()

	SetupRoutes(router, cfg)

	// 測試 POST 到只接受 GET 的端點
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Gorilla mux 對於未註冊的方法返回 405 或 404
	if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d or %d for wrong method", w.Code, http.StatusMethodNotAllowed, http.StatusNotFound)
	}
}
