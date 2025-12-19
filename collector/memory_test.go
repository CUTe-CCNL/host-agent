package collector

import (
	"testing"
)

func TestCollectMemoryMetrics(t *testing.T) {
	metrics, err := CollectMemoryMetrics()
	if err != nil {
		t.Fatalf("CollectMemoryMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Fatal("CollectMemoryMetrics() returned nil")
	}

	// 檢查總記憶體大於 0
	if metrics.Total == 0 {
		t.Error("Total memory should be greater than 0")
	}

	// 檢查已使用記憶體不超過總記憶體
	if metrics.Used > metrics.Total {
		t.Errorf("Used (%v) should not exceed Total (%v)", metrics.Used, metrics.Total)
	}

	// 檢查使用率範圍
	if metrics.UsedPercent < 0 || metrics.UsedPercent > 100 {
		t.Errorf("UsedPercent = %v, want between 0 and 100", metrics.UsedPercent)
	}

	// 檢查可用記憶體不超過總記憶體
	if metrics.Available > metrics.Total {
		t.Errorf("Available (%v) should not exceed Total (%v)", metrics.Available, metrics.Total)
	}
}

func TestMemoryMetricsSwap(t *testing.T) {
	metrics, err := CollectMemoryMetrics()
	if err != nil {
		t.Fatalf("CollectMemoryMetrics() error = %v", err)
	}

	// Swap 使用率範圍檢查（如果有 swap）
	if metrics.Swap.Total > 0 {
		if metrics.Swap.UsedPercent < 0 || metrics.Swap.UsedPercent > 100 {
			t.Errorf("Swap.UsedPercent = %v, want between 0 and 100", metrics.Swap.UsedPercent)
		}

		if metrics.Swap.Used > metrics.Swap.Total {
			t.Errorf("Swap.Used (%v) should not exceed Swap.Total (%v)", metrics.Swap.Used, metrics.Swap.Total)
		}
	}
}

func BenchmarkCollectMemoryMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = CollectMemoryMetrics()
	}
}
