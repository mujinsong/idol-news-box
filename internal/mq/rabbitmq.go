package mq

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// RabbitMQ RabbitMQ客户端
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	cfg     *config.RabbitMQConfig
	mu      sync.Mutex
}

// NewRabbitMQ 创建RabbitMQ客户端
func NewRabbitMQ(cfg *config.RabbitMQConfig) (*RabbitMQ, error) {
	r := &RabbitMQ{cfg: cfg}
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
