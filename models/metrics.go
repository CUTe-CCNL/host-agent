package models

import "time"

type Metrics struct {
	Hostname  string           `json:"hostname"`
	Timestamp time.Time        `json:"timestamp"`
	CPU       *CPUMetrics      `json:"cpu,omitempty"`
	Memory    *MemoryMetrics   `json:"memory,omitempty"`
	Disk      []DiskMetrics    `json:"disk,omitempty"`
	Network   []NetworkMetrics `json:"network,omitempty"`
	System    *SystemMetrics   `json:"system,omitempty"`
	Processes []ProcessMetrics `json:"processes,omitempty"`
}

type CPUMetrics struct {
	UsagePercent float64   `json:"usage_percent"`
	Cores        int       `json:"cores"`
	PerCore      []float64 `json:"per_core,omitempty"`
	LoadAverage  []float64 `json:"load_average,omitempty"` // 1, 5, 15 分鐘
}

type MemoryMetrics struct {
	Total       uint64  `json:"total"`        // bytes
	Used        uint64  `json:"used"`         // bytes
	Free        uint64  `json:"free"`         // bytes
	Available   uint64  `json:"available"`    // bytes
	UsedPercent float64 `json:"used_percent"` // %
	Cached      uint64  `json:"cached"`       // bytes
	Buffers     uint64  `json:"buffers"`      // bytes
	Swap        struct {
		Total       uint64  `json:"total"`
		Used        uint64  `json:"used"`
		Free        uint64  `json:"free"`
		UsedPercent float64 `json:"used_percent"`
	} `json:"swap"`
}

type DiskMetrics struct {
	Device      string  `json:"device"`
	MountPoint  string  `json:"mount_point"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`        // bytes
	Used        uint64  `json:"used"`         // bytes
	Free        uint64  `json:"free"`         // bytes
	UsedPercent float64 `json:"used_percent"` // %
	InodesTotal uint64  `json:"inodes_total"`
	InodesUsed  uint64  `json:"inodes_used"`
	InodesFree  uint64  `json:"inodes_free"`
}

type NetworkMetrics struct {
	Interface   string   `json:"interface"`
	MAC         string   `json:"mac,omitempty"`
	IP          []string `json:"ip,omitempty"`
	BytesSent   uint64   `json:"bytes_sent"`
	BytesRecv   uint64   `json:"bytes_recv"`
	PacketsSent uint64   `json:"packets_sent"`
	PacketsRecv uint64   `json:"packets_recv"`
	ErrorsIn    uint64   `json:"errors_in"`
	ErrorsOut   uint64   `json:"errors_out"`
	DropIn      uint64   `json:"drop_in"`
	DropOut     uint64   `json:"drop_out"`
}

type SystemMetrics struct {
	// HostID 是主機的唯一標識符，用於區分不同的主機。
	HostID          string    `json:"host_id"`
	OS              string    `json:"os"`
	Platform        string    `json:"platform"`
	PlatformVersion string    `json:"platform_version"`
	KernelVersion   string    `json:"kernel_version"`
	Architecture    string    `json:"architecture"`
	Uptime          uint64    `json:"uptime"` // seconds
	BootTime        time.Time `json:"boot_time"`
	Processes       uint64    `json:"processes"`
}

type ProcessMetrics struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      uint64  `json:"memory_mb"`
	MemoryPercent float32 `json:"memory_percent"`
}
