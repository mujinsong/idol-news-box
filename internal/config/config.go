package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ServerConfig HTTP服务配置
type ServerConfig struct {
	Port int    `json:"port"`
	Mode string `json:"mode"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// Config 爬虫配置
type Config struct {
	Server            ServerConfig   `json:"server"`
	Database          DatabaseConfig `json:"database"`
	RandomWaitPages   [2]int         `json:"random_wait_pages"`
	RandomWaitSeconds [2]int         `json:"random_wait_seconds"`
	WriteMode         []string       `json:"write_mode"`
	Cookie            string         `json:"cookie"`
	OutputDir         string         `json:"output_dir"`
}

// Default 返回默认配置
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
			Mode: "release",
		},
		Database: DatabaseConfig{
			Host:    "localhost",
			Port:    5432,
			User:    "postgres",
			DBName:  "weibo_spider",
			SSLMode: "disable",
		},
		RandomWaitPages:   [2]int{1, 5},
		RandomWaitSeconds: [2]int{6, 10},
		WriteMode:         []string{"csv"},
		OutputDir:         "./output",
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Cookie == "" {
		return fmt.Errorf("cookie 不能为空")
	}
	return nil
}

// DSN 返回数据库连接字符串
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}
