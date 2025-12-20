package collector

import (
	"testing"
)

func TestCollectNetworkMetrics(t *testing.T) {
	metrics, err := CollectNetworkMetrics()
	if err != nil {
		t.Fatalf("CollectNetworkMetrics() error = %v", err)
	}

	// 可能沒有網路介面（除了 loopback）
	if len(metrics) == 0 {
		t.Skip("No network interfaces found (excluding loopback), skipping test")
	}

	for i, net := range metrics {
		// 檢查介面名稱不為空
		if net.Interface == "" {
			t.Errorf("metrics[%d].Interface is empty", i)
		}

		// 檢查不是 loopback
		if net.Interface == "lo" {
			t.Errorf("metrics[%d].Interface should not be loopback", i)
		}

		// 驗證 MAC 和 IP 字段存在（可能為空，但結構應該存在）
		// MAC 和 IP 可能為空，這是正常的（例如某些虛擬介面可能沒有 MAC）
		_ = net.MAC
		_ = net.IP
	}
}

func TestNetworkMetricsExcludesLoopback(t *testing.T) {
	metrics, err := CollectNetworkMetrics()
	if err != nil {
		t.Fatalf("CollectNetworkMetrics() error = %v", err)
	}

	for _, net := range metrics {
		if net.Interface == "lo" {
			t.Error("Loopback interface should be excluded")
		}
	}
}

func BenchmarkCollectNetworkMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = CollectNetworkMetrics()
	}
}
