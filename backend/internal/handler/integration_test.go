// 集成测试：走完整链路（HTTP 路由 → 中间件 → handler → service → repository → 真实 PostgreSQL）。
//
// 和 service 层单测的区别：单测用 mock repo，验证业务分支；
// 集成测试用真实数据库，验证的是「各层拼起来之后真的能跑」——
// 路由注册、中间件顺序、鉴权、参数绑定、SQL、数据隔离这些单测覆盖不到的地方。
//
// 数据库用独立的 go_task_db_test 库，和开发库隔离，不会污染开发数据。
// 容器没启动时整个文件的测试会 SKIP，不会让 go test ./... 失败。
//
// 跑之前先确保数据库在跑：docker compose up -d db redis
package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go-task-api/internal/cache"
	"go-task-api/internal/common"
	"go-task-api/internal/handler"
	"go-task-api/internal/model"
	"go-task-api/internal/repository"
	"go-task-api/internal/router"
	"go-task-api/internal/service"
	"go-task-api/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	testDB      *gorm.DB
	dbSkipError error
)

// TestMain 在所有测试之前建好测试库，连不上就标记原因，由各测试 SKIP。
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	// 测试期间把默认日志丢掉，保持输出干净；需要验证日志的测试会自己临时换掉
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))

	common.InitJWT("integration-test-secret", 3)
	cache.NewRedisClient(env("TEST_REDIS_ADDR", "localhost:6379"))

	testDB, dbSkipError = connectTestDB()
	os.Exit(m.Run())
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// dsn 拼接测试库连接串。
// 注意每个值都用单引号包住——空密码不加引号会把后面的字段吞掉（Day 20 踩过）。
func dsn(dbName string) string {
	return fmt.Sprintf(
		"host='%s' port='%s' user='%s' password='%s' dbname='%s' sslmode='%s'",
		env("TEST_DB_HOST", "localhost"),
		env("TEST_DB_PORT", "5433"),
		env("TEST_DB_USER", "postgres"),
		env("TEST_DB_PASSWORD", ""),
		dbName,
		env("TEST_DB_SSLMODE", "disable"),
	)
}

// connectTestDB 连上测试库；库不存在就先创建，然后建表。
func connectTestDB() (*gorm.DB, error) {
	testDBName := env("TEST_DB_NAME", "go_task_db_test")
	silent := &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)}

	// 先连到维护库（postgres），用来判断测试库存在不存在
	admin, err := gorm.Open(postgres.Open(dsn("postgres")), silent)
	if err != nil {
		return nil, fmt.Errorf("连不上数据库: %w", err)
	}

	var count int64
	if err := admin.Raw("select count(*) from pg_database where datname = ?", testDBName).Scan(&count).Error; err != nil {
		return nil, fmt.Errorf("查询测试库是否存在失败: %w", err)
	}
	if count == 0 {
		// CREATE DATABASE 不能用参数占位符，只能拼字符串。
		// 库名来自本文件常量或环境变量，不是外部输入，没有注入面。
		if err := admin.Exec("create database " + testDBName).Error; err != nil {
			return nil, fmt.Errorf("创建测试库失败: %w", err)
		}
	}
	if sqlDB, err := admin.DB(); err == nil {
		_ = sqlDB.Close()
	}

	db, err := gorm.Open(postgres.Open(dsn(testDBName)), silent)
	if err != nil {
		return nil, fmt.Errorf("连不上测试库: %w", err)
	}
	if err := db.AutoMigrate(&model.Task{}, &model.User{}); err != nil {
		return nil, fmt.Errorf("测试库建表失败: %w", err)
	}
	return db, nil
}

// setup 准备一次干净的测试环境：清空数据表 + 组装完整路由。
//
// 限流默认 10 rps / burst 20，测试里连续发几十个请求会被自己拦掉，
// 所以这里放宽；需要验证限流本身的测试自己传小值。
func setup(t *testing.T) *gin.Engine {
	t.Helper()
	if testDB == nil {
		t.Skipf("跳过集成测试：%v（提示：docker compose up -d db redis）", dbSkipError)
	}

	// RESTART IDENTITY 让自增 ID 从 1 重新开始，测试之间互不影响
	if err := testDB.Exec("truncate table tasks, users restart identity cascade").Error; err != nil {
		t.Fatalf("清空测试表失败: %v", err)
	}

	return buildRouter(router.Options{RateLimitRPS: 10000, RateLimitBurst: 10000})
}

func buildRouter(opts router.Options) *gin.Engine {
	taskSvc := service.NewTaskService(repository.NewTaskRepository(testDB))
	userSvc := service.NewUserService(repository.NewUserRepository(testDB))

	hub := ws.NewHub()
	go hub.Run()

	return router.RegisterRouter(handler.New(taskSvc, userSvc, hub), opts)
}

// apiResponse 对应 common.Response 的 JSON 结构。
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// request 发一个请求并解析响应。token 为空则不带 Authorization 头。
func request(t *testing.T, r *gin.Engine, method, path, token string, body any) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var res apiResponse
	if w.Body.Len() > 0 {
		// 有些分支（如 AuthToken 里的 AbortWithStatus）不返回 JSON body，解析失败不算错
		_ = json.Unmarshal(w.Body.Bytes(), &res)
	}
	return w, res
}

// registerAndLogin 注册一个用户并登录，返回 token 和用户 ID。
func registerAndLogin(t *testing.T, r *gin.Engine, email string) (string, uint64) {
	t.Helper()

	credentials := map[string]string{"email": email, "password": "test1234"}

	w, _ := request(t, r, http.MethodPost, "/api/v1/auth/register", "", credentials)
	if w.Code != http.StatusOK {
		t.Fatalf("注册失败: status=%d body=%s", w.Code, w.Body.String())
	}

	w, res := request(t, r, http.MethodPost, "/api/v1/auth/login", "", credentials)
	if w.Code != http.StatusOK {
		t.Fatalf("登录失败: status=%d body=%s", w.Code, w.Body.String())
	}

	var login struct {
		Token string `json:"token"`
		User  struct {
			ID uint64 `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(res.Data, &login); err != nil {
		t.Fatalf("解析登录响应失败: %v, body=%s", err, w.Body.String())
	}
	if login.Token == "" {
		t.Fatal("登录成功但没拿到 token")
	}
	return login.Token, login.User.ID
}

// createTask 创建一个任务并返回它的 ID。
func createTask(t *testing.T, r *gin.Engine, token, title string) uint {
	t.Helper()

	w, res := request(t, r, http.MethodPost, "/api/v1/tasks", token, map[string]any{
		"title":       title,
		"description": "集成测试创建",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("创建任务失败: status=%d body=%s", w.Code, w.Body.String())
	}

	var task model.Task
	if err := json.Unmarshal(res.Data, &task); err != nil {
		t.Fatalf("解析任务响应失败: %v", err)
	}
	if task.ID == 0 {
		t.Fatal("创建成功但 ID 为 0")
	}
	return task.ID
}

func TestPing(t *testing.T) {
	r := setup(t)

	w, res := request(t, r, http.MethodGet, "/ping", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", w.Code)
	}
	if res.Code != 0 {
		t.Fatalf("期望业务码 0，实际 %d", res.Code)
	}
}

func TestAuthFlow(t *testing.T) {
	r := setup(t)

	credentials := map[string]string{"email": "flow@test.com", "password": "test1234"}

	t.Run("注册成功", func(t *testing.T) {
		w, res := request(t, r, http.MethodPost, "/api/v1/auth/register", "", credentials)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
		}

		var user model.User
		if err := json.Unmarshal(res.Data, &user); err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if user.Email != "flow@test.com" {
			t.Fatalf("邮箱不对: %s", user.Email)
		}
		// 密码字段打了 json:"-"，绝不能出现在响应里
		if strings.Contains(w.Body.String(), "password") {
			t.Fatalf("响应里泄漏了密码字段: %s", w.Body.String())
		}
	})

	t.Run("重复邮箱注册被拒", func(t *testing.T) {
		w, res := request(t, r, http.MethodPost, "/api/v1/auth/register", "", credentials)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d body=%s", w.Code, w.Body.String())
		}
		if res.Message != "该邮箱已注册" {
			t.Fatalf("错误提示不对: %s", res.Message)
		}
	})

	t.Run("登录成功拿到 token", func(t *testing.T) {
		w, res := request(t, r, http.MethodPost, "/api/v1/auth/login", "", credentials)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", w.Code)
		}

		var login struct {
			Token string `json:"token"`
		}
		_ = json.Unmarshal(res.Data, &login)
		if login.Token == "" {
			t.Fatal("没拿到 token")
		}
	})

	t.Run("密码错误返回 401", func(t *testing.T) {
		w, _ := request(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": "flow@test.com", "password": "wrongpassword",
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("注册参数非法返回 400", func(t *testing.T) {
		cases := []struct {
			name string
			body map[string]string
		}{
			{"邮箱格式不对", map[string]string{"email": "not-an-email", "password": "test1234"}},
			{"密码不足 8 位", map[string]string{"email": "short@test.com", "password": "123"}},
			{"缺少邮箱字段", map[string]string{"password": "test1234"}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w, _ := request(t, r, http.MethodPost, "/api/v1/auth/register", "", tc.body)
				if w.Code != http.StatusBadRequest {
					t.Fatalf("期望 400，实际 %d body=%s", w.Code, w.Body.String())
				}
			})
		}
	})

	t.Run("邮箱不存在返回 401", func(t *testing.T) {
		w, _ := request(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": "nobody@test.com", "password": "test1234",
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d", w.Code)
		}
	})
}

func TestTaskCRUD(t *testing.T) {
	r := setup(t)
	token, _ := registerAndLogin(t, r, "crud@test.com")

	var taskID uint

	t.Run("创建", func(t *testing.T) {
		taskID = createTask(t, r, token, "写集成测试")
	})

	t.Run("按 ID 查询", func(t *testing.T) {
		w, res := request(t, r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", taskID), token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
		}

		var task model.Task
		_ = json.Unmarshal(res.Data, &task)
		if task.Title != "写集成测试" {
			t.Fatalf("标题不对: %s", task.Title)
		}
	})

	t.Run("列表能查到", func(t *testing.T) {
		w, res := request(t, r, http.MethodGet, "/api/v1/tasks?pageNo=1&pageSize=10", token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", w.Code)
		}

		var list struct {
			List  []model.Task `json:"list"`
			Total int64        `json:"total"`
		}
		_ = json.Unmarshal(res.Data, &list)
		if list.Total != 1 || len(list.List) != 1 {
			t.Fatalf("期望 1 条，实际 total=%d len=%d", list.Total, len(list.List))
		}
	})

	t.Run("更新", func(t *testing.T) {
		w, res := request(t, r, http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", taskID), token, map[string]any{
			"title":       "改过的标题",
			"description": "改过的描述",
			"status":      true,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
		}

		var task model.Task
		_ = json.Unmarshal(res.Data, &task)
		if task.Title != "改过的标题" || !task.Status {
			t.Fatalf("更新没生效: %+v", task)
		}
	})

	t.Run("更新后重新查询确认落库", func(t *testing.T) {
		w, res := request(t, r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", taskID), token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", w.Code)
		}

		var task model.Task
		_ = json.Unmarshal(res.Data, &task)
		if task.Title != "改过的标题" || !task.Status {
			t.Fatalf("改动没落到数据库: %+v", task)
		}
	})

	t.Run("删除", func(t *testing.T) {
		w, _ := request(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", taskID), token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("删除后列表为空", func(t *testing.T) {
		w, res := request(t, r, http.MethodGet, "/api/v1/tasks?pageNo=1&pageSize=10", token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", w.Code)
		}

		var list struct {
			Total int64 `json:"total"`
		}
		_ = json.Unmarshal(res.Data, &list)
		if list.Total != 0 {
			t.Fatalf("删除后 total 应为 0，实际 %d", list.Total)
		}
	})

	t.Run("查询不存在的任务返回 404", func(t *testing.T) {
		w, res := request(t, r, http.MethodGet, "/api/v1/tasks/99999999", token, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d body=%s", w.Code, w.Body.String())
		}
		if res.Message != "任务不存在" {
			t.Fatalf("错误提示不对: %s", res.Message)
		}
	})

	t.Run("更新不存在的任务返回 404", func(t *testing.T) {
		w, _ := request(t, r, http.MethodPut, "/api/v1/tasks/99999999", token, map[string]any{
			"title": "改不存在的", "description": "", "status": false,
		})
		if w.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("重复删除返回 404", func(t *testing.T) {
		w, _ := request(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", taskID), token, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d body=%s", w.Code, w.Body.String())
		}
	})
}

// TestTaskUserIsolation 是整套测试里最重要的一个：
// 验证用户之间的数据隔离。repository 每个方法都带 user_id 过滤，
// 哪天有人图省事去掉其中一个，这里会立刻红。
func TestTaskUserIsolation(t *testing.T) {
	r := setup(t)

	tokenA, _ := registerAndLogin(t, r, "user-a@test.com")
	tokenB, _ := registerAndLogin(t, r, "user-b@test.com")

	taskID := createTask(t, r, tokenA, "A 的私密任务")

	t.Run("B 的列表里看不到 A 的任务", func(t *testing.T) {
		w, res := request(t, r, http.MethodGet, "/api/v1/tasks?pageNo=1&pageSize=10", tokenB, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", w.Code)
		}

		var list struct {
			List  []model.Task `json:"list"`
			Total int64        `json:"total"`
		}
		_ = json.Unmarshal(res.Data, &list)
		if list.Total != 0 || len(list.List) != 0 {
			t.Fatalf("数据隔离失效：B 看到了 A 的任务 total=%d len=%d", list.Total, len(list.List))
		}
	})

	t.Run("B 按 ID 查不到 A 的任务", func(t *testing.T) {
		w, _ := request(t, r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", taskID), tokenB, nil)
		// 返回 404 而不是 403：对 B 来说这个任务就是「不存在」，
		// 用 403 反而会泄漏「这个 ID 确实存在，只是不属于你」
		if w.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d body=%s", w.Code, w.Body.String())
		}
		// 响应体里不能出现 A 的任务内容
		if strings.Contains(w.Body.String(), "A 的私密任务") {
			t.Fatalf("响应泄漏了 A 的任务内容: %s", w.Body.String())
		}
	})

	t.Run("B 改不了 A 的任务", func(t *testing.T) {
		w, _ := request(t, r, http.MethodPut, fmt.Sprintf("/api/v1/tasks/%d", taskID), tokenB, map[string]any{
			"title": "B 篡改的标题", "description": "", "status": true,
		})
		if w.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d body=%s", w.Code, w.Body.String())
		}

		// 再用 A 的身份确认内容没被动过
		w, res := request(t, r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", taskID), tokenA, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("A 自己查不到了: %d", w.Code)
		}

		var task model.Task
		_ = json.Unmarshal(res.Data, &task)
		if task.Title != "A 的私密任务" {
			t.Fatalf("A 的任务被 B 改了: %s", task.Title)
		}
	})

	t.Run("B 删不掉 A 的任务", func(t *testing.T) {
		w, _ := request(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/tasks/%d", taskID), tokenB, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d body=%s", w.Code, w.Body.String())
		}

		// 确认 A 的任务还在
		w, _ = request(t, r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", taskID), tokenA, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("A 的任务被 B 删掉了: %d", w.Code)
		}
	})
}

func TestAuthRequired(t *testing.T) {
	r := setup(t)

	// 所有需要登录的接口，都必须挡住没带 token 的请求
	protected := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/tasks"},
		{http.MethodPost, "/api/v1/tasks"},
		{http.MethodGet, "/api/v1/tasks/1"},
		{http.MethodPut, "/api/v1/tasks/1"},
		{http.MethodDelete, "/api/v1/tasks/1"},
		{http.MethodGet, "/api/v1/userInfo"},
		{http.MethodPost, "/api/v1/avatar"},
	}

	for _, route := range protected {
		t.Run("无 token "+route.method+" "+route.path, func(t *testing.T) {
			w, _ := request(t, r, route.method, route.path, "", nil)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("期望 401，实际 %d", w.Code)
			}
		})
	}

	t.Run("token 乱填返回 401", func(t *testing.T) {
		w, _ := request(t, r, http.MethodGet, "/api/v1/tasks", "not-a-real-token", nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d", w.Code)
		}
	})

	t.Run("Authorization 头格式不对返回 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
		req.Header.Set("Authorization", "Token abc") // 不是 Bearer 开头
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d", w.Code)
		}
	})

	t.Run("用别的密钥签的 token 返回 401", func(t *testing.T) {
		// 换一个密钥签一个结构完全合法的 token，验证签名校验真的生效
		common.InitJWT("another-secret", 3)
		forged, err := common.GenerateToken(999)
		common.InitJWT("integration-test-secret", 3) // 立刻还原，别影响后面的测试
		if err != nil {
			t.Fatalf("生成伪造 token 失败: %v", err)
		}

		w, _ := request(t, r, http.MethodGet, "/api/v1/tasks", forged, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("期望 401，实际 %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestTaskValidation(t *testing.T) {
	r := setup(t)
	token, _ := registerAndLogin(t, r, "validate@test.com")

	cases := []struct {
		name string
		body map[string]any
	}{
		{"标题为空", map[string]any{"title": "", "description": "x"}},
		{"标题只有 1 个字（min=2）", map[string]any{"title": "a", "description": "x"}},
		{"标题超过 20 个字（max=20）", map[string]any{"title": strings.Repeat("长", 21), "description": "x"}},
		{"描述超过 132 个字", map[string]any{"title": "正常标题", "description": strings.Repeat("x", 133)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := request(t, r, http.MethodPost, "/api/v1/tasks", token, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("期望 400，实际 %d body=%s", w.Code, w.Body.String())
			}
		})
	}

	t.Run("ID 不是数字返回 400", func(t *testing.T) {
		w, _ := request(t, r, http.MethodGet, "/api/v1/tasks/abc", token, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestPagination(t *testing.T) {
	r := setup(t)
	token, _ := registerAndLogin(t, r, "page@test.com")

	const total = 25
	for i := 1; i <= total; i++ {
		createTask(t, r, token, fmt.Sprintf("任务 %d", i))
	}

	cases := []struct {
		name      string
		query     string
		wantLen   int
		wantTotal int64
	}{
		{"第 1 页 10 条", "pageNo=1&pageSize=10", 10, total},
		{"第 3 页只剩 5 条", "pageNo=3&pageSize=10", 5, total},
		{"第 4 页没有数据", "pageNo=4&pageSize=10", 0, total},
		{"pageSize 超过 100 时回落到默认 20", "pageNo=1&pageSize=500", 20, total},
		{"pageSize 为负数时回落到默认 20", "pageNo=1&pageSize=-5", 20, total},
		{"pageSize 为 0 时回落到默认 20", "pageNo=1&pageSize=0", 20, total},
		{"参数缺失时用默认值", "", 20, total},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, res := request(t, r, http.MethodGet, "/api/v1/tasks?"+tc.query, token, nil)
			if w.Code != http.StatusOK {
				t.Fatalf("期望 200，实际 %d", w.Code)
			}

			var list struct {
				List  []model.Task `json:"list"`
				Total int64        `json:"total"`
			}
			_ = json.Unmarshal(res.Data, &list)

			if len(list.List) != tc.wantLen {
				t.Fatalf("期望 %d 条，实际 %d 条", tc.wantLen, len(list.List))
			}
			if list.Total != tc.wantTotal {
				t.Fatalf("期望 total=%d，实际 %d", tc.wantTotal, list.Total)
			}
		})
	}

	t.Run("按 id 倒序返回", func(t *testing.T) {
		w, res := request(t, r, http.MethodGet, "/api/v1/tasks?pageNo=1&pageSize=5", token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际 %d", w.Code)
		}

		var list struct {
			List []model.Task `json:"list"`
		}
		_ = json.Unmarshal(res.Data, &list)

		for i := 1; i < len(list.List); i++ {
			if list.List[i-1].ID <= list.List[i].ID {
				t.Fatalf("排序不是倒序: %d 出现在 %d 前面", list.List[i-1].ID, list.List[i].ID)
			}
		}
	})
}

func TestRateLimit(t *testing.T) {
	if testDB == nil {
		t.Skipf("跳过集成测试：%v", dbSkipError)
	}

	// 用极紧的限流参数：每秒 1 个、桶容量 1，第二个请求必被拦
	r := buildRouter(router.Options{RateLimitRPS: 1, RateLimitBurst: 1})

	w, _ := request(t, r, http.MethodGet, "/ping", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("第 1 个请求应该通过，实际 %d", w.Code)
	}

	w, res := request(t, r, http.MethodGet, "/ping", "", nil)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("第 2 个请求应该被限流，实际 %d", w.Code)
	}
	if res.Message != "请求过于频繁" {
		t.Fatalf("限流提示不对: %s", res.Message)
	}
}

// TestRequestIDAndLogging 验证可观测性：请求 ID 的生成、沿用、回写，以及日志里带上它。
func TestRequestIDAndLogging(t *testing.T) {
	r := setup(t)

	t.Run("没带 ID 时自动生成并回写响应头", func(t *testing.T) {
		w, _ := request(t, r, http.MethodGet, "/ping", "", nil)

		id := w.Header().Get(common.HeaderRequestID)
		if id == "" {
			t.Fatal("响应头里没有 X-Request-Id")
		}
		if len(id) != 32 {
			t.Fatalf("自动生成的 ID 应该是 32 位十六进制，实际 %q", id)
		}
	})

	t.Run("两个请求的 ID 不重复", func(t *testing.T) {
		w1, _ := request(t, r, http.MethodGet, "/ping", "", nil)
		w2, _ := request(t, r, http.MethodGet, "/ping", "", nil)

		if w1.Header().Get(common.HeaderRequestID) == w2.Header().Get(common.HeaderRequestID) {
			t.Fatal("两个请求拿到了相同的 request id")
		}
	})

	t.Run("上游带了 ID 就沿用", func(t *testing.T) {
		const upstreamID = "upstream-trace-id-12345"

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set(common.HeaderRequestID, upstreamID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if got := w.Header().Get(common.HeaderRequestID); got != upstreamID {
			t.Fatalf("期望沿用上游 ID %q，实际 %q", upstreamID, got)
		}
	})

	t.Run("访问日志带上 request_id", func(t *testing.T) {
		// 临时把日志接到 buffer 上，断言完再还原
		var buf bytes.Buffer
		original := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
		defer slog.SetDefault(original)

		const traceID = "log-assert-id-67890"
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set(common.HeaderRequestID, traceID)
		r.ServeHTTP(httptest.NewRecorder(), req)

		logged := buf.String()
		if !strings.Contains(logged, traceID) {
			t.Fatalf("访问日志里没有 request_id，日志内容: %s", logged)
		}
		for _, field := range []string{`"route":"/ping"`, `"status":200`, `"duration_ms"`, `"method":"GET"`} {
			if !strings.Contains(logged, field) {
				t.Fatalf("访问日志缺字段 %s，日志内容: %s", field, logged)
			}
		}
	})

	t.Run("错误日志和访问日志共用同一个 request_id", func(t *testing.T) {
		var buf bytes.Buffer
		original := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
		defer slog.SetDefault(original)

		token, _ := registerAndLogin(t, r, "logpair@test.com")

		const traceID = "error-pair-id-13579"
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/999999", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(common.HeaderRequestID, traceID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("期望 404，实际 %d", w.Code)
		}

		// 同一个 ID 应该同时出现在「请求处理失败」和「请求完成」两条日志里，
		// 这样线上才能靠一个 ID 把一次请求的全部日志串起来
		var sawError, sawAccess bool
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if !strings.Contains(line, traceID) {
				continue
			}
			if strings.Contains(line, "请求处理失败") {
				sawError = true
			}
			if strings.Contains(line, "请求完成") {
				sawAccess = true
			}
		}
		if !sawError || !sawAccess {
			t.Fatalf("错误日志和访问日志没能通过同一个 request_id 关联: sawError=%v sawAccess=%v 日志=%s",
				sawError, sawAccess, buf.String())
		}
	})
}

// TestMetricsEndpoint 验证 /metrics 端点和指标标签。
func TestMetricsEndpoint(t *testing.T) {
	r := setup(t)

	// 先产生一些流量，指标才有内容
	token, _ := registerAndLogin(t, r, "metrics@test.com")
	createTask(t, r, token, "指标测试任务")
	request(t, r, http.MethodGet, "/ping", "", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("/metrics 期望 200，实际 %d", w.Code)
	}

	body := w.Body.String()
	for _, want := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"http_requests_in_flight",
		`route="/ping"`,
		`route="/api/v1/tasks"`,
		`status="200"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("/metrics 输出里缺少 %q", want)
		}
	}

	// 关键：路由标签必须是模板而不是真实路径，
	// 否则每个任务 ID 都会生成一条独立的时间序列（标签基数爆炸）
	if strings.Contains(body, `route="/api/v1/tasks/1"`) {
		t.Fatalf("指标里出现了真实路径而不是路由模板，会导致标签基数爆炸: %s", body)
	}
}
