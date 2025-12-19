package collector

import (
	"host-agent/models"

	"github.com/shirou/gopsutil/v3/net"
)

func CollectNetworkMetrics() ([]models.NetworkMetrics, error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	var metrics []models.NetworkMetrics
	for _, counter := range ioCounters {
		// 跳過 loopback 介面
		if counter.Name == "lo" {
			continue
		}

		metrics = append(metrics, models.NetworkMetrics{
			Interface:   counter.Name,
			BytesSent:   counter.BytesSent,
			BytesRecv:   counter.BytesRecv,
			PacketsSent: counter.PacketsSent,
			PacketsRecv: counter.PacketsRecv,
			ErrorsIn:    counter.Errin,
			ErrorsOut:   counter.Errout,
			DropIn:      counter.Dropin,
			DropOut:     counter.Dropout,
		})
	}

	return metrics, nil
}
