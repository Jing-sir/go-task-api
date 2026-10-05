package handler

import "go-task-api/internal/model"

// 本文件只为生成 Swagger 文档而存在，运行时代码不会用到这些类型。
//
// 为什么需要它们：项目所有接口都用 common.Response 包一层，它的 Data 字段是 any。
// any 在文档里只能显示成「任意类型」，看文档的人不知道实际会返回什么。
// 所以这里按接口把 Data 的真实类型写成具体结构，注解里引用这些类型，
// 生成出来的文档才能展开每个字段。
//
// 代价是多一份和 common.Response 平行的定义；收益是文档真的有用。
// 这是 swag 这类「从注解生成文档」工具的常见做法。

// TaskResponse 单个任务的响应。
type TaskResponse struct {
	Code    int        `json:"code" example:"0"`
	Message string     `json:"message" example:"ok"`
	Data    model.Task `json:"data"`
}

// TaskListData 任务列表的分页数据。
type TaskListData struct {
	List     []model.Task `json:"list"`
	Total    int64        `json:"total" example:"25"`
	PageNo   int          `json:"page_no" example:"1"`
	PageSize int          `json:"page_size" example:"20"`
}

// TaskListResponse 任务列表的响应。
type TaskListResponse struct {
	Code    int          `json:"code" example:"0"`
	Message string       `json:"message" example:"ok"`
	Data    TaskListData `json:"data"`
}

// UserResponse 用户信息的响应。
type UserResponse struct {
	Code    int        `json:"code" example:"0"`
	Message string     `json:"message" example:"ok"`
	Data    model.User `json:"data"`
}

// LoginData 登录成功返回的 token 和用户信息。
type LoginData struct {
	Token string     `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  model.User `json:"user"`
}

// LoginResponse 登录接口的响应。
type LoginResponse struct {
	Code    int       `json:"code" example:"0"`
	Message string    `json:"message" example:"ok"`
	Data    LoginData `json:"data"`
}

// AvatarResponse 头像上传接口的响应，data 是文件保存路径。
type AvatarResponse struct {
	Code    int    `json:"code" example:"0"`
	Message string `json:"message" example:"ok"`
	Data    string `json:"data" example:"uploads/avatars/1_1759400000.png"`
}

// EmptyResponse 只表示成功、没有业务数据的响应（如删除成功）。
type EmptyResponse struct {
	Code    int      `json:"code" example:"0"`
	Message string   `json:"message" example:"ok"`
	Data    struct{} `json:"data"`
}

// ErrorResponse 失败响应。
//
// 注意 code 和 HTTP 状态码是同一个值（common.Error 里 c.JSON(code, ...) 这么写的），
// 400 参数错误、401 未授权、404 找不到、429 请求过频、500 服务内部错误。
type ErrorResponse struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"标题不能为空"`
}
