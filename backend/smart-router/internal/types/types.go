// Package types 定义智能路由服务中使用的所有数据类型
// 包含聚合请求响应、路由算法配置、性能指标等
// 遵循领域驱动设计原则，确保类型安全和业务语义清晰
package types

import (
	"time"

	"github.com/shopspring/decimal"
)

// ========================================
// 核心业务类型定义
// ========================================

// QuoteRequest 报价请求
// 前端或业务逻辑层发送给智能路由的报价请求
type QuoteRequest struct {
	RequestID   string           `json:"request_id" validate:"required"`     // 唯一请求ID
	FromToken   string           `json:"from_token" validate:"required"`     // 源代币合约地址
	ToToken     string           `json:"to_token" validate:"required"`       // 目标代币合约地址
	AmountIn    decimal.Decimal  `json:"amount_in" validate:"required,gt=0"` // 输入数量(wei格式)
	ChainID     uint             `json:"chain_id" validate:"required"`       // 区块链ID
	Slippage    decimal.Decimal  `json:"slippage" validate:"gte=0,lte=0.5"`  // 滑点容忍度
	UserAddress string           `json:"user_address,omitempty"`             // 用户钱包地址(可选)
	GasPrice    *decimal.Decimal `json:"gas_price,omitempty"`                // 指定Gas价格(可选)
	Deadline    *time.Time       `json:"deadline,omitempty"`                 // 交易截止时间(可选)
}

// QuoteResponse 聚合报价响应
// 智能路由返回的最优报价结果
type QuoteResponse struct {
	RequestID       string                 `json:"request_id"`              // 请求ID
	Success         bool                   `json:"success"`                 // 是否成功
	BestProvider    string                 `json:"best_provider"`           // 最佳聚合器
	BestPrice       decimal.Decimal        `json:"best_price"`              // 最佳价格(输出数量)
	BestGasEstimate uint64                 `json:"best_gas_estimate"`       // 最佳Gas估算
	PriceImpact     decimal.Decimal        `json:"price_impact"`            // 价格冲击
	ExchangeRate    decimal.Decimal        `json:"exchange_rate"`           // 汇率
	Route           []RouteStep            `json:"route,omitempty"`         // 交易路径
	AllQuotes       []*ProviderQuote       `json:"all_quotes"`              // 所有聚合器报价
	Performance     AggregationPerformance `json:"performance"`             // 聚合性能指标
	ValidUntil      time.Time              `json:"valid_until"`             // 报价有效期
	CacheHit        bool                   `json:"cache_hit"`               // 是否命中缓存
	ErrorMessage    string                 `json:"error_message,omitempty"` // 错误信息
	Timestamp       time.Time              `json:"timestamp"`               // 响应时间戳
}

// RouteStep 交易路径步骤
// 描述单个交易路径的协议和比例
type RouteStep struct {
	Protocol   string          `json:"protocol"`       // 协议名称 (UNISWAP_V3, SUSHISWAP等)
	Percentage decimal.Decimal `json:"percentage"`     // 在该协议上的交易比例
	Pool       string          `json:"pool,omitempty"` // 流动性池地址(可选)
}

// ProviderQuote 单个聚合器的报价
// 记录每个聚合器的响应结果和性能指标
type ProviderQuote struct {
	Provider     string          `json:"provider"`                // 聚合器名称
	Success      bool            `json:"success"`                 // 是否成功响应
	AmountOut    decimal.Decimal `json:"amount_out"`              // 输出数量
	GasEstimate  uint64          `json:"gas_estimate"`            // Gas估算
	PriceImpact  decimal.Decimal `json:"price_impact"`            // 价格冲击
	Route        []RouteStep     `json:"route,omitempty"`         // 交易路径
	ResponseTime time.Duration   `json:"response_time"`           // 响应时间
	Confidence   decimal.Decimal `json:"confidence"`              // 置信度评分
	Rank         int             `json:"rank"`                    // 价格排名
	ErrorCode    string          `json:"error_code,omitempty"`    // 错误代码
	ErrorMessage string          `json:"error_message,omitempty"` // 错误信息
	RawResponse  interface{}     `json:"raw_response,omitempty"`  // 原始响应(调试用)
}

// AggregationPerformance 聚合性能指标
// 记录本次聚合的性能和质量指标
type AggregationPerformance struct {
	TotalDuration    time.Duration   `json:"total_duration"`    // 总耗时
	ProvidersQueried int             `json:"providers_queried"` // 查询的聚合器数量
	ProvidersSuccess int             `json:"providers_success"` // 成功响应的数量
	FastestProvider  string          `json:"fastest_provider"`  // 最快响应的聚合器
	SlowestProvider  string          `json:"slowest_provider"`  // 最慢响应的聚合器
	AvgResponseTime  time.Duration   `json:"avg_response_time"` // 平均响应时间
	CacheHitRate     decimal.Decimal `json:"cache_hit_rate"`    // 缓存命中率
	StrategyUsed     string          `json:"strategy_used"`     // 使用的聚合策略
	QualityScore     decimal.Decimal `json:"quality_score"`     // 聚合质量评分
}

// ========================================
// 聚合器配置类型
// ========================================

// ProviderConfig 聚合器配置
// 定义每个第三方聚合器的连接和行为配置
type ProviderConfig struct {
	Name            string          `json:"name"`             // 聚合器名称
	DisplayName     string          `json:"display_name"`     // 显示名称
	BaseURL         string          `json:"base_url"`         // API基础URL
	APIKey          string          `json:"api_key"`          // API密钥
	Timeout         time.Duration   `json:"timeout"`          // 请求超时时间
	RetryCount      int             `json:"retry_count"`      // 重试次数
	Priority        int             `json:"priority"`         // 优先级(1最高)
	Weight          decimal.Decimal `json:"weight"`           // 权重系数
	IsActive        bool            `json:"is_active"`        // 是否启用
	SupportedChains []uint          `json:"supported_chains"` // 支持的链ID列表

	// 性能统计
	SuccessRate     decimal.Decimal `json:"success_rate"`      // 成功率
	AvgResponseTime time.Duration   `json:"avg_response_time"` // 平均响应时间
	LastHealthCheck time.Time       `json:"last_health_check"` // 最后健康检查
}

// ========================================
// 聚合策略配置
// ========================================

// AggregationStrategy 聚合策略配置
// 定义智能路由的决策算法和时间窗口配置
type AggregationStrategy struct {
	// 时间窗口配置
	MinWaitTime      time.Duration `json:"min_wait_time"`      // 最小等待时间
	MaxWaitTime      time.Duration `json:"max_wait_time"`      // 最大等待时间
	FastResponseTime time.Duration `json:"fast_response_time"` // 快速响应阈值
	EmergencyTimeout time.Duration `json:"emergency_timeout"`  // 紧急超时时间

	// 质量控制配置
	MinConfidence      decimal.Decimal `json:"min_confidence"`      // 最低置信度要求
	MinProviders       int             `json:"min_providers"`       // 最少响应聚合器数
	PreferredProviders int             `json:"preferred_providers"` // 理想响应聚合器数
	OptimalProviders   int             `json:"optimal_providers"`   // 最优响应聚合器数

	// 决策权重配置
	TimeWeight       decimal.Decimal `json:"time_weight"`       // 时间因素权重
	ConfidenceWeight decimal.Decimal `json:"confidence_weight"` // 置信度权重
	ProviderWeight   decimal.Decimal `json:"provider_weight"`   // 聚合器数量权重
	MarketWeight     decimal.Decimal `json:"market_weight"`     // 市场条件权重

	// 决策阈值
	CompositeScoreThreshold decimal.Decimal `json:"composite_score_threshold"` // 综合评分阈值
}

// ========================================
// 缓存相关类型
// ========================================

// CacheKey 缓存键结构
// 用于生成标准化的缓存键
type CacheKey struct {
	FromToken string `json:"from_token"` // 源代币地址
	ToToken   string `json:"to_token"`   // 目标代币地址
	AmountIn  string `json:"amount_in"`  // 输入数量
	ChainID   uint   `json:"chain_id"`   // 链ID
	Slippage  string `json:"slippage"`   // 滑点
}

// CacheEntry 缓存条目
// 存储在Redis中的缓存数据结构
type CacheEntry struct {
	QuoteResponse QuoteResponse `json:"quote_response"` // 报价响应
	CreatedAt     time.Time     `json:"created_at"`     // 创建时间
	ExpiresAt     time.Time     `json:"expires_at"`     // 过期时间
	HitCount      int           `json:"hit_count"`      // 命中次数
}

// ========================================
// 统计和监控类型
// ========================================

// ProviderMetrics 聚合器性能指标
// 用于监控和评估聚合器的表现
type ProviderMetrics struct {
	Provider        string          `json:"provider"`          // 聚合器名称
	TotalRequests   int64           `json:"total_requests"`    // 总请求数
	SuccessRequests int64           `json:"success_requests"`  // 成功请求数
	FailedRequests  int64           `json:"failed_requests"`   // 失败请求数
	SuccessRate     decimal.Decimal `json:"success_rate"`      // 成功率
	AvgResponseTime time.Duration   `json:"avg_response_time"` // 平均响应时间
	MinResponseTime time.Duration   `json:"min_response_time"` // 最小响应时间
	MaxResponseTime time.Duration   `json:"max_response_time"` // 最大响应时间
	TotalVolume     decimal.Decimal `json:"total_volume"`      // 总交易量
	BestQuoteCount  int64           `json:"best_quote_count"`  // 最优报价次数
	LastUpdated     time.Time       `json:"last_updated"`      // 最后更新时间
}

// SystemMetrics 系统整体指标
// 智能路由服务的整体性能指标
type SystemMetrics struct {
	TotalRequests      int64           `json:"total_requests"`       // 总请求数
	CacheHitRate       decimal.Decimal `json:"cache_hit_rate"`       // 缓存命中率
	AvgAggregationTime time.Duration   `json:"avg_aggregation_time"` // 平均聚合时间
	ActiveProviders    int             `json:"active_providers"`     // 活跃聚合器数
	Uptime             time.Duration   `json:"uptime"`               // 服务运行时间
	LastRestart        time.Time       `json:"last_restart"`         // 最后重启时间
	MemoryUsage        uint64          `json:"memory_usage"`         // 内存使用量
	GoroutineCount     int             `json:"goroutine_count"`      // 协程数量
}

// ========================================
// 错误类型定义
// ========================================

// RouterError 路由服务错误
type RouterError struct {
	Code      string                 `json:"code"`               // 错误代码
	Message   string                 `json:"message"`            // 错误消息
	Details   map[string]interface{} `json:"details"`            // 错误详情
	Provider  string                 `json:"provider,omitempty"` // 相关聚合器
	Timestamp time.Time              `json:"timestamp"`          // 错误时间
}

func (e *RouterError) Error() string {
	return e.Message
}

// 预定义错误代码
const (
	ErrCodeInvalidRequest        = "INVALID_REQUEST"        // 无效请求
	ErrCodeProviderTimeout       = "PROVIDER_TIMEOUT"       // 聚合器超时
	ErrCodeProviderError         = "PROVIDER_ERROR"         // 聚合器错误
	ErrCodeNoValidQuotes         = "NO_VALID_QUOTES"        // 无有效报价
	ErrCodeCacheError            = "CACHE_ERROR"            // 缓存错误
	ErrCodeInternalError         = "INTERNAL_ERROR"         // 内部错误
	ErrCodeRateLimitExceeded     = "RATE_LIMIT_EXCEEDED"    // 频率限制
	ErrCodeUnsupportedChain      = "UNSUPPORTED_CHAIN"      // 不支持的链
	ErrCodeInsufficientLiquidity = "INSUFFICIENT_LIQUIDITY" // 流动性不足
)

// ========================================
// 配置类型
// ========================================

// Config 智能路由服务配置
type Config struct {
	Server     ServerConfig        `json:"server"`     // 服务器配置
	Redis      RedisConfig         `json:"redis"`      // Redis配置
	Providers  []ProviderConfig    `json:"providers"`  // 聚合器配置
	Strategy   AggregationStrategy `json:"strategy"`   // 聚合策略
	Cache      CacheConfig         `json:"cache"`      // 缓存配置
	Monitoring MonitoringConfig    `json:"monitoring"` // 监控配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port        int    `json:"port"`        // 监听端口
	Environment string `json:"environment"` // 运行环境
	LogLevel    string `json:"log_level"`   // 日志级别
	Debug       bool   `json:"debug"`       // 调试模式
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `json:"host"`      // Redis主机
	Port     int    `json:"port"`      // Redis端口
	Password string `json:"password"`  // Redis密码
	DB       int    `json:"db"`        // 数据库编号
	PoolSize int    `json:"pool_size"` // 连接池大小
}

// CacheConfig 缓存配置
type CacheConfig struct {
	DefaultTTL      time.Duration `json:"default_ttl"`      // 默认TTL
	MaxEntries      int           `json:"max_entries"`      // 最大缓存条目
	CleanupInterval time.Duration `json:"cleanup_interval"` // 清理间隔
	PrefixKey       string        `json:"prefix_key"`       // 缓存键前缀
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	MetricsEnabled  bool          `json:"metrics_enabled"`   // 是否启用指标
	MetricsPath     string        `json:"metrics_path"`      // 指标路径
	HealthCheckPath string        `json:"health_check_path"` // 健康检查路径
	StatsInterval   time.Duration `json:"stats_interval"`    // 统计间隔
}

// ========================================
// HTTP响应类型
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

// HealthCheckResponse 健康检查响应
type HealthCheckResponse struct {
	Status    string                    `json:"status"`    // 整体状态
	Timestamp time.Time                 `json:"timestamp"` // 检查时间
	Version   string                    `json:"version"`   // 服务版本
	Uptime    time.Duration             `json:"uptime"`    // 运行时间
	Providers map[string]ProviderHealth `json:"providers"` // 聚合器健康状态
	Cache     CacheHealth               `json:"cache"`     // 缓存健康状态
	System    SystemHealth              `json:"system"`    // 系统健康状态
}

// ProviderHealth 聚合器健康状态
type ProviderHealth struct {
	Status       string          `json:"status"`                  // healthy, unhealthy, degraded
	LastChecked  time.Time       `json:"last_checked"`            // 最后检查时间
	ResponseTime time.Duration   `json:"response_time"`           // 响应时间
	SuccessRate  decimal.Decimal `json:"success_rate"`            // 成功率
	ErrorMessage string          `json:"error_message,omitempty"` // 错误信息
}

// CacheHealth 缓存健康状态
type CacheHealth struct {
	Status       string          `json:"status"`        // 连接状态
	HitRate      decimal.Decimal `json:"hit_rate"`      // 命中率
	TotalEntries int             `json:"total_entries"` // 总条目数
	MemoryUsage  uint64          `json:"memory_usage"`  // 内存使用量
}

// SystemHealth 系统健康状态
type SystemHealth struct {
	CPUUsage       decimal.Decimal `json:"cpu_usage"`       // CPU使用率
	MemoryUsage    uint64          `json:"memory_usage"`    // 内存使用量
	GoroutineCount int             `json:"goroutine_count"` // 协程数量
	OpenFDs        int             `json:"open_fds"`        // 打开的文件描述符
}

// ========================================
// 常量定义
// ========================================

// 支持的聚合器列表
const (
	Provider1inch    = "1inch"    // 1inch聚合器
	ProviderParaswap = "paraswap" // ParaSwap聚合器
	Provider0x       = "0x"       // 0x Protocol
	ProviderCowswap  = "cowswap"  // CoW Protocol
)

// 聚合策略类型
const (
	StrategyProgressive = "progressive" // 渐进式响应策略
	StrategyComplete    = "complete"    // 等待所有响应策略
	StrategyFast        = "fast"        // 快速响应策略
)

// 缓存键前缀
const (
	CacheKeyQuote   = "quote:"   // 报价缓存前缀
	CacheKeyMetrics = "metrics:" // 指标缓存前缀
	CacheKeyHealth  = "health:"  // 健康状态缓存前缀
)

// HTTP状态码
const (
	StatusHealthy   = "healthy"   // 健康状态
	StatusUnhealthy = "unhealthy" // 不健康状态
	StatusDegraded  = "degraded"  // 降级状态
)
