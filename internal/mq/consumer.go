package mq

import (
	"context"
	"encoding/json"

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

				if err := c.handler(&task); err != nil {
					logger.Error.Printf("处理任务失败: %s, %v", task.TaskID, err)
					msg.Nack(false, true) // 拒绝消息，重新入队
					continue
				}

				msg.Ack(false) // 确认消息
				logger.Info.Printf("任务处理完成: %s", task.TaskID)
			}
		}
	}()

	return nil
}
