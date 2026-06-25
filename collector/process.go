package collector

import (
	"sort"

	"github.com/CUTe-CCNL/host-agent/models"

	"github.com/shirou/gopsutil/v4/process"
)

func CollectProcessMetrics(limit int) ([]models.ProcessMetrics, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	var processes []models.ProcessMetrics

	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue
		}

		name, _ := proc.Name()
		status, _ := proc.Status()
		cpuPercent, _ := proc.CPUPercent()
		memInfo, _ := proc.MemoryInfo()
		memPercent, _ := proc.MemoryPercent()

		var memMB uint64
		if memInfo != nil {
			memMB = memInfo.RSS / 1024 / 1024
		}

		statusStr := ""
		if len(status) > 0 {
			statusStr = status[0]
		}

		processes = append(processes, models.ProcessMetrics{
			PID:           pid,
			Name:          name,
			Status:        statusStr,
			CPUPercent:    cpuPercent,
			MemoryMB:      memMB,
			MemoryPercent: memPercent,
		})
	}

	// 按 CPU 使用率排序
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPUPercent > processes[j].CPUPercent
	})

	// 限制數量
	if limit > 0 && len(processes) > limit {
		processes = processes[:limit]
	}

	return processes, nil
}
