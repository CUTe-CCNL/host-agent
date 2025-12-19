package reporter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"host-agent/config"
	"host-agent/models"
)

func newTestConfig() *config.Config {
	cfg := config.Default()
	cfg.Report.Enabled = true
	cfg.Report.Mode = "http"
	cfg.Report.Interval = 100 * time.Millisecond
	cfg.Report.Timeout = 5 * time.Second
	return cfg
}

func TestNewReporter(t *testing.T) {
	cfg := newTestConfig()

	reporter := NewReporter(cfg)
	if reporter == nil {
		t.Fatal("NewReporter() returned nil")
	}

	if reporter.config != cfg {
		t.Error("Reporter config not set correctly")
	}

	if reporter.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}

	if reporter.stop == nil {
		t.Error("Stop channel should not be nil")
	}
}

func TestReporterCollectMetrics(t *testing.T) {
	cfg := newTestConfig()
	cfg.Collector.EnableCPU = true
	cfg.Collector.EnableMemory = true
	cfg.Collector.EnableDisk = true
	cfg.Collector.EnableNetwork = true

	reporter := NewReporter(cfg)
	metrics := reporter.collectMetrics()

	if metrics == nil {
		t.Fatal("collectMetrics() returned nil")
	}

	if metrics.Hostname == "" {
		t.Error("Hostname should not be empty")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if metrics.CPU == nil {
		t.Error("CPU metrics should not be nil when enabled")
	}

	if metrics.Memory == nil {
		t.Error("Memory metrics should not be nil when enabled")
	}

	if metrics.System == nil {
		t.Error("System metrics should not be nil")
	}
}

func TestReporterSendHTTP(t *testing.T) {
	var receivedMetrics *models.Metrics
	var requestCount int32

	// 建立測試伺服器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}

		var metrics models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		receivedMetrics = &metrics

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := newTestConfig()
	cfg.Report.HTTP.Endpoint = server.URL

	reporter := NewReporter(cfg)
	metrics := reporter.collectMetrics()

	reporter.sendHTTP(metrics)

	// 等待請求處理
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("Request count = %d, want 1", requestCount)
	}

	if receivedMetrics == nil {
		t.Fatal("Server did not receive metrics")
	}

	if receivedMetrics.Hostname != metrics.Hostname {
		t.Errorf("Received hostname = %s, want %s", receivedMetrics.Hostname, metrics.Hostname)
	}
}

func TestReporterSendHTTPError(t *testing.T) {
	// 測試伺服器返回錯誤
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := newTestConfig()
	cfg.Report.HTTP.Endpoint = server.URL

	reporter := NewReporter(cfg)
	metrics := reporter.collectMetrics()

	// 不應該 panic
	reporter.sendHTTP(metrics)
}

func TestReporterSendHTTPTimeout(t *testing.T) {
	// 測試超時
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := newTestConfig()
	cfg.Report.HTTP.Endpoint = server.URL
	cfg.Report.Timeout = 100 * time.Millisecond

	reporter := NewReporter(cfg)
	metrics := reporter.collectMetrics()

	// 不應該 panic，應該超時後返回
	reporter.sendHTTP(metrics)
}

func TestReporterSendHTTPEmptyEndpoint(t *testing.T) {
	cfg := newTestConfig()
	cfg.Report.HTTP.Endpoint = ""

	reporter := NewReporter(cfg)
	metrics := reporter.collectMetrics()

	// 空 endpoint 不應該 panic
	reporter.sendHTTP(metrics)
}

func TestReporterStartStop(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := newTestConfig()
	cfg.Report.HTTP.Endpoint = server.URL
	cfg.Report.Interval = 50 * time.Millisecond

	reporter := NewReporter(cfg)

	// 啟動 reporter
	go reporter.Start()

	// 等待第一次報告完成（Start() 會立即執行一次 report()）
	// 增加等待時間以確保 goroutine 有時間執行
	time.Sleep(300 * time.Millisecond)

	// 停止 reporter
	reporter.Stop()

	// 等待停止完成
	time.Sleep(100 * time.Millisecond)

	// 應該至少發送過一次
	if atomic.LoadInt32(&requestCount) < 1 {
		t.Errorf("Request count = %d, want >= 1", requestCount)
	}
}

func TestReporterModeHTTP(t *testing.T) {
	var httpCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&httpCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := newTestConfig()
	cfg.Report.Mode = "http"
	cfg.Report.HTTP.Endpoint = server.URL

	reporter := NewReporter(cfg)
	reporter.report()

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&httpCount) != 1 {
		t.Errorf("HTTP request count = %d, want 1", httpCount)
	}
}

func TestReporterModeKafkaFallback(t *testing.T) {
	// 測試 Kafka 連線失敗時降級到 HTTP
	cfg := newTestConfig()
	cfg.Report.Mode = "kafka"
	cfg.Report.Kafka.Brokers = []string{"nonexistent:9092"}

	reporter := NewReporter(cfg)

	// 由於 Kafka 連線失敗，應該降級到 HTTP
	if reporter.kafkaProducer != nil {
		t.Error("Kafka producer should be nil when connection fails")
	}
}

func BenchmarkCollectMetrics(b *testing.B) {
	cfg := newTestConfig()
	reporter := NewReporter(cfg)

	for i := 0; i < b.N; i++ {
		_ = reporter.collectMetrics()
	}
}
