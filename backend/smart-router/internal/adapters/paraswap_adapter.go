// Package adapters ParaSwap聚合器适配器实现
// 实现ParaSwap API的标准化接口，处理API格式转换
// 支持ParaSwap v5 API规范
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

// ParaSwapAdapter ParaSwap聚合器适配器
type ParaSwapAdapter struct {
	*BaseAdapter
}

// NewParaSwapAdapter 创建ParaSwap适配器实例
func NewParaSwapAdapter(config *types.ProviderConfig, logger *logrus.Logger) ProviderAdapter {
	return &ParaSwapAdapter{
		BaseAdapter: NewBaseAdapter(config, logger),
	}
}

// ========================================
// ParaSwap API响应结构定义
// ========================================

// ParaSwapPriceResponse ParaSwap价格API响应
type ParaSwapPriceResponse struct {
	PriceRoute struct {
		SrcToken struct {
			Symbol   string `json:"symbol"`
			Decimals int    `json:"decimals"`
			Address  string `json:"address"`
		} `json:"srcToken"`

		DestToken struct {
			Symbol   string `json:"symbol"`
			Decimals int    `json:"decimals"`
			Address  string `json:"address"`
		} `json:"destToken"`

		SrcAmount          string      `json:"srcAmount"`          // 输入数量
		DestAmount         string      `json:"destAmount"`         // 输出数量
		BestRoute          []RouteData `json:"bestRoute"`          // 最佳路径
		GasCostUSD         string      `json:"gasCostUSD"`         // Gas费用USD
		GasCost            string      `json:"gasCost"`            // Gas费用
		Side               string      `json:"side"`               // 交易方向
		TokenTransferProxy string      `json:"tokenTransferProxy"` // 代币转移代理
	} `json:"priceRoute"`
}

// RouteData ParaSwap路径数据
type RouteData struct {
	Exchange   string `json:"exchange"`   // 交易所名称
	Percent    int    `json:"percent"`    // 百分比
	SrcAmount  string `json:"srcAmount"`  // 源数量
	DestAmount string `json:"destAmount"` // 目标数量
}

// ParaSwapErrorResponse ParaSwap错误响应
type ParaSwapErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// ========================================
// 接口实现
// ========================================

// GetQuote 获取ParaSwap报价
// 调用ParaSwap价格API获取最优报价
func (a *ParaSwapAdapter) GetQuote(ctx context.Context, req *types.QuoteRequest) (*types.ProviderQuote, error) {
	startTime := time.Now()

	// 检查链支持
	if !a.IsSupported(req.ChainID) {
		return &types.ProviderQuote{
			Provider:     types.ProviderParaswap,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeUnsupportedChain,
			ErrorMessage: fmt.Sprintf("ParaSwap不支持链ID: %d", req.ChainID),
		}, nil
	}

	// 构建请求URL
	apiURL, err := a.buildPriceURL(req)
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.ProviderParaswap,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeInvalidRequest,
			ErrorMessage: fmt.Sprintf("构建请求URL失败: %v", err),
		}, nil
	}

	a.logger.Debugf("[ParaSwap] 请求URL: %s", apiURL)

	// 发送HTTP请求
	responseBody, err := a.makeHTTPRequest(ctx, "GET", apiURL, nil, nil)
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.ProviderParaswap,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 解析响应
	var priceResp ParaSwapPriceResponse
	if err := a.parseJSONResponse(responseBody, &priceResp); err != nil {
		// 尝试解析错误响应
		var errorResp ParaSwapErrorResponse
		if parseErr := a.parseJSONResponse(responseBody, &errorResp); parseErr == nil {
			return &types.ProviderQuote{
				Provider:     types.ProviderParaswap,
				Success:      false,
				ResponseTime: time.Since(startTime),
				ErrorCode:    strconv.Itoa(errorResp.Error.Code),
				ErrorMessage: errorResp.Error.Message,
			}, nil
		}

		return &types.ProviderQuote{
			Provider:     types.ProviderParaswap,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: "响应解析失败",
		}, nil
	}

	// 转换为标准格式
	providerQuote, err := a.convertToStandardQuote(&priceResp, time.Since(startTime))
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.ProviderParaswap,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: fmt.Sprintf("数据转换失败: %v", err),
		}, nil
	}

	a.logger.Debugf("[ParaSwap] 报价获取成功: amountOut=%s, duration=%v",
		providerQuote.AmountOut.String(), time.Since(startTime))

	return providerQuote, nil
}

// HealthCheck ParaSwap健康检查
func (a *ParaSwapAdapter) HealthCheck(ctx context.Context) error {
	// ParaSwap健康检查：调用支持的代币列表接口
	if len(a.config.SupportedChains) == 0 {
		return fmt.Errorf("没有配置支持的链")
	}

	chainID := a.config.SupportedChains[0]
	healthURL := fmt.Sprintf("%s/tokens/%d", a.config.BaseURL, chainID)

	_, err := a.makeHTTPRequest(ctx, "GET", healthURL, nil, nil)
	if err != nil {
		return fmt.Errorf("ParaSwap健康检查失败: %w", err)
	}

	a.logger.Debugf("[ParaSwap] 健康检查通过")
	return nil
}

// ========================================
// 辅助方法
// ========================================

// buildPriceURL 构建ParaSwap价格请求URL
func (a *ParaSwapAdapter) buildPriceURL(req *types.QuoteRequest) (string, error) {
	// ParaSwap API URL格式: /prices
	baseURL := fmt.Sprintf("%s/prices", a.config.BaseURL)

	// 构建查询参数
	params := url.Values{}
	params.Set("srcToken", req.FromToken)
	params.Set("destToken", req.ToToken)
	params.Set("amount", req.AmountIn.String())
	params.Set("network", strconv.Itoa(int(req.ChainID)))
	params.Set("side", "SELL") // 固定为SELL模式

	// 添加可选参数
	if req.UserAddress != "" {
		params.Set("userAddress", req.UserAddress)
	}

	// 组装完整URL
	fullURL := baseURL + "?" + params.Encode()
	return fullURL, nil
}

// convertToStandardQuote 将ParaSwap响应转换为标准格式
func (a *ParaSwapAdapter) convertToStandardQuote(resp *ParaSwapPriceResponse, responseTime time.Duration) (*types.ProviderQuote, error) {
	// 解析输出数量
	amountOut, err := a.standardizeAmount(resp.PriceRoute.DestAmount)
	if err != nil {
		return nil, fmt.Errorf("解析输出数量失败: %w", err)
	}

	// 转换交易路径
	var route []types.RouteStep
	for _, routeData := range resp.PriceRoute.BestRoute {
		route = append(route, types.RouteStep{
			Protocol:   routeData.Exchange,
			Percentage: decimal.NewFromInt(int64(routeData.Percent)).Div(decimal.NewFromInt(100)),
		})
	}

	// 计算Gas估算（ParaSwap返回的是Gas费用，需要转换为Gas数量）
	var gasEstimate uint64 = 180000 // 默认Gas估算
	if resp.PriceRoute.GasCost != "" {
		if gasCost, err := decimal.NewFromString(resp.PriceRoute.GasCost); err == nil {
			// 假设Gas价格为20 Gwei，计算Gas数量
			gasPrice := decimal.NewFromInt(20000000000) // 20 Gwei in wei
			if !gasPrice.IsZero() {
				gasAmount := gasCost.Div(gasPrice)
				gasEstimate = uint64(gasAmount.IntPart())
			}
		}
	}

	// 计算置信度
	confidence := a.calculateConfidence(responseTime, gasEstimate > 0)

	// 计算价格冲击（简化）
	priceImpact := decimal.NewFromFloat(0.002) // 默认0.2%

	return &types.ProviderQuote{
		Provider:     types.ProviderParaswap,
		Success:      true,
		AmountOut:    amountOut,
		GasEstimate:  gasEstimate,
		PriceImpact:  priceImpact,
		Route:        route,
		ResponseTime: responseTime,
		Confidence:   confidence,
		RawResponse:  resp,
	}, nil
}

// calculateConfidence 计算ParaSwap报价置信度
func (a *ParaSwapAdapter) calculateConfidence(responseTime time.Duration, hasGasData bool) decimal.Decimal {
	confidence := decimal.NewFromFloat(0.85) // ParaSwap基础置信度

	// 响应时间因子
	if responseTime < 600*time.Millisecond {
		confidence = confidence.Add(decimal.NewFromFloat(0.05))
	} else if responseTime > 2*time.Second {
		confidence = confidence.Sub(decimal.NewFromFloat(0.1))
	}

	// 数据完整性因子
	if hasGasData {
		confidence = confidence.Add(decimal.NewFromFloat(0.05))
	}

	// 确保置信度在合理范围内
	if confidence.GreaterThan(decimal.NewFromFloat(1.0)) {
		confidence = decimal.NewFromFloat(1.0)
	}
	if confidence.LessThan(decimal.NewFromFloat(0.0)) {
		confidence = decimal.NewFromFloat(0.0)
	}

	return confidence
}
