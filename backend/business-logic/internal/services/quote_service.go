// Package services 报价业务服务实现
// 实现DeFi聚合器的核心报价功能，集成智能路由服务
// 提供报价缓存、历史记录、性能监控等企业级功能
package services

import (
	"context"
	"fmt"
	"time"

	"defi-aggregator/business-logic/internal/models"
	"defi-aggregator/business-logic/internal/repository"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"
	"defi-aggregator/business-logic/pkg/utils"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// quoteService 报价业务服务实现
// 负责协调智能路由服务，管理报价历史，提供缓存策略
type quoteService struct {
	repos      *repository.Repositories // 数据访问层
	cfg        *config.Config           // 应用配置
	logger     *logrus.Logger           // 日志记录器
	httpClient utils.HTTPClient         // HTTP客户端
}

// SmartRouterQuoteRequest 智能路由服务请求格式
// 对应智能路由服务的API接口格式
type SmartRouterQuoteRequest struct {
	RequestID   string          `json:"request_id"`             // 请求ID
	FromToken   string          `json:"from_token"`             // 源代币合约地址
	ToToken     string          `json:"to_token"`               // 目标代币合约地址
	AmountIn    decimal.Decimal `json:"amount_in"`              // 输入数量
	ChainID     uint            `json:"chain_id"`               // 链ID
	Slippage    decimal.Decimal `json:"slippage"`               // 滑点
	UserAddress string          `json:"user_address,omitempty"` // 用户地址
}

// SmartRouterQuoteResponse 智能路由服务响应格式
type SmartRouterQuoteResponse struct {
	Success   bool                  `json:"success"`
	Data      *SmartRouterQuoteData `json:"data"`
	Error     *SmartRouterError     `json:"error,omitempty"`
	Timestamp int64                 `json:"timestamp"`
	RequestID string                `json:"request_id"`
}

// SmartRouterQuoteData 智能路由报价数据
type SmartRouterQuoteData struct {
	RequestID       string          `json:"request_id"`
	BestProvider    string          `json:"best_provider"`
	BestPrice       decimal.Decimal `json:"best_price"`
	BestGasEstimate uint64          `json:"best_gas_estimate"`
	PriceImpact     decimal.Decimal `json:"price_impact"`
	ExchangeRate    decimal.Decimal `json:"exchange_rate"`
	Route           []RouteStep     `json:"route,omitempty"`
	ValidUntil      time.Time       `json:"valid_until"`
	CacheHit        bool            `json:"cache_hit"`
	Performance     Performance     `json:"performance"`
}

// RouteStep 交易路径步骤
type RouteStep struct {
	Protocol   string          `json:"protocol"`
	Percentage decimal.Decimal `json:"percentage"`
}

// Performance 性能指标
type Performance struct {
	TotalDuration    int64 `json:"total_duration"` // 改为int64以匹配Smart Router的time.Duration序列化
	ProvidersQueried int   `json:"providers_queried"`
	ProvidersSuccess int   `json:"providers_success"`
}

// SmartRouterError 智能路由错误
type SmartRouterError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewQuoteService 创建报价服务实例
func NewQuoteService(repos *repository.Repositories, cfg *config.Config, logger *logrus.Logger) QuoteService {
	// 创建HTTP客户端用于调用智能路由服务
	httpClient := utils.NewHTTPClient(30*time.Second, 2, logger)

	return &quoteService{
		repos:      repos,
		cfg:        cfg,
		logger:     logger,
		httpClient: httpClient,
	}
}

// ========================================
// 报价核心功能实现
// ========================================

// GetQuote 获取最优报价
// 调用智能路由服务获取聚合报价，记录请求历史
// 参数:
//   - req: 报价请求参数
//
// 返回:
//   - *types.QuoteResponse: 最优报价响应
//   - error: 获取过程中的错误
func (s *quoteService) GetQuote(req *types.QuoteRequest) (*types.QuoteResponse, error) {
	requestID := uuid.New().String()
	startTime := time.Now()

	s.logger.Infof("[%s] 开始处理报价请求: %d->%d, amount=%s",
		requestID, req.FromTokenID, req.ToTokenID, req.AmountIn.String())

	// 1. 验证请求参数
	if err := s.validateQuoteRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取代币信息
	fromToken, toToken, fromTokenChain, _, err := s.getTokenInfo(req.FromTokenID, req.ToTokenID, req.ChainID)
	if err != nil {
		return nil, err
	}

	// 3. 记录报价请求到数据库
	quoteRequest, err := s.createQuoteRequest(req, requestID, fromToken, toToken)
	if err != nil {
		s.logger.Warnf("[%s] 记录报价请求失败: %v", requestID, err)
		// 不影响主流程，继续处理
	}

	// 4. 调用智能路由服务
	// 使用数据库中的外部chain_id，这样智能路由服务可以正确识别网络
	smartRouterReq := &SmartRouterQuoteRequest{
		RequestID: requestID,
		FromToken: fromToken.ContractAddress,
		ToToken:   toToken.ContractAddress,
		AmountIn:  req.AmountIn,
		ChainID:   fromTokenChain.ChainID, // 使用外部chain_id（真实的区块链ID）
		Slippage:  req.Slippage,
	}

	// 设置用户地址（如果存在）
	if req.UserAddress != nil {
		smartRouterReq.UserAddress = *req.UserAddress
	}

	routerResponse, err := s.callSmartRouter(smartRouterReq)
	if err != nil {
		// 更新数据库记录为失败状态
		if quoteRequest != nil {
			s.updateQuoteRequestStatus(quoteRequest.ID, "failed", err.Error())
		}
		return nil, err
	}

	// 5. 转换为业务层响应格式
	response := s.convertToQuoteResponse(routerResponse, fromToken, toToken, startTime)

	// 6. 更新数据库记录为成功状态
	if quoteRequest != nil {
		s.updateQuoteRequestSuccess(quoteRequest, routerResponse)
	}

	s.logger.Infof("[%s] 报价请求处理完成: provider=%s, duration=%v",
		requestID, response.BestAggregator, time.Since(startTime))

	return response, nil
}

// GetQuoteHistory 获取报价历史
// 返回用户的历史报价记录，支持分页
// 参数:
//   - userID: 用户ID（可为空，返回所有用户的记录）
//   - req: 分页请求参数
//
// 返回:
//   - []*types.QuoteResponse: 报价历史列表
//   - *types.Meta: 分页元数据
//   - error: 查询错误
func (s *quoteService) GetQuoteHistory(userID *uint, req *types.PaginationRequest) ([]*types.QuoteResponse, *types.Meta, error) {
	s.logger.Debugf("获取报价历史: userID=%v, page=%d", userID, req.Page)

	// 从数据库获取报价请求历史
	var quoteRequests []*models.QuoteRequest
	var total int64
	var err error

	if userID != nil {
		// 获取特定用户的报价历史
		quoteRequests, total, err = s.repos.QuoteRequest.GetByUserID(*userID, req)
	} else {
		// 获取所有报价历史
		quoteRequests, total, err = s.repos.QuoteRequest.List(req)
	}

	if err != nil {
		s.logger.Errorf("获取报价历史失败: %v", err)
		return nil, nil, NewServiceError(types.ErrCodeInternal, "获取报价历史失败", err)
	}

	// 转换为API响应格式
	var responses []*types.QuoteResponse
	for _, quoteReq := range quoteRequests {
		response := s.convertQuoteRequestToResponse(quoteReq)
		responses = append(responses, response)
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

	s.logger.Debugf("获取报价历史成功: total=%d", total)
	return responses, meta, nil
}

// GetQuoteDetails 获取报价详情
// 根据请求ID获取详细的报价信息
// 参数:
//   - requestID: 报价请求ID
//
// 返回:
//   - *types.QuoteResponse: 报价详情
//   - error: 查询错误
func (s *quoteService) GetQuoteDetails(requestID string) (*types.QuoteResponse, error) {
	quoteRequest, err := s.repos.QuoteRequest.GetByRequestID(requestID)
	if err != nil {
		s.logger.Warnf("获取报价详情失败: requestID=%s, error=%v", requestID, err)
		return nil, NewServiceError(types.ErrCodeNotFound, "报价请求不存在", err)
	}

	response := s.convertQuoteRequestToResponse(quoteRequest)
	s.logger.Debugf("获取报价详情成功: requestID=%s", requestID)
	return response, nil
}

// ========================================
// 智能路由服务调用
// ========================================

// callSmartRouter 调用智能路由服务
// 发送HTTP请求到智能路由服务获取聚合报价
func (s *quoteService) callSmartRouter(req *SmartRouterQuoteRequest) (*SmartRouterQuoteResponse, error) {
	// 构建智能路由服务URL
	smartRouterURL := fmt.Sprintf("%s/api/v1/quote", s.cfg.ExternalServices.SmartRouterURL)

	s.logger.Debugf("[%s] 调用智能路由服务: %s", req.RequestID, smartRouterURL)
	s.logger.Debugf("[%s] 请求参数: %+v", req.RequestID, req)

	// 创建上下文（带超时）
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ExternalServices.Timeout)
	defer cancel()

	// 发送请求
	var response SmartRouterQuoteResponse
	if err := s.httpClient.PostJSON(ctx, smartRouterURL, req, &response); err != nil {
		s.logger.Errorf("[%s] 智能路由服务HTTP调用失败: URL=%s, 错误=%v", req.RequestID, smartRouterURL, err)
		s.logger.Errorf("[%s] 请求详情: FromToken=%s, ToToken=%s, ChainID=%d", req.RequestID, req.FromToken, req.ToToken, req.ChainID)
		return nil, NewServiceError(types.ErrCodeExternalAPI, fmt.Sprintf("智能路由服务调用失败: %v", err), err)
	}

	// 检查响应状态
	if !response.Success {
		errorMsg := "智能路由服务返回错误"
		if response.Error != nil {
			errorMsg = response.Error.Message
		}
		return nil, NewServiceError(types.ErrCodeExternalAPI, errorMsg, nil)
	}

	if response.Data == nil {
		return nil, NewServiceError(types.ErrCodeExternalAPI, "智能路由服务返回空数据", nil)
	}

	s.logger.Infof("[%s] 智能路由服务调用成功: provider=%s",
		req.RequestID, response.Data.BestProvider)

	return &response, nil
}

// ========================================
// 数据库操作
// ========================================

// createQuoteRequest 创建报价请求记录
func (s *quoteService) createQuoteRequest(req *types.QuoteRequest, requestID string, fromToken, toToken *models.Token) (*models.QuoteRequest, error) {
	quoteRequest := &models.QuoteRequest{
		RequestID:     requestID,
		ChainID:       req.ChainID,
		FromTokenID:   req.FromTokenID,
		ToTokenID:     req.ToTokenID,
		AmountIn:      req.AmountIn,
		Slippage:      req.Slippage,
		IPAddress:     "127.0.0.1", // 默认本地IP，避免inet类型错误
		RequestSource: "api",
		Status:        "pending",
	}

	// 设置用户地址（如果存在）
	if req.UserAddress != nil {
		quoteRequest.UserAddress = *req.UserAddress
	}

	// 如果有用户ID，设置用户关联
	// TODO: 从上下文或JWT中获取用户ID

	if err := s.repos.QuoteRequest.Create(quoteRequest); err != nil {
		return nil, fmt.Errorf("创建报价请求记录失败: %w", err)
	}

	return quoteRequest, nil
}

// updateQuoteRequestStatus 更新报价请求状态
func (s *quoteService) updateQuoteRequestStatus(quoteRequestID uint, status, errorMessage string) {
	// 异步更新，不影响主流程
	go func() {
		if quoteReq, err := s.repos.QuoteRequest.GetByID(quoteRequestID); err == nil {
			quoteReq.Status = status
			quoteReq.ErrorMessage = errorMessage
			now := time.Now()
			quoteReq.CompletedAt = &now

			if updateErr := s.repos.QuoteRequest.Update(quoteReq); updateErr != nil {
				s.logger.Warnf("更新报价请求状态失败: %v", updateErr)
			}
		}
	}()
}

// updateQuoteRequestSuccess 更新报价请求为成功状态
func (s *quoteService) updateQuoteRequestSuccess(quoteRequest *models.QuoteRequest, routerResponse *SmartRouterQuoteResponse) {
	// 异步更新
	go func() {
		quoteRequest.Status = "completed"
		now := time.Now()
		quoteRequest.CompletedAt = &now

		// 更新最佳结果信息
		if routerResponse.Data != nil {
			quoteRequest.BestAmountOut = &routerResponse.Data.BestPrice
			quoteRequest.BestGasEstimate = &routerResponse.Data.BestGasEstimate
			quoteRequest.BestPriceImpact = &routerResponse.Data.PriceImpact
			quoteRequest.CacheHit = routerResponse.Data.CacheHit

			// TODO: 根据provider名称查找aggregator_id
		}

		if err := s.repos.QuoteRequest.Update(quoteRequest); err != nil {
			s.logger.Warnf("更新报价请求成功状态失败: %v", err)
		}
	}()
}

// ========================================
// 数据转换方法
// ========================================

// convertToQuoteResponse 转换智能路由响应为业务层响应
func (s *quoteService) convertToQuoteResponse(routerResp *SmartRouterQuoteResponse, fromToken, toToken *models.Token, startTime time.Time) *types.QuoteResponse {
	data := routerResp.Data

	// 转换路径信息
	var route []types.RouteInfo
	for _, step := range data.Route {
		route = append(route, types.RouteInfo{
			Protocol:   step.Protocol,
			Percentage: step.Percentage,
		})
	}

	return &types.QuoteResponse{
		RequestID:       data.RequestID,
		FromToken:       *s.convertTokenToInfo(fromToken),
		ToToken:         *s.convertTokenToInfo(toToken),
		AmountIn:        data.BestPrice, // TODO: 使用原始输入数量
		AmountOut:       data.BestPrice,
		BestAggregator:  data.BestProvider,
		GasEstimate:     data.BestGasEstimate,
		PriceImpact:     data.PriceImpact,
		ExchangeRate:    data.ExchangeRate,
		Route:           route,
		ValidUntil:      data.ValidUntil,
		TotalDurationMS: int(time.Since(startTime).Milliseconds()),
		CacheHit:        data.CacheHit,
	}
}

// convertQuoteRequestToResponse 将数据库记录转换为API响应
func (s *quoteService) convertQuoteRequestToResponse(quoteReq *models.QuoteRequest) *types.QuoteResponse {
	response := &types.QuoteResponse{
		RequestID:      quoteReq.RequestID,
		BestAggregator: "unknown", // TODO: 从aggregator_id获取名称
		CacheHit:       quoteReq.CacheHit,
	}

	// 设置最佳结果（如果存在）
	if quoteReq.BestAmountOut != nil {
		response.AmountOut = *quoteReq.BestAmountOut
	}
	if quoteReq.BestGasEstimate != nil {
		response.GasEstimate = *quoteReq.BestGasEstimate
	}
	if quoteReq.BestPriceImpact != nil {
		response.PriceImpact = *quoteReq.BestPriceImpact
	}

	// 设置代币信息（需要预加载）
	if quoteReq.FromToken.ID > 0 {
		response.FromToken = *s.convertTokenToInfo(&quoteReq.FromToken)
	}
	if quoteReq.ToToken.ID > 0 {
		response.ToToken = *s.convertTokenToInfo(&quoteReq.ToToken)
	}

	return response
}

// convertTokenToInfo 将Token模型转换为TokenInfo
func (s *quoteService) convertTokenToInfo(token *models.Token) *types.TokenInfo {
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

	// 设置价格信息
	if token.PriceUSD != nil {
		tokenInfo.PriceUSD = token.PriceUSD
	}

	return tokenInfo
}

// ========================================
// 验证和辅助方法
// ========================================

// validateQuoteRequest 验证报价请求
func (s *quoteService) validateQuoteRequest(req *types.QuoteRequest) error {
	if req.FromTokenID == 0 {
		return NewServiceError(types.ErrCodeValidation, "源代币ID不能为0", nil)
	}

	if req.ToTokenID == 0 {
		return NewServiceError(types.ErrCodeValidation, "目标代币ID不能为0", nil)
	}

	if req.FromTokenID == req.ToTokenID {
		return NewServiceError(types.ErrCodeValidation, "源代币和目标代币不能相同", nil)
	}

	if req.AmountIn.IsZero() || req.AmountIn.IsNegative() {
		return NewServiceError(types.ErrCodeValidation, "输入数量必须大于0", nil)
	}

	if req.ChainID == 0 {
		return NewServiceError(types.ErrCodeValidation, "链ID不能为0", nil)
	}

	if req.Slippage.IsNegative() || req.Slippage.GreaterThan(decimal.NewFromFloat(0.5)) {
		return NewServiceError(types.ErrCodeValidation, "滑点必须在0-50%之间", nil)
	}

	return nil
}

// getTokenInfo 获取代币信息
func (s *quoteService) getTokenInfo(fromTokenID, toTokenID uint, requestChainID uint) (*models.Token, *models.Token, *models.Chain, *models.Chain, error) {
	// 获取源代币信息
	fromToken, err := s.repos.Token.GetByID(fromTokenID)
	if err != nil {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeNotFound, "源代币不存在", err)
	}

	// 获取目标代币信息
	toToken, err := s.repos.Token.GetByID(toTokenID)
	if err != nil {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeNotFound, "目标代币不存在", err)
	}

	// 验证代币是否在同一链上
	if fromToken.ChainID != toToken.ChainID {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeValidation, "代币必须在同一区块链上", nil)
	}

	// 验证代币是否在同一链上，这里requestChainID应该是外部链ID，需要与数据库中chains表的chain_id字段对比
	// 首先获取代币所在的链信息
	fromTokenChain, err := s.repos.Chain.GetByID(fromToken.ChainID)
	if err != nil {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeValidation, "获取源代币链信息失败", err)
	}

	toTokenChain, err := s.repos.Chain.GetByID(toToken.ChainID)
	if err != nil {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeValidation, "获取目标代币链信息失败", err)
	}

	// 验证请求的链ID是否与代币所在链匹配
	if fromTokenChain.ChainID != requestChainID {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeValidation, fmt.Sprintf("源代币不在请求的区块链上: 代币链ID=%d, 请求链ID=%d", fromTokenChain.ChainID, requestChainID), nil)
	}

	if toTokenChain.ChainID != requestChainID {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeValidation, fmt.Sprintf("目标代币不在请求的区块链上: 代币链ID=%d, 请求链ID=%d", toTokenChain.ChainID, requestChainID), nil)
	}

	// 验证代币是否活跃
	if !fromToken.IsActive {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeValidation, "源代币已停用", nil)
	}

	if !toToken.IsActive {
		return nil, nil, nil, nil, NewServiceError(types.ErrCodeValidation, "目标代币已停用", nil)
	}

	return fromToken, toToken, fromTokenChain, toTokenChain, nil
}

// ========================================
// 临时实现的其他接口方法
// ========================================

func (s *quoteService) GetQuoteResponses(requestID string) ([]*types.AggregatorQuoteResponse, error) {
	// TODO: 实现获取所有聚合器响应
	return nil, nil
}

func (s *quoteService) InvalidateQuoteCache(fromTokenID, toTokenID uint) error {
	// TODO: 实现缓存失效
	return nil
}

func (s *quoteService) GetCacheStats() (map[string]interface{}, error) {
	// TODO: 实现缓存统计
	return nil, nil
}

func (s *quoteService) CompareQuotes(requestID string) (*types.QuoteComparison, error) {
	// TODO: 实现报价比较
	return &types.QuoteComparison{}, nil
}

func (s *quoteService) GetPriceImpactAnalysis(req *types.QuoteRequest) (*types.PriceImpactAnalysis, error) {
	// TODO: 实现价格冲击分析
	return &types.PriceImpactAnalysis{}, nil
}
