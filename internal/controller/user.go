package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/service"
	"github.com/yuanhuaxi/weibo-spider/internal/webserver/base"
)

// UserController 用户控制器
type UserController struct {
	userSvc *service.UserService
}

// AddUserRoute 注册用户相关路由
func AddUserRoute(r *gin.RouterGroup, userSvc *service.UserService) {
	ctrl := &UserController{userSvc: userSvc}

	r.GET("/users/me", base.RecoverWrap(ctrl.me))
	r.POST("/users", base.RecoverWrap(ctrl.create))
	r.GET("/users", base.RecoverWrap(ctrl.list))
	r.GET("/users/:id", base.RecoverWrap(ctrl.get))
	r.PUT("/users/:id", base.RecoverWrap(ctrl.update))
	r.DELETE("/users/:id", base.RecoverWrap(ctrl.delete))
}

// me 获取当前用户信息
func (ctrl *UserController) me(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		base.ErrResponse(c, 401, "未登录")
		return
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	userID, err := service.ParseToken(token)
	if err != nil {
		base.ErrResponse(c, 401, "登录已过期")
		return
	}

	result, err := ctrl.userSvc.GetByID(userID)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}

// create 创建用户
func (ctrl *UserController) create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "参数错误: "+err.Error())
		return
	}

	result, err := ctrl.userSvc.Create(&req)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}

// delete 删除用户
func (ctrl *UserController) delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "无效的用户ID")
		return
	}

	if err := ctrl.userSvc.Delete(uint(id)); err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, gin.H{"message": "删除成功"})
}
func (ctrl *UserController) update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "无效的用户ID")
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "参数错误: "+err.Error())
		return
	}

	result, err := ctrl.userSvc.Update(uint(id), &req)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}
func (ctrl *UserController) get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "无效的用户ID")
		return
	}

	result, err := ctrl.userSvc.GetByID(uint(id))
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}
func (ctrl *UserController) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	result, err := ctrl.userSvc.List(page, size)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}
