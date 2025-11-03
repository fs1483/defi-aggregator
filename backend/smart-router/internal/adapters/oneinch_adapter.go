// Package adapters 1inch聚合器适配器实现
// 实现1inch API的标准化接口，处理API格式转换和错误处理
// 支持1inch v5.0 API规范
package adapters

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"defi-aggregator/smart-router/internal/types"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// OneInchAdapter 1inch聚合器适配器
// 封装1inch API调用，提供标准化的报价接口
type OneInchAdapter struct {
	*BaseAdapter // 嵌入基础适配器
}

// NewOneInchAdapter 创建1inch适配器实例
func NewOneInchAdapter(config *types.ProviderConfig, logger *logrus.Logger) ProviderAdapter {
	return &OneInchAdapter{
		BaseAdapter: NewBaseAdapter(config, logger),
	}
}

// ========================================
// 1inch API响应结构定义
// ========================================

// OneInchQuoteResponse 1inch报价API响应
// 对应1inch /quote接口的响应格式
type OneInchQuoteResponse struct {
	FromToken struct {
		Symbol   string `json:"symbol"`
		Name     string `json:"name"`
		Decimals int    `json:"decimals"`
		Address  string `json:"address"`
	} `json:"fromToken"`

	ToToken struct {
		Symbol   string `json:"symbol"`
		Name     string `json:"name"`
		Decimals int    `json:"decimals"`
		Address  string `json:"address"`
	} `json:"toToken"`

	ToTokenAmount   string `json:"toTokenAmount"`   // 输出数量（字符串格式）
	FromTokenAmount string `json:"fromTokenAmount"` // 输入数量

	Protocols [][]struct {
		Name string `json:"name"`
		Part int    `json:"part"`
	} `json:"protocols"`

	EstimatedGas int64 `json:"estimatedGas"` // Gas估算
}

// OneInchErrorResponse 1inch错误响应
type OneInchErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Error      string `json:"error"`
	Message    string `json:"message"`
}

// ========================================
// 核心接口实现
// ========================================

// GetQuote 获取1inch报价
// 调用1inch /quote接口，获取最优交换报价
// 参数:
//   - ctx: 上下文，用于超时控制
//   - req: 标准化的报价请求
//
// 返回:
//   - *types.ProviderQuote: 标准化的报价响应
//   - error: 请求或解析错误
func (a *OneInchAdapter) GetQuote(ctx context.Context, req *types.QuoteRequest) (*types.ProviderQuote, error) {
	startTime := time.Now()

	// 检查链支持
	if !a.IsSupported(req.ChainID) {
		return nil, &types.RouterError{
			Code:    types.ErrCodeUnsupportedChain,
			Message: fmt.Sprintf("1inch不支持链ID: %d", req.ChainID),
		}
	}

	// 构建请求URL
	apiURL, err := a.buildQuoteURL(req)
	if err != nil {
		return nil, fmt.Errorf("构建请求URL失败: %w", err)
	}

	a.logger.Debugf("[1inch] 请求URL: %s", apiURL)

	// 发送HTTP请求
	responseBody, err := a.makeHTTPRequest(ctx, "GET", apiURL, nil, nil)
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.Provider1inch,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: err.Error(),
		}, nil // 返回失败的报价，不返回error，让聚合器继续处理其他提供商
	}

	// 解析响应
	var quoteResp OneInchQuoteResponse
	if err := a.parseJSONResponse(responseBody, &quoteResp); err != nil {
		// 尝试解析错误响应
		var errorResp OneInchErrorResponse
		if parseErr := a.parseJSONResponse(responseBody, &errorResp); parseErr == nil {
			return &types.ProviderQuote{
				Provider:     types.Provider1inch,
				Success:      false,
				ResponseTime: time.Since(startTime),
				ErrorCode:    strconv.Itoa(errorResp.StatusCode),
				ErrorMessage: errorResp.Message,
			}, nil
		}

		return &types.ProviderQuote{
			Provider:     types.Provider1inch,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: "响应解析失败",
		}, nil
	}

	// 转换为标准格式
	providerQuote, err := a.convertToStandardQuote(&quoteResp, time.Since(startTime))
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.Provider1inch,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: fmt.Sprintf("数据转换失败: %v", err),
		}, nil
	}

	a.logger.Debugf("[1inch] 报价获取成功: amountOut=%s, gas=%d, duration=%v",
		providerQuote.AmountOut.String(), providerQuote.GasEstimate, time.Since(startTime))

	return providerQuote, nil
}

// HealthCheck 1inch健康检查
// 检查1inch API的可用性和响应时间
func (a *OneInchAdapter) HealthCheck(ctx context.Context) error {
	// 构建健康检查URL（使用支持的链进行简单查询）
	if len(a.config.SupportedChains) == 0 {
		return fmt.Errorf("没有配置支持的链")
	}

	chainID := a.config.SupportedChains[0] // 使用第一个支持的链
	healthURL := fmt.Sprintf("%s/%d/healthcheck", a.config.BaseURL, chainID)

	// 发送健康检查请求
	_, err := a.makeHTTPRequest(ctx, "GET", healthURL, nil, nil)
	if err != nil {
		return fmt.Errorf("1inch健康检查失败: %w", err)
	}

	a.logger.Debugf("[1inch] 健康检查通过")
	return nil
}

// ========================================
// 辅助方法
// ========================================

// buildQuoteURL 构建1inch报价请求URL
// 根据请求参数构建符合1inch API规范的URL
func (a *OneInchAdapter) buildQuoteURL(req *types.QuoteRequest) (string, error) {
	// 1inch API URL格式: /{chainId}/quote
	baseURL := fmt.Sprintf("%s/%d/quote", a.config.BaseURL, req.ChainID)

	// 构建查询参数
	params := url.Values{}
	params.Set("fromTokenAddress", req.FromToken)
	params.Set("toTokenAddress", req.ToToken)
	params.Set("amount", req.AmountIn.String())

	// 添加可选参数
	if !req.Slippage.IsZero() {
		// 1inch接受百分比格式的滑点 (0.5 = 0.5%)
		slippagePercent := req.Slippage.Mul(decimal.NewFromInt(100))
		params.Set("slippage", slippagePercent.String())
	}

	if req.UserAddress != "" {
		params.Set("fromAddress", req.UserAddress)
	}

	// 组装完整URL
	fullURL := baseURL + "?" + params.Encode()
	return fullURL, nil
}

// convertToStandardQuote 将1inch响应转换为标准格式
// 统一不同聚合器的响应格式差异
func (a *OneInchAdapter) convertToStandardQuote(resp *OneInchQuoteResponse, responseTime time.Duration) (*types.ProviderQuote, error) {
	// 解析输出数量
	amountOut, err := a.standardizeAmount(resp.ToTokenAmount)
	if err != nil {
		return nil, fmt.Errorf("解析输出数量失败: %w", err)
	}

	// 解析输入数量（用于验证，但不在返回值中使用）
	_, err = a.standardizeAmount(resp.FromTokenAmount)
	if err != nil {
		return nil, fmt.Errorf("解析输入数量失败: %w", err)
	}

	// 转换交易路径
	var route []types.RouteStep
	for _, protocolGroup := range resp.Protocols {
		for _, protocol := range protocolGroup {
			route = append(route, types.RouteStep{
				Protocol:   protocol.Name,
				Percentage: decimal.NewFromInt(int64(protocol.Part)).Div(decimal.NewFromInt(100)),
			})
		}
	}

	// 计算置信度（基于响应时间和数据完整性）
	confidence := a.calculateConfidence(responseTime, resp.EstimatedGas > 0)

	// 计算价格冲击（简化计算，实际应该基于市场价格）
	priceImpact := decimal.NewFromFloat(0.001) // 默认0.1%

	return &types.ProviderQuote{
		Provider:     types.Provider1inch,
		Success:      true,
		AmountOut:    amountOut,
		GasEstimate:  uint64(resp.EstimatedGas),
		PriceImpact:  priceImpact,
		Route:        route,
		ResponseTime: responseTime,
		Confidence:   confidence,
		RawResponse:  resp, // 保存原始响应用于调试
	}, nil
}

// calculateConfidence 计算报价置信度
// 基于响应时间、数据完整性等因素计算置信度评分
func (a *OneInchAdapter) calculateConfidence(responseTime time.Duration, hasGasEstimate bool) decimal.Decimal {
	confidence := decimal.NewFromFloat(0.8) // 基础置信度

	// 响应时间因子 (越快置信度越高)
	if responseTime < 500*time.Millisecond {
		confidence = confidence.Add(decimal.NewFromFloat(0.1))
	} else if responseTime > 2*time.Second {
		confidence = confidence.Sub(decimal.NewFromFloat(0.1))
	}

	// 数据完整性因子
	if hasGasEstimate {
		confidence = confidence.Add(decimal.NewFromFloat(0.1))
	}

	// 确保置信度在0-1范围内
	if confidence.GreaterThan(decimal.NewFromFloat(1.0)) {
		confidence = decimal.NewFromFloat(1.0)
	}
	if confidence.LessThan(decimal.NewFromFloat(0.0)) {
		confidence = decimal.NewFromFloat(0.0)
	}

	return confidence
}
