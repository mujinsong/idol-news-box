package store

import (
	"github.com/yuanhuaxi/weibo-spider/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SpecialFollowStore 特别关注存储
type SpecialFollowStore struct {
	db *gorm.DB
}

// NewSpecialFollowStore 创建特别关注存储
func NewSpecialFollowStore(db *gorm.DB) *SpecialFollowStore {
	return &SpecialFollowStore{db: db}
}

// Create 创建特别关注记录
func (s *SpecialFollowStore) Create(follow *model.SpecialFollow) error {
	return s.db.Create(follow).Error
}

// BatchUpsert 批量插入或更新
func (s *SpecialFollowStore) BatchUpsert(follows []*model.SpecialFollow) error {
	if len(follows) == 0 {
		return nil
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "owner_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"nickname", "synced_at", "updated_at"}),
	}).Create(&follows).Error
}

// List 获取所有特别关注
func (s *SpecialFollowStore) List() ([]*model.SpecialFollow, error) {
	var follows []*model.SpecialFollow
	err := s.db.Order("created_at DESC").Find(&follows).Error
	return follows, err
}

// ListByOwner 获取指定用户的特别关注
func (s *SpecialFollowStore) ListByOwner(ownerID string) ([]*model.SpecialFollow, error) {
	var follows []*model.SpecialFollow
	err := s.db.Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&follows).Error
	return follows, err
}

// Delete 删除指定用户
func (s *SpecialFollowStore) Delete(ownerID, userID string) error {
	return s.db.Where("owner_id = ? AND user_id = ?", ownerID, userID).Delete(&model.SpecialFollow{}).Error
}

// DeleteByOwner 删除指定所有者的所有记录
func (s *SpecialFollowStore) DeleteByOwner(ownerID string) error {
	return s.db.Where("owner_id = ?", ownerID).Delete(&model.SpecialFollow{}).Error
}

// DeleteAll 删除所有记录
func (s *SpecialFollowStore) DeleteAll() error {
	return s.db.Where("1 = 1").Delete(&model.SpecialFollow{}).Error
}
