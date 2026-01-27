package model

import (
	"gorm.io/gorm"
)

// User 平台用户
type User struct {
	gorm.Model
	Username    string `gorm:"size:50;not null;uniqueIndex" json:"username"` // 用户名
	Password    string `gorm:"size:255;not null" json:"-"`                   // 密码（加密存储）
	Nickname    string `gorm:"size:100" json:"nickname"`                     // 昵称
	WeiboCookie string `gorm:"type:text" json:"-"`                           // 微博Cookie
	WeiboUID    string `gorm:"size:50" json:"weibo_uid"`                     // 微博用户ID（从Cookie获取）
	Status      int    `gorm:"default:1" json:"status"`                      // 状态：1启用 0禁用
}

// TableName 表名
func (User) TableName() string {
	return "users"
}
