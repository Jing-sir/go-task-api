// Package metrics 收集并暴露 Prometheus 指标。
//
// 回答的是「整体健康度」问题：现在多少 QPS、延迟分布如何、
// 错误率多少、连接池用满了没有。单个请求的排查靠 request_id 日志。
package metrics

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// 请求总数。按 方法 / 路由 / 状态码 分维度，可算 QPS 和错误率。
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP 请求总数",
		},
		[]string{"method", "route", "status"},
	)

	// 请求耗时直方图。Histogram 才能算 P95/P99 分位数，普通计数器只能算平均值，
	// 而平均值会把少数很慢的请求平掉，看不出真实体验。
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP 请求耗时（秒）",
			// 针对本服务的实际量级（毫秒级）选的分桶边界
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "route"},
	)

	// 正在处理中的请求数。突然堆高说明下游（数据库、Redis）卡住了。
	requestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "正在处理中的 HTTP 请求数",
		},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal, requestDuration, requestsInFlight)
}

// RegisterDBStats 把数据库连接池状态注册成指标。
//
// 暴露的是 Day 20 配的那套上限的实时使用情况：
// 已开连接数、使用中、空闲、等待次数、等待总时长。
func RegisterDBStats(sqlDB *sql.DB, dbName string) error {
	return prometheus.Register(collectors.NewDBStatsCollector(sqlDB, dbName))
}

// Middleware 统计每个请求的数量和耗时。
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// /metrics 自己不统计，否则抓取动作会污染被抓取的数据
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		requestsInFlight.Inc()
		defer requestsInFlight.Dec()

		c.Next()

		// 用路由模板而不是真实路径。真实路径会让 /tasks/1、/tasks/2 各占一个时间序列，
		// 数据量随 ID 无限膨胀（标签基数爆炸），是 Prometheus 最常见的事故。
		route := c.FullPath()
		if route == "" {
			route = "undefined"
		}

		method := c.Request.Method
		requestsTotal.WithLabelValues(method, route, strconv.Itoa(c.Writer.Status())).Inc()
		requestDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
	}
}

// Handler 返回 /metrics 端点的处理函数，供 Prometheus 抓取。
func Handler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}
