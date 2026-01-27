package dto

import (
	"fmt"
	"time"
)

// User 用户信息
type User struct {
	ID             string `json:"id"`
	Nickname       string `json:"nickname"`
	Gender         string `json:"gender"`
	Location       string `json:"location"`
	Birthday       string `json:"birthday"`
	Description    string `json:"description"`
	VerifiedReason string `json:"verified_reason"`
	WeiboNum       int    `json:"weibo_num"`
	Following      int    `json:"following"`
	Followers      int    `json:"followers"`
}

// CSVHeader 返回CSV表头
func (u *User) CSVHeader() []string {
	return []string{
		"id", "昵称", "性别", "地区", "生日",
		"简介", "认证信息", "微博数", "关注数", "粉丝数",
	}
}

// CSVRow 返回CSV行数据
func (u *User) CSVRow() []string {
	return []string{
		u.ID, u.Nickname, u.Gender, u.Location, u.Birthday,
		u.Description, u.VerifiedReason,
		fmt.Sprintf("%d", u.WeiboNum),
		fmt.Sprintf("%d", u.Following),
		fmt.Sprintf("%d", u.Followers),
	}
}

// SpecialFollowUser 特别关注用户（简化版）
type SpecialFollowUser struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

// SpecialFollowList 特别关注列表响应
type SpecialFollowList struct {
	Users []SpecialFollowUser `json:"users"`
	Total int                 `json:"total"`
}

// SpecialFollowSyncResult 同步结果
type SpecialFollowSyncResult struct {
	OwnerID  string    `json:"owner_id"`
	Total    int       `json:"total"`
	SyncedAt time.Time `json:"synced_at"`
}

// SpecialFollowDBUser 数据库中的特别关注用户
type SpecialFollowDBUser struct {
	ID        uint      `json:"id"`
	OwnerID   string    `json:"owner_id"`
	UserID    string    `json:"user_id"`
	Nickname  string    `json:"nickname"`
	SyncedAt  time.Time `json:"synced_at"`
	CreatedAt time.Time `json:"created_at"`
}

// SpecialFollowDBList 数据库特别关注列表响应
type SpecialFollowDBList struct {
	Users []SpecialFollowDBUser `json:"users"`
	Total int                   `json:"total"`
}
