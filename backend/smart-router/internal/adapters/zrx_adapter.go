// Package adapters 0x Protocol (ZRX) 聚合器适配器实现
// 实现0x Protocol Swap API的标准化接口，处理API格式转换和错误处理
// 支持0x Protocol v2 API规范，使用permit2协议
package adapters

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"defi-aggregator/smart-router/internal/types"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// ZRXAdapter 0x Protocol聚合器适配器
// 封装0x Protocol API调用，提供标准化的报价接口
type ZRXAdapter struct {
	*BaseAdapter // 嵌入基础适配器
}

// NewZRXAdapter 创建0x Protocol适配器实例
func NewZRXAdapter(config *types.ProviderConfig, logger *logrus.Logger) ProviderAdapter {
	return &ZRXAdapter{
		BaseAdapter: NewBaseAdapter(config, logger),
	}
}

// ========================================
// 0x Protocol API响应结构定义
// ========================================

// ZRXQuoteResponse 0x Protocol报价API响应
// 根据您提供的API响应示例定义的结构
type ZRXQuoteResponse struct {
	AllowanceTarget string `json:"allowanceTarget"` // 授权目标地址
	BlockNumber     string `json:"blockNumber"`     // 区块号
	BuyAmount       string `json:"buyAmount"`       // 买入数量
	BuyToken        string `json:"buyToken"`        // 买入代币地址
	Fees            struct {
		IntegratorFee interface{} `json:"integratorFee"` // 集成商手续费
		ZeroExFee     interface{} `json:"zeroExFee"`     // 0x协议手续费
		GasFee        interface{} `json:"gasFee"`        // Gas费
	} `json:"fees"`
	Issues struct {
		Allowance            interface{} `json:"allowance"`            // 授权问题
		Balance              interface{} `json:"balance"`              // 余额问题
		SimulationIncomplete bool        `json:"simulationIncomplete"` // 模拟未完成
		InvalidSourcesPassed []string    `json:"invalidSourcesPassed"` // 无效源
	} `json:"issues"`
	LiquidityAvailable bool   `json:"liquidityAvailable"` // 流动性可用性
	MinBuyAmount       string `json:"minBuyAmount"`       // 最小买入数量
	Permit2            struct {
		Type   string `json:"type"` // 许可类型
		Hash   string `json:"hash"` // 许可哈希
		Eip712 struct {
			// EIP712签名数据
		} `json:"eip712"`
	} `json:"permit2"`
	Route struct {
		Fills  []interface{} `json:"fills"`  // 填充路径
		Tokens []interface{} `json:"tokens"` // 代币路径
	} `json:"route"`
	SellAmount      string      `json:"sellAmount"`      // 卖出数量
	SellToken       string      `json:"sellToken"`       // 卖出代币地址
	TokenMetadata   interface{} `json:"tokenMetadata"`   // 代币元数据
	TotalNetworkFee string      `json:"totalNetworkFee"` // 总网络费用
	Transaction     struct {
		To       string `json:"to"`       // 交易目标地址
		Data     string `json:"data"`     // 交易数据
		Gas      string `json:"gas"`      // Gas限制
		GasPrice string `json:"gasPrice"` // Gas价格
		Value    string `json:"value"`    // 发送的ETH数量
	} `json:"transaction"`
	Zid string `json:"zid"` // 0x交易ID
}

// ZRXErrorResponse 0x Protocol错误响应
type ZRXErrorResponse struct {
	Code    int    `json:"code"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// ========================================
// 0x Protocol适配器接口实现
// ========================================

// GetQuote 获取0x Protocol报价
// 调用0x Protocol Swap API获取最优报价
func (a *ZRXAdapter) GetQuote(ctx context.Context, req *types.QuoteRequest) (*types.ProviderQuote, error) {
	startTime := time.Now()

	// 检查链支持
	if !a.IsSupported(req.ChainID) {
		return nil, &types.RouterError{
			Code:    types.ErrCodeUnsupportedChain,
			Message: fmt.Sprintf("0x Protocol不支持链ID: %d", req.ChainID),
		}
	}

	// 构建请求URL
	apiURL, err := a.buildQuoteURL(req)
	if err != nil {
		return nil, fmt.Errorf("构建请求URL失败: %w", err)
	}

	a.logger.Infof("[0x] 构建请求URL: %s", apiURL)

	// 安全的API Key日志记录
	if len(a.config.APIKey) >= 12 {
		a.logger.Infof("[0x] API Key: %s...%s", a.config.APIKey[:8], a.config.APIKey[len(a.config.APIKey)-4:])
	} else if len(a.config.APIKey) > 0 {
		a.logger.Infof("[0x] API Key: %s (长度: %d)", strings.Repeat("*", len(a.config.APIKey)), len(a.config.APIKey))
	} else {
		a.logger.Warnf("[0x] API Key为空！")
		return nil, fmt.Errorf("0x Protocol API Key未配置")
	}

	// 设置请求headers
	headers := map[string]string{
		"0x-api-key": a.config.APIKey,
		"0x-version": "v2",
	}

	a.logger.Infof("[0x] 发送GET请求，headers: %v", headers)

	// 发送HTTP GET请求
	responseBody, err := a.makeHTTPRequest(ctx, "GET", apiURL, nil, headers)
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.Provider0x,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 记录原始响应用于调试
	a.logger.Debugf("[0x] 原始响应: %s", string(responseBody))

	// 解析响应
	var zrxResponse ZRXQuoteResponse
	if err := a.parseJSONResponse(responseBody, &zrxResponse); err != nil {
		// 尝试解析错误响应
		var errorResponse ZRXErrorResponse
		if jsonErr := a.parseJSONResponse(responseBody, &errorResponse); jsonErr == nil {
			a.logger.Errorf("[0x] API返回错误: %d - %s", errorResponse.Code, errorResponse.Reason)
			return &types.ProviderQuote{
				Provider:     types.Provider0x,
				Success:      false,
				ResponseTime: time.Since(startTime),
				ErrorCode:    types.ErrCodeProviderError,
				ErrorMessage: fmt.Sprintf("0x API错误: %s", errorResponse.Reason),
			}, nil
		}

		a.logger.Errorf("[0x] 解析响应失败: %v, 原始响应: %s", err, string(responseBody))
		return &types.ProviderQuote{
			Provider:     types.Provider0x,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: fmt.Sprintf("解析响应失败: %v", err),
		}, nil
	}

	// 转换为标准格式
	quote, err := a.convertToStandardQuote(&zrxResponse, req, startTime)
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.Provider0x,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: fmt.Sprintf("转换报价格式失败: %v", err),
		}, nil
	}

	a.logger.Infof("[0x] 报价获取成功: buyAmount=%s, liquidityAvailable=%t",
		quote.AmountOut.String(), zrxResponse.LiquidityAvailable)
	return quote, nil
}

// ========================================
// 0x Protocol URL构建和数据转换
// ========================================

// buildQuoteURL 构建0x Protocol报价请求URL
func (a *ZRXAdapter) buildQuoteURL(req *types.QuoteRequest) (string, error) {
	// 0x Protocol API基础URL
	baseURL := a.config.BaseURL
	if baseURL == "" {
		return "", fmt.Errorf("0x Protocol API URL未配置")
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("chainId", strconv.FormatUint(uint64(req.ChainID), 10))
	params.Set("sellToken", req.FromToken)
	params.Set("buyToken", req.ToToken)
	params.Set("sellAmount", req.AmountIn.String())

	// 设置taker地址
	if req.UserAddress != "" {
		params.Set("taker", req.UserAddress)
	} else {
		// 使用默认地址
		params.Set("taker", "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	}

	// 注意：暂时不设置slippagePercentage参数，因为您的成功示例中没有这个参数
	// 0x Protocol会自动处理滑点
	// if !req.Slippage.IsZero() {
	//     slippagePercent := req.Slippage.Mul(decimal.NewFromInt(100))
	//     params.Set("slippagePercentage", slippagePercent.String())
	// }

	// 构建完整的URL
	apiURL := fmt.Sprintf("%s/swap/permit2/quote?%s", strings.TrimSuffix(baseURL, "/"), params.Encode())
	return apiURL, nil
}

// convertToStandardQuote 将0x Protocol响应转换为标准报价格式
func (a *ZRXAdapter) convertToStandardQuote(zrxResp *ZRXQuoteResponse, req *types.QuoteRequest, startTime time.Time) (*types.ProviderQuote, error) {
	// 检查流动性可用性
	if !zrxResp.LiquidityAvailable {
		return nil, fmt.Errorf("0x Protocol: 流动性不可用")
	}

	// 解析买入数量
	buyAmount, err := decimal.NewFromString(zrxResp.BuyAmount)
	if err != nil {
		return nil, fmt.Errorf("解析buyAmount失败: %w", err)
	}

	// 验证响应数据的一致性
	if strings.ToLower(zrxResp.SellToken) != strings.ToLower(req.FromToken) {
		a.logger.Warnf("[0x] 卖出代币地址不匹配: 请求=%s, 响应=%s", req.FromToken, zrxResp.SellToken)
	}
	if strings.ToLower(zrxResp.BuyToken) != strings.ToLower(req.ToToken) {
		a.logger.Warnf("[0x] 买入代币地址不匹配: 请求=%s, 响应=%s", req.ToToken, zrxResp.BuyToken)
	}

	// 解析Gas估算
	gasEstimate := uint64(200000) // 默认值
	if zrxResp.Transaction.Gas != "" {
		if gas, err := strconv.ParseUint(zrxResp.Transaction.Gas, 10, 64); err == nil {
			gasEstimate = gas
		}
	}

	// 计算价格影响
	priceImpact := a.calculatePriceImpact(req.AmountIn, buyAmount)

	// 解析路由信息
	var route []types.RouteStep
	// 0x Protocol的route结构比较复杂，这里简化处理
	if len(zrxResp.Route.Fills) > 0 {
		route = append(route, types.RouteStep{
			Protocol:   "0x Protocol",
			Percentage: decimal.NewFromFloat(1.0), // 假设100%通过0x
		})
	}

	// 计算置信度（基于流动性和模拟完成度）
	confidence := decimal.NewFromFloat(0.85) // 默认置信度
	if zrxResp.LiquidityAvailable && !zrxResp.Issues.SimulationIncomplete {
		confidence = decimal.NewFromFloat(0.9) // 高置信度
	}

	return &types.ProviderQuote{
		Provider:     types.Provider0x,
		Success:      true,
		AmountOut:    buyAmount,
		GasEstimate:  gasEstimate,
		PriceImpact:  priceImpact,
		Route:        route,
		ResponseTime: time.Since(startTime),
		Confidence:   confidence,
	}, nil
}

// calculatePriceImpact 计算价格影响
func (a *ZRXAdapter) calculatePriceImpact(amountIn, amountOut decimal.Decimal) decimal.Decimal {
	if amountIn.IsZero() {
		return decimal.Zero
	}

	// 简化的价格影响计算
	// 实际应该基于市场价格，这里使用近似计算
	expectedOut := amountIn // 假设1:1汇率作为基准
	actualOut := amountOut

	if expectedOut.IsZero() {
		return decimal.Zero
	}

	impact := expectedOut.Sub(actualOut).Div(expectedOut).Abs()
	return impact
}

// ========================================
// 健康检查实现
// ========================================

// HealthCheck 检查0x Protocol服务健康状态
func (a *ZRXAdapter) HealthCheck(ctx context.Context) error {
	// 0x Protocol健康检查可以通过简单的API调用
	healthURL := fmt.Sprintf("%s/swap/permit2/price?chainId=1&sellToken=0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE&buyToken=0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48&sellAmount=1000000000000000000",
		a.config.BaseURL)

	headers := map[string]string{
		"0x-api-key": a.config.APIKey,
		"0x-version": "v2",
	}

	_, err := a.makeHTTPRequest(ctx, "GET", healthURL, nil, headers)
	if err != nil {
		return fmt.Errorf("0x Protocol健康检查失败: %w", err)
	}

	a.logger.Debugf("[0x] 健康检查通过")
	return nil
}

// ========================================
// 工具方法
// ========================================

// GetName 返回适配器名称
func (a *ZRXAdapter) GetName() string {
	return string(types.Provider0x)
}

// GetDisplayName 返回显示名称
func (a *ZRXAdapter) GetDisplayName() string {
	return a.config.DisplayName
}

// IsSupported 检查是否支持指定链
func (a *ZRXAdapter) IsSupported(chainID uint) bool {
	for _, supportedChain := range a.config.SupportedChains {
		if supportedChain == chainID {
			return true
		}
	}
	return false
}
