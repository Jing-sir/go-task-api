package service

import (
	"context"
	"errors"
	"fmt"
	"go-task-api/internal/common"
	"go-task-api/internal/model"
	"go-task-api/internal/repository"
	"strings"

	"gorm.io/gorm"
)

// TaskService 负责承接任务相关业务逻辑。
type TaskService struct {
	repo repository.TaskRepository
}

// CreateTaskRequest 定义创建任务接口的请求参数。
type CreateTaskRequest struct {
	Title       string `json:"title" binding:"required,min=2,max=20"`
	Description string `json:"description" binding:"omitempty,max=132"`
	Status      bool   `json:"status"`
}

// UpdateTaskRequest 定义更新任务接口的请求参数。
type UpdateTaskRequest struct {
	Title       string `json:"title" binding:"required,min=2,max=20"`
	Description string `json:"description" binding:"omitempty,max=132"`
	Status      bool   `json:"status"`
}

// NewTaskService 创建任务服务实例。
func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// Create 校验并创建任务。
func (s *TaskService) Create(req *CreateTaskRequest, userID uint64, ctx context.Context) (*model.Task, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, common.BadRequest("标题不能为空")
	}

	task := model.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		UserID:      userID,
	}

	if err := s.repo.Create(&task, ctx); err != nil {
		return nil, common.InternalError(fmt.Sprintf("创建任务失败: %v", err))
	}

	return &task, nil
}

// GetByID 查询单个任务。
func (s *TaskService) GetByID(id uint, userID uint64, ctx context.Context) (*model.Task, error) {
	task, err := s.repo.GetByID(id, userID, ctx)
	if err != nil {
		return nil, translateGetError(err)
	}

	return task, nil
}

// translateGetError 把查询错误翻译成合适的业务错误。
//
// 查不到记录是「正常的找不到」，要返回 404；只有真正的数据库故障才算 500。
// 注意 repo 的查询都带 user_id 过滤，所以别人的任务对当前用户同样是「不存在」，
// 这里返回 404 既是正确的语义，也避免泄漏「这个 ID 其实存在」的信息。
func translateGetError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return common.NotFound("任务不存在")
	}
	return common.InternalError(fmt.Sprintf("查询任务失败: %v", err))
}

// List 查询任务列表。
func (s *TaskService) List(userID uint64, pageNo int, pageSize int, ctx context.Context) ([]model.Task, int64, error) {
	tasks, total, err := s.repo.List(userID, pageNo, pageSize, ctx)
	if err != nil {
		return nil, 0, common.InternalError(fmt.Sprintf("获取任务列表失败: %v", err))
	}

	return tasks, total, nil
}

// Update 先查询任务，再更新字段并保存。
func (s *TaskService) Update(id uint, req *UpdateTaskRequest, userID uint64, ctx context.Context) (*model.Task, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, common.BadRequest("标题不能为空")
	}

	task, err := s.repo.GetByID(id, userID, ctx)
	if err != nil {
		return nil, translateGetError(err)
	}

	task.Title = req.Title
	task.Status = req.Status
	task.Description = req.Description

	err = s.repo.Update(task, ctx)
	if err != nil {
		return nil, common.InternalError(fmt.Sprintf("更新任务失败: %v", err))
	}

	return task, nil
}

// Delete 删除指定任务。
func (s *TaskService) Delete(id uint, userID uint64, ctx context.Context) error {
	res, err := s.repo.Delete(id, userID, ctx)
	if err != nil {
		return common.InternalError(fmt.Sprintf("删除任务失败: %v", err))
	}
	if res == 0 {
		return common.NotFound("任务不存在")
	}

	return nil
}
