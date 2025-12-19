package collector

import (
	"runtime"
	"time"

	"host-agent/models"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
)

func CollectCPUMetrics() (*models.CPUMetrics, error) {
	// CPU 使用率
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, err
	}

	// 每個核心的使用率
	perCore, err := cpu.Percent(time.Second, true)
	if err != nil {
		perCore = nil
	}

	// 負載平均
	loadAvg, err := load.Avg()
	var loadAvgSlice []float64
	if err == nil {
		loadAvgSlice = []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}
	}

	return &models.CPUMetrics{
		UsagePercent: percentages[0],
		Cores:        runtime.NumCPU(),
		PerCore:      perCore,
		LoadAverage:  loadAvgSlice,
	}, nil
}
