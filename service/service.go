package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"host-agent/api"
	"host-agent/config"
	agentplugin "host-agent/plugin"
	"host-agent/reporter"

	"github.com/gorilla/mux"
	"github.com/kardianos/service"
)

type Program struct {
	cfg      *config.Config
	server   *api.Server
	reporter *reporter.Reporter
	plugins  *agentplugin.Registry
	exit     chan struct{}
}

func NewProgram(cfg *config.Config) *Program {
	return &Program{
		cfg:  cfg,
		exit: make(chan struct{}),
	}
}

// Start 服務啟動
func (p *Program) Start(s service.Service) error {
	log.Println("Host Agent 服務啟動中...")
	go p.run()
	return nil
}

// Stop 服務停止
func (p *Program) Stop(s service.Service) error {
	log.Println("Host Agent 服務停止中...")
	close(p.exit)

	// 優雅關閉
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.server.Shutdown(ctx); err != nil {
			log.Printf("關閉伺服器時發生錯誤: %v", err)
		}
	}

	if p.reporter != nil {
		p.reporter.Stop()
	}

	if p.plugins != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.plugins.Stop(ctx); err != nil {
			log.Printf("停止插件時發生錯誤: %v", err)
		}
	}

	return nil
}

// run 主要運行邏輯
func (p *Program) run() {
	if p.cfg.Plugins.Enabled {
		p.plugins = agentplugin.NewRegistry(p.cfg.Plugins)
		if err := p.plugins.Load(p.cfg.Plugins.Directory); err != nil {
			log.Printf("載入插件 manifest 時發生錯誤: %v", err)
			p.plugins = nil
		} else if err := p.plugins.Start(context.Background()); err != nil {
			log.Printf("啟動插件時發生錯誤: %v", err)
		}
	}

	// 啟動 HTTP API 服務
	router := mux.NewRouter()
	api.SetupRoutesWithPlugins(router, p.cfg, p.plugins)

	p.server = api.NewServer(p.cfg, router)
	go func() {
		if err := p.server.Start(); err != nil {
			log.Printf("HTTP 服務錯誤: %v", err)
		}
	}()

	// 啟動資料回報器（如果啟用）
	if p.cfg.Report.Enabled {
		p.reporter = reporter.NewReporter(p.cfg)
		go p.reporter.Start()
	}

	log.Printf("Host Agent 服務已啟動於端口 %d", p.cfg.Server.Port)

	// 等待退出信號
	<-p.exit
}

// ServiceConfig 服務配置
type ServiceConfig struct {
	Name        string
	DisplayName string
	Description string
}

var DefaultServiceConfig = &ServiceConfig{
	Name:        "HostAgent",
	DisplayName: "Host Monitoring Agent",
	Description: "收集主機系統指標並提供 API 查詢",
}

// InstallService 安裝服務
func InstallService(configPath string) error {
	svcConfig := &service.Config{
		Name:        DefaultServiceConfig.Name,
		DisplayName: DefaultServiceConfig.DisplayName,
		Description: DefaultServiceConfig.Description,
		Arguments:   []string{"-service", "-config", configPath},
	}

	// 載入配置
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("無法載入配置: %v", err)
	}

	prg := NewProgram(cfg)
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Install()
}

// UninstallService 卸載服務
func UninstallService() error {
	svcConfig := &service.Config{
		Name:        DefaultServiceConfig.Name,
		DisplayName: DefaultServiceConfig.DisplayName,
		Description: DefaultServiceConfig.Description,
	}

	prg := &Program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Uninstall()
}

// StartService 啟動服務
func StartService() error {
	svcConfig := &service.Config{
		Name: DefaultServiceConfig.Name,
	}

	prg := &Program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Start()
}

// StopService 停止服務
func StopService() error {
	svcConfig := &service.Config{
		Name: DefaultServiceConfig.Name,
	}

	prg := &Program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Stop()
}

// RestartService 重啟服務
func RestartService() error {
	svcConfig := &service.Config{
		Name: DefaultServiceConfig.Name,
	}

	prg := &Program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Restart()
}

// RunService 運行服務
func RunService(cfg *config.Config) error {
	svcConfig := &service.Config{
		Name:        DefaultServiceConfig.Name,
		DisplayName: DefaultServiceConfig.DisplayName,
		Description: DefaultServiceConfig.Description,
	}

	prg := NewProgram(cfg)
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	// 設置日誌
	logger, err := s.Logger(nil)
	if err != nil {
		return err
	}

	err = s.Run()
	if err != nil {
		if logErr := logger.Error(err); logErr != nil {
			log.Printf("記錄錯誤時發生問題: %v", logErr)
		}
	}

	return err
}
