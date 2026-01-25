package dto

import "time"

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"    // 等待中
	TaskStatusRunning    TaskStatus = "running"    // 运行中
	TaskStatusCompleted  TaskStatus = "completed"  // 已完成
	TaskStatusFailed     TaskStatus = "failed"     // 失败
	TaskStatusDownloading TaskStatus = "downloading" // 下载媒体中
)

// CrawlTask 爬取任务参数
type CrawlTask struct {
	TaskID        string     `json:"task_id"`
	UserID        string     `json:"user_id"`
	SinceDate     time.Time  `json:"since_date"`
	EndDate       time.Time  `json:"end_date"`
	Filter        int        `json:"filter"`
	DownloadMedia bool       `json:"download_media"` // 是否下载图片和视频
	Status        TaskStatus `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Error         string     `json:"error,omitempty"`
	Progress      *TaskProgress `json:"progress,omitempty"`
}

// TaskProgress 任务进度
type TaskProgress struct {
	TotalWeibos      int `json:"total_weibos"`
	CrawledWeibos    int `json:"crawled_weibos"`
	TotalImages      int `json:"total_images"`
	DownloadedImages int `json:"downloaded_images"`
	TotalVideos      int `json:"total_videos"`
	DownloadedVideos int `json:"downloaded_videos"`
}

// CrawlRequest 爬取请求
type CrawlRequest struct {
	UserID        string `json:"user_id" binding:"required"`
	SinceDate     string `json:"since_date"`
	EndDate       string `json:"end_date"`
	Filter        int    `json:"filter"`
	DownloadMedia bool   `json:"download_media"` // 是否下载图片和视频
}

// CrawlResult 爬取结果
type CrawlResult struct {
	User   *User    `json:"user"`
	Weibos []*Weibo `json:"weibos"`
	Total  int      `json:"total"`
}

// TaskResponse 任务提交响应
type TaskResponse struct {
	TaskID  string     `json:"task_id"`
	Status  TaskStatus `json:"status"`
	Message string     `json:"message"`
}

// TaskStatusResponse 任务状态响应
type TaskStatusResponse struct {
	TaskID    string        `json:"task_id"`
	UserID    string        `json:"user_id"`
	Status    TaskStatus    `json:"status"`
	Progress  *TaskProgress `json:"progress,omitempty"`
	Error     string        `json:"error,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}