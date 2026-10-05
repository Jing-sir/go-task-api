package common

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// 成功回调
func Success(c *gin.Context, data any) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(200, Response{Code: 0, Message: "ok", Data: data})
}

// 失败/错误回调
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{Code: code, Message: message, Data: nil})
}
