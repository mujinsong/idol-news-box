package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/service"
	"github.com/yuanhuaxi/weibo-spider/internal/webserver/base"
)

// SpiderController 爬虫控制器
type SpiderController struct {
	spider *service.SpiderService
}

// AddSpiderRoute 注册爬虫相关路由
func AddSpiderRoute(r *gin.RouterGroup, spider *service.SpiderService) {
	ctrl := &SpiderController{spider: spider}

	r.GET("/status", base.RecoverWrap(ctrl.status))
	r.GET("/user/:user_id", base.RecoverWrap(ctrl.getUserInfo))
	r.POST("/weibos", base.RecoverWrap(ctrl.getWeibos))
	r.POST("/crawl", base.RecoverWrap(ctrl.crawl))
}

// status 获取爬虫状态
func (ctrl *SpiderController) status(c *gin.Context) {
	base.OkResponse(c, gin.H{"running": ctrl.spider.IsRunning()})
}

// getUserInfo 获取用户信息
func (ctrl *SpiderController) getUserInfo(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		base.ErrResponse(c, base.CodeBadRequest, "user_id 不能为空")
		return
	}

	result, err := ctrl.spider.GetUserInfo(userID)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}

// getWeibos 提交微博爬取任务（异步）
func (ctrl *SpiderController) getWeibos(c *gin.Context) {
	var req dto.CrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "参数错误: "+err.Error())
		return
	}

	task, err := req.ToTask()
	if err != nil {
		base.ErrResponse(c, base.CodeBadRequest, err.Error())
		return
	}

	result, err := ctrl.spider.SubmitTask(task)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}

// crawl 触发爬取任务（同步）
func (ctrl *SpiderController) crawl(c *gin.Context) {
	if ctrl.spider.IsRunning() {
		base.ErrResponse(c, base.CodeBusy, base.MsgBusy)
		return
	}

	var req dto.CrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "参数错误: "+err.Error())
		return
	}

	task, err := req.ToTask()
	if err != nil {
		base.ErrResponse(c, base.CodeBadRequest, err.Error())
		return
	}

	result, err := ctrl.spider.Run(task)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}
