package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type PluginConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Directory      string        `yaml:"directory"`
	StartupTimeout time.Duration `yaml:"startup_timeout"`
	HealthInterval time.Duration `yaml:"health_interval"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

type Config struct {
	Server struct {
		Port         int           `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
		EnableAuth   bool          `yaml:"enable_auth"`
		AuthToken    string        `yaml:"auth_token"`
	} `yaml:"server"`

	Collector struct {
		Interval        time.Duration `yaml:"interval"`
		EnableCPU       bool          `yaml:"enable_cpu"`
		EnableMemory    bool          `yaml:"enable_memory"`
		EnableDisk      bool          `yaml:"enable_disk"`
		EnableNetwork   bool          `yaml:"enable_network"`
		EnableProcess   bool          `yaml:"enable_process"`
		DiskMountPoints []string      `yaml:"disk_mount_points"`
		ProcessLimit    int           `yaml:"process_limit"`
	} `yaml:"collector"`

	Report struct {
		Enabled  bool          `yaml:"enabled"`
		Mode     string        `yaml:"mode"` // "http", "rabbitmq", "both"
		Interval time.Duration `yaml:"interval"`
		Timeout  time.Duration `yaml:"timeout"`

		// HTTP 模式設定
		HTTP struct {
			Endpoint string `yaml:"endpoint"`
		} `yaml:"http"`

		// RabbitMQ 模式設定
		RabbitMQ struct {
			URL                string `yaml:"url"`
			Exchange           string `yaml:"exchange"`
			ExchangeType       string `yaml:"exchange_type"`
			RoutingKeyTemplate string `yaml:"routing_key_template"`
			Durable            bool   `yaml:"durable"`
			AutoDelete         bool   `yaml:"auto_delete"`
		} `yaml:"rabbitmq"`
	} `yaml:"report"`

	Plugins PluginConfig `yaml:"plugins"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Default() *Config {
	cfg := &Config{}
	cfg.Server.Port = 9100
	cfg.Server.ReadTimeout = 15 * time.Second
	cfg.Server.WriteTimeout = 15 * time.Second
	cfg.Server.EnableAuth = false

	cfg.Collector.Interval = 5 * time.Second
	cfg.Collector.EnableCPU = true
	cfg.Collector.EnableMemory = true
	cfg.Collector.EnableDisk = true
	cfg.Collector.EnableNetwork = true
	cfg.Collector.EnableProcess = false
	cfg.Collector.ProcessLimit = 10

	cfg.Report.Enabled = false
	cfg.Report.Mode = "rabbitmq"
	cfg.Report.Interval = 30 * time.Second
	cfg.Report.Timeout = 10 * time.Second

	// RabbitMQ 預設值
	cfg.Report.RabbitMQ.URL = "amqp://guest:guest@localhost:5672/"
	cfg.Report.RabbitMQ.Exchange = "host-metrics"
	cfg.Report.RabbitMQ.ExchangeType = "topic"
	cfg.Report.RabbitMQ.RoutingKeyTemplate = "host.metrics"
	cfg.Report.RabbitMQ.Durable = true
	cfg.Report.RabbitMQ.AutoDelete = false

	cfg.Plugins.Enabled = false
	cfg.Plugins.Directory = "/etc/host-agent/plugins.d"
	cfg.Plugins.StartupTimeout = 10 * time.Second
	cfg.Plugins.HealthInterval = 15 * time.Second
	cfg.Plugins.RequestTimeout = 30 * time.Second

	return cfg
}
