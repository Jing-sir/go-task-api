package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-task-api/docs"
	"go-task-api/internal/router"
)

// TestSwaggerDocsCoverAllRoutes 防止「加了接口忘了写注解」。
//
// 文档类工作最容易腐坏：第一次写得很全，后面加接口没人补，
// 半年后文档就只剩误导作用。这个测试把「补注解」变成硬约束——
// 漏一个接口，CI 直接红。
func TestSwaggerDocsCoverAllRoutes(t *testing.T) {
	if testDB == nil {
		t.Skipf("跳过集成测试：%v", dbSkipError)
	}

	r := buildRouter(router.DefaultOptions())

	// 解析生成出来的文档，收集已记录的「方法 + 路径」
	var spec struct {
		BasePath string                    `json:"basePath"`
		Paths    map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal([]byte(docs.SwaggerInfo.ReadDoc()), &spec); err != nil {
		t.Fatalf("解析 swagger 文档失败: %v", err)
	}

	documented := make(map[string]bool)
	for path, ops := range spec.Paths {
		for method := range ops {
			documented[strings.ToUpper(method)+" "+path] = true
		}
	}

	// 这些不是业务接口，不需要写进 API 文档
	skipPaths := map[string]bool{
		"/metrics":           true,
		"/swagger/*any":      true,
		"/uploads/*filepath": true,
	}

	for _, route := range r.Routes() {
		if skipPaths[route.Path] {
			continue
		}
		// HEAD 是 gin.Static 自动注册的，不是手写接口
		if route.Method == http.MethodHead {
			continue
		}

		// gin 的路径参数写 :id，swagger 写 {id}，比较前先统一
		swaggerPath := convertGinPathToSwagger(route.Path)

		if !documented[route.Method+" "+swaggerPath] {
			t.Errorf("接口 %s %s 没有 Swagger 注解。\n"+
				"请在对应 handler 上补 @Summary/@Router 等注解，然后重新执行：\n"+
				"  swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal",
				route.Method, route.Path)
		}
	}
}

// TestSwaggerDocsHaveNoStaleRoutes 反方向检查：文档里不许有已经删掉的接口。
func TestSwaggerDocsHaveNoStaleRoutes(t *testing.T) {
	if testDB == nil {
		t.Skipf("跳过集成测试：%v", dbSkipError)
	}

	r := buildRouter(router.DefaultOptions())

	registered := make(map[string]bool)
	for _, route := range r.Routes() {
		registered[route.Method+" "+convertGinPathToSwagger(route.Path)] = true
	}

	var spec struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal([]byte(docs.SwaggerInfo.ReadDoc()), &spec); err != nil {
		t.Fatalf("解析 swagger 文档失败: %v", err)
	}

	for path, ops := range spec.Paths {
		for method := range ops {
			key := strings.ToUpper(method) + " " + path
			if !registered[key] {
				t.Errorf("文档里的 %s 在路由表里不存在，是不是接口删了文档没更新？", key)
			}
		}
	}
}

func convertGinPathToSwagger(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			segments[i] = "{" + strings.TrimPrefix(seg, ":") + "}"
		}
	}
	return strings.Join(segments, "/")
}

// TestSwaggerEndpoint 验证文档页能打开，以及开关真的能关掉它。
func TestSwaggerEndpoint(t *testing.T) {
	if testDB == nil {
		t.Skipf("跳过集成测试：%v", dbSkipError)
	}

	t.Run("开关打开时文档页可访问", func(t *testing.T) {
		r := buildRouter(router.Options{RateLimitRPS: 10000, RateLimitBurst: 10000, EnableSwagger: true})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("/swagger/index.html 期望 200，实际 %d", w.Code)
		}

		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("/swagger/doc.json 期望 200，实际 %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "go-task-api") {
			t.Fatalf("doc.json 内容不对: %s", w.Body.String()[:200])
		}
	})

	t.Run("开关关闭时文档页不存在", func(t *testing.T) {
		r := buildRouter(router.Options{RateLimitRPS: 10000, RateLimitBurst: 10000, EnableSwagger: false})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("关掉开关后应该 404，实际 %d", w.Code)
		}
	})
}

// TestSwaggerSpecQuality 检查文档本身的质量，避免生成出一堆空壳。
func TestSwaggerSpecQuality(t *testing.T) {
	if testDB == nil {
		t.Skipf("跳过集成测试：%v", dbSkipError)
	}

	var spec struct {
		Info struct {
			Title       string `json:"title"`
			Version     string `json:"version"`
			Description string `json:"description"`
		} `json:"info"`
		SecurityDefinitions map[string]any `json:"securityDefinitions"`
		Paths               map[string]map[string]struct {
			Summary   string           `json:"summary"`
			Tags      []string         `json:"tags"`
			Responses map[string]any   `json:"responses"`
			Security  []map[string]any `json:"security"`
		} `json:"paths"`
	}
	if err := json.Unmarshal([]byte(docs.SwaggerInfo.ReadDoc()), &spec); err != nil {
		t.Fatalf("解析 swagger 文档失败: %v", err)
	}

	if spec.Info.Title == "" || spec.Info.Version == "" || spec.Info.Description == "" {
		t.Fatalf("文档缺少标题/版本/描述: %+v", spec.Info)
	}
	if _, ok := spec.SecurityDefinitions["BearerAuth"]; !ok {
		t.Fatal("文档里没定义 BearerAuth 鉴权方式，Swagger UI 上就没法填 token 调试")
	}

	// 需要登录的接口必须标 @Security，否则文档读者不知道要带 token
	needAuth := map[string]bool{
		"/api/v1/tasks":      true,
		"/api/v1/tasks/{id}": true,
		"/api/v1/userInfo":   true,
		"/api/v1/avatar":     true,
	}

	for path, ops := range spec.Paths {
		for method, op := range ops {
			label := strings.ToUpper(method) + " " + path

			if op.Summary == "" {
				t.Errorf("%s 缺少 @Summary", label)
			}
			if len(op.Tags) == 0 {
				t.Errorf("%s 缺少 @Tags，文档页里会散落在未分类区", label)
			}
			if len(op.Responses) == 0 {
				t.Errorf("%s 没写任何响应", label)
			}
			if needAuth[path] && len(op.Security) == 0 {
				t.Errorf("%s 需要登录但没标 @Security BearerAuth", label)
			}
		}
	}
}
