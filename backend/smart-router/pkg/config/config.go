// Package config 智能路由服务配置管理
// 提供配置加载、验证、环境变量处理等功能
// 支持多环境配置和热重载
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"defi-aggregator/smart-router/internal/types"

	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// Load 加载智能路由服务配置
// 从环境变量和.env文件加载配置，设置默认值
// 返回:
//   - *types.Config: 完整的服务配置
//   - error: 配置加载或验证错误
func Load() (*types.Config, error) {
	// 尝试加载.env文件
	if err := godotenv.Load(); err != nil {
		logrus.Info("未找到.env文件，使用环境变量配置")
	}

	config := &types.Config{
		Server: types.ServerConfig{
			Port:        getEnvAsInt("PORT", 0),  // 必填
			Environment: getEnv("APP_ENV", ""),   // 必填
			LogLevel:    getEnv("LOG_LEVEL", ""), // 必填
			Debug:       getEnvAsBool("DEBUG", false),
		},
		Redis: types.RedisConfig{
			Host:     getEnv("REDIS_HOST", ""),     // 必填
			Port:     getEnvAsInt("REDIS_PORT", 0), // 必填
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB_SMART_ROUTER", 0), // 从全局配置读取
			PoolSize: getEnvAsInt("REDIS_POOL_SIZE", 10),
		},
		Providers: loadProviderConfigs(),
		Strategy:  loadAggregationStrategy(),
		Cache: types.CacheConfig{
			DefaultTTL:      getEnvAsDuration("CACHE_DEFAULT_TTL", 10*time.Second),
			MaxEntries:      getEnvAsInt("CACHE_MAX_ENTRIES", 10000),
			CleanupInterval: getEnvAsDuration("CACHE_CLEANUP_INTERVAL", 5*time.Minute),
			PrefixKey:       getEnv("CACHE_PREFIX", "smart_router:"),
		},
		Monitoring: types.MonitoringConfig{
			MetricsEnabled:  getEnvAsBool("METRICS_ENABLED", true),
			MetricsPath:     getEnv("METRICS_PATH", "/metrics"),
			HealthCheckPath: getEnv("HEALTH_CHECK_PATH", "/health"),
			StatsInterval:   getEnvAsDuration("STATS_INTERVAL", 1*time.Minute),
		},
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// loadProviderConfigs 加载聚合器配置
// 从环境变量加载各个聚合器的配置信息
func loadProviderConfigs() []types.ProviderConfig {
	providers := []types.ProviderConfig{
		// 1inch配置（从全局配置读取）
		{
			Name:            types.Provider1inch,
			DisplayName:     "1inch",
			BaseURL:         getEnv("ONEINCH_API_URL", ""), // 必填，从全局配置读取
			APIKey:          getEnv("ONEINCH_API_KEY", ""), // 从全局配置读取
			Timeout:         getEnvAsDuration("ONEINCH_TIMEOUT", 3*time.Second),
			RetryCount:      getEnvAsInt("ONEINCH_RETRY_COUNT", 2),
			Priority:        1,
			Weight:          decimal.NewFromFloat(1.0),
			IsActive:        getEnvAsBool("ONEINCH_ENABLED", false),
			SupportedChains: []uint{1, 137, 42161, 10, 11155111}, // Ethereum, Polygon, Arbitrum, Optimism, Sepolia
		},

		// ParaSwap配置（从全局配置读取）
		{
			Name:            types.ProviderParaswap,
			DisplayName:     "ParaSwap",
			BaseURL:         getEnv("PARASWAP_API_URL", ""), // 必填，从全局配置读取
			APIKey:          getEnv("PARASWAP_API_KEY", ""), // 从全局配置读取
			Timeout:         getEnvAsDuration("PARASWAP_TIMEOUT", 4*time.Second),
			RetryCount:      getEnvAsInt("PARASWAP_RETRY_COUNT", 2),
			Priority:        2,
			Weight:          decimal.NewFromFloat(0.9),
			IsActive:        getEnvAsBool("PARASWAP_ENABLED", false),
			SupportedChains: []uint{1, 137, 42161, 11155111}, // Ethereum, Polygon, Arbitrum, Sepolia
		},

		// 0x Protocol配置（从全局配置读取）
		{
			Name:            types.Provider0x,
			DisplayName:     "0x Protocol",
			BaseURL:         getEnv("ZRX_API_URL", ""), // 必填，从全局配置读取
			APIKey:          getEnv("ZRX_API_KEY", ""), // 从全局配置读取
			Timeout:         getEnvAsDuration("ZRX_TIMEOUT", 5*time.Second),
			RetryCount:      getEnvAsInt("ZRX_RETRY_COUNT", 2),
			Priority:        3,
			Weight:          decimal.NewFromFloat(0.8),
			IsActive:        getEnvAsBool("ZRX_ENABLED", false),
			SupportedChains: []uint{1, 137, 11155111}, // Ethereum, Polygon, Sepolia
		},

		// CoW Protocol配置（从全局配置读取）
		{
			Name:            types.ProviderCowswap,
			DisplayName:     "CoW Protocol",
			BaseURL:         getEnv("COW_API_URL", ""), // 必填，从全局配置读取
			APIKey:          getEnv("COW_API_KEY", ""), // 从全局配置读取
			Timeout:         getEnvAsDuration("COW_TIMEOUT", 6*time.Second),
			RetryCount:      getEnvAsInt("COW_RETRY_COUNT", 1),
			Priority:        4,
			Weight:          decimal.NewFromFloat(0.7),
			IsActive:        getEnvAsBool("COW_ENABLED", false),
			SupportedChains: []uint{1, 11155111}, // Ethereum, Sepolia
		},
	}

	return providers
}

// loadAggregationStrategy 加载聚合策略配置
// 配置智能路由的决策算法参数
func loadAggregationStrategy() types.AggregationStrategy {
	return types.AggregationStrategy{
		// 时间窗口配置
		MinWaitTime:      getEnvAsDuration("STRATEGY_MIN_WAIT", 300*time.Millisecond),
		MaxWaitTime:      getEnvAsDuration("STRATEGY_MAX_WAIT", 2*time.Second),
		FastResponseTime: getEnvAsDuration("STRATEGY_FAST_RESPONSE", 500*time.Millisecond),
		EmergencyTimeout: getEnvAsDuration("STRATEGY_EMERGENCY_TIMEOUT", 5*time.Second),

		// 质量控制配置
		MinConfidence:      decimal.NewFromFloat(getEnvAsFloat("STRATEGY_MIN_CONFIDENCE", 0.85)),
		MinProviders:       getEnvAsInt("STRATEGY_MIN_PROVIDERS", 1),
		PreferredProviders: getEnvAsInt("STRATEGY_PREFERRED_PROVIDERS", 2),
		OptimalProviders:   getEnvAsInt("STRATEGY_OPTIMAL_PROVIDERS", 3),

		// 决策权重配置
		TimeWeight:       decimal.NewFromFloat(getEnvAsFloat("STRATEGY_TIME_WEIGHT", 0.3)),
		ConfidenceWeight: decimal.NewFromFloat(getEnvAsFloat("STRATEGY_CONFIDENCE_WEIGHT", 0.4)),
		ProviderWeight:   decimal.NewFromFloat(getEnvAsFloat("STRATEGY_PROVIDER_WEIGHT", 0.2)),
		MarketWeight:     decimal.NewFromFloat(getEnvAsFloat("STRATEGY_MARKET_WEIGHT", 0.1)),

		// 决策阈值
		CompositeScoreThreshold: decimal.NewFromFloat(getEnvAsFloat("STRATEGY_COMPOSITE_THRESHOLD", 0.8)),
	}
}

// validateConfig 验证配置的有效性
func validateConfig(cfg *types.Config) error {
	// 验证必填的服务器配置
	if cfg.Server.Port == 0 {
		return fmt.Errorf("PORT环境变量是必填项")
	}
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", cfg.Server.Port)
	}
	if cfg.Server.Environment == "" {
		return fmt.Errorf("APP_ENV环境变量是必填项")
	}
	if cfg.Server.LogLevel == "" {
		return fmt.Errorf("LOG_LEVEL环境变量是必填项")
	}

	// 验证必填的Redis配置
	if cfg.Redis.Host == "" {
		return fmt.Errorf("REDIS_HOST环境变量是必填项")
	}
	if cfg.Redis.Port == 0 {
		return fmt.Errorf("REDIS_PORT环境变量是必填项")
	}

	// 验证第三方API配置
	requiredAPIs := []struct {
		url  string
		name string
	}{
		{getEnv("ONEINCH_API_URL", ""), "ONEINCH_API_URL"},
		{getEnv("PARASWAP_API_URL", ""), "PARASWAP_API_URL"},
		{getEnv("ZRX_API_URL", ""), "ZRX_API_URL"},
		{getEnv("COW_API_URL", ""), "COW_API_URL"},
	}

	for _, api := range requiredAPIs {
		if api.url == "" {
			return fmt.Errorf("%s环境变量是必填项", api.name)
		}
	}

	// 验证至少有一个活跃的聚合器
	activeProviders := 0
	for _, provider := range cfg.Providers {
		if provider.IsActive {
			activeProviders++
		}
	}
	if activeProviders == 0 {
		return fmt.Errorf("至少需要一个活跃的聚合器")
	}

	// 验证聚合策略
	if cfg.Strategy.MinWaitTime > cfg.Strategy.MaxWaitTime {
		return fmt.Errorf("最小等待时间不能大于最大等待时间")
	}

	if cfg.Strategy.MinProviders > cfg.Strategy.PreferredProviders {
		return fmt.Errorf("最小聚合器数不能大于首选聚合器数")
	}

	// 验证权重总和
	totalWeight := cfg.Strategy.TimeWeight.Add(cfg.Strategy.ConfidenceWeight).
		Add(cfg.Strategy.ProviderWeight).Add(cfg.Strategy.MarketWeight)

	expectedWeight := decimal.NewFromFloat(1.0)
	if !totalWeight.Equal(expectedWeight) {
		return fmt.Errorf("决策权重总和必须为1.0，当前为: %s", totalWeight.String())
	}

	return nil
}

// ========================================
// 环境变量辅助函数
// ========================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		logrus.Warnf("无法解析环境变量 %s 为整数，使用默认值 %d", key, defaultValue)
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		logrus.Warnf("无法解析环境变量 %s 为布尔值，使用默认值 %t", key, defaultValue)
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		logrus.Warnf("无法解析环境变量 %s 为时间间隔，使用默认值 %v", key, defaultValue)
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
		logrus.Warnf("无法解析环境变量 %s 为浮点数，使用默认值 %f", key, defaultValue)
	}
	return defaultValue
}

// LoadConfigWithDatabase 加载包含数据库聚合器配置的完整配置
// 使用优雅的配置管理器：数据库控制启用状态，环境变量提供敏感信息
func LoadConfigWithDatabase() (*types.Config, error) {
	// 加载基础配置
	config, err := Load()
	if err != nil {
		return nil, fmt.Errorf("加载基础配置失败: %w", err)
	}

	// 构建数据库连接URL（复用业务逻辑服务的配置方式）
	dbURL := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		getEnv("DB_USER", "admin"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_HOST", "localhost"),
		getEnvAsInt("DB_PORT", 5432),
		getEnv("DB_NAME", "defi_aggregator"),
		getEnv("DB_SSL_MODE", "disable"),
	)

	// 创建优雅的聚合器配置管理器
	configManager, err := NewAggregatorConfigManager(dbURL, logrus.New())
	if err != nil {
		logrus.Warnf("创建聚合器配置管理器失败: %v，使用环境变量配置", err)
		return config, nil // 使用环境变量配置作为后备
	}
	defer configManager.Close()

	// 从数据库加载活跃聚合器配置
	providers, err := configManager.LoadActiveProviders()
	if err != nil {
		logrus.Warnf("从数据库加载聚合器配置失败: %v，使用环境变量配置", err)
		return config, nil // 使用环境变量配置作为后备
	}

	// 替换聚合器配置
	config.Providers = providers
	logrus.Infof("🎉 成功使用数据库聚合器配置，共 %d 个活跃聚合器", len(providers))

	return config, nil
}
