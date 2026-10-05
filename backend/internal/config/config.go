package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost         string `json:"db_host"`           //  — 数据库地址，如 localhost
	DBPort         string `json:"db_port,omitempty"` //  — 数据库端口，如 5432
	DBUser         string `json:"db_user"`           // — 数据库用户名
	DBPassword     string `json:"db_password"`       //  — 数据库密码
	DBName         string `json:"db_name"`           //  — 数据库名，当前是 go_task_db
	DBSSLmode      string `json:"dbssl_mode"`        // — SSL 模式，当前是 disable
	JWTSecret      string `json:"jwt_secret"`        // 签名密钥
	JWTExpireHours int    `json:"jwt_expire_hours"`  // 过期时长
	ServerPort     string `json:"server_port"`       // 监听端口·
	RedisAddr      string `json:"redis_addr"`        // redis address
}

// 加载静态环境变量配置
func Load() Config {
	godotenv.Load()

	config := Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBSSLmode:  os.Getenv("DB_SSLMODE"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		ServerPort: os.Getenv("SERVER_PORT"),
		RedisAddr:  os.Getenv("REDIS_ADDR"),
	}

	hours, err := strconv.Atoi(os.Getenv("JWT_EXPIRE_HOURS"))
	if err != nil {
		hours = 3
	}
	config.JWTExpireHours = hours

	return config
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"host='%s' port='%s' user='%s' password='%s' dbname='%s' sslmode='%s'",
		c.DBHost,
		c.DBPort,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBSSLmode,
	)
}

func (c Config) Port() string {
	return c.ServerPort
}
