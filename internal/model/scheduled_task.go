package model

import (
	"time"

	"gorm.io/gorm"
)

// ScheduledTask 定时任务
type ScheduledTask struct {
	gorm.Model
	Name      string    `gorm:"size:100;not null" json:"name"`
	Cron      string    `gorm:"size:50;not null" json:"cron"`
	UserID    string    `gorm:"size:50;not null" json:"user_id"`
	SinceDate time.Time `json:"since_date"`
	EndDate   time.Time `json:"end_date"`
	Filter    int       `gorm:"default:0" json:"filter"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
}

// TableName 表名
func (ScheduledTask) TableName() string {
	return "scheduled_tasks"
}
