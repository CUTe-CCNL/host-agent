package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"host-agent/config"

	"github.com/gorilla/mux"
)

func TestNewServer(t *testing.T) {
	cfg := config.Default()
	router := mux.NewRouter()

	server := NewServer(cfg, router)

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	if server.config != cfg {
		t.Error("Server config not set correctly")
	}

	if server.httpServer == nil {
		t.Error("HTTP server should not be nil")
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	cfg := config.Default()
	cfg.Server.Port = 19100 // 使用不同的端口避免衝突

	router := mux.NewRouter()
	SetupRoutes(router, cfg)

	server := NewServer(cfg, router)

	// 啟動伺服器
	errChan := make(chan error, 1)
	go func() {
		err := server.Start()
		if err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// 等待伺服器啟動
	time.Sleep(100 * time.Millisecond)

	// 測試健康檢查
	resp, err := http.Get("http://localhost:19100/health")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// 優雅關閉
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// 檢查是否有啟動錯誤
	select {
	case err := <-errChan:
		t.Errorf("Server error: %v", err)
	default:
	}
}

func TestServerTimeout(t *testing.T) {
	cfg := config.Default()
	cfg.Server.ReadTimeout = 1 * time.Second
	cfg.Server.WriteTimeout = 1 * time.Second

	router := mux.NewRouter()
	server := NewServer(cfg, router)

	if server.httpServer.ReadTimeout != 1*time.Second {
		t.Errorf("ReadTimeout = %v, want 1s", server.httpServer.ReadTimeout)
	}

	if server.httpServer.WriteTimeout != 1*time.Second {
		t.Errorf("WriteTimeout = %v, want 1s", server.httpServer.WriteTimeout)
	}
}
