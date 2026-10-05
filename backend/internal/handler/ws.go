package handler

import (
	"go-task-api/internal/common"
	"go-task-api/internal/ws"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 创建一个 ws upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleWebSocket 把 HTTP 连接升级成 WebSocket 长连接。
//
//	@Summary		WebSocket 连接
//	@Description	升级成 WebSocket，接收属于当前用户的任务变更事件。
//	@Description	推送消息格式为 {"type":"task.created|task.updated|task.deleted","data":...}，
//	@Description	删除事件的 data 只有任务 ID。
//	@Description	token 只能走 query 参数，因为浏览器原生 WebSocket 不支持自定义请求头。
//	@Tags			实时
//	@Param			token	query	string	true	"登录拿到的 JWT token"
//	@Success		101		"切换协议成功，连接已建立"
//	@Failure		401		{object}	ErrorResponse	"token 缺失或无效"
//	@Router			/api/v1/ws [get]
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// 获取token
	token := c.Query("token")

	claims, err := common.ParseToken(token)

	if err != nil {
		c.AbortWithStatus(401)
		return
	}

	// 把HTTP 升级为 webSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	//  如果err 不为 空 return
	if err != nil {
		return
	}

	// 创建 Client
	client := ws.NewClient(h.hub, claims.UserID, conn)
	// 注册到 hub
	h.hub.Register <- client

	// 启动两个 goroutine
	go client.ReadPump()
	go client.WritePump()
}
