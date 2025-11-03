// Package types API网关服务类型定义
// 定义网关相关的配置、路由、负载均衡等类型
// 遵循企业级API网关设计规范
package types

import (
	"net/url"
	"time"
)

// ========================================
// 配置类型定义
// ========================================

// Config API网关服务配置
type Config struct {
	Server       ServerConfig       `json:"server"`        // 服务器配置
	Routing      RoutingConfig      `json:"routing"`       // 路由配置
	Security     SecurityConfig     `json:"security"`      // 安全配置
	LoadBalancer LoadBalancerConfig `json:"load_balancer"` // 负载均衡配置
	Monitoring   MonitoringConfig   `json:"monitoring"`    // 监控配置
	RateLimit    RateLimitConfig    `json:"rate_limit"`    // 限流配置
}

// ServerConfig 服务器基础配置
type ServerConfig struct {
	Port         int           `json:"port"`          // 监听端口
	Environment  string        `json:"environment"`   // 运行环境
	LogLevel     string        `json:"log_level"`     // 日志级别
	ReadTimeout  time.Duration `json:"read_timeout"`  // 读取超时
	WriteTimeout time.Duration `json:"write_timeout"` // 写入超时
	IdleTimeout  time.Duration `json:"idle_timeout"`  // 空闲超时
	Debug        bool          `json:"debug"`         // 调试模式
}

// RoutingConfig 路由配置
type RoutingConfig struct {
	Services   []ServiceRoute `json:"services"`    // 后端服务路由配置
	APIPrefix  string         `json:"api_prefix"`  // API路径前缀
	APIVersion string         `json:"api_version"` // API版本
}

// ServiceRoute 服务路由配置
// 定义单个后端服务的路由规则和负载均衡配置
type ServiceRoute struct {
	Name        string        `json:"name"`         // 服务名称
	PathPrefix  string        `json:"path_prefix"`  // 路径前缀
	Targets     []Target      `json:"targets"`      // 目标实例列表
	HealthCheck HealthCheck   `json:"health_check"` // 健康检查配置
	Timeout     time.Duration `json:"timeout"`      // 请求超时
	RetryCount  int           `json:"retry_count"`  // 重试次数
	Strategy    string        `json:"strategy"`     // 负载均衡策略
}

// Target 目标服务实例
type Target struct {
	URL    *url.URL     `json:"url"`    // 目标URL
	Weight int          `json:"weight"` // 权重
	Active bool         `json:"active"` // 是否激活
	Health HealthStatus `json:"health"` // 健康状态
}

// HealthCheck 健康检查配置
type HealthCheck struct {
	Path     string        `json:"path"`     // 健康检查路径
	Interval time.Duration `json:"interval"` // 检查间隔
	Timeout  time.Duration `json:"timeout"`  // 超时时间
	Enabled  bool          `json:"enabled"`  // 是否启用
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy     bool      `json:"healthy"`         // 是否健康
	LastChecked time.Time `json:"last_checked"`    // 最后检查时间
	Error       string    `json:"error,omitempty"` // 错误信息
}

// ========================================
// 安全配置类型
// ========================================

// SecurityConfig 安全配置
type SecurityConfig struct {
	CORS           CORSConfig `json:"cors"`            // CORS配置
	JWT            JWTConfig  `json:"jwt"`             // JWT配置
	TLS            TLSConfig  `json:"tls"`             // TLS配置
	TrustedProxies []string   `json:"trusted_proxies"` // 信任的代理IP
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`   // 允许的源
	AllowedMethods   []string `json:"allowed_methods"`   // 允许的方法
	AllowedHeaders   []string `json:"allowed_headers"`   // 允许的头部
	ExposedHeaders   []string `json:"exposed_headers"`   // 暴露的头部
	AllowCredentials bool     `json:"allow_credentials"` // 是否允许凭证
	MaxAge           int      `json:"max_age"`           // 预检缓存时间
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey string `json:"-"`         // JWT密钥（不序列化）
	Algorithm string `json:"algorithm"` // 签名算法
	Issuer    string `json:"issuer"`    // 签发者
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled  bool   `json:"enabled"`   // 是否启用TLS
	CertFile string `json:"cert_file"` // 证书文件路径
	KeyFile  string `json:"key_file"`  // 私钥文件路径
}

// ========================================
// 负载均衡配置
// ========================================

// LoadBalancerConfig 负载均衡配置
type LoadBalancerConfig struct {
	Strategy       string               `json:"strategy"`        // 负载均衡策略
	HealthCheck    bool                 `json:"health_check"`    // 是否启用健康检查
	CheckInterval  time.Duration        `json:"check_interval"`  // 健康检查间隔
	MaxRetries     int                  `json:"max_retries"`     // 最大重试次数
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"` // 熔断器配置
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	Enabled          bool          `json:"enabled"`             // 是否启用
	FailureThreshold int           `json:"failure_threshold"`   // 失败阈值
	RecoveryTimeout  time.Duration `json:"recovery_timeout"`    // 恢复超时
	HalfOpenMaxCalls int           `json:"half_open_max_calls"` // 半开状态最大调用数
}

// ========================================
// 限流配置
// ========================================

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled     bool          `json:"enabled"`       // 是否启用限流
	GlobalRate  RateConfig    `json:"global_rate"`   // 全局限流配置
	PerIPRate   RateConfig    `json:"per_ip_rate"`   // 单IP限流配置
	PerUserRate RateConfig    `json:"per_user_rate"` // 单用户限流配置
	Window      time.Duration `json:"window"`        // 时间窗口
}

// RateConfig 限流速率配置
type RateConfig struct {
	Requests int           `json:"requests"` // 请求数量
	Duration time.Duration `json:"duration"` // 时间间隔
}

// ========================================
// 监控配置
// ========================================

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	MetricsEnabled  bool   `json:"metrics_enabled"`   // 是否启用指标
	MetricsPath     string `json:"metrics_path"`      // 指标路径
	HealthCheckPath string `json:"health_check_path"` // 健康检查路径
	LogRequests     bool   `json:"log_requests"`      // 是否记录请求日志
	LogResponses    bool   `json:"log_responses"`     // 是否记录响应日志
	SlowRequestMs   int    `json:"slow_request_ms"`   // 慢请求阈值（毫秒）
}

// ========================================
// 运行时类型
// ========================================

// ServiceInstance 服务实例
// 运行时的服务实例信息
type ServiceInstance struct {
	ID        string          `json:"id"`         // 实例ID
	Name      string          `json:"name"`       // 服务名称
	URL       *url.URL        `json:"url"`        // 服务URL
	Weight    int             `json:"weight"`     // 权重
	Health    HealthStatus    `json:"health"`     // 健康状态
	Metrics   InstanceMetrics `json:"metrics"`    // 实例指标
	LastUsed  time.Time       `json:"last_used"`  // 最后使用时间
	CreatedAt time.Time       `json:"created_at"` // 创建时间
}

// InstanceMetrics 实例指标
type InstanceMetrics struct {
	TotalRequests   int64         `json:"total_requests"`    // 总请求数
	SuccessRequests int64         `json:"success_requests"`  // 成功请求数
	FailedRequests  int64         `json:"failed_requests"`   // 失败请求数
	AvgResponseTime time.Duration `json:"avg_response_time"` // 平均响应时间
	LastRequestTime time.Time     `json:"last_request_time"` // 最后请求时间
}

// ========================================
// 请求和响应类型
// ========================================

// ProxyRequest 代理请求信息
// 包含原始请求和路由信息
type ProxyRequest struct {
	OriginalURL string            `json:"original_url"`      // 原始URL
	TargetURL   string            `json:"target_url"`        // 目标URL
	Method      string            `json:"method"`            // HTTP方法
	Headers     map[string]string `json:"headers"`           // 请求头
	ServiceName string            `json:"service_name"`      // 目标服务名称
	UserID      string            `json:"user_id,omitempty"` // 用户ID（如果已认证）
	RequestID   string            `json:"request_id"`        // 请求ID
	StartTime   time.Time         `json:"start_time"`        // 请求开始时间
}

// ProxyResponse 代理响应信息
type ProxyResponse struct {
	StatusCode  int               `json:"status_code"`     // HTTP状态码
	Headers     map[string]string `json:"headers"`         // 响应头
	Body        []byte            `json:"body,omitempty"`  // 响应体
	Duration    time.Duration     `json:"duration"`        // 处理时间
	TargetURL   string            `json:"target_url"`      // 实际目标URL
	ServiceName string            `json:"service_name"`    // 服务名称
	Error       string            `json:"error,omitempty"` // 错误信息
}

// ========================================
// API响应类型
// ========================================

// APIResponse 统一API响应格式
type APIResponse struct {
	Success   bool        `json:"success"`         // 是否成功
	Data      interface{} `json:"data,omitempty"`  // 响应数据
	Error     *APIError   `json:"error,omitempty"` // 错误信息
	Meta      interface{} `json:"meta,omitempty"`  // 元数据
	Timestamp int64       `json:"timestamp"`       // 时间戳
	RequestID string      `json:"request_id"`      // 请求ID
}

// APIError API错误信息
type APIError struct {
	Code    string                 `json:"code"`              // 错误代码
	Message string                 `json:"message"`           // 错误消息
	Details map[string]interface{} `json:"details,omitempty"` // 详细信息
}

// GatewayStats 网关统计信息
type GatewayStats struct {
	TotalRequests     int64                   `json:"total_requests"`      // 总请求数
	SuccessRequests   int64                   `json:"success_requests"`    // 成功请求数
	FailedRequests    int64                   `json:"failed_requests"`     // 失败请求数
	AvgResponseTime   time.Duration           `json:"avg_response_time"`   // 平均响应时间
	RequestsPerSecond float64                 `json:"requests_per_second"` // 每秒请求数
	ServiceStats      map[string]ServiceStats `json:"service_stats"`       // 各服务统计
	Uptime            time.Duration           `json:"uptime"`              // 运行时间
	LastRestart       time.Time               `json:"last_restart"`        // 最后重启时间
}

// ServiceStats 服务统计信息
type ServiceStats struct {
	Name            string        `json:"name"`              // 服务名称
	TotalRequests   int64         `json:"total_requests"`    // 总请求数
	SuccessRate     float64       `json:"success_rate"`      // 成功率
	AvgResponseTime time.Duration `json:"avg_response_time"` // 平均响应时间
	ActiveInstances int           `json:"active_instances"`  // 活跃实例数
	LastRequestTime time.Time     `json:"last_request_time"` // 最后请求时间
}

// ========================================
// 常量定义
// ========================================

// 负载均衡策略
const (
	StrategyRoundRobin = "round_robin" // 轮询
	StrategyWeighted   = "weighted"    // 加权
	StrategyLeastConn  = "least_conn"  // 最少连接
	StrategyRandom     = "random"      // 随机
)

// 服务名称常量
const (
	ServiceBusinessLogic = "business-logic" // 业务逻辑服务
	ServiceSmartRouter   = "smart-router"   // 智能路由服务
)

// 错误代码常量
const (
	ErrCodeInvalidRequest     = "INVALID_REQUEST"     // 无效请求
	ErrCodeUnauthorized       = "UNAUTHORIZED"        // 未授权
	ErrCodeForbidden          = "FORBIDDEN"           // 禁止访问
	ErrCodeNotFound           = "NOT_FOUND"           // 资源不存在
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE" // 服务不可用
	ErrCodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED" // 频率限制
	ErrCodeInternalError      = "INTERNAL_ERROR"      // 内部错误
	ErrCodeBadGateway         = "BAD_GATEWAY"         // 网关错误
	ErrCodeGatewayTimeout     = "GATEWAY_TIMEOUT"     // 网关超时
)

// HTTP头常量
const (
	HeaderRequestID      = "X-Request-ID"      // 请求ID头
	HeaderForwardedFor   = "X-Forwarded-For"   // 转发IP头
	HeaderForwardedHost  = "X-Forwarded-Host"  // 转发主机头
	HeaderForwardedProto = "X-Forwarded-Proto" // 转发协议头
	HeaderRealIP         = "X-Real-IP"         // 真实IP头
	HeaderUserAgent      = "User-Agent"        // 用户代理头
	HeaderAuthorization  = "Authorization"     // 认证头
	HeaderContentType    = "Content-Type"      // 内容类型头
)

// 服务发现相关常量
const (
	ServiceStatusHealthy   = "healthy"   // 健康状态
	ServiceStatusUnhealthy = "unhealthy" // 不健康状态
	ServiceStatusDegraded  = "degraded"  // 降级状态
	ServiceStatusUnknown   = "unknown"   // 未知状态
)
