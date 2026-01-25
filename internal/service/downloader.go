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

// MediaDownloader 媒体下载器
type MediaDownloader struct {
	cfg    *config.Config
	client *http.Client
}

// NewMediaDownloader 创建媒体下载器
func NewMediaDownloader(cfg *config.Config) *MediaDownloader {
	return &MediaDownloader{
		cfg: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
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
	return d.download(imageURL, savePath, "https://weibo.cn/")
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
	return d.download(videoURL, savePath, "https://m.weibo.cn/")
}

// parseVideoURL 解析视频页面获取真实视频URL
func (d *MediaDownloader) parseVideoURL(pageURL string) (string, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148")
	req.Header.Set("Referer", "https://m.weibo.cn/")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 尝试从页面中提取视频URL
	// 通常在 stream_url 或 video_src 字段中
	content := string(body)

	// 查找 stream_url
	patterns := []string{
		`"stream_url":"([^"]+)"`,
		`"video_src":"([^"]+)"`,
		`"url":"(https?://[^"]*\.mp4[^"]*)"`,
	}

	for _, pattern := range patterns {
		if idx := strings.Index(content, "stream_url"); idx != -1 {
			// 简单提取，实际可能需要更复杂的解析
			start := strings.Index(content[idx:], "http")
			if start != -1 {
				end := strings.Index(content[idx+start:], `"`)
				if end != -1 {
					url := content[idx+start : idx+start+end]
					url = strings.ReplaceAll(url, `\/`, `/`)
					return url, nil
				}
			}
		}
		_ = pattern // 避免未使用警告
	}

	return "", fmt.Errorf("无法从页面中提取视频URL")
}

// download 通用下载方法
func (d *MediaDownloader) download(url, savePath, referer string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148")
	req.Header.Set("Referer", referer)

	resp, err := d.client.Do(req)
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
