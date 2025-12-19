package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg == nil {
		t.Fatal("Default() returned nil")
	}

	// 檢查 Server 預設值
	if cfg.Server.Port != 9100 {
		t.Errorf("Server.Port = %d, want 9100", cfg.Server.Port)
	}

	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 15s", cfg.Server.ReadTimeout)
	}

	if cfg.Server.WriteTimeout != 15*time.Second {
		t.Errorf("Server.WriteTimeout = %v, want 15s", cfg.Server.WriteTimeout)
	}

	if cfg.Server.EnableAuth != false {
		t.Errorf("Server.EnableAuth = %v, want false", cfg.Server.EnableAuth)
	}

	// 檢查 Collector 預設值
	if cfg.Collector.Interval != 5*time.Second {
		t.Errorf("Collector.Interval = %v, want 5s", cfg.Collector.Interval)
	}

	if !cfg.Collector.EnableCPU {
		t.Error("Collector.EnableCPU should be true by default")
	}

	if !cfg.Collector.EnableMemory {
		t.Error("Collector.EnableMemory should be true by default")
	}

	if !cfg.Collector.EnableDisk {
		t.Error("Collector.EnableDisk should be true by default")
	}

	if !cfg.Collector.EnableNetwork {
		t.Error("Collector.EnableNetwork should be true by default")
	}

	if cfg.Collector.EnableProcess {
		t.Error("Collector.EnableProcess should be false by default")
	}

	if cfg.Collector.ProcessLimit != 10 {
		t.Errorf("Collector.ProcessLimit = %d, want 10", cfg.Collector.ProcessLimit)
	}

	// 檢查 Report 預設值
	if cfg.Report.Enabled {
		t.Error("Report.Enabled should be false by default")
	}

	if cfg.Report.Mode != "kafka" {
		t.Errorf("Report.Mode = %s, want kafka", cfg.Report.Mode)
	}

	if cfg.Report.Interval != 30*time.Second {
		t.Errorf("Report.Interval = %v, want 30s", cfg.Report.Interval)
	}

	// 檢查 Kafka 預設值
	if len(cfg.Report.Kafka.Brokers) != 1 || cfg.Report.Kafka.Brokers[0] != "localhost:9092" {
		t.Errorf("Report.Kafka.Brokers = %v, want [localhost:9092]", cfg.Report.Kafka.Brokers)
	}

	if cfg.Report.Kafka.Topic != "host-metrics" {
		t.Errorf("Report.Kafka.Topic = %s, want host-metrics", cfg.Report.Kafka.Topic)
	}

	if cfg.Report.Kafka.Compression != "gzip" {
		t.Errorf("Report.Kafka.Compression = %s, want gzip", cfg.Report.Kafka.Compression)
	}
}

func TestLoad(t *testing.T) {
	// 建立臨時配置檔
	content := `
server:
  port: 8080
  read_timeout: 10s
  write_timeout: 10s
  enable_auth: true
  auth_token: "test-token"

collector:
  interval: 10s
  enable_cpu: true
  enable_memory: true
  enable_disk: false
  enable_network: false
  enable_process: true
  process_limit: 5

report:
  enabled: true
  mode: "http"
  interval: 60s
  timeout: 5s
  http:
    endpoint: "http://example.com/metrics"
  kafka:
    brokers:
      - "kafka1:9092"
      - "kafka2:9092"
    topic: "test-metrics"
    compression: "snappy"
`

	// 建立臨時檔案
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// 載入配置
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 驗證值
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}

	if cfg.Server.AuthToken != "test-token" {
		t.Errorf("Server.AuthToken = %s, want test-token", cfg.Server.AuthToken)
	}

	if !cfg.Server.EnableAuth {
		t.Error("Server.EnableAuth should be true")
	}

	if cfg.Collector.Interval != 10*time.Second {
		t.Errorf("Collector.Interval = %v, want 10s", cfg.Collector.Interval)
	}

	if cfg.Collector.EnableDisk {
		t.Error("Collector.EnableDisk should be false")
	}

	if !cfg.Collector.EnableProcess {
		t.Error("Collector.EnableProcess should be true")
	}

	if cfg.Collector.ProcessLimit != 5 {
		t.Errorf("Collector.ProcessLimit = %d, want 5", cfg.Collector.ProcessLimit)
	}

	if cfg.Report.Mode != "http" {
		t.Errorf("Report.Mode = %s, want http", cfg.Report.Mode)
	}

	if cfg.Report.HTTP.Endpoint != "http://example.com/metrics" {
		t.Errorf("Report.HTTP.Endpoint = %s, want http://example.com/metrics", cfg.Report.HTTP.Endpoint)
	}

	if len(cfg.Report.Kafka.Brokers) != 2 {
		t.Errorf("Report.Kafka.Brokers length = %d, want 2", len(cfg.Report.Kafka.Brokers))
	}

	if cfg.Report.Kafka.Topic != "test-metrics" {
		t.Errorf("Report.Kafka.Topic = %s, want test-metrics", cfg.Report.Kafka.Topic)
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load() should return error for non-existent file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	// 建立無效的 YAML 檔案
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid YAML")
	}
}
