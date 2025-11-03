// Package services 智能路由核心服务实现
// 实现并发聚合算法、最优路径选择、渐进式响应策略
// 这是整个DeFi聚合器的核心大脑，负责智能决策和性能优化
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"defi-aggregator/smart-router/internal/adapters"
	"defi-aggregator/smart-router/internal/types"
	"defi-aggregator/smart-router/pkg/cache"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// ProviderAdapter 聚合器适配器接口（在services包中定义避免循环导入）
type ProviderAdapter interface {
	GetName() string
	GetDisplayName() string
	IsSupported(chainID uint) bool
	GetQuote(ctx context.Context, req *types.QuoteRequest) (*types.ProviderQuote, error)
	HealthCheck(ctx context.Context) error
	UpdateConfig(config *types.ProviderConfig) error
	GetConfig() *types.ProviderConfig
}

// RouterService 智能路由服务
// 核心聚合服务，协调多个聚合器适配器，实现智能报价聚合
type RouterService struct {
	adapters map[string]ProviderAdapter // 聚合器适配器集合
	cache    cache.CacheManager         // 缓存管理器
	config   *types.Config              // 服务配置
	logger   *logrus.Logger             // 日志记录器
	metrics  *RouterMetrics             // 服务指标
}

// RouterMetrics 路由服务指标
type RouterMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	CacheHits          int64         `json:"cache_hits"`
	CacheMisses        int64         `json:"cache_misses"`
	AvgAggregationTime time.Duration `json:"avg_aggregation_time"`
	LastRequestTime    time.Time     `json:"last_request_time"`
	mutex              sync.RWMutex  // 指标读写锁
}

// NewRouterService 创建智能路由服务实例
// 初始化所有聚合器适配器和缓存管理器
func NewRouterService(config *types.Config, cacheManager cache.CacheManager, logger *logrus.Logger) *RouterService {
	service := &RouterService{
		adapters: make(map[string]ProviderAdapter),
		cache:    cacheManager,
		config:   config,
		logger:   logger,
		metrics:  &RouterMetrics{},
	}

	// 初始化聚合器适配器
	service.initializeAdapters()

	return service
}

// ========================================
// 核心聚合算法实现
// ========================================

// GetOptimalQuote 获取最优报价
// 智能路由的核心方法，实现并发聚合和渐进式响应策略
// 参数:
//   - ctx: 上下文，用于超时控制
//   - req: 报价请求
//
// 返回:
//   - *types.QuoteResponse: 聚合后的最优报价
//   - error: 聚合过程中的错误
func (s *RouterService) GetOptimalQuote(ctx context.Context, req *types.QuoteRequest) (*types.QuoteResponse, error) {
	startTime := time.Now()
	sessionID := req.RequestID

	s.logger.Infof("[%s] 🚀 聚合请求: %s->%s, 金额=%s, 链=%d",
		sessionID, req.FromToken, req.ToToken, req.AmountIn.String(), req.ChainID)

	// 1. 检查缓存
	if cachedQuote := s.checkCache(req); cachedQuote != nil {
		s.updateMetrics(true, time.Since(startTime), true)
		s.logger.Infof("[%s] 缓存命中，直接返回结果", sessionID)
		return cachedQuote, nil
	}

	// 2. 获取支持该链的活跃聚合器
	activeAdapters := s.getActiveAdapters(req.ChainID)
	if len(activeAdapters) == 0 {
		return nil, &types.RouterError{
			Code:    types.ErrCodeUnsupportedChain,
			Message: fmt.Sprintf("没有聚合器支持链ID: %d", req.ChainID),
		}
	}

	s.logger.Infof("[%s] 🔍 找到 %d 个支持的聚合器", sessionID, len(activeAdapters))

	// 3. 执行并发聚合
	quotes := s.executeParallelAggregation(ctx, req, activeAdapters)

	// 4. 选择最优报价
	bestQuote, allQuotes := s.selectBestQuote(quotes, req)
	if bestQuote == nil {
		return nil, &types.RouterError{
			Code:    types.ErrCodeNoValidQuotes,
			Message: "所有聚合器都返回失败",
		}
	}

	// 5. 构建聚合响应
	response := s.buildAggregationResponse(req, bestQuote, allQuotes, startTime)

	// 6. 缓存结果
	s.cacheResult(req, response)

	// 7. 更新指标
	s.updateMetrics(true, time.Since(startTime), false)

	s.logger.Infof("[%s] 🎉 智能路由聚合完成: 最优聚合器=%s, amountOut=%s, gasEstimate=%d, priceImpact=%s, 总耗时=%v",
		sessionID, bestQuote.Provider, bestQuote.AmountOut.String(), bestQuote.GasEstimate,
		bestQuote.PriceImpact.String(), time.Since(startTime))

	return response, nil
}

// ========================================
// 并发聚合实现
// ========================================

// executeParallelAggregation 执行并发聚合
// 同时调用多个聚合器API，收集所有报价结果
func (s *RouterService) executeParallelAggregation(ctx context.Context, req *types.QuoteRequest, adapters []ProviderAdapter) []*types.ProviderQuote {
	quoteChan := make(chan *types.ProviderQuote, len(adapters))
	var wg sync.WaitGroup

	s.logger.Infof("[%s] 🚀 并发调用 %d 个聚合器", req.RequestID, len(adapters))

	// 为每个聚合器启动独立的goroutine
	for i, adapter := range adapters {
		wg.Add(1)
		go func(index int, adp ProviderAdapter) {
			defer wg.Done()

			adapterStartTime := time.Now()
			s.logger.Infof("[%s] 📞 调用: %s", req.RequestID, adp.GetName())

			// 创建带超时的上下文
			adapterCtx, cancel := context.WithTimeout(ctx, adp.GetConfig().Timeout)
			defer cancel()

			// 调用聚合器获取报价
			quote, err := adp.GetQuote(adapterCtx, req)
			if err != nil {
				s.logger.Errorf("[%s] 💥 聚合器 %s 调用异常: %v, 耗时=%v",
					req.RequestID, adp.GetName(), err, time.Since(adapterStartTime))

				// 即使出错也要发送结果到channel
				quote = &types.ProviderQuote{
					Provider:     adp.GetName(),
					Success:      false,
					ResponseTime: time.Since(adapterStartTime),
					ErrorCode:    types.ErrCodeProviderError,
					ErrorMessage: err.Error(),
				}
			} else {
				s.logger.Infof("[%s] 🎯 聚合器 %s 响应完成: success=%t, 耗时=%v",
					req.RequestID, adp.GetName(), quote.Success, time.Since(adapterStartTime))
			}

			// 发送结果到channel
			select {
			case quoteChan <- quote:
			case <-adapterCtx.Done():
				s.logger.Warnf("[%s] ⏰ 聚合器 %s 上下文已取消", req.RequestID, adp.GetName())
			}
		}(i, adapter)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(quoteChan)
	}()

	// 收集所有报价结果
	var quotes []*types.ProviderQuote
	for quote := range quoteChan {
		quotes = append(quotes, quote)

		if quote.Success {
			s.logger.Infof("[%s] ✅ %s: 报价=%s, Gas=%d, 耗时=%v",
				req.RequestID, quote.Provider, quote.AmountOut.String(), quote.GasEstimate, quote.ResponseTime)
		} else {
			s.logger.Warnf("[%s] ❌ %s: %s",
				req.RequestID, quote.Provider, quote.ErrorMessage)
		}
	}

	return quotes
}

// ========================================
// 最优选择算法
// ========================================

// selectBestQuote 选择最优报价
// 基于价格、Gas费用、置信度等因素选择最佳报价
func (s *RouterService) selectBestQuote(quotes []*types.ProviderQuote, req *types.QuoteRequest) (*types.ProviderQuote, []*types.ProviderQuote) {
	if len(quotes) == 0 {
		return nil, quotes
	}

	// 筛选成功的报价
	var validQuotes []*types.ProviderQuote
	for _, quote := range quotes {
		if quote.Success && !quote.AmountOut.IsZero() {
			validQuotes = append(validQuotes, quote)
		}
	}

	if len(validQuotes) == 0 {
		return nil, quotes
	}

	// 计算每个报价的综合评分
	var bestQuote *types.ProviderQuote
	var bestScore decimal.Decimal

	for i, quote := range validQuotes {
		score := s.calculateQuoteScore(quote, validQuotes)

		if i == 0 || score.GreaterThan(bestScore) {
			bestQuote = quote
			bestScore = score
		}

		// 设置排名
		quote.Rank = i + 1

		s.logger.Infof("[%s] 📊 聚合器 %s 评分: %.4f, amountOut=%s, gas=%d",
			req.RequestID, quote.Provider, score.InexactFloat64(), quote.AmountOut.String(), quote.GasEstimate)
	}

	s.logger.Infof("最优聚合器: %s, 评分: %.4f", bestQuote.Provider, bestScore.InexactFloat64())
	return bestQuote, quotes
}

// calculateQuoteScore 计算报价综合评分
// 基于多个维度计算报价的综合评分
func (s *RouterService) calculateQuoteScore(quote *types.ProviderQuote, allQuotes []*types.ProviderQuote) decimal.Decimal {
	// 价格评分 (权重50%)：输出数量越多评分越高
	priceScore := s.calculatePriceScore(quote, allQuotes)

	// Gas效率评分 (权重20%)：Gas费用越低评分越高
	gasScore := s.calculateGasScore(quote, allQuotes)

	// 置信度评分 (权重20%)：直接使用置信度
	confidenceScore := quote.Confidence

	// 响应时间评分 (权重10%)：响应越快评分越高
	timeScore := s.calculateTimeScore(quote)

	// 计算加权综合评分
	totalScore := priceScore.Mul(decimal.NewFromFloat(0.5)).
		Add(gasScore.Mul(decimal.NewFromFloat(0.2))).
		Add(confidenceScore.Mul(decimal.NewFromFloat(0.2))).
		Add(timeScore.Mul(decimal.NewFromFloat(0.1)))

	return totalScore
}

// calculatePriceScore 计算价格评分
func (s *RouterService) calculatePriceScore(quote *types.ProviderQuote, allQuotes []*types.ProviderQuote) decimal.Decimal {
	if len(allQuotes) == 1 {
		return decimal.NewFromFloat(1.0)
	}

	// 找到最高和最低输出金额
	var maxAmount, minAmount decimal.Decimal
	for i, q := range allQuotes {
		if !q.Success {
			continue
		}
		if i == 0 || q.AmountOut.GreaterThan(maxAmount) {
			maxAmount = q.AmountOut
		}
		if i == 0 || q.AmountOut.LessThan(minAmount) {
			minAmount = q.AmountOut
		}
	}

	// 计算相对评分
	if maxAmount.Equal(minAmount) {
		return decimal.NewFromFloat(1.0)
	}

	score := quote.AmountOut.Sub(minAmount).Div(maxAmount.Sub(minAmount))
	return score
}

// calculateGasScore 计算Gas效率评分
func (s *RouterService) calculateGasScore(quote *types.ProviderQuote, allQuotes []*types.ProviderQuote) decimal.Decimal {
	if len(allQuotes) == 1 {
		return decimal.NewFromFloat(1.0)
	}

	// 找到最高和最低Gas估算
	var maxGas, minGas uint64
	for i, q := range allQuotes {
		if !q.Success || q.GasEstimate == 0 {
			continue
		}
		if i == 0 || q.GasEstimate > maxGas {
			maxGas = q.GasEstimate
		}
		if i == 0 || q.GasEstimate < minGas {
			minGas = q.GasEstimate
		}
	}

	if maxGas == minGas || quote.GasEstimate == 0 {
		return decimal.NewFromFloat(0.5) // 中等评分
	}

	// Gas越低评分越高
	gasRange := decimal.NewFromInt(int64(maxGas - minGas))
	gasOffset := decimal.NewFromInt(int64(maxGas - quote.GasEstimate))
	score := gasOffset.Div(gasRange)

	return score
}

// calculateTimeScore 计算响应时间评分
func (s *RouterService) calculateTimeScore(quote *types.ProviderQuote) decimal.Decimal {
	// 基于响应时间计算评分，越快评分越高
	responseMs := float64(quote.ResponseTime.Milliseconds())

	if responseMs <= 200 {
		return decimal.NewFromFloat(1.0) // 极快
	} else if responseMs <= 500 {
		return decimal.NewFromFloat(0.8) // 快
	} else if responseMs <= 1000 {
		return decimal.NewFromFloat(0.6) // 中等
	} else if responseMs <= 2000 {
		return decimal.NewFromFloat(0.4) // 慢
	} else {
		return decimal.NewFromFloat(0.2) // 很慢
	}
}

// ========================================
// 缓存管理
// ========================================

// checkCache 检查缓存
// 根据请求参数检查是否有有效的缓存结果
func (s *RouterService) checkCache(req *types.QuoteRequest) *types.QuoteResponse {
	cacheKey := s.generateCacheKey(req)

	cachedData, err := s.cache.Get(cacheKey)
	if err != nil {
		s.logger.Debugf("缓存查询失败: %v", err)
		return nil
	}

	if cachedData == nil {
		return nil
	}

	// 尝试转换为QuoteResponse
	if cachedQuote, ok := cachedData.(*types.QuoteResponse); ok {
		// 检查缓存是否过期
		if time.Now().Before(cachedQuote.ValidUntil) {
			cachedQuote.CacheHit = true
			return cachedQuote
		}
	}

	return nil
}

// cacheResult 缓存聚合结果
func (s *RouterService) cacheResult(req *types.QuoteRequest, response *types.QuoteResponse) {
	cacheKey := s.generateCacheKey(req)

	// 设置缓存TTL
	ttl := s.config.Cache.DefaultTTL

	if err := s.cache.Set(cacheKey, response, ttl); err != nil {
		s.logger.Warnf("缓存结果失败: %v", err)
	} else {
		s.logger.Debugf("缓存结果成功: key=%s, ttl=%v", cacheKey, ttl)
	}
}

// generateCacheKey 生成缓存键
func (s *RouterService) generateCacheKey(req *types.QuoteRequest) string {
	return fmt.Sprintf("%s%s_%s_%s_%d_%s",
		s.config.Cache.PrefixKey,
		req.FromToken,
		req.ToToken,
		req.AmountIn.String(),
		req.ChainID,
		req.Slippage.String(),
	)
}

// ========================================
// 辅助方法
// ========================================

// initializeAdapters 初始化聚合器适配器
// 优雅的适配器初始化：确保配置正确传递，避免映射错误
func (s *RouterService) initializeAdapters() {
	s.logger.Infof("🚀 开始初始化聚合器适配器系统...")
	s.logger.Infof("📊 总配置数量: %d", len(s.config.Providers))

	// 清空现有适配器
	s.adapters = make(map[string]ProviderAdapter)

	activeCount := 0

	// 逐个初始化聚合器适配器
	for i, providerConfig := range s.config.Providers {
		s.logger.Infof("📦 处理聚合器 %d/%d: %s", i+1, len(s.config.Providers), providerConfig.Name)

		// 检查启用状态（数据库is_active控制）
		if !providerConfig.IsActive {
			s.logger.Infof("⏭️ 跳过未启用的聚合器: %s (is_active=false)", providerConfig.DisplayName)
			continue
		}

		// 创建独立的配置副本，避免引用污染
		config := types.ProviderConfig{
			Name:            providerConfig.Name,
			DisplayName:     providerConfig.DisplayName,
			BaseURL:         providerConfig.BaseURL,
			APIKey:          providerConfig.APIKey,
			Timeout:         providerConfig.Timeout,
			RetryCount:      providerConfig.RetryCount,
			Priority:        providerConfig.Priority,
			Weight:          providerConfig.Weight,
			IsActive:        providerConfig.IsActive,
			SupportedChains: append([]uint{}, providerConfig.SupportedChains...), // 深拷贝
		}

		s.logger.Infof("🔧 聚合器配置详情: name=%s, display=%s, url=%s, apiKey=%s, chains=%v",
			config.Name, config.DisplayName, config.BaseURL,
			func() string {
				if config.APIKey != "" {
					return fmt.Sprintf("已配置(%d字符)", len(config.APIKey))
				}
				return "未配置"
			}(),
			config.SupportedChains)

		// 根据聚合器名称创建对应的适配器
		adapter, err := s.createAdapter(config)
		if err != nil {
			s.logger.Errorf("❌ 创建适配器失败: %s - %v", config.Name, err)
			continue
		}

		// 验证适配器配置
		if err := s.validateAdapter(adapter, config); err != nil {
			s.logger.Errorf("❌ 适配器验证失败: %s - %v", config.Name, err)
			continue
		}

		// 注册适配器
		s.adapters[config.Name] = adapter
		activeCount++

		s.logger.Infof("✅ 适配器注册成功: %s -> 实际名称:%s, 显示名称:%s",
			config.Name, adapter.GetName(), adapter.GetDisplayName())
	}

	s.logger.Infof("🎉 聚合器适配器初始化完成: %d/%d 个适配器活跃", activeCount, len(s.config.Providers))

	// 输出最终的适配器映射
	for name, adapter := range s.adapters {
		s.logger.Infof("📋 最终映射: %s -> %s (%s)", name, adapter.GetName(), adapter.GetDisplayName())
	}
}

// createAdapter 创建聚合器适配器
func (s *RouterService) createAdapter(config types.ProviderConfig) (ProviderAdapter, error) {
	switch config.Name {
	case "cowswap":
		return adapters.NewCowAdapter(&config, s.logger), nil
	case "1inch":
		return adapters.NewOneInchAdapter(&config, s.logger), nil
	case "paraswap":
		return adapters.NewParaSwapAdapter(&config, s.logger), nil
	case "0x":
		return adapters.NewZRXAdapter(&config, s.logger), nil
	default:
		// 创建模拟适配器
		return &MockAdapter{
			name:   config.Name,
			config: &config,
			logger: s.logger,
		}, nil
	}
}

// validateAdapter 验证适配器配置
func (s *RouterService) validateAdapter(adapter ProviderAdapter, expectedConfig types.ProviderConfig) error {
	// 验证适配器名称
	if adapter.GetName() != expectedConfig.Name {
		return fmt.Errorf("适配器名称不匹配: 期望=%s, 实际=%s", expectedConfig.Name, adapter.GetName())
	}

	// 验证配置URL
	actualConfig := adapter.GetConfig()
	if actualConfig.BaseURL != expectedConfig.BaseURL {
		return fmt.Errorf("适配器URL不匹配: 期望=%s, 实际=%s", expectedConfig.BaseURL, actualConfig.BaseURL)
	}

	return nil
}

// MockAdapter 模拟适配器（临时实现）
type MockAdapter struct {
	name   string
	config *types.ProviderConfig
	logger *logrus.Logger
}

func (m *MockAdapter) GetName() string        { return m.name }
func (m *MockAdapter) GetDisplayName() string { return m.config.DisplayName }
func (m *MockAdapter) IsSupported(chainID uint) bool {
	for _, supported := range m.config.SupportedChains {
		if supported == chainID {
			return true
		}
	}
	return false
}
func (m *MockAdapter) GetQuote(ctx context.Context, req *types.QuoteRequest) (*types.ProviderQuote, error) {
	// 模拟报价响应
	return &types.ProviderQuote{
		Provider:     m.name,
		Success:      true,
		AmountOut:    req.AmountIn.Mul(decimal.NewFromFloat(0.99)), // 模拟1%的价格冲击
		GasEstimate:  180000,
		PriceImpact:  decimal.NewFromFloat(0.01),
		Route:        []types.RouteStep{},
		ResponseTime: 200 * time.Millisecond,
		Confidence:   decimal.NewFromFloat(0.8),
	}, nil
}
func (m *MockAdapter) HealthCheck(ctx context.Context) error { return nil }
func (m *MockAdapter) UpdateConfig(config *types.ProviderConfig) error {
	m.config = config
	return nil
}
func (m *MockAdapter) GetConfig() *types.ProviderConfig { return m.config }

// getActiveAdapters 获取支持指定链的活跃适配器
func (s *RouterService) getActiveAdapters(chainID uint) []ProviderAdapter {
	var activeAdapters []ProviderAdapter

	for _, adapter := range s.adapters {
		if adapter.IsSupported(chainID) {
			activeAdapters = append(activeAdapters, adapter)
		}
	}

	return activeAdapters
}

// buildAggregationResponse 构建聚合响应
func (s *RouterService) buildAggregationResponse(
	req *types.QuoteRequest,
	bestQuote *types.ProviderQuote,
	allQuotes []*types.ProviderQuote,
	startTime time.Time,
) *types.QuoteResponse {
	// 计算性能指标
	performance := s.calculatePerformance(allQuotes, startTime)

	// 计算汇率
	exchangeRate := decimal.Zero
	if !req.AmountIn.IsZero() {
		exchangeRate = bestQuote.AmountOut.Div(req.AmountIn)
	}

	return &types.QuoteResponse{
		RequestID:       req.RequestID,
		Success:         true,
		BestProvider:    bestQuote.Provider,
		BestPrice:       bestQuote.AmountOut,
		BestGasEstimate: bestQuote.GasEstimate,
		PriceImpact:     bestQuote.PriceImpact,
		ExchangeRate:    exchangeRate,
		Route:           bestQuote.Route,
		AllQuotes:       allQuotes,
		Performance:     performance,
		ValidUntil:      time.Now().Add(s.config.Cache.DefaultTTL),
		CacheHit:        false,
		Timestamp:       time.Now(),
	}
}

// calculatePerformance 计算聚合性能指标
func (s *RouterService) calculatePerformance(quotes []*types.ProviderQuote, startTime time.Time) types.AggregationPerformance {
	totalDuration := time.Since(startTime)
	successCount := 0
	var totalResponseTime time.Duration
	var fastestProvider, slowestProvider string
	var minTime, maxTime time.Duration

	for i, quote := range quotes {
		if quote.Success {
			successCount++
		}

		totalResponseTime += quote.ResponseTime

		if i == 0 || quote.ResponseTime < minTime {
			minTime = quote.ResponseTime
			fastestProvider = quote.Provider
		}

		if i == 0 || quote.ResponseTime > maxTime {
			maxTime = quote.ResponseTime
			slowestProvider = quote.Provider
		}
	}

	avgResponseTime := time.Duration(0)
	if len(quotes) > 0 {
		avgResponseTime = totalResponseTime / time.Duration(len(quotes))
	}

	// 计算质量评分
	qualityScore := decimal.NewFromFloat(float64(successCount) / float64(len(quotes)))

	return types.AggregationPerformance{
		TotalDuration:    totalDuration,
		ProvidersQueried: len(quotes),
		ProvidersSuccess: successCount,
		FastestProvider:  fastestProvider,
		SlowestProvider:  slowestProvider,
		AvgResponseTime:  avgResponseTime,
		CacheHitRate:     decimal.Zero, // 由缓存管理器计算
		StrategyUsed:     types.StrategyProgressive,
		QualityScore:     qualityScore,
	}
}

// ========================================
// 指标管理
// ========================================

// updateMetrics 更新服务指标
func (s *RouterService) updateMetrics(success bool, duration time.Duration, cacheHit bool) {
	s.metrics.mutex.Lock()
	defer s.metrics.mutex.Unlock()

	s.metrics.TotalRequests++
	s.metrics.LastRequestTime = time.Now()

	if cacheHit {
		s.metrics.CacheHits++
	} else {
		s.metrics.CacheMisses++
	}

	// 更新平均聚合时间
	if s.metrics.TotalRequests == 1 {
		s.metrics.AvgAggregationTime = duration
	} else {
		alpha := 0.1
		s.metrics.AvgAggregationTime = time.Duration(
			float64(s.metrics.AvgAggregationTime)*(1-alpha) + float64(duration)*alpha,
		)
	}
}

// GetMetrics 获取服务指标
func (s *RouterService) GetMetrics() *RouterMetrics {
	s.metrics.mutex.RLock()
	defer s.metrics.mutex.RUnlock()

	// 返回指标副本
	return &RouterMetrics{
		TotalRequests:      s.metrics.TotalRequests,
		CacheHits:          s.metrics.CacheHits,
		CacheMisses:        s.metrics.CacheMisses,
		AvgAggregationTime: s.metrics.AvgAggregationTime,
		LastRequestTime:    s.metrics.LastRequestTime,
	}
}
