package reporter

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/CUTe-CCNL/host-agent/collector"
	"github.com/CUTe-CCNL/host-agent/config"
	"github.com/CUTe-CCNL/host-agent/models"
)

type Reporter struct {
	config           *config.Config
	httpClient       *http.Client
	rabbitMQProducer *RabbitMQProducer
	stop             chan struct{}
}

func NewReporter(cfg *config.Config) *Reporter {
	r := &Reporter{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Report.Timeout,
		},
		stop: make(chan struct{}),
	}

	// 如果啟用 RabbitMQ，初始化 Producer
	if cfg.Report.Mode == "rabbitmq" || cfg.Report.Mode == "both" {
		producer, err := NewRabbitMQProducer(cfg)
		if err != nil {
			log.Printf("警告: 無法建立 RabbitMQ Producer: %v", err)
			log.Println("將只使用 HTTP 模式")
			if cfg.Report.Mode == "rabbitmq" {
				cfg.Report.Mode = "http" // 降級到 HTTP
			} else {
				cfg.Report.Mode = "http" // both -> http
			}
		} else {
			r.rabbitMQProducer = producer
		}
	}

	return r
}

func (r *Reporter) Start() {
	ticker := time.NewTicker(r.config.Report.Interval)
	defer ticker.Stop()

	log.Printf("資料回報器啟動，模式: %s, 間隔: %v", r.config.Report.Mode, r.config.Report.Interval)

	// 立即執行一次
	r.report()

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

	// 關閉 RabbitMQ Producer
	if r.rabbitMQProducer != nil {
		if err := r.rabbitMQProducer.Close(); err != nil {
			log.Printf("關閉 RabbitMQ Producer 失敗: %v", err)
		}
	}
}

func (r *Reporter) report() {
	// 收集指標
	metrics := r.collectMetrics()

	// 根據模式發送
	switch r.config.Report.Mode {
	case "http":
		r.sendHTTP(metrics)
	case "rabbitmq":
		r.sendRabbitMQ(metrics)
	case "both":
		// 並行發送
		go r.sendHTTP(metrics)
		go r.sendRabbitMQ(metrics)
	default:
		log.Printf("未知的回報模式: %s", r.config.Report.Mode)
	}
}

func (r *Reporter) collectMetrics() *models.Metrics {
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

	if r.config.Collector.EnableProcess {
		processes, _ := collector.CollectProcessMetrics(r.config.Collector.ProcessLimit)
		metrics.Processes = processes
	}

	system, _ := collector.CollectSystemMetrics()
	metrics.System = system

	return metrics
}

func (r *Reporter) sendHTTP(metrics *models.Metrics) {
	if r.config.Report.HTTP.Endpoint == "" {
		return
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		log.Printf("HTTP: 序列化指標失敗: %v", err)
		return
	}

	resp, err := r.httpClient.Post(
		r.config.Report.HTTP.Endpoint,
		"application/json",
		bytes.NewBuffer(data),
	)

	if err != nil {
		log.Printf("HTTP: 回報指標失敗: %v", err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP: 回報指標失敗，狀態碼: %d", resp.StatusCode)
		return
	}

	log.Printf("HTTP: 成功回報指標到 %s [%d bytes]", r.config.Report.HTTP.Endpoint, len(data))
}

func (r *Reporter) sendRabbitMQ(metrics *models.Metrics) {
	if r.rabbitMQProducer == nil {
		return
	}

	if err := r.rabbitMQProducer.SendMetrics(metrics); err != nil {
		log.Printf("RabbitMQ: 發送指標失敗: %v", err)
		return
	}
}
