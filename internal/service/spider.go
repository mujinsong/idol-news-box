package service

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/model"
	"github.com/yuanhuaxi/weibo-spider/internal/mq"
	"github.com/yuanhuaxi/weibo-spider/internal/spider"
	"github.com/yuanhuaxi/weibo-spider/internal/store"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// SpiderService 爬虫服务
type SpiderService struct {
	cfg                *config.Config
	spider             *spider.Spider            // 复用的爬虫实例
	downloader         *MediaDownloader          // 复用的下载器
	producer           *mq.TaskProducer          // MQ生产者
	mediaProducer      *mq.MediaProducer         // 媒体下载MQ生产者
	specialFollowStore *store.SpecialFollowStore // 特别关注存储
	running            bool
	mu                 sync.Mutex
	tasks              map[string]*dto.CrawlTask // 任务存储
	taskMu             sync.RWMutex              // 任务存储锁
}

// NewSpiderService 创建爬虫服务
func NewSpiderService(cfg *config.Config) *SpiderService {
	return &SpiderService{
		cfg:        cfg,
		spider:     spider.New(cfg),
		downloader: NewMediaDownloader(cfg),
		tasks:      make(map[string]*dto.CrawlTask),
	}
}

// SetSpecialFollowStore 设置特别关注存储
func (s *SpiderService) SetSpecialFollowStore(store *store.SpecialFollowStore) {
	s.specialFollowStore = store
}

// SetProducer 设置MQ生产者
func (s *SpiderService) SetProducer(producer *mq.TaskProducer) {
	s.producer = producer
}

// SetMediaProducer 设置媒体下载MQ生产者
func (s *SpiderService) SetMediaProducer(producer *mq.MediaProducer) {
	s.mediaProducer = producer
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

// downloadTask 下载任务
type downloadTask struct {
	userID  string
	weiboID string
	url     string
	taskID  string
	isVideo bool
}

// downloadMedia 下载图片和视频（发送到MQ队列）
func (s *SpiderService) downloadMedia(task *dto.CrawlTask, weibos []*dto.Weibo) {
	// 如果有媒体生产者，发送到队列异步处理
	if s.mediaProducer != nil {
		s.sendMediaToQueue(task, weibos)
		return
	}

	// 否则直接下载（兼容旧逻辑）
	s.downloadMediaDirect(task, weibos)
}

// sendMediaToQueue 发送媒体下载任务到队列
func (s *SpiderService) sendMediaToQueue(task *dto.CrawlTask, weibos []*dto.Weibo) {
	for _, weibo := range weibos {
		for _, imgURL := range weibo.OriginalPictures {
			mediaTask := &dto.MediaDownloadTask{
				TaskID:    task.TaskID,
				UserID:    task.UserID,
				WeiboID:   weibo.ID,
				MediaType: dto.MediaTypeImage,
				URL:       imgURL,
				CreatedAt: time.Now(),
			}
			if err := s.mediaProducer.Publish(mediaTask); err != nil {
				logger.Error.Printf("发送图片下载任务失败: %v", err)
			}
		}
		if weibo.VideoURL != "" {
			mediaTask := &dto.MediaDownloadTask{
				TaskID:    task.TaskID,
				UserID:    task.UserID,
				WeiboID:   weibo.ID,
				MediaType: dto.MediaTypeVideo,
				URL:       weibo.VideoURL,
				CreatedAt: time.Now(),
			}
			if err := s.mediaProducer.Publish(mediaTask); err != nil {
				logger.Error.Printf("发送视频下载任务失败: %v", err)
			}
		}
	}
	logger.Info.Printf("已发送媒体下载任务到队列")
}

// downloadMediaDirect 直接下载媒体（并发下载）
func (s *SpiderService) downloadMediaDirect(task *dto.CrawlTask, weibos []*dto.Weibo) {
	// 收集所有下载任务
	var tasks []downloadTask
	for _, weibo := range weibos {
		for _, imgURL := range weibo.OriginalPictures {
			tasks = append(tasks, downloadTask{
				userID:  task.UserID,
				weiboID: weibo.ID,
				url:     imgURL,
				taskID:  task.TaskID,
				isVideo: false,
			})
		}
		if weibo.VideoURL != "" {
			tasks = append(tasks, downloadTask{
				userID:  task.UserID,
				weiboID: weibo.ID,
				url:     weibo.VideoURL,
				taskID:  task.TaskID,
				isVideo: true,
			})
		}
	}

	if len(tasks) == 0 {
		return
	}

	// 并发下载，限制并发数为5
	const maxWorkers = 10
	taskChan := make(chan downloadTask, len(tasks))
	var wg sync.WaitGroup

	// 启动 worker
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dt := range taskChan {
				var err error
				if dt.isVideo {
					err = s.downloader.DownloadVideo(dt.userID, dt.weiboID, dt.url)
				} else {
					err = s.downloader.DownloadImage(dt.userID, dt.weiboID, dt.url)
				}

				if err != nil {
					logger.Error.Printf("下载失败: %s, %v", dt.url, err)
				} else {
					if dt.isVideo {
						s.incrementProgress(dt.taskID, "video")
					} else {
						s.incrementProgress(dt.taskID, "image")
					}
				}
			}
		}()
	}

	// 发送任务
	for _, t := range tasks {
		taskChan <- t
	}
	close(taskChan)

	// 等待完成
	wg.Wait()
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

// HandleMediaDownload 处理媒体下载任务（供MQ消费者调用）
func (s *SpiderService) HandleMediaDownload(task *dto.MediaDownloadTask) error {
	var err error
	if task.MediaType == dto.MediaTypeVideo {
		err = s.downloader.DownloadVideo(task.UserID, task.WeiboID, task.URL)
	} else {
		err = s.downloader.DownloadImage(task.UserID, task.WeiboID, task.URL)
	}

	if err != nil {
		logger.Error.Printf("媒体下载失败: %s, %v", task.URL, err)
		return err
	}

	// 更新进度
	if task.MediaType == dto.MediaTypeVideo {
		s.incrementProgress(task.TaskID, "video")
	} else {
		s.incrementProgress(task.TaskID, "image")
	}

	return nil
}

// GetSpecialFollows 获取特别关注列表
func (s *SpiderService) GetSpecialFollows() (*dto.SpecialFollowList, error) {
	return s.spider.FetchSpecialFollows()
}

// SyncSpecialFollows 同步特别关注到数据库
func (s *SpiderService) SyncSpecialFollows() (*dto.SpecialFollowSyncResult, error) {
	if s.specialFollowStore == nil {
		return nil, fmt.Errorf("特别关注存储未初始化")
	}

	// 获取当前登录用户ID（Cookie所有者）
	ownerID, err := s.spider.FetchCurrentUserID()
	if err != nil {
		return nil, fmt.Errorf("获取当前用户ID失败: %w", err)
	}

	// 从API获取特别关注列表
	list, err := s.spider.FetchSpecialFollows()
	if err != nil {
		return nil, fmt.Errorf("获取特别关注列表失败: %w", err)
	}

	// 转换为model并保存
	now := time.Now()
	follows := make([]*model.SpecialFollow, len(list.Users))
	for i, u := range list.Users {
		follows[i] = &model.SpecialFollow{
			OwnerID:  ownerID,
			UserID:   u.ID,
			Nickname: u.Nickname,
			SyncedAt: now,
		}
	}

	if err := s.specialFollowStore.BatchUpsert(follows); err != nil {
		return nil, fmt.Errorf("保存特别关注失败: %w", err)
	}

	return &dto.SpecialFollowSyncResult{
		OwnerID:  ownerID,
		Total:    len(follows),
		SyncedAt: now,
	}, nil
}

// GetSpecialFollowsFromDB 从数据库获取特别关注列表
func (s *SpiderService) GetSpecialFollowsFromDB() (*dto.SpecialFollowDBList, error) {
	if s.specialFollowStore == nil {
		return nil, fmt.Errorf("特别关注存储未初始化")
	}

	follows, err := s.specialFollowStore.List()
	if err != nil {
		return nil, fmt.Errorf("查询特别关注失败: %w", err)
	}

	users := make([]dto.SpecialFollowDBUser, len(follows))
	for i, f := range follows {
		users[i] = dto.SpecialFollowDBUser{
			ID:        f.ID,
			OwnerID:   f.OwnerID,
			UserID:    f.UserID,
			Nickname:  f.Nickname,
			SyncedAt:  f.SyncedAt,
			CreatedAt: f.CreatedAt,
		}
	}

	return &dto.SpecialFollowDBList{
		Users: users,
		Total: len(users),
	}, nil
}

// DeleteSpecialFollow 删除特别关注记录
func (s *SpiderService) DeleteSpecialFollow(userID string) error {
	if s.specialFollowStore == nil {
		return fmt.Errorf("特别关注存储未初始化")
	}

	// 获取当前登录用户ID
	ownerID, err := s.spider.FetchCurrentUserID()
	if err != nil {
		return fmt.Errorf("获取当前用户ID失败: %w", err)
	}

	return s.specialFollowStore.Delete(ownerID, userID)
}
