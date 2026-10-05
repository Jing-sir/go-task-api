package middleware

import (
	"go-task-api/internal/common"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthToken(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		// 不是 Bearer + 空格
		c.AbortWithStatus(401)
		return
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	claims, err := common.ParseToken(token)

	if err != nil {
		common.Error(c, 401, "未授权")
		c.Abort()
		return
	}

	c.Set("user_id", claims.UserID)
	c.Next()
}
