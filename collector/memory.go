package collector

import (
	"github.com/CUTe-CCNL/host-agent/models"

	"github.com/shirou/gopsutil/v4/mem"
)

func CollectMemoryMetrics() (*models.MemoryMetrics, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	swapStat, err := mem.SwapMemory()
	if err != nil {
		swapStat = &mem.SwapMemoryStat{}
	}

	metrics := &models.MemoryMetrics{
		Total:       vmStat.Total,
		Used:        vmStat.Used,
		Free:        vmStat.Free,
		Available:   vmStat.Available,
		UsedPercent: vmStat.UsedPercent,
		Cached:      vmStat.Cached,
		Buffers:     vmStat.Buffers,
	}

	metrics.Swap.Total = swapStat.Total
	metrics.Swap.Used = swapStat.Used
	metrics.Swap.Free = swapStat.Free
	metrics.Swap.UsedPercent = swapStat.UsedPercent

	return metrics, nil
}
