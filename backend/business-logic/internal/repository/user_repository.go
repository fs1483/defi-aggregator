// Package repository 用户数据访问层实现
// 实现UserRepository接口，提供用户相关的数据库操作
// 包括用户CRUD、认证、偏好设置等功能，遵循Repository模式最佳实践
package repository

import (
	"fmt"
	"time"

	"defi-aggregator/business-logic/internal/models"
	"defi-aggregator/business-logic/internal/types"

	"gorm.io/gorm"
)

// userRepository 用户数据访问层实现
// 实现UserRepository接口，封装所有用户相关的数据库操作
type userRepository struct {
	db *gorm.DB // 数据库连接实例
}

// NewUserRepository 创建用户数据访问层实例
// 注入数据库连接，返回UserRepository接口实现
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

// ========================================
// 基础CRUD操作实现
// ========================================

// Create 创建新用户
// 在数据库中创建新的用户记录，自动设置创建时间
// 参数:
//   - user: 要创建的用户模型，包含用户基本信息
//
// 返回:
//   - error: 创建过程中的错误，如违反唯一约束等
func (r *userRepository) Create(user *models.User) error {
	// 执行数据库插入操作
	// GORM会自动设置ID、CreatedAt、UpdatedAt等字段
	if err := r.db.Create(user).Error; err != nil {
		return NewRepositoryError("Create", "User", err)
	}
	return nil
}

// GetByID 根据用户ID获取用户信息
// 根据主键ID查询用户，包含用户偏好设置的预加载
// 参数:
//   - id: 用户主键ID
//
// 返回:
//   - *models.User: 用户模型指针，包含完整用户信息
//   - error: 查询错误，如用户不存在等
func (r *userRepository) GetByID(id uint) (*models.User, error) {
	var user models.User

	// 执行查询，预加载用户偏好设置
	// 使用Preload确保关联数据一次性加载，避免N+1查询问题
	err := r.db.Preload("Preferences").First(&user, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, NewRepositoryError("GetByID", "User", fmt.Errorf("用户不存在: ID=%d", id))
		}
		return nil, NewRepositoryError("GetByID", "User", err)
	}

	return &user, nil
}

// GetByWalletAddress 根据钱包地址获取用户信息
// 钱包地址是用户的唯一标识，用于Web3身份验证
// 参数:
//   - address: 用户钱包地址，格式为0x开头的42位十六进制字符串
//
// 返回:
//   - *models.User: 用户模型指针
//   - error: 查询错误
func (r *userRepository) GetByWalletAddress(address string) (*models.User, error) {
	var user models.User

	// 根据钱包地址查询用户，预加载偏好设置
	err := r.db.Preload("Preferences").
		Where("wallet_address = ?", address).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, NewRepositoryError("GetByWalletAddress", "User",
				fmt.Errorf("钱包地址对应的用户不存在: %s", address))
		}
		return nil, NewRepositoryError("GetByWalletAddress", "User", err)
	}

	return &user, nil
}

// Update 更新用户信息
// 更新用户的基本信息，自动更新UpdatedAt字段
// 参数:
//   - user: 包含更新信息的用户模型
//
// 返回:
//   - error: 更新过程中的错误
func (r *userRepository) Update(user *models.User) error {
	// 使用Save方法更新所有字段
	// GORM会自动更新UpdatedAt字段
	if err := r.db.Save(user).Error; err != nil {
		return NewRepositoryError("Update", "User", err)
	}
	return nil
}

// Delete 删除用户
// 执行软删除，设置DeletedAt字段而不是物理删除记录
// 参数:
//   - id: 要删除的用户ID
//
// 返回:
//   - error: 删除过程中的错误
func (r *userRepository) Delete(id uint) error {
	// 软删除用户记录
	// GORM会自动设置DeletedAt字段，查询时自动排除已删除记录
	result := r.db.Delete(&models.User{}, id)
	if result.Error != nil {
		return NewRepositoryError("Delete", "User", result.Error)
	}

	// 检查是否实际删除了记录
	if result.RowsAffected == 0 {
		return NewRepositoryError("Delete", "User",
			fmt.Errorf("要删除的用户不存在: ID=%d", id))
	}

	return nil
}

// ========================================
// 用户偏好设置操作实现
// ========================================

// GetPreferences 获取用户偏好设置
// 根据用户ID获取个性化偏好配置
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - *models.UserPreferences: 用户偏好模型
//   - error: 查询错误
func (r *userRepository) GetPreferences(userID uint) (*models.UserPreferences, error) {
	var preferences models.UserPreferences

	err := r.db.Where("user_id = ?", userID).First(&preferences).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, NewRepositoryError("GetPreferences", "UserPreferences",
				fmt.Errorf("用户偏好设置不存在: UserID=%d", userID))
		}
		return nil, NewRepositoryError("GetPreferences", "UserPreferences", err)
	}

	return &preferences, nil
}

// UpdatePreferences 更新用户偏好设置
// 更新现有的用户偏好配置
// 参数:
//   - preferences: 包含更新信息的偏好模型
//
// 返回:
//   - error: 更新错误
func (r *userRepository) UpdatePreferences(preferences *models.UserPreferences) error {
	if err := r.db.Save(preferences).Error; err != nil {
		return NewRepositoryError("UpdatePreferences", "UserPreferences", err)
	}
	return nil
}

// CreatePreferences 创建用户偏好设置
// 为新用户创建默认的偏好配置
// 参数:
//   - preferences: 要创建的偏好模型
//
// 返回:
//   - error: 创建错误
func (r *userRepository) CreatePreferences(preferences *models.UserPreferences) error {
	if err := r.db.Create(preferences).Error; err != nil {
		return NewRepositoryError("CreatePreferences", "UserPreferences", err)
	}
	return nil
}

// ========================================
// 查询操作实现
// ========================================

// List 分页获取用户列表
// 支持分页、排序和基本筛选功能
// 参数:
//   - req: 分页请求参数，包含页码、每页大小、排序等
//
// 返回:
//   - []*models.User: 用户列表
//   - int64: 总记录数
//   - error: 查询错误
func (r *userRepository) List(req *types.PaginationRequest) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	// 构建基础查询
	query := r.db.Model(&models.User{})

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, NewRepositoryError("List", "User", err)
	}

	// 应用分页和排序
	offset := (req.Page - 1) * req.PageSize
	orderBy := "created_at DESC"
	if req.SortBy != "" {
		if req.SortDesc {
			orderBy = fmt.Sprintf("%s DESC", req.SortBy)
		} else {
			orderBy = fmt.Sprintf("%s ASC", req.SortBy)
		}
	}

	// 执行分页查询
	err := query.Preload("Preferences").
		Order(orderBy).
		Offset(offset).
		Limit(req.PageSize).
		Find(&users).Error

	if err != nil {
		return nil, 0, NewRepositoryError("List", "User", err)
	}

	return users, total, nil
}

// GetActiveUsers 获取活跃用户列表
// 返回状态为活跃的所有用户
// 返回:
//   - []*models.User: 活跃用户列表
//   - error: 查询错误
func (r *userRepository) GetActiveUsers() ([]*models.User, error) {
	var users []*models.User

	err := r.db.Where("is_active = ?", true).
		Preload("Preferences").
		Order("last_login_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, NewRepositoryError("GetActiveUsers", "User", err)
	}

	return users, nil
}

// UpdateLastLogin 更新用户最后登录时间
// 在用户成功登录后调用，用于统计活跃度
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - error: 更新错误
func (r *userRepository) UpdateLastLogin(userID uint) error {
	now := time.Now()

	// 只更新last_login_at字段，提高性能
	result := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", now)

	if result.Error != nil {
		return NewRepositoryError("UpdateLastLogin", "User", result.Error)
	}

	if result.RowsAffected == 0 {
		return NewRepositoryError("UpdateLastLogin", "User",
			fmt.Errorf("用户不存在: ID=%d", userID))
	}

	return nil
}

// ========================================
// 认证相关操作实现
// ========================================

// UpdateNonce 更新用户登录随机数
// 用于Web3钱包签名认证，每次登录请求都会生成新的随机数
// 参数:
//   - userID: 用户ID
//   - nonce: 新的随机数字符串
//
// 返回:
//   - error: 更新错误
func (r *userRepository) UpdateNonce(userID uint, nonce string) error {
	// 只更新nonce字段
	result := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("nonce", nonce)

	if result.Error != nil {
		return NewRepositoryError("UpdateNonce", "User", result.Error)
	}

	if result.RowsAffected == 0 {
		return NewRepositoryError("UpdateNonce", "User",
			fmt.Errorf("用户不存在: ID=%d", userID))
	}

	return nil
}

// ========================================
// 高级查询操作实现
// ========================================

// GetUsersByDateRange 根据注册时间范围获取用户
// 用于统计分析，获取特定时间段注册的用户
// 参数:
//   - startDate: 开始日期
//   - endDate: 结束日期
//
// 返回:
//   - []*models.User: 用户列表
//   - error: 查询错误
func (r *userRepository) GetUsersByDateRange(startDate, endDate time.Time) ([]*models.User, error) {
	var users []*models.User

	err := r.db.Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Order("created_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, NewRepositoryError("GetUsersByDateRange", "User", err)
	}

	return users, nil
}

// SearchUsers 搜索用户
// 根据用户名或邮箱进行模糊搜索
// 参数:
//   - query: 搜索关键词
//
// 返回:
//   - []*models.User: 匹配的用户列表
//   - error: 查询错误
func (r *userRepository) SearchUsers(query string) ([]*models.User, error) {
	var users []*models.User

	searchPattern := "%" + query + "%"
	err := r.db.Where("username ILIKE ? OR email ILIKE ?", searchPattern, searchPattern).
		Limit(50). // 限制搜索结果数量
		Find(&users).Error

	if err != nil {
		return nil, NewRepositoryError("SearchUsers", "User", err)
	}

	return users, nil
}

// GetUserStats 获取用户统计信息
// 返回用户相关的统计数据，如总数、活跃数等
// 返回:
//   - map[string]interface{}: 统计信息键值对
//   - error: 查询错误
func (r *userRepository) GetUserStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总用户数
	var totalUsers int64
	if err := r.db.Model(&models.User{}).Count(&totalUsers).Error; err != nil {
		return nil, NewRepositoryError("GetUserStats", "User", err)
	}
	stats["total_users"] = totalUsers

	// 活跃用户数
	var activeUsers int64
	if err := r.db.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUsers).Error; err != nil {
		return nil, NewRepositoryError("GetUserStats", "User", err)
	}
	stats["active_users"] = activeUsers

	// 今日新增用户数
	today := time.Now().Truncate(24 * time.Hour)
	var todayNewUsers int64
	if err := r.db.Model(&models.User{}).Where("created_at >= ?", today).Count(&todayNewUsers).Error; err != nil {
		return nil, NewRepositoryError("GetUserStats", "User", err)
	}
	stats["today_new_users"] = todayNewUsers

	// 最近登录用户数（24小时内）
	yesterday := time.Now().Add(-24 * time.Hour)
	var recentActiveUsers int64
	if err := r.db.Model(&models.User{}).Where("last_login_at >= ?", yesterday).Count(&recentActiveUsers).Error; err != nil {
		return nil, NewRepositoryError("GetUserStats", "User", err)
	}
	stats["recent_active_users"] = recentActiveUsers

	return stats, nil
}

// ========================================
// 事务支持和辅助方法
// ========================================

// WithTx 返回使用指定事务的Repository实例
// 用于在事务中执行多个相关操作，确保数据一致性
// 参数:
//   - tx: GORM事务实例
//
// 返回:
//   - interface{}: 使用事务的Repository实例
func (r *userRepository) WithTx(tx *gorm.DB) interface{} {
	return &userRepository{db: tx}
}

// HealthCheck 检查Repository健康状态
// 执行简单的数据库查询，验证连接和权限
// 返回:
//   - error: 健康检查失败的错误
func (r *userRepository) HealthCheck() error {
	// 执行简单的count查询测试数据库连接
	var count int64
	if err := r.db.Model(&models.User{}).Limit(1).Count(&count).Error; err != nil {
		return NewRepositoryError("HealthCheck", "User", err)
	}
	return nil
}

// BatchCreate 批量创建用户
// 用于数据迁移或批量导入场景
// 参数:
//   - users: 要创建的用户列表
//   - batchSize: 每批处理的数量
//
// 返回:
//   - error: 批量创建错误
func (r *userRepository) BatchCreate(users []*models.User, batchSize int) error {
	// 分批插入，避免单次操作数据量过大
	for i := 0; i < len(users); i += batchSize {
		end := i + batchSize
		if end > len(users) {
			end = len(users)
		}

		batch := users[i:end]
		if err := r.db.Create(&batch).Error; err != nil {
			return NewRepositoryError("BatchCreate", "User", err)
		}
	}

	return nil
}

// GetOrCreate 获取或创建用户
// 如果用户不存在则创建，存在则返回现有用户
// 这是Web3应用中常见的模式，用户首次连接钱包时自动注册
// 参数:
//   - walletAddress: 钱包地址
//   - userInfo: 用户基本信息
//
// 返回:
//   - *models.User: 用户模型
//   - bool: 是否为新创建的用户
//   - error: 操作错误
func (r *userRepository) GetOrCreate(walletAddress string, userInfo *models.User) (*models.User, bool, error) {
	// 首先尝试获取现有用户
	existingUser, err := r.GetByWalletAddress(walletAddress)
	if err == nil {
		// 用户已存在，返回现有用户
		return existingUser, false, nil
	}

	// 检查错误类型，如果不是记录不存在的错误，则返回错误
	if repoErr, ok := err.(*RepositoryError); ok {
		if repoErr.Err != gorm.ErrRecordNotFound {
			return nil, false, err
		}
	}

	// 用户不存在，创建新用户
	userInfo.WalletAddress = walletAddress
	if err := r.Create(userInfo); err != nil {
		return nil, false, err
	}

	return userInfo, true, nil
}
