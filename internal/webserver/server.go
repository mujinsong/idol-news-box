package webserver

import (
	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/controller"
	
)

// Server Web服务器
type Server struct {
	engine *gin.Engine
}

// NewServer 创建Web服务器
func NewServer(spider *service.SpiderService, mode string) *Server {
	gin.SetMode(mode)
	engine := gin.Default()

	s := &Server{
		engine: engine,
	}
	s.setupRoutes(spider)
	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes(spider *service.SpiderService) {
	api := s.engine.Group("/api/v1")

	// 注册各业务路由
	controller.AddHealthRoute(api)
	controller.AddSpiderRoute(api, spider)
	controller.AddTaskRoute(api, spider)
	controller.AddImageRoute(api)
}

// Run 启动服务
func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}
