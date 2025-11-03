// DeFi聚合器API网关主程序
// 企业级API网关，提供统一的API入口、负载均衡、安全防护等功能
// 作为微服务架构的前置网关，协调所有后端服务
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"defi-aggregator/api-gateway/internal/handlers"
	"defi-aggregator/api-gateway/internal/middleware"
	"defi-aggregator/api-gateway/internal/proxy"
	"defi-aggregator/api-gateway/internal/types"
	"defi-aggregator/api-gateway/pkg/balancer"
	"defi-aggregator/api-gateway/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Application API网关应用程序
type Application struct {
	Config       *types.Config            // 网关配置
	LoadBalancer balancer.LoadBalancer    // 负载均衡器
	Proxy        *proxy.ReverseProxy      // 反向代理
	Handler      *handlers.GatewayHandler // 网关处理器
	RateLimiter  *middleware.RateLimiter  // 限流器
	Server       *http.Server             // HTTP服务器
	Logger       *logrus.Logger           // 日志记录器
}

// main 主函数
func main() {
	// 创建应用程序实例
	app, err := NewApplication()
	if err != nil {
		logrus.Fatalf("创建API网关应用失败: %v", err)
	}

	// 启动应用程序
	if err := app.Run(); err != nil {
		logrus.Fatalf("运行API网关应用失败: %v", err)
	}
}

// NewApplication 创建API网关应用实例
// 初始化所有网关组件，包括负载均衡器、代理、中间件等
func NewApplication() (*Application, error) {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 2. 初始化日志记录器
	logger := initLogger(cfg)
	logger.Infof("启动DeFi聚合器API网关 - 环境: %s", cfg.Server.Environment)

	// 3. 初始化负载均衡器
	logger.Info("初始化负载均衡器...")
	lb := balancer.NewRoundRobinBalancer(cfg, logger)

	// 4. 启动健康检查
	if cfg.LoadBalancer.HealthCheck {
		logger.Info("启动健康检查...")
		if err := lb.StartHealthChecks(); err != nil {
			return nil, fmt.Errorf("启动健康检查失败: %w", err)
		}
	}

	// 5. 初始化反向代理
	logger.Info("初始化反向代理...")
	reverseProxy := proxy.NewReverseProxy(cfg, lb, logger)

	// 6. 初始化限流器
	logger.Info("初始化限流器...")
	rateLimiter := middleware.NewRateLimiter(&cfg.RateLimit, logger)

	// 7. 初始化网关处理器
	logger.Info("初始化网关处理器...")
	gatewayHandler := handlers.NewGatewayHandler(cfg, reverseProxy, lb, logger)

	// 8. 设置Gin模式
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 9. 创建HTTP路由器
	router := setupRouter(cfg, gatewayHandler, rateLimiter, logger)

	// 10. 创建HTTP服务器
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	return &Application{
		Config:       cfg,
		LoadBalancer: lb,
		Proxy:        reverseProxy,
		Handler:      gatewayHandler,
		RateLimiter:  rateLimiter,
		Server:       server,
		Logger:       logger,
	}, nil
}

// Run 启动API网关应用
func (app *Application) Run() error {
	// 创建用于监听系统信号的通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 在goroutine中启动HTTP服务器
	go func() {
		app.Logger.Infof("API网关启动，监听端口: %s", app.Server.Addr)
		app.Logger.Infof("网关地址: http://localhost%s", app.Server.Addr)
		app.Logger.Info("路由规则:")
		app.Logger.Info("  业务API: http://localhost:5176/api/v1/*")
		app.Logger.Info("  智能路由: http://localhost:5176/api/v1/router/*")
		app.Logger.Info("  健康检查: http://localhost:5176/health")
		app.Logger.Info("  性能指标: http://localhost:5176/metrics")

		if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Fatalf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	<-quit
	app.Logger.Info("接收到关闭信号，开始优雅关闭...")

	// 执行优雅关闭
	return app.Shutdown()
}

// Shutdown 优雅关闭应用程序
func (app *Application) Shutdown() error {
	// 设置关闭超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	app.Logger.Info("正在关闭HTTP服务器...")

	// 关闭HTTP服务器
	if err := app.Server.Shutdown(ctx); err != nil {
		app.Logger.Errorf("HTTP服务器关闭失败: %v", err)
		return err
	}

	app.Logger.Info("正在停止健康检查...")

	// 停止健康检查
	app.LoadBalancer.StopHealthChecks()

	app.Logger.Info("API网关已优雅关闭")
	return nil
}

// initLogger 初始化日志记录器
func initLogger(cfg *types.Config) *logrus.Logger {
	logger := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Server.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// 设置日志格式
	if cfg.Server.Environment == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	return logger
}

// setupRouter 设置HTTP路由器
// 配置中间件栈和路由规则
func setupRouter(cfg *types.Config, handler *handlers.GatewayHandler, rateLimiter *middleware.RateLimiter, logger *logrus.Logger) *gin.Engine {
	router := gin.New()

	// ========================================
	// 全局中间件栈
	// ========================================

	// 1. 基础中间件
	router.Use(middleware.Recovery(logger))                       // 恐慌恢复
	router.Use(middleware.RequestID())                            // 请求ID
	router.Use(middleware.RequestLogger(logger, &cfg.Monitoring)) // 请求日志

	// 2. 安全中间件
	router.Use(middleware.Security())               // 安全头
	router.Use(middleware.CORS(&cfg.Security.CORS)) // CORS处理

	// 3. 限流中间件
	router.Use(rateLimiter.RateLimit()) // 限流控制

	// ========================================
	// 网关自身路由
	// ========================================

	// 健康检查（网关自身处理）
	router.GET(cfg.Monitoring.HealthCheckPath, handler.HealthCheck)

	// 性能指标（网关自身处理）
	if cfg.Monitoring.MetricsEnabled {
		router.GET(cfg.Monitoring.MetricsPath, handler.GetMetrics)
	}

	// ========================================
	// 代理路由（按优先级排序，避免路径冲突）
	// ========================================

	// 1. 网关管理接口（使用独立路径，避免冲突）
	gateway := router.Group("/gateway")
	{
		gateway.GET("/services/status", handler.GetServiceStatus) // 服务状态
		// gateway.POST("/services/reload", handler.ReloadServices)  // 重载服务配置
		// gateway.POST("/cache/clear", handler.ClearCache)          // 清除缓存
	}

	// 2. 智能路由服务代理（特定前缀）
	router.Any("/api/v1/router/*path", handler.HandleRequest)

	// 3. 业务逻辑服务代理（具体路径，避免通配符冲突）
	// 代币相关路由
	router.Any("/api/v1/tokens", handler.HandleRequest)
	router.Any("/api/v1/tokens/*path", handler.HandleRequest)
	// 链相关路由
	router.Any("/api/v1/chains", handler.HandleRequest)
	router.Any("/api/v1/chains/*path", handler.HandleRequest)
	// 报价相关路由
	router.Any("/api/v1/quotes", handler.HandleRequest)
	router.Any("/api/v1/quotes/*path", handler.HandleRequest)
	// 交易相关路由
	router.Any("/api/v1/swaps", handler.HandleRequest)
	router.Any("/api/v1/swaps/*path", handler.HandleRequest)
	// 用户相关路由
	router.Any("/api/v1/users/*path", handler.HandleRequest)
	// 认证相关路由
	router.Any("/api/v1/auth/*path", handler.HandleRequest)
	// 交易记录相关路由
	router.Any("/api/v1/transactions", handler.HandleRequest)
	router.Any("/api/v1/transactions/*path", handler.HandleRequest)
	// 统计相关路由
	router.Any("/api/v1/stats/*path", handler.HandleRequest)

	// 4. 静态资源路由（特定前缀，避免与/api冲突）
	router.Any("/static/*path", handler.HandleRequest)

	// 5. 根路径（单独处理，不使用通配符）
	router.Any("/", handler.HandleRequest)
	router.Any("/favicon.ico", handler.HandleRequest)
	router.Any("/robots.txt", handler.HandleRequest)

	// ========================================
	// 错误处理
	// ========================================

	// 404处理
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeNotFound,
				Message: "请求的资源不存在",
			},
			Timestamp: time.Now().Unix(),
			RequestID: c.GetString("request_id"),
		})
	})

	// 405处理
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "请求方法不被允许",
			},
			Timestamp: time.Now().Unix(),
			RequestID: c.GetString("request_id"),
		})
	})

	return router
}
