package repository

import (
	"context"
	"go-task-api/internal/model"

	"gorm.io/gorm"
)

// UserRepository 定义用户数据访问层需要提供的方法。
type UserRepository interface {
	Create(user *model.User, ctx context.Context) error
	FindByEmail(email string, ctx context.Context) (*model.User, error)
	GetByID(userID uint64, ctx context.Context) (*model.User, error)
	UpdateAvatar(userID uint64, avatar string, ctx context.Context) error
}

// userRepository 是 UserRepository 的 GORM 实现。
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 向数据库写入一条新用户记录。
func (r *userRepository) Create(user *model.User, ctx context.Context) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByEmail 按邮箱查询用户记录。
func (r *userRepository) FindByEmail(email string, ctx context.Context) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// 通过id 查找用户当前信息
func (r *userRepository) GetByID(userID uint64, ctx context.Context) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// 更新用户头像
func (r *userRepository) UpdateAvatar(userID uint64, avatar string, ctx context.Context) error {
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("avatar", avatar).Error; err != nil {
		return err
	}

	return nil
}
