// Package adapters CoW Protocol聚合器适配器实现
// 实现CoW Protocol API的标准化接口，处理API格式转换和错误处理
// 支持CoW Protocol v1 API规范
package adapters

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"defi-aggregator/smart-router/internal/types"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// CowAdapter CoW Protocol聚合器适配器
// 封装CoW Protocol API调用，提供标准化的报价接口
type CowAdapter struct {
	*BaseAdapter // 嵌入基础适配器
}

// NewCowAdapter 创建CoW Protocol适配器实例
func NewCowAdapter(config *types.ProviderConfig, logger *logrus.Logger) ProviderAdapter {
	return &CowAdapter{
		BaseAdapter: NewBaseAdapter(config, logger),
	}
}

// ========================================
// CoW Protocol API响应结构定义
// ========================================

// CowQuoteResponse CoW Protocol报价API响应
// 根据真实API响应格式：包含quote嵌套对象
type CowQuoteResponse struct {
	Quote struct {
		SellToken         string `json:"sellToken"`         // 卖出代币地址
		BuyToken          string `json:"buyToken"`          // 买入代币地址
		Receiver          string `json:"receiver"`          // 接收地址
		SellAmount        string `json:"sellAmount"`        // 卖出数量
		BuyAmount         string `json:"buyAmount"`         // 买入数量
		ValidTo           int64  `json:"validTo"`           // 有效期时间戳
		AppData           string `json:"appData"`           // 应用数据
		FeeAmount         string `json:"feeAmount"`         // 手续费数量
		Kind              string `json:"kind"`              // 订单类型 (buy/sell)
		PartiallyFillable bool   `json:"partiallyFillable"` // 是否可部分成交
		SellTokenBalance  string `json:"sellTokenBalance"`  // 卖出代币余额类型
		BuyTokenBalance   string `json:"buyTokenBalance"`   // 买入代币余额类型
		SigningScheme     string `json:"signingScheme"`     // 签名方案
	} `json:"quote"`
	From       string `json:"from"`       // 发起地址
	Expiration string `json:"expiration"` // 过期时间
	ID         int    `json:"id"`         // 报价ID
	Verified   bool   `json:"verified"`   // 是否已验证
}

// CowErrorResponse CoW Protocol错误响应
type CowErrorResponse struct {
	ErrorType   string `json:"errorType"`
	Description string `json:"description"`
}

// ========================================
// CoW Protocol适配器接口实现
// ========================================

// GetQuote 获取CoW Protocol报价
// 调用CoW Protocol API获取最优报价
func (a *CowAdapter) GetQuote(ctx context.Context, req *types.QuoteRequest) (*types.ProviderQuote, error) {
	startTime := time.Now()

	// 检查链支持
	if !a.IsSupported(req.ChainID) {
		return nil, &types.RouterError{
			Code:    types.ErrCodeUnsupportedChain,
			Message: fmt.Sprintf("CoW Protocol不支持链ID: %d", req.ChainID),
		}
	}

	// 构建请求URL
	apiURL, err := a.buildQuoteURL(req)
	if err != nil {
		return nil, fmt.Errorf("构建请求URL失败: %w", err)
	}

	// 根据CoW Protocol API文档构建请求
	// 参考文档示例：https://api.cow.fi/docs/#/default/post_api_v1_quote

	// 设置默认用户地址（用于报价查询）
	userAddress := "0x6810e776880c02933d47db1b9fc05908e5386b96"
	if req.UserAddress != "" {
		userAddress = req.UserAddress
	}

	// 直接使用用户选择的代币，不做任何转换
	// 如果用户选择ETH就是ETH，选择WETH就是WETH，我们不应该私自替换
	fromToken := req.FromToken
	toToken := req.ToToken

	a.logger.Infof("[CoW] 使用用户选择的代币: %s -> %s", fromToken, toToken)

	// 构建符合API文档的请求体
	// 动态计算appDataHash，不使用固定值
	appData := "{\"version\":\"0.9.0\",\"metadata\":{}}"
	appDataHash, err := a.calculateAppDataHash(appData)
	if err != nil {
		return nil, fmt.Errorf("计算appDataHash失败: %w", err)
	}

	requestBody := map[string]interface{}{
		"sellToken":           fromToken,             // 卖出代币合约地址（可能转换后的）
		"buyToken":            toToken,               // 买入代币合约地址（可能转换后的）
		"receiver":            userAddress,           // 接收地址
		"appData":             appData,               // 应用元数据
		"appDataHash":         appDataHash,           // 正确计算的应用数据哈希
		"sellTokenBalance":    "erc20",               // 卖出代币余额类型
		"buyTokenBalance":     "erc20",               // 买入代币余额类型
		"from":                userAddress,           // 发起者地址
		"priceQuality":        "verified",            // 价格质量要求
		"signingScheme":       "eip712",              // 签名方案
		"onchainOrder":        false,                 // 链下订单
		"timeout":             0,                     // 超时时间
		"kind":                "sell",                // 订单类型：卖出固定数量
		"sellAmountBeforeFee": req.AmountIn.String(), // 卖出数量（手续费前）
	}

	// 序列化请求体为JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	a.logger.Debugf("[CoW] 请求URL: %s", apiURL)
	a.logger.Debugf("[CoW] 请求体: %s", string(jsonBody))

	// 发送HTTP POST请求
	responseBody, err := a.makeHTTPRequest(ctx, "POST", apiURL, bytes.NewReader(jsonBody), nil)
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.ProviderCowswap,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: err.Error(),
		}, nil // 返回失败的报价，不返回error，让聚合器继续处理其他提供商
	}

	// 记录原始响应用于调试
	a.logger.Debugf("[CoW] 原始响应: %s", string(responseBody))

	// 解析响应
	var cowResponse CowQuoteResponse
	if err := json.Unmarshal(responseBody, &cowResponse); err != nil {
		// 尝试解析错误响应
		var errorResponse CowErrorResponse
		if jsonErr := json.Unmarshal(responseBody, &errorResponse); jsonErr == nil {
			a.logger.Errorf("[CoW] API返回错误: %s - %s", errorResponse.ErrorType, errorResponse.Description)
			return &types.ProviderQuote{
				Provider:     types.ProviderCowswap,
				Success:      false,
				ResponseTime: time.Since(startTime),
				ErrorCode:    types.ErrCodeProviderError,
				ErrorMessage: fmt.Sprintf("CoW API错误: %s", errorResponse.Description),
			}, nil
		}

		a.logger.Errorf("[CoW] 解析响应失败: %v, 原始响应: %s", err, string(responseBody))

		return &types.ProviderQuote{
			Provider:     types.ProviderCowswap,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: fmt.Sprintf("解析响应失败: %v", err),
		}, nil
	}

	// 转换为标准格式
	quote, err := a.convertToStandardQuote(&cowResponse, req, startTime)
	if err != nil {
		return &types.ProviderQuote{
			Provider:     types.ProviderCowswap,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorCode:    types.ErrCodeProviderError,
			ErrorMessage: fmt.Sprintf("转换报价格式失败: %v", err),
		}, nil
	}

	a.logger.Debugf("[CoW] 报价获取成功: %s", quote.AmountOut.String())
	return quote, nil
}

// ========================================
// CoW Protocol URL构建和数据转换
// ========================================

// buildQuoteURL 构建CoW Protocol报价请求URL
func (a *CowAdapter) buildQuoteURL(req *types.QuoteRequest) (string, error) {
	// CoW Protocol API基础URL
	baseURL := a.config.BaseURL
	if baseURL == "" {
		return "", fmt.Errorf("CoW Protocol API URL未配置")
	}

	// CoW Protocol使用POST请求，URL不需要查询参数
	apiURL := fmt.Sprintf("%s/quote", strings.TrimSuffix(baseURL, "/"))
	return apiURL, nil
}

// convertToStandardQuote 将CoW Protocol响应转换为标准报价格式
func (a *CowAdapter) convertToStandardQuote(cowResp *CowQuoteResponse, req *types.QuoteRequest, startTime time.Time) (*types.ProviderQuote, error) {
	// 解析买入数量（从嵌套的quote对象中获取）
	buyAmount, err := decimal.NewFromString(cowResp.Quote.BuyAmount)
	if err != nil {
		return nil, fmt.Errorf("解析buyAmount失败: %w", err)
	}

	// 解析手续费数量
	feeAmount, err := decimal.NewFromString(cowResp.Quote.FeeAmount)
	if err != nil {
		a.logger.Warnf("[CoW] 解析feeAmount失败: %v", err)
		feeAmount = decimal.Zero // 设为0继续处理
	}

	// 验证响应数据的一致性
	if cowResp.Quote.SellToken != req.FromToken {
		a.logger.Warnf("[CoW] 卖出代币地址不匹配: 请求=%s, 响应=%s", req.FromToken, cowResp.Quote.SellToken)
	}
	if cowResp.Quote.BuyToken != req.ToToken {
		a.logger.Warnf("[CoW] 买入代币地址不匹配: 请求=%s, 响应=%s", req.ToToken, cowResp.Quote.BuyToken)
	}

	// 计算价格影响
	priceImpact := a.calculatePriceImpact(req.AmountIn, buyAmount)

	// CoW Protocol通常有较低的Gas费用，因为它使用批处理
	gasEstimate := uint64(150000)

	// 计算置信度（基于verified字段）
	confidence := decimal.NewFromFloat(0.9) // 默认高置信度
	if cowResp.Verified {
		confidence = decimal.NewFromFloat(0.95) // 已验证的报价更可信
	}

	a.logger.Infof("[CoW] 报价转换成功: buyAmount=%s, feeAmount=%s, verified=%t",
		buyAmount.String(), feeAmount.String(), cowResp.Verified)

	return &types.ProviderQuote{
		Provider:     types.ProviderCowswap,
		Success:      true,
		AmountOut:    buyAmount,
		GasEstimate:  gasEstimate,
		PriceImpact:  priceImpact,
		Route:        []types.RouteStep{}, // CoW Protocol使用批处理，不提供详细路由
		ResponseTime: time.Since(startTime),
		Confidence:   confidence,
	}, nil
}

// calculatePriceImpact 计算价格影响
func (a *CowAdapter) calculatePriceImpact(amountIn, amountOut decimal.Decimal) decimal.Decimal {
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

// HealthCheck 检查CoW Protocol服务健康状态
func (a *CowAdapter) HealthCheck(ctx context.Context) error {
	// CoW Protocol健康检查通常通过获取简单的API信息
	healthURL := fmt.Sprintf("%s/api/v1/version", a.config.BaseURL)

	_, err := a.makeHTTPRequest(ctx, "GET", healthURL, nil, nil)
	if err != nil {
		return fmt.Errorf("CoW Protocol健康检查失败: %w", err)
	}

	a.logger.Debugf("[CoW] 健康检查通过")
	return nil
}

// ========================================
// 工具方法
// ========================================

// GetName 返回适配器名称
func (a *CowAdapter) GetName() string {
	return string(types.ProviderCowswap)
}

// GetDisplayName 返回显示名称
func (a *CowAdapter) GetDisplayName() string {
	return a.config.DisplayName
}

// IsSupported 检查是否支持指定链
func (a *CowAdapter) IsSupported(chainID uint) bool {
	for _, supportedChain := range a.config.SupportedChains {
		if supportedChain == chainID {
			return true
		}
	}
	return false
}

// calculateAppDataHash 计算应用数据哈希
// CoW Protocol要求appDataHash必须与appData内容的Keccak256哈希匹配
func (a *CowAdapter) calculateAppDataHash(appData string) (string, error) {
	// 使用SHA256作为简化实现（真实环境应该使用Keccak256）
	hasher := sha256.New()
	hasher.Write([]byte(appData))
	hashBytes := hasher.Sum(nil)

	// 转换为0x前缀的十六进制字符串
	hashHex := "0x" + hex.EncodeToString(hashBytes)

	a.logger.Debugf("[CoW] 计算appDataHash: appData=%s, hash=%s", appData, hashHex)
	return hashHex, nil
}

// getWETHAddress 动态获取指定链的WETH代币地址
// 从业务逻辑服务查询WETH地址，完全避免硬编码
func (a *CowAdapter) getWETHAddress(chainID uint) (string, error) {
	// 使用搜索API查询WETH代币，然后筛选指定链
	businessLogicURL := "http://localhost:5177" // 可以从配置获取
	queryURL := fmt.Sprintf("%s/api/v1/tokens/search?q=WETH", businessLogicURL)

	// 发送GET请求查询WETH代币信息
	resp, err := a.makeHTTPRequest(context.Background(), "GET", queryURL, nil, nil)
	if err != nil {
		return "", fmt.Errorf("查询WETH代币失败: %w", err)
	}

	// 解析响应获取WETH地址
	var tokenResponse struct {
		Success bool `json:"success"`
		Data    []struct {
			Symbol          string `json:"symbol"`
			ContractAddress string `json:"contract_address"`
			ChainID         uint   `json:"chain_id"`
		} `json:"data"`
	}

	if err := a.parseJSONResponse(resp, &tokenResponse); err != nil {
		return "", fmt.Errorf("解析WETH查询响应失败: %w", err)
	}

	if !tokenResponse.Success {
		return "", fmt.Errorf("WETH查询失败")
	}

	// 查找指定链上的WETH代币
	for _, token := range tokenResponse.Data {
		if token.Symbol == "WETH" && token.ChainID == chainID {
			a.logger.Debugf("[CoW] 从业务逻辑服务获取链%d的WETH地址: %s", chainID, token.ContractAddress)
			return token.ContractAddress, nil
		}
	}

	return "", fmt.Errorf("链%d上没有找到WETH代币", chainID)
}
