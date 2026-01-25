package writer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuanhuaxi/weibo-spider/internal/dto"
)

// TXT TXT文件写入器
type TXT struct {
	outputDir string
	file      *os.File
}

// NewTXT 创建TXT写入器
func NewTXT(outputDir string) (*TXT, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}
	return &TXT{outputDir: outputDir}, nil
}

// WriteUser 写入用户信息
func (w *TXT) WriteUser(user *dto.User) error {
	f, err := os.OpenFile(
		filepath.Join(w.outputDir, "weibos.txt"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
	)
	if err != nil {
		return err
	}
	w.file = f
	fmt.Fprintf(f, "用户: %s (%s)\n", user.Nickname, user.ID)
	fmt.Fprintf(f, "粉丝: %d, 关注: %d, 微博: %d\n---\n",
		user.Followers, user.Following, user.WeiboNum)
	return nil
}

// WriteWeibo 写入微博信息
func (w *TXT) WriteWeibo(weibo *dto.Weibo) error {
	if w.file == nil {
		return fmt.Errorf("file not opened")
	}
	fmt.Fprintf(w.file, "[%s] %s\n",
		weibo.PublishTime.Format("2006-01-02 15:04"), weibo.Content)
	fmt.Fprintf(w.file, "赞:%d 转发:%d 评论:%d\n\n",
		weibo.UpNum, weibo.RetweetNum, weibo.CommentNum)
	return nil
}

// Close 关闭文件
func (w *TXT) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
