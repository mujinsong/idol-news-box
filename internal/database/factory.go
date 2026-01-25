package database

import (
	"fmt"
	"sync"

	"github.com/yuanhuaxi/weibo-spider/internal/config"
	"github.com/yuanhuaxi/weibo-spider/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DBFactory 数据库工厂
type DBFactory struct {
	db  *gorm.DB
	cfg *config.DatabaseConfig
	mu  sync.Once
}

var factory *DBFactory

// NewFactory 创建工厂实例
func NewFactory(cfg *config.DatabaseConfig) *DBFactory {
	factory = &DBFactory{cfg: cfg}
	return factory
}

// GetFactory 获取工厂实例
func GetFactory() *DBFactory {
	return factory
}

// Init 初始化数据库连接
func (f *DBFactory) Init() error {
	var initErr error
	f.mu.Do(func() {
		// 先连接到 postgres 数据库检查目标库是否存在
		if err := f.ensureDatabase(); err != nil {
			initErr = err
			return
		}

		// 连接到目标数据库
		db, err := gorm.Open(postgres.Open(f.cfg.DSN()), &gorm.Config{})
		if err != nil {
			initErr = fmt.Errorf("连接数据库失败: %w", err)
			return
		}
		f.db = db
	})
	return initErr
}

// GetDB 获取数据库连接
func (f *DBFactory) GetDB() *gorm.DB {
	return f.db
}

// ensureDatabase 确保数据库存在，不存在则创建
func (f *DBFactory) ensureDatabase() error {
	// 连接到 postgres 默认库
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		f.cfg.Host, f.cfg.Port, f.cfg.User, f.cfg.Password, f.cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接 postgres 失败: %w", err)
	}

	// 检查数据库是否存在
	var count int64
	db.Raw("SELECT COUNT(*) FROM pg_database WHERE datname = ?", f.cfg.DBName).Scan(&count)

	if count == 0 {
		// 创建数据库
		if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", f.cfg.DBName)).Error; err != nil {
			return fmt.Errorf("创建数据库失败: %w", err)
		}
	}

	// 关闭临时连接
	sqlDB, _ := db.DB()
	sqlDB.Close()

	return nil
}

// AutoMigrate 自动迁移表结构
func (f *DBFactory) AutoMigrate() error {
	return f.db.AutoMigrate(&model.ScheduledTask{})
}
