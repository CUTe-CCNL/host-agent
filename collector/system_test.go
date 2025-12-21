package collector

import (
	"testing"
	"time"
)

func TestCollectSystemMetrics(t *testing.T) {
	metrics, err := CollectSystemMetrics()
	if err != nil {
		t.Fatalf("CollectSystemMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Fatal("CollectSystemMetrics() returned nil")
	}

	// 檢查 HostID 不為空
	if metrics.HostID == "" {
		t.Error("HostID should not be empty")
	}

	// 檢查 OS 不為空
	if metrics.OS == "" {
		t.Error("OS should not be empty")
	}

	// 檢查架構不為空
	if metrics.Architecture == "" {
		t.Error("Architecture should not be empty")
	}

	// 檢查 Uptime 大於 0
	if metrics.Uptime == 0 {
		t.Error("Uptime should be greater than 0")
	}

	// 檢查 BootTime 在合理範圍內
	if metrics.BootTime.After(time.Now()) {
		t.Error("BootTime should not be in the future")
	}

	if metrics.BootTime.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("BootTime seems too old")
	}
}

func TestGetHostname(t *testing.T) {
	hostname := GetHostname()

	// 檢查 hostname 不為空且不是 "unknown"
	if hostname == "" {
		t.Error("Hostname should not be empty")
	}

	// 即使取得失敗也應該返回 "unknown"
	if hostname == "" {
		t.Error("Hostname should return 'unknown' on error, not empty string")
	}
}

func BenchmarkCollectSystemMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = CollectSystemMetrics()
	}
}

func BenchmarkGetHostname(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetHostname()
	}
}
