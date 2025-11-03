// Package handlers 智能路由HTTP处理器
// 提供RESTful API接口，处理报价请求和系统监控
// 实现标准的HTTP错误处理和响应格式
package handlers

import (
	"fmt"
	"net/http"
	"time"

	"defi-aggregator/smart-router/internal/services"
	"defi-aggregator/smart-router/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// RouterHandler 智能路由处理器
// 处理报价聚合相关的HTTP请求
type RouterHandler struct {
	routerService *services.RouterService // 路由服务
	logger        *logrus.Logger          // 日志记录器
}

// NewRouterHandler 创建路由处理器实例
func NewRouterHandler(routerService *services.RouterService, logger *logrus.Logger) *RouterHandler {
	return &RouterHandler{
		routerService: routerService,
		logger:        logger,
	}
}

// ========================================
// 核心API接口
// ========================================

// GetQuote 获取最优报价
// POST /api/v1/quote
// 智能路由的核心接口，返回聚合后的最优报价
func (h *RouterHandler) GetQuote(c *gin.Context) {
	requestID := h.getOrGenerateRequestID(c)
	startTime := time.Now()

	h.logger.Infof("[%s] 收到报价请求", requestID)

	// 绑定请求参数
	var req types.QuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnf("[%s] 报价请求参数无效: %v", requestID, err)
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeInvalidRequest,
				Message: "请求参数无效",
				Details: map[string]interface{}{"error": err.Error()},
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	// 设置请求ID
	if req.RequestID == "" {
		req.RequestID = requestID
	}

	// 验证请求参数
	if err := h.validateQuoteRequest(&req); err != nil {
		h.logger.Warnf("[%s] 报价请求验证失败: %v", requestID, err)
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeInvalidRequest,
				Message: err.Error(),
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	// 调用智能路由服务
	ctx := c.Request.Context()
	quote, err := h.routerService.GetOptimalQuote(ctx, &req)
	if err != nil {
		h.handleRouterError(c, err, requestID)
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    quote,
		Meta: map[string]interface{}{
			"processing_time": time.Since(startTime).Milliseconds(),
			"cache_hit":       quote.CacheHit,
			"providers_used":  quote.Performance.ProvidersQueried,
		},
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	h.logger.Infof("[%s] 报价请求处理完成: provider=%s, duration=%v",
		requestID, quote.BestProvider, time.Since(startTime))
}

// ========================================
// 监控和管理接口
// ========================================

// GetMetrics 获取服务指标
// GET /api/v1/metrics
// 返回智能路由服务的性能指标
func (h *RouterHandler) GetMetrics(c *gin.Context) {
	requestID := h.getOrGenerateRequestID(c)

	// 获取路由服务指标
	routerMetrics := h.routerService.GetMetrics()

	// 获取缓存统计
	// cacheStats, err := h.routerService.GetCacheStats()
	// if err != nil {
	//     h.logger.Warnf("[%s] 获取缓存统计失败: %v", requestID, err)
	// }

	// 构建响应
	metrics := map[string]interface{}{
		"router": routerMetrics,
		// "cache":  cacheStats,
		"timestamp": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      metrics,
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	h.logger.Debugf("[%s] 指标查询完成", requestID)
}

// HealthCheck 健康检查
// GET /health
// 检查服务和依赖组件的健康状态
func (h *RouterHandler) HealthCheck(c *gin.Context) {
	requestID := h.getOrGenerateRequestID(c)

	// TODO: 实现完整的健康检查
	// 1. 检查各个聚合器的健康状态
	// 2. 检查缓存连接
	// 3. 检查系统资源

	healthResponse := &types.HealthCheckResponse{
		Status:    types.StatusHealthy,
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    time.Since(time.Now()), // TODO: 计算真实的运行时间
	}

	c.JSON(http.StatusOK, healthResponse)
	h.logger.Debugf("[%s] 健康检查完成", requestID)
}

// GetProviderStatus 获取聚合器状态
// GET /api/v1/providers/status
// 返回所有聚合器的健康状态和性能指标
func (h *RouterHandler) GetProviderStatus(c *gin.Context) {
	requestID := h.getOrGenerateRequestID(c)

	// TODO: 实现聚合器状态查询
	// 从RouterService获取各个适配器的状态

	status := map[string]interface{}{
		"message": "Provider status not implemented yet",
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      status,
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	h.logger.Debugf("[%s] 聚合器状态查询完成", requestID)
}

// ========================================
// 辅助方法
// ========================================

// validateQuoteRequest 验证报价请求参数
func (h *RouterHandler) validateQuoteRequest(req *types.QuoteRequest) error {
	if req.FromToken == "" {
		return fmt.Errorf("源代币地址不能为空")
	}

	if req.ToToken == "" {
		return fmt.Errorf("目标代币地址不能为空")
	}

	if req.FromToken == req.ToToken {
		return fmt.Errorf("源代币和目标代币不能相同")
	}

	if req.AmountIn.IsZero() || req.AmountIn.IsNegative() {
		return fmt.Errorf("输入数量必须大于0")
	}

	if req.ChainID == 0 {
		return fmt.Errorf("链ID不能为0")
	}

	if req.Slippage.IsNegative() || req.Slippage.GreaterThan(decimal.NewFromFloat(0.5)) {
		return fmt.Errorf("滑点必须在0-50%%之间")
	}

	return nil
}

// getOrGenerateRequestID 获取或生成请求ID
func (h *RouterHandler) getOrGenerateRequestID(c *gin.Context) string {
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}

	if requestID := c.GetString("request_id"); requestID != "" {
		return requestID
	}

	// 生成新的请求ID
	requestID := uuid.New().String()
	c.Set("request_id", requestID)
	return requestID
}

// handleRouterError 处理路由服务错误
func (h *RouterHandler) handleRouterError(c *gin.Context, err error, requestID string) {
	// 检查是否为路由服务错误
	if routerErr, ok := err.(*types.RouterError); ok {
		var statusCode int
		switch routerErr.Code {
		case types.ErrCodeInvalidRequest:
			statusCode = http.StatusBadRequest
		case types.ErrCodeUnsupportedChain:
			statusCode = http.StatusBadRequest
		case types.ErrCodeNoValidQuotes:
			statusCode = http.StatusServiceUnavailable
		case types.ErrCodeRateLimitExceeded:
			statusCode = http.StatusTooManyRequests
		default:
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    routerErr.Code,
				Message: routerErr.Message,
				Details: routerErr.Details,
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})

		if statusCode >= 500 {
			h.logger.Errorf("[%s] 路由服务错误: %v", requestID, err)
		} else {
			h.logger.Warnf("[%s] 路由服务错误: %v", requestID, err)
		}
	} else {
		// 未知错误
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeInternalError,
				Message: "内部服务错误",
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})

		h.logger.Errorf("[%s] 未知错误: %v", requestID, err)
	}
}
