// DeFi聚合器智能路由服务主程序
// 负责启动智能路由服务，初始化聚合器适配器和缓存
// 提供高性能的并发报价聚合功能
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"defi-aggregator/smart-router/internal/handlers"
	"defi-aggregator/smart-router/internal/services"
	"defi-aggregator/smart-router/internal/types"
	"defi-aggregator/smart-router/pkg/cache"
	"defi-aggregator/smart-router/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Application 智能路由应用程序
type Application struct {
	Config        *types.Config           // 应用配置
	Cache         cache.CacheManager      // 缓存管理器
	RouterService *services.RouterService // 路由服务
	Handler       *handlers.RouterHandler // HTTP处理器
	Server        *http.Server            // HTTP服务器
	Logger        *logrus.Logger          // 日志记录器
}

// main 主函数
func main() {
	// 创建应用程序实例
	app, err := NewApplication()
	if err != nil {
		logrus.Fatalf("创建智能路由应用失败: %v", err)
	}

	// 启动应用程序
	if err := app.Run(); err != nil {
		logrus.Fatalf("运行智能路由应用失败: %v", err)
	}
}

// NewApplication 创建智能路由应用实例
func NewApplication() (*Application, error) {
	// 1. 临时使用环境变量配置，确保0x Protocol正常工作
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	logrus.Info("🔧 临时使用环境变量聚合器配置（避免数据库配置混乱）")

	// 2. 初始化日志记录器
	logger := initLogger(cfg)
	logger.Infof("启动DeFi聚合器智能路由服务 - 环境: %s", cfg.Server.Environment)

	// 3. 初始化缓存管理器
	logger.Info("初始化Redis缓存...")
	cacheManager, err := cache.NewRedisCache(&cfg.Redis, cfg.Cache.PrefixKey, logger)
	if err != nil {
		return nil, fmt.Errorf("缓存初始化失败: %w", err)
	}

	// 4. 初始化智能路由服务
	logger.Info("初始化智能路由服务...")
	routerService := services.NewRouterService(cfg, cacheManager, logger)

	// 5. 初始化HTTP处理器
	logger.Info("初始化HTTP处理器...")
	routerHandler := handlers.NewRouterHandler(routerService, logger)

	// 6. 设置Gin模式
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 7. 创建HTTP路由器
	router := setupRouter(cfg, routerHandler, logger)

	// 8. 创建HTTP服务器
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	return &Application{
		Config:        cfg,
		Cache:         cacheManager,
		RouterService: routerService,
		Handler:       routerHandler,
		Server:        server,
		Logger:        logger,
	}, nil
}

// Run 启动应用程序
func (app *Application) Run() error {
	// 创建用于监听系统信号的通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 在goroutine中启动HTTP服务器
	go func() {
		app.Logger.Infof("智能路由服务启动，监听端口: %s", app.Server.Addr)
		app.Logger.Infof("服务器地址: http://localhost%s", app.Server.Addr)
		app.Logger.Info("API接口:")
		app.Logger.Info("  报价聚合: POST http://localhost:5178/api/v1/quote")
		app.Logger.Info("  健康检查: GET  http://localhost:5178/health")
		app.Logger.Info("  性能指标: GET  http://localhost:5178/api/v1/metrics")

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

	app.Logger.Info("正在关闭缓存连接...")

	// 关闭缓存连接
	if err := app.Cache.Close(); err != nil {
		app.Logger.Errorf("缓存关闭失败: %v", err)
		return err
	}

	app.Logger.Info("智能路由服务已优雅关闭")
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
func setupRouter(cfg *types.Config, handler *handlers.RouterHandler, logger *logrus.Logger) *gin.Engine {
	router := gin.New()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS由API Gateway统一处理，此处不再设置
	// router.Use(func(c *gin.Context) {
	//	c.Header("Access-Control-Allow-Origin", "*")
	//	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	//	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
	//
	//	if c.Request.Method == "OPTIONS" {
	//		c.AbortWithStatus(http.StatusNoContent)
	//		return
	//	}
	//
	//	c.Next()
	// })

	// 健康检查路由
	router.GET(cfg.Monitoring.HealthCheckPath, handler.HealthCheck)

	// API路由组
	v1 := router.Group("/api/v1")
	{
		// 核心聚合接口
		v1.POST("/quote", handler.GetQuote)

		// 监控接口
		if cfg.Monitoring.MetricsEnabled {
			v1.GET("/metrics", handler.GetMetrics)
			v1.GET("/providers/status", handler.GetProviderStatus)
		}
	}

	// 404处理
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "NOT_FOUND",
				Message: "请求的资源不存在",
			},
			Timestamp: time.Now().Unix(),
		})
	})

	return router
}
