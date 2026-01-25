package writer

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/yuanhuaxi/weibo-spider/internal/dto"
)

// JSON JSON文件写入器
type JSON struct {
	outputDir string
	users     []*dto.User
	weibos    []*dto.Weibo
}

// NewJSON 创建JSON写入器
func NewJSON(outputDir string) (*JSON, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}
	return &JSON{outputDir: outputDir}, nil
}

// WriteUser 写入用户信息
func (w *JSON) WriteUser(user *dto.User) error {
	w.users = append(w.users, user)
	return nil
}

// WriteWeibo 写入微博信息
func (w *JSON) WriteWeibo(weibo *dto.Weibo) error {
	w.weibos = append(w.weibos, weibo)
	return nil
}

// Close 保存并关闭
func (w *JSON) Close() error {
	if len(w.users) > 0 {
		data, _ := json.MarshalIndent(w.users, "", "  ")
		os.WriteFile(filepath.Join(w.outputDir, "users.json"), data, 0644)
	}
	if len(w.weibos) > 0 {
		data, _ := json.MarshalIndent(w.weibos, "", "  ")
		os.WriteFile(filepath.Join(w.outputDir, "weibos.json"), data, 0644)
	}
	return nil
}
