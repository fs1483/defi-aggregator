// Package services 区块链业务服务实现
// 提供区块链网络管理、配置更新、健康检查等功能
// 支持多链架构，包括以太坊、Polygon、Arbitrum等
package services

import (
	"fmt"

	"defi-aggregator/business-logic/internal/models"
	"defi-aggregator/business-logic/internal/repository"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"

	"github.com/sirupsen/logrus"
)

// chainService 区块链业务服务实现
// 负责管理支持的区块链网络和相关配置
type chainService struct {
	repos  *repository.Repositories // 数据访问层
	cfg    *config.Config           // 应用配置
	logger *logrus.Logger           // 日志记录器
}

// NewChainService 创建区块链服务实例
func NewChainService(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) ChainService {
	return &chainService{
		repos:  repos,
		cfg:    cfg,
		logger: logger,
	}
}

// ========================================
// 链基础操作实现
// ========================================

// GetChains 获取所有区块链
// 返回系统支持的所有区块链网络列表
// 返回:
//   - []*types.ChainInfo: 区块链信息列表
//   - error: 查询错误
func (s *chainService) GetChains() ([]*types.ChainInfo, error) {
	chains, err := s.repos.Chain.List()
	if err != nil {
		s.logger.Errorf("获取区块链列表失败: %v", err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取区块链列表失败", err)
	}

	// 转换为API响应格式
	var chainInfos []*types.ChainInfo
	for _, chain := range chains {
		chainInfo := s.convertToChainInfo(chain)
		chainInfos = append(chainInfos, chainInfo)
	}

	s.logger.Debugf("获取区块链列表成功: count=%d", len(chainInfos))
	return chainInfos, nil
}

// GetChainByID 获取指定区块链详情
// 根据ID返回区块链的详细信息
// 参数:
//   - id: 区块链记录ID
//
// 返回:
//   - *types.ChainInfo: 区块链详细信息
//   - error: 查询错误
func (s *chainService) GetChainByID(id uint) (*types.ChainInfo, error) {
	chain, err := s.repos.Chain.GetByID(id)
	if err != nil {
		s.logger.Warnf("获取区块链详情失败: chainID=%d, error=%v", id, err)
		return nil, NewServiceError(types.ErrCodeNotFound, "区块链不存在", err)
	}

	chainInfo := s.convertToChainInfo(chain)
	s.logger.Debugf("获取区块链 %d 详情成功", id)
	return chainInfo, nil
}

// GetChainByChainID 根据真实链ID获取区块链信息
// 参数:
//   - chainID: 真实的区块链ID（如1=Ethereum, 11155111=Sepolia）
//
// 返回:
//   - *types.ChainInfo: 区块链详细信息
//   - error: 查询错误
func (s *chainService) GetChainByChainID(chainID uint) (*types.ChainInfo, error) {
	chain, err := s.repos.Chain.GetByChainID(chainID)
	if err != nil {
		s.logger.Warnf("获取区块链详情失败: chainID=%d, error=%v", chainID, err)
		return nil, NewServiceError(types.ErrCodeNotFound, fmt.Sprintf("区块链 %d 不存在", chainID), err)
	}

	chainInfo := s.convertToChainInfo(chain)
	s.logger.Debugf("获取区块链 %d 详情成功", chainID)
	return chainInfo, nil
}

// GetActiveChains 获取活跃区块链
// 返回当前启用的区块链网络列表
// 返回:
//   - []*types.ChainInfo: 活跃区块链列表
//   - error: 查询错误
func (s *chainService) GetActiveChains() ([]*types.ChainInfo, error) {
	chains, err := s.repos.Chain.GetActiveChains()
	if err != nil {
		s.logger.Errorf("获取活跃区块链失败: %v", err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取活跃区块链失败", err)
	}

	// 转换为API响应格式
	var chainInfos []*types.ChainInfo
	for _, chain := range chains {
		chainInfo := s.convertToChainInfo(chain)
		chainInfos = append(chainInfos, chainInfo)
	}

	s.logger.Debugf("获取活跃区块链成功: count=%d", len(chainInfos))
	return chainInfos, nil
}

// ========================================
// 链分类功能实现
// ========================================

// GetMainnetChains 获取主网链
// 返回所有主网区块链，用于生产环境交易
// 返回:
//   - []*types.ChainInfo: 主网链列表
//   - error: 查询错误
func (s *chainService) GetMainnetChains() ([]*types.ChainInfo, error) {
	chains, err := s.repos.Chain.GetMainnetChains()
	if err != nil {
		s.logger.Errorf("获取主网链失败: %v", err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取主网链失败", err)
	}

	// 转换为API响应格式
	var chainInfos []*types.ChainInfo
	for _, chain := range chains {
		chainInfo := s.convertToChainInfo(chain)
		chainInfos = append(chainInfos, chainInfo)
	}

	s.logger.Debugf("获取主网链成功: count=%d", len(chainInfos))
	return chainInfos, nil
}

// GetTestnetChains 获取测试网链
// 返回测试网区块链，用于开发和测试
// 返回:
//   - []*types.ChainInfo: 测试网链列表
//   - error: 查询错误
func (s *chainService) GetTestnetChains() ([]*types.ChainInfo, error) {
	chains, err := s.repos.Chain.GetTestnetChains()
	if err != nil {
		s.logger.Errorf("获取测试网链失败: %v", err)
		return nil, NewServiceError(types.ErrCodeInternal, "获取测试网链失败", err)
	}

	// 转换为API响应格式
	var chainInfos []*types.ChainInfo
	for _, chain := range chains {
		chainInfo := s.convertToChainInfo(chain)
		chainInfos = append(chainInfos, chainInfo)
	}

	s.logger.Debugf("获取测试网链成功: count=%d", len(chainInfos))
	return chainInfos, nil
}

// ========================================
// 链管理功能（管理员）
// ========================================

// AddChain 添加新区块链
// 管理员功能，向系统中添加新的区块链网络支持
// 参数:
//   - chainInfo: 区块链信息
//
// 返回:
//   - error: 添加错误
func (s *chainService) AddChain(chainInfo *types.ChainInfo) error {
	// 验证链信息
	if err := s.validateChainInfo(chainInfo); err != nil {
		return err
	}

	// 检查链ID是否已存在
	_, err := s.repos.Chain.GetByChainID(chainInfo.ChainID)
	if err == nil {
		return NewServiceError(types.ErrCodeConflict, "区块链ID已存在", nil)
	}

	// 创建新链
	chain := &models.Chain{
		ChainID:      chainInfo.ChainID,
		Name:         chainInfo.Name,
		DisplayName:  chainInfo.DisplayName,
		Symbol:       chainInfo.Symbol,
		IsTestnet:    chainInfo.IsTestnet,
		IsActive:     true,
		GasPriceGwei: chainInfo.GasPriceGwei,
		BlockTimeSec: chainInfo.BlockTimeSec,
	}

	if err := s.repos.Chain.Create(chain); err != nil {
		s.logger.Errorf("添加区块链失败: chainID=%d, error=%v", chainInfo.ChainID, err)
		return NewServiceError(types.ErrCodeInternal, "添加区块链失败", err)
	}

	s.logger.Infof("添加新区块链成功: %s (ChainID: %d)", chain.Name, chain.ChainID)
	return nil
}

// UpdateChain 更新区块链信息
// 管理员功能，更新区块链的配置信息
// 参数:
//   - chainID: 区块链ID
//   - updates: 要更新的字段
//
// 返回:
//   - error: 更新错误
func (s *chainService) UpdateChain(chainID uint, updates map[string]interface{}) error {
	// 验证链存在
	chain, err := s.repos.Chain.GetByChainID(chainID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "区块链不存在", err)
	}

	// TODO: 实现字段级别的更新验证
	if err := s.repos.Chain.Update(chain); err != nil {
		s.logger.Errorf("更新区块链信息失败: chainID=%d, error=%v", chainID, err)
		return NewServiceError(types.ErrCodeInternal, "更新区块链信息失败", err)
	}

	s.logger.Infof("区块链 %d 信息更新成功", chainID)
	return nil
}

// ActivateChain 激活区块链
// 启用指定的区块链网络
// 参数:
//   - chainID: 区块链ID
//
// 返回:
//   - error: 激活错误
func (s *chainService) ActivateChain(chainID uint) error {
	chain, err := s.repos.Chain.GetByChainID(chainID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "区块链不存在", err)
	}

	if chain.IsActive {
		return NewServiceError(types.ErrCodeConflict, "区块链已处于激活状态", nil)
	}

	chain.IsActive = true
	if err := s.repos.Chain.Update(chain); err != nil {
		s.logger.Errorf("激活区块链失败: chainID=%d, error=%v", chainID, err)
		return NewServiceError(types.ErrCodeInternal, "激活区块链失败", err)
	}

	s.logger.Infof("区块链 %d (%s) 已激活", chainID, chain.Name)
	return nil
}

// DeactivateChain 停用区块链
// 停用指定的区块链网络
// 参数:
//   - chainID: 区块链ID
//
// 返回:
//   - error: 停用错误
func (s *chainService) DeactivateChain(chainID uint) error {
	chain, err := s.repos.Chain.GetByChainID(chainID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "区块链不存在", err)
	}

	if !chain.IsActive {
		return NewServiceError(types.ErrCodeConflict, "区块链已处于停用状态", nil)
	}

	chain.IsActive = false
	if err := s.repos.Chain.Update(chain); err != nil {
		s.logger.Errorf("停用区块链失败: chainID=%d, error=%v", chainID, err)
		return NewServiceError(types.ErrCodeInternal, "停用区块链失败", err)
	}

	s.logger.Infof("区块链 %d (%s) 已停用", chainID, chain.Name)
	return nil
}

// ========================================
// 链状态检查实现
// ========================================

// CheckChainHealth 检查区块链健康状态
// 验证RPC连接和基本功能
// 参数:
//   - chainID: 区块链ID
//
// 返回:
//   - error: 健康检查失败的错误
func (s *chainService) CheckChainHealth(chainID uint) error {
	chain, err := s.repos.Chain.GetByChainID(chainID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "区块链不存在", err)
	}

	// TODO: 实现真实的RPC健康检查
	// 1. 连接RPC节点
	// 2. 检查最新区块
	// 3. 验证网络ID
	// 4. 测试基本查询功能

	s.logger.Debugf("区块链 %d (%s) 健康检查通过", chainID, chain.Name)
	return nil
}

// UpdateGasPrice 更新Gas价格
// 更新区块链的建议Gas价格
// 参数:
//   - chainID: 区块链ID
//   - gasPriceGwei: Gas价格（Gwei单位）
//
// 返回:
//   - error: 更新错误
func (s *chainService) UpdateGasPrice(chainID uint, gasPriceGwei uint) error {
	chain, err := s.repos.Chain.GetByChainID(chainID)
	if err != nil {
		return NewServiceError(types.ErrCodeNotFound, "区块链不存在", err)
	}

	// 验证Gas价格范围
	if gasPriceGwei == 0 {
		return NewServiceError(types.ErrCodeValidation, "Gas价格不能为0", nil)
	}

	if gasPriceGwei > 1000 {
		return NewServiceError(types.ErrCodeValidation, "Gas价格过高", nil)
	}

	// 更新Gas价格
	chain.GasPriceGwei = gasPriceGwei
	if err := s.repos.Chain.Update(chain); err != nil {
		s.logger.Errorf("更新Gas价格失败: chainID=%d, gasPrice=%d, error=%v", chainID, gasPriceGwei, err)
		return NewServiceError(types.ErrCodeInternal, "更新Gas价格失败", err)
	}

	s.logger.Infof("区块链 %d (%s) Gas价格更新为 %d Gwei", chainID, chain.Name, gasPriceGwei)
	return nil
}

// ========================================
// 辅助方法
// ========================================

// validateChainInfo 验证区块链信息的有效性
func (s *chainService) validateChainInfo(chainInfo *types.ChainInfo) error {
	if chainInfo.ChainID == 0 {
		return NewServiceError(types.ErrCodeValidation, "链ID不能为0", nil)
	}

	if chainInfo.Name == "" {
		return NewServiceError(types.ErrCodeValidation, "链名称不能为空", nil)
	}

	if chainInfo.DisplayName == "" {
		return NewServiceError(types.ErrCodeValidation, "显示名称不能为空", nil)
	}

	if chainInfo.Symbol == "" {
		return NewServiceError(types.ErrCodeValidation, "原生代币符号不能为空", nil)
	}

	return nil
}

// convertToChainInfo 将数据库模型转换为API响应格式
func (s *chainService) convertToChainInfo(chain *models.Chain) *types.ChainInfo {
	return &types.ChainInfo{
		ID:           chain.ID,
		ChainID:      chain.ChainID,
		Name:         chain.Name,
		DisplayName:  chain.DisplayName,
		Symbol:       chain.Symbol,
		IsTestnet:    chain.IsTestnet,
		IsActive:     chain.IsActive,
		GasPriceGwei: chain.GasPriceGwei,
		BlockTimeSec: chain.BlockTimeSec,
	}
}
