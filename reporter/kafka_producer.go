package reporter

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"host-agent/config"
	"host-agent/models"

	"github.com/IBM/sarama"
)

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
	config   *config.Config
}

func NewKafkaProducer(cfg *config.Config) (*KafkaProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	// 設定壓縮
	switch cfg.Report.Kafka.Compression {
	case "gzip":
		saramaConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		saramaConfig.Producer.Compression = sarama.CompressionNone
	}

	// 設定 ACK
	saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(cfg.Report.Kafka.RequiredAcks)

	// 設定重試
	saramaConfig.Producer.Retry.Max = cfg.Report.Kafka.MaxRetries
	saramaConfig.Producer.Retry.Backoff = cfg.Report.Kafka.RetryBackoff

	// 設定超時
	saramaConfig.Producer.Timeout = cfg.Report.Timeout
	saramaConfig.Net.DialTimeout = 10 * time.Second
	saramaConfig.Net.ReadTimeout = 10 * time.Second
	saramaConfig.Net.WriteTimeout = 10 * time.Second

	// SASL 認證
	if cfg.Report.Kafka.SASL.Enabled {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = cfg.Report.Kafka.SASL.Username
		saramaConfig.Net.SASL.Password = cfg.Report.Kafka.SASL.Password

		switch cfg.Report.Kafka.SASL.Mechanism {
		case "SCRAM-SHA-256":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			}
		case "SCRAM-SHA-512":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
			saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
			}
		case "PLAIN":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		default:
			return nil, fmt.Errorf("不支援的 SASL 機制: %s", cfg.Report.Kafka.SASL.Mechanism)
		}
	}

	// TLS 設定
	if cfg.Report.Kafka.TLS.Enabled {
		tlsConfig, err := createTLSConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("建立 TLS 配置失敗: %v", err)
		}
		saramaConfig.Net.TLS.Enable = true
		saramaConfig.Net.TLS.Config = tlsConfig
	}

	// 建立 Producer
	producer, err := sarama.NewSyncProducer(cfg.Report.Kafka.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("建立 Kafka Producer 失敗: %v", err)
	}

	log.Printf("Kafka Producer 已連接到: %v, Topic: %s", cfg.Report.Kafka.Brokers, cfg.Report.Kafka.Topic)

	return &KafkaProducer{
		producer: producer,
		topic:    cfg.Report.Kafka.Topic,
		config:   cfg,
	}, nil
}

func (kp *KafkaProducer) SendMetrics(metrics *models.Metrics) error {
	// 序列化為 JSON
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("序列化指標失敗: %v", err)
	}

	// 建立訊息
	message := &sarama.ProducerMessage{
		Topic:     kp.topic,
		Key:       sarama.StringEncoder(metrics.Hostname), // 使用 hostname 作為 key，相同主機的資料會到同一個 partition
		Value:     sarama.ByteEncoder(data),
		Timestamp: metrics.Timestamp,
	}

	// 發送
	partition, offset, err := kp.producer.SendMessage(message)
	if err != nil {
		return fmt.Errorf("發送訊息到 Kafka 失敗: %v", err)
	}

	log.Printf("成功發送到 Kafka [Topic: %s, Partition: %d, Offset: %d, Size: %d bytes]",
		kp.topic, partition, offset, len(data))

	return nil
}

func (kp *KafkaProducer) Close() error {
	if kp.producer != nil {
		return kp.producer.Close()
	}
	return nil
}

// createTLSConfig 建立 TLS 配置
func createTLSConfig(cfg *config.Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Report.Kafka.TLS.InsecureSkipVerify,
	}

	// 載入 CA 證書
	if cfg.Report.Kafka.TLS.CAFile != "" {
		caCert, err := os.ReadFile(cfg.Report.Kafka.TLS.CAFile)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("無法解析 CA 證書")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// 載入客戶端證書
	if cfg.Report.Kafka.TLS.CertFile != "" && cfg.Report.Kafka.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.Report.Kafka.TLS.CertFile, cfg.Report.Kafka.TLS.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}
