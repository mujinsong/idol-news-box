package service

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/mq"
	"github.com/yuanhuaxi/weibo-spider/internal/spider"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// SpiderService 爬虫服务
type SpiderService struct {
	cfg      *config.Config
	spider   *spider.Spider   // 复用的爬虫实例
	producer *mq.TaskProducer // MQ生产者
	running  bool
	mu       sync.Mutex
	tasks    map[string]*dto.CrawlTask // 任务存储
	taskMu   sync.RWMutex              // 任务存储锁
}

// NewSpiderService 创建爬虫服务
func NewSpiderService(cfg *config.Config) *SpiderService {
	return &SpiderService{
		cfg:    cfg,
		spider: spider.New(cfg),
		tasks:  make(map[string]*dto.CrawlTask),
	}
}

// SetProducer 设置MQ生产者
func (s *SpiderService) SetProducer(producer *mq.TaskProducer) {
	s.producer = producer
}

// ProcessTask 处理单个任务（供MQ消费者调用）
func (s *SpiderService) ProcessTask(task *dto.CrawlTask) error {
	// 存储任务用于状态查询
	s.taskMu.Lock()
	s.tasks[task.TaskID] = task
	s.taskMu.Unlock()

	// 更新任务状态为运行中
	s.updateTaskStatus(task.TaskID, dto.TaskStatusRunning, "")

	logger.Info.Printf("开始处理任务: %s, 用户: %s", task.TaskID, task.UserID)

	// 爬取微博
	weibos, err := s.spider.FetchWeibos(task)
	if err != nil {
		s.updateTaskStatus(task.TaskID, dto.TaskStatusFailed, err.Error())
		return err
	}

	// 更新进度
	s.taskMu.Lock()
	if t, ok := s.tasks[task.TaskID]; ok {
		t.Progress = &dto.TaskProgress{
			TotalWeibos:   len(weibos),
			CrawledWeibos: len(weibos),
		}
		for _, w := range weibos {
			t.Progress.TotalImages += len(w.OriginalPictures)
			if w.VideoURL != "" {
				t.Progress.TotalVideos++
			}
		}
	}
	s.taskMu.Unlock()

	// 如果需要下载媒体文件
	if task.DownloadMedia {
		s.updateTaskStatus(task.TaskID, dto.TaskStatusDownloading, "")
		s.downloadMedia(task, weibos)
	}

	s.updateTaskStatus(task.TaskID, dto.TaskStatusCompleted, "")
	logger.Info.Printf("任务完成: %s, 共爬取 %d 条微博", task.TaskID, len(weibos))
	return nil
}

// downloadMedia 下载图片和视频
func (s *SpiderService) downloadMedia(task *dto.CrawlTask, weibos []*dto.Weibo) {
	downloader := NewMediaDownloader(s.cfg)

	for _, weibo := range weibos {
		// 下载图片
		for _, imgURL := range weibo.OriginalPictures {
			if err := downloader.DownloadImage(task.UserID, weibo.ID, imgURL); err != nil {
				logger.Error.Printf("下载图片失败: %s, %v", imgURL, err)
			} else {
				s.incrementProgress(task.TaskID, "image")
			}
		}

		// 下载视频
		if weibo.VideoURL != "" {
			if err := downloader.DownloadVideo(task.UserID, weibo.ID, weibo.VideoURL); err != nil {
				logger.Error.Printf("下载视频失败: %s, %v", weibo.VideoURL, err)
			} else {
				s.incrementProgress(task.TaskID, "video")
			}
		}
	}
}

// incrementProgress 增加进度
func (s *SpiderService) incrementProgress(taskID, mediaType string) {
	s.taskMu.Lock()
	defer s.taskMu.Unlock()
	if t, ok := s.tasks[taskID]; ok && t.Progress != nil {
		if mediaType == "image" {
			t.Progress.DownloadedImages++
		} else if mediaType == "video" {
			t.Progress.DownloadedVideos++
		}
	}
}

// updateTaskStatus 更新任务状态
func (s *SpiderService) updateTaskStatus(taskID string, status dto.TaskStatus, errMsg string) {
	s.taskMu.Lock()
	defer s.taskMu.Unlock()
	if task, ok := s.tasks[taskID]; ok {
		task.Status = status
		task.UpdatedAt = time.Now()
		task.Error = errMsg
	}
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// SubmitTask 提交异步任务到MQ
func (s *SpiderService) SubmitTask(task *dto.CrawlTask) (*dto.TaskResponse, error) {
	// 生成任务ID
	task.TaskID = generateTaskID()
	task.Status = dto.TaskStatusPending
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Progress = &dto.TaskProgress{}

	// 存储任务用于状态查询
	s.taskMu.Lock()
	s.tasks[task.TaskID] = task
	s.taskMu.Unlock()

	// 发布到MQ
	if s.producer != nil {
		if err := s.producer.Publish(task); err != nil {
			s.taskMu.Lock()
			delete(s.tasks, task.TaskID)
			s.taskMu.Unlock()
			return nil, fmt.Errorf("发布任务到队列失败: %w", err)
		}
	} else {
		// 如果没有配置MQ，直接在协程中处理
		go s.ProcessTask(task)
	}

	return &dto.TaskResponse{
		TaskID:  task.TaskID,
		Status:  task.Status,
		Message: "任务提交成功",
	}, nil
}

// GetTaskStatus 获取任务状态
func (s *SpiderService) GetTaskStatus(taskID string) (*dto.TaskStatusResponse, error) {
	s.taskMu.RLock()
	defer s.taskMu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	return &dto.TaskStatusResponse{
		TaskID:    task.TaskID,
		UserID:    task.UserID,
		Status:    task.Status,
		Progress:  task.Progress,
		Error:     task.Error,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}, nil
}

// IsRunning 检查是否正在运行
func (s *SpiderService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Run 运行爬虫任务（同步方式，保留兼容）
func (s *SpiderService) Run(task *dto.CrawlTask) (*dto.CrawlResult, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("任务正在运行中")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	logger.Info.Printf("开始爬取任务: 用户 %s", task.UserID)
	result, err := s.spider.Run(task)
	if err != nil {
		return nil, err
	}

	// 转换为 DTO
	return &dto.CrawlResult{
		User:   result.User,
		Weibos: result.Weibos,
		Total:  len(result.Weibos),
	}, nil
}

// GetUserInfo 获取用户信息
func (s *SpiderService) GetUserInfo(userID string) (*dto.User, error) {
	return s.spider.FetchUserInfo(userID)
}

// GetWeibos 获取用户微博（同步方式，保留兼容）
func (s *SpiderService) GetWeibos(task *dto.CrawlTask) (*dto.WeiboList, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("任务正在运行中")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	weibos, err := s.spider.FetchWeibos(task)
	if err != nil {
		return nil, err
	}

	return &dto.WeiboList{
		Weibos: weibos,
		Total:  len(weibos),
	}, nil
}
