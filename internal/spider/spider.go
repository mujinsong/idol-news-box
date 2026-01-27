package spider

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/spider/parser"
	"github.com/yuanhuaxi/weibo-spider/internal/writer"
	"github.com/yuanhuaxi/weibo-spider/pkg/logger"
	"github.com/yuanhuaxi/weibo-spider/pkg/proxy"
	"strings"
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
	proxyPool  *proxy.Pool
}

// New 创建爬虫实例
func New(cfg *config.Config) *Spider {
	s := &Spider{
		cfg:        cfg,
		infoParser: parser.NewInfoParser(),
		pageParser: parser.NewPageParser(),
		limiter:    NewLimiter(cfg.RandomWaitPages, cfg.RandomWaitSeconds),
	}
	// 初始化代理池
	if len(cfg.Proxies) > 0 {
		s.proxyPool = proxy.NewPool(cfg.Proxies)
		logger.Info.Printf("代理池已初始化，共 %d 个代理", s.proxyPool.Size())
	}
	return s
}

// newCollector 创建新的 Collector（每次请求都需要新建，避免回调累积）
func (s *Spider) newCollector() *colly.Collector {
	c := colly.NewCollector()

	// 设置代理
	if s.proxyPool != nil && !s.proxyPool.IsEmpty() {
		proxyURL := s.proxyPool.Get()
		if proxyURL != "" {
			c.SetProxy(proxyURL)
			logger.Info.Printf("使用代理: %s", proxyURL)
		}
	}

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
				// 如果有组图链接，获取全部图片
				if w.ArticleURL != "" && strings.Contains(w.ArticleURL, "/mblog/picAll/") {
					allPics := s.fetchAllPictures(w.ArticleURL)
					if len(allPics) > 0 {
						w.OriginalPictures = allPics
					}
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

// fetchAllPictures 获取组图页面的所有图片
func (s *Spider) fetchAllPictures(articleURL string) []string {
	var pictures []string

	// 确保URL是完整的
	if !strings.HasPrefix(articleURL, "http") {
		articleURL = baseURL + articleURL
	}

	c := s.newCollector()
	c.OnHTML("body", func(e *colly.HTMLElement) {
		e.DOM.Find("img").Each(func(i int, img *goquery.Selection) {
			src, exists := img.Attr("src")
			if !exists {
				return
			}
			// 只提取微博配图
			if strings.Contains(src, "sinaimg.cn") &&
				!strings.Contains(src, "emoticon") &&
				!strings.Contains(src, "/upload/") &&
				!strings.Contains(src, "expression") {
				// 转换为大图URL
				src = convertToLargeImage(src)
				pictures = append(pictures, src)
			}
		})
	})

	if err := c.Visit(articleURL); err != nil {
		logger.Error.Printf("获取组图失败: %s, %v", articleURL, err)
		return pictures
	}
	s.limiter.Wait()

	return pictures
}

// convertToLargeImage 将缩略图URL转换为大图URL
func convertToLargeImage(src string) string {
	// 微博图片常见的缩略图格式
	thumbFormats := []string{
		"/thumb180/",
		"/thumb300/",
		"/wap180/",
		"/wap360/",
		"/orj360/",
		"/mw690/",
		"/mw1024/",
		"/small/",
		"/square/",
		"/thumbnail/",
	}
	for _, format := range thumbFormats {
		if strings.Contains(src, format) {
			return strings.Replace(src, format, "/large/", 1)
		}
	}
	return src
}

// FetchSpecialFollows 获取当前登录用户的特别关注列表
func (s *Spider) FetchSpecialFollows() (*dto.SpecialFollowList, error) {
	logger.Info.Println("获取特别关注列表")

	result := &dto.SpecialFollowList{
		Users: make([]dto.SpecialFollowUser, 0),
	}

	// 特别关注页面 URL
	url := baseURL + "/attgroup/opening?gid=4286498122756239"

	c := s.newCollector()
	c.OnHTML("body", func(e *colly.HTMLElement) {
		e.DOM.Find("table").Each(func(i int, table *goquery.Selection) {
			// 查找用户链接
			table.Find("a[href]").Each(func(j int, a *goquery.Selection) {
				href, _ := a.Attr("href")
				// 匹配用户主页链接格式: /u/1234567890 或 /1234567890
				if strings.HasPrefix(href, "/u/") || (strings.HasPrefix(href, "/") && len(href) > 1) {
					userID := strings.TrimPrefix(href, "/u/")
					userID = strings.TrimPrefix(userID, "/")
					// 过滤非数字ID
					if isNumeric(userID) {
						nickname := strings.TrimSpace(a.Text())
						if nickname != "" && nickname != "关注" && nickname != "取消" {
							result.Users = append(result.Users, dto.SpecialFollowUser{
								ID:       userID,
								Nickname: nickname,
							})
						}
					}
				}
			})
		})
	})

	if err := c.Visit(url); err != nil {
		return nil, fmt.Errorf("获取特别关注列表失败: %w", err)
	}
	s.limiter.Wait()

	result.Total = len(result.Users)
	logger.Info.Printf("获取到 %d 个特别关注用户", result.Total)
	return result, nil
}

// isNumeric 检查字符串是否为纯数字
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
