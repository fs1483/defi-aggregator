// Package types 定义业务逻辑服务中使用的所有数据类型
// 包含请求响应结构体、业务实体、枚举类型等
// 遵循领域驱动设计(DDD)原则，确保类型安全和业务语义清晰
package types

import (
	"time"

	"github.com/shopspring/decimal"
)

// ========================================
// 通用类型定义
// ========================================

// APIResponse 统一的API响应格式
// 遵循RESTful API设计规范，提供一致的响应结构
type APIResponse struct {
	Success   bool        `json:"success"`           // 请求是否成功
	Data      interface{} `json:"data,omitempty"`    // 响应数据，成功时包含
	Error     *APIError   `json:"error,omitempty"`   // 错误信息，失败时包含
	Message   string      `json:"message,omitempty"` // 可选的消息
	Meta      *Meta       `json:"meta,omitempty"`    // 元数据，如分页信息
	Timestamp int64       `json:"timestamp"`         // 响应时间戳
	RequestID string      `json:"request_id"`        // 请求ID，用于追踪
}

// APIError API错误信息结构
type APIError struct {
	Code    string                 `json:"code"`              // 错误代码
	Message string                 `json:"message"`           // 错误消息
	Details map[string]interface{} `json:"details,omitempty"` // 详细错误信息
}

// Meta 响应元数据
// 主要用于分页等场景
type Meta struct {
	Page       int `json:"page,omitempty"`        // 当前页码
	PageSize   int `json:"page_size,omitempty"`   // 每页大小
	Total      int `json:"total,omitempty"`       // 总记录数
	TotalPages int `json:"total_pages,omitempty"` // 总页数
}

// PaginationRequest 分页请求参数
type PaginationRequest struct {
	Page     int    `form:"page" validate:"min=1"`              // 页码，从1开始
	PageSize int    `form:"page_size" validate:"min=1,max=100"` // 每页大小
	SortBy   string `form:"sort_by" validate:"omitempty,alpha"` // 排序字段
	SortDesc bool   `form:"sort_desc"`                          // 是否降序
}

// ========================================
// 用户相关类型
// ========================================

// UserLoginRequest 用户登录请求
type UserLoginRequest struct {
	WalletAddress string `json:"wallet_address" validate:"required,eth_addr"` // 钱包地址
	Signature     string `json:"signature" validate:"required"`               // 签名
	Message       string `json:"message" validate:"required"`                 // 签名的消息
	Nonce         string `json:"nonce" validate:"required"`                   // 随机数
}

// UserLoginResponse 用户登录响应
type UserLoginResponse struct {
	AccessToken  string    `json:"access_token"`  // 访问令牌
	RefreshToken string    `json:"refresh_token"` // 刷新令牌
	ExpiresIn    int64     `json:"expires_in"`    // 令牌过期时间（秒）
	User         *UserInfo `json:"user"`          // 用户信息
}

// UserInfo 用户基础信息
type UserInfo struct {
	ID            uint       `json:"id"`                      // 用户ID
	WalletAddress string     `json:"wallet_address"`          // 钱包地址
	Username      string     `json:"username,omitempty"`      // 用户名
	Email         string     `json:"email,omitempty"`         // 邮箱
	AvatarURL     string     `json:"avatar_url,omitempty"`    // 头像URL
	PreferredLang string     `json:"preferred_language"`      // 首选语言
	Timezone      string     `json:"timezone"`                // 时区
	IsActive      bool       `json:"is_active"`               // 账户状态
	CreatedAt     time.Time  `json:"created_at"`              // 创建时间
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"` // 最后登录时间
}

// UserPreferences 用户偏好设置
type UserPreferences struct {
	DefaultSlippage     decimal.Decimal `json:"default_slippage"`     // 默认滑点
	PreferredGasSpeed   string          `json:"preferred_gas_speed"`  // 偏好Gas速度
	AutoApproveTokens   bool            `json:"auto_approve_tokens"`  // 自动批准代币
	ShowTestTokens      bool            `json:"show_test_tokens"`     // 显示测试代币
	NotificationEmail   bool            `json:"notification_email"`   // 邮件通知
	NotificationBrowser bool            `json:"notification_browser"` // 浏览器通知
	PrivacyAnalytics    bool            `json:"privacy_analytics"`    // 隐私分析
}

// UpdateUserRequest 更新用户信息请求
type UpdateUserRequest struct {
	Username      *string `json:"username" validate:"omitempty,min=3,max=50"`    // 用户名
	Email         *string `json:"email" validate:"omitempty,email"`              // 邮箱
	AvatarURL     *string `json:"avatar_url" validate:"omitempty,url"`           // 头像URL
	PreferredLang *string `json:"preferred_language" validate:"omitempty,len=2"` // 首选语言
	Timezone      *string `json:"timezone" validate:"omitempty"`                 // 时区
}

// ========================================
// 代币相关类型
// ========================================

// TokenInfo 代币信息
type TokenInfo struct {
	ID              uint             `json:"id"`                         // 代币ID
	ChainID         uint             `json:"chain_id"`                   // 链ID
	ContractAddress string           `json:"contract_address"`           // 合约地址
	Symbol          string           `json:"symbol"`                     // 代币符号
	Name            string           `json:"name"`                       // 代币名称
	Decimals        int              `json:"decimals"`                   // 小数位数
	LogoURL         string           `json:"logo_url,omitempty"`         // 图标URL
	IsNative        bool             `json:"is_native"`                  // 是否为原生代币
	IsStable        bool             `json:"is_stable"`                  // 是否为稳定币
	IsVerified      bool             `json:"is_verified"`                // 是否已验证
	IsActive        bool             `json:"is_active"`                  // 是否启用
	PriceUSD        *decimal.Decimal `json:"price_usd,omitempty"`        // 当前价格USD
	DailyVolumeUSD  *decimal.Decimal `json:"daily_volume_usd,omitempty"` // 24小时交易量
	MarketCapUSD    *decimal.Decimal `json:"market_cap_usd,omitempty"`   // 市值
	PriceUpdatedAt  *time.Time       `json:"price_updated_at,omitempty"` // 价格更新时间
}

// TokenListRequest 代币列表请求
type TokenListRequest struct {
	PaginationRequest
	ChainID    *uint   `form:"chain_id"`                          // 链ID过滤
	IsActive   *bool   `form:"is_active"`                         // 状态过滤
	IsVerified *bool   `form:"is_verified"`                       // 验证状态过滤
	Search     *string `form:"search" validate:"omitempty,min=1"` // 搜索关键词
}

// TokenInfoWithChain 包含链信息的代币信息
// 用于前端代币选择器，提供代币和链的完整信息
type TokenInfoWithChain struct {
	TokenInfo `json:",inline"` // 内嵌代币信息
	Chain     ChainInfo        `json:"chain"` // 关联的链信息
}

// ========================================
// 报价相关类型
// ========================================

// QuoteRequest 报价请求
type QuoteRequest struct {
	FromTokenID uint            `json:"from_token_id" validate:"required"`          // 源代币ID
	ToTokenID   uint            `json:"to_token_id" validate:"required"`            // 目标代币ID
	AmountIn    decimal.Decimal `json:"amount_in" validate:"required,gt=0"`         // 输入数量(wei)
	Slippage    decimal.Decimal `json:"slippage" validate:"required,gte=0,lte=0.5"` // 滑点
	UserAddress *string         `json:"user_address" validate:"omitempty,eth_addr"` // 用户地址
	ChainID     uint            `json:"chain_id" validate:"required"`               // 链ID
}

// QuoteResponse 报价响应
type QuoteResponse struct {
	RequestID       string          `json:"request_id"`        // 请求ID
	FromToken       TokenInfo       `json:"from_token"`        // 源代币信息
	ToToken         TokenInfo       `json:"to_token"`          // 目标代币信息
	AmountIn        decimal.Decimal `json:"amount_in"`         // 输入数量
	AmountOut       decimal.Decimal `json:"amount_out"`        // 输出数量
	BestAggregator  string          `json:"best_aggregator"`   // 最佳聚合器
	GasEstimate     uint64          `json:"gas_estimate"`      // Gas估算
	PriceImpact     decimal.Decimal `json:"price_impact"`      // 价格冲击
	ExchangeRate    decimal.Decimal `json:"exchange_rate"`     // 汇率
	Route           []RouteInfo     `json:"route,omitempty"`   // 交易路径
	ValidUntil      time.Time       `json:"valid_until"`       // 有效期
	TotalDurationMS int             `json:"total_duration_ms"` // 总耗时
	CacheHit        bool            `json:"cache_hit"`         // 是否命中缓存
}

// RouteInfo 交易路径信息
type RouteInfo struct {
	Protocol   string          `json:"protocol"`   // 协议名称
	Percentage decimal.Decimal `json:"percentage"` // 比例
}

// ========================================
// 交易相关类型
// ========================================

// SwapRequest 交易请求
type SwapRequest struct {
	RequestID   string          `json:"request_id" validate:"required"`             // 报价请求ID
	UserAddress string          `json:"user_address" validate:"required,eth_addr"`  // 用户地址
	Slippage    decimal.Decimal `json:"slippage" validate:"required,gte=0,lte=0.5"` // 滑点
	Deadline    *time.Time      `json:"deadline,omitempty"`                         // 交易截止时间
}

// SwapResponse 交易响应
type SwapResponse struct {
	TransactionID uint    `json:"transaction_id"`  // 交易ID
	To            string  `json:"to"`              // 目标合约地址
	Data          string  `json:"data"`            // 交易数据(calldata)
	Value         string  `json:"value"`           // 交易价值
	GasLimit      string  `json:"gas_limit"`       // Gas限制
	GasPrice      string  `json:"gas_price"`       // Gas价格
	Nonce         *uint64 `json:"nonce,omitempty"` // 交易nonce
}

// TransactionInfo 交易信息
type TransactionInfo struct {
	ID                uint              `json:"id"`                          // 交易ID
	TxHash            *string           `json:"tx_hash,omitempty"`           // 交易哈希
	UserID            uint              `json:"user_id"`                     // 用户ID
	ChainID           uint              `json:"chain_id"`                    // 链ID
	FromToken         TokenInfo         `json:"from_token"`                  // 源代币
	ToToken           TokenInfo         `json:"to_token"`                    // 目标代币
	Aggregator        string            `json:"aggregator"`                  // 使用的聚合器
	AmountIn          decimal.Decimal   `json:"amount_in"`                   // 输入数量
	AmountOutExpected decimal.Decimal   `json:"amount_out_expected"`         // 预期输出
	AmountOutActual   *decimal.Decimal  `json:"amount_out_actual,omitempty"` // 实际输出
	SlippageSet       decimal.Decimal   `json:"slippage_set"`                // 设置滑点
	SlippageActual    *decimal.Decimal  `json:"slippage_actual,omitempty"`   // 实际滑点
	GasLimit          *uint64           `json:"gas_limit,omitempty"`         // Gas限制
	GasUsed           *uint64           `json:"gas_used,omitempty"`          // 使用Gas
	GasPrice          *decimal.Decimal  `json:"gas_price,omitempty"`         // Gas价格
	GasFeeETH         *decimal.Decimal  `json:"gas_fee_eth,omitempty"`       // Gas费用ETH
	GasFeeUSD         *decimal.Decimal  `json:"gas_fee_usd,omitempty"`       // Gas费用USD
	PriceImpact       decimal.Decimal   `json:"price_impact"`                // 价格冲击
	Status            TransactionStatus `json:"status"`                      // 交易状态
	UserAddress       string            `json:"user_address"`                // 用户地址
	BlockNumber       *uint64           `json:"block_number,omitempty"`      // 区块号
	BlockTimestamp    *time.Time        `json:"block_timestamp,omitempty"`   // 区块时间
	CreatedAt         time.Time         `json:"created_at"`                  // 创建时间
	ConfirmedAt       *time.Time        `json:"confirmed_at,omitempty"`      // 确认时间
}

// TransactionStatus 交易状态枚举
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"   // 待处理
	TransactionStatusConfirmed TransactionStatus = "confirmed" // 已确认
	TransactionStatusFailed    TransactionStatus = "failed"    // 失败
	TransactionStatusCancelled TransactionStatus = "cancelled" // 已取消
)

// TransactionListRequest 交易列表请求
type TransactionListRequest struct {
	PaginationRequest
	UserID   *uint              `form:"user_id"`   // 用户ID过滤
	Status   *TransactionStatus `form:"status"`    // 状态过滤
	ChainID  *uint              `form:"chain_id"`  // 链ID过滤
	FromDate *time.Time         `form:"from_date"` // 开始日期
	ToDate   *time.Time         `form:"to_date"`   // 结束日期
}

// ========================================
// 统计相关类型
// ========================================

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	TotalTransactions      int             `json:"total_transactions"`      // 总交易数
	SuccessfulTransactions int             `json:"successful_transactions"` // 成功交易数
	TotalVolumeUSD         decimal.Decimal `json:"total_volume_usd"`        // 总交易量USD
	TotalGasFeeUSD         decimal.Decimal `json:"total_gas_fee_usd"`       // 总Gas费用USD
	AvgPriceImpact         decimal.Decimal `json:"avg_price_impact"`        // 平均价格冲击
	LastTransactionAt      *time.Time      `json:"last_transaction_at"`     // 最后交易时间
}

// SystemStatsResponse 系统统计响应
type SystemStatsResponse struct {
	TotalUsers        int             `json:"total_users"`        // 总用户数
	ActiveUsers24h    int             `json:"active_users_24h"`   // 24小时活跃用户
	TotalTransactions int             `json:"total_transactions"` // 总交易数
	TotalVolumeUSD    decimal.Decimal `json:"total_volume_usd"`   // 总交易量USD
	SupportedTokens   int             `json:"supported_tokens"`   // 支持代币数
	SupportedChains   int             `json:"supported_chains"`   // 支持链数
}

// ========================================
// 健康检查类型
// ========================================

// HealthCheckResponse 健康检查响应
type HealthCheckResponse struct {
	Status    string                   `json:"status"`    // 整体状态: healthy, unhealthy, degraded
	Timestamp time.Time                `json:"timestamp"` // 检查时间
	Version   string                   `json:"version"`   // 服务版本
	Uptime    time.Duration            `json:"uptime"`    // 运行时间
	Services  map[string]ServiceHealth `json:"services"`  // 各服务状态
}

// ServiceHealth 服务健康状态
type ServiceHealth struct {
	Status       string                 `json:"status"`                  // healthy, unhealthy
	LastChecked  time.Time              `json:"last_checked"`            // 最后检查时间
	ResponseTime time.Duration          `json:"response_time,omitempty"` // 响应时间
	Error        string                 `json:"error,omitempty"`         // 错误信息
	Details      map[string]interface{} `json:"details,omitempty"`       // 详细信息
}

// ========================================
// 常量定义
// ========================================

// 错误代码定义
const (
	ErrCodeValidation   = "VALIDATION_ERROR"    // 参数验证错误
	ErrCodeUnauthorized = "UNAUTHORIZED"        // 未授权
	ErrCodeForbidden    = "FORBIDDEN"           // 禁止访问
	ErrCodeNotFound     = "NOT_FOUND"           // 资源不存在
	ErrCodeConflict     = "CONFLICT"            // 资源冲突
	ErrCodeInternal     = "INTERNAL_ERROR"      // 内部错误
	ErrCodeExternalAPI  = "EXTERNAL_API_ERROR"  // 外部API错误
	ErrCodeDatabase     = "DATABASE_ERROR"      // 数据库错误
	ErrCodeCache        = "CACHE_ERROR"         // 缓存错误
	ErrCodeRateLimit    = "RATE_LIMIT_EXCEEDED" // 频率限制
)

// Gas速度枚举
const (
	GasSpeedSlow     = "slow"     // 慢速
	GasSpeedStandard = "standard" // 标准
	GasSpeedFast     = "fast"     // 快速
)

// 支持的语言代码
const (
	LangEnglish  = "en" // 英语
	LangChinese  = "zh" // 中文
	LangJapanese = "ja" // 日语
	LangKorean   = "ko" // 韩语
)

// 缓存键前缀
const (
	CacheKeyUserPrefix  = "user:"  // 用户缓存前缀
	CacheKeyTokenPrefix = "token:" // 代币缓存前缀
	CacheKeyQuotePrefix = "quote:" // 报价缓存前缀
	CacheKeyStatsPrefix = "stats:" // 统计缓存前缀
)

// JWT Claims键
const (
	JWTClaimUserID     = "user_id"     // 用户ID
	JWTClaimWalletAddr = "wallet_addr" // 钱包地址
	JWTClaimRole       = "role"        // 用户角色
)

// 请求头键名
const (
	HeaderRequestID    = "X-Request-ID"    // 请求ID头
	HeaderUserAgent    = "User-Agent"      // 用户代理头
	HeaderRealIP       = "X-Real-IP"       // 真实IP头
	HeaderForwardedFor = "X-Forwarded-For" // 转发IP头
)

// ========================================
// 补充类型定义
// ========================================

// ChainInfo 区块链信息
type ChainInfo struct {
	ID           uint   `json:"id"`
	ChainID      uint   `json:"chain_id"`
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	Symbol       string `json:"symbol"`
	IsTestnet    bool   `json:"is_testnet"`
	IsActive     bool   `json:"is_active"`
	GasPriceGwei uint   `json:"gas_price_gwei"`
	BlockTimeSec uint   `json:"block_time_sec"`
}

// 为services包需要的类型补充定义（临时）
type AggregatorStats struct{}
type AggregatorRanking struct{}
type AggregatorComparison struct{}
type TokenPairStats struct{}
type PopularTokenPair struct{}
type VolumeData struct{}
type UserAnalytics struct{}
type UserTrend struct{}
type PerformanceMetrics struct{}
type LatencyStats struct{}
type TransactionCost struct{}
type SlippageAnalysis struct{}
type AggregatorQuoteResponse struct{}
type QuoteComparison struct{}
type PriceImpactAnalysis struct{}
