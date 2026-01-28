package scheduler

import (
	"time"

	"github.com/robfig/cron/v3"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/model"
	"github.com/yuanhuaxi/weibo-spider/internal/repository"
	"github.com/yuanhuaxi/weibo-spider/internal/service"
	"github.com/yuanhuaxi/weibo-spider/internal/store"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron      *cron.Cron
	spider    *service.SpiderService
	taskRepo  *repository.TaskRepository
	userStore *store.UserStore
}

// New 创建调度器
func New(spider *service.SpiderService, taskRepo *repository.TaskRepository, userStore *store.UserStore) *Scheduler {
	return &Scheduler{
		cron:      cron.New(),
		spider:    spider,
		taskRepo:  taskRepo,
		userStore: userStore,
	}
}

// LoadTasks 从数据库加载任务
func (s *Scheduler) LoadTasks() error {
	tasks, err := s.taskRepo.FindAllEnabled()
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if err := s.addTask(task); err != nil {
			logger.Error.Printf("添加任务失败 [%s]: %v", task.Name, err)
		}
	}

	logger.Info.Printf("已加载 %d 个定时任务", len(tasks))
	return nil
}

// addTask 添加单个任务
func (s *Scheduler) addTask(task *model.ScheduledTask) error {
	_, err := s.cron.AddFunc(task.Cron, func() {
		logger.Info.Printf("执行定时任务: %s", task.Name)

		crawlTask := &dto.CrawlTask{
			UserID:    task.UserID,
			SinceDate: task.SinceDate,
			EndDate:   task.EndDate,
			Filter:    task.Filter,
		}

		// 如果 EndDate 为零值，使用当前时间
		if crawlTask.EndDate.IsZero() {
			crawlTask.EndDate = time.Now()
		}

		if _, err := s.spider.Run(crawlTask); err != nil {
			logger.Error.Printf("任务执行失败 [%s]: %v", task.Name, err)
		}
	})
	return err
}

// Start 启动调度器
func (s *Scheduler) Start() {
	// 添加内置任务：每天凌晨2点同步特别关注
	s.addSpecialFollowSyncTask()

	s.cron.Start()
	logger.Info.Println("定时任务调度器已启动")
}

// addSpecialFollowSyncTask 添加特别关注同步定时任务
func (s *Scheduler) addSpecialFollowSyncTask() {
	// 每天凌晨2点执行
	_, err := s.cron.AddFunc("0 2 * * *", func() {
		logger.Info.Println("执行定时任务: 同步所有用户的特别关注")

		// 获取所有用户
		users, _, err := s.userStore.List(1, 1000)
		if err != nil {
			logger.Error.Printf("获取用户列表失败: %v", err)
			return
		}

		for _, user := range users {
			if user.WeiboCookie == "" {
				continue
			}
			result, err := s.spider.SyncSpecialFollows(user.ID)
			if err != nil {
				logger.Error.Printf("同步用户 %d 的特别关注失败: %v", user.ID, err)
				continue
			}
			logger.Info.Printf("用户 %d 同步特别关注完成，共 %d 个", user.ID, result.Total)
		}
	})

	if err != nil {
		logger.Error.Printf("添加特别关注同步任务失败: %v", err)
	} else {
		logger.Info.Println("已添加定时任务: 每天凌晨2点同步特别关注")
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	logger.Info.Println("定时任务调度器已停止")
}
