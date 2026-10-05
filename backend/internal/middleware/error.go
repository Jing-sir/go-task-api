package middleware

import (
	"errors"
	"go-task-api/internal/common"

	"github.com/gin-gonic/gin"
)

func Error(c *gin.Context) {
	c.Next()

	if len(c.Errors) > 0 {
		var appErr *common.AppError

		err := c.Errors.Last().Err

		// 用请求级 logger，错误日志自带 request_id，
		// 能和同一请求的访问日志对上
		log := common.LoggerFrom(c)

		if errors.As(err, &appErr) {
			log.Error("请求处理失败", "method", c.Request.Method, "path", c.Request.URL.Path, "error", err.Error())
			common.Error(c, appErr.Code, appErr.Message)
		} else {
			log.Error("请求处理失败", "method", c.Request.Method, "path", c.Request.URL.Path, "error", err.Error())
			common.Error(c, 500, err.Error())
		}
	}
}
