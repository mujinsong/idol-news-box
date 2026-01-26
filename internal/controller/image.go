package controller

import (
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yuanhuaxi/weibo-spider/internal/webserver/base"
)

// AddImageRoute 注册图片代理路由
func AddImageRoute(r *gin.RouterGroup) {
	r.GET("/image", base.RecoverWrap(downloadImage))
}

// downloadImage 下载微博图片（带Referer绕过防盗链）
func downloadImage(c *gin.Context) {
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
