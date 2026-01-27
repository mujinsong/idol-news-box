package repository

import (
	"github.com/yuanhuaxi/weibo-spider/internal/model"
	"github.com/yu
	"gorm.io/gorm"
)

// TaskRepository 定时任务仓库
type TaskRepository struct {
	getDB func() *gorm.DB
}

// NewTaskRepository 创建仓库实例
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		getDB: func() *gorm.DB {
			return store.GetFactory().GetDB()
		},
	}
}

// FindAllEnabled 获取所有启用的任务
func (r *TaskRepository) FindAllEnabled() ([]*model.ScheduledTask, error) {
	var tasks []*model.ScheduledTask
	err := r.getDB().Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}

// FindByID 根据ID获取任务
func (r *TaskRepository) FindByID(id uint) (*model.ScheduledTask, error) {
	var task model.ScheduledTask
	err := r.getDB().First(&task, id).Error
	return &task, err
}

// Create 创建任务
func (r *TaskRepository) Create(task *model.ScheduledTask) error {
	return r.getDB().Create(task).Error
}

// Update 更新任务
func (r *TaskRepository) Update(task *model.ScheduledTask) error {
	return r.getDB().Save(task).Error
}

// Delete 删除任务
func (r *TaskRepository) Delete(id uint) error {
	return r.getDB().Delete(&model.ScheduledTask{}, id).Error
}
