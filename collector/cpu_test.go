package collector

import (
	"runtime"
	"testing"
)

func TestCollectCPUMetrics(t *testing.T) {
	metrics, err := CollectCPUMetrics()
	if err != nil {
		t.Fatalf("CollectCPUMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Fatal("CollectCPUMetrics() returned nil")
	}

	// 檢查 CPU 使用率範圍
	if metrics.UsagePercent < 0 || metrics.UsagePercent > 100 {
		t.Errorf("UsagePercent = %v, want between 0 and 100", metrics.UsagePercent)
	}

	// 檢查核心數
	expectedCores := runtime.NumCPU()
	if metrics.Cores != expectedCores {
		t.Errorf("Cores = %v, want %v", metrics.Cores, expectedCores)
	}

	// 檢查每核心使用率（如果有）
	if metrics.PerCore != nil {
		for i, usage := range metrics.PerCore {
			if usage < 0 || usage > 100 {
				t.Errorf("PerCore[%d] = %v, want between 0 and 100", i, usage)
			}
		}
	}
}

func BenchmarkCollectCPUMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = CollectCPUMetrics()
	}
}
