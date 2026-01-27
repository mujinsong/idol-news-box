package mq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// TaskHandler 任务处理函数
type TaskHandler func(task *dto.CrawlTask) error

// TaskConsumer 任务消费者
type TaskConsumer struct {
	mq      *RabbitMQ
	handler TaskHandler
}

// NewTaskConsumer 创建任务消费者
func NewTaskConsumer(mq *RabbitMQ, handler TaskHandler) *TaskConsumer {
	return &TaskConsumer{
		mq:      mq,
		handler: handler,
	}
}

// getRetryCount 从消息头获取重试次数
func getRetryCount(msg amqp.Delivery) int {
	if msg.Headers == nil {
		return 0
	}
	if count, ok := msg.Headers[RetryCountHeader]; ok {
		if n, ok := count.(int32); ok {
			return int(n)
		}
	}
	return 0
}

// Start 启动消费者
func (c *TaskConsumer) Start(ctx context.Context) error {
	// 设置预取数量，控制并发
	if err := c.mq.Channel().Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := c.mq.Channel().Consume(
		c.mq.QueueName(), // 队列名
		"",               // 消费者标签
		false,            // 手动确认
		false,            // 非排他
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	if err != nil {
		return err
	}

	logger.Info.Printf("开始监听任务队列: %s", c.mq.QueueName())

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info.Println("任务消费者已停止")
				return
			case msg, ok := <-msgs:
				if !ok {
					logger.Warn.Println("消息通道已关闭")
					return
				}

				var task dto.CrawlTask
				if err := json.Unmarshal(msg.Body, &task); err != nil {
					logger.Error.Printf("解析任务失败: %v", err)
					msg.Nack(false, false) // 拒绝消息，不重新入队
					continue
				}

				logger.Info.Printf("收到任务: %s, 用户: %s", task.TaskID, task.UserID)

				retryCount := getRetryCount(msg)
				if err := c.handler(&task); err != nil {
					logger.Error.Printf("处理任务失败: %s, 重试次数: %d, %v", task.TaskID, retryCount, err)

					if retryCount >= MaxRetryCount {
						// 超过最大重试次数，发送到死信队列
						logger.Warn.Printf("任务 %s 超过最大重试次数，移入死信队列", task.TaskID)
						c.publishToDeadLetter(msg.Body, retryCount, err.Error())
						_ = msg.Ack(false)
					} else {
						// 重新发布消息，增加重试次数
						c.republishWithRetry(msg.Body, retryCount+1)
						_ = msg.Ack(false)
					}
					continue
				}

				_ = msg.Ack(false) // 确认消息
				logger.Info.Printf("任务处理完成: %s", task.TaskID)
			}
		}
	}()

	return nil
}

// republishWithRetry 重新发布消息并增加重试次数
func (c *TaskConsumer) republishWithRetry(body []byte, retryCount int) {
	err := c.mq.Channel().Publish(
		"",
		c.mq.QueueName(),
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
		logger.Error.Printf("重新发布消息失败: %v", err)
	}
}

// publishToDeadLetter 发送消息到死信队列
func (c *TaskConsumer) publishToDeadLetter(body []byte, retryCount int, errMsg string) {
	err := c.mq.Channel().Publish(
		"",
		c.mq.DeadLetterQueueName(),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Headers: amqp.Table{
				RetryCountHeader: int32(retryCount),
				"x-error":        errMsg,
			},
		},
	)
	if err != nil {
		logger.Error.Printf("发送到死信队列失败: %v", err)
	}
}
