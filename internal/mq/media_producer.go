package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
)

// MediaProducer 媒体下载任务生产者
type MediaProducer struct {
	mq *RabbitMQ
}

// NewMediaProducer 创建媒体下载任务生产者
func NewMediaProducer(mq *RabbitMQ) *MediaProducer {
	return &MediaProducer{mq: mq}
}

// Publish 发布媒体下载任务
func (p *MediaProducer) Publish(task *dto.MediaDownloadTask) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化任务失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.mq.Channel().PublishWithContext(
		ctx,
		"",
		p.mq.MediaQueueName(),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		return fmt.Errorf("发布任务失败: %w", err)
	}

	return nil
}
