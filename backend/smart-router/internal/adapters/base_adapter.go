// Package adapters 第三方聚合器适配器
// 提供统一的聚合器接口，封装不同聚合器的API差异
// 实现适配器模式，支持新聚合器的热插拔
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"defi-aggregator/smart-router/internal/types"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// BaseAdapter 基础适配器结构
// 提供所有适配器的通用功能和配置
type BaseAdapter struct {
	config     *types.ProviderConfig // 聚合器配置
	httpClient *http.Client          // HTTP客户端
	logger     *logrus.Logger        // 日志记录器
	metrics    *AdapterMetrics       // 性能指标
}

// AdapterMetrics 适配器性能指标
// 记录适配器的运行时性能数据
type AdapterMetrics struct {
	TotalRequests   int64         `json:"total_requests"`    // 总请求数
	SuccessRequests int64         `json:"success_requests"`  // 成功请求数
	FailedRequests  int64         `json:"failed_requests"`   // 失败请求数
	AvgResponseTime time.Duration `json:"avg_response_time"` // 平均响应时间
	LastRequestTime time.Time     `json:"last_request_time"` // 最后请求时间
}

// NewBaseAdapter 创建基础适配器
// 初始化通用的HTTP客户端和配置
func NewBaseAdapter(config *types.ProviderConfig, logger *logrus.Logger) *BaseAdapter {
	// 创建专用的HTTP客户端
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	return &BaseAdapter{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
		metrics:    &AdapterMetrics{},
	}
}

// ========================================
// 通用HTTP请求方法
// ========================================

// makeHTTPRequest 发送HTTP请求
// 统一的HTTP请求方法，包含重试、超时、错误处理等
// 参数:
//   - ctx: 上下文，用于超时控制
//   - method: HTTP方法
//   - url: 请求URL
//   - body: 请求体
//   - headers: 请求头
//
// 返回:
//   - []byte: 响应体
//   - error: 请求错误
func (b *BaseAdapter) makeHTTPRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) ([]byte, error) {
	startTime := time.Now()

	// 记录请求开始
	b.logger.Debugf("[%s] 开始请求: %s %s", b.config.Name, method, url)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DeFi-Aggregator-Smart-Router/1.0")

	// 添加API密钥（如果存在）
	if b.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.config.APIKey))
	}

	// 添加自定义请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 执行请求（带重试）
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= b.config.RetryCount; attempt++ {
		if attempt > 0 {
			// 重试前等待
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 100 * time.Millisecond):
			}
			b.logger.Debugf("[%s] 重试请求: attempt=%d", b.config.Name, attempt)
		}

		resp, lastErr = b.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode < 500 {
			// 请求成功或客户端错误（不重试）
			break
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	if lastErr != nil {
		b.updateMetrics(false, time.Since(startTime))
		return nil, fmt.Errorf("HTTP请求失败: %w", lastErr)
	}
	defer resp.Body.Close()

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		b.updateMetrics(false, time.Since(startTime))
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode >= 400 {
		b.updateMetrics(false, time.Since(startTime))
		return nil, fmt.Errorf("HTTP错误: status=%d, body=%s", resp.StatusCode, string(responseBody))
	}

	// 更新性能指标
	duration := time.Since(startTime)
	b.updateMetrics(true, duration)

	b.logger.Debugf("[%s] 请求完成: duration=%v, status=%d",
		b.config.Name, duration, resp.StatusCode)

	return responseBody, nil
}

// ========================================
// 通用数据处理方法
// ========================================

// parseJSONResponse 解析JSON响应
// 统一的JSON解析方法，包含错误处理
func (b *BaseAdapter) parseJSONResponse(data []byte, target interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		b.logger.Errorf("[%s] JSON解析失败: %v, data=%s", b.config.Name, err, string(data))
		return fmt.Errorf("JSON解析失败: %w", err)
	}
	return nil
}

// standardizeAmount 标准化金额格式
// 将不同聚合器的金额格式转换为统一的decimal.Decimal
func (b *BaseAdapter) standardizeAmount(amount interface{}) (decimal.Decimal, error) {
	switch v := amount.(type) {
	case string:
		return decimal.NewFromString(v)
	case float64:
		return decimal.NewFromFloat(v), nil
	case int64:
		return decimal.NewFromInt(v), nil
	case int:
		return decimal.NewFromInt(int64(v)), nil
	default:
		return decimal.Zero, fmt.Errorf("不支持的金额类型: %T", amount)
	}
}

// calculatePriceImpact 计算价格冲击
// 基于输入输出金额计算价格冲击百分比
func (b *BaseAdapter) calculatePriceImpact(amountIn, amountOut, marketPrice decimal.Decimal) decimal.Decimal {
	if marketPrice.IsZero() || amountIn.IsZero() {
		return decimal.Zero
	}

	// 计算实际汇率
	actualRate := amountOut.Div(amountIn)

	// 计算价格冲击 = (市场价格 - 实际汇率) / 市场价格
	priceImpact := marketPrice.Sub(actualRate).Div(marketPrice)

	// 确保价格冲击为正数
	if priceImpact.IsNegative() {
		priceImpact = priceImpact.Neg()
	}

	return priceImpact
}

// ========================================
// 性能指标管理
// ========================================

// updateMetrics 更新适配器性能指标
// 记录每次请求的结果和响应时间
func (b *BaseAdapter) updateMetrics(success bool, duration time.Duration) {
	b.metrics.TotalRequests++
	b.metrics.LastRequestTime = time.Now()

	if success {
		b.metrics.SuccessRequests++
	} else {
		b.metrics.FailedRequests++
	}

	// 更新平均响应时间（简化计算）
	if b.metrics.TotalRequests == 1 {
		b.metrics.AvgResponseTime = duration
	} else {
		// 使用滑动平均
		alpha := 0.1 // 平滑因子
		b.metrics.AvgResponseTime = time.Duration(
			float64(b.metrics.AvgResponseTime)*(1-alpha) + float64(duration)*alpha,
		)
	}
}

// GetMetrics 获取适配器性能指标
func (b *BaseAdapter) GetMetrics() *AdapterMetrics {
	return b.metrics
}

// ResetMetrics 重置适配器性能指标
func (b *BaseAdapter) ResetMetrics() {
	b.metrics = &AdapterMetrics{}
}

// ========================================
// 配置管理
// ========================================

// UpdateConfig 更新适配器配置
func (b *BaseAdapter) UpdateConfig(config *types.ProviderConfig) error {
	b.config = config

	// 更新HTTP客户端超时
	b.httpClient.Timeout = config.Timeout

	b.logger.Infof("[%s] 配置已更新", config.Name)
	return nil
}

// GetConfig 获取当前配置
func (b *BaseAdapter) GetConfig() *types.ProviderConfig {
	return b.config
}

// GetName 获取聚合器名称
func (b *BaseAdapter) GetName() string {
	return b.config.Name
}

// GetDisplayName 获取显示名称
func (b *BaseAdapter) GetDisplayName() string {
	return b.config.DisplayName
}

// IsSupported 检查是否支持指定链
func (b *BaseAdapter) IsSupported(chainID uint) bool {
	for _, supportedChain := range b.config.SupportedChains {
		if supportedChain == chainID {
			return true
		}
	}
	return false
}
