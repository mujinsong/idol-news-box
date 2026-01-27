package model

import (
	"time"

	"gorm.io/gorm"
)

// SpecialFollow 特别关注用户
type SpecialFollow struct {
	gorm.Model
	OwnerID  string    `gorm:"size:50;not null;index" json:"owner_id"`                      // Cookie 所有者的用户ID
	UserID   string    `gorm:"size:50;not null;index:idx_owner_user,unique" json:"user_id"` // 被关注的微博用户ID
	Nickname string    `gorm:"size:100" json:"nickname"`                                    // 昵称
	SyncedAt time.Time `json:"synced_at"`                                                   // 最后同步时间
}

// TableName 表名
func (SpecialFollow) TableName() string {
	return "special_follows"
}
