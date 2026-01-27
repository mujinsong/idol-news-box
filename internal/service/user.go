package service

import (
	"fmt"

	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/model"
	"github.com/yuanhuaxi/weibo-spider/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务
type UserService struct {
	store *store.UserStore
}

// NewUserService 创建用户服务
func NewUserService(store *store.UserStore) *UserService {
	return &UserService{store: store}
}

// Create 创建用户
func (s *UserService) Create(req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	// 检查用户名是否已存在
	existing, _ := s.store.GetByUsername(req.Username)
	if existing != nil {
		return nil, fmt.Errorf("用户名已存在")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	user := &model.User{
		Username:    req.Username,
		Password:    string(hashedPassword),
		Nickname:    req.Nickname,
		WeiboCookie: req.WeiboCookie,
		Status:      1,
	}

	if err := s.store.Create(user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return s.toResponse(user), nil
}

// GetByID 根据ID获取用户
func (s *UserService) GetByID(id uint) (*dto.UserResponse, error) {
	user, err := s.store.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return s.toResponse(user), nil
}

// Update 更新用户
func (s *UserService) Update(id uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := s.store.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.WeiboUID != "" {
		user.WeiboUID = req.WeiboUID
	}
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("密码加密失败: %w", err)
		}
		user.Password = string(hashedPassword)
	}
	if req.WeiboCookie != "" {
		user.WeiboCookie = req.WeiboCookie
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.store.Update(user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	return s.toResponse(user), nil
}

// Delete 删除用户
func (s *UserService) Delete(id uint) error {
	_, err := s.store.GetByID(id)
	if err != nil {
		return fmt.Errorf("用户不存在")
	}
	return s.store.Delete(id)
}

// List 获取用户列表
func (s *UserService) List(page, pageSize int) (*dto.UserListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	users, total, err := s.store.List(page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	resp := &dto.UserListResponse{
		Users: make([]*dto.UserResponse, len(users)),
		Total: total,
		Page:  page,
		Size:  pageSize,
	}

	for i, u := range users {
		resp.Users[i] = s.toResponse(u)
	}

	return resp, nil
}

// toResponse 转换为响应
func (s *UserService) toResponse(user *model.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		WeiboUID:    user.WeiboUID,
		WeiboCookie: user.WeiboCookie,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
