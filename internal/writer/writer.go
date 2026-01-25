package writer

import "github.com/yuanhuaxi/weibo-spider/internal/dto"

// Writer 数据写入接口
type Writer interface {
	WriteUser(user *dto.User) error
	WriteWeibo(weibo *dto.Weibo) error
	Close() error
}
