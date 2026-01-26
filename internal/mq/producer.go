package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// TaskProducer 任务生产者
type TaskProducer struct {
	mq *RabbitMQ
}

// NewTaskProducer 创建任务生产者
func NewTaskProducer(mq *RabbitMQ) *TaskProducer {
	return &TaskProducer{mq: mq}
}

// Publish 发布任务到队列
func (p *TaskProducer) Publish(task *dto.CrawlTask) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化任务失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.mq.Channel().PublishWithContext(
		ctx,
		"",               // exchange
		p.mq.QueueName(), // routing key (队列名)
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // 持久化消息
			ContentType:  "application/json",
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("发布任务失败: %w", err)
	}

	logger.Info.Printf("任务已发布到队列: %s", task.TaskID)
	return nil
}
