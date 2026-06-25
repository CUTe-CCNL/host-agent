package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CUTe-CCNL/host-agent/config"
	"github.com/CUTe-CCNL/host-agent/models"
)

func newTestConfig() *config.Config {
	cfg := config.Default()
	cfg.Collector.EnableCPU = true
	cfg.Collector.EnableMemory = true
	cfg.Collector.EnableDisk = true
	cfg.Collector.EnableNetwork = true
	cfg.Collector.EnableProcess = false
	return cfg
}

func TestHealthCheck(t *testing.T) {
	handler := NewHandler(newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.HealthCheck(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", contentType)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("status = %s, want ok", result["status"])
	}

	if result["time"] == "" {
		t.Error("time should not be empty")
	}
}

func TestGetMetrics(t *testing.T) {
	handler := NewHandler(newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.GetMetrics(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var metrics models.Metrics
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if metrics.Hostname == "" {
		t.Error("Hostname should not be empty")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if metrics.CPU == nil {
		t.Error("CPU metrics should not be nil")
	}

	if metrics.Memory == nil {
		t.Error("Memory metrics should not be nil")
	}
}

func TestGetMetricsWithAuth(t *testing.T) {
	cfg := newTestConfig()
	cfg.Server.EnableAuth = true
	cfg.Server.AuthToken = "test-secret-token"

	handler := NewHandler(cfg)

	// 測試無認證
	t.Run("NoAuth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.GetMetrics(w, req)

		resp := w.Result()
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
	})

	// 測試錯誤認證
	t.Run("WrongAuth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("Authorization", "Bearer wrong-token")
		w := httptest.NewRecorder()

		handler.GetMetrics(w, req)

		resp := w.Result()
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
	})

	// 測試正確認證
	t.Run("CorrectAuth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("Authorization", "Bearer test-secret-token")
		w := httptest.NewRecorder()

		handler.GetMetrics(w, req)

		resp := w.Result()
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})
}

func TestGetCPUMetrics(t *testing.T) {
	handler := NewHandler(newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/metrics/cpu", nil)
	w := httptest.NewRecorder()

	handler.GetCPUMetrics(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var cpu models.CPUMetrics
	if err := json.NewDecoder(resp.Body).Decode(&cpu); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if cpu.Cores <= 0 {
		t.Errorf("Cores = %d, want > 0", cpu.Cores)
	}
}

func TestGetMemoryMetrics(t *testing.T) {
	handler := NewHandler(newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/metrics/memory", nil)
	w := httptest.NewRecorder()

	handler.GetMemoryMetrics(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var memory models.MemoryMetrics
	if err := json.NewDecoder(resp.Body).Decode(&memory); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if memory.Total == 0 {
		t.Error("Total should be greater than 0")
	}
}

func TestGetDiskMetrics(t *testing.T) {
	handler := NewHandler(newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/metrics/disk", nil)
	w := httptest.NewRecorder()

	handler.GetDiskMetrics(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var disks []models.DiskMetrics
	if err := json.NewDecoder(resp.Body).Decode(&disks); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestGetNetworkMetrics(t *testing.T) {
	handler := NewHandler(newTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/metrics/network", nil)
	w := httptest.NewRecorder()

	handler.GetNetworkMetrics(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var networks []models.NetworkMetrics
	if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestGetProcessMetrics(t *testing.T) {
	cfg := newTestConfig()
	cfg.Collector.ProcessLimit = 5
	handler := NewHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics/process", nil)
	w := httptest.NewRecorder()

	handler.GetProcessMetrics(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var processes []models.ProcessMetrics
	if err := json.NewDecoder(resp.Body).Decode(&processes); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(processes) > cfg.Collector.ProcessLimit {
		t.Errorf("len(processes) = %d, want <= %d", len(processes), cfg.Collector.ProcessLimit)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	handler := NewHandler(newTestConfig())

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		handler.HealthCheck(w, req)
	}
}

func BenchmarkGetMetrics(b *testing.B) {
	handler := NewHandler(newTestConfig())

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		handler.GetMetrics(w, req)
	}
}
