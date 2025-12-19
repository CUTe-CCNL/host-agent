package collector

import (
	"testing"
)

func TestCollectProcessMetrics(t *testing.T) {
	limit := 10
	metrics, err := CollectProcessMetrics(limit)
	if err != nil {
		t.Fatalf("CollectProcessMetrics() error = %v", err)
	}

	// 至少應該有一個進程
	if len(metrics) == 0 {
		t.Skip("No processes found, skipping test")
	}

	// 檢查不超過限制
	if len(metrics) > limit {
		t.Errorf("len(metrics) = %d, want <= %d", len(metrics), limit)
	}

	for i, proc := range metrics {
		// 檢查 PID 大於 0
		if proc.PID <= 0 {
			t.Errorf("metrics[%d].PID = %d, want > 0", i, proc.PID)
		}

		// 檢查 CPU 使用率範圍
		if proc.CPUPercent < 0 {
			t.Errorf("metrics[%d].CPUPercent = %v, want >= 0", i, proc.CPUPercent)
		}

		// 檢查記憶體使用率範圍
		if proc.MemoryPercent < 0 || proc.MemoryPercent > 100 {
			t.Errorf("metrics[%d].MemoryPercent = %v, want between 0 and 100", i, proc.MemoryPercent)
		}
	}
}

func TestCollectProcessMetricsNoLimit(t *testing.T) {
	// 測試無限制
	metrics, err := CollectProcessMetrics(0)
	if err != nil {
		t.Fatalf("CollectProcessMetrics() error = %v", err)
	}

	// 應該返回所有進程
	if len(metrics) == 0 {
		t.Skip("No processes found, skipping test")
	}
}

func TestCollectProcessMetricsSorted(t *testing.T) {
	metrics, err := CollectProcessMetrics(20)
	if err != nil {
		t.Fatalf("CollectProcessMetrics() error = %v", err)
	}

	if len(metrics) < 2 {
		t.Skip("Not enough processes to test sorting")
	}

	// 檢查按 CPU 使用率排序（降序）
	for i := 1; i < len(metrics); i++ {
		if metrics[i].CPUPercent > metrics[i-1].CPUPercent {
			t.Errorf("Processes not sorted by CPU usage: metrics[%d].CPUPercent (%v) > metrics[%d].CPUPercent (%v)",
				i, metrics[i].CPUPercent, i-1, metrics[i-1].CPUPercent)
		}
	}
}

func BenchmarkCollectProcessMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = CollectProcessMetrics(10)
	}
}
