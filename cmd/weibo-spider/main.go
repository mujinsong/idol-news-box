package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/internal/mq"
	"github.com/yuanhuaxi/weibo-spider/internal/repository"
	"github.com/yuanhuaxi/weibo-spider/internal/scheduler"
	"github.com/yuanhuaxi/weibo-spider/internal/service"
	"github.com/yuanhuaxi/weibo-spider/internal/store"
	"github.com/yuanhuaxi/weibo-spider/internal/webserver"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "config.json", "配置文件路径")
}

func main() {
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error.Printf("加载配置失败: %v", err)
		os.Exit(1)
	}

	// 初始化数据库
	dbFactory, err := store.InitDB(&cfg.Database)
	if err != nil {
		logger.Error.Printf("%v", err)
		os.Exit(1)
	}

	// 创建上下文用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建爬虫服务
	spiderSvc := service.NewSpiderService(cfg)

	// 设置特别关注存储
	specialFollowStore := store.NewSpecialFollowStore(dbFactory.GetDB())
	spiderSvc.SetSpecialFollowStore(specialFollowStore)

	// 初始化RabbitMQ
	var rabbitMQ *mq.RabbitMQ
	if cfg.RabbitMQ.URL != "" {
		rabbitMQ, err = mq.NewRabbitMQ(&cfg.RabbitMQ)
		if err != nil {
			logger.Error.Printf("RabbitMQ连接失败: %v", err)
			os.Exit(1)
		}
		defer rabbitMQ.Close()

		// 设置任务生产者
		producer := mq.NewTaskProducer(rabbitMQ)
		spiderSvc.SetProducer(producer)

		// 设置媒体下载生产者
		mediaProducer := mq.NewMediaProducer(rabbitMQ)
		spiderSvc.SetMediaProducer(mediaProducer)

		// 启动任务消费者
		consumer := mq.NewTaskConsumer(rabbitMQ, spiderSvc.ProcessTask)
		if err := consumer.Start(ctx); err != nil {
			logger.Error.Printf("启动MQ消费者失败: %v", err)
			os.Exit(1)
		}

		// 启动媒体下载消费者
		mediaConsumer := mq.NewMediaConsumer(rabbitMQ, spiderSvc.HandleMediaDownload)
		if err := mediaConsumer.Start(ctx); err != nil {
			logger.Error.Printf("启动媒体下载消费者失败: %v", err)
			os.Exit(1)
		}
		logger.Info.Println("RabbitMQ任务队列已启动")
	} else {
		logger.Warn.Println("未配置RabbitMQ，任务将在本地处理")
	}

	// 创建任务仓库
	taskRepo := repository.NewTaskRepository()

	// 创建用户服务
	userStore := store.NewUserStore(dbFactory.GetDB())
	userSvc := service.NewUserService(userStore)

	// 设置用户存储到爬虫服务（用于同步特别关注时获取用户Cookie）
	spiderSvc.SetUserStore(userStore)

	// 创建认证服务
	authSvc := service.NewAuthService(userStore)

	// 启动定时任务调度器
	sched := scheduler.New(spiderSvc, taskRepo, userStore)
	if err := sched.LoadTasks(); err != nil {
		logger.Error.Printf("加载定时任务失败: %v", err)
	}
	sched.Start()

	// 启动HTTP服务
	server := webserver.NewServer(spiderSvc, userSvc, authSvc, cfg.Server.Mode)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logger.Info.Printf("服务启动在 %s", addr)

	go server.Run(addr)

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info.Println("正在关闭服务...")
	cancel() // 通知消费者停止
	sched.Stop()
}
