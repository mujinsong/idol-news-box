package service

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/spider"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// SpiderService 爬虫服务
type SpiderService struct {
	cfg       *config.Config
	running   bool
	mu        sync.Mutex
	taskChan  chan *dto.CrawlTask       // 任务通道
	tasks     map[string]*dto.CrawlTask // 任务存储
	taskMu    sync.RWMutex              // 任务存储锁
}

// NewSpiderService 创建爬虫服务
func NewSpiderService(cfg *config.Config) *SpiderService {
	s := &SpiderService{
		cfg:      cfg,
		taskChan: make(chan *dto.CrawlTask, 100), // 缓冲100个任务
		tasks:    make(map[string]*dto.CrawlTask),
	}
	return s
}

// Start 启动任务监听
func (s *SpiderService) Start() {
	go s.taskWorker()
	logger.Info.Println("任务监听器已启动")
}

// taskWorker 任务处理协程
func (s *SpiderService) taskWorker() {
	for task := range s.taskChan {
		s.processTask(task)
	}
}

// processTask 处理单个任务
func (s *SpiderService) processTask(task *dto.CrawlTask) {
	// 更新任务状态为运行中
	s.updateTaskStatus(task.TaskID, dto.TaskStatusRunning, "")

	sp, err := spider.New(s.cfg)
	if err != nil {
		s.updateTaskStatus(task.TaskID, dto.TaskStatusFailed, err.Error())
		return
	}

	logger.Info.Printf("开始处理任务: %s, 用户: %s", task.TaskID, task.UserID)

	// 爬取微博
	weibos, err := sp.FetchWeibos(task)
	if err != nil {
		s.updateTaskStatus(task.TaskID, dto.TaskStatusFailed, err.Error())
		return
	}

	// 更新进度
	s.taskMu.Lock()
	if t, ok := s.tasks[task.TaskID]; ok {
		t.Progress = &dto.TaskProgress{
			TotalWeibos:   len(weibos),
			CrawledWeibos: len(weibos),
		}
		// 统计图片和视频数量
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
		s.downloadMedia(task, weibos, sp)
	}

	s.updateTaskStatus(task.TaskID, dto.TaskStatusCompleted, "")
	logger.Info.Printf("任务完成: %s, 共爬取 %d 条微博", task.TaskID, len(weibos))
}

// downloadMedia 下载图片和视频
func (s *SpiderService) downloadMedia(task *dto.CrawlTask, weibos []*dto.Weibo, sp *spider.Spider) {
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

// SubmitTask 提交异步任务
func (s *SpiderService) SubmitTask(task *dto.CrawlTask) (*dto.TaskResponse, error) {
	// 生成任务ID
	task.TaskID = generateTaskID()
	task.Status = dto.TaskStatusPending
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Progress = &dto.TaskProgress{}

	// 存储任务
	s.taskMu.Lock()
	s.tasks[task.TaskID] = task
	s.taskMu.Unlock()

	// 发送到任务通道
	select {
	case s.taskChan <- task:
		return &dto.TaskResponse{
			TaskID:  task.TaskID,
			Status:  task.Status,
			Message: "任务提交成功",
		}, nil
	default:
		s.taskMu.Lock()
		delete(s.tasks, task.TaskID)
		s.taskMu.Unlock()
		return nil, fmt.Errorf("任务队列已满，请稍后重试")
	}
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

	sp, err := spider.New(s.cfg)
	if err != nil {
		return nil, err
	}

	logger.Info.Printf("开始爬取任务: 用户 %s", task.UserID)
	result, err := sp.Run(task)
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
	sp, err := spider.New(s.cfg)
	if err != nil {
		return nil, err
	}

	return sp.FetchUserInfo(userID)
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

	sp, err := spider.New(s.cfg)
	if err != nil {
		return nil, err
	}

	weibos, err := sp.FetchWeibos(task)
	if err != nil {
		return nil, err
	}

	return &dto.WeiboList{
		Weibos: weibos,
		Total:  len(weibos),
	}, nil
}
