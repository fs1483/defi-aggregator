// Package controllers 区块链控制器实现
// 处理区块链网络相关的HTTP请求，包括链列表、链详情等
// 提供公开的API接口，支持前端获取支持的区块链信息
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

// ChainController 区块链控制器
// 处理区块链相关的HTTP请求
type ChainController struct {
	chainService services.ChainService // 区块链业务服务
	cfg          *config.Config        // 应用配置
	logger       *logrus.Logger        // 日志记录器
}

// NewChainController 创建区块链控制器实例
func NewChainController(chainService services.ChainService, cfg *config.Config, logger *logrus.Logger) *ChainController {
	return &ChainController{
		chainService: chainService,
		cfg:          cfg,
		logger:       logger,
	}
}

// ========================================
// 区块链查询接口
// ========================================

// GetChains 获取区块链列表
// GET /api/v1/chains
// 返回系统支持的所有区块链网络
func (c *ChainController) GetChains(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析筛选参数
	filterType := ctx.DefaultQuery("type", "all") // all, mainnet, testnet, active

	c.logger.Debugf("[%s] 获取区块链列表: type=%s", requestID, filterType)

	var chains []*types.ChainInfo
	var err error

	// 根据筛选类型调用相应的服务方法
	switch filterType {
	case "active":
		chains, err = c.chainService.GetActiveChains()
	case "mainnet":
		chains, err = c.chainService.GetMainnetChains()
	case "testnet":
		chains, err = c.chainService.GetTestnetChains()
	default:
		chains, err = c.chainService.GetChains()
	}

	if err != nil {
		c.handleServiceError(ctx, err, "获取区块链列表失败")
		return
	}

	// 返回区块链列表
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      chains,
		Message:   "获取区块链列表成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 区块链列表获取成功: type=%s, count=%d", requestID, filterType, len(chains))
}

// GetChain 获取区块链详情
// GET /api/v1/chains/:id
// 返回指定区块链的详细信息
func (c *ChainController) GetChain(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析区块链ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.logger.Warnf("[%s] 无效的区块链ID: %s", requestID, idStr)
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "无效的区块链ID",
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	c.logger.Debugf("[%s] 获取区块链详情: chainID=%d", requestID, uint(id))

	// 调用区块链服务获取详情
	chain, err := c.chainService.GetChainByID(uint(id))
	if err != nil {
		c.handleServiceError(ctx, err, "获取区块链详情失败")
		return
	}

	// 返回区块链详情
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      chain,
		Message:   "获取区块链详情成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 区块链 %d 详情获取成功", requestID, uint(id))
}

// GetChainTokens 获取区块链的代币列表
// GET /api/v1/chains/:id/tokens
// 返回指定区块链上的所有代币
func (c *ChainController) GetChainTokens(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 解析区块链ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.logger.Warnf("[%s] 无效的区块链ID: %s", requestID, idStr)
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "无效的区块链ID",
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	c.logger.Debugf("[%s] 获取区块链代币: chainID=%d", requestID, uint(id))

	// 临时空实现
	// TODO: 需要在ChainService中添加GetTokensByChain方法
	// 或者调用TokenService.GetTokensByChain
	tokens := []*types.TokenInfo{}

	// 返回区块链代币列表
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      tokens,
		Message:   "获取区块链代币成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 区块链 %d 代币获取成功: count=%d", requestID, uint(id), len(tokens))
}

// ========================================
// 辅助方法
// ========================================

// handleServiceError 处理业务服务错误
func (c *ChainController) handleServiceError(ctx *gin.Context, err error, defaultMessage string) {
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
