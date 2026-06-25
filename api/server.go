package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/CUTe-CCNL/host-agent/config"
)

type Server struct {
	httpServer *http.Server
	config     *config.Config
}

func NewServer(cfg *config.Config, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
			Handler:      handler,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  60 * time.Second,
		},
		config: cfg,
	}
}

func (s *Server) Start() error {
	log.Printf("HTTP 服務啟動於端口 %d", s.config.Server.Port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("HTTP 服務關閉中...")
	return s.httpServer.Shutdown(ctx)
}
