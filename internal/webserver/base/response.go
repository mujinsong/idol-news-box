package base

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// Response 统一响应结构
type Response struct {
	Code    Code        `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OkResponse 成功响应
func OkResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: MsgSuccess,
		Data:    data,
	})
}

// ErrResponse 错误响应
func ErrResponse(c *gin.Context, code Code, message string) {
	c.AbortWithStatusJSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// BadResponse 错误响应（从 interface{} 获取消息）
func BadResponse(c *gin.Context, data interface{}) {
	var msg string
	switch v := data.(type) {
	case error:
		msg = v.Error()
		logger.Error.Printf("API Error: %v", v)
	case string:
		msg = v
	default:
		msg = MsgBadRequest
	}
	if len(msg) > 256 {
		msg = msg[:256]
	}
	ErrResponse(c, CodeBadRequest, msg)
}
