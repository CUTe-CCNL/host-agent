package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

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
		Mode     string        `yaml:"mode"` // "http", "kafka", "both"
		Interval time.Duration `yaml:"interval"`
		Timeout  time.Duration `yaml:"timeout"`

		// HTTP 模式設定
		HTTP struct {
			Endpoint string `yaml:"endpoint"`
		} `yaml:"http"`

		// Kafka 模式設定
		Kafka struct {
			Brokers      []string      `yaml:"brokers"`       // ["localhost:9092"]
			Topic        string        `yaml:"topic"`         // "host-metrics"
			Compression  string        `yaml:"compression"`   // "none", "gzip", "snappy", "lz4", "zstd"
			RequiredAcks int           `yaml:"required_acks"` // 0, 1, -1 (all)
			MaxRetries   int           `yaml:"max_retries"`   // 重試次數
			RetryBackoff time.Duration `yaml:"retry_backoff"` // 重試間隔

			// SASL 認證（可選）
			SASL struct {
				Enabled   bool   `yaml:"enabled"`
				Mechanism string `yaml:"mechanism"` // "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512"
				Username  string `yaml:"username"`
				Password  string `yaml:"password"`
			} `yaml:"sasl"`

			// TLS 設定（可選）
			TLS struct {
				Enabled            bool   `yaml:"enabled"`
				CertFile           string `yaml:"cert_file"`
				KeyFile            string `yaml:"key_file"`
				CAFile             string `yaml:"ca_file"`
				InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
			} `yaml:"tls"`
		} `yaml:"kafka"`
	} `yaml:"report"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
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
	cfg.Report.Mode = "kafka"
	cfg.Report.Interval = 30 * time.Second
	cfg.Report.Timeout = 10 * time.Second

	// Kafka 預設值
	cfg.Report.Kafka.Brokers = []string{"localhost:9092"}
	cfg.Report.Kafka.Topic = "host-metrics"
	cfg.Report.Kafka.Compression = "gzip"
	cfg.Report.Kafka.RequiredAcks = 1
	cfg.Report.Kafka.MaxRetries = 3
	cfg.Report.Kafka.RetryBackoff = 100 * time.Millisecond

	return cfg
}
