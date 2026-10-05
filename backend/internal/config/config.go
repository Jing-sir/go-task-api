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

// Validate 校验配置是否可用，有问题返回说明原因的错误。
//
// 目前只查一件事：JWT_SECRET 不能为空。
//
// 为什么这条值得单独拦：密钥为空时服务照样能启动、接口也都通，
// 但签出来的 token 全是无效的，表现为「一登录就莫名失效」，
// 很难想到根源是 .env 没建（.env 不在版本库里，换机器或重新克隆时不会跟着来）。
// 启动时一句话说清，比事后排查半天划算。
func (c Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET 为空，请复制 backend/.env.example 为 backend/.env 再填写")
	}
	return nil
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
