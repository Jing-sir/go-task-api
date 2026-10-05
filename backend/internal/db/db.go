package db

import (
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func DBRegister(dsn string) (*gorm.DB, error) {
	var err error

	// dsn := "host=localhost user=postgres password= dbname=go_task_db port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		slog.Error("数据库连接失败", "error", err)
		return nil, err
	}

	// 判断db链接
	sqlDb, err := DB.DB()

	if err != nil {
		slog.Error("数据库连接失败", "error", err)
		return nil, err
	}

	// 最大连接数
	sqlDb.SetMaxOpenConns(25)
	// 最大空闲连接数
	sqlDb.SetMaxIdleConns(10)
	// 连接最长存活时间
	sqlDb.SetConnMaxLifetime(time.Hour)

	return DB, err
}
