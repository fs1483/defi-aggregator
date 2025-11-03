// Package controllers 代币控制器实现
// 处理代币相关的HTTP请求，包括代币列表、搜索、价格查询等
// 提供公开的RESTful API接口，无需用户认证
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

// TokenController 代币控制器
// 处理代币相关的HTTP请求
type TokenController struct {
	tokenService services.TokenService // 代币业务服务
	chainService services.ChainService // 区块链业务服务
	cfg          *config.Config        // 应用配置
	logger       *logrus.Logger        // 日志记录器
}

// NewTokenController 创建代币控制器实例
func NewTokenController(tokenService services.TokenService, chainService services.ChainService, cfg *config.Config, logger *logrus.Logger) *TokenController {
	return &TokenController{
		tokenService: tokenService,
		chainService: chainService,
		cfg:          cfg,
		logger:       logger,
	}
}

// ========================================
// 代币查询接口
// ========================================

// GetTokens 获取代币列表
// GET /api/v1/tokens
// 支持分页、筛选、搜索等功能
func (c *TokenController) GetTokens(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析查询参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	// 筛选参数 - 直接使用内部chain_id（数据库chains表的主键ID）
	var chainID *uint
	if chainIDStr := ctx.Query("chain_id"); chainIDStr != "" {
		if parsedChainID, err := strconv.ParseUint(chainIDStr, 10, 32); err == nil {
			chainIDUint := uint(parsedChainID)
			chainID = &chainIDUint
		}
	}

	var isActive *bool
	if isActiveStr := ctx.Query("is_active"); isActiveStr != "" {
		if active, err := strconv.ParseBool(isActiveStr); err == nil {
			isActive = &active
		}
	}

	var isVerified *bool
	if isVerifiedStr := ctx.Query("is_verified"); isVerifiedStr != "" {
		if verified, err := strconv.ParseBool(isVerifiedStr); err == nil {
			isVerified = &verified
		}
	}

	var search *string
	if searchStr := ctx.Query("search"); searchStr != "" {
		search = &searchStr
	}

	// 验证分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > c.cfg.Business.MaxPageSize {
		pageSize = c.cfg.Business.DefaultPageSize
	}

	// 构建请求
	req := &types.TokenListRequest{
		PaginationRequest: types.PaginationRequest{
			Page:     page,
			PageSize: pageSize,
			SortBy:   ctx.DefaultQuery("sort_by", "symbol"),
			SortDesc: ctx.DefaultQuery("sort_desc", "false") == "true",
		},
		ChainID:    chainID,
		IsActive:   isActive,
		IsVerified: isVerified,
		Search:     search,
	}

	c.logger.Debugf("[%s] 获取代币列表: page=%d, pageSize=%d, chainID=%v",
		requestID, page, pageSize, chainID)

	// 调用代币服务获取列表
	tokens, meta, err := c.tokenService.GetTokens(req)
	if err != nil {
		c.handleServiceError(ctx, err, "获取代币列表失败")
		return
	}

	// 返回代币列表
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      tokens,
		Meta:      meta,
		Message:   "获取代币列表成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 代币列表获取成功: total=%d", requestID, meta.Total)
}

// GetToken 获取代币详情
// GET /api/v1/tokens/:id
// 返回指定代币的详细信息
func (c *TokenController) GetToken(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析代币ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.logger.Warnf("[%s] 无效的代币ID: %s", requestID, idStr)
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "无效的代币ID",
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	c.logger.Debugf("[%s] 获取代币详情: tokenID=%d", requestID, uint(id))

	// 调用代币服务获取详情
	token, err := c.tokenService.GetTokenByID(uint(id))
	if err != nil {
		c.handleServiceError(ctx, err, "获取代币详情失败")
		return
	}

	// 返回代币详情
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      token,
		Message:   "获取代币详情成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 代币 %d 详情获取成功", requestID, uint(id))
}

// SearchTokens 搜索代币
// GET /api/v1/tokens/search
// 根据代币符号或名称进行模糊搜索
func (c *TokenController) SearchTokens(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 获取搜索关键词
	query := ctx.Query("q")
	if query == "" {
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "搜索关键词不能为空",
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	c.logger.Debugf("[%s] 搜索代币: query=%s", requestID, query)

	// 调用代币服务搜索
	tokens, err := c.tokenService.SearchTokens(query)
	if err != nil {
		c.handleServiceError(ctx, err, "搜索代币失败")
		return
	}

	// 返回搜索结果
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      tokens,
		Message:   "代币搜索成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 代币搜索成功: query=%s, results=%d", requestID, query, len(tokens))
}

// GetTokensByChain 获取指定链的代币
// GET /api/v1/tokens/chain/:chainId
// 返回特定区块链上的所有代币
func (c *TokenController) GetTokensByChain(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析链ID
	chainIDStr := ctx.Param("chainId")
	chainID, err := strconv.ParseUint(chainIDStr, 10, 32)
	if err != nil {
		c.logger.Warnf("[%s] 无效的链ID: %s", requestID, chainIDStr)
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "无效的链ID",
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	c.logger.Debugf("[%s] 获取链代币: chainID=%d", requestID, uint(chainID))

	// 调用代币服务获取链代币
	tokens, err := c.tokenService.GetTokensByChain(uint(chainID))
	if err != nil {
		c.handleServiceError(ctx, err, "获取链代币失败")
		return
	}

	// 返回链代币列表
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      tokens,
		Message:   "获取链代币成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 链 %d 代币获取成功: count=%d", requestID, uint(chainID), len(tokens))
}

// GetPopularTokens 获取热门代币
// GET /api/v1/tokens/popular
// 返回交易量大、市值高的热门代币
func (c *TokenController) GetPopularTokens(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析限制数量
	limitStr := ctx.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	c.logger.Debugf("[%s] 获取热门代币: limit=%d", requestID, limit)

	// 调用代币服务获取热门代币
	tokens, err := c.tokenService.GetPopularTokens(limit)
	if err != nil {
		c.handleServiceError(ctx, err, "获取热门代币失败")
		return
	}

	// 返回热门代币列表
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      tokens,
		Message:   "获取热门代币成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 热门代币获取成功: count=%d", requestID, len(tokens))
}

// ========================================
// 辅助方法
// ========================================

// handleServiceError 处理业务服务错误
// 将业务层错误转换为适当的HTTP响应
func (c *TokenController) handleServiceError(ctx *gin.Context, err error, defaultMessage string) {
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
