package router

import (
	"go-task-api/internal/common"
	"go-task-api/internal/handler"
	"go-task-api/internal/metrics"
	"go-task-api/internal/middleware"
	"net/url"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"

	// 引入生成出来的文档定义。只为触发它的 init() 把文档注册进 swag，
	// 代码里不直接调用它的任何东西，所以用 _ 空导入。
	_ "go-task-api/docs"
)

// Options 是路由的可调参数。
// 抽出来是为了让集成测试能放宽限流，否则连续发几十个请求会被自己的限流器拦掉。
type Options struct {
	RateLimitRPS   rate.Limit
	RateLimitBurst int
	// EnableSwagger 控制是否挂载 /swagger 文档页。
	// 单独开关是因为文档会把全部接口、参数和鉴权方式公开，
	// 线上环境通常不该对公网开放。
	EnableSwagger bool
}

// DefaultOptions 返回默认参数。
func DefaultOptions() Options {
	return Options{
		RateLimitRPS:   10,
		RateLimitBurst: 20,
		// 本地开发默认开着方便联调；部署前通过 Options 关掉，或改成只对内网开放
		EnableSwagger: true,
	}
}

func RegisterRouter(h *handler.Handler, opts Options) *gin.Engine {
	// 用 gin.New() 而不是 gin.Default()：
	// Default 自带的 Logger 输出的是非结构化文本，和项目的 slog JSON 日志格式打架。
	// 这里换成自己的 AccessLog，日志全部统一成 JSON。
	router := gin.New()

	// 顺序有讲究：
	// 1. RequestID 必须最先，后面所有中间件和 handler 都要用它准备的 logger
	// 2. AccessLog / Metrics 紧跟其后，才能量到包含后面所有环节的完整耗时和最终状态码
	// 3. Recovery 放在它们之内，panic 兜底返回的 500 才会被记录和统计
	router.Use(middleware.RequestID)
	router.Use(middleware.AccessLog)
	router.Use(metrics.Middleware())
	router.Use(recovery())

	// /metrics 在限流中间件之前注册，这样 Prometheus 抓取不会被限流拦掉，
	// 也不需要 CORS（它不是给浏览器用的）。
	router.GET("/metrics", metrics.Handler())

	// Swagger 文档页。同样放在限流之前：文档页会并发加载多个静态资源，
	// 走限流很容易把自己的页面打成 429。
	if opts.EnableSwagger {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			parsed, err := url.Parse(origin)
			if err != nil {
				return false
			}

			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				return false
			}

			host := strings.ToLower(parsed.Hostname())
			return host == "localhost" || host == "127.0.0.1" || host == "::1"
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", common.HeaderRequestID},
		ExposeHeaders:    []string{common.HeaderRequestID},
		AllowCredentials: true,
	}))
	router.Use(middleware.Error)
	router.Use(middleware.RateLimit(opts.RateLimitRPS, opts.RateLimitBurst))

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/register", h.Register)
		v1.POST("/auth/login", h.Login)
		v1.GET("/ws", h.HandleWebSocket)

		// 需要鉴权的接口
		auth := v1.Group("", middleware.AuthToken)
		auth.GET("/userInfo", h.GetUserInfo)
		auth.GET("/tasks", h.List)
		auth.POST("/tasks", h.Create)
		auth.DELETE("/tasks/:id", h.Delete)
		auth.GET("/tasks/:id", h.GetByID)
		auth.PUT("/tasks/:id", h.Update)
		auth.POST("/avatar", h.UploadAvatar)
	}

	router.Static("/uploads", "./uploads")
	router.GET("/ping", handler.GetPing)

	return router
}

// recovery 兜住 handler 里的 panic，用结构化日志记录并返回 500。
// 不用 gin.Recovery()，是为了让 panic 日志也带上 request_id 并走 slog。
func recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		common.LoggerFrom(c).Error("请求处理发生 panic",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"panic", err,
		)
		common.Error(c, 500, "服务内部错误")
		c.Abort()
	})
}
