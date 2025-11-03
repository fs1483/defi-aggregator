// Package services 代币业务服务实现
// 提供代币信息管理、价格更新、搜索筛选等功能
// 支持多链代币管理，包括以太坊、Polygon等主流区块链
package services

import (
	"strings"
	"time"

	"defi-aggregator/business-logic/internal/models"
	"defi-aggregator/business-logic/internal/repository"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// tokenService 代币业务服务实现
// 负责代币信息管理、价格数据维护、搜索和筛选等功能
type tokenService struct {
	repos  *repository.Repositories // 数据访问层
	cfg    *config.Config           // 应用配置
	logger *logrus.Logger           // 日志记录器
}

// NewTokenService 创建代币服务实例
// 注入必要的依赖，初始化代币管理服务
func NewTokenService(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) TokenService {
	return &tokenService{
		repos:  repos,
		cfg:    cfg,
		logger: logger,
	}
}

// ========================================
// 代币基础操作实现
// ========================================

// GetTokens 获取代币列表
// 支持分页、筛选、搜索等功能，返回符合条件的代币列表
// 参数:
//   - req: 代币列表请求，包含分页、筛选条件等
//
// 返回:
//   - []*types.TokenInfo: 代币信息列表
//   - *types.Meta: 分页元数据
//   - error: 查询错误
func (s *tokenService) GetTokens(req *types.TokenListRequest) ([]*types.TokenInfo, *types.Meta, error) {
	s.logger.Debugf("获取代币列表: chainID=%v, isActive=%v, search=%v",
		req.ChainID, req.IsActive, req.Search)

	// 调用Repository获取代币列表
	tokens, total, err := s.repos.Token.List(req)
	if err != nil {
		s.logger.Errorf("获取代币列表失败: %v", err)
		return nil, nil, NewServiceError(types.ErrCodeInternal, "获取代币列表失败", err)
	}

	// 转换为API响应格式
	var tokenInfos []*types.TokenInfo
	for _, token := range tokens {
		tokenInfo := s.convertToTokenInfo(token)
		tokenInfos = append(tokenInfos, tokenInfo)
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

	s.logger.Debugf("获取代币列表成功: total=%d, page=%d", total, req.Page)
	return tokenInfos, meta, nil
}

// GetTokenByID 根据ID获取代币详情
// 返回指定代币的完整信息，包括价格、市值等数据
// 参数:
//   - id: 代币ID
//
// 返回:
//   - *types.TokenInfo: 代币详细信息
//   - error: 查询错误
func (s *tokenService) GetTokenByID(id uint) (*types.TokenInfo, error) {
	token, err := s.repos.Token.GetByID(id)
	if err != nil {
		s.logger.Warnf("获取代币详情失败: tokenID=%d, error=%v", id, err)
		return nil, NewServiceError(types.ErrCodeNotFound, "代币不存在", err)
	}

	tokenInfo := s.convertToTokenInfo(token)
	s.logger.Debugf("获取代币 %d 详情成功", id)
	return tokenInfo, nil
}

// GetTokenByAddress 根据合约地址获取代币
// 通过链ID和合约地址查找代币，用于交易参数验证
// 参数:
//   - chainID: 区块链ID
//   - address: 代币合约地址
//
// 返回:
//   - *types.TokenInfo: 代币信息
//   - error: 查询错误
func (s *tokenService) GetTokenByAddress(chainID uint, address string) (*types.TokenInfo, error) {
	// 标准化合约地址
	normalizedAddress := strings.ToLower(address)

	token, err := s.repos.Token.GetByContractAddress(chainID, normalizedAddress)
	if err != nil {
		s.logger.Warnf("根据合约地址获取代币失败: chainID=%d, address=%s", chainID, address)
		return nil, NewServiceError(types.ErrCodeNotFound, "代币不存在", err)
	}

	tokenInfo := s.convertToTokenInfo(token)
	s.logger.Debugf("根据合约地址获取代币成功: %s", address)
	return tokenInfo, nil
}

// ========================================
// 代币搜索和筛选实现
// ========================================

// SearchTokens 搜索代币
// 根据代币符号或名称进行模糊搜索
// 参数:
//   - query: 搜索关键词
//
// 返回:
//   - []*types.TokenInfo: 匹配的代币列表
//   - error: 搜索错误
func (s *tokenService) SearchTokens(query string) ([]*types.TokenInfo, error) {
	if strings.TrimSpace(query) == "" {
		return nil, NewServiceError(types.ErrCodeValidation, "搜索关键词不能为空", nil)
	}

	tokens, err := s.repos.Token.Search(query)
	if err != nil {
		s.logger.Errorf("搜索代币失败: query=%s, error=%v", query, err)
		return nil, NewServiceError(types.ErrCodeInternal, "搜索代币失败", err)
	}

	// 转换为API响应格式
	var tokenInfos []*types.TokenInfo
	for _, token := range tokens {
		tokenInfo := s.convertToTokenInfo(token)
		tokenInfos = append(tokenInfos, tokenInfo)
	}

	s.logger.Debugf("搜索代币成功: query=%s, results=%d", query, len(tokenInfos))
	return tokenInfos, nil
}

// GetTokensByChain 获取指定链的代币
// 返回特定区块链上支持的所有代币
// 参数:
//   - chainID: 区块链ID
//
// 返回:
//   - []*types.TokenInfo: 代币列表
//   - error: 查询错误
func (s *tokenService) GetTokensByChain(chainID uint) ([]*types.TokenInfo, error) {
	tokens, err := s.repos.Token.GetByChainID(chainID)
	if err != nil {
		s.logger.Errorf("获取链代币失败: chainID=%d, error=%v", chainID, err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取链代币失败", err)
	}

	// 转换为API响应格式
	var tokenInfos []*types.TokenInfo
	for _, token := range tokens {
		tokenInfo := s.convertToTokenInfo(token)
		tokenInfos = append(tokenInfos, tokenInfo)
	}

	s.logger.Debugf("获取链 %d 的代币成功: count=%d", chainID, len(tokenInfos))
	return tokenInfos, nil
}

// GetVerifiedTokens 获取已验证代币列表
// 返回经过验证的安全代币，用于推荐给用户
// 返回:
//   - []*types.TokenInfo: 已验证代币列表
//   - error: 查询错误
func (s *tokenService) GetVerifiedTokens() ([]*types.TokenInfo, error) {
	tokens, err := s.repos.Token.GetVerifiedTokens()
	if err != nil {
		s.logger.Errorf("获取已验证代币失败: %v", err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取已验证代币失败", err)
	}

	// 转换为API响应格式
	var tokenInfos []*types.TokenInfo
	for _, token := range tokens {
		tokenInfo := s.convertToTokenInfo(token)
		tokenInfos = append(tokenInfos, tokenInfo)
	}

	s.logger.Debugf("获取已验证代币成功: count=%d", len(tokenInfos))
	return tokenInfos, nil
}

// GetPopularTokens 获取热门代币
// 根据交易量、市值等指标返回热门代币列表
// 参数:
//   - limit: 返回数量限制
//
// 返回:
//   - []*types.TokenInfo: 热门代币列表
//   - error: 查询错误
func (s *tokenService) GetPopularTokens(limit int) ([]*types.TokenInfo, error) {
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认返回20个
	}

	// TODO: 实现基于交易量和市值的排序逻辑
	// 目前先返回已验证的活跃代币
	tokens, err := s.repos.Token.GetVerifiedTokens()
	if err != nil {
		s.logger.Errorf("获取热门代币失败: %v", err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取热门代币失败", err)
	}

	// 限制返回数量
	if len(tokens) > limit {
		tokens = tokens[:limit]
	}

	// 转换为API响应格式
	var tokenInfos []*types.TokenInfo
	for _, token := range tokens {
		tokenInfo := s.convertToTokenInfo(token)
		tokenInfos = append(tokenInfos, tokenInfo)
	}

	s.logger.Debugf("获取热门代币成功: count=%d", len(tokenInfos))
	return tokenInfos, nil
}

// ========================================
// 代币价格管理实现
// ========================================

// UpdateTokenPrice 更新代币价格
// 更新指定代币的USD价格，通常由定时任务调用
// 参数:
//   - tokenID: 代币ID
//   - priceUSD: 美元价格字符串
//
// 返回:
//   - error: 更新错误
func (s *tokenService) UpdateTokenPrice(tokenID uint, priceUSD string) error {
	// 验证价格格式
	price, err := decimal.NewFromString(priceUSD)
	if err != nil {
		return NewServiceError(types.ErrCodeValidation, "无效的价格格式", err)
	}

	if price.IsNegative() {
		return NewServiceError(types.ErrCodeValidation, "价格不能为负数", nil)
	}

	// 获取代币信息
	token, err := s.repos.Token.GetByID(tokenID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "代币不存在", err)
	}

	// 更新价格和更新时间
	token.PriceUSD = &price
	now := time.Now()
	token.PriceUpdatedAt = &now

	if err := s.repos.Token.Update(token); err != nil {
		s.logger.Errorf("更新代币价格失败: tokenID=%d, price=%s, error=%v", tokenID, priceUSD, err)
		return NewServiceError(types.ErrCodeInternal, "更新代币价格失败", err)
	}

	s.logger.Infof("代币 %d (%s) 价格更新成功: $%s", tokenID, token.Symbol, priceUSD)
	return nil
}

// RefreshAllPrices 刷新所有代币价格
// 批量更新所有活跃代币的价格，通常由定时任务调用
// 返回:
//   - error: 刷新过程中的错误
func (s *tokenService) RefreshAllPrices() error {
	s.logger.Info("开始刷新所有代币价格...")

	// 获取需要更新价格的代币
	tokens, err := s.repos.Token.GetTokensWithOutdatedPrices()
	if err != nil {
		s.logger.Errorf("获取待更新价格的代币失败: %v", err)
		return NewServiceError(types.ErrCodeInternal, "获取代币列表失败", err)
	}

	if len(tokens) == 0 {
		s.logger.Info("没有需要更新价格的代币")
		return nil
	}

	// TODO: 集成第三方价格API (CoinGecko, CoinMarketCap等)
	// 目前使用模拟价格更新
	successCount := 0
	for _, token := range tokens {
		// 模拟价格更新
		mockPrice := s.generateMockPrice(token)
		if err := s.UpdateTokenPrice(token.ID, mockPrice); err != nil {
			s.logger.Warnf("更新代币 %s 价格失败: %v", token.Symbol, err)
			continue
		}
		successCount++
	}

	s.logger.Infof("代币价格刷新完成: 成功=%d, 总数=%d", successCount, len(tokens))
	return nil
}

// GetPriceHistory 获取代币价格历史
// 返回指定代币的历史价格数据，用于图表展示
// 参数:
//   - tokenID: 代币ID
//   - days: 历史天数
//
// 返回:
//   - []map[string]interface{}: 价格历史数据
//   - error: 查询错误
func (s *tokenService) GetPriceHistory(tokenID uint, days int) ([]map[string]interface{}, error) {
	if days <= 0 || days > 365 {
		days = 30 // 默认30天
	}

	// 验证代币存在
	token, err := s.repos.Token.GetByID(tokenID)
	if err != nil {
		return nil, NewServiceError(types.ErrCodeNotFound, "代币不存在", err)
	}

	// TODO: 实现价格历史查询
	// 需要从时序数据库或第三方API获取历史价格
	// 目前返回模拟数据
	history := s.generateMockPriceHistory(token, days)

	s.logger.Debugf("获取代币 %d 的价格历史成功: days=%d", tokenID, days)
	return history, nil
}

// ========================================
// 代币管理功能（管理员）
// ========================================

// AddToken 添加新代币
// 管理员功能，向系统中添加新的支持代币
// 参数:
//   - tokenInfo: 代币信息
//
// 返回:
//   - error: 添加错误
func (s *tokenService) AddToken(tokenInfo *types.TokenInfo) error {
	// 验证代币信息
	if err := s.validateTokenInfo(tokenInfo); err != nil {
		return err
	}

	// 检查代币是否已存在
	_, err := s.repos.Token.GetByContractAddress(tokenInfo.ChainID, tokenInfo.ContractAddress)
	if err == nil {
		return NewServiceError(types.ErrCodeConflict, "代币已存在", nil)
	}

	// 创建新代币
	token := &models.Token{
		ChainID:         tokenInfo.ChainID,
		ContractAddress: strings.ToLower(tokenInfo.ContractAddress),
		Symbol:          strings.ToUpper(tokenInfo.Symbol),
		Name:            tokenInfo.Name,
		Decimals:        tokenInfo.Decimals,
		LogoURL:         tokenInfo.LogoURL,
		IsNative:        tokenInfo.IsNative,
		IsStable:        tokenInfo.IsStable,
		IsVerified:      false, // 新添加的代币默认未验证
		IsActive:        true,
	}

	if err := s.repos.Token.Create(token); err != nil {
		s.logger.Errorf("添加代币失败: symbol=%s, error=%v", tokenInfo.Symbol, err)
		return NewServiceError(types.ErrCodeInternal, "添加代币失败", err)
	}

	s.logger.Infof("添加新代币成功: %s (ID: %d)", token.Symbol, token.ID)
	return nil
}

// UpdateTokenInfo 更新代币信息
// 管理员功能，更新代币的基本信息
// 参数:
//   - tokenID: 代币ID
//   - updates: 要更新的字段
//
// 返回:
//   - error: 更新错误
func (s *tokenService) UpdateTokenInfo(tokenID uint, updates map[string]interface{}) error {
	// 验证代币存在
	token, err := s.repos.Token.GetByID(tokenID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "代币不存在", err)
	}

	// 应用更新
	// TODO: 实现字段级别的更新验证
	if err := s.repos.Token.Update(token); err != nil {
		s.logger.Errorf("更新代币信息失败: tokenID=%d, error=%v", tokenID, err)
		return NewServiceError(types.ErrCodeInternal, "更新代币信息失败", err)
	}

	s.logger.Infof("代币 %d 信息更新成功", tokenID)
	return nil
}

// VerifyToken 验证代币
// 管理员功能，将代币标记为已验证状态
// 参数:
//   - tokenID: 代币ID
//
// 返回:
//   - error: 验证错误
func (s *tokenService) VerifyToken(tokenID uint) error {
	token, err := s.repos.Token.GetByID(tokenID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "代币不存在", err)
	}

	if token.IsVerified {
		return NewServiceError(types.ErrCodeConflict, "代币已经验证", nil)
	}

	// 标记为已验证
	token.IsVerified = true
	if err := s.repos.Token.Update(token); err != nil {
		s.logger.Errorf("验证代币失败: tokenID=%d, error=%v", tokenID, err)
		return NewServiceError(types.ErrCodeInternal, "验证代币失败", err)
	}

	s.logger.Infof("代币 %d (%s) 验证成功", tokenID, token.Symbol)
	return nil
}

// DeactivateToken 停用代币
// 管理员功能，停用不再支持的代币
// 参数:
//   - tokenID: 代币ID
//
// 返回:
//   - error: 停用错误
func (s *tokenService) DeactivateToken(tokenID uint) error {
	token, err := s.repos.Token.GetByID(tokenID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "代币不存在", err)
	}

	if !token.IsActive {
		return NewServiceError(types.ErrCodeConflict, "代币已处于停用状态", nil)
	}

	// 停用代币
	token.IsActive = false
	if err := s.repos.Token.Update(token); err != nil {
		s.logger.Errorf("停用代币失败: tokenID=%d, error=%v", tokenID, err)
		return NewServiceError(types.ErrCodeInternal, "停用代币失败", err)
	}

	s.logger.Infof("代币 %d (%s) 已停用", tokenID, token.Symbol)
	return nil
}

// ========================================
// 辅助方法
// ========================================

// validateTokenInfo 验证代币信息的有效性
func (s *tokenService) validateTokenInfo(tokenInfo *types.TokenInfo) error {
	if tokenInfo.ChainID == 0 {
		return NewServiceError(types.ErrCodeValidation, "链ID不能为空", nil)
	}

	if tokenInfo.ContractAddress == "" {
		return NewServiceError(types.ErrCodeValidation, "合约地址不能为空", nil)
	}

	if !strings.HasPrefix(tokenInfo.ContractAddress, "0x") || len(tokenInfo.ContractAddress) != 42 {
		return NewServiceError(types.ErrCodeValidation, "无效的合约地址格式", nil)
	}

	if tokenInfo.Symbol == "" {
		return NewServiceError(types.ErrCodeValidation, "代币符号不能为空", nil)
	}

	if tokenInfo.Name == "" {
		return NewServiceError(types.ErrCodeValidation, "代币名称不能为空", nil)
	}

	if tokenInfo.Decimals < 0 || tokenInfo.Decimals > 30 {
		return NewServiceError(types.ErrCodeValidation, "代币精度必须在0-30之间", nil)
	}

	return nil
}

// convertToTokenInfo 将数据库模型转换为API响应格式
func (s *tokenService) convertToTokenInfo(token *models.Token) *types.TokenInfo {
	tokenInfo := &types.TokenInfo{
		ID:              token.ID,
		ChainID:         token.ChainID,
		ContractAddress: token.ContractAddress,
		Symbol:          token.Symbol,
		Name:            token.Name,
		Decimals:        token.Decimals,
		LogoURL:         token.LogoURL,
		IsNative:        token.IsNative,
		IsStable:        token.IsStable,
		IsVerified:      token.IsVerified,
		IsActive:        token.IsActive,
	}

	// 设置价格信息（如果存在）
	if token.PriceUSD != nil {
		tokenInfo.PriceUSD = token.PriceUSD
	}
	if token.DailyVolumeUSD != nil {
		tokenInfo.DailyVolumeUSD = token.DailyVolumeUSD
	}
	if token.MarketCapUSD != nil {
		tokenInfo.MarketCapUSD = token.MarketCapUSD
	}
	if token.PriceUpdatedAt != nil {
		tokenInfo.PriceUpdatedAt = token.PriceUpdatedAt
	}

	return tokenInfo
}

// GetTokensWithChainInfo 获取代币列表（包含链信息）
// 新增方法：同时返回代币和关联的链信息，供前端直接使用
func (s *tokenService) GetTokensWithChainInfo(req *types.TokenListRequest) ([]*types.TokenInfoWithChain, *types.Meta, error) {
	s.logger.Debugf("获取代币列表（含链信息）: chainID=%v, isActive=%v, search=%v",
		req.ChainID, req.IsActive, req.Search)

	// 调用Repository获取代币列表（这里需要扩展Repository支持JOIN查询）
	tokens, total, err := s.repos.Token.List(req)
	if err != nil {
		s.logger.Errorf("获取代币列表失败: %v", err)
		return nil, nil, NewServiceError(types.ErrCodeInternal, "获取代币列表失败", err)
	}

	// 转换为包含链信息的API响应格式
	var tokenInfos []*types.TokenInfoWithChain
	for _, token := range tokens {
		// 获取链信息
		chain, err := s.repos.Chain.GetByID(token.ChainID)
		if err != nil {
			s.logger.Warnf("获取链信息失败: chainID=%d, error=%v", token.ChainID, err)
			continue
		}

		tokenInfo := &types.TokenInfoWithChain{
			TokenInfo: *s.convertToTokenInfo(token),
			Chain: types.ChainInfo{
				ID:           chain.ID,
				ChainID:      chain.ChainID,
				Name:         chain.Name,
				DisplayName:  chain.DisplayName,
				Symbol:       chain.Symbol,
				IsTestnet:    chain.IsTestnet,
				IsActive:     chain.IsActive,
				GasPriceGwei: chain.GasPriceGwei,
				BlockTimeSec: chain.BlockTimeSec,
			},
		}
		tokenInfos = append(tokenInfos, tokenInfo)
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

	s.logger.Debugf("获取代币列表（含链信息）成功: total=%d, page=%d", total, req.Page)
	return tokenInfos, meta, nil
}

// generateMockPrice 生成模拟价格（开发用）
// 仅用于开发和测试环境
func (s *tokenService) generateMockPrice(token *models.Token) string {
	// 根据代币类型生成模拟价格
	switch strings.ToUpper(token.Symbol) {
	case "ETH", "WETH":
		return "2000.00"
	case "BTC", "WBTC":
		return "43000.00"
	case "USDC", "USDT", "DAI":
		return "1.00"
	case "MATIC":
		return "0.85"
	case "UNI":
		return "6.50"
	case "LINK":
		return "14.20"
	case "AAVE":
		return "95.30"
	default:
		return "1.00"
	}
}

// generateMockPriceHistory 生成模拟价格历史（开发用）
func (s *tokenService) generateMockPriceHistory(token *models.Token, days int) []map[string]interface{} {
	var history []map[string]interface{}
	basePrice, _ := decimal.NewFromString(s.generateMockPrice(token))

	for i := days; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		// 生成轻微波动的价格
		variation := decimal.NewFromFloat(0.95 + (float64(i%10) * 0.01))
		price := basePrice.Mul(variation)

		history = append(history, map[string]interface{}{
			"date":  date.Format("2006-01-02"),
			"price": price.String(),
		})
	}

	return history
}
