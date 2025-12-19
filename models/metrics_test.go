package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMetricsJSON(t *testing.T) {
	metrics := &Metrics{
		Hostname:  "test-host",
		Timestamp: time.Now(),
		CPU: &CPUMetrics{
			UsagePercent: 50.5,
			Cores:        4,
			PerCore:      []float64{45.0, 55.0, 48.0, 52.0},
			LoadAverage:  []float64{1.5, 2.0, 1.8},
		},
		Memory: &MemoryMetrics{
			Total:       16 * 1024 * 1024 * 1024, // 16GB
			Used:        8 * 1024 * 1024 * 1024,  // 8GB
			Free:        8 * 1024 * 1024 * 1024,
			Available:   10 * 1024 * 1024 * 1024,
			UsedPercent: 50.0,
		},
		Disk: []DiskMetrics{
			{
				Device:      "/dev/sda1",
				MountPoint:  "/",
				Fstype:      "ext4",
				Total:       100 * 1024 * 1024 * 1024,
				Used:        50 * 1024 * 1024 * 1024,
				Free:        50 * 1024 * 1024 * 1024,
				UsedPercent: 50.0,
			},
		},
		Network: []NetworkMetrics{
			{
				Interface:   "eth0",
				BytesSent:   1024 * 1024,
				BytesRecv:   2048 * 1024,
				PacketsSent: 1000,
				PacketsRecv: 2000,
			},
		},
		System: &SystemMetrics{
			OS:              "linux",
			Platform:        "ubuntu",
			PlatformVersion: "22.04",
			KernelVersion:   "5.15.0",
			Architecture:    "amd64",
			Uptime:          86400,
			BootTime:        time.Now().Add(-24 * time.Hour),
			Processes:       150,
		},
	}

	// 測試序列化
	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal metrics: %v", err)
	}

	if len(data) == 0 {
		t.Error("Marshaled data should not be empty")
	}

	// 測試反序列化
	var decoded Metrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal metrics: %v", err)
	}

	// 驗證值
	if decoded.Hostname != metrics.Hostname {
		t.Errorf("Hostname = %s, want %s", decoded.Hostname, metrics.Hostname)
	}

	if decoded.CPU.UsagePercent != metrics.CPU.UsagePercent {
		t.Errorf("CPU.UsagePercent = %v, want %v", decoded.CPU.UsagePercent, metrics.CPU.UsagePercent)
	}

	if decoded.Memory.Total != metrics.Memory.Total {
		t.Errorf("Memory.Total = %v, want %v", decoded.Memory.Total, metrics.Memory.Total)
	}

	if len(decoded.Disk) != len(metrics.Disk) {
		t.Errorf("len(Disk) = %d, want %d", len(decoded.Disk), len(metrics.Disk))
	}

	if len(decoded.Network) != len(metrics.Network) {
		t.Errorf("len(Network) = %d, want %d", len(decoded.Network), len(metrics.Network))
	}
}

func TestMetricsOmitEmpty(t *testing.T) {
	// 測試 omitempty 標籤
	metrics := &Metrics{
		Hostname:  "test-host",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal metrics: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// 空值不應該出現在 JSON 中
	if _, ok := decoded["cpu"]; ok {
		t.Error("cpu should be omitted when nil")
	}

	if _, ok := decoded["memory"]; ok {
		t.Error("memory should be omitted when nil")
	}

	if _, ok := decoded["disk"]; ok {
		t.Error("disk should be omitted when nil or empty")
	}
}

func TestCPUMetricsJSON(t *testing.T) {
	cpu := &CPUMetrics{
		UsagePercent: 75.5,
		Cores:        8,
		PerCore:      []float64{70, 80, 75, 72, 78, 76, 74, 77},
		LoadAverage:  []float64{2.5, 2.2, 2.0},
	}

	data, err := json.Marshal(cpu)
	if err != nil {
		t.Fatalf("Failed to marshal CPU metrics: %v", err)
	}

	var decoded CPUMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal CPU metrics: %v", err)
	}

	if decoded.Cores != cpu.Cores {
		t.Errorf("Cores = %d, want %d", decoded.Cores, cpu.Cores)
	}

	if len(decoded.PerCore) != len(cpu.PerCore) {
		t.Errorf("len(PerCore) = %d, want %d", len(decoded.PerCore), len(cpu.PerCore))
	}
}

func TestProcessMetricsJSON(t *testing.T) {
	process := &ProcessMetrics{
		PID:           1234,
		Name:          "test-process",
		Status:        "running",
		CPUPercent:    5.5,
		MemoryMB:      256,
		MemoryPercent: 1.5,
	}

	data, err := json.Marshal(process)
	if err != nil {
		t.Fatalf("Failed to marshal process metrics: %v", err)
	}

	var decoded ProcessMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal process metrics: %v", err)
	}

	if decoded.PID != process.PID {
		t.Errorf("PID = %d, want %d", decoded.PID, process.PID)
	}

	if decoded.Name != process.Name {
		t.Errorf("Name = %s, want %s", decoded.Name, process.Name)
	}
}

func BenchmarkMetricsMarshal(b *testing.B) {
	metrics := &Metrics{
		Hostname:  "test-host",
		Timestamp: time.Now(),
		CPU: &CPUMetrics{
			UsagePercent: 50.5,
			Cores:        4,
		},
		Memory: &MemoryMetrics{
			Total:       16 * 1024 * 1024 * 1024,
			Used:        8 * 1024 * 1024 * 1024,
			UsedPercent: 50.0,
		},
	}

	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(metrics)
	}
}

func BenchmarkMetricsUnmarshal(b *testing.B) {
	metrics := &Metrics{
		Hostname:  "test-host",
		Timestamp: time.Now(),
		CPU: &CPUMetrics{
			UsagePercent: 50.5,
			Cores:        4,
		},
	}

	data, _ := json.Marshal(metrics)

	for i := 0; i < b.N; i++ {
		var decoded Metrics
		_ = json.Unmarshal(data, &decoded)
	}
}
