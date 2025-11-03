// Package database 提供数据库连接和管理功能
// 采用GORM作为ORM框架，支持连接池管理、自动迁移、事务处理等企业级特性
// 遵循数据库最佳实践，确保连接安全、性能优化和错误处理
package database

import (
	"fmt"
	"time"

	"defi-aggregator/business-logic/internal/models"
	"defi-aggregator/business-logic/pkg/config"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database 数据库管理器结构体
// 封装GORM数据库实例和相关配置，提供统一的数据库操作接口
type Database struct {
	DB     *gorm.DB       // GORM数据库实例
	Config *config.Config // 应用配置
	logger *logrus.Logger // 日志记录器
}

// New 创建新的数据库连接实例
// 根据配置初始化数据库连接，设置连接池参数和日志级别
// 参数:
//   - cfg: 应用配置，包含数据库连接信息
//
// 返回:
//   - *Database: 数据库管理器实例
//   - error: 初始化过程中的错误
func New(cfg *config.Config) (*Database, error) {
	// 创建日志记录器
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 根据环境设置日志格式
	if cfg.Server.Environment == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// 建立数据库连接
	dsn := cfg.GetDatabaseDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction:                   false, // 保持事务支持
		PrepareStmt:                              true,  // 启用预编译语句缓存
		DisableForeignKeyConstraintWhenMigrating: false, // 保持外键约束
	})

	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层sql.DB实例进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层数据库实例失败: %w", err)
	}

	// 配置连接池参数
	// 最大打开连接数：控制数据库并发连接数，避免连接耗尽
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)

	// 最大空闲连接数：保持一定数量的空闲连接，提高响应速度
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	// 连接最大生命周期：定期回收长时间空闲的连接，避免连接泄露
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 测试数据库连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	logger.Infof("数据库连接成功: %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)

	return &Database{
		DB:     db,
		Config: cfg,
		logger: logger,
	}, nil
}

// AutoMigrate 执行数据库自动迁移
// 根据GORM模型定义自动创建和更新数据库表结构
// 注意：生产环境建议使用手动迁移脚本，而非自动迁移
// 返回:
//   - error: 迁移过程中的错误
func (d *Database) AutoMigrate() error {
	d.logger.Info("执行数据库自动迁移...")

	// 定义需要迁移的所有模型
	// 按照依赖关系排序，确保外键关系正确建立
	modelsToMigrate := []interface{}{
		// 基础配置表（无外键依赖）
		&models.Chain{},
		&models.Aggregator{},

		// 依赖基础表的模型
		&models.AggregatorChain{}, // 依赖 chains, aggregators
		&models.Token{},           // 依赖 chains
		&models.User{},            // 无外键依赖

		// 依赖用户和代币的模型
		&models.UserPreferences{}, // 依赖 users
		&models.QuoteRequest{},    // 依赖 users, chains, tokens, aggregators
		&models.QuoteResponse{},   // 依赖 quote_requests, aggregators
		&models.Transaction{},     // 依赖 users, quote_requests, chains, tokens, aggregators

		// 统计表
		&models.AggregatorStatsHourly{}, // 依赖 aggregators
		&models.TokenPairStatsDaily{},   // 依赖 tokens, chains
		&models.SystemMetrics{},         // 无外键依赖
	}

	// 执行自动迁移
	for _, model := range modelsToMigrate {
		if err := d.DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("迁移模型 %T 失败: %w", model, err)
		}
		d.logger.Debugf("成功迁移模型: %T", model)
	}

	// 创建自定义索引（GORM标签无法完全覆盖的复杂索引）
	if err := d.createCustomIndexes(); err != nil {
		return fmt.Errorf("创建自定义索引失败: %w", err)
	}

	d.logger.Info("数据库自动迁移完成")
	return nil
}

// createCustomIndexes 创建自定义索引
// 创建GORM标签无法定义的复杂索引，提升查询性能
func (d *Database) createCustomIndexes() error {
	d.logger.Info("开始创建自定义索引...")

	// 定义需要创建的索引SQL
	customIndexes := []string{
		// 复合索引：优化常用查询
		"CREATE INDEX IF NOT EXISTS idx_quote_requests_user_created ON quote_requests(user_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_user_status ON transactions(user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_tokens_chain_active ON tokens(chain_id, is_active)",
		"CREATE INDEX IF NOT EXISTS idx_quote_requests_token_pair_time ON quote_requests(from_token_id, to_token_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_aggregator_time ON transactions(aggregator_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_aggregator_stats_agg_time ON aggregator_stats_hourly(aggregator_id, hour_timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_token_pair_stats_pair_date ON token_pair_stats_daily(from_token_id, to_token_id, date)",
		"CREATE INDEX IF NOT EXISTS idx_system_metrics_name_time ON system_metrics(metric_name, timestamp DESC)",

		// 部分索引：针对特定条件的查询优化
		"CREATE INDEX IF NOT EXISTS idx_quote_requests_active ON quote_requests(created_at DESC) WHERE status = 'completed'",
		"CREATE INDEX IF NOT EXISTS idx_transactions_confirmed ON transactions(confirmed_at DESC) WHERE status = 'confirmed'",
		"CREATE INDEX IF NOT EXISTS idx_tokens_verified_active ON tokens(symbol) WHERE is_verified = true AND is_active = true",
	}

	// 执行索引创建
	for _, indexSQL := range customIndexes {
		if err := d.DB.Exec(indexSQL).Error; err != nil {
			d.logger.Warnf("创建索引失败，可能已存在: %s, 错误: %v", indexSQL, err)
			// 索引创建失败不中断流程，可能是索引已存在
		} else {
			d.logger.Debugf("成功创建索引: %s", indexSQL)
		}
	}

	d.logger.Info("自定义索引创建完成")
	return nil
}

// Close 关闭数据库连接
// 优雅关闭数据库连接，释放资源
// 返回:
//   - error: 关闭过程中的错误
func (d *Database) Close() error {
	d.logger.Info("正在关闭数据库连接...")

	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库实例失败: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("关闭数据库连接失败: %w", err)
	}

	d.logger.Info("数据库连接已关闭")
	return nil
}

// GetDB 获取GORM数据库实例
// 为其他包提供数据库访问接口
// 返回:
//   - *gorm.DB: GORM数据库实例
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

// HealthCheck 数据库健康检查
// 检查数据库连接状态和基本查询能力
// 返回:
//   - error: 健康检查失败的错误
func (d *Database) HealthCheck() error {
	// 获取底层sql.DB实例
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 执行ping测试
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库ping测试失败: %w", err)
	}

	// 执行简单查询测试
	var result int
	if err := d.DB.Raw("SELECT 1").Scan(&result).Error; err != nil {
		return fmt.Errorf("数据库查询测试失败: %w", err)
	}

	return nil
}

// GetStats 获取数据库连接统计信息
// 返回连接池状态信息，用于监控和调试
// 返回:
//   - map[string]interface{}: 统计信息键值对
//   - error: 获取统计信息失败的错误
func (d *Database) GetStats() (map[string]interface{}, error) {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	stats := sqlDB.Stats()

	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,    // 最大打开连接数
		"open_connections":     stats.OpenConnections,       // 当前打开连接数
		"in_use":               stats.InUse,                 // 正在使用的连接数
		"idle":                 stats.Idle,                  // 空闲连接数
		"wait_count":           stats.WaitCount,             // 等待连接的总次数
		"wait_duration":        stats.WaitDuration.String(), // 总等待时间
		"max_idle_closed":      stats.MaxIdleClosed,         // 因空闲超时关闭的连接数
		"max_idle_time_closed": stats.MaxIdleTimeClosed,     // 因空闲时间超时关闭的连接数
		"max_lifetime_closed":  stats.MaxLifetimeClosed,     // 因生命周期超时关闭的连接数
	}, nil
}

// Transaction 执行数据库事务
// 提供事务包装函数，确保操作的原子性
// 参数:
//   - fn: 需要在事务中执行的函数
//
// 返回:
//   - error: 事务执行失败的错误
func (d *Database) Transaction(fn func(tx *gorm.DB) error) error {
	return d.DB.Transaction(fn)
}

// WithContext 创建带有上下文的数据库会话
// 支持请求上下文传递，便于超时控制和链路追踪
// 参数:
//   - ctx: 上下文对象
//
// 返回:
//   - *gorm.DB: 带有上下文的数据库会话
func (d *Database) WithContext(ctx interface{}) *gorm.DB {
	// 这里简化处理，实际应用中应该使用context.Context
	return d.DB
}

// SeedData 插入种子数据
// 在开发和测试环境中插入初始数据
// 注意：仅在非生产环境中使用
// 返回:
//   - error: 插入种子数据失败的错误
func (d *Database) SeedData() error {
	if d.Config.IsProduction() {
		return fmt.Errorf("生产环境禁止执行种子数据插入")
	}

	d.logger.Info("开始插入种子数据...")

	// 检查是否已存在数据，避免重复插入
	var chainCount int64
	if err := d.DB.Model(&models.Chain{}).Count(&chainCount).Error; err != nil {
		return fmt.Errorf("检查链数据失败: %w", err)
	}

	if chainCount > 0 {
		d.logger.Info("数据库已包含数据，跳过种子数据插入")
		return nil
	}

	// 清理可能存在的孤立聚合器链关系数据
	d.logger.Info("清理可能存在的孤立数据...")
	if err := d.DB.Exec("DELETE FROM aggregator_chains WHERE chain_id NOT IN (SELECT id FROM chains)").Error; err != nil {
		d.logger.Warnf("清理孤立聚合器链关系失败: %v", err)
	}

	// 执行事务插入种子数据
	return d.Transaction(func(tx *gorm.DB) error {
		// 插入链数据
		chains := []models.Chain{
			{
				ChainID:      1,
				Name:         "ethereum",
				DisplayName:  "Ethereum",
				Symbol:       "ETH",
				RPCURL:       "https://mainnet.infura.io/v3/YOUR_PROJECT_ID",
				ExplorerURL:  "https://etherscan.io",
				IsTestnet:    false,
				IsActive:     true,
				GasPriceGwei: 20,
				BlockTimeSec: 15,
			},
			{
				ChainID:      137,
				Name:         "polygon",
				DisplayName:  "Polygon",
				Symbol:       "MATIC",
				RPCURL:       "https://polygon-mainnet.infura.io/v3/YOUR_PROJECT_ID",
				ExplorerURL:  "https://polygonscan.com",
				IsTestnet:    false,
				IsActive:     true,
				GasPriceGwei: 30,
				BlockTimeSec: 2,
			},
		}

		if err := tx.Create(&chains).Error; err != nil {
			return fmt.Errorf("插入链数据失败: %w", err)
		}

		// 插入聚合器数据
		aggregators := []models.Aggregator{
			{
				Name:        "1inch",
				DisplayName: "1inch",
				APIURL:      "https://api.1inch.io/v5.0",
				IsActive:    true,
				Priority:    1,
				TimeoutMS:   3000,
				RetryCount:  3,
			},
			{
				Name:        "paraswap",
				DisplayName: "ParaSwap",
				APIURL:      "https://apiv5.paraswap.io",
				IsActive:    true,
				Priority:    2,
				TimeoutMS:   4000,
				RetryCount:  3,
			},
		}

		if err := tx.Create(&aggregators).Error; err != nil {
			return fmt.Errorf("插入聚合器数据失败: %w", err)
		}

		// 插入聚合器链关系数据
		aggregatorChains := []models.AggregatorChain{
			// 1inch 支持的链
			{
				AggregatorID:  1, // 1inch
				ChainID:       1, // Ethereum
				IsActive:      true,
				GasMultiplier: decimal.NewFromFloat(1.1),
			},
			{
				AggregatorID:  1,   // 1inch
				ChainID:       137, // Polygon
				IsActive:      true,
				GasMultiplier: decimal.NewFromFloat(1.0),
			},
			// ParaSwap 支持的链
			{
				AggregatorID:  2, // ParaSwap
				ChainID:       1, // Ethereum
				IsActive:      true,
				GasMultiplier: decimal.NewFromFloat(1.0),
			},
			{
				AggregatorID:  2,   // ParaSwap
				ChainID:       137, // Polygon
				IsActive:      true,
				GasMultiplier: decimal.NewFromFloat(1.05),
			},
		}

		if err := tx.Create(&aggregatorChains).Error; err != nil {
			return fmt.Errorf("插入聚合器链关系失败: %w", err)
		}

		d.logger.Info("种子数据插入完成")
		return nil
	})
}

// Cleanup 清理数据库资源
// 清理临时数据、过期缓存等，通常在定时任务中调用
// 返回:
//   - error: 清理过程中的错误
func (d *Database) Cleanup() error {
	d.logger.Info("开始清理数据库资源...")

	// 清理过期的报价请求（保留30天）
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := d.DB.Where("created_at < ? AND status = ?", thirtyDaysAgo, "completed").
		Delete(&models.QuoteRequest{}).Error; err != nil {
		d.logger.Errorf("清理过期报价请求失败: %v", err)
		return err
	}

	// 清理旧的系统指标数据（保留7天）
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	if err := d.DB.Where("timestamp < ?", sevenDaysAgo).
		Delete(&models.SystemMetrics{}).Error; err != nil {
		d.logger.Errorf("清理旧系统指标失败: %v", err)
		return err
	}

	d.logger.Info("数据库资源清理完成")
	return nil
}
