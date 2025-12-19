package collector

import (
	"testing"
)

func TestCollectDiskMetrics(t *testing.T) {
	// 測試自動偵測掛載點
	metrics, err := CollectDiskMetrics(nil)
	if err != nil {
		t.Fatalf("CollectDiskMetrics() error = %v", err)
	}

	// 至少應該有一個磁碟
	if len(metrics) == 0 {
		t.Skip("No disk partitions found, skipping test")
	}

	for i, disk := range metrics {
		// 檢查掛載點不為空
		if disk.MountPoint == "" {
			t.Errorf("metrics[%d].MountPoint is empty", i)
		}

		// 檢查總容量大於 0
		if disk.Total == 0 {
			t.Errorf("metrics[%d].Total should be greater than 0", i)
		}

		// 檢查已使用不超過總容量
		if disk.Used > disk.Total {
			t.Errorf("metrics[%d].Used (%v) should not exceed Total (%v)", i, disk.Used, disk.Total)
		}

		// 檢查使用率範圍
		if disk.UsedPercent < 0 || disk.UsedPercent > 100 {
			t.Errorf("metrics[%d].UsedPercent = %v, want between 0 and 100", i, disk.UsedPercent)
		}
	}
}

func TestCollectDiskMetricsWithMountPoints(t *testing.T) {
	// 測試指定掛載點（使用根目錄）
	mountPoints := []string{"/"}
	metrics, err := CollectDiskMetrics(mountPoints)
	if err != nil {
		t.Fatalf("CollectDiskMetrics() error = %v", err)
	}

	if len(metrics) == 0 {
		t.Skip("Root partition not accessible, skipping test")
	}

	// 檢查返回的掛載點是否正確
	found := false
	for _, disk := range metrics {
		if disk.MountPoint == "/" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find root mount point in results")
	}
}

func BenchmarkCollectDiskMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = CollectDiskMetrics(nil)
	}
}
