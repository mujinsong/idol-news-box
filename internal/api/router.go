package api

import (
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/base"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/service"
)

// Router API路由
type Router struct {
	engine *gin.Engine
	spider *service.SpiderService
}

// NewRouter 创建路由
func NewRouter(spider *service.SpiderService, mode string) *Router {
	gin.SetMode(mode)
	r := &Router{
		engine: gin.Default(),
		spider: spider,
	}
	r.setup()
	return r
}

// setup 设置路由
func (r *Router) setup() {
	api := r.engine.Group("/api/v1")
	{
		api.GET("/health", base.RecoverWrap(r.health))
		api.GET("/status", base.RecoverWrap(r.status))
		api.GET("/user/:user_id", base.RecoverWrap(r.getUserInfo))
		api.POST("/weibos", base.RecoverWrap(r.getWeibos))
		api.POST("/crawl", base.RecoverWrap(r.crawl))
		api.GET("/task/:task_id", base.RecoverWrap(r.getTaskStatus))
		api.GET("/image", base.RecoverWrap(r.downloadImage))
	}
}

// Run 启动服务
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}

// health 健康检查
func (r *Router) health(c *gin.Context) {
	base.OkResponse(c, gin.H{"status": "ok"})
}

// status 获取爬虫状态
func (r *Router) status(c *gin.Context) {
	base.OkResponse(c, gin.H{"running": r.spider.IsRunning()})
}

// getUserInfo 获取用户信息
func (r *Router) getUserInfo(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		base.ErrResponse(c, base.CodeBadRequest, "user_id 不能为空")
		return
	}

	result, err := r.spider.GetUserInfo(userID)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}

// getWeibos 提交微博爬取任务（异步）
func (r *Router) getWeibos(c *gin.Context) {
	var req dto.CrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "参数错误: "+err.Error())
		return
	}

	task := &dto.CrawlTask{
		UserID:        req.UserID,
		Filter:        req.Filter,
		DownloadMedia: req.DownloadMedia,
	}

	if req.SinceDate != "" {
		t, err := time.Parse("2006-01-02", req.SinceDate)
		if err != nil {
			base.ErrResponse(c, base.CodeBadRequest, "since_date 格式错误")
			return
		}
		task.SinceDate = t
	} else {
		// 默认一周前
		task.SinceDate = time.Now().AddDate(0, 0, -7)
	}

	if req.EndDate == "" || req.EndDate == "now" {
		task.EndDate = time.Now()
	} else {
		t, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			base.ErrResponse(c, base.CodeBadRequest, "end_date 格式错误")
			return
		}
		task.EndDate = t
	}

	// 提交异步任务
	result, err := r.spider.SubmitTask(task)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}

// crawl 触发爬取任务
func (r *Router) crawl(c *gin.Context) {
	if r.spider.IsRunning() {
		base.ErrResponse(c, base.CodeBusy, base.MsgBusy)
		return
	}

	var req dto.CrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "参数错误: "+err.Error())
		return
	}

	task := &dto.CrawlTask{
		UserID: req.UserID,
		Filter: req.Filter,
	}

	if req.SinceDate != "" {
		t, err := time.Parse("2006-01-02", req.SinceDate)
		if err != nil {
			base.ErrResponse(c, base.CodeBadRequest, "since_date 格式错误，应为 YYYY-MM-DD")
			return
		}
		task.SinceDate = t
	}

	if req.EndDate == "" || req.EndDate == "now" {
		task.EndDate = time.Now()
	} else {
		t, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			base.ErrResponse(c, base.CodeBadRequest, "end_date 格式错误，应为 YYYY-MM-DD")
			return
		}
		task.EndDate = t
	}

	result, err := r.spider.Run(task)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}

// getTaskStatus 获取任务状态
func (r *Router) getTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		base.ErrResponse(c, base.CodeBadRequest, "task_id 不能为空")
		return
	}

	result, err := r.spider.GetTaskStatus(taskID)
	if err != nil {
		base.ErrResponse(c, base.CodeNotFound, err.Error())
		return
	}

	base.OkResponse(c, result)
}

// downloadImage 下载微博图片（带Referer绕过防盗链）
func (r *Router) downloadImage(c *gin.Context) {
	imageURL := c.Query("url")
	if imageURL == "" {
		base.ErrResponse(c, base.CodeBadRequest, "url 参数不能为空")
		return
	}

	// 验证是否为微博图片URL
	if !strings.Contains(imageURL, "sinaimg.cn") {
		base.ErrResponse(c, base.CodeBadRequest, "仅支持微博图片URL")
		return
	}

	// 创建HTTP请求
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		base.ErrResponse(c, base.CodeServerError, "创建请求失败: "+err.Error())
		return
	}

	// 设置请求头，绕过防盗链
	req.Header.Set("Referer", "https://weibo.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		base.ErrResponse(c, base.CodeServerError, "请求图片失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		base.ErrResponse(c, base.CodeServerError, "图片服务器返回错误: "+resp.Status)
		return
	}

	// 获取文件名
	filename := path.Base(imageURL)

	// 设置响应头
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)

	// 流式传输图片内容
	_, _ = io.Copy(c.Writer, resp.Body)
}
