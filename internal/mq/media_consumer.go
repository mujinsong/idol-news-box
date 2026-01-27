package mq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// MediaHandler 媒体下载处理函数
type MediaHandler func(task *dto.MediaDownloadTask) error

// MediaConsumer 媒体下载消费者
type MediaConsumer struct {
	mq      *RabbitMQ
	handler MediaHandler
}

// NewMediaConsumer 创建媒体下载消费者
func NewMediaConsumer(mq *RabbitMQ, handler MediaHandler) *MediaConsumer {
	return &MediaConsumer{
		mq:      mq,
		handler: handler,
	}
}

// Start 启动消费者
func (c *MediaConsumer) Start(ctx context.Context) error {
	if err := c.mq.Channel().Qos(5, 0, false); err != nil {
		return err
	}

	msgs, err := c.mq.Channel().Consume(
		c.mq.MediaQueueName(),
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	logger.Info.Printf("开始监听媒体下载队列: %s", c.mq.MediaQueueName())

	go c.processMessages(ctx, msgs)
	return nil
}

// processMessages 处理消息
func (c *MediaConsumer) processMessages(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			logger.Info.Println("媒体下载消费者已停止")
			return
		case msg, ok := <-msgs:
			if !ok {
				logger.Warn.Println("媒体消息通道已关闭")
				return
			}
			c.handleMessage(msg)
		}
	}
}

// handleMessage 处理单条消息
func (c *MediaConsumer) handleMessage(msg amqp.Delivery) {
	var task dto.MediaDownloadTask
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		logger.Error.Printf("解析媒体任务失败: %v", err)
		_ = msg.Nack(false, false)
		return
	}

	logger.Info.Printf("收到媒体下载任务: %s, 类型: %s", task.URL, task.MediaType)

	retryCount := getRetryCount(msg)
	if err := c.handler(&task); err != nil {
		logger.Error.Printf("媒体下载失败: %s, 重试次数: %d, %v", task.URL, retryCount, err)

		if retryCount >= MaxRetryCount {
			logger.Warn.Printf("媒体下载 %s 超过最大重试次数，丢弃", task.URL)
			_ = msg.Ack(false)
		} else {
			c.republishWithRetry(msg.Body, retryCount+1)
			_ = msg.Ack(false)
		}
		return
	}

	_ = msg.Ack(false)
	logger.Info.Printf("媒体下载完成: %s", task.URL)
}

// republishWithRetry 重新发布消息并增加重试次数
func (c *MediaConsumer) republishWithRetry(body []byte, retryCount int) {
	err := c.mq.Channel().Publish(
		"",
		c.mq.MediaQueueName(),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Headers: amqp.Table{
				RetryCountHeader: int32(retryCount),
			},
		},
	)
	if err != nil {
		logger.Error.Printf("重新发布媒体任务失败: %v", err)
	}
}
