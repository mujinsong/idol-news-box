package dto

import "time"

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Nickname    string `json:"nickname"`
	WeiboCookie string `json:"weibo_cookie"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Nickname    string `json:"nickname"`
	Password    string `json:"password"`
	WeiboUID    string `json:"weibo_uid"`
	WeiboCookie string `json:"weibo_cookie"`
	Status      *int   `json:"status"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	Nickname    string    `json:"nickname"`
	WeiboUID    string    `json:"weibo_uid"`
	WeiboCookie string    `json:"weibo_cookie"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Users []*UserResponse `json:"users"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}
