package dto

import (
	"fmt"
	"time"
)

// Weibo 微博信息
type Weibo struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Content          string    `json:"content"`
	ArticleURL       string    `json:"article_url"`
	OriginalPictures []string  `json:"original_pictures"`
	VideoURL         string    `json:"video_url"`
	PublishPlace     string    `json:"publish_place"`
	PublishTime      time.Time `json:"publish_time"`
	PublishTool      string    `json:"publish_tool"`
	UpNum            int       `json:"up_num"`
	RetweetNum       int       `json:"retweet_num"`
	CommentNum       int       `json:"comment_num"`
	IsOriginal       bool      `json:"is_original"`
}

// CSVHeader 返回CSV表头
func (w *Weibo) CSVHeader() []string {
	return []string{
		"id", "正文", "发布位置", "发布时间",
		"发布工具", "点赞数", "转发数", "评论数", "原创",
	}
}

// CSVRow 返回CSV行数据
func (w *Weibo) CSVRow() []string {
	isOriginal := "否"
	if w.IsOriginal {
		isOriginal = "是"
	}
	return []string{
		w.ID, w.Content, w.PublishPlace,
		w.PublishTime.Format("2006-01-02 15:04"),
		w.PublishTool,
		fmt.Sprintf("%d", w.UpNum),
		fmt.Sprintf("%d", w.RetweetNum),
		fmt.Sprintf("%d", w.CommentNum),
		isOriginal,
	}
}

// WeiboList 微博列表响应
type WeiboList struct {
	Weibos []*Weibo `json:"weibos"`
	Total  int      `json:"total"`
}
