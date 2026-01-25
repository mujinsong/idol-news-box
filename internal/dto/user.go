package dto

import "fmt"

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
