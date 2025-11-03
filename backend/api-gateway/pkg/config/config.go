// Package config API网关配置管理
// 提供配置加载、验证、热重载等功能
// 支持多环境配置和服务发现
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"defi-aggregator/api-gateway/internal/types"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Load 加载API网关配置
// 从环境变量和配置文件加载完整的网关配置
// 返回:
//   - *types.Config: 完整的网关配置
//   - error: 配置加载或验证错误
func Load() (*types.Config, error) {
	// 尝试加载.env文件
	if err := godotenv.Load(); err != nil {
		logrus.Info("未找到.env文件，使用环境变量配置")
	}

	config := &types.Config{
		Server: types.ServerConfig{
			Port:         getEnvAsInt("PORT", 0),  // 必填
			Environment:  getEnv("APP_ENV", ""),   // 必填
			LogLevel:     getEnv("LOG_LEVEL", ""), // 必填
			ReadTimeout:  getEnvAsDuration("READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvAsDuration("IDLE_TIMEOUT", 120*time.Second),
			Debug:        getEnvAsBool("DEBUG", false),
		},
		Routing: loadRoutingConfig(),
		Security: types.SecurityConfig{
			CORS: types.CORSConfig{
				AllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{}), // 必填，从全局配置读取
				AllowedMethods: getEnvAsSlice("CORS_ALLOWED_METHODS", []string{
					"GET", "POST", "PUT", "DELETE", "OPTIONS",
				}),
				AllowedHeaders: getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{
					"Content-Type", "Authorization", "X-Request-ID", "X-User-Agent",
				}),
				AllowCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
				MaxAge:           getEnvAsInt("CORS_MAX_AGE", 86400),
			},
			JWT: types.JWTConfig{
				SecretKey: getEnv("JWT_SECRET_KEY", ""), // 必填，从全局配置读取
				Algorithm: getEnv("JWT_ALGORITHM", "HS256"),
				Issuer:    getEnv("JWT_ISSUER", "defi-aggregator-gateway"),
			},
			TrustedProxies: getEnvAsSlice("TRUSTED_PROXIES", []string{"127.0.0.1", "::1"}),
		},
		LoadBalancer: types.LoadBalancerConfig{
			Strategy:      getEnv("LB_STRATEGY", types.StrategyRoundRobin),
			HealthCheck:   getEnvAsBool("LB_HEALTH_CHECK", true),
			CheckInterval: getEnvAsDuration("LB_CHECK_INTERVAL", 30*time.Second),
			MaxRetries:    getEnvAsInt("LB_MAX_RETRIES", 3),
			CircuitBreaker: types.CircuitBreakerConfig{
				Enabled:          getEnvAsBool("CB_ENABLED", true),
				FailureThreshold: getEnvAsInt("CB_FAILURE_THRESHOLD", 5),
				RecoveryTimeout:  getEnvAsDuration("CB_RECOVERY_TIMEOUT", 60*time.Second),
				HalfOpenMaxCalls: getEnvAsInt("CB_HALF_OPEN_MAX_CALLS", 3),
			},
		},
		Monitoring: types.MonitoringConfig{
			MetricsEnabled:  getEnvAsBool("METRICS_ENABLED", true),
			MetricsPath:     getEnv("METRICS_PATH", "/metrics"),
			HealthCheckPath: getEnv("HEALTH_CHECK_PATH", "/health"),
			LogRequests:     getEnvAsBool("LOG_REQUESTS", true),
			LogResponses:    getEnvAsBool("LOG_RESPONSES", false),
			SlowRequestMs:   getEnvAsInt("SLOW_REQUEST_MS", 1000),
		},
		RateLimit: types.RateLimitConfig{
			Enabled: getEnvAsBool("RATE_LIMIT_ENABLED", true),
			GlobalRate: types.RateConfig{
				Requests: getEnvAsInt("GLOBAL_RATE_REQUESTS", 1000),
				Duration: getEnvAsDuration("GLOBAL_RATE_DURATION", 1*time.Minute),
			},
			PerIPRate: types.RateConfig{
				Requests: getEnvAsInt("PER_IP_RATE_REQUESTS", 100),
				Duration: getEnvAsDuration("PER_IP_RATE_DURATION", 1*time.Minute),
			},
			Window: getEnvAsDuration("RATE_LIMIT_WINDOW", 1*time.Minute),
		},
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// loadRoutingConfig 加载路由配置
// 配置后端服务的路由规则和负载均衡
func loadRoutingConfig() types.RoutingConfig {
	// 业务逻辑服务配置（必填，从全局配置读取）
	businessLogicTargets := parseTargets(getEnv("BUSINESS_LOGIC_TARGETS", ""))

	// 智能路由服务配置（必填，从全局配置读取）
	smartRouterTargets := parseTargets(getEnv("SMART_ROUTER_TARGETS", ""))

	return types.RoutingConfig{
		APIPrefix:  getEnv("API_PREFIX", "/api"),
		APIVersion: getEnv("API_VERSION", "v1"),
		Services: []types.ServiceRoute{
			{
				Name:       types.ServiceBusinessLogic,
				PathPrefix: "/api/v1",
				Targets:    businessLogicTargets,
				HealthCheck: types.HealthCheck{
					Path:     "/health",
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					Enabled:  true,
				},
				Timeout:    getEnvAsDuration("BUSINESS_LOGIC_TIMEOUT", 30*time.Second),
				RetryCount: getEnvAsInt("BUSINESS_LOGIC_RETRIES", 2),
				Strategy:   types.StrategyRoundRobin,
			},
			{
				Name:       types.ServiceSmartRouter,
				PathPrefix: "/api/v1/router", // 网关特定路径，直接访问智能路由
				Targets:    smartRouterTargets,
				HealthCheck: types.HealthCheck{
					Path:     "/health",
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					Enabled:  true,
				},
				Timeout:    getEnvAsDuration("SMART_ROUTER_TIMEOUT", 10*time.Second),
				RetryCount: getEnvAsInt("SMART_ROUTER_RETRIES", 1),
				Strategy:   types.StrategyRoundRobin,
			},
		},
	}
}

// parseTargets 解析目标服务列表
// 支持逗号分隔的多个URL，用于负载均衡
func parseTargets(targetURLs string) []types.Target {
	var targets []types.Target

	urls := strings.Split(targetURLs, ",")
	for _, urlStr := range urls {
		urlStr = strings.TrimSpace(urlStr)
		if urlStr == "" {
			continue
		}

		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			logrus.Warnf("无效的目标URL: %s, 错误: %v", urlStr, err)
			continue
		}

		targets = append(targets, types.Target{
			URL:    parsedURL,
			Weight: 1, // 默认权重
			Active: true,
			Health: types.HealthStatus{
				Healthy: true, // 初始假设健康
			},
		})
	}

	return targets
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

	// 验证必填的安全配置
	if cfg.Security.JWT.SecretKey == "" {
		return fmt.Errorf("JWT_SECRET_KEY环境变量是必填项")
	}
	if len(cfg.Security.JWT.SecretKey) < 32 {
		return fmt.Errorf("JWT密钥长度必须至少32个字符")
	}
	if len(cfg.Security.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS环境变量是必填项")
	}

	// 验证必填的路由配置
	if getEnv("BUSINESS_LOGIC_TARGETS", "") == "" {
		return fmt.Errorf("BUSINESS_LOGIC_TARGETS环境变量是必填项")
	}
	if getEnv("SMART_ROUTER_TARGETS", "") == "" {
		return fmt.Errorf("SMART_ROUTER_TARGETS环境变量是必填项")
	}

	// 验证路由配置
	if len(cfg.Routing.Services) == 0 {
		return fmt.Errorf("至少需要配置一个后端服务")
	}

	for _, service := range cfg.Routing.Services {
		if service.Name == "" {
			return fmt.Errorf("服务名称不能为空")
		}

		if len(service.Targets) == 0 {
			return fmt.Errorf("服务 %s 至少需要一个目标实例", service.Name)
		}

		for _, target := range service.Targets {
			if target.URL == nil {
				return fmt.Errorf("服务 %s 的目标URL不能为空", service.Name)
			}
		}
	}

	// 生产环境额外验证
	if cfg.Server.Environment == "production" {
		if cfg.Server.Debug {
			return fmt.Errorf("生产环境不应启用调试模式")
		}
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

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
