package base

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

// RecoverWrap 异常处理包装器
func RecoverWrap(f func(c *gin.Context)) func(c *gin.Context) {
	return func(c *gin.Context) {
		defer func() {
			if e := recover(); e != nil {
				logger.Error.Printf("panic: %v\n%s", e, debug.Stack())
				BadResponse(c, e)
			}
		}()
		f(c)
	}
}
