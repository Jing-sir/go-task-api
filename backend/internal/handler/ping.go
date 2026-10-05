package handler

import (
	"go-task-api/internal/common"

	"github.com/gin-gonic/gin"
)

// GetPing 用于快速检查服务是否已启动、路由是否可访问。
//
//	@Summary		健康检查
//	@Description	不查数据库、不需要鉴权，只确认进程活着、路由能通
//	@Tags			系统
//	@Produce		json
//	@Success		200	{object}	EmptyResponse
//	@Router			/ping [get]
func GetPing(c *gin.Context) {
	common.Success(c, nil)
}
