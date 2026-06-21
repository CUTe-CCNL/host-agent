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

	if cfg.Report.Mode != "rabbitmq" {
		t.Errorf("Report.Mode = %s, want rabbitmq", cfg.Report.Mode)
	}

	if cfg.Report.Interval != 30*time.Second {
		t.Errorf("Report.Interval = %v, want 30s", cfg.Report.Interval)
	}

	// 檢查 RabbitMQ 預設值
	if cfg.Report.RabbitMQ.URL != "amqp://guest:guest@localhost:5672/" {
		t.Errorf("Report.RabbitMQ.URL = %s, want amqp://guest:guest@localhost:5672/", cfg.Report.RabbitMQ.URL)
	}

	if cfg.Report.RabbitMQ.Exchange != "host-metrics" {
		t.Errorf("Report.RabbitMQ.Exchange = %s, want host-metrics", cfg.Report.RabbitMQ.Exchange)
	}

	if cfg.Report.RabbitMQ.ExchangeType != "topic" {
		t.Errorf("Report.RabbitMQ.ExchangeType = %s, want topic", cfg.Report.RabbitMQ.ExchangeType)
	}

	if cfg.Report.RabbitMQ.RoutingKeyTemplate != "host.metrics" {
		t.Errorf("Report.RabbitMQ.RoutingKeyTemplate = %s, want host.metrics", cfg.Report.RabbitMQ.RoutingKeyTemplate)
	}

	if !cfg.Report.RabbitMQ.Durable {
		t.Error("Report.RabbitMQ.Durable should be true by default")
	}

	if cfg.Report.RabbitMQ.AutoDelete {
		t.Error("Report.RabbitMQ.AutoDelete should be false by default")
	}

	// 檢查 Plugins 預設值
	if cfg.Plugins.Enabled {
		t.Error("Plugins.Enabled should be false by default")
	}

	if cfg.Plugins.Directory != "/etc/host-agent/plugins.d" {
		t.Errorf("Plugins.Directory = %s, want /etc/host-agent/plugins.d", cfg.Plugins.Directory)
	}

	if cfg.Plugins.StartupTimeout != 10*time.Second {
		t.Errorf("Plugins.StartupTimeout = %v, want 10s", cfg.Plugins.StartupTimeout)
	}

	if cfg.Plugins.HealthInterval != 15*time.Second {
		t.Errorf("Plugins.HealthInterval = %v, want 15s", cfg.Plugins.HealthInterval)
	}

	if cfg.Plugins.RequestTimeout != 30*time.Second {
		t.Errorf("Plugins.RequestTimeout = %v, want 30s", cfg.Plugins.RequestTimeout)
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
  rabbitmq:
    url: "amqp://test:test@rabbitmq:5672/"
    exchange: "test-metrics"
    exchange_type: "fanout"
    routing_key_template: "host.metrics.{hostname}"
    durable: false
    auto_delete: true

plugins:
  enabled: true
  directory: "/tmp/host-agent/plugins.d"
  startup_timeout: 2s
  health_interval: 3s
  request_timeout: 4s
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

	if cfg.Report.RabbitMQ.URL != "amqp://test:test@rabbitmq:5672/" {
		t.Errorf("Report.RabbitMQ.URL = %s, want amqp://test:test@rabbitmq:5672/", cfg.Report.RabbitMQ.URL)
	}

	if cfg.Report.RabbitMQ.Exchange != "test-metrics" {
		t.Errorf("Report.RabbitMQ.Exchange = %s, want test-metrics", cfg.Report.RabbitMQ.Exchange)
	}

	if cfg.Report.RabbitMQ.ExchangeType != "fanout" {
		t.Errorf("Report.RabbitMQ.ExchangeType = %s, want fanout", cfg.Report.RabbitMQ.ExchangeType)
	}

	if cfg.Report.RabbitMQ.RoutingKeyTemplate != "host.metrics.{hostname}" {
		t.Errorf("Report.RabbitMQ.RoutingKeyTemplate = %s, want host.metrics.{hostname}", cfg.Report.RabbitMQ.RoutingKeyTemplate)
	}

	if cfg.Report.RabbitMQ.Durable {
		t.Error("Report.RabbitMQ.Durable should be false")
	}

	if !cfg.Report.RabbitMQ.AutoDelete {
		t.Error("Report.RabbitMQ.AutoDelete should be true")
	}

	if !cfg.Plugins.Enabled {
		t.Error("Plugins.Enabled should be true")
	}

	if cfg.Plugins.Directory != "/tmp/host-agent/plugins.d" {
		t.Errorf("Plugins.Directory = %s, want /tmp/host-agent/plugins.d", cfg.Plugins.Directory)
	}

	if cfg.Plugins.StartupTimeout != 2*time.Second {
		t.Errorf("Plugins.StartupTimeout = %v, want 2s", cfg.Plugins.StartupTimeout)
	}

	if cfg.Plugins.HealthInterval != 3*time.Second {
		t.Errorf("Plugins.HealthInterval = %v, want 3s", cfg.Plugins.HealthInterval)
	}

	if cfg.Plugins.RequestTimeout != 4*time.Second {
		t.Errorf("Plugins.RequestTimeout = %v, want 4s", cfg.Plugins.RequestTimeout)
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
