// Package config 提供应用程序配置管理功能
// 支持从环境变量、配置文件等多种来源加载配置
// 采用企业级配置管理最佳实践，确保配置的安全性和可维护性
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config 应用程序配置结构体
// 包含所有服务运行所需的配置项，按功能模块分组
// 遵循十二因子应用配置原则
type Config struct {
	// 服务基础配置
	Server ServerConfig `json:"server"`

	// 数据库配置
	Database DatabaseConfig `json:"database"`

	// Redis缓存配置
	Redis RedisConfig `json:"redis"`

	// JWT认证配置
	JWT JWTConfig `json:"jwt"`

	// 外部服务配置
	ExternalServices ExternalServicesConfig `json:"external_services"`

	// 安全配置
	Security SecurityConfig `json:"security"`

	// 业务配置
	Business BusinessConfig `json:"business"`

	// 监控配置
	Monitoring MonitoringConfig `json:"monitoring"`
}

// ServerConfig 服务器相关配置
type ServerConfig struct {
	Port        int    `json:"port"`         // 服务监听端口
	Environment string `json:"environment"`  // 运行环境: development, staging, production
	ServiceName string `json:"service_name"` // 服务名称，用于日志和监控
	LogLevel    string `json:"log_level"`    // 日志级别: debug, info, warn, error
	Debug       bool   `json:"debug"`        // 是否启用调试模式
}

// DatabaseConfig 数据库配置
// 包含连接池、超时等性能优化配置
type DatabaseConfig struct {
	Host            string        `json:"host"`              // 数据库主机地址
	Port            int           `json:"port"`              // 数据库端口
	User            string        `json:"user"`              // 用户名
	Password        string        `json:"-"`                 // 密码，不序列化到JSON
	Database        string        `json:"database"`          // 数据库名
	SSLMode         string        `json:"ssl_mode"`          // SSL模式
	MaxOpenConns    int           `json:"max_open_conns"`    // 最大打开连接数
	MaxIdleConns    int           `json:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"` // 连接最大生命周期
	TestDatabase    string        `json:"test_database"`     // 测试数据库名
}

// RedisConfig Redis缓存配置
type RedisConfig struct {
	Host       string `json:"host"`        // Redis主机地址
	Port       int    `json:"port"`        // Redis端口
	Password   string `json:"-"`           // 密码，不序列化到JSON
	DB         int    `json:"db"`          // 数据库编号
	MaxRetries int    `json:"max_retries"` // 最大重试次数
	PoolSize   int    `json:"pool_size"`   // 连接池大小
}

// JWTConfig JWT认证配置
type JWTConfig struct {
	SecretKey        string        `json:"-"`                  // JWT密钥，不序列化
	ExpiresIn        time.Duration `json:"expires_in"`         // 访问令牌过期时间
	RefreshExpiresIn time.Duration `json:"refresh_expires_in"` // 刷新令牌过期时间
	Issuer           string        `json:"issuer"`             // 令牌签发者
	Algorithm        string        `json:"algorithm"`          // 签名算法
}

// ExternalServicesConfig 外部服务配置
type ExternalServicesConfig struct {
	SmartRouterURL string        `json:"smart_router_url"` // 智能路由服务URL
	Timeout        time.Duration `json:"timeout"`          // 外部服务调用超时时间
}

// SecurityConfig 安全相关配置
type SecurityConfig struct {
	CORSAllowedOrigins []string      `json:"cors_allowed_origins"` // CORS允许的源
	CORSAllowedMethods []string      `json:"cors_allowed_methods"` // CORS允许的方法
	CORSAllowedHeaders []string      `json:"cors_allowed_headers"` // CORS允许的头部
	RateLimitRequests  int           `json:"rate_limit_requests"`  // 限流请求数
	RateLimitDuration  time.Duration `json:"rate_limit_duration"`  // 限流时间窗口
}

// BusinessConfig 业务相关配置
type BusinessConfig struct {
	DefaultPageSize int           `json:"default_page_size"` // 默认分页大小
	MaxPageSize     int           `json:"max_page_size"`     // 最大分页大小
	CacheTTLShort   time.Duration `json:"cache_ttl_short"`   // 短期缓存TTL
	CacheTTLMedium  time.Duration `json:"cache_ttl_medium"`  // 中期缓存TTL
	CacheTTLLong    time.Duration `json:"cache_ttl_long"`    // 长期缓存TTL
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	MetricsEnabled  bool   `json:"metrics_enabled"`   // 是否启用指标收集
	MetricsPath     string `json:"metrics_path"`      // 指标暴露路径
	HealthCheckPath string `json:"health_check_path"` // 健康检查路径
	EnableSwagger   bool   `json:"enable_swagger"`    // 是否启用API文档
	LogFormat       string `json:"log_format"`        // 日志格式: json, text
	LogOutput       string `json:"log_output"`        // 日志输出: stdout, file
}

// Load 加载配置
// 按优先级从多个来源加载配置：环境变量 > .env文件 > 默认值
// 返回完整的配置对象和可能的错误
func Load() (*Config, error) {
	// 尝试加载.env文件，忽略错误（生产环境可能不存在）
	if err := godotenv.Load(); err != nil {
		logrus.Info("未找到.env文件，使用环境变量配置")
	}

	config := &Config{
		Server: ServerConfig{
			Port:        getEnvAsInt("PORT", 0), // 必填
			Environment: getEnv("APP_ENV", ""),  // 必填
			ServiceName: getEnv("SERVICE_NAME", "defi-aggregator-business-logic"),
			LogLevel:    getEnv("LOG_LEVEL", ""),
			Debug:       getEnvAsBool("DEBUG", false),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", ""),     // 必填
			Port:            getEnvAsInt("DB_PORT", 0), // 必填
			User:            getEnv("DB_USER", ""),     // 必填
			Password:        getEnv("DB_PASSWORD", ""), // 必填
			Database:        getEnv("DB_NAME", ""),     // 必填
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("DB_MAX_LIFETIME", 300*time.Second),
			TestDatabase:    getEnv("TEST_DB_NAME", "defi_aggregator_test"),
		},
		Redis: RedisConfig{
			Host:       getEnv("REDIS_HOST", ""),     // 必填
			Port:       getEnvAsInt("REDIS_PORT", 0), // 必填
			Password:   getEnv("REDIS_PASSWORD", ""),
			DB:         getEnvAsInt("REDIS_DB_BUSINESS_LOGIC", 0),
			MaxRetries: getEnvAsInt("REDIS_MAX_RETRIES", 3),
			PoolSize:   getEnvAsInt("REDIS_POOL_SIZE", 10),
		},
		JWT: JWTConfig{
			SecretKey:        getEnv("JWT_SECRET_KEY", ""), // 必填
			ExpiresIn:        getEnvAsDuration("JWT_EXPIRES_IN", 24*time.Hour),
			RefreshExpiresIn: getEnvAsDuration("JWT_REFRESH_EXPIRES_IN", 168*time.Hour),
			Issuer:           getEnv("JWT_ISSUER", "defi-aggregator"),
			Algorithm:        getEnv("JWT_ALGORITHM", "HS256"),
		},
		ExternalServices: ExternalServicesConfig{
			SmartRouterURL: getEnv("SMART_ROUTER_URL", ""), // 必填
			Timeout:        getEnvAsDuration("EXTERNAL_SERVICE_TIMEOUT", 30*time.Second),
		},
		Security: SecurityConfig{
			CORSAllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{}), // 必填
			CORSAllowedMethods: getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			CORSAllowedHeaders: getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"Content-Type", "Authorization", "X-Requested-With"}),
			RateLimitRequests:  getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
			RateLimitDuration:  getEnvAsDuration("RATE_LIMIT_DURATION", time.Minute),
		},
		Business: BusinessConfig{
			DefaultPageSize: getEnvAsInt("DEFAULT_PAGE_SIZE", 20),
			MaxPageSize:     getEnvAsInt("MAX_PAGE_SIZE", 100),
			CacheTTLShort:   getEnvAsDuration("CACHE_TTL_SHORT", 60*time.Second),
			CacheTTLMedium:  getEnvAsDuration("CACHE_TTL_MEDIUM", 300*time.Second),
			CacheTTLLong:    getEnvAsDuration("CACHE_TTL_LONG", 3600*time.Second),
		},
		Monitoring: MonitoringConfig{
			MetricsEnabled:  getEnvAsBool("METRICS_ENABLED", true),
			MetricsPath:     getEnv("METRICS_PATH", "/metrics"),
			HealthCheckPath: getEnv("HEALTH_CHECK_PATH", "/health"),
			EnableSwagger:   getEnvAsBool("ENABLE_SWAGGER", true),
			LogFormat:       getEnv("LOG_FORMAT", "json"),
			LogOutput:       getEnv("LOG_OUTPUT", "stdout"),
		},
	}

	// 验证关键配置项
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// validate 验证配置的有效性
// 检查关键配置项是否符合要求，防止运行时错误
func (c *Config) validate() error {
	// 验证必填的服务器配置
	if c.Server.Port == 0 {
		return fmt.Errorf("PORT环境变量是必填项")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("服务端口必须在1-65535范围内，当前值: %d", c.Server.Port)
	}
	if c.Server.Environment == "" {
		return fmt.Errorf("APP_ENV环境变量是必填项")
	}
	if c.Server.LogLevel == "" {
		return fmt.Errorf("LOG_LEVEL环境变量是必填项")
	}

	// 验证环境配置
	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
	}
	if !validEnvs[c.Server.Environment] {
		return fmt.Errorf("无效的环境配置: %s，支持的值: development, staging, production", c.Server.Environment)
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Server.LogLevel] {
		return fmt.Errorf("无效的日志级别: %s，支持的值: debug, info, warn, error", c.Server.LogLevel)
	}

	// 验证必填的数据库配置
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST环境变量是必填项")
	}
	if c.Database.Port == 0 {
		return fmt.Errorf("DB_PORT环境变量是必填项")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER环境变量是必填项")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD环境变量是必填项")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("DB_NAME环境变量是必填项")
	}

	// 验证必填的Redis配置
	if c.Redis.Host == "" {
		return fmt.Errorf("REDIS_HOST环境变量是必填项")
	}
	if c.Redis.Port == 0 {
		return fmt.Errorf("REDIS_PORT环境变量是必填项")
	}

	// 验证必填的JWT配置
	if c.JWT.SecretKey == "" {
		return fmt.Errorf("JWT_SECRET_KEY环境变量是必填项")
	}
	if len(c.JWT.SecretKey) < 32 {
		return fmt.Errorf("JWT密钥长度必须至少32个字符，当前长度: %d", len(c.JWT.SecretKey))
	}

	// 验证必填的外部服务配置
	if c.ExternalServices.SmartRouterURL == "" {
		return fmt.Errorf("SMART_ROUTER_URL环境变量是必填项")
	}

	// 验证必填的安全配置
	if len(c.Security.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS环境变量是必填项")
	}

	// 在生产环境验证更严格的安全配置
	if c.Server.Environment == "production" {
		if c.Server.Debug {
			return fmt.Errorf("生产环境不应启用调试模式")
		}
	}

	return nil
}

// GetDatabaseDSN 获取数据库连接字符串
// 根据配置生成PostgreSQL连接DSN
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// GetTestDatabaseDSN 获取测试数据库连接字符串
func (c *Config) GetTestDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.TestDatabase,
		c.Database.SSLMode,
	)
}

// GetRedisAddress 获取Redis连接地址
func (c *Config) GetRedisAddress() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// IsProduction 判断是否为生产环境
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// IsDevelopment 判断是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// 辅助函数：从环境变量获取字符串值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 辅助函数：从环境变量获取整数值
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		logrus.Warnf("无法解析环境变量 %s 为整数，使用默认值 %d", key, defaultValue)
	}
	return defaultValue
}

// 辅助函数：从环境变量获取布尔值
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		logrus.Warnf("无法解析环境变量 %s 为布尔值，使用默认值 %t", key, defaultValue)
	}
	return defaultValue
}

// 辅助函数：从环境变量获取时间间隔值
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		logrus.Warnf("无法解析环境变量 %s 为时间间隔，使用默认值 %v", key, defaultValue)
	}
	return defaultValue
}

// 辅助函数：从环境变量获取字符串切片
func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// 使用逗号分割字符串
		parts := make([]string, 0)
		for _, part := range splitAndTrim(value, ",") {
			if part != "" {
				parts = append(parts, part)
			}
		}
		return parts
	}
	return defaultValue
}

// splitAndTrim 分割字符串并去除空白
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	parts := make([]string, 0)
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
