package service

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

const (
	maxDownloadRetry = 3
	baseRetryDelay   = 2 * time.Second
	imageTimeout     = 60 * time.Second
	videoTimeout     = 5 * time.Minute
)

// MediaDownloader 媒体下载器
type MediaDownloader struct {
	cfg         *config.Config
	imageClient *http.Client
	videoClient *http.Client
}

// NewMediaDownloader 创建媒体下载器
func NewMediaDownloader(cfg *config.Config) *MediaDownloader {
	return &MediaDownloader{
		cfg: cfg,
		imageClient: &http.Client{
			Timeout: imageTimeout,
		},
		videoClient: &http.Client{
			Timeout: videoTimeout,
		},
	}
}

// DownloadImage 下载图片
func (d *MediaDownloader) DownloadImage(userID, weiboID, imageURL string) error {
	// 创建保存目录
	saveDir := filepath.Join(d.cfg.OutputDir, userID, "images", weiboID)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 获取文件名
	filename := filepath.Base(imageURL)
	// 处理URL中的查询参数
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}
	savePath := filepath.Join(saveDir, filename)

	// 如果文件已存在，跳过
	if _, err := os.Stat(savePath); err == nil {
		logger.Info.Printf("图片已存在，跳过: %s", savePath)
		return nil
	}

	// 下载图片
	return d.download(imageURL, savePath, "https://weibo.cn/", false)
}

// DownloadVideo 下载视频
func (d *MediaDownloader) DownloadVideo(userID, weiboID, videoURL string) error {
	// 创建保存目录
	saveDir := filepath.Join(d.cfg.OutputDir, userID, "videos", weiboID)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 视频URL可能是页面链接，需要特殊处理
	// 如果是 m.weibo.cn/s/video/show 格式，需要解析获取真实视频URL
	if strings.Contains(videoURL, "video/show") {
		realURL, err := d.parseVideoURL(videoURL)
		if err != nil {
			return fmt.Errorf("解析视频URL失败: %w", err)
		}
		videoURL = realURL
	}

	// 获取文件名
	filename := fmt.Sprintf("%s.mp4", weiboID)
	savePath := filepath.Join(saveDir, filename)

	// 如果文件已存在，跳过
	if _, err := os.Stat(savePath); err == nil {
		logger.Info.Printf("视频已存在，跳过: %s", savePath)
		return nil
	}

	// 下载视频
	return d.download(videoURL, savePath, "https://m.weibo.cn/", true)
}

// parseVideoURL 解析视频页面获取真实视频URL
func (d *MediaDownloader) parseVideoURL(pageURL string) (string, error) {
	// 将 video/show 替换为 video/object 获取视频信息API
	videoObjectURL := strings.Replace(pageURL, "m.weibo.cn/s/video/show", "m.weibo.cn/s/video/object", 1)

	logger.Info.Printf("请求视频API: %s", videoObjectURL)

	req, err := http.NewRequest("GET", videoObjectURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36")
	req.Header.Set("Referer", "https://m.weibo.cn/")
	req.Header.Set("Cookie", d.cfg.Cookie)

	resp, err := d.imageClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	content := string(body)

	// 尝试提取 hd_url 或 url
	// 格式: {"data":{"object":{"stream":{"hd_url":"...","url":"..."}}}}
	for _, key := range []string{"hd_url", "url"} {
		pattern := `"` + key + `":"([^"]+)"`
		if idx := strings.Index(content, `"`+key+`"`); idx != -1 {
			start := strings.Index(content[idx:], "http")
			if start != -1 {
				end := strings.Index(content[idx+start:], `"`)
				if end != -1 {
					url := content[idx+start : idx+start+end]
					url = strings.ReplaceAll(url, `\/`, `/`)
					logger.Info.Printf("找到视频URL (%s): %s", key, url)
					return url, nil
				}
			}
		}
		_ = pattern
	}

	// 调试日志
	if len(content) > 500 {
		logger.Info.Printf("视频API返回(前500字符): %s", content[:500])
	} else {
		logger.Info.Printf("视频API返回: %s", content)
	}

	return "", fmt.Errorf("无法从API响应中提取视频URL")
}

// download 通用下载方法（带重试）
func (d *MediaDownloader) download(url, savePath, referer string, isVideo bool) error {
	var lastErr error
	for i := 0; i < maxDownloadRetry; i++ {
		if i > 0 {
			logger.Info.Printf("重试下载 (%d/%d): %s", i+1, maxDownloadRetry, url)
			time.Sleep(baseRetryDelay)
		}

		lastErr = d.doDownload(url, savePath, referer, isVideo)
		if lastErr == nil {
			return nil
		}
		logger.Warn.Printf("下载失败 (%d/%d): %v", i+1, maxDownloadRetry, lastErr)
	}
	return fmt.Errorf("下载失败，已重试%d次: %w", maxDownloadRetry, lastErr)
}

// doDownload 执行单次下载
func (d *MediaDownloader) doDownload(url, savePath, referer string, isVideo bool) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148")
	req.Header.Set("Referer", referer)

	// 根据类型选择客户端
	client := d.imageClient
	if isVideo {
		client = d.videoClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误: %s", resp.Status)
	}

	// 创建文件
	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 写入文件
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(savePath) // 删除不完整的文件
		return fmt.Errorf("写入文件失败: %w", err)
	}

	logger.Info.Printf("下载完成: %s (%d bytes)", savePath, written)
	return nil
}
