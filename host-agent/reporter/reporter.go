package reporter

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"host-agent/collector"
	"host-agent/config"
	"host-agent/models"
)

type Reporter struct {
	config *config.Config
	client *http.Client
	stop   chan struct{}
}

func NewReporter(cfg *config.Config) *Reporter {
	return &Reporter{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Report.Timeout,
		},
		stop: make(chan struct{}),
	}
}

func (r *Reporter) Start() {
	ticker := time.NewTicker(r.config.Report.Interval)
	defer ticker.Stop()

	log.Printf("資料回報器啟動，間隔 %v", r.config.Report.Interval)

	for {
		select {
		case <-ticker.C:
			r.report()
		case <-r.stop:
			log.Println("資料回報器停止")
			return
		}
	}
}

func (r *Reporter) Stop() {
	close(r.stop)
}

func (r *Reporter) report() {
	// 收集指標
	metrics := &models.Metrics{
		Hostname:  collector.GetHostname(),
		Timestamp: time.Now(),
	}

	if r.config.Collector.EnableCPU {
		cpu, _ := collector.CollectCPUMetrics()
		metrics.CPU = cpu
	}

	if r.config.Collector.EnableMemory {
		memory, _ := collector.CollectMemoryMetrics()
		metrics.Memory = memory
	}

	if r.config.Collector.EnableDisk {
		disk, _ := collector.CollectDiskMetrics(r.config.Collector.DiskMountPoints)
		metrics.Disk = disk
	}

	if r.config.Collector.EnableNetwork {
		network, _ := collector.CollectNetworkMetrics()
		metrics.Network = network
	}

	system, _ := collector.CollectSystemMetrics()
	metrics.System = system

	// 發送到後端
	data, err := json.Marshal(metrics)
	if err != nil {
		log.Printf("序列化指標失敗: %v", err)
		return
	}

	resp, err := r.client.Post(
		r.config.Report.Endpoint,
		"application/json",
		bytes.NewBuffer(data),
	)

	if err != nil {
		log.Printf("回報指標失敗: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("回報指標失敗，狀態碼: %d", resp.StatusCode)
		return
	}

	log.Printf("成功回報指標到 %s", r.config.Report.Endpoint)
}
