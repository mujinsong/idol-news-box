package mq

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

const (
	// MaxRetryCount 最大重试次数
	MaxRetryCount = 3
	// RetryCountHeader 重试次数头
	RetryCountHeader = "x-retry-count"
)

// RabbitMQ RabbitMQ客户端
type RabbitMQ struct {
	conn            *amqp.Connection
	channel         *amqp.Channel
	cfg             *config.RabbitMQConfig
	deadLetterQueue string
	mediaQueue      string
	mu              sync.Mutex
}

// NewRabbitMQ 创建RabbitMQ客户端
func NewRabbitMQ(cfg *config.RabbitMQConfig) (*RabbitMQ, error) {
	r := &RabbitMQ{
		cfg:             cfg,
		deadLetterQueue: cfg.TaskQueue + "_dead",
		mediaQueue:      cfg.MediaQueue,
	}
	if err := r.connect(); err != nil {
		return nil, err
	}
	return r, nil
}

// connect 建立连接
func (r *RabbitMQ) connect() error {
	conn, err := amqp.Dial(r.cfg.URL)
	if err != nil {
		return fmt.Errorf("连接RabbitMQ失败: %w", err)
	}
	r.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("创建Channel失败: %w", err)
	}
	r.channel = ch

	// 声明死信队列
	_, err = ch.QueueDeclare(
		r.deadLetterQueue,
		true,  // 持久化
		false, // 不自动删除
		false, // 非排他
		false, // 不等待
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明死信队列失败: %w", err)
	}

	// 声明任务队列
	_, err = ch.QueueDeclare(
		r.cfg.TaskQueue, // 队列名
		true,            // 持久化
		false,           // 不自动删除
		false,           // 非排他
		false,           // 不等待
		nil,             // 参数
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明队列失败: %w", err)
	}

	// 声明媒体下载队列
	if r.mediaQueue != "" {
		_, err = ch.QueueDeclare(
			r.mediaQueue,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			ch.Close()
			conn.Close()
			return fmt.Errorf("声明媒体队列失败: %w", err)
		}
	}

	logger.Info.Printf("RabbitMQ连接成功: %s", r.cfg.TaskQueue)
	return nil
}

// Channel 获取Channel
func (r *RabbitMQ) Channel() *amqp.Channel {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.channel
}

// QueueName 获取队列名
func (r *RabbitMQ) QueueName() string {
	return r.cfg.TaskQueue
}

// DeadLetterQueueName 获取死信队列名
func (r *RabbitMQ) DeadLetterQueueName() string {
	return r.deadLetterQueue
}

// MediaQueueName 获取媒体下载队列名
func (r *RabbitMQ) MediaQueueName() string {
	return r.mediaQueue
}

// Close 关闭连接
func (r *RabbitMQ) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
	logger.Info.Println("RabbitMQ连接已关闭")
}
