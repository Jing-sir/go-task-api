package repository

import (
	"context"
	"go-task-api/internal/model"

	"gorm.io/gorm"
)

// TaskRepository 定义任务数据访问层需要提供的方法。
type TaskRepository interface {
	Create(task *model.Task, ctx context.Context) error
	GetByID(id uint, userID uint64, ctx context.Context) (*model.Task, error)
	List(userID uint64, pageNo, pageSize int, ctx context.Context) ([]model.Task, int64, error)
	Update(task *model.Task, ctx context.Context) error
	Delete(id uint, userID uint64, ctx context.Context) (int64, error)
}

// taskRepository 是 TaskRepository 的 GORM 实现。
type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓储实例。
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Create 向数据库写入一条新任务记录。
func (r *taskRepository) Create(task *model.Task, ctx context.Context) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetByID 按主键查询单个任务。
func (r *taskRepository) GetByID(id uint, userID uint64, ctx context.Context) (*model.Task, error) {
	var task model.Task
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&task, id).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

// List 查询全部任务。
func (r *taskRepository) List(userID uint64, pageNo int, pageSize int, ctx context.Context) ([]model.Task, int64, error) {
	var tasks []model.Task
	var total int64

	query := r.db.WithContext(ctx).Where("user_id = ?", userID)

	// 总数量
	if err := query.Model(&model.Task{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (pageNo - 1) * pageSize

	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// Update 更新一条已有任务记录。
func (r *taskRepository) Update(task *model.Task, ctx context.Context) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// Delete 按主键删除任务记录。
func (r *taskRepository) Delete(id uint, userID uint64, ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.Task{}, id)
	return result.RowsAffected, result.Error
}
