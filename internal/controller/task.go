package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/service"
	"github.com/yuanhuaxi/weibo-spider/internal/webserver/base"
)

// TaskController 任务控制器
type TaskController struct {
	spider *service.SpiderService
}

// AddTaskRoute 注册任务相关路由
func AddTaskRoute(r *gin.RouterGroup, spider *service.SpiderService) {
	ctrl := &TaskController{spider: spider}

	r.GET("/task/:task_id", base.RecoverWrap(ctrl.getTaskStatus))
}

// getTaskStatus 获取任务状态
func (ctrl *TaskController) getTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		base.ErrResponse(c, base.CodeBadRequest, "task_id 不能为空")
		return
	}

	result, err := ctrl.spider.GetTaskStatus(taskID)
	if err != nil {
		base.ErrResponse(c, base.CodeNotFound, err.Error())
		return
	}

	base.OkResponse(c, result)
}
