package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"host-agent/config"
	"host-agent/service"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
)

func main() {
	var (
		configPath    = flag.String("config", getDefaultConfigPath(), "配置檔案路徑")
		showVersion   = flag.Bool("version", false, "顯示版本資訊")
		installFlag   = flag.Bool("install", false, "安裝為系統服務")
		uninstallFlag = flag.Bool("uninstall", false, "卸載系統服務")
		startFlag     = flag.Bool("start", false, "啟動服務")
		stopFlag      = flag.Bool("stop", false, "停止服務")
		restartFlag   = flag.Bool("restart", false, "重啟服務")
		serviceFlag   = flag.Bool("service", false, "以服務模式運行（由系統調用）")
	)

	flag.Parse()

	// 顯示版本
	if *showVersion {
		fmt.Printf("Host Agent v%s (built at %s)\n", version, buildTime)
		os.Exit(0)
	}

	// 安裝服務
	if *installFlag {
		if err := service.InstallService(*configPath); err != nil {
			log.Fatalf("安裝服務失敗: %v", err)
		}
		fmt.Println("✓ 服務安裝成功")
		fmt.Println("執行以下命令啟動服務:")
		if isWindows() {
			fmt.Println("  net start HostAgent")
		} else {
			fmt.Println("  sudo systemctl start HostAgent")
		}
		return
	}

	// 卸載服務
	if *uninstallFlag {
		// 先停止服務
		if err := service.StopService(); err != nil {
			log.Printf("警告: 停止服務時發生錯誤: %v", err)
		}

		if err := service.UninstallService(); err != nil {
			log.Fatalf("卸載服務失敗: %v", err)
		}
		fmt.Println("✓ 服務卸載成功")
		return
	}

	// 啟動服務
	if *startFlag {
		if err := service.StartService(); err != nil {
			log.Fatalf("啟動服務失敗: %v", err)
		}
		fmt.Println("✓ 服務啟動成功")
		return
	}

	// 停止服務
	if *stopFlag {
		if err := service.StopService(); err != nil {
			log.Fatalf("停止服務失敗: %v", err)
		}
		fmt.Println("✓ 服務停止成功")
		return
	}

	// 重啟服務
	if *restartFlag {
		if err := service.RestartService(); err != nil {
			log.Fatalf("重啟服務失敗: %v", err)
		}
		fmt.Println("✓ 服務重啟成功")
		return
	}

	// 載入配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("警告: 無法載入配置檔 %s: %v，使用預設配置", *configPath, err)
		cfg = config.Default()
	}

	// 以服務模式運行
	if *serviceFlag {
		if err := service.RunService(cfg); err != nil {
			log.Fatal(err)
		}
		return
	}

	// 一般模式運行（前台）
	fmt.Printf("Host Agent v%s 啟動於端口 %d (前台模式)\n", version, cfg.Server.Port)
	fmt.Println("按 Ctrl+C 停止")

	prg := service.NewProgram(cfg)
	if err := prg.Start(nil); err != nil {
		log.Fatalf("啟動服務失敗: %v", err)
	}

	// 等待中斷信號
	waitForSignal()

	if err := prg.Stop(nil); err != nil {
		log.Printf("停止服務時發生錯誤: %v", err)
	}
}

func getDefaultConfigPath() string {
	if isWindows() {
		return "C:\\Program Files\\HostAgent\\config.yaml"
	}
	return "/etc/host-agent/config.yaml"
}

func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

func waitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}
