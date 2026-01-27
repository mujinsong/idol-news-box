package store

import (
	"github.com/yuanhuaxi/weibo-spider/internal/model"
	"gorm.io/gorm"
)

// UserStore 用户存储
type UserStore struct {
	db *gorm.DB
}

// NewUserStore 创建用户存储
func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

// Create 创建用户
func (s *UserStore) Create(user *model.User) error {
	return s.db.Create(user).Error
}

// GetByID 根据ID获取用户
func (s *UserStore) GetByID(id uint) (*model.User, error) {
	var user model.User
	err := s.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (s *UserStore) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := s.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (s *UserStore) Update(user *model.User) error {
	return s.db.Save(user).Error
}

// Delete 删除用户
func (s *UserStore) Delete(id uint) error {
	return s.db.Delete(&model.User{}, id).Error
}

// List 获取用户列表
func (s *UserStore) List(page, pageSize int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	s.db.Model(&model.User{}).Count(&total)

	offset := (page - 1) * pageSize
	err := s.db.Offset(offset).Limit(pageSize).Order("id DESC").Find(&users).Error
	return users, total, err
}
