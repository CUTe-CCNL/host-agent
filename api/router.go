package api

import (
	"net/http"

	"github.com/CUTe-CCNL/host-agent/config"

	"github.com/gorilla/mux"
)

func SetupRoutes(router *mux.Router, cfg *config.Config) {
	SetupRoutesWithPlugins(router, cfg, nil)
}

func SetupRoutesWithPlugins(router *mux.Router, cfg *config.Config, plugins PluginRegistry) {
	handler := NewHandlerWithPlugins(cfg, plugins)

	// CORS middleware 需要在路由之前應用
	router.Use(corsMiddleware)

	// 健康檢查
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET", "OPTIONS")

	// 完整指標
	router.HandleFunc("/metrics", handler.GetMetrics).Methods("GET", "OPTIONS")

	// 個別指標
	router.HandleFunc("/metrics/cpu", handler.GetCPUMetrics).Methods("GET", "OPTIONS")
	router.HandleFunc("/metrics/memory", handler.GetMemoryMetrics).Methods("GET", "OPTIONS")
	router.HandleFunc("/metrics/disk", handler.GetDiskMetrics).Methods("GET", "OPTIONS")
	router.HandleFunc("/metrics/network", handler.GetNetworkMetrics).Methods("GET", "OPTIONS")
	router.HandleFunc("/metrics/process", handler.GetProcessMetrics).Methods("GET", "OPTIONS")

	// 插件管理與代理
	router.HandleFunc("/plugins", handler.ListPlugins).Methods("GET", "OPTIONS")
	router.HandleFunc("/plugins/{id}", handler.GetPlugin).Methods("GET", "OPTIONS")
	router.HandleFunc("/plugins/{id}/restart", handler.RestartPlugin).Methods("POST", "OPTIONS")
	router.HandleFunc("/plugin-api/{id}", handler.ProxyPlugin).Methods("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS")
	router.HandleFunc("/plugin-api/{id}/{path:.*}", handler.ProxyPlugin).Methods("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
