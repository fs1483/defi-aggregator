// Package services 用户认证服务实现
// 实现Web3钱包签名认证、JWT令牌管理、会话管理等功能
// 遵循Web3认证最佳实践和企业级安全标准
package services

import (
	"strings"

	"defi-aggregator/business-logic/internal/models"
	"defi-aggregator/business-logic/internal/repository"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"
	"defi-aggregator/business-logic/pkg/utils"

	"github.com/sirupsen/logrus"
)

// authService 认证服务实现
// 负责处理用户认证、JWT令牌管理、会话控制等安全相关功能
type authService struct {
	repos  *repository.Repositories // 数据访问层
	cfg    *config.Config           // 应用配置
	logger *logrus.Logger           // 日志记录器
}

// NewAuthService 创建认证服务实例
// 注入必要的依赖，初始化认证服务
func NewAuthService(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) AuthService {
	return &authService{
		repos:  repos,
		cfg:    cfg,
		logger: logger,
	}
}

// ========================================
// Web3钱包认证实现
// ========================================

// GenerateNonce 为钱包地址生成登录随机数
// 每次登录请求都会生成新的随机数，防止重放攻击
// 参数:
//   - walletAddress: 用户钱包地址
//
// 返回:
//   - string: 生成的随机数
//   - error: 生成过程中的错误
func (s *authService) GenerateNonce(walletAddress string) (string, error) {
	// 验证钱包地址格式
	normalizedAddress, err := utils.NormalizeEthereumAddress(walletAddress)
	if err != nil {
		s.logger.Warnf("无效的钱包地址: %s", walletAddress)
		return "", NewServiceError(types.ErrCodeValidation, "无效的钱包地址格式", err)
	}

	// 生成随机数
	nonce, err := utils.GenerateNonce()
	if err != nil {
		s.logger.Errorf("生成随机数失败: %v", err)
		return "", NewServiceError(types.ErrCodeInternal, "随机数生成失败", err)
	}

	// 查找或创建用户记录
	user, isNewUser, err := s.getOrCreateUser(normalizedAddress)
	if err != nil {
		s.logger.Errorf("获取或创建用户失败: %v", err)
		return "", NewServiceError(types.ErrCodeInternal, "用户处理失败", err)
	}

	// 更新用户的nonce
	if err := s.repos.User.UpdateNonce(user.ID, nonce); err != nil {
		s.logger.Errorf("更新用户nonce失败: %v", err)
		return "", NewServiceError(types.ErrCodeInternal, "更新随机数失败", err)
	}

	s.logger.Infof("为用户 %s 生成登录随机数 (新用户: %t)", normalizedAddress, isNewUser)
	return nonce, nil
}

// VerifySignature 验证钱包签名并完成登录
// 验证用户提供的签名，生成JWT令牌
// 参数:
//   - req: 登录请求，包含钱包地址、签名、消息等
//
// 返回:
//   - *types.UserLoginResponse: 登录响应，包含JWT令牌和用户信息
//   - error: 验证或登录过程中的错误
func (s *authService) VerifySignature(req *types.UserLoginRequest) (*types.UserLoginResponse, error) {
	// 1. 验证请求参数
	if err := s.validateLoginRequest(req); err != nil {
		return nil, err
	}

	// 2. 标准化钱包地址
	normalizedAddress, err := utils.NormalizeEthereumAddress(req.WalletAddress)
	if err != nil {
		return nil, NewServiceError(types.ErrCodeValidation, "无效的钱包地址", err)
	}

	// 3. 获取用户信息
	user, err := s.repos.User.GetByWalletAddress(normalizedAddress)
	if err != nil {
		s.logger.Warnf("用户不存在: %s", normalizedAddress)
		return nil, NewServiceError(types.ErrCodeUnauthorized, "用户不存在", err)
	}

	// 4. 验证nonce
	if user.Nonce != req.Nonce {
		s.logger.Warnf("用户 %s 的nonce不匹配", normalizedAddress)
		return nil, NewServiceError(types.ErrCodeUnauthorized, "随机数不匹配", nil)
	}

	// 5. 验证签名
	isValid, err := utils.VerifySignature(req.Message, req.Signature, normalizedAddress)
	if err != nil {
		s.logger.Errorf("签名验证失败: %v", err)
		return nil, NewServiceError(types.ErrCodeUnauthorized, "签名验证失败", err)
	}

	if !isValid {
		s.logger.Warnf("用户 %s 提供的签名无效", normalizedAddress)
		return nil, NewServiceError(types.ErrCodeUnauthorized, "签名无效", nil)
	}

	// 6. 生成JWT令牌
	accessToken, refreshToken, err := s.GenerateTokens(user.ID, normalizedAddress)
	if err != nil {
		return nil, err
	}

	// 7. 更新用户最后登录时间
	if err := s.repos.User.UpdateLastLogin(user.ID); err != nil {
		s.logger.Warnf("更新用户最后登录时间失败: %v", err)
		// 不影响登录流程，只记录警告
	}

	// 8. 转换用户信息
	userInfo := s.convertToUserInfo(user)

	s.logger.Infof("用户 %s (ID: %d) 登录成功", normalizedAddress, user.ID)

	return &types.UserLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.JWT.ExpiresIn.Seconds()),
		User:         userInfo,
	}, nil
}

// ========================================
// JWT令牌管理实现
// ========================================

// GenerateTokens 生成访问令牌和刷新令牌
// 为认证用户创建一对JWT令牌
// 参数:
//   - userID: 用户ID
//   - walletAddress: 钱包地址
//
// 返回:
//   - accessToken: 访问令牌
//   - refreshToken: 刷新令牌
//   - error: 生成错误
func (s *authService) GenerateTokens(userID uint, walletAddress string) (accessToken, refreshToken string, err error) {
	// 生成访问令牌
	accessToken, err = utils.GenerateJWT(
		userID,
		walletAddress,
		"user", // 默认角色
		s.cfg.JWT.SecretKey,
		s.cfg.JWT.ExpiresIn,
		"access",
	)
	if err != nil {
		s.logger.Errorf("生成访问令牌失败: %v", err)
		return "", "", NewServiceError(types.ErrCodeInternal, "访问令牌生成失败", err)
	}

	// 生成刷新令牌
	refreshToken, err = utils.GenerateJWT(
		userID,
		walletAddress,
		"user",
		s.cfg.JWT.SecretKey,
		s.cfg.JWT.RefreshExpiresIn,
		"refresh",
	)
	if err != nil {
		s.logger.Errorf("生成刷新令牌失败: %v", err)
		return "", "", NewServiceError(types.ErrCodeInternal, "刷新令牌生成失败", err)
	}

	s.logger.Debugf("为用户 %d 生成令牌对成功", userID)
	return accessToken, refreshToken, nil
}

// RefreshToken 刷新访问令牌
// 使用有效的刷新令牌生成新的访问令牌
// 参数:
//   - refreshTokenString: 刷新令牌字符串
//
// 返回:
//   - newAccessToken: 新的访问令牌
//   - error: 刷新过程中的错误
func (s *authService) RefreshToken(refreshTokenString string) (newAccessToken string, err error) {
	// 解析刷新令牌
	claims, err := utils.ParseJWT(refreshTokenString, s.cfg.JWT.SecretKey)
	if err != nil {
		s.logger.Warnf("刷新令牌解析失败: %v", err)
		return "", NewServiceError(types.ErrCodeUnauthorized, "无效的刷新令牌", err)
	}

	// 验证令牌类型
	if claims.TokenType != "refresh" {
		s.logger.Warnf("令牌类型错误: %s", claims.TokenType)
		return "", NewServiceError(types.ErrCodeUnauthorized, "令牌类型错误", nil)
	}

	// 验证用户是否仍然存在且活跃
	user, err := s.repos.User.GetByID(claims.UserID)
	if err != nil {
		s.logger.Warnf("用户不存在: ID=%d", claims.UserID)
		return "", NewServiceError(types.ErrCodeUnauthorized, "用户不存在", err)
	}

	if !user.IsActive {
		s.logger.Warnf("用户已停用: ID=%d", claims.UserID)
		return "", NewServiceError(types.ErrCodeUnauthorized, "用户已停用", nil)
	}

	// 生成新的访问令牌
	newAccessToken, err = utils.GenerateJWT(
		user.ID,
		user.WalletAddress,
		"user",
		s.cfg.JWT.SecretKey,
		s.cfg.JWT.ExpiresIn,
		"access",
	)
	if err != nil {
		s.logger.Errorf("生成新访问令牌失败: %v", err)
		return "", NewServiceError(types.ErrCodeInternal, "令牌生成失败", err)
	}

	s.logger.Infof("用户 %d 刷新令牌成功", user.ID)
	return newAccessToken, nil
}

// RevokeToken 撤销用户令牌
// 在用户登出或安全事件时撤销令牌
// 参数:
//   - userID: 用户ID
//   - tokenType: 令牌类型 ("access", "refresh", "all")
//
// 返回:
//   - error: 撤销过程中的错误
func (s *authService) RevokeToken(userID uint, tokenType string) error {
	// TODO: 实现令牌黑名单机制
	// 可以使用Redis存储被撤销的令牌ID，直到其自然过期
	// 或者在JWT中添加版本号，用户登出时递增版本号

	s.logger.Infof("撤销用户 %d 的 %s 令牌", userID, tokenType)
	return nil
}

// ========================================
// 会话管理实现
// ========================================

// ValidateSession 验证用户会话有效性
// 检查用户是否仍然活跃且有效
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - error: 验证失败的错误
func (s *authService) ValidateSession(userID uint) error {
	user, err := s.repos.User.GetByID(userID)
	if err != nil {
		return NewServiceError(types.ErrCodeUnauthorized, "用户不存在", err)
	}

	if !user.IsActive {
		return NewServiceError(types.ErrCodeUnauthorized, "用户已停用", nil)
	}

	return nil
}

// LogoutUser 用户登出
// 撤销用户的访问令牌，结束会话
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - error: 登出过程中的错误
func (s *authService) LogoutUser(userID uint) error {
	// 撤销访问令牌
	if err := s.RevokeToken(userID, "access"); err != nil {
		return err
	}

	s.logger.Infof("用户 %d 登出成功", userID)
	return nil
}

// LogoutAllSessions 登出用户的所有会话
// 撤销用户的所有令牌，用于安全事件处理
// 参数:
//   - userID: 用户ID
//
// 返回:
//   - error: 登出过程中的错误
func (s *authService) LogoutAllSessions(userID uint) error {
	// 撤销所有令牌
	if err := s.RevokeToken(userID, "all"); err != nil {
		return err
	}

	s.logger.Infof("用户 %d 的所有会话已登出", userID)
	return nil
}

// ========================================
// 辅助方法
// ========================================

// validateLoginRequest 验证登录请求参数
// 检查请求参数的完整性和格式正确性
func (s *authService) validateLoginRequest(req *types.UserLoginRequest) error {
	// 验证必填字段
	if req.WalletAddress == "" {
		return NewServiceError(types.ErrCodeValidation, "钱包地址不能为空", nil)
	}

	if req.Signature == "" {
		return NewServiceError(types.ErrCodeValidation, "签名不能为空", nil)
	}

	if req.Message == "" {
		return NewServiceError(types.ErrCodeValidation, "签名消息不能为空", nil)
	}

	if req.Nonce == "" {
		return NewServiceError(types.ErrCodeValidation, "随机数不能为空", nil)
	}

	// 验证钱包地址格式
	if !utils.IsValidEthereumAddress(req.WalletAddress) {
		return NewServiceError(types.ErrCodeValidation, "钱包地址格式无效", nil)
	}

	// 验证签名格式
	if len(req.Signature) != 132 || !strings.HasPrefix(req.Signature, "0x") {
		return NewServiceError(types.ErrCodeValidation, "签名格式无效", nil)
	}

	return nil
}

// getOrCreateUser 获取或创建用户
// 如果用户不存在则自动创建，这是Web3应用的常见模式
// 参数:
//   - walletAddress: 标准化的钱包地址
//
// 返回:
//   - *models.User: 用户模型
//   - bool: 是否为新创建的用户
//   - error: 操作错误
func (s *authService) getOrCreateUser(walletAddress string) (*models.User, bool, error) {
	// 尝试获取现有用户
	existingUser, err := s.repos.User.GetByWalletAddress(walletAddress)
	if err == nil {
		// 用户已存在
		return existingUser, false, nil
	}

	// 检查是否为"用户不存在"错误
	if repoErr, ok := err.(*repository.RepositoryError); ok {
		if !strings.Contains(repoErr.Error(), "用户不存在") {
			// 其他类型的错误，直接返回
			return nil, false, err
		}
	}

	// 用户不存在，创建新用户
	newUser := &models.User{
		WalletAddress:     walletAddress,
		PreferredLanguage: "en",
		Timezone:          "UTC",
		IsActive:          true,
	}

	if err := s.repos.User.Create(newUser); err != nil {
		return nil, false, err
	}

	// 为新用户创建默认偏好设置
	preferences := &models.UserPreferences{
		UserID: newUser.ID,
		// 其他字段使用数据库默认值
	}

	if err := s.repos.User.CreatePreferences(preferences); err != nil {
		s.logger.Warnf("创建用户偏好设置失败: %v", err)
		// 不影响用户创建流程
	}

	s.logger.Infof("创建新用户: %s (ID: %d)", walletAddress, newUser.ID)
	return newUser, true, nil
}

// convertToUserInfo 将数据库用户模型转换为API响应格式
// 过滤敏感信息，只返回客户端需要的数据
func (s *authService) convertToUserInfo(user *models.User) *types.UserInfo {
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
