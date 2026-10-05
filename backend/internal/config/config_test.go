package config

import (
	"strings"
	"testing"
)

// TestValidate 覆盖配置校验。
//
// 现在只有一条规则：JWT_SECRET 不能为空。
// 值得测是因为出错后果隐蔽——密钥为空时服务能正常启动、接口也都通，
// 只是签出来的 token 全部无效，表现为「一登录就莫名失效」。
func TestValidate(t *testing.T) {
	t.Run("密钥为空时拒绝，并提示去复制 .env.example", func(t *testing.T) {
		err := Config{JWTSecret: ""}.Validate()
		if err == nil {
			t.Fatal("期望报错，实际通过了")
		}
		if !strings.Contains(err.Error(), ".env.example") {
			t.Fatalf("错误信息应该告诉用户怎么修，实际是: %v", err)
		}
	})

	t.Run("密钥非空时通过", func(t *testing.T) {
		if err := (Config{JWTSecret: "test-secret"}).Validate(); err != nil {
			t.Fatalf("期望通过，实际报错: %v", err)
		}
	})
}

// TestDSNQuotesEveryValue 盯住 Day 20 修过的那个 bug。
//
// 连接串里 key= 后面如果是空值且不加引号，PostgreSQL 的解析器
// 会跳过空格把下一个字段整段当成这个 key 的值吞掉。
// 当时 password 为空，结果 dbname 被吞，程序连到了错误的数据库。
// 这个测试确保那个修复不会被人不小心改回去。
func TestDSNQuotesEveryValue(t *testing.T) {
	cfg := Config{
		DBHost:     "localhost",
		DBPort:     "5433",
		DBUser:     "postgres",
		DBPassword: "", // 关键：空密码就是当年触发 bug 的输入
		DBName:     "go_task_db",
		DBSSLmode:  "disable",
	}

	dsn := cfg.DSN()

	// 空密码必须渲染成 password=''，而不是 password= 后面直接跟空格
	if !strings.Contains(dsn, "password=''") {
		t.Fatalf("空密码必须加引号，否则会吞掉后面的字段。实际: %s", dsn)
	}

	// dbname 必须还在，没被吞
	if !strings.Contains(dsn, "dbname='go_task_db'") {
		t.Fatalf("dbname 不见了或没加引号。实际: %s", dsn)
	}

	for _, field := range []string{"host", "port", "user", "password", "dbname", "sslmode"} {
		if !strings.Contains(dsn, field+"='") {
			t.Fatalf("字段 %s 的值没用单引号包住。实际: %s", field, dsn)
		}
	}
}
