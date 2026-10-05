package handler

import (
	"go-task-api/internal/common"
	"go-task-api/internal/service"

	"github.com/gin-gonic/gin"
)

// Register 处理用户注册请求。
//
//	@Summary		注册
//	@Description	用邮箱和密码注册新用户。密码用 bcrypt 加密后入库，响应里不会返回密码字段
//	@Tags			认证
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CreateUserRequest	true	"邮箱和密码"
//	@Success		200		{object}	UserResponse
//	@Failure		400		{object}	ErrorResponse	"参数校验失败，或邮箱已注册"
//	@Router			/api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.BadRequest(err.Error()))
		return
	}

	user, err := h.userService.Register(&req, c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	common.Success(c, user)
}

// Login 处理用户登录请求。
//
//	@Summary		登录
//	@Description	校验邮箱密码，成功返回 JWT token。后续请求把它放进 Authorization 头，格式 "Bearer {token}"
//	@Tags			认证
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CreateUserRequest	true	"邮箱和密码"
//	@Success		200		{object}	LoginResponse
//	@Failure		400		{object}	ErrorResponse	"参数校验失败"
//	@Failure		401		{object}	ErrorResponse	"邮箱或密码错误"
//	@Router			/api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.BadRequest(err.Error()))
		return
	}

	user, err := h.userService.Login(&req, c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	common.Success(c, user)
}

// GetUserInfo 返回当前登录用户的信息。
//
//	@Summary		当前用户信息
//	@Description	用 token 里的用户 ID 查询。走 Cache Aside：先查 Redis，没命中再查数据库并回填缓存，缓存 10 分钟
//	@Tags			用户
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserResponse
//	@Failure		401	{object}	ErrorResponse	"未授权"
//	@Failure		404	{object}	ErrorResponse	"用户不存在"
//	@Router			/api/v1/userInfo [get]
func (h *Handler) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")

	if !exists {
		c.Error(common.Unauthorized("未授权"))
		return
	}

	user, err := h.userService.GetByID(userID.(uint64), c.Request.Context())

	if err != nil {
		c.Error(err)
		return
	}
	common.Success(c, user)
}

// UploadAvatar 处理头像上传请求。
//
//	@Summary		上传头像
//	@Description	上传头像文件并更新用户记录。限制 2MB 以内，只接受 png / jpeg / jpg。返回的是服务器上的保存路径
//	@Tags			用户
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"头像文件，2MB 以内，png/jpeg/jpg"
//	@Success		200		{object}	AvatarResponse
//	@Failure		400		{object}	ErrorResponse	"文件超出大小，或类型不支持"
//	@Failure		401		{object}	ErrorResponse	"未授权"
//	@Failure		404		{object}	ErrorResponse	"没有选择文件"
//	@Router			/api/v1/avatar [post]
func (h *Handler) UploadAvatar(c *gin.Context) {
	file, err := c.FormFile("file")

	if err != nil {
		c.Error(common.NotFound("请选择文件"))
		return
	}

	userID, exists := c.Get("user_id")

	if !exists {
		c.Error(common.Unauthorized("未授权"))
		return
	}

	url, err := h.userService.UploadAvatar(file, userID.(uint64), c.Request.Context())

	if err != nil {
		c.Error(err)
		return
	}
	common.Success(c, url)
}
