package model

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID          uint           `gorm:"primaryKey;autoIncrement;index:idx_tasks_user_id_id,priority:2" json:"id"`
	UserID      uint64         `gorm:"not null;index:idx_tasks_user_id_id,priority:1" json:"user_id"`
	Title       string         `gorm:"type:varchar(20);not null" json:"title"`
	Description string         `gorm:"type:varchar(100)" json:"description"`
	Status      bool           `gorm:"default:false" json:"status"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
