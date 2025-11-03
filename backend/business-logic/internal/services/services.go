// Package services 提供业务逻辑服务层
// 封装复杂的业务逻辑，协调多个Repository的操作，实现业务规则
// 采用Service模式分离业务逻辑和数据访问，提高代码的可测试性和可维护性
package services

import (
	"fmt"

	"defi-aggregator/business-logic/internal/repository"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"

	"github.com/sirupsen/logrus"
)

// Services 业务逻辑服务集合
// 包含所有业务域的服务接口，便于依赖注入和统一管理
type Services struct {
	User   UserService   // 用户业务服务
	Auth   AuthService   // 认证业务服务
	Token  TokenService  // 代币业务服务
	Chain  ChainService  // 区块链业务服务
	Quote  QuoteService  // 报价业务服务
	Swap   SwapService   // 交易业务服务
	Stats  StatsService  // 统计业务服务
	Health HealthService // 健康检查服务
}

// New 创建新的业务服务实例
// 初始化所有业务服务，注入依赖的Repository和配置
// 参数:
//   - repos: 数据访问层实例
//   - cfg: 应用配置
//   - logger: 日志记录器
//
// 返回:
//   - *Services: 完整的业务服务集合
func New(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) *Services {
	return &Services{
		User:   NewUserService(repos, cfg, logger),
		Auth:   NewAuthService(repos, cfg, logger),
		Token:  NewTokenService(repos, cfg, logger),
		Chain:  NewChainService(repos, cfg, logger),
		Quote:  NewQuoteService(repos, cfg, logger),
		Swap:   NewSwapService(repos, cfg, logger),
		Stats:  NewStatsService(repos, cfg, logger),
		Health: NewHealthService(repos, cfg, logger),
	}
}

// ========================================
// 临时构造函数实现（待实现具体服务）
// ========================================

func NewSwapService(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) SwapService {
	return &swapService{repos: repos, cfg: cfg, logger: logger}
}

func NewStatsService(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) StatsService {
	return &statsService{repos: repos, cfg: cfg, logger: logger}
}

func NewHealthService(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) HealthService {
	return &healthService{repos: repos, cfg: cfg, logger: logger}
}

// 临时服务实现结构体
type swapService struct {
	repos  *repository.Repositories
	cfg    *config.Config
	logger *logrus.Logger
}

type statsService struct {
	repos  *repository.Repositories
	cfg    *config.Config
	logger *logrus.Logger
}

type healthService struct {
	repos  *repository.Repositories
	cfg    *config.Config
	logger *logrus.Logger
}

// ========================================
// 用户业务服务接口
// ========================================

// UserService 用户业务服务接口
// 定义用户相关的业务操作，包括用户管理、偏好设置等
type UserService interface {
	// 用户基础操作
	GetProfile(userID uint) (*types.UserInfo, error)               // 获取用户资料
	UpdateProfile(userID uint, req *types.UpdateUserRequest) error // 更新用户资料
	DeactivateUser(userID uint) error                              // 停用用户账户

	// 用户偏好设置
	GetPreferences(userID uint) (*types.UserPreferences, error)        // 获取用户偏好
	UpdatePreferences(userID uint, prefs *types.UserPreferences) error // 更新用户偏好
	ResetPreferences(userID uint) error                                // 重置用户偏好为默认值

	// 用户统计
	GetUserStats(userID uint) (*types.UserStatsResponse, error)                       // 获取用户统计信息
	GetUserList(req *types.PaginationRequest) ([]*types.UserInfo, *types.Meta, error) // 获取用户列表（管理员）

	// 用户活动
	UpdateLastActivity(userID uint) error       // 更新用户最后活动时间
	GetActiveUsers() ([]*types.UserInfo, error) // 获取活跃用户列表
}

// ========================================
// 认证业务服务接口
// ========================================

// AuthService 认证业务服务接口
// 处理Web3钱包认证、JWT令牌管理等安全相关功能
type AuthService interface {
	// Web3钱包认证
	GenerateNonce(walletAddress string) (string, error)                            // 生成登录随机数
	VerifySignature(req *types.UserLoginRequest) (*types.UserLoginResponse, error) // 验证签名并登录

	// JWT令牌管理
	GenerateTokens(userID uint, walletAddress string) (accessToken, refreshToken string, err error) // 生成访问令牌
	RefreshToken(refreshToken string) (newAccessToken string, err error)                            // 刷新访问令牌
	RevokeToken(userID uint, tokenType string) error                                                // 撤销令牌

	// 会话管理
	ValidateSession(userID uint) error   // 验证会话有效性
	LogoutUser(userID uint) error        // 用户登出
	LogoutAllSessions(userID uint) error // 登出所有会话
}

// ========================================
// 代币业务服务接口
// ========================================

// TokenService 代币业务服务接口
// 管理支持的代币信息、价格更新、代币验证等功能
type TokenService interface {
	// 代币基础操作
	GetTokens(req *types.TokenListRequest) ([]*types.TokenInfo, *types.Meta, error) // 获取代币列表
	GetTokenByID(id uint) (*types.TokenInfo, error)                                 // 获取代币详情
	GetTokenByAddress(chainID uint, address string) (*types.TokenInfo, error)       // 根据合约地址获取代币

	// 代币查询（包含链信息）
	GetTokensWithChainInfo(req *types.TokenListRequest) ([]*types.TokenInfoWithChain, *types.Meta, error) // 获取代币列表（含链信息）

	// 代币搜索和筛选
	SearchTokens(query string) ([]*types.TokenInfo, error)     // 搜索代币
	GetTokensByChain(chainID uint) ([]*types.TokenInfo, error) // 获取指定链的代币
	GetVerifiedTokens() ([]*types.TokenInfo, error)            // 获取已验证代币
	GetPopularTokens(limit int) ([]*types.TokenInfo, error)    // 获取热门代币

	// 代币价格管理
	UpdateTokenPrice(tokenID uint, priceUSD string) error                     // 更新代币价格
	RefreshAllPrices() error                                                  // 刷新所有代币价格
	GetPriceHistory(tokenID uint, days int) ([]map[string]interface{}, error) // 获取价格历史

	// 代币管理（管理员功能）
	AddToken(tokenInfo *types.TokenInfo) error                          // 添加新代币
	UpdateTokenInfo(tokenID uint, updates map[string]interface{}) error // 更新代币信息
	VerifyToken(tokenID uint) error                                     // 验证代币
	DeactivateToken(tokenID uint) error                                 // 停用代币
}

// ========================================
// 区块链业务服务接口
// ========================================

// ChainService 区块链业务服务接口
// 管理支持的区块链网络信息和配置
type ChainService interface {
	// 链基础操作
	GetChains() ([]*types.ChainInfo, error)                   // 获取所有链
	GetChainByID(id uint) (*types.ChainInfo, error)           // 获取链详情
	GetChainByChainID(chainID uint) (*types.ChainInfo, error) // 根据真实链ID获取链信息
	GetActiveChains() ([]*types.ChainInfo, error)             // 获取活跃链

	// 链分类
	GetMainnetChains() ([]*types.ChainInfo, error) // 获取主网链
	GetTestnetChains() ([]*types.ChainInfo, error) // 获取测试网链

	// 链管理（管理员功能）
	AddChain(chainInfo *types.ChainInfo) error                      // 添加新链
	UpdateChain(chainID uint, updates map[string]interface{}) error // 更新链信息
	ActivateChain(chainID uint) error                               // 激活链
	DeactivateChain(chainID uint) error                             // 停用链

	// 链状态检查
	CheckChainHealth(chainID uint) error                  // 检查链健康状态
	UpdateGasPrice(chainID uint, gasPriceGwei uint) error // 更新Gas价格
}

// ========================================
// 报价业务服务接口
// ========================================

// QuoteService 报价业务服务接口
// 处理报价请求、聚合器调用、最优价格选择等核心业务逻辑
type QuoteService interface {
	// 报价核心操作
	GetQuote(req *types.QuoteRequest) (*types.QuoteResponse, error)                                          // 获取最优报价
	GetQuoteHistory(userID *uint, req *types.PaginationRequest) ([]*types.QuoteResponse, *types.Meta, error) // 获取报价历史

	// 报价详情
	GetQuoteDetails(requestID string) (*types.QuoteResponse, error)               // 获取报价详情
	GetQuoteResponses(requestID string) ([]*types.AggregatorQuoteResponse, error) // 获取所有聚合器响应

	// 缓存管理
	InvalidateQuoteCache(fromTokenID, toTokenID uint) error // 失效报价缓存
	GetCacheStats() (map[string]interface{}, error)         // 获取缓存统计

	// 报价分析
	CompareQuotes(requestID string) (*types.QuoteComparison, error)                     // 比较报价结果
	GetPriceImpactAnalysis(req *types.QuoteRequest) (*types.PriceImpactAnalysis, error) // 价格冲击分析
}

// ========================================
// 交易业务服务接口
// ========================================

// SwapService 交易业务服务接口
// 处理交易创建、状态跟踪、交易历史等功能
type SwapService interface {
	// 交易核心操作
	CreateSwap(req *types.SwapRequest) (*types.SwapResponse, error)                        // 创建交易
	GetSwapStatus(txHash string) (*types.TransactionInfo, error)                           // 获取交易状态
	UpdateSwapStatus(txHash string, status string, blockData map[string]interface{}) error // 更新交易状态

	// 交易历史
	GetTransactionHistory(userID *uint, req *types.TransactionListRequest) ([]*types.TransactionInfo, *types.Meta, error) // 获取交易历史
	GetTransactionDetails(id uint) (*types.TransactionInfo, error)                                                        // 获取交易详情

	// 交易分析
	CalculateTransactionCost(req *types.SwapRequest) (*types.TransactionCost, error) // 计算交易成本
	GetSlippageAnalysis(txHash string) (*types.SlippageAnalysis, error)              // 滑点分析

	// 交易管理
	CancelTransaction(id uint, userID uint) error                       // 取消交易
	RetryTransaction(id uint, userID uint) (*types.SwapResponse, error) // 重试交易
}

// 交易相关类型补充定义
type TransactionCost struct {
	GasEstimate    uint64            `json:"gas_estimate"`
	GasPriceGwei   uint64            `json:"gas_price_gwei"`
	GasCostETH     string            `json:"gas_cost_eth"`
	GasCostUSD     string            `json:"gas_cost_usd"`
	TotalCostUSD   string            `json:"total_cost_usd"`
	BreakdownByGas map[string]string `json:"breakdown_by_gas"`
}

type SlippageAnalysis struct {
	ExpectedSlippage  float64 `json:"expected_slippage"`
	ActualSlippage    float64 `json:"actual_slippage"`
	SlippageDiff      float64 `json:"slippage_diff"`
	IsWithinTolerance bool    `json:"is_within_tolerance"`
	Impact            string  `json:"impact"` // positive, negative, neutral
}

// ========================================
// 统计业务服务接口
// ========================================

// StatsService 统计业务服务接口
// 提供各种业务统计和分析功能
type StatsService interface {
	// 系统统计
	GetSystemStats() (*types.SystemStatsResponse, error) // 获取系统统计
	GetDashboardStats() (map[string]interface{}, error)  // 获取仪表板统计

	// 聚合器统计
	GetAggregatorStats(aggregatorID *uint, timeRange string) ([]*types.AggregatorStats, error) // 获取聚合器统计
	GetAggregatorRankings() ([]*types.AggregatorRanking, error)                                // 获取聚合器排名
	GetAggregatorComparison() (*types.AggregatorComparison, error)                             // 聚合器对比分析

	// 代币对统计
	GetTokenPairStats(fromTokenID, toTokenID uint, timeRange string) ([]*types.TokenPairStats, error) // 获取代币对统计
	GetPopularTokenPairs(limit int) ([]*types.PopularTokenPair, error)                                // 获取热门代币对
	GetTradingVolume(timeRange string) ([]*types.VolumeData, error)                                   // 获取交易量数据

	// 用户统计
	GetUserAnalytics(timeRange string) (*types.UserAnalytics, error) // 获取用户分析
	GetActiveUserTrends() ([]*types.UserTrend, error)                // 获取用户活跃趋势

	// 性能统计
	GetPerformanceMetrics() (*types.PerformanceMetrics, error) // 获取性能指标
	GetLatencyStats() ([]*types.LatencyStats, error)           // 获取延迟统计
}

// 统计相关类型补充定义
type AggregatorStats struct {
	AggregatorName  string  `json:"aggregator_name"`
	TotalRequests   int     `json:"total_requests"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime int     `json:"avg_response_time"`
	TotalVolume     string  `json:"total_volume"`
	BestQuoteCount  int     `json:"best_quote_count"`
}

type AggregatorRanking struct {
	Rank           int                    `json:"rank"`
	AggregatorName string                 `json:"aggregator_name"`
	Score          float64                `json:"score"`
	Metrics        map[string]interface{} `json:"metrics"`
}

type AggregatorComparison struct {
	Period      string                 `json:"period"`
	Aggregators []*AggregatorStats     `json:"aggregators"`
	Summary     map[string]interface{} `json:"summary"`
}

// ========================================
// 健康检查服务接口
// ========================================

// HealthService 健康检查服务接口
// 提供系统健康状态检查和监控功能
type HealthService interface {
	// 健康检查
	PerformHealthCheck() (*types.HealthCheckResponse, error) // 执行完整健康检查
	CheckDatabase() error                                    // 检查数据库连接
	CheckExternalServices() map[string]error                 // 检查外部服务
	CheckCache() error                                       // 检查缓存服务

	// 系统信息
	GetSystemInfo() (map[string]interface{}, error) // 获取系统信息
	GetMetrics() (map[string]interface{}, error)    // 获取监控指标

	// 服务状态
	GetServiceStatus() (map[string]*types.ServiceHealth, error)                // 获取各服务状态
	UpdateServiceHealth(serviceName string, health *types.ServiceHealth) error // 更新服务健康状态
}

// ========================================
// 通用业务接口定义
// ========================================

// BaseService 基础服务接口
// 定义所有服务都应该实现的基本方法
type BaseService interface {
	// 服务生命周期
	Initialize() error // 初始化服务
	Shutdown() error   // 关闭服务

	// 健康检查
	HealthCheck() error // 服务健康检查
}

// TransactionManager 事务管理接口
// 提供跨多个Repository的事务支持
type TransactionManager interface {
	// 事务执行
	WithTransaction(fn func() error) error    // 在事务中执行函数
	BeginTransaction() (interface{}, error)   // 开始事务
	CommitTransaction(tx interface{}) error   // 提交事务
	RollbackTransaction(tx interface{}) error // 回滚事务
}

// CacheManager 缓存管理接口
// 提供统一的缓存操作接口
type CacheManager interface {
	// 缓存操作
	Get(key string) (interface{}, error)              // 获取缓存
	Set(key string, value interface{}, ttl int) error // 设置缓存
	Delete(key string) error                          // 删除缓存
	Clear(pattern string) error                       // 清除匹配的缓存

	// 缓存统计
	GetStats() (map[string]interface{}, error) // 获取缓存统计
}

// EventPublisher 事件发布接口
// 支持业务事件的发布和订阅
type EventPublisher interface {
	// 事件发布
	PublishEvent(eventType string, data interface{}) error            // 发布事件
	PublishQuoteEvent(quote *types.QuoteResponse) error               // 发布报价事件
	PublishTransactionEvent(transaction *types.TransactionInfo) error // 发布交易事件

	// 事件订阅管理
	Subscribe(eventType string, handler func(interface{}) error) error // 订阅事件
	Unsubscribe(eventType string) error                                // 取消订阅
}

// ========================================
// 业务错误定义
// ========================================

// ServiceError 业务服务错误类型
type ServiceError struct {
	Code    string                 `json:"code"`    // 错误代码
	Message string                 `json:"message"` // 错误消息
	Details map[string]interface{} `json:"details"` // 错误详情
	Cause   error                  `json:"-"`       // 原始错误
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewServiceError 创建业务服务错误
func NewServiceError(code, message string, cause error) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Details: make(map[string]interface{}),
	}
}

// 预定义的业务错误
var (
	ErrInvalidInput       = &ServiceError{Code: "INVALID_INPUT", Message: "输入参数无效"}
	ErrUserNotFound       = &ServiceError{Code: "USER_NOT_FOUND", Message: "用户不存在"}
	ErrTokenNotFound      = &ServiceError{Code: "TOKEN_NOT_FOUND", Message: "代币不存在"}
	ErrInsufficientFunds  = &ServiceError{Code: "INSUFFICIENT_FUNDS", Message: "资金不足"}
	ErrQuoteExpired       = &ServiceError{Code: "QUOTE_EXPIRED", Message: "报价已过期"}
	ErrTransactionFailed  = &ServiceError{Code: "TRANSACTION_FAILED", Message: "交易失败"}
	ErrExternalAPIError   = &ServiceError{Code: "EXTERNAL_API_ERROR", Message: "外部API调用失败"}
	ErrServiceUnavailable = &ServiceError{Code: "SERVICE_UNAVAILABLE", Message: "服务暂时不可用"}
)
