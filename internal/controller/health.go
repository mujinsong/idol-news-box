package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/webserver/base"
)

// AddHealthRoute 注册健康检查路由
func AddHealthRoute(r *gin.RouterGroup) {
	r.GET("/health", base.RecoverWrap(health))
}

// health 健康检查
func health(c *gin.Context) {
	base.OkResponse(c, gin.H{"status": "ok"})
}
