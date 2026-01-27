package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/service"
	"github.com/yuanhuaxi/weibo-spider/internal/webserver/base"
)

// AuthController 认证控制器
type AuthController struct {
	authSvc *service.AuthService
}

// AddAuthRoute 注册认证路由
func AddAuthRoute(r *gin.RouterGroup, authSvc *service.AuthService) {
	ctrl := &AuthController{authSvc: authSvc}
	r.POST("/auth/login", base.RecoverWrap(ctrl.login))
}

// login 登录
func (ctrl *AuthController) login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.ErrResponse(c, base.CodeBadRequest, "参数错误")
		return
	}

	result, err := ctrl.authSvc.Login(&req)
	if err != nil {
		base.BadResponse(c, err)
		return
	}

	base.OkResponse(c, result)
}
