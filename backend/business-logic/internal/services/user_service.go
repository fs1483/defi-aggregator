// Package services 用户业务服务实现
// 提供用户信息管理、偏好设置、统计查询等功能
// 遵循业务逻辑分离原则，确保数据安全和操作审计
package services

import (
	"fmt"

	"defi-aggregator/business-logic/internal/models"
	"defi-aggregator/business-logic/internal/repository"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// userService 用户业务服务实现
// 负责处理用户信息管理、偏好设置、活动追踪等功能
type userService struct {
	repos  *repository.Repositories // 数据访问层
	cfg    *config.Config           // 应用配置
	logger *logrus.Logger           // 日志记录器
}

// NewUserService 创建用户服务实例
// 注入必要的依赖，初始化用户服务
func NewUserService(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) UserService {
	return &userService{
		repos:  repos,
		cfg:    cfg,
		logger: logger,
	}
}

// ========================================
// 用户基础操作实现
// ========================================

// GetProfile 获取用户资料
// 返回用户的基本信息，过滤敏感数据
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - *types.UserInfo: 用户信息
//   - error: 查询错误
func (s *userService) GetProfile(userID uint) (*types.UserInfo, error) {
	// 从数据库获取用户信息
	user, err := s.repos.User.GetByID(userID)
	if err != nil {
		s.logger.Warnf("获取用户资料失败: userID=%d, error=%v", userID, err)
		return nil, NewServiceError(types.ErrCodeNotFound, "用户不存在", err)
	}

	// 转换为API响应格式
	userInfo := s.convertToUserInfo(user)

	s.logger.Debugf("获取用户 %d 的资料成功", userID)
	return userInfo, nil
}

// UpdateProfile 更新用户资料
// 允许用户更新基本信息，验证数据有效性
// 参数:
//   - userID: 用户ID
//   - req: 更新请求，包含要更新的字段
//
// 返回:
//   - error: 更新过程中的错误
func (s *userService) UpdateProfile(userID uint, req *types.UpdateUserRequest) error {
	// 获取现有用户信息
	user, err := s.repos.User.GetByID(userID)
	if err != nil {
		s.logger.Warnf("更新用户资料时获取用户失败: userID=%d", userID)
		return NewServiceError(types.ErrCodeNotFound, "用户不存在", err)
	}

	// 记录更新前的状态（用于审计）
	s.logger.Infof("开始更新用户 %d 的资料", userID)

	// 应用更新（只更新非空字段）
	hasChanges := false

	if req.Username != nil && *req.Username != user.Username {
		user.Username = *req.Username
		hasChanges = true
		s.logger.Debugf("用户 %d 更新用户名: %s", userID, *req.Username)
	}

	if req.Email != nil && *req.Email != user.Email {
		user.Email = *req.Email
		hasChanges = true
		s.logger.Debugf("用户 %d 更新邮箱: %s", userID, *req.Email)
	}

	if req.AvatarURL != nil && *req.AvatarURL != user.AvatarURL {
		user.AvatarURL = *req.AvatarURL
		hasChanges = true
		s.logger.Debugf("用户 %d 更新头像URL", userID)
	}

	if req.PreferredLang != nil && *req.PreferredLang != user.PreferredLanguage {
		user.PreferredLanguage = *req.PreferredLang
		hasChanges = true
		s.logger.Debugf("用户 %d 更新首选语言: %s", userID, *req.PreferredLang)
	}

	if req.Timezone != nil && *req.Timezone != user.Timezone {
		user.Timezone = *req.Timezone
		hasChanges = true
		s.logger.Debugf("用户 %d 更新时区: %s", userID, *req.Timezone)
	}

	// 如果有变更，保存到数据库
	if hasChanges {
		if err := s.repos.User.Update(user); err != nil {
			s.logger.Errorf("保存用户资料更新失败: userID=%d, error=%v", userID, err)
			return NewServiceError(types.ErrCodeInternal, "保存用户信息失败", err)
		}
		s.logger.Infof("用户 %d 资料更新成功", userID)
	} else {
		s.logger.Debugf("用户 %d 的资料无变更", userID)
	}

	return nil
}

// DeactivateUser 停用用户账户
// 停用用户而不是删除，保留历史数据的完整性
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - error: 操作错误
func (s *userService) DeactivateUser(userID uint) error {
	user, err := s.repos.User.GetByID(userID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "用户不存在", err)
	}

	if !user.IsActive {
		return NewServiceError(types.ErrCodeConflict, "用户已处于停用状态", nil)
	}

	// 停用用户
	user.IsActive = false
	if err := s.repos.User.Update(user); err != nil {
		s.logger.Errorf("停用用户失败: userID=%d, error=%v", userID, err)
		return NewServiceError(types.ErrCodeInternal, "停用用户失败", err)
	}

	// 撤销所有令牌
	// TODO: 调用认证服务撤销令牌

	s.logger.Infof("用户 %d 已停用", userID)
	return nil
}

// ========================================
// 用户偏好设置实现
// ========================================

// GetPreferences 获取用户偏好设置
// 返回用户的个性化配置，如默认滑点、Gas速度等
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - *types.UserPreferences: 用户偏好设置
//   - error: 查询错误
func (s *userService) GetPreferences(userID uint) (*types.UserPreferences, error) {
	// 获取用户偏好设置
	prefs, err := s.repos.User.GetPreferences(userID)
	if err != nil {
		s.logger.Warnf("获取用户偏好失败: userID=%d, error=%v", userID, err)
		return nil, NewServiceError(types.ErrCodeNotFound, "用户偏好设置不存在", err)
	}

	// 转换为API响应格式
	userPrefs := &types.UserPreferences{
		DefaultSlippage:     prefs.DefaultSlippage,
		PreferredGasSpeed:   prefs.PreferredGasSpeed,
		AutoApproveTokens:   prefs.AutoApproveTokens,
		ShowTestTokens:      prefs.ShowTestTokens,
		NotificationEmail:   prefs.NotificationEmail,
		NotificationBrowser: prefs.NotificationBrowser,
		PrivacyAnalytics:    prefs.PrivacyAnalytics,
	}

	s.logger.Debugf("获取用户 %d 的偏好设置成功", userID)
	return userPrefs, nil
}

// UpdatePreferences 更新用户偏好设置
// 允许用户自定义交易相关的偏好配置
// 参数:
//   - userID: 用户ID
//   - prefs: 新的偏好设置
//
// 返回:
//   - error: 更新错误
func (s *userService) UpdatePreferences(userID uint, prefs *types.UserPreferences) error {
	// 验证偏好设置的有效性
	if err := s.validatePreferences(prefs); err != nil {
		return err
	}

	// 获取现有偏好设置
	existingPrefs, err := s.repos.User.GetPreferences(userID)
	if err != nil {
		s.logger.Warnf("获取现有偏好设置失败: userID=%d", userID)
		return NewServiceError(types.ErrCodeNotFound, "用户偏好设置不存在", err)
	}

	// 应用更新
	existingPrefs.DefaultSlippage = prefs.DefaultSlippage
	existingPrefs.PreferredGasSpeed = prefs.PreferredGasSpeed
	existingPrefs.AutoApproveTokens = prefs.AutoApproveTokens
	existingPrefs.ShowTestTokens = prefs.ShowTestTokens
	existingPrefs.NotificationEmail = prefs.NotificationEmail
	existingPrefs.NotificationBrowser = prefs.NotificationBrowser
	existingPrefs.PrivacyAnalytics = prefs.PrivacyAnalytics

	// 保存更新
	if err := s.repos.User.UpdatePreferences(existingPrefs); err != nil {
		s.logger.Errorf("保存用户偏好设置失败: userID=%d, error=%v", userID, err)
		return NewServiceError(types.ErrCodeInternal, "保存偏好设置失败", err)
	}

	s.logger.Infof("用户 %d 的偏好设置更新成功", userID)
	return nil
}

// ResetPreferences 重置用户偏好设置为默认值
// 将用户偏好恢复到系统默认配置
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - error: 重置错误
func (s *userService) ResetPreferences(userID uint) error {
	// 获取现有偏好设置
	existingPrefs, err := s.repos.User.GetPreferences(userID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "用户偏好设置不存在", err)
	}

	// 重置为默认值
	existingPrefs.DefaultSlippage = decimal.NewFromFloat(0.005) // 0.5%
	existingPrefs.PreferredGasSpeed = types.GasSpeedStandard
	existingPrefs.AutoApproveTokens = false
	existingPrefs.ShowTestTokens = false
	existingPrefs.NotificationEmail = true
	existingPrefs.NotificationBrowser = true
	existingPrefs.PrivacyAnalytics = true

	// 保存重置后的设置
	if err := s.repos.User.UpdatePreferences(existingPrefs); err != nil {
		s.logger.Errorf("重置用户偏好设置失败: userID=%d, error=%v", userID, err)
		return NewServiceError(types.ErrCodeInternal, "重置偏好设置失败", err)
	}

	s.logger.Infof("用户 %d 的偏好设置已重置为默认值", userID)
	return nil
}

// ========================================
// 用户统计实现
// ========================================

// GetUserStats 获取用户统计信息
// 返回用户的交易统计、资金统计等数据
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - *types.UserStatsResponse: 用户统计信息
//   - error: 查询错误
func (s *userService) GetUserStats(userID uint) (*types.UserStatsResponse, error) {
	// 验证用户存在
	if _, err := s.repos.User.GetByID(userID); err != nil {
		return nil, NewServiceError(types.ErrCodeNotFound, "用户不存在", err)
	}

	// 获取用户统计（通过TransactionRepository）
	stats, err := s.repos.Transaction.GetUserStats(userID)
	if err != nil {
		s.logger.Errorf("获取用户统计失败: userID=%d, error=%v", userID, err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取统计数据失败", err)
	}

	s.logger.Debugf("获取用户 %d 的统计信息成功", userID)
	return stats, nil
}

// GetUserList 获取用户列表（管理员功能）
// 分页返回用户列表，包含基本信息和统计数据
// 参数:
//   - req: 分页请求参数
//
// 返回:
//   - []*types.UserInfo: 用户信息列表
//   - *types.Meta: 分页元数据
//   - error: 查询错误
func (s *userService) GetUserList(req *types.PaginationRequest) ([]*types.UserInfo, *types.Meta, error) {
	// 获取用户列表
	users, total, err := s.repos.User.List(req)
	if err != nil {
		s.logger.Errorf("获取用户列表失败: error=%v", err)
		return nil, nil, NewServiceError(types.ErrCodeInternal, "获取用户列表失败", err)
	}

	// 转换为API响应格式
	var userInfos []*types.UserInfo
	for _, user := range users {
		userInfo := s.convertToUserInfo(user)
		userInfos = append(userInfos, userInfo)
	}

	// 计算分页信息
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	meta := &types.Meta{
		Page:       req.Page,
		PageSize:   req.PageSize,
		Total:      int(total),
		TotalPages: totalPages,
	}

	s.logger.Debugf("获取用户列表成功: total=%d, page=%d", total, req.Page)
	return userInfos, meta, nil
}

// ========================================
// 用户活动管理实现
// ========================================

// UpdateLastActivity 更新用户最后活动时间
// 在用户执行重要操作时更新活动时间，用于活跃度统计
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - error: 更新错误
func (s *userService) UpdateLastActivity(userID uint) error {
	// 更新最后登录时间（复用此字段作为活动时间）
	if err := s.repos.User.UpdateLastLogin(userID); err != nil {
		s.logger.Errorf("更新用户活动时间失败: userID=%d, error=%v", userID, err)
		return NewServiceError(types.ErrCodeInternal, "更新活动时间失败", err)
	}

	s.logger.Debugf("用户 %d 活动时间已更新", userID)
	return nil
}

// GetActiveUsers 获取活跃用户列表
// 返回最近有活动的用户列表，用于统计分析
// 返回:
//   - []*types.UserInfo: 活跃用户列表
//   - error: 查询错误
func (s *userService) GetActiveUsers() ([]*types.UserInfo, error) {
	// 获取活跃用户
	users, err := s.repos.User.GetActiveUsers()
	if err != nil {
		s.logger.Errorf("获取活跃用户失败: error=%v", err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取活跃用户失败", err)
	}

	// 转换为API响应格式
	var userInfos []*types.UserInfo
	for _, user := range users {
		userInfo := s.convertToUserInfo(user)
		userInfos = append(userInfos, userInfo)
	}

	s.logger.Debugf("获取活跃用户成功: count=%d", len(userInfos))
	return userInfos, nil
}

// ========================================
// 辅助方法
// ========================================

// validatePreferences 验证用户偏好设置的有效性
// 检查偏好设置中的各项参数是否在合理范围内
func (s *userService) validatePreferences(prefs *types.UserPreferences) error {
	// 验证滑点范围 (0% - 50%)
	minSlippage := decimal.NewFromFloat(0.0)
	maxSlippage := decimal.NewFromFloat(0.5)
	if prefs.DefaultSlippage.LessThan(minSlippage) || prefs.DefaultSlippage.GreaterThan(maxSlippage) {
		return NewServiceError(types.ErrCodeValidation,
			fmt.Sprintf("滑点设置必须在0%%到50%%之间，当前值: %s", prefs.DefaultSlippage.String()), nil)
	}

	// 验证Gas速度设置
	validGasSpeeds := map[string]bool{
		types.GasSpeedSlow:     true,
		types.GasSpeedStandard: true,
		types.GasSpeedFast:     true,
	}
	if !validGasSpeeds[prefs.PreferredGasSpeed] {
		return NewServiceError(types.ErrCodeValidation,
			fmt.Sprintf("无效的Gas速度设置: %s", prefs.PreferredGasSpeed), nil)
	}

	return nil
}

// convertToUserInfo 将数据库用户模型转换为API响应格式
// 过滤敏感信息，只返回前端需要的数据
func (s *userService) convertToUserInfo(user *models.User) *types.UserInfo {
	userInfo := &types.UserInfo{
		ID:            user.ID,
		WalletAddress: user.WalletAddress,
		Username:      user.Username,
		Email:         user.Email,
		AvatarURL:     user.AvatarURL,
		PreferredLang: user.PreferredLanguage,
		Timezone:      user.Timezone,
		IsActive:      user.IsActive,
		CreatedAt:     user.CreatedAt,
	}

	// 设置最后登录时间（如果不为空）
	if user.LastLoginAt != nil {
		userInfo.LastLoginAt = user.LastLoginAt
	}

	return userInfo
}
