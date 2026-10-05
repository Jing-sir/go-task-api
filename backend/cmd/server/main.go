package main

import (
	"context"
	"go-task-api/internal/cache"
	"go-task-api/internal/common"
	"go-task-api/internal/config"
	"go-task-api/internal/db"
	"go-task-api/internal/handler"
	"go-task-api/internal/metrics"
	"go-task-api/internal/model"
	"go-task-api/internal/repository"
	"go-task-api/internal/router"
	"go-task-api/internal/service"
	"go-task-api/internal/ws"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Swagger 文档的总体信息。swag 从这里读取标题、版本、服务地址和鉴权方式。
//
//	@title			go-task-api
//	@version		1.0
//	@description	任务管理服务。Gin + GORM + PostgreSQL + Redis + WebSocket。
//	@description	除注册、登录、健康检查外，所有接口都需要 JWT 鉴权。
//
//	@host		localhost:8080
//	@BasePath	/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				先调 /auth/login 拿 token，再填入 "Bearer {token}"（Bearer 和 token 之间有一个空格）
func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg := config.Load()

	// 配置有致命问题就别启动，免得带着坏配置跑起来、事后才发现
	if err := cfg.Validate(); err != nil {
		log.Error("配置校验失败", "error", err)
		os.Exit(1)
	}

	common.InitJWT(cfg.JWTSecret, cfg.JWTExpireHours)

	cache.NewRedisClient(cfg.RedisAddr)

	log.Info("准备连接数据库")

	// 连接数据库
	dbBase, err := db.DBRegister(cfg.DSN())

	if err != nil {
		log.Error("数据库连接失败:", "error", err)
		os.Exit(1)
	}

	err = dbBase.AutoMigrate(&model.Task{}, &model.User{})
	if err != nil {
		log.Error("数据库建表失败:", "error", err)
		os.Exit(1)
	}

	// 把连接池状态暴露成 Prometheus 指标（已开连接、使用中、空闲、等待次数等）
	if sqlDB, err := dbBase.DB(); err == nil {
		if err := metrics.RegisterDBStats(sqlDB, "go_task_db"); err != nil {
			log.Warn("注册数据库连接池指标失败", "error", err)
		}
	}

	taskRepo := repository.NewTaskRepository(dbBase)
	taskSvc := service.NewTaskService(taskRepo)

	userRepo := repository.NewUserRepository(dbBase)
	userSvc := service.NewUserService(userRepo)

	hub := ws.NewHub()
	go hub.Run()

	h := handler.New(taskSvc, userSvc, hub)

	// 注册路由
	route := router.RegisterRouter(h, router.DefaultOptions())

	// 1. 把 Gin 引擎塞进 http.Server（Address 端口、Handler 路由）
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: route,
	}

	// 2. 监听退出信号：Ctrl+C（os.Interrupt）和 kill（SIGTERM）
	//    收到信号时，ctx 会自动取消
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. 启动服务。ListenAndServe 会一直阻塞，所以放 goroutine，
	//    main 才能继续往下走到第 4 步等信号
	go func() {
		log.Info("服务启动", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("服务启动失败", "error", err)
		}
	}()

	// 4. main 主流程卡在这里，直到收到退出信号
	<-ctx.Done()
	log.Info("收到退出信号，开始优雅关闭...")

	// 5. 给 5 秒让正在处理的请求跑完，跑不完强制关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("优雅关闭失败", "error", err)
	}

	log.Info("服务已优雅退出")
}
