package api

import (
	"encoding/json"
	"net/http"
	"time"

	"host-agent/collector"
	"host-agent/config"
	"host-agent/models"
)

type Handler struct {
	config *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{config: cfg}
}

// GetMetrics 取得所有指標
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	// 驗證
	if h.config.Server.EnableAuth {
		token := r.Header.Get("Authorization")
		if token != "Bearer "+h.config.Server.AuthToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	metrics := &models.Metrics{
		Hostname:  collector.GetHostname(),
		Timestamp: time.Now(),
	}

	// 收集各項指標
	if h.config.Collector.EnableCPU {
		cpu, err := collector.CollectCPUMetrics()
		if err == nil {
			metrics.CPU = cpu
		}
	}

	if h.config.Collector.EnableMemory {
		memory, err := collector.CollectMemoryMetrics()
		if err == nil {
			metrics.Memory = memory
		}
	}

	if h.config.Collector.EnableDisk {
		disk, err := collector.CollectDiskMetrics(h.config.Collector.DiskMountPoints)
		if err == nil {
			metrics.Disk = disk
		}
	}

	if h.config.Collector.EnableNetwork {
		network, err := collector.CollectNetworkMetrics()
		if err == nil {
			metrics.Network = network
		}
	}

	if h.config.Collector.EnableProcess {
		processes, err := collector.CollectProcessMetrics(h.config.Collector.ProcessLimit)
		if err == nil {
			metrics.Processes = processes
		}
	}

	// 系統資訊
	system, err := collector.CollectSystemMetrics()
	if err == nil {
		metrics.System = system
	}

	// 回傳 JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HealthCheck 健康檢查
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetCPUMetrics 只取得 CPU 指標
func (h *Handler) GetCPUMetrics(w http.ResponseWriter, r *http.Request) {
	cpu, err := collector.CollectCPUMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cpu); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetMemoryMetrics 只取得記憶體指標
func (h *Handler) GetMemoryMetrics(w http.ResponseWriter, r *http.Request) {
	memory, err := collector.CollectMemoryMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memory)
}

// GetDiskMetrics 只取得磁碟指標
func (h *Handler) GetDiskMetrics(w http.ResponseWriter, r *http.Request) {
	disk, err := collector.CollectDiskMetrics(h.config.Collector.DiskMountPoints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(disk)
}

// GetNetworkMetrics 只取得網路指標
func (h *Handler) GetNetworkMetrics(w http.ResponseWriter, r *http.Request) {
	network, err := collector.CollectNetworkMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(network)
}

// GetProcessMetrics 只取得行程指標
func (h *Handler) GetProcessMetrics(w http.ResponseWriter, r *http.Request) {
	processes, err := collector.CollectProcessMetrics(h.config.Collector.ProcessLimit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}
