package handler

import (
	"encoding/json"
	"go-task-api/internal/common"
	"go-task-api/internal/model"
	"go-task-api/internal/service"
	"go-task-api/internal/ws"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler 负责接收 HTTP 请求，并调用 service 层处理任务业务。
type Handler struct {
	taskService *service.TaskService
	userService *service.UserService
	hub         *ws.Hub
}

type taskEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// New 创建任务 Handler。
func New(taskService *service.TaskService, userService *service.UserService, hubService *ws.Hub) *Handler {
	return &Handler{taskService: taskService, userService: userService, hub: hubService}
}

// Create 处理创建任务请求。
//
//	@Summary		创建任务
//	@Description	创建一条属于当前登录用户的任务，创建成功后通过 WebSocket 推送 task.created 事件
//	@Tags			任务
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		service.CreateTaskRequest	true	"任务内容"
//	@Success		200		{object}	TaskResponse
//	@Failure		400		{object}	ErrorResponse	"参数校验失败"
//	@Failure		401		{object}	ErrorResponse	"未授权"
//	@Router			/api/v1/tasks [post]
func (h *Handler) Create(c *gin.Context) {
	var req service.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.BadRequest(err.Error()))
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	task, err := h.taskService.Create(&req, userID, c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	// struct 转 []byte 统一处理错误，发送单人广播
	data, err := json.Marshal(taskEvent{Type: "task.created", Data: task})
	if err != nil {
		c.Error(err)
		return
	}
	h.hub.BroadcastToUser(userID, data)

	common.Success(c, task)
}

// List 处理任务列表查询请求。
//
//	@Summary		任务列表
//	@Description	分页查询当前登录用户的任务，按 id 倒序。只返回自己的数据
//	@Tags			任务
//	@Produce		json
//	@Security		BearerAuth
//	@Param			pageNo		query		int	false	"页码，从 1 开始，非法值回落为 1"		default(1)
//	@Param			pageSize	query		int	false	"每页条数，1-100，非法值回落为 20"	default(20)
//	@Success		200			{object}	TaskListResponse
//	@Failure		401			{object}	ErrorResponse	"未授权"
//	@Router			/api/v1/tasks [get]
func (h *Handler) List(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	pageNoVal := c.Query("pageNo")
	pageNo, err := strconv.Atoi(pageNoVal)

	if err != nil || pageNo < 1 {
		pageNo = 1
	}

	pageSizeVal := c.Query("pageSize")
	pageSize, err := strconv.Atoi(pageSizeVal)

	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tasks, total, err := h.taskService.List(userID, pageNo, pageSize, c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	common.Success(c, struct {
		List     []model.Task `json:"list"`
		Total    int64        `json:"total"`
		PageNo   int          `json:"page_no"`
		PageSize int          `json:"page_size"`
	}{
		List:     tasks,
		Total:    total,
		PageSize: pageSize,
		PageNo:   pageNo,
	})
}

// Update 处理更新任务请求。
//
//	@Summary		更新任务
//	@Description	全量更新一条自己的任务，成功后通过 WebSocket 推送 task.updated 事件
//	@Tags			任务
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"任务 ID"
//	@Param			request	body		service.UpdateTaskRequest	true	"要更新的内容"
//	@Success		200		{object}	TaskResponse
//	@Failure		400		{object}	ErrorResponse	"参数校验失败或 ID 非法"
//	@Failure		401		{object}	ErrorResponse	"未授权"
//	@Failure		404		{object}	ErrorResponse	"任务不存在，或不属于当前用户"
//	@Router			/api/v1/tasks/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req service.UpdateTaskRequest

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.BadRequest(err.Error()))
		return
	}

	task, err := h.taskService.Update(id, &req, userID, c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	// struct 转 []byte 统一处理错误，发送单人广播
	data, err := json.Marshal(taskEvent{Type: "task.updated", Data: task})
	if err != nil {
		c.Error(err)
		return
	}
	h.hub.BroadcastToUser(userID, data)

	common.Success(c, task)
}

// Delete 处理删除任务请求。
//
//	@Summary		删除任务
//	@Description	软删除一条自己的任务，成功后通过 WebSocket 推送 task.deleted 事件
//	@Tags			任务
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"任务 ID"
//	@Success		200	{object}	EmptyResponse
//	@Failure		400	{object}	ErrorResponse	"ID 非法"
//	@Failure		401	{object}	ErrorResponse	"未授权"
//	@Failure		404	{object}	ErrorResponse	"任务不存在，或不属于当前用户"
//	@Router			/api/v1/tasks/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	err := h.taskService.Delete(id, userID, c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	// struct 转 []byte 统一处理错误，发送单人广播
	data, err := json.Marshal(taskEvent{Type: "task.deleted", Data: id})
	if err != nil {
		c.Error(err)
		return
	}
	h.hub.BroadcastToUser(userID, data)

	common.Success(c, nil)
}

// GetByID 处理按 ID 查询单个任务请求。
//
//	@Summary		任务详情
//	@Description	查询一条自己的任务。别人的任务一律返回 404，不区分「不存在」和「不属于你」，避免泄漏 ID 是否有效
//	@Tags			任务
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"任务 ID"
//	@Success		200	{object}	TaskResponse
//	@Failure		400	{object}	ErrorResponse	"ID 非法"
//	@Failure		401	{object}	ErrorResponse	"未授权"
//	@Failure		404	{object}	ErrorResponse	"任务不存在，或不属于当前用户"
//	@Router			/api/v1/tasks/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	task, err := h.taskService.GetByID(id, userID, c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	common.Success(c, task)
}

func parseID(c *gin.Context) (uint, bool) {
	id := c.Param("id")
	u64, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.Error(common.BadRequest("无效的 ID 参数"))
		return 0, false
	}
	return uint(u64), true
}

// 获取user id
func getUserID(c *gin.Context) (uint64, bool) {
	userID, exists := c.Get("user_id")

	if !exists {
		c.Error(common.Unauthorized("未授权"))
		return 0, false
	}
	return userID.(uint64), true
}
