package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"go-task-api/internal/common"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// RequestID 给每个请求分配唯一 ID，并准备一个带该 ID 的 logger。
//
// 上游（网关、前端、另一个服务）已经带了 X-Request-Id 就沿用，
// 这样一次调用链跨多个服务时能用同一个 ID 串起来；没带才新生成。
func RequestID(c *gin.Context) {
	id := c.GetHeader(common.HeaderRequestID)
	if id == "" {
		id = newRequestID()
	}

	c.Set(common.ContextKeyRequestID, id)
	// 本次请求的所有日志自动带上 request_id，不用每处手写
	c.Set(common.ContextKeyLogger, slog.Default().With("request_id", id))
	// 回写响应头，调用方（含前端）能拿到 ID 来报问题
	c.Header(common.HeaderRequestID, id)

	c.Next()
}

// newRequestID 生成 16 字节随机值的十六进制串。
func newRequestID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
