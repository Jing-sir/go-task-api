package model

import "time"

// User 定义用户表结构。
type User struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Email     string    `gorm:"column:email;type:varchar(100);uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"column:password;type:varchar(255);not null" json:"-"`
	Avatar    string    `gorm:"column:avatar;type:varchar(255)" json:"avatar"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}
