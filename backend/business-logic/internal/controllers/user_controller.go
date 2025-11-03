// Package controllers 用户控制器实现
// 处理用户信息管理相关的HTTP请求，包括资料管理、偏好设置等
// 提供RESTful API接口，确保数据安全和操作审计
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

// UserController 用户控制器
// 处理用户相关的HTTP请求
type UserController struct {
	userService services.UserService // 用户业务服务
	cfg         *config.Config       // 应用配置
	logger      *logrus.Logger       // 日志记录器
}

// NewUserController 创建用户控制器实例
func NewUserController(userService services.UserService, cfg *config.Config, logger *logrus.Logger) *UserController {
	return &UserController{
		userService: userService,
		cfg:         cfg,
		logger:      logger,
	}
}

// ========================================
// 用户资料管理接口
// ========================================

// GetProfile 获取用户资料
// GET /api/v1/users/profile
// 返回当前认证用户的详细资料信息
func (c *UserController) GetProfile(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	userID := ctx.GetUint("user_id")

	c.logger.Debugf("[%s] 获取用户资料: userID=%d", requestID, userID)

	// 调用用户服务获取资料
	profile, err := c.userService.GetProfile(userID)
	if err != nil {
		c.handleServiceError(ctx, err, "获取用户资料失败")
		return
	}

	// 返回用户资料
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      profile,
		Message:   "获取用户资料成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 用户 %d 资料获取成功", requestID, userID)
}

// UpdateProfile 更新用户资料
// PUT /api/v1/users/profile
// 允许用户更新基本信息，如用户名、邮箱等
func (c *UserController) UpdateProfile(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	userID := ctx.GetUint("user_id")

	// 绑定更新请求
	var req types.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warnf("[%s] 更新用户资料请求参数无效: %v", requestID, err)
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

	c.logger.Infof("[%s] 开始更新用户 %d 的资料", requestID, userID)

	// 调用用户服务更新资料
	if err := c.userService.UpdateProfile(userID, &req); err != nil {
		c.handleServiceError(ctx, err, "更新用户资料失败")
		return
	}

	// 返回更新成功响应
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      gin.H{"message": "用户资料更新成功"},
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 用户 %d 资料更新成功", requestID, userID)
}

// ========================================
// 用户偏好设置接口
// ========================================

// GetPreferences 获取用户偏好设置
// GET /api/v1/users/preferences
// 返回用户的个性化配置，如默认滑点、Gas速度等
func (c *UserController) GetPreferences(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	userID := ctx.GetUint("user_id")

	c.logger.Debugf("[%s] 获取用户偏好设置: userID=%d", requestID, userID)

	// 调用用户服务获取偏好设置
	preferences, err := c.userService.GetPreferences(userID)
	if err != nil {
		c.handleServiceError(ctx, err, "获取用户偏好设置失败")
		return
	}

	// 返回偏好设置
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      preferences,
		Message:   "获取偏好设置成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 用户 %d 偏好设置获取成功", requestID, userID)
}

// UpdatePreferences 更新用户偏好设置
// PUT /api/v1/users/preferences
// 允许用户自定义交易相关的配置参数
func (c *UserController) UpdatePreferences(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	userID := ctx.GetUint("user_id")

	// 绑定偏好设置更新请求
	var req types.UserPreferences
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warnf("[%s] 更新偏好设置请求参数无效: %v", requestID, err)
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

	c.logger.Infof("[%s] 开始更新用户 %d 的偏好设置", requestID, userID)

	// 调用用户服务更新偏好设置
	if err := c.userService.UpdatePreferences(userID, &req); err != nil {
		c.handleServiceError(ctx, err, "更新偏好设置失败")
		return
	}

	// 返回更新成功响应
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      gin.H{"message": "偏好设置更新成功"},
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 用户 %d 偏好设置更新成功", requestID, userID)
}

// ResetPreferences 重置用户偏好设置
// POST /api/v1/users/preferences/reset
// 将用户偏好重置为系统默认值
func (c *UserController) ResetPreferences(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	userID := ctx.GetUint("user_id")

	c.logger.Infof("[%s] 开始重置用户 %d 的偏好设置", requestID, userID)

	// 调用用户服务重置偏好设置
	if err := c.userService.ResetPreferences(userID); err != nil {
		c.handleServiceError(ctx, err, "重置偏好设置失败")
		return
	}

	// 返回重置成功响应
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      gin.H{"message": "偏好设置已重置为默认值"},
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Infof("[%s] 用户 %d 偏好设置重置成功", requestID, userID)
}

// ========================================
// 用户统计接口
// ========================================

// GetStats 获取用户统计信息
// GET /api/v1/users/stats
// 返回用户的交易统计、资金统计等数据
func (c *UserController) GetStats(ctx *gin.Context) {
	requestID := ctx.GetString("request_id")
	userID := ctx.GetUint("user_id")

	c.logger.Debugf("[%s] 获取用户统计信息: userID=%d", requestID, userID)

	// 调用用户服务获取统计信息
	stats, err := c.userService.GetUserStats(userID)
	if err != nil {
		c.handleServiceError(ctx, err, "获取用户统计信息失败")
		return
	}

	// 返回统计信息
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      stats,
		Message:   "获取统计信息成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 用户 %d 统计信息获取成功", requestID, userID)
}

// GetUserList 获取用户列表（管理员功能）
// GET /api/v1/admin/users
// 分页返回用户列表，仅管理员可访问
func (c *UserController) GetUserList(ctx *gin.Context) {
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

	req := &types.PaginationRequest{
		Page:     page,
		PageSize: pageSize,
		SortBy:   sortBy,
		SortDesc: sortDesc,
	}

	c.logger.Debugf("[%s] 获取用户列表: page=%d, pageSize=%d", requestID, page, pageSize)

	// 调用用户服务获取列表
	users, meta, err := c.userService.GetUserList(req)
	if err != nil {
		c.handleServiceError(ctx, err, "获取用户列表失败")
		return
	}

	// 返回用户列表
	ctx.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      users,
		Meta:      meta,
		Message:   "获取用户列表成功",
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	c.logger.Debugf("[%s] 用户列表获取成功: total=%d", requestID, meta.Total)
}

// ========================================
// 辅助方法
// ========================================

// handleServiceError 处理业务服务错误
// 与AuthController中的方法相同，考虑提取为公共方法
func (c *UserController) handleServiceError(ctx *gin.Context, err error, defaultMessage string) {
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
