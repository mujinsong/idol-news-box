package webserver

import (
	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/controller"
	"github.com/yuanhuaxi/weibo-spider/internal/service"
)

// Server Web服务器
type Server struct {
	engine *gin.Engine
}

// NewServer 创建Web服务器
func NewServer(spider *service.SpiderService, userSvc *service.UserService, authSvc *service.AuthService, mode string) *Server {
	gin.SetMode(mode)
	engine := gin.Default()

	s := &Server{
		engine: engine,
	}
	s.setupRoutes(spider, userSvc, authSvc)
	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes(spider *service.SpiderService, userSvc *service.UserService, authSvc *service.AuthService) {
	api := s.engine.Group("/api/v1")

	// 注册各业务路由
	controller.AddHealthRoute(api)
	controller.AddAuthRoute(api, authSvc)
	controller.AddSpiderRoute(api, spider)
	controller.AddTaskRoute(api, spider)
	controller.AddImageRoute(api)
	controller.AddUserRoute(api, userSvc)
}

// Run 启动服务
func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}
