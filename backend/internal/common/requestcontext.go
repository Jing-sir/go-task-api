package common

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// HeaderRequestID 是请求 ID 在 HTTP 头里的名字，请求和响应都用它。
const HeaderRequestID = "X-Request-Id"

// 请求级信息在 gin.Context 里的 key。取值请走下面两个函数，不要直接 c.Get。
const (
	ContextKeyRequestID = "request_id"
	ContextKeyLogger    = "logger"
)

// RequestIDFrom 取出本次请求的 ID，取不到返回空串。
func RequestIDFrom(c *gin.Context) string {
	if v, ok := c.Get(ContextKeyRequestID); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// LoggerFrom 取出带 request_id 的 logger。
//
// 中间件没跑到（比如单测直接调 handler）时退回默认 logger，
// 这样调用方永远不用判空，拿到的 logger 一定可用。
func LoggerFrom(c *gin.Context) *slog.Logger {
	if v, ok := c.Get(ContextKeyLogger); ok {
		if log, ok := v.(*slog.Logger); ok {
			return log
		}
	}
	return slog.Default()
}
