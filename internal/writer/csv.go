package writer

import (
	"encoding/csv"
	"os"
	"path/filepath"

	"github.com/yuanhuaxi/weibo-spider/internal/dto"
)

// CSV CSV文件写入器
type CSV struct {
	outputDir   string
	userFile    *os.File
	weiboFile   *os.File
	userWriter  *csv.Writer
	weiboWriter *csv.Writer
}

// NewCSV 创建CSV写入器
func NewCSV(outputDir string) (*CSV, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}
	return &CSV{outputDir: outputDir}, nil
}

// WriteUser 写入用户信息
func (w *CSV) WriteUser(user *dto.User) error {
	if w.userFile == nil {
		f, err := os.Create(filepath.Join(w.outputDir, "users.csv"))
		if err != nil {
			return err
		}
		w.userFile = f
		w.userWriter = csv.NewWriter(f)
		w.userWriter.Write(user.CSVHeader())
	}
	return w.userWriter.Write(user.CSVRow())
}

// WriteWeibo 写入微博信息
func (w *CSV) WriteWeibo(weibo *dto.Weibo) error {
	if w.weiboFile == nil {
		f, err := os.Create(filepath.Join(w.outputDir, "weibos.csv"))
		if err != nil {
			return err
		}
		w.weiboFile = f
		w.weiboWriter = csv.NewWriter(f)
		w.weiboWriter.Write(weibo.CSVHeader())
	}
	return w.weiboWriter.Write(weibo.CSVRow())
}

// Close 关闭文件
func (w *CSV) Close() error {
	if w.userWriter != nil {
		w.userWriter.Flush()
	}
	if w.weiboWriter != nil {
		w.weiboWriter.Flush()
	}
	if w.userFile != nil {
		w.userFile.Close()
	}
	if w.weiboFile != nil {
		w.weiboFile.Close()
	}
	return nil
}
