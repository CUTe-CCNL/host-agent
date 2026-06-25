package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/CUTe-CCNL/host-agent/collector"
	"github.com/CUTe-CCNL/host-agent/config"
	"github.com/CUTe-CCNL/host-agent/models"
	agentplugin "github.com/CUTe-CCNL/host-agent/plugin"

	"github.com/gorilla/mux"
)

type PluginRegistry interface {
	List() []agentplugin.Info
	Get(id string) (agentplugin.Info, bool)
	Restart(ctx context.Context, id string) error
	ProxyHTTP(w http.ResponseWriter, r *http.Request, id, path string)
}

type Handler struct {
	config  *config.Config
	plugins PluginRegistry
}

func NewHandler(cfg *config.Config) *Handler {
	return NewHandlerWithPlugins(cfg, nil)
}

func NewHandlerWithPlugins(cfg *config.Config, plugins PluginRegistry) *Handler {
	return &Handler{config: cfg, plugins: plugins}
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
	if err := json.NewEncoder(w).Encode(memory); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetDiskMetrics 只取得磁碟指標
func (h *Handler) GetDiskMetrics(w http.ResponseWriter, r *http.Request) {
	disk, err := collector.CollectDiskMetrics(h.config.Collector.DiskMountPoints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(disk); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetNetworkMetrics 只取得網路指標
func (h *Handler) GetNetworkMetrics(w http.ResponseWriter, r *http.Request) {
	network, err := collector.CollectNetworkMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(network); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetProcessMetrics 只取得行程指標
func (h *Handler) GetProcessMetrics(w http.ResponseWriter, r *http.Request) {
	processes, err := collector.CollectProcessMetrics(h.config.Collector.ProcessLimit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(processes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}

	plugins := []agentplugin.Info{}
	if h.plugins != nil {
		plugins = h.plugins.List()
	}
	h.writeJSON(w, plugins)
}

func (h *Handler) GetPlugin(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	if h.plugins == nil {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}

	id := mux.Vars(r)["id"]
	info, ok := h.plugins.Get(id)
	if !ok {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	h.writeJSON(w, info)
}

func (h *Handler) RestartPlugin(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	if h.plugins == nil {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}

	id := mux.Vars(r)["id"]
	if _, ok := h.plugins.Get(id); !ok {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	if err := h.plugins.Restart(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	h.writeJSON(w, map[string]string{"status": "restarted"})
}

func (h *Handler) ProxyPlugin(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	if h.plugins == nil {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}

	vars := mux.Vars(r)
	path := vars["path"]
	if path == "" {
		path = "/"
	} else {
		path = "/" + path
	}

	h.plugins.ProxyHTTP(w, r, vars["id"], path)
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) bool {
	if !h.config.Server.EnableAuth {
		return true
	}

	token := r.Header.Get("Authorization")
	if token != "Bearer "+h.config.Server.AuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func (h *Handler) writeJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
