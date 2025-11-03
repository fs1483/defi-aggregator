// Package controllers 认证控制器实现
// 处理用户认证相关的HTTP请求，包括Web3钱包登录、JWT管理等
// 遵循RESTful API设计规范和企业级安全最佳实践
package controllers

import (
	"net/http"
	"time"

	"defi-aggregator/business-logic/internal/services"
	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"
	"defi-aggregator/business-logic/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthController 认证控制器
// 处理用户认证相关的HTTP请求
type AuthController struct {
	authService services.AuthService // 认证业务服务
	cfg         *config.Config       // 应用配置
	logger      *logrus.Logger       // 日志记录器
}

// NewAuthController 创建认证控制器实例
func NewAuthController(authService services.AuthService, cfg *config.Config, logger *logrus.Logger) *AuthController {
	return &AuthController{
		authService: authService,
		cfg:         cfg,
		logger:      logger,
	}
}

// ========================================
// Web3钱包认证接口
// ========================================

// GetNonce 获取登录随机数
// POST /api/v1/auth/nonce
// 为指定钱包地址生成登录用的随机数
func (c *AuthController) GetNonce(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 定义请求结构体
	var req struct {
		WalletAddress string `json:"wallet_address" binding:"required"` // 钱包地址
	}

	// 绑定请求参数
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warnf("[%s] 获取nonce请求参数无效: %v", requestID, err)
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

	// 调用业务服务生成nonce
	nonce, err := c.authService.GenerateNonce(req.WalletAddress)
	if err != nil {
		c.handleServiceError(ctx, err, "生成随机数失败")
		return
	}

	// 构建签名消息
	timestamp := utils.GetCurrentTimestamp()
	message := utils.FormatSignMessage(req.WalletAddress, nonce, timestamp)

	// 返回成功响应
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data: gin.H{
			"nonce":          nonce,
			"message":        message,
			"timestamp":      timestamp,
			"wallet_address": req.WalletAddress,
		},
		Message:   "随机数生成成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 为钱包 %s 生成nonce成功", requestID, req.WalletAddress)
}

// Login 用户登录
// POST /api/v1/auth/login
// 验证钱包签名并返回JWT令牌
func (c *AuthController) Login(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 绑定登录请求
	var req types.UserLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warnf("[%s] 登录请求参数无效: %v", requestID, err)
		ctx.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeValidation,
				Message: "登录参数无效",
				Details: map[string]interface{}{"error": err.Error()},
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	// 记录登录尝试
	c.logger.Infof("[%s] 用户登录尝试: wallet=%s", requestID, req.WalletAddress)

	// 调用认证服务验证签名
	loginResponse, err := c.authService.VerifySignature(&req)
	if err != nil {
		c.handleServiceError(ctx, err, "登录验证失败")
		return
	}

	// 返回登录成功响应
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      loginResponse,
		Message:   "登录成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 用户 %s 登录成功", requestID, req.WalletAddress)
}

// Logout 用户登出
// POST /api/v1/auth/logout
// 撤销用户的访问令牌
func (c *AuthController) Logout(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	userID := ctx.GetUint("user_id")

	// 调用认证服务登出用户
	if err := c.authService.LogoutUser(userID); err != nil {
		c.handleServiceError(ctx, err, "登出失败")
		return
	}

	// 返回登出成功响应
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      gin.H{"message": "登出成功"},
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 用户 %d 登出成功", requestID, userID)
}

// RefreshToken 刷新访问令牌
// POST /api/v1/auth/refresh
// 使用刷新令牌获取新的访问令牌
func (c *AuthController) RefreshToken(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")

	// 定义请求结构体
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"` // 刷新令牌
	}

	// 绑定请求参数
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warnf("[%s] 刷新令牌请求参数无效: %v", requestID, err)
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

	// 调用认证服务刷新令牌
	newAccessToken, err := c.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.handleServiceError(ctx, err, "令牌刷新失败")
		return
	}

	// 返回新的访问令牌
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data: gin.H{
			"access_token": newAccessToken,
			"expires_in":   int64(c.cfg.JWT.ExpiresIn.Seconds()),
			"token_type":   "Bearer",
		},
		Message:   "令牌刷新成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 刷新令牌成功", requestID)
}

// ========================================
// 辅助方法
// ========================================

// handleServiceError 处理业务服务错误
// 将业务层错误转换为适当的HTTP响应
func (c *AuthController) handleServiceError(ctx *gin.Context, err error, defaultMessage string) {
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
