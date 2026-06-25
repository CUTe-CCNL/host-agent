package collector

import (
	"github.com/CUTe-CCNL/host-agent/models"

	"github.com/shirou/gopsutil/v4/net"
)

func CollectNetworkMetrics() ([]models.NetworkMetrics, error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	// 獲取所有網路介面的詳細信息（包含 MAC 和 IP）
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// 建立介面名稱到介面信息的映射
	interfaceMap := make(map[string]net.InterfaceStat)
	for _, iface := range interfaces {
		interfaceMap[iface.Name] = iface
	}

	var metrics []models.NetworkMetrics
	for _, counter := range ioCounters {
		// 跳過 loopback 介面
		if counter.Name == "lo" {
			continue
		}

		// 從映射中獲取介面詳細信息
		iface, exists := interfaceMap[counter.Name]
		mac := ""
		var ips []string

		if exists {
			mac = iface.HardwareAddr
			// 收集所有 IP 地址（IPv4 和 IPv6）
			for _, addr := range iface.Addrs {
				ips = append(ips, addr.Addr)
			}
		}

		metrics = append(metrics, models.NetworkMetrics{
			Interface:   counter.Name,
			MAC:         mac,
			IP:          ips,
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
