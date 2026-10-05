package service

import (
	"context"
	"errors"
	"testing"

	"go-task-api/internal/model"
)

// mockTaskRepo 是 TaskRepository 的假实现：不碰数据库，用字段控制每次调用返回什么。
// 这样测试只关注 service 的业务逻辑，不依赖真实数据库。
type mockTaskRepo struct {
	err     error       // 非 nil 时，所有方法都返回这个错误，模拟数据库出错
	task    *model.Task // 非 nil 时，GetByID 返回它
	deleted bool        // 记录 Delete 是否被调用过（可选的校验用）
}

func (m *mockTaskRepo) Create(task *model.Task, ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	task.ID = 1
	return nil
}

func (m *mockTaskRepo) GetByID(id uint, userID uint64, ctx context.Context) (*model.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.task == nil {
		return nil, errors.New("record not found")
	}
	return m.task, nil
}

func (m *mockTaskRepo) List(userID uint64, pageNo, pageSize int, ctx context.Context) ([]model.Task, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	if m.task == nil {
		return nil, 0, errors.New("record not found")
	}
	return []model.Task{*m.task}, 1, nil
}

func (m *mockTaskRepo) Update(task *model.Task, ctx context.Context) error {
	return m.err
}

func (m *mockTaskRepo) Delete(id uint, userID uint64, ctx context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.deleted = true
	return 1, nil
}

// TestCreateSuccess 创建任务成功：user_id 应该被绑定到 task 上。
func TestCreateSuccess(t *testing.T) {
	svc := NewTaskService(&mockTaskRepo{})

	task, err := svc.Create(&CreateTaskRequest{Title: "买牛奶", Description: "楼下便利店", Status: false}, 42, context.Background())

	if err != nil {
		t.Fatalf("期望无错误，得到: %v", err)
	}
	if task.UserID != 42 {
		t.Errorf("期望 UserID=42，得到: %d", task.UserID)
	}
	if task.Title != "买牛奶" {
		t.Errorf("期望 Title=买牛奶，得到: %s", task.Title)
	}
}

// TestCreateEmptyTitle 标题为空：service 应该自己拦下，不调用 repo。
func TestCreateEmptyTitle(t *testing.T) {
	svc := NewTaskService(&mockTaskRepo{})

	_, err := svc.Create(&CreateTaskRequest{Title: ""}, 42, context.Background())

	if err == nil {
		t.Fatal("期望报错'标题不能为空'，实际没有错误")
	}
	if err.Error() != "标题不能为空" {
		t.Errorf("期望错误'标题不能为空'，得到: %v", err)
	}
}

// TestCreateRepoError repo 层出错：service 应该把它包成业务错误返回。
func TestCreateRepoError(t *testing.T) {
	svc := NewTaskService(&mockTaskRepo{err: errors.New("db down")})

	_, err := svc.Create(&CreateTaskRequest{Title: "买牛奶"}, 42, context.Background())

	if err == nil {
		t.Fatal("期望报错，实际没有错误")
	}
}

// TestGetByIDNotFound 任务不存在：repo 返回 record not found，service 透传。
func TestGetByIDNotFound(t *testing.T) {
	svc := NewTaskService(&mockTaskRepo{}) // task 为 nil → GetByID 返回 not found

	_, err := svc.GetByID(999, 42, context.Background())

	if err == nil {
		t.Fatal("期望报错，实际没有错误")
	}
}

// TestGetByIDSuccess 查询单个任务成功。
func TestGetByIDSuccess(t *testing.T) {
	want := &model.Task{ID: 1, UserID: 42, Title: "买牛奶"}
	svc := NewTaskService(&mockTaskRepo{task: want})

	task, err := svc.GetByID(1, 42, context.Background())

	if err != nil {
		t.Fatalf("期望无错误，得到: %v", err)
	}
	if task.Title != "买牛奶" {
		t.Errorf("期望 Title=买牛奶，得到: %s", task.Title)
	}
}
