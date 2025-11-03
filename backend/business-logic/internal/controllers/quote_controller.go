// Package controllers 报价控制器实现
// 处理DeFi聚合器报价相关的HTTP请求
// 提供获取最优报价、报价历史等功能的RESTful API接口
package controllers

import (
	"net/http"
	"strconv"
	"time"

	"defi-aggregator/business-logic/internal/services"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// QuoteController 报价控制器
// 处理报价相关的HTTP请求
type QuoteController struct {
	quoteService services.QuoteService // 报价业务服务
	cfg          *config.Config        // 应用配置
	logger       *logrus.Logger        // 日志记录器
}

// NewQuoteController 创建报价控制器实例
func NewQuoteController(quoteService services.QuoteService, cfg *config.Config, logger *logrus.Logger) *QuoteController {
	return &QuoteController{
		quoteService: quoteService,
		cfg:          cfg,
		logger:       logger,
	}
}

// ========================================
// 报价核心接口
// ========================================

// GetQuote 获取最优报价
// POST /api/v1/quotes
// DeFi聚合器的核心接口，返回多个聚合器中的最优报价
func (c *QuoteController) GetQuote(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	c.logger.Infof("[%s] 收到报价请求", requestID)

	// 绑定报价请求参数
	var req types.QuoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warnf("[%s] 报价请求参数无效: %v", requestID, err)
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "报价请求参数无效",
				Details: map[string]interface{}{"error": err.Error()},
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	// 记录请求详情
	c.logger.Debugf("[%s] 报价请求详情: fromToken=%d, toToken=%d, amount=%s, chain=%d",
		requestID, req.FromTokenID, req.ToTokenID, req.AmountIn.String(), req.ChainID)

	// 调用报价服务获取最优报价
	startTime := time.Now()
	quote, err := c.quoteService.GetQuote(&req)
	if err != nil {
		c.handleServiceError(ctx, err, "获取报价失败")
		return
	}

	// 返回报价结果
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    quote,
		Meta: &types.Meta{
			Page:     1,
			PageSize: 1,
			Total:    1,
		},
		Message:   "获取最优报价成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 报价请求处理完成: provider=%s, amountOut=%s, duration=%v",
		requestID, quote.BestAggregator, quote.AmountOut.String(), time.Since(startTime))
}

// GetQuoteHistory 获取报价历史
// GET /api/v1/quotes/history
// 返回用户的历史报价记录，支持分页和筛选
func (c *QuoteController) GetQuoteHistory(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	sortBy := ctx.DefaultQuery("sort_by", "created_at")
	sortDesc := ctx.DefaultQuery("sort_desc", "true") == "true"

	// 验证分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > c.cfg.Business.MaxPageSize {
		pageSize = c.cfg.Business.DefaultPageSize
	}

	paginationReq := &types.PaginationRequest{
		Page:     page,
		PageSize: pageSize,
		SortBy:   sortBy,
		SortDesc: sortDesc,
	}

	// 获取用户ID（如果已认证）
	var userID *uint
	if uid, exists := ctx.Get("user_id"); exists {
		if id, ok := uid.(uint); ok {
			userID = &id
		}
	}

	c.logger.Debugf("[%s] 获取报价历史: userID=%v, page=%d, pageSize=%d",
		requestID, userID, page, pageSize)

	// 调用报价服务获取历史
	quotes, meta, err := c.quoteService.GetQuoteHistory(userID, paginationReq)
	if err != nil {
		c.handleServiceError(ctx, err, "获取报价历史失败")
		return
	}

	// 返回历史记录
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      quotes,
		Meta:      meta,
		Message:   "获取报价历史成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 报价历史获取成功: total=%d", requestID, meta.Total)
}

// GetQuoteDetails 获取报价详情
// GET /api/v1/quotes/:requestId
// 根据请求ID获取详细的报价信息和聚合器响应
func (c *QuoteController) GetQuoteDetails(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	quoteRequestID := ctx.Param("requestId")

	c.logger.Debugf("[%s] 获取报价详情: quoteRequestID=%s", requestID, quoteRequestID)

	// 调用报价服务获取详情
	quote, err := c.quoteService.GetQuoteDetails(quoteRequestID)
	if err != nil {
		c.handleServiceError(ctx, err, "获取报价详情失败")
		return
	}

	// 返回报价详情
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      quote,
		Message:   "获取报价详情成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 报价详情获取成功: quoteRequestID=%s", requestID, quoteRequestID)
}

// ========================================
// 报价分析接口
// ========================================

// CompareQuotes 比较报价结果
// GET /api/v1/quotes/:requestId/compare
// 返回所有聚合器的报价对比分析
func (c *QuoteController) CompareQuotes(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	quoteRequestID := ctx.Param("requestId")

	c.logger.Debugf("[%s] 比较报价: quoteRequestID=%s", requestID, quoteRequestID)

	// 调用报价服务进行比较
	comparison, err := c.quoteService.CompareQuotes(quoteRequestID)
	if err != nil {
		c.handleServiceError(ctx, err, "比较报价失败")
		return
	}

	// 返回比较结果
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      comparison,
		Message:   "报价比较成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 报价比较完成: quoteRequestID=%s", requestID, quoteRequestID)
}

// GetPriceImpactAnalysis 获取价格冲击分析
// POST /api/v1/quotes/price-impact
// 分析指定交易的价格冲击和市场影响
func (c *QuoteController) GetPriceImpactAnalysis(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 绑定价格冲击分析请求
	var req types.QuoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warnf("[%s] 价格冲击分析请求参数无效: %v", requestID, err)
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "请求参数无效",
				Details: map[string]interface{}{"error": err.Error()},
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	c.logger.Debugf("[%s] 价格冲击分析: amount=%s", requestID, req.AmountIn.String())

	// 调用报价服务进行价格冲击分析
	analysis, err := c.quoteService.GetPriceImpactAnalysis(&req)
	if err != nil {
		c.handleServiceError(ctx, err, "价格冲击分析失败")
		return
	}

	// 返回分析结果
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      analysis,
		Message:   "价格冲击分析完成",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 价格冲击分析完成", requestID)
}

// ========================================
// 缓存管理接口
// ========================================

// InvalidateCache 失效报价缓存
// DELETE /api/v1/quotes/cache/:fromTokenId/:toTokenId
// 管理员功能，手动失效特定代币对的缓存
func (c *QuoteController) InvalidateCache(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析代币ID
	fromTokenIDStr := ctx.Param("fromTokenId")
	toTokenIDStr := ctx.Param("toTokenId")

	fromTokenID, err := strconv.ParseUint(fromTokenIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "无效的源代币ID",
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	toTokenID, err := strconv.ParseUint(toTokenIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "无效的目标代币ID",
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	c.logger.Infof("[%s] 失效缓存: fromTokenID=%d, toTokenID=%d", requestID, fromTokenID, toTokenID)

	// 调用报价服务失效缓存
	if err := c.quoteService.InvalidateQuoteCache(uint(fromTokenID), uint(toTokenID)); err != nil {
		c.handleServiceError(ctx, err, "失效缓存失败")
		return
	}

	// 返回成功响应
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      gin.H{"message": "缓存失效成功"},
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 缓存失效成功", requestID)
}

// ========================================
// 辅助方法
// ========================================

// handleServiceError 处理业务服务错误
func (c *QuoteController) handleServiceError(ctx *gin.Context, err error, defaultMessage string) {
	requestID := ctx.GetString("request_id")

	// 检查是否为业务服务错误
	if serviceErr, ok := err.(*services.ServiceError); ok {
		// 根据错误代码确定HTTP状态码
		var statusCode int
		switch serviceErr.Code {
		case types.ErrCodeValidation:
			statusCode = http.StatusBadRequest
		case types.ErrCodeUnauthorized:
			statusCode = http.StatusUnauthorized
		case types.ErrCodeForbidden:
			statusCode = http.StatusForbidden
		case types.ErrCodeNotFound:
			statusCode = http.StatusNotFound
		case types.ErrCodeConflict:
			statusCode = http.StatusConflict
		case types.ErrCodeExternalAPI:
			statusCode = http.StatusServiceUnavailable
		case types.ErrCodeRateLimit:
			statusCode = http.StatusTooManyRequests
		default:
			statusCode = http.StatusInternalServerError
		}

		ctx.JSON(statusCode, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    serviceErr.Code,
				Message: serviceErr.Message,
				Details: serviceErr.Details,
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})

		// 记录错误日志
		if statusCode >= 500 {
			c.logger.Errorf("[%s] %s: %v", requestID, defaultMessage, err)
		} else {
			c.logger.Warnf("[%s] %s: %v", requestID, defaultMessage, err)
		}
	} else {
		// 未知错误，返回通用内部错误
		ctx.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeInternal,
				Message: defaultMessage,
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})

		c.logger.Errorf("[%s] %s: %v", requestID, defaultMessage, err)
	}
}
