// Package models 定义与数据库表对应的GORM模型
// 所有模型严格按照数据库schema.sql设计，确保数据一致性
// 采用GORM标签进行数据库映射，支持自动迁移和关系定义
package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ========================================
// 基础模型定义
// ========================================

// BaseModel 基础模型，包含通用字段
// 大多数业务模型使用此结构体，确保数据表的一致性
// 注意：移除软删除功能，因为数据库设计中没有deleted_at列
type BaseModel struct {
	ID        uint      `gorm:"primarykey" json:"id"`             // 主键ID
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"` // 创建时间，自动设置
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"` // 更新时间，自动更新
}

// SimpleBaseModel 简化基础模型，不包含updated_at字段
// 用于那些数据库表设计中没有updated_at列的模型
type SimpleBaseModel struct {
	ID        uint      `gorm:"primarykey" json:"id"`             // 主键ID
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"` // 创建时间，自动设置
}

// ========================================
// 区块链相关模型
// ========================================

// Chain 区块链网络模型
// 对应数据库表: chains
// 存储支持的区块链网络信息，如以太坊、Polygon等
type Chain struct {
	BaseModel
	ChainID      uint   `gorm:"uniqueIndex;not null" json:"chain_id"`  // 链ID (1=Ethereum, 137=Polygon等)
	Name         string `gorm:"size:50;not null" json:"name"`          // 链名称 (ethereum, polygon等)
	DisplayName  string `gorm:"size:50;not null" json:"display_name"`  // 显示名称 (Ethereum, Polygon等)
	Symbol       string `gorm:"size:10;not null" json:"symbol"`        // 原生代币符号 (ETH, MATIC等)
	RPCURL       string `gorm:"size:500;not null" json:"rpc_url"`      // RPC节点URL
	ExplorerURL  string `gorm:"size:500;not null" json:"explorer_url"` // 区块浏览器URL
	IsTestnet    bool   `gorm:"default:false" json:"is_testnet"`       // 是否为测试网
	IsActive     bool   `gorm:"default:true" json:"is_active"`         // 是否启用
	GasPriceGwei uint   `gorm:"default:20" json:"gas_price_gwei"`      // 建议Gas价格(Gwei)
	BlockTimeSec uint   `gorm:"default:15" json:"block_time_sec"`      // 平均出块时间(秒)

	// 关系定义
	Tokens              []Token               `gorm:"foreignKey:ChainID" json:"tokens,omitempty"`                 // 一对多：链拥有多个代币
	QuoteRequests       []QuoteRequest        `gorm:"foreignKey:ChainID" json:"quote_requests,omitempty"`         // 一对多：链有多个报价请求
	Transactions        []Transaction         `gorm:"foreignKey:ChainID" json:"transactions,omitempty"`           // 一对多：链有多个交易
	AggregatorChains    []AggregatorChain     `gorm:"foreignKey:ChainID" json:"aggregator_chains,omitempty"`      // 一对多：聚合器链关系
	TokenPairStatsDaily []TokenPairStatsDaily `gorm:"foreignKey:ChainID" json:"token_pair_stats_daily,omitempty"` // 一对多：代币对统计
}

// ========================================
// 聚合器相关模型
// ========================================

// Aggregator 第三方聚合器模型
// 对应数据库表: aggregators
// 存储如1inch、Paraswap等第三方聚合器配置信息
type Aggregator struct {
	BaseModel
	Name            string          `gorm:"size:50;uniqueIndex;not null" json:"name"`             // 聚合器名称 (1inch, paraswap等)
	DisplayName     string          `gorm:"size:100;not null" json:"display_name"`                // 显示名称
	APIURL          string          `gorm:"size:500;not null" json:"api_url"`                     // API基础URL
	APIKey          string          `gorm:"size:255" json:"-"`                                    // API密钥 (敏感信息不序列化)
	LogoURL         string          `gorm:"size:500" json:"logo_url"`                             // Logo URL
	IsActive        bool            `gorm:"default:true" json:"is_active"`                        // 是否启用
	Priority        int             `gorm:"default:1" json:"priority"`                            // 优先级 (1最高)
	TimeoutMS       int             `gorm:"default:5000" json:"timeout_ms"`                       // 超时时间毫秒
	RetryCount      int             `gorm:"default:3" json:"retry_count"`                         // 重试次数
	SuccessRate     decimal.Decimal `gorm:"type:decimal(5,4);default:1.0000" json:"success_rate"` // 成功率统计
	AvgResponseMS   int             `gorm:"default:1000" json:"avg_response_ms"`                  // 平均响应时间
	LastHealthCheck *time.Time      `gorm:"null" json:"last_health_check"`                        // 最后健康检查时间

	// 关系定义
	AggregatorChains      []AggregatorChain       `gorm:"foreignKey:AggregatorID" json:"aggregator_chains,omitempty"`       // 一对多：聚合器链关系
	QuoteResponses        []QuoteResponse         `gorm:"foreignKey:AggregatorID" json:"quote_responses,omitempty"`         // 一对多：报价响应
	Transactions          []Transaction           `gorm:"foreignKey:AggregatorID" json:"transactions,omitempty"`            // 一对多：交易记录
	AggregatorStatsHourly []AggregatorStatsHourly `gorm:"foreignKey:AggregatorID" json:"aggregator_stats_hourly,omitempty"` // 一对多：性能统计
}

// AggregatorChain 聚合器支持的链关系模型
// 对应数据库表: aggregator_chains
// 表示特定聚合器在特定链上的配置信息
type AggregatorChain struct {
	BaseModel
	AggregatorID  uint            `json:"aggregator_id"`  // 聚合器ID
	ChainID       uint            `json:"chain_id"`       // 链ID
	IsActive      bool            `json:"is_active"`      // 是否启用
	GasMultiplier decimal.Decimal `json:"gas_multiplier"` // Gas费用乘数

	// 关系定义（仅用于查询，约束由数据库schema定义）
	Aggregator Aggregator `gorm:"foreignKey:AggregatorID" json:"aggregator,omitempty"` // 多对一：属于某个聚合器
	Chain      Chain      `gorm:"foreignKey:ChainID" json:"chain,omitempty"`           // 多对一：属于某个链
}

// ========================================
// 代币相关模型
// ========================================

// Token 代币信息模型
// 对应数据库表: tokens
// 存储支持的代币信息，包括价格、市值等市场数据
type Token struct {
	BaseModel
	ChainID         uint             `gorm:"not null;index:idx_chain_address" json:"chain_id"`                 // 链ID
	ContractAddress string           `gorm:"size:42;not null;index:idx_chain_address" json:"contract_address"` // 合约地址
	Symbol          string           `gorm:"size:20;not null;index" json:"symbol"`                             // 代币符号
	Name            string           `gorm:"size:100;not null" json:"name"`                                    // 代币全名
	Decimals        int              `gorm:"not null" json:"decimals"`                                         // 小数位数
	LogoURL         string           `gorm:"size:500" json:"logo_url"`                                         // 代币图标URL
	CoingeckoID     string           `gorm:"size:100" json:"coingecko_id"`                                     // CoinGecko ID
	CoinmarketcapID *int             `gorm:"null" json:"coinmarketcap_id"`                                     // CoinMarketCap ID
	IsNative        bool             `gorm:"default:false" json:"is_native"`                                   // 是否为原生代币
	IsStable        bool             `gorm:"default:false" json:"is_stable"`                                   // 是否为稳定币
	IsVerified      bool             `gorm:"default:false" json:"is_verified"`                                 // 是否已验证
	IsActive        bool             `gorm:"default:true;index" json:"is_active"`                              // 是否启用
	DailyVolumeUSD  *decimal.Decimal `gorm:"type:decimal(20,2);null" json:"daily_volume_usd"`                  // 24小时交易量USD
	MarketCapUSD    *decimal.Decimal `gorm:"type:decimal(20,2);null" json:"market_cap_usd"`                    // 市值USD
	PriceUSD        *decimal.Decimal `gorm:"type:decimal(20,8);null" json:"price_usd"`                         // 当前价格USD
	PriceUpdatedAt  *time.Time       `gorm:"null;index" json:"price_updated_at"`                               // 价格更新时间

	// 关系定义
	Chain             Chain          `gorm:"foreignKey:ChainID" json:"chain,omitempty"`                   // 多对一：属于某个链
	FromQuoteRequests []QuoteRequest `gorm:"foreignKey:FromTokenID" json:"from_quote_requests,omitempty"` // 一对多：作为源代币的报价请求
	ToQuoteRequests   []QuoteRequest `gorm:"foreignKey:ToTokenID" json:"to_quote_requests,omitempty"`     // 一对多：作为目标代币的报价请求
	FromTransactions  []Transaction  `gorm:"foreignKey:FromTokenID" json:"from_transactions,omitempty"`   // 一对多：作为源代币的交易
	ToTransactions    []Transaction  `gorm:"foreignKey:ToTokenID" json:"to_transactions,omitempty"`       // 一对多：作为目标代币的交易
}

// ========================================
// 用户相关模型
// ========================================

// User 用户基础信息模型
// 对应数据库表: users
// 存储用户基本信息，采用钱包地址作为唯一标识
type User struct {
	BaseModel
	WalletAddress     string     `gorm:"size:42;uniqueIndex;not null" json:"wallet_address"` // 钱包地址 (以太坊地址格式)
	Nonce             string     `gorm:"size:64;not null;default:''" json:"-"`               // 登录随机数 (敏感信息)
	Username          string     `gorm:"size:50" json:"username"`                            // 可选用户名
	Email             string     `gorm:"size:255" json:"email"`                              // 可选邮箱
	AvatarURL         string     `gorm:"size:500" json:"avatar_url"`                         // 头像URL
	PreferredLanguage string     `gorm:"size:10;default:'en'" json:"preferred_language"`     // 首选语言
	Timezone          string     `gorm:"size:50;default:'UTC'" json:"timezone"`              // 时区
	IsActive          bool       `gorm:"default:true" json:"is_active"`                      // 账户状态
	LastLoginAt       *time.Time `gorm:"null;index" json:"last_login_at"`                    // 最后登录时间

	// 关系定义
	Preferences   *UserPreferences `gorm:"foreignKey:UserID" json:"preferences,omitempty"`    // 一对一：用户偏好
	QuoteRequests []QuoteRequest   `gorm:"foreignKey:UserID" json:"quote_requests,omitempty"` // 一对多：报价请求
	Transactions  []Transaction    `gorm:"foreignKey:UserID" json:"transactions,omitempty"`   // 一对多：交易记录
}

// UserPreferences 用户偏好设置模型
// 对应数据库表: user_preferences
// 存储用户的个性化设置，如默认滑点、Gas速度偏好等
type UserPreferences struct {
	BaseModel
	UserID              uint            `gorm:"uniqueIndex;not null" json:"user_id"`                     // 用户ID
	DefaultSlippage     decimal.Decimal `gorm:"type:decimal(5,4);default:0.005" json:"default_slippage"` // 默认滑点 0.5%
	PreferredGasSpeed   string          `gorm:"size:20;default:'standard'" json:"preferred_gas_speed"`   // fast/standard/slow
	AutoApproveTokens   bool            `gorm:"default:false" json:"auto_approve_tokens"`                // 是否自动批准代币授权
	ShowTestTokens      bool            `gorm:"default:false" json:"show_test_tokens"`                   // 是否显示测试代币
	NotificationEmail   bool            `gorm:"default:true" json:"notification_email"`                  // 邮件通知
	NotificationBrowser bool            `gorm:"default:true" json:"notification_browser"`                // 浏览器通知
	PrivacyAnalytics    bool            `gorm:"default:true" json:"privacy_analytics"`                   // 是否允许分析

	// 关系定义
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"` // 多对一：属于某个用户
}

// ========================================
// 报价相关模型
// ========================================

// QuoteRequest 报价请求记录模型
// 对应数据库表: quote_requests
// 记录用户的报价请求参数和聚合结果
type QuoteRequest struct {
	SimpleBaseModel
	RequestID        string           `gorm:"size:64;uniqueIndex;not null" json:"request_id"`     // 唯一请求ID
	UserID           *uint            `gorm:"null;index" json:"user_id"`                          // 用户ID (可为空，支持匿名)
	ChainID          uint             `gorm:"not null;index" json:"chain_id"`                     // 链ID
	FromTokenID      uint             `gorm:"not null;index:idx_token_pair" json:"from_token_id"` // 源代币ID
	ToTokenID        uint             `gorm:"not null;index:idx_token_pair" json:"to_token_id"`   // 目标代币ID
	AmountIn         decimal.Decimal  `gorm:"type:decimal(78,0);not null" json:"amount_in"`       // 输入数量 (wei格式)
	Slippage         decimal.Decimal  `gorm:"type:decimal(5,4);not null" json:"slippage"`         // 滑点设置
	UserAddress      string           `gorm:"size:42" json:"user_address"`                        // 用户钱包地址
	IPAddress        string           `gorm:"size:45" json:"ip_address"`                          // 用户IP地址
	UserAgent        string           `gorm:"type:text" json:"user_agent"`                        // 用户代理
	RequestSource    string           `gorm:"size:20;default:'web'" json:"request_source"`        // 请求来源: web/mobile/api
	AggregatorCount  int              `gorm:"default:0" json:"aggregator_count"`                  // 调用的聚合器数量
	SuccessCount     int              `gorm:"default:0" json:"success_count"`                     // 成功响应数量
	BestAggregatorID *uint            `gorm:"null" json:"best_aggregator_id"`                     // 最佳聚合器ID
	BestAmountOut    *decimal.Decimal `gorm:"type:decimal(78,0);null" json:"best_amount_out"`     // 最佳输出数量
	BestGasEstimate  *uint64          `gorm:"null" json:"best_gas_estimate"`                      // 最佳Gas估算
	BestPriceImpact  *decimal.Decimal `gorm:"type:decimal(8,6);null" json:"best_price_impact"`    // 最佳价格冲击
	TotalDurationMS  *int             `gorm:"null" json:"total_duration_ms"`                      // 总耗时毫秒
	CacheHit         bool             `gorm:"default:false" json:"cache_hit"`                     // 是否命中缓存
	Status           string           `gorm:"size:20;default:'pending';index" json:"status"`      // 状态: pending/completed/failed
	ErrorMessage     string           `gorm:"type:text" json:"error_message"`                     // 错误信息
	CompletedAt      *time.Time       `gorm:"null" json:"completed_at"`                           // 完成时间

	// 关系定义
	User           *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`                      // 多对一：属于某个用户 (可为空)
	Chain          Chain           `gorm:"foreignKey:ChainID" json:"chain,omitempty"`                    // 多对一：属于某个链
	FromToken      Token           `gorm:"foreignKey:FromTokenID" json:"from_token,omitempty"`           // 多对一：源代币
	ToToken        Token           `gorm:"foreignKey:ToTokenID" json:"to_token,omitempty"`               // 多对一：目标代币
	BestAggregator *Aggregator     `gorm:"foreignKey:BestAggregatorID" json:"best_aggregator,omitempty"` // 多对一：最佳聚合器
	QuoteResponses []QuoteResponse `gorm:"foreignKey:QuoteRequestID" json:"quote_responses,omitempty"`   // 一对多：报价响应
	Transaction    *Transaction    `gorm:"foreignKey:QuoteRequestID" json:"transaction,omitempty"`       // 一对一：关联交易
}

// QuoteResponse 聚合器响应详情模型
// 对应数据库表: quote_responses
// 记录每个聚合器对特定报价请求的响应详情
type QuoteResponse struct {
	SimpleBaseModel
	QuoteRequestID  uint             `gorm:"not null;index" json:"quote_request_id"`         // 报价请求ID
	AggregatorID    uint             `gorm:"not null;index" json:"aggregator_id"`            // 聚合器ID
	ResponseTimeMS  int              `gorm:"not null" json:"response_time_ms"`               // 响应时间毫秒
	Success         bool             `gorm:"not null;index" json:"success"`                  // 是否成功
	AmountOut       *decimal.Decimal `gorm:"type:decimal(78,0);null" json:"amount_out"`      // 输出数量
	GasEstimate     *uint64          `gorm:"null" json:"gas_estimate"`                       // Gas估算
	PriceImpact     *decimal.Decimal `gorm:"type:decimal(8,6);null" json:"price_impact"`     // 价格冲击
	ConfidenceScore *decimal.Decimal `gorm:"type:decimal(3,2);null" json:"confidence_score"` // 置信度分数
	PriceRank       *int             `gorm:"null" json:"price_rank"`                         // 价格排名
	RouteData       string           `gorm:"type:jsonb" json:"route_data"`                   // 交易路径数据 (JSONB)
	RawResponse     string           `gorm:"type:jsonb" json:"raw_response"`                 // 原始响应数据 (JSONB)
	ErrorCode       string           `gorm:"size:50" json:"error_code"`                      // 错误代码
	ErrorMessage    string           `gorm:"type:text" json:"error_message"`                 // 错误信息

	// 关系定义
	QuoteRequest QuoteRequest `gorm:"foreignKey:QuoteRequestID" json:"quote_request,omitempty"` // 多对一：属于某个报价请求
	Aggregator   Aggregator   `gorm:"foreignKey:AggregatorID" json:"aggregator,omitempty"`      // 多对一：属于某个聚合器
}

// ========================================
// 交易相关模型
// ========================================

// Transaction 交易记录模型
// 对应数据库表: transactions
// 记录用户执行的交易的完整信息，从创建到确认的全生命周期
type Transaction struct {
	BaseModel
	UserID         *uint  `gorm:"null;index" json:"user_id"`                          // 用户ID
	QuoteRequestID *uint  `gorm:"null" json:"quote_request_id"`                       // 关联的报价请求ID
	TxHash         string `gorm:"size:66;uniqueIndex" json:"tx_hash"`                 // 交易哈希
	ChainID        uint   `gorm:"not null;index" json:"chain_id"`                     // 链ID
	FromTokenID    uint   `gorm:"not null;index:idx_token_pair" json:"from_token_id"` // 源代币ID
	ToTokenID      uint   `gorm:"not null;index:idx_token_pair" json:"to_token_id"`   // 目标代币ID
	AggregatorID   uint   `gorm:"not null;index" json:"aggregator_id"`                // 聚合器ID

	// 交易参数
	AmountIn          decimal.Decimal  `gorm:"type:decimal(78,0);not null" json:"amount_in"`           // 实际输入数量
	AmountOutExpected decimal.Decimal  `gorm:"type:decimal(78,0);not null" json:"amount_out_expected"` // 预期输出数量
	AmountOutActual   *decimal.Decimal `gorm:"type:decimal(78,0);null" json:"amount_out_actual"`       // 实际输出数量
	SlippageSet       decimal.Decimal  `gorm:"type:decimal(5,4);not null" json:"slippage_set"`         // 设置的滑点
	SlippageActual    *decimal.Decimal `gorm:"type:decimal(8,6);null" json:"slippage_actual"`          // 实际滑点

	// Gas相关
	GasLimit  *uint64          `gorm:"null" json:"gas_limit"`                      // Gas限制
	GasUsed   *uint64          `gorm:"null" json:"gas_used"`                       // 实际使用Gas
	GasPrice  *decimal.Decimal `gorm:"type:decimal(20,0);null" json:"gas_price"`   // Gas价格 (wei)
	GasFeeETH *decimal.Decimal `gorm:"type:decimal(18,9);null" json:"gas_fee_eth"` // Gas费用 (ETH)
	GasFeeUSD *decimal.Decimal `gorm:"type:decimal(10,2);null" json:"gas_fee_usd"` // Gas费用 (USD)

	// 价格相关
	PriceImpact  decimal.Decimal  `gorm:"type:decimal(8,6);not null" json:"price_impact"` // 价格冲击
	ExchangeRate *decimal.Decimal `gorm:"type:decimal(36,18);null" json:"exchange_rate"`  // 汇率 (amount_out/amount_in)
	AmountInUSD  *decimal.Decimal `gorm:"type:decimal(12,2);null" json:"amount_in_usd"`   // 输入金额USD
	AmountOutUSD *decimal.Decimal `gorm:"type:decimal(12,2);null" json:"amount_out_usd"`  // 输出金额USD

	// 交易状态和时间
	Status            string     `gorm:"size:20;default:'pending';index" json:"status"` // 状态: pending/confirmed/failed/cancelled
	UserAddress       string     `gorm:"size:42;not null" json:"user_address"`          // 用户钱包地址
	ToAddress         string     `gorm:"size:42" json:"to_address"`                     // 目标合约地址
	BlockNumber       *uint64    `gorm:"null;index" json:"block_number"`                // 区块号
	BlockTimestamp    *time.Time `gorm:"null" json:"block_timestamp"`                   // 区块时间戳
	ConfirmationCount int        `gorm:"default:0" json:"confirmation_count"`           // 确认数

	// 元数据
	RouteData   string     `gorm:"type:jsonb" json:"route_data"`  // 交易路径详情 (JSONB)
	ErrorReason string     `gorm:"type:text" json:"error_reason"` // 失败原因
	ConfirmedAt *time.Time `gorm:"null" json:"confirmed_at"`      // 确认时间

	// 关系定义
	User         *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`                  // 多对一：属于某个用户
	QuoteRequest *QuoteRequest `gorm:"foreignKey:QuoteRequestID" json:"quote_request,omitempty"` // 多对一：关联报价请求
	Chain        Chain         `gorm:"foreignKey:ChainID" json:"chain,omitempty"`                // 多对一：属于某个链
	FromToken    Token         `gorm:"foreignKey:FromTokenID" json:"from_token,omitempty"`       // 多对一：源代币
	ToToken      Token         `gorm:"foreignKey:ToTokenID" json:"to_token,omitempty"`           // 多对一：目标代币
	Aggregator   Aggregator    `gorm:"foreignKey:AggregatorID" json:"aggregator,omitempty"`      // 多对一：使用的聚合器
}

// ========================================
// 统计相关模型
// ========================================

// AggregatorStatsHourly 聚合器性能统计模型 (按小时聚合)
// 对应数据库表: aggregator_stats_hourly
// 存储聚合器的小时级性能统计数据，用于监控和分析
type AggregatorStatsHourly struct {
	SimpleBaseModel
	AggregatorID       uint            `gorm:"not null;index:idx_agg_hour" json:"aggregator_id"`     // 聚合器ID
	HourTimestamp      time.Time       `gorm:"not null;index:idx_agg_hour" json:"hour_timestamp"`    // 小时时间戳
	TotalRequests      int             `gorm:"default:0" json:"total_requests"`                      // 总请求数
	SuccessfulRequests int             `gorm:"default:0" json:"successful_requests"`                 // 成功请求数
	FailedRequests     int             `gorm:"default:0" json:"failed_requests"`                     // 失败请求数
	AvgResponseTime    int             `gorm:"default:0" json:"avg_response_time"`                   // 平均响应时间ms
	MinResponseTime    int             `gorm:"default:0" json:"min_response_time"`                   // 最小响应时间ms
	MaxResponseTime    int             `gorm:"default:0" json:"max_response_time"`                   // 最大响应时间ms
	TotalVolumeUSD     decimal.Decimal `gorm:"type:decimal(15,2);default:0" json:"total_volume_usd"` // 总交易量USD
	BestQuotesCount    int             `gorm:"default:0" json:"best_quotes_count"`                   // 最优报价次数

	// 关系定义
	Aggregator Aggregator `gorm:"foreignKey:AggregatorID" json:"aggregator,omitempty"` // 多对一：属于某个聚合器
}

// TokenPairStatsDaily 代币对交易统计模型 (按日聚合)
// 对应数据库表: token_pair_stats_daily
// 存储代币对的日级交易统计数据，用于业务分析
type TokenPairStatsDaily struct {
	SimpleBaseModel
	FromTokenID      uint             `gorm:"not null;index:idx_pair_date" json:"from_token_id"`     // 源代币ID
	ToTokenID        uint             `gorm:"not null;index:idx_pair_date" json:"to_token_id"`       // 目标代币ID
	ChainID          uint             `gorm:"not null;index:idx_pair_date" json:"chain_id"`          // 链ID
	Date             time.Time        `gorm:"type:date;not null;index:idx_pair_date" json:"date"`    // 日期
	TransactionCount int              `gorm:"default:0" json:"transaction_count"`                    // 交易次数
	TotalVolumeFrom  decimal.Decimal  `gorm:"type:decimal(78,0);default:0" json:"total_volume_from"` // 总输入量
	TotalVolumeTo    decimal.Decimal  `gorm:"type:decimal(78,0);default:0" json:"total_volume_to"`   // 总输出量
	TotalVolumeUSD   decimal.Decimal  `gorm:"type:decimal(15,2);default:0" json:"total_volume_usd"`  // 总交易量USD
	AvgPriceImpact   *decimal.Decimal `gorm:"type:decimal(8,6);null" json:"avg_price_impact"`        // 平均价格冲击
	AvgGasFeeUSD     *decimal.Decimal `gorm:"type:decimal(10,2);null" json:"avg_gas_fee_usd"`        // 平均Gas费用USD
	UniqueUsers      int              `gorm:"default:0" json:"unique_users"`                         // 独立用户数

	// 关系定义
	FromToken Token `gorm:"foreignKey:FromTokenID" json:"from_token,omitempty"` // 多对一：源代币
	ToToken   Token `gorm:"foreignKey:ToTokenID" json:"to_token,omitempty"`     // 多对一：目标代币
	Chain     Chain `gorm:"foreignKey:ChainID" json:"chain,omitempty"`          // 多对一：链
}

// SystemMetrics 系统性能监控模型
// 对应数据库表: system_metrics
// 存储系统运行时的各种指标数据
type SystemMetrics struct {
	SimpleBaseModel
	MetricName  string          `gorm:"size:100;not null;index:idx_metric_time" json:"metric_name"`       // 指标名称
	MetricValue decimal.Decimal `gorm:"type:decimal(15,6);not null" json:"metric_value"`                  // 指标值
	MetricUnit  string          `gorm:"size:20" json:"metric_unit"`                                       // 单位
	Tags        string          `gorm:"type:jsonb" json:"tags"`                                           // 标签数据 (JSONB)
	Timestamp   time.Time       `gorm:"default:CURRENT_TIMESTAMP;index:idx_metric_time" json:"timestamp"` // 时间戳
}

// ========================================
// 数据库索引和约束定义
// ========================================

// 在GORM中定义复合索引和唯一约束
func init() {
	// 这些索引会在自动迁移时创建
	// 对应SQL中的CREATE INDEX语句
}

// ========================================
// 模型方法定义
// ========================================

// TableName 为所有模型指定表名，确保与数据库schema一致

func (Chain) TableName() string {
	return "chains"
}

func (Aggregator) TableName() string {
	return "aggregators"
}

func (AggregatorChain) TableName() string {
	return "aggregator_chains"
}

func (Token) TableName() string {
	return "tokens"
}

func (User) TableName() string {
	return "users"
}

func (UserPreferences) TableName() string {
	return "user_preferences"
}

func (QuoteRequest) TableName() string {
	return "quote_requests"
}

func (QuoteResponse) TableName() string {
	return "quote_responses"
}

func (Transaction) TableName() string {
	return "transactions"
}

func (AggregatorStatsHourly) TableName() string {
	return "aggregator_stats_hourly"
}

func (TokenPairStatsDaily) TableName() string {
	return "token_pair_stats_daily"
}

func (SystemMetrics) TableName() string {
	return "system_metrics"
}

// 模型验证方法
// 在保存前进行数据验证，确保数据完整性

// BeforeCreate GORM钩子：创建前验证
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 验证钱包地址格式
	if len(u.WalletAddress) != 42 || u.WalletAddress[:2] != "0x" {
		return fmt.Errorf("无效的钱包地址格式: %s", u.WalletAddress)
	}
	return nil
}

// BeforeCreate 代币创建前验证
func (t *Token) BeforeCreate(tx *gorm.DB) error {
	// 验证合约地址格式
	if len(t.ContractAddress) != 42 || t.ContractAddress[:2] != "0x" {
		return fmt.Errorf("无效的合约地址格式: %s", t.ContractAddress)
	}

	// 验证小数位数范围
	if t.Decimals < 0 || t.Decimals > 30 {
		return fmt.Errorf("代币小数位数必须在0-30之间: %d", t.Decimals)
	}

	return nil
}

// BeforeCreate 交易创建前验证
func (tr *Transaction) BeforeCreate(tx *gorm.DB) error {
	// 验证交易哈希格式
	if tr.TxHash != "" && (len(tr.TxHash) != 66 || tr.TxHash[:2] != "0x") {
		return fmt.Errorf("无效的交易哈希格式: %s", tr.TxHash)
	}

	// 验证用户地址格式
	if len(tr.UserAddress) != 42 || tr.UserAddress[:2] != "0x" {
		return fmt.Errorf("无效的用户地址格式: %s", tr.UserAddress)
	}

	return nil
}
