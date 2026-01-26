package spider

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/parser"
	"github.com/yuanhuaxi/weibo-spider/internal/writer"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
)

const baseURL = "https://weibo.cn"

// CrawlResult 爬取结果
type CrawlResult struct {
	User   *dto.User
	Weibos []*dto.Weibo
}

// Spider 微博爬虫
type Spider struct {
	cfg        *config.Config
	limiter    *Limiter
	writers    []writer.Writer
	infoParser *parser.InfoParser
	pageParser *parser.PageParser
}

// New 创建爬虫实例
func New(cfg *config.Config) *Spider {
	return &Spider{
		cfg:        cfg,
		infoParser: parser.NewInfoParser(),
		pageParser: parser.NewPageParser(),
		limiter:    NewLimiter(cfg.RandomWaitPages, cfg.RandomWaitSeconds),
	}
}

// newCollector 创建新的 Collector（每次请求都需要新建，避免回调累积）
func (s *Spider) newCollector() *colly.Collector {
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		logger.Info.Printf("请求URL: %s", r.URL.String())
		r.Headers.Set("Cookie", s.cfg.Cookie)
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	})
	return c
}

// initWriters 初始化写入器
func (s *Spider) initWriters(outputDir string) error {
	for _, mode := range s.cfg.WriteMode {
		var w writer.Writer
		var err error
		switch strings.ToLower(mode) {
		case "csv":
			w, err = writer.NewCSV(outputDir)
		case "txt":
			w, err = writer.NewTXT(outputDir)
		case "json":
			w, err = writer.NewJSON(outputDir)
		}
		if err != nil {
			return err
		}
		if w != nil {
			s.writers = append(s.writers, w)
		}
	}
	return nil
}

// Run 运行爬虫任务
func (s *Spider) Run(task *dto.CrawlTask) (*CrawlResult, error) {
	return s.crawlUser(task)
}

// crawlUser 抓取单个用户（同时获取用户信息和微博）
func (s *Spider) crawlUser(task *dto.CrawlTask) (*CrawlResult, error) {
	result := &CrawlResult{
		Weibos: make([]*dto.Weibo, 0),
	}

	user, err := s.FetchUserInfo(task.UserID)
	if err != nil {
		return nil, err
	}
	result.User = user

	weibos, err := s.FetchWeibos(task)
	if err != nil {
		return nil, err
	}
	result.Weibos = weibos

	return result, nil
}

// FetchUserInfo 获取用户信息（公开方法）
func (s *Spider) FetchUserInfo(userID string) (*dto.User, error) {
	logger.Info.Printf("获取用户信息: %s", userID)
	var user *dto.User
	url := fmt.Sprintf("%s/%s/info", baseURL, userID)

	c := s.newCollector()
	c.OnHTML("body", func(e *colly.HTMLElement) {
		html, _ := e.DOM.Html()
		logger.Info.Printf("爬取到的HTML内容: %s", html)
		user = s.infoParser.Parse(e.DOM, userID)
	})

	if err := c.Visit(url); err != nil {
		return nil, err
	}
	s.limiter.Wait()
	return user, nil
}

// FetchWeibos 抓取微博列表（公开方法）
func (s *Spider) FetchWeibos(task *dto.CrawlTask) ([]*dto.Weibo, error) {
	logger.Info.Printf("获取用户微博: %s", task.UserID)

	// 初始化写入器
	outputDir := fmt.Sprintf("%s/%s", s.cfg.OutputDir, task.UserID)
	if err := s.initWriters(outputDir); err != nil {
		return nil, err
	}
	defer s.closeWriters()

	var result []*dto.Weibo
	page := 1

	for {
		url := fmt.Sprintf("%s/%s?page=%d", baseURL, task.UserID, page)
		hasMore := false

		c := s.newCollector()
		c.OnHTML("body", func(e *colly.HTMLElement) {
			weibos := s.pageParser.Parse(e.DOM, task.UserID)
			for _, w := range weibos {
				// 时间过滤
				if w.PublishTime.Before(task.SinceDate) {
					continue
				}
				if w.PublishTime.After(task.EndDate) {
					continue
				}
				// 原创过滤
				if task.Filter == 1 && !w.IsOriginal {
					continue
				}
				s.writeWeibo(w)
				result = append(result, w)
				hasMore = true
			}
		})

		if err := c.Visit(url); err != nil {
			return result, err
		}
		s.limiter.Wait()

		if !hasMore {
			break
		}
		page++
	}
	return result, nil
}

// writeUser 写入用户信息
func (s *Spider) writeUser(user *dto.User) {
	for _, w := range s.writers {
		w.WriteUser(user)
	}
}

// writeWeibo 写入微博信息
func (s *Spider) writeWeibo(weibo *dto.Weibo) {
	for _, w := range s.writers {
		w.WriteWeibo(weibo)
	}
}

// closeWriters 关闭所有写入器
func (s *Spider) closeWriters() {
	for _, w := range s.writers {
		w.Close()
	}
	s.writers = nil
}
