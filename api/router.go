package api

import (
	"net/http"

	"host-agent/config"

	"github.com/gorilla/mux"
)

func SetupRoutes(router *mux.Router, cfg *config.Config) {
	handler := NewHandler(cfg)

	// 健康檢查
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")

	// 完整指標
	router.HandleFunc("/metrics", handler.GetMetrics).Methods("GET")

	// 個別指標
	router.HandleFunc("/metrics/cpu", handler.GetCPUMetrics).Methods("GET")
	router.HandleFunc("/metrics/memory", handler.GetMemoryMetrics).Methods("GET")
	router.HandleFunc("/metrics/disk", handler.GetDiskMetrics).Methods("GET")
	router.HandleFunc("/metrics/network", handler.GetNetworkMetrics).Methods("GET")
	router.HandleFunc("/metrics/process", handler.GetProcessMetrics).Methods("GET")

	// CORS
	router.Use(corsMiddleware)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
