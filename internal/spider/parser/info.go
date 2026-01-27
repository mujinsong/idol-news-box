package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
)

// InfoParser 用户信息解析器
type InfoParser struct{}

// NewInfoParser 创建用户信息解析器
func NewInfoParser() *InfoParser {
	return &InfoParser{}
}

// Parse 解析用户信息页面
func (p *InfoParser) Parse(doc *goquery.Selection, userID string) *dto.User {
	user := &dto.User{ID: userID}

	// 解析基本信息
	doc.Find(".c").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "昵称:") {
			user.Nickname = p.extractField(text, "昵称:")
		}
		if strings.Contains(text, "性别:") {
			user.Gender = p.extractField(text, "性别:")
		}
		if strings.Contains(text, "地区:") {
			user.Location = p.extractField(text, "地区:")
		}
		if strings.Contains(text, "生日:") {
			user.Birthday = p.extractField(text, "生日:")
		}
		if strings.Contains(text, "简介:") {
			user.Description = p.extractField(text, "简介:")
		}
		if strings.Contains(text, "认证:") {
			user.VerifiedReason = p.extractField(text, "认证:")
		}
	})

	// 解析统计数据
	doc.Find(".tip2").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "微博") {
			user.WeiboNum = p.extractNumber(text, "微博")
		}
		if strings.Contains(text, "关注") {
			user.Following = p.extractNumber(text, "关注")
		}
		if strings.Contains(text, "粉丝") {
			user.Followers = p.extractNumber(text, "粉丝")
		}
	})

	return user
}

// extractNumber 从文本中提取数字
func (p *InfoParser) extractNumber(text, prefix string) int {
	re := regexp.MustCompile(prefix + `\[(\d+)\]`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n
	}
	return 0
}

// extractField 从文本中提取字段值
func (p *InfoParser) extractField(text, prefix string) string {
	idx := strings.Index(text, prefix)
	if idx == -1 {
		return ""
	}
	start := idx + len(prefix)
	rest := text[start:]
	// 找到下一个字段（以中文冒号结尾的词）或结束
	re := regexp.MustCompile(`^([^:：]+)`)
	matches := re.FindStringSubmatch(rest)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return strings.TrimSpace(rest)
}
