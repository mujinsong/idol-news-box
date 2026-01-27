package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
)

// PageParser 微博页面解析器
type PageParser struct{}

// NewPageParser 创建微博页面解析器
func NewPageParser() *PageParser {
	return &PageParser{}
}

// Parse 解析微博列表页面
func (p *PageParser) Parse(doc *goquery.Selection, userID string) []*dto.Weibo {
	var weibos []*dto.Weibo

	doc.Find(".c").Each(func(i int, s *goquery.Selection) {
		id, exists := s.Attr("id")
		if !exists || !strings.HasPrefix(id, "M_") {
			return
		}
		weibo := p.parseWeibo(s, userID, id[2:])
		if weibo != nil {
			weibos = append(weibos, weibo)
		}
	})

	return weibos
}

// parseWeibo 解析单条微博
func (p *PageParser) parseWeibo(s *goquery.Selection, userID, weiboID string) *dto.Weibo {
	weibo := &dto.Weibo{
		ID:     weiboID,
		UserID: userID,
	}
	weibo.Content = strings.TrimSpace(s.Find(".ctt").Text())
	weibo.IsOriginal = !strings.Contains(s.Text(), "转发理由")
	weibo.PublishTime = p.parseTime(s)
	p.parseStats(s, weibo)
	p.parseMedia(s, weibo)
	return weibo
}

// parseTime 解析发布时间
func (p *PageParser) parseTime(s *goquery.Selection) time.Time {
	text := s.Find(".ct").Text()
	now := time.Now()

	// 匹配 "今天 HH:MM"
	reToday := regexp.MustCompile(`今天\s*(\d{1,2}):(\d{2})`)
	if m := reToday.FindStringSubmatch(text); len(m) == 3 {
		hour, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		return time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, time.Local)
	}

	// 匹配 "X分钟前"
	reMinAgo := regexp.MustCompile(`(\d+)分钟前`)
	if m := reMinAgo.FindStringSubmatch(text); len(m) == 2 {
		mins, _ := strconv.Atoi(m[1])
		return now.Add(-time.Duration(mins) * time.Minute)
	}

	// 匹配 "月日 HH:MM"
	re := regexp.MustCompile(`(\d{1,2})月(\d{1,2})日\s*(\d{1,2}):(\d{2})`)
	if m := re.FindStringSubmatch(text); len(m) == 5 {
		month, _ := strconv.Atoi(m[1])
		day, _ := strconv.Atoi(m[2])
		hour, _ := strconv.Atoi(m[3])
		min, _ := strconv.Atoi(m[4])
		return time.Date(now.Year(), time.Month(month), day, hour, min, 0, 0, time.Local)
	}

	// 匹配 "年-月-日 HH:MM:SS"
	reFullDate := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})\s*(\d{2}):(\d{2}):(\d{2})`)
	if m := reFullDate.FindStringSubmatch(text); len(m) == 7 {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		hour, _ := strconv.Atoi(m[4])
		min, _ := strconv.Atoi(m[5])
		sec, _ := strconv.Atoi(m[6])
		return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
	}

	return now
}

// parseStats 解析互动数据
func (p *PageParser) parseStats(s *goquery.Selection, weibo *dto.Weibo) {
	text := s.Text()

	reUp := regexp.MustCompile(`赞\[(\d+)\]`)
	if m := reUp.FindStringSubmatch(text); len(m) > 1 {
		weibo.UpNum, _ = strconv.Atoi(m[1])
	}

	reRetweet := regexp.MustCompile(`转发\[(\d+)\]`)
	if m := reRetweet.FindStringSubmatch(text); len(m) > 1 {
		weibo.RetweetNum, _ = strconv.Atoi(m[1])
	}

	reComment := regexp.MustCompile(`评论\[(\d+)\]`)
	if m := reComment.FindStringSubmatch(text); len(m) > 1 {
		weibo.CommentNum, _ = strconv.Atoi(m[1])
	}
}

// parseMedia 解析图片和视频
func (p *PageParser) parseMedia(s *goquery.Selection, weibo *dto.Weibo) {
	// 解析链接中的组图和视频
	s.Find("a").Each(func(i int, a *goquery.Selection) {
		href, exists := a.Attr("href")
		if !exists {
			return
		}
		text := a.Text()

		// 组图链接（保存供参考，但不是直接图片URL）
		if strings.Contains(href, "/mblog/picAll/") || strings.Contains(text, "组图") {
			weibo.ArticleURL = href // 组图页面链接
		}
		// 视频链接
		if strings.Contains(href, "video") || strings.Contains(text, "视频") {
			weibo.VideoURL = href
		}
	})

	// 解析图片 img 标签，提取微博配图（排除表情和图标）
	s.Find("img").Each(func(i int, img *goquery.Selection) {
		src, exists := img.Attr("src")
		if !exists {
			return
		}
		// 只提取微博配图（sinaimg.cn 的 wap180 缩略图或大图）
		// 排除表情图片（emoticon）和认证图标（upload/2016）
		if strings.Contains(src, "sinaimg.cn") &&
			!strings.Contains(src, "emoticon") &&
			!strings.Contains(src, "/upload/") &&
			!strings.Contains(src, "expression") {
			// 将缩略图URL转换为原图URL
			originalURL := p.convertToOriginalPic(src)
			weibo.OriginalPictures = append(weibo.OriginalPictures, originalURL)
		}
	})
}

// convertToOriginalPic 将缩略图URL转换为原图URL
func (p *PageParser) convertToOriginalPic(thumbURL string) string {
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
		if strings.Contains(thumbURL, format) {
			return strings.Replace(thumbURL, format, "/large/", 1)
		}
	}
	return thumbURL
}
