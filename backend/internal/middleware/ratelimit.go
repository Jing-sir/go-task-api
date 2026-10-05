package middleware

import (
	"go-task-api/internal/common"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func RateLimit(r rate.Limit, b int) gin.HandlerFunc {
	var m sync.Map
	return func(c *gin.Context) {
		ip := c.ClientIP()
		actual, _ := m.LoadOrStore(ip, rate.NewLimiter(r, b))

		limiter := actual.(*rate.Limiter)

		if limiter.Allow() {
			c.Next()
		} else {
			c.Error(&common.AppError{Code: 429, Message: "请求过于频繁"})
			c.Abort()
		}

	}
}
