// DeFi聚合器业务逻辑服务主程序入口
// 负责初始化所有组件、启动HTTP服务器、处理优雅关闭
// 遵循企业级应用程序结构，确保服务的可靠性和可维护性
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"defi-aggregator/business-logic/internal/controllers"
	"defi-aggregator/business-logic/internal/repository"
	"defi-aggregator/business-logic/internal/services"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"
	"defi-aggregator/business-logic/pkg/database"
	"defi-aggregator/business-logic/pkg/middleware"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Application 应用程序结构体
// 封装所有应用程序组件，便于统一管理和测试
type Application struct {
	Config   *config.Config     // 应用配置
	Database *database.Database // 数据库连接
	Router   *gin.Engine        // HTTP路由器
	Server   *http.Server       // HTTP服务器
	Logger   *logrus.Logger     // 日志记录器

	// 业务组件
	Repositories *repository.Repositories // 数据访问层
	Services     *services.Services       // 业务逻辑层
	Controllers  *controllers.Controllers // 控制器层
}

// main 主函数
// 程序入口点，负责初始化应用程序并启动服务
func main() {
	// 创建应用程序实例
	app, err := NewApplication()
	if err != nil {
		logrus.Fatalf("创建应用程序失败: %v", err)
	}

	// 启动应用程序
	if err := app.Run(); err != nil {
		logrus.Fatalf("运行应用程序失败: %v", err)
	}
}

// NewApplication 创建新的应用程序实例
// 初始化所有必要的组件，按依赖关系顺序进行初始化
func NewApplication() (*Application, error) {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 2. 初始化日志记录器
	logger := initLogger(cfg)
	logger.Infof("启动DeFi聚合器业务逻辑服务 - 环境: %s", cfg.Server.Environment)

	// 3. 初始化数据库连接
	logger.Info("初始化数据库连接...")
	db, err := database.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("数据库初始化失败: %w", err)
	}

	// 4. 数据库初始化检查
	// 注意：现在使用Database First架构，不再使用GORM AutoMigrate
	// 数据库结构由 database/migrations/001_initial_schema.sql 管理
	// 种子数据由 database/seed_data.sql 管理
	logger.Info("数据库连接已建立，请确保已执行数据库迁移脚本")
	if cfg.IsDevelopment() {
		logger.Info("如需初始化数据库，请运行：")
		logger.Info("  1. psql -f database/migrations/001_initial_schema.sql")
		logger.Info("  2. psql -f database/seed_data.sql")
	}

	// 基本的数据库健康检查
	if err := db.HealthCheck(); err != nil {
		return nil, fmt.Errorf("数据库健康检查失败: %w", err)
	}

	// 5. 初始化数据访问层
	logger.Info("初始化数据访问层...")
	repos := repository.New(db.GetDB())

	// 6. 初始化业务逻辑层
	logger.Info("初始化业务逻辑层...")
	srvs := services.New(repos, cfg, logger)

	// 7. 初始化控制器层
	logger.Info("初始化控制器层...")
	ctrlrs := controllers.New(srvs, cfg, logger)

	// 8. 设置Gin模式
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 9. 创建HTTP路由器
	router := setupRouter(cfg, ctrlrs, logger)

	// 10. 创建HTTP服务器
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,  // 读取超时
		WriteTimeout:   30 * time.Second,  // 写入超时
		IdleTimeout:    120 * time.Second, // 空闲超时
		MaxHeaderBytes: 1 << 20,           // 最大请求头大小 1MB
	}

	return &Application{
		Config:       cfg,
		Database:     db,
		Router:       router,
		Server:       server,
		Logger:       logger,
		Repositories: repos,
		Services:     srvs,
		Controllers:  ctrlrs,
	}, nil
}

// Run 启动应用程序
// 启动HTTP服务器并处理优雅关闭
func (app *Application) Run() error {
	// 创建用于监听系统信号的通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 在goroutine中启动HTTP服务器
	go func() {
		app.Logger.Infof("HTTP服务器启动，监听端口: %s", app.Server.Addr)
		app.Logger.Infof("服务器地址: http://localhost%s", app.Server.Addr)

		// 启动服务器
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
// 按照正确的顺序关闭各个组件，确保数据完整性
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

	app.Logger.Info("正在关闭数据库连接...")

	// 关闭数据库连接
	if err := app.Database.Close(); err != nil {
		app.Logger.Errorf("数据库关闭失败: %v", err)
		return err
	}

	app.Logger.Info("应用程序已优雅关闭")
	return nil
}

// initLogger 初始化日志记录器
// 根据配置设置日志格式、级别和输出目标
func initLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Server.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// 设置日志格式
	if cfg.Monitoring.LogFormat == "json" {
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
			ForceColors:     cfg.IsDevelopment(),
		})
	}

	// 设置输出目标
	switch cfg.Monitoring.LogOutput {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		logger.SetOutput(os.Stdout)
	}

	return logger
}

// setupRouter 设置HTTP路由器
// 配置中间件、路由和错误处理
func setupRouter(cfg *config.Config, ctrlrs *controllers.Controllers, logger *logrus.Logger) *gin.Engine {
	// 创建Gin引擎
	router := gin.New()

	// 添加全局中间件
	router.Use(middleware.Logger(logger))   // 请求日志中间件
	router.Use(middleware.Recovery(logger)) // 恐慌恢复中间件
	// router.Use(middleware.CORS(cfg))        // CORS由API Gateway统一处理
	router.Use(middleware.RequestID())    // 请求ID中间件
	router.Use(middleware.RateLimit(cfg)) // 限流中间件
	router.Use(middleware.Security())     // 安全头中间件

	// 健康检查路由
	router.GET(cfg.Monitoring.HealthCheckPath, ctrlrs.Health.HealthCheck)

	// 指标路由（如果启用）
	if cfg.Monitoring.MetricsEnabled {
		router.GET(cfg.Monitoring.MetricsPath, ctrlrs.Health.Metrics)
	}

	// API路由组
	v1 := router.Group("/api/v1")
	{
		// 认证相关路由（无需JWT）
		auth := v1.Group("/auth")
		{
			auth.POST("/nonce", ctrlrs.Auth.GetNonce)       // 获取登录随机数
			auth.POST("/login", ctrlrs.Auth.Login)          // 钱包登录
			auth.POST("/refresh", ctrlrs.Auth.RefreshToken) // 刷新令牌
		}

		// 需要认证的路由
		protected := v1.Group("")
		protected.Use(middleware.JWT(cfg)) // JWT认证中间件
		{
			// 认证用户操作
			protected.POST("/auth/logout", ctrlrs.Auth.Logout) // 用户登出

			// 用户相关路由
			users := protected.Group("/users")
			{
				users.GET("/profile", ctrlrs.User.GetProfile)                  // 获取用户资料
				users.PUT("/profile", ctrlrs.User.UpdateProfile)               // 更新用户资料
				users.GET("/preferences", ctrlrs.User.GetPreferences)          // 获取偏好设置
				users.PUT("/preferences", ctrlrs.User.UpdatePreferences)       // 更新偏好设置
				users.POST("/preferences/reset", ctrlrs.User.ResetPreferences) // 重置偏好设置
				users.GET("/stats", ctrlrs.User.GetStats)                      // 获取用户统计
			}

			// 交易历史路由
			transactions := protected.Group("/transactions")
			{
				transactions.GET("", ctrlrs.Transaction.GetTransactions)    // 获取交易列表
				transactions.GET("/:id", ctrlrs.Transaction.GetTransaction) // 获取交易详情
			}
		}

		// 公开路由（无需认证）
		public := v1.Group("")
		{
			// 代币相关路由
			tokens := public.Group("/tokens")
			{
				tokens.GET("", ctrlrs.Token.GetTokens)    // 获取代币列表
				tokens.GET("/:id", ctrlrs.Token.GetToken) // 获取代币详情
			}

			// 链相关路由
			chains := public.Group("/chains")
			{
				chains.GET("", ctrlrs.Chain.GetChains)    // 获取链列表
				chains.GET("/:id", ctrlrs.Chain.GetChain) // 获取链详情
			}

			// 报价相关路由
			quotes := public.Group("/quotes")
			{
				quotes.POST("", ctrlrs.Quote.GetQuote)               // 获取报价
				quotes.GET("/history", ctrlrs.Quote.GetQuoteHistory) // 报价历史
			}

			// 交易相关路由
			swaps := public.Group("/swaps")
			{
				swaps.POST("", ctrlrs.Swap.CreateSwap)           // 创建交易
				swaps.GET("/:txHash", ctrlrs.Swap.GetSwapStatus) // 查询交易状态
			}

			// 统计相关路由
			stats := public.Group("/stats")
			{
				stats.GET("/system", ctrlrs.Stats.GetSystemStats)          // 系统统计
				stats.GET("/aggregators", ctrlrs.Stats.GetAggregatorStats) // 聚合器统计
				stats.GET("/token-pairs", ctrlrs.Stats.GetTokenPairStats)  // 代币对统计
			}
		}
	}

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
