package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"host-agent/config"
	"host-agent/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQProducer struct {
	conn               *amqp.Connection
	channel            *amqp.Channel
	confirms           chan amqp.Confirmation
	exchange           string
	routingKeyTemplate string
	timeoutConfig      *config.Config
	mu                 sync.Mutex
}

func NewRabbitMQProducer(cfg *config.Config) (*RabbitMQProducer, error) {
	conn, err := amqp.DialConfig(cfg.Report.RabbitMQ.URL, amqp.Config{
		Locale: "en_US",
		Dial:   amqp.DefaultDial(cfg.Report.Timeout),
	})
	if err != nil {
		return nil, fmt.Errorf("連接 RabbitMQ 失敗: %v", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("開啟 RabbitMQ channel 失敗: %v", err)
	}

	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("啟用 RabbitMQ publisher confirms 失敗: %v", err)
	}

	if err := channel.ExchangeDeclare(
		cfg.Report.RabbitMQ.Exchange,
		cfg.Report.RabbitMQ.ExchangeType,
		cfg.Report.RabbitMQ.Durable,
		cfg.Report.RabbitMQ.AutoDelete,
		false,
		false,
		nil,
	); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("宣告 RabbitMQ exchange 失敗: %v", err)
	}

	confirms := make(chan amqp.Confirmation, 16)
	channel.NotifyPublish(confirms)

	log.Printf(
		"RabbitMQ Producer 已連接到: %s, Exchange: %s",
		cfg.Report.RabbitMQ.URL,
		cfg.Report.RabbitMQ.Exchange,
	)

	return &RabbitMQProducer{
		conn:               conn,
		channel:            channel,
		confirms:           confirms,
		exchange:           cfg.Report.RabbitMQ.Exchange,
		routingKeyTemplate: cfg.Report.RabbitMQ.RoutingKeyTemplate,
		timeoutConfig:      cfg,
	}, nil
}

func (rp *RabbitMQProducer) SendMetrics(metrics *models.Metrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("序列化指標失敗: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rp.timeoutConfig.Report.Timeout)
	defer cancel()

	routingKey := renderRoutingKey(rp.routingKeyTemplate, metrics.Hostname)

	rp.mu.Lock()
	defer rp.mu.Unlock()

	expectedDeliveryTag := rp.channel.GetNextPublishSeqNo()

	err = rp.channel.PublishWithContext(
		ctx,
		rp.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    metrics.Timestamp,
			Body:         data,
		},
	)
	if err != nil {
		return fmt.Errorf("發送訊息到 RabbitMQ 失敗: %v", err)
	}

	for {
		select {
		case confirmation, ok := <-rp.confirms:
			if !ok {
				return fmt.Errorf("RabbitMQ publisher confirms channel 已關閉")
			}
			if confirmation.DeliveryTag < expectedDeliveryTag {
				continue
			}
			if confirmation.DeliveryTag > expectedDeliveryTag {
				return fmt.Errorf("RabbitMQ publisher confirm 順序錯誤 [DeliveryTag: %d, Expected: %d]", confirmation.DeliveryTag, expectedDeliveryTag)
			}
			if !confirmation.Ack {
				return fmt.Errorf("RabbitMQ 未確認訊息 [DeliveryTag: %d]", confirmation.DeliveryTag)
			}
			log.Printf("成功發送到 RabbitMQ [Exchange: %s, RoutingKey: %s, Size: %d bytes]", rp.exchange, routingKey, len(data))
			return nil
		case <-ctx.Done():
			return fmt.Errorf("等待 RabbitMQ publisher confirm 超時: %v", ctx.Err())
		}
	}
}

func (rp *RabbitMQProducer) Close() error {
	if rp.channel != nil {
		if err := rp.channel.Close(); err != nil {
			_ = rp.conn.Close()
			return err
		}
	}
	if rp.conn != nil {
		return rp.conn.Close()
	}
	return nil
}

func renderRoutingKey(template, hostname string) string {
	return strings.ReplaceAll(template, "{hostname}", hostname)
}
