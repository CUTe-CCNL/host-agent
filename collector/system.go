package collector

import (
	"os"
	"time"

	"github.com/CUTe-CCNL/host-agent/models"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/process"
)

func CollectSystemMetrics() (*models.SystemMetrics, error) {
	info, err := host.Info()
	if err != nil {
		return nil, err
	}

	processes, _ := process.Pids()

	return &models.SystemMetrics{
		HostID:          info.HostID,
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		Architecture:    info.KernelArch,
		Uptime:          info.Uptime,
		BootTime:        time.Unix(int64(info.BootTime), 0),
		Processes:       uint64(len(processes)),
	}, nil
}

func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
