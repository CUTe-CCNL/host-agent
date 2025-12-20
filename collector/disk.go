package collector

import (
	"host-agent/models"

	"github.com/shirou/gopsutil/v4/disk"
)

func CollectDiskMetrics(mountPoints []string) ([]models.DiskMetrics, error) {
	var metrics []models.DiskMetrics

	// 如果沒有指定掛載點，獲取所有分區
	if len(mountPoints) == 0 {
		partitions, err := disk.Partitions(false)
		if err != nil {
			return nil, err
		}

		for _, partition := range partitions {
			mountPoints = append(mountPoints, partition.Mountpoint)
		}
	}

	for _, mountPoint := range mountPoints {
		usage, err := disk.Usage(mountPoint)
		if err != nil {
			continue // 跳過無法訪問的掛載點
		}

		partitions, _ := disk.Partitions(false)
		var device, fstype string
		for _, p := range partitions {
			if p.Mountpoint == mountPoint {
				device = p.Device
				fstype = p.Fstype
				break
			}
		}

		metrics = append(metrics, models.DiskMetrics{
			Device:      device,
			MountPoint:  mountPoint,
			Fstype:      fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
			InodesTotal: usage.InodesTotal,
			InodesUsed:  usage.InodesUsed,
			InodesFree:  usage.InodesFree,
		})
	}

	return metrics, nil
}
