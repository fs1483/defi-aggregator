// Package repository 提供数据访问层接口和实现
// 采用Repository模式封装数据库操作，提供清晰的数据访问接口
// 支持事务处理、分页查询、复杂条件查询等企业级数据访问需求
package repository

import (
	"fmt"

	"defi-aggregator/business-logic/internal/models"
	"defi-aggregator/business-logic/internal/types"

	"gorm.io/gorm"
)

// Repositories 数据访问层集合
// 包含所有业务实体的数据访问接口，便于依赖注入和测试
type Repositories struct {
	User         UserRepository         // 用户数据访问
	Token        TokenRepository        // 代币数据访问
	Chain        ChainRepository        // 区块链数据访问
	Aggregator   AggregatorRepository   // 聚合器数据访问
	QuoteRequest QuoteRequestRepository // 报价请求数据访问
	Transaction  TransactionRepository  // 交易数据访问
	Stats        StatsRepository        // 统计数据访问
}

// New 创建新的数据访问层实例
// 初始化所有Repository实现，注入数据库连接
func New(db *gorm.DB) *Repositories {
	return &Repositories{
		User:         NewUserRepository(db),
		Token:        NewTokenRepository(db),
		Chain:        NewChainRepository(db),
		Aggregator:   NewAggregatorRepository(db),
		QuoteRequest: NewQuoteRequestRepository(db),
		Transaction:  NewTransactionRepository(db),
		Stats:        NewStatsRepository(db),
	}
}

// ========================================
// 用户相关数据访问接口
// ========================================

// UserRepository 用户数据访问接口
// 定义用户相关的数据库操作方法
type UserRepository interface {
	// 基础CRUD操作
	Create(user *models.User) error                          // 创建用户
	GetByID(id uint) (*models.User, error)                   // 根据ID获取用户
	GetByWalletAddress(address string) (*models.User, error) // 根据钱包地址获取用户
	Update(user *models.User) error                          // 更新用户信息
	Delete(id uint) error                                    // 删除用户

	// 用户偏好操作
	GetPreferences(userID uint) (*models.UserPreferences, error) // 获取用户偏好
	UpdatePreferences(preferences *models.UserPreferences) error // 更新用户偏好
	CreatePreferences(preferences *models.UserPreferences) error // 创建用户偏好

	// 查询操作
	List(req *types.PaginationRequest) ([]*models.User, int64, error) // 分页获取用户列表
	GetActiveUsers() ([]*models.User, error)                          // 获取活跃用户
	UpdateLastLogin(userID uint) error                                // 更新最后登录时间

	// 认证相关
	UpdateNonce(userID uint, nonce string) error // 更新登录随机数
}

// ========================================
// 代币相关数据访问接口
// ========================================

// TokenRepository 代币数据访问接口
type TokenRepository interface {
	// 基础CRUD操作
	Create(token *models.Token) error                                         // 创建代币
	GetByID(id uint) (*models.Token, error)                                   // 根据ID获取代币
	GetByContractAddress(chainID uint, address string) (*models.Token, error) // 根据合约地址获取代币
	Update(token *models.Token) error                                         // 更新代币信息
	Delete(id uint) error                                                     // 删除代币

	// 查询操作
	List(req *types.TokenListRequest) ([]*models.Token, int64, error) // 分页获取代币列表
	GetByChainID(chainID uint) ([]*models.Token, error)               // 获取指定链的代币
	GetActiveTokens() ([]*models.Token, error)                        // 获取活跃代币
	GetVerifiedTokens() ([]*models.Token, error)                      // 获取已验证代币
	Search(query string) ([]*models.Token, error)                     // 搜索代币

	// 价格相关
	UpdatePrice(tokenID uint, priceUSD string) error       // 更新代币价格
	GetTokensWithOutdatedPrices() ([]*models.Token, error) // 获取价格过期的代币
}

// ========================================
// 区块链相关数据访问接口
// ========================================

// ChainRepository 区块链数据访问接口
type ChainRepository interface {
	// 基础CRUD操作
	Create(chain *models.Chain) error                 // 创建区块链
	GetByID(id uint) (*models.Chain, error)           // 根据ID获取区块链
	GetByChainID(chainID uint) (*models.Chain, error) // 根据链ID获取区块链
	Update(chain *models.Chain) error                 // 更新区块链信息
	Delete(id uint) error                             // 删除区块链

	// 查询操作
	List() ([]*models.Chain, error)             // 获取所有区块链
	GetActiveChains() ([]*models.Chain, error)  // 获取活跃区块链
	GetMainnetChains() ([]*models.Chain, error) // 获取主网链
	GetTestnetChains() ([]*models.Chain, error) // 获取测试网链
}

// ========================================
// 聚合器相关数据访问接口
// ========================================

// AggregatorRepository 聚合器数据访问接口
type AggregatorRepository interface {
	// 基础CRUD操作
	Create(aggregator *models.Aggregator) error        // 创建聚合器
	GetByID(id uint) (*models.Aggregator, error)       // 根据ID获取聚合器
	GetByName(name string) (*models.Aggregator, error) // 根据名称获取聚合器
	Update(aggregator *models.Aggregator) error        // 更新聚合器信息
	Delete(id uint) error                              // 删除聚合器

	// 查询操作
	List() ([]*models.Aggregator, error)                     // 获取所有聚合器
	GetActiveAggregators() ([]*models.Aggregator, error)     // 获取活跃聚合器
	GetByChainID(chainID uint) ([]*models.Aggregator, error) // 获取支持指定链的聚合器
	GetSortedByPriority() ([]*models.Aggregator, error)      // 按优先级排序获取聚合器

	// 性能统计
	UpdateStats(aggregatorID uint, stats map[string]interface{}) error // 更新统计信息
	UpdateHealthCheck(aggregatorID uint) error                         // 更新健康检查时间
}

// ========================================
// 报价请求相关数据访问接口
// ========================================

// QuoteRequestRepository 报价请求数据访问接口
type QuoteRequestRepository interface {
	// 基础CRUD操作
	Create(request *models.QuoteRequest) error                     // 创建报价请求
	GetByID(id uint) (*models.QuoteRequest, error)                 // 根据ID获取报价请求
	GetByRequestID(requestID string) (*models.QuoteRequest, error) // 根据请求ID获取报价请求
	Update(request *models.QuoteRequest) error                     // 更新报价请求
	Delete(id uint) error                                          // 删除报价请求

	// 查询操作
	List(req *types.PaginationRequest) ([]*models.QuoteRequest, int64, error)                     // 分页获取报价请求
	GetByUserID(userID uint, req *types.PaginationRequest) ([]*models.QuoteRequest, int64, error) // 获取用户的报价请求
	GetByTokenPair(fromTokenID, toTokenID uint) ([]*models.QuoteRequest, error)                   // 获取代币对的报价请求
	GetRecentRequests(limit int) ([]*models.QuoteRequest, error)                                  // 获取最近的报价请求

	// 报价响应操作
	CreateResponse(response *models.QuoteResponse) error          // 创建报价响应
	GetResponses(requestID uint) ([]*models.QuoteResponse, error) // 获取报价响应

	// 统计查询
	GetTotalCount() (int64, error)     // 获取总请求数
	GetCompletedCount() (int64, error) // 获取完成请求数
	GetSuccessRate() (float64, error)  // 获取成功率
}

// ========================================
// 交易相关数据访问接口
// ========================================

// TransactionRepository 交易数据访问接口
type TransactionRepository interface {
	// 基础CRUD操作
	Create(transaction *models.Transaction) error           // 创建交易
	GetByID(id uint) (*models.Transaction, error)           // 根据ID获取交易
	GetByTxHash(txHash string) (*models.Transaction, error) // 根据交易哈希获取交易
	Update(transaction *models.Transaction) error           // 更新交易信息
	Delete(id uint) error                                   // 删除交易

	// 查询操作
	List(req *types.TransactionListRequest) ([]*models.Transaction, int64, error)                // 分页获取交易列表
	GetByUserID(userID uint, req *types.PaginationRequest) ([]*models.Transaction, int64, error) // 获取用户交易
	GetByStatus(status string) ([]*models.Transaction, error)                                    // 根据状态获取交易
	GetPendingTransactions() ([]*models.Transaction, error)                                      // 获取待处理交易
	GetRecentTransactions(limit int) ([]*models.Transaction, error)                              // 获取最近交易

	// 统计操作
	GetUserStats(userID uint) (*types.UserStatsResponse, error)             // 获取用户统计
	GetVolumeByDateRange(from, to string) ([]map[string]interface{}, error) // 获取时间范围内的交易量
	GetTopTokenPairs(limit int) ([]map[string]interface{}, error)           // 获取热门交易对
}

// ========================================
// 统计相关数据访问接口
// ========================================

// StatsRepository 统计数据访问接口
type StatsRepository interface {
	// 系统统计
	GetSystemStats() (*types.SystemStatsResponse, error) // 获取系统统计
	GetDailyActiveUsers(date string) (int, error)        // 获取日活跃用户数
	GetTotalVolume() (string, error)                     // 获取总交易量

	// 聚合器统计
	CreateAggregatorStats(stats *models.AggregatorStatsHourly) error                          // 创建聚合器统计
	GetAggregatorStats(aggregatorID uint, hours int) ([]*models.AggregatorStatsHourly, error) // 获取聚合器统计
	GetAggregatorRankings() ([]map[string]interface{}, error)                                 // 获取聚合器排名

	// 代币对统计
	CreateTokenPairStats(stats *models.TokenPairStatsDaily) error                                   // 创建代币对统计
	GetTokenPairStats(fromTokenID, toTokenID uint, days int) ([]*models.TokenPairStatsDaily, error) // 获取代币对统计
	GetPopularTokenPairs(limit int) ([]map[string]interface{}, error)                               // 获取热门代币对

	// 系统指标
	CreateSystemMetric(metric *models.SystemMetrics) error                          // 创建系统指标
	GetSystemMetrics(metricName string, hours int) ([]*models.SystemMetrics, error) // 获取系统指标
	GetLatestMetrics() (map[string]interface{}, error)                              // 获取最新指标
}

// ========================================
// 通用接口定义
// ========================================

// BaseRepository 基础Repository接口
// 定义所有Repository都应该实现的基本方法
type BaseRepository interface {
	// 事务支持
	WithTx(tx *gorm.DB) interface{} // 返回使用指定事务的Repository实例

	// 健康检查
	HealthCheck() error // 检查Repository健康状态
}

// Paginator 分页接口
// 提供统一的分页查询功能
type Paginator interface {
	Paginate(query *gorm.DB, page, pageSize int) *gorm.DB // 应用分页条件
	Count(query *gorm.DB) (int64, error)                  // 获取总记录数
}

// CacheInterface 缓存接口
// 为Repository提供缓存功能支持
type CacheInterface interface {
	Get(key string) (interface{}, error)              // 获取缓存
	Set(key string, value interface{}, ttl int) error // 设置缓存
	Delete(key string) error                          // 删除缓存
	Clear() error                                     // 清空缓存
}

// QueryBuilder 查询构建器接口
// 提供灵活的查询条件构建功能
type QueryBuilder interface {
	Where(query interface{}, args ...interface{}) QueryBuilder // 添加WHERE条件
	Order(value interface{}) QueryBuilder                      // 添加排序
	Limit(limit int) QueryBuilder                              // 添加LIMIT
	Offset(offset int) QueryBuilder                            // 添加OFFSET
	Preload(query string, args ...interface{}) QueryBuilder    // 添加预加载
	Build() *gorm.DB                                           // 构建查询
}

// ========================================
// 错误定义
// ========================================

// Repository层错误定义
var (
	ErrRecordNotFound     = gorm.ErrRecordNotFound     // 记录不存在
	ErrDuplicateKey       = gorm.ErrDuplicatedKey      // 重复键
	ErrInvalidData        = gorm.ErrInvalidData        // 无效数据
	ErrInvalidTransaction = gorm.ErrInvalidTransaction // 无效事务
)

// 自定义错误类型
type RepositoryError struct {
	Op    string // 操作名称
	Model string // 模型名称
	Err   error  // 原始错误
}

func (e *RepositoryError) Error() string {
	return fmt.Sprintf("repository %s.%s: %v", e.Model, e.Op, e.Err)
}

// 辅助函数：创建Repository错误
func NewRepositoryError(op, model string, err error) *RepositoryError {
	return &RepositoryError{
		Op:    op,
		Model: model,
		Err:   err,
	}
}
