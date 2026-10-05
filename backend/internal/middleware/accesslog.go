package middleware

import (
	"go-task-api/internal/common"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLog 为每个请求输出一条结构化访问日志。
//
// 取代 gin 默认的文本 Logger：字段固定、可被日志系统直接解析，
// 并且带上 request_id，能和业务日志、错误日志串到一起。
func AccessLog(c *gin.Context) {
	start := time.Now()

	c.Next()

	// 用路由模板而不是真实路径：/api/v1/tasks/:id 而不是 /api/v1/tasks/123。
	// 真实路径会让日志和指标按 ID 散开，无法聚合。
	route := c.FullPath()
	if route == "" {
		// 没匹配到任何路由（404），FullPath 为空，用 undefined 占位，
		// 避免把乱扫的 URL 原样记进来
		route = "undefined"
	}

	log := common.LoggerFrom(c)
	attrs := []any{
		"method", c.Request.Method,
		"route", route,
		"path", c.Request.URL.Path,
		"status", c.Writer.Status(),
		"duration_ms", float64(time.Since(start).Microseconds()) / 1000,
		"client_ip", c.ClientIP(),
		"bytes", c.Writer.Size(),
	}

	// 5xx 用 Error 级别，便于日志系统按级别告警
	if c.Writer.Status() >= 500 {
		log.Error("请求完成", attrs...)
		return
	}
	log.Info("请求完成", attrs...)
}
