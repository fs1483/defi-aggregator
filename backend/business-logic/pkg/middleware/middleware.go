// Package middleware 提供HTTP中间件功能
// 包含认证、授权、日志、限流、CORS等企业级中间件
// 遵循Gin框架中间件开发规范，确保请求处理的安全性和可观测性
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"defi-aggregator/business-logic/internal/types"
	"defi-aggregator/business-logic/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Logger 请求日志中间件
// 记录HTTP请求的详细信息，包括方法、路径、状态码、响应时间等
// 支持结构化日志输出，便于日志分析和监控
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		startTime := time.Now()

		// 获取请求ID（如果存在）
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("request_id", requestID)
		}

		// 处理请求
		c.Next()

		// 计算处理时间
		latency := time.Since(startTime)

		// 获取响应信息
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		userAgent := c.Request.UserAgent()
		bodySize := c.Writer.Size()

		// 创建日志字段
		logFields := logrus.Fields{
			"request_id":  requestID,
			"method":      method,
			"path":        path,
			"status_code": statusCode,
			"latency_ms":  latency.Milliseconds(),
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"body_size":   bodySize,
		}

		// 添加用户信息（如果已认证）
		if userID, exists := c.Get("user_id"); exists {
			logFields["user_id"] = userID
		}

		// 根据状态码选择日志级别
		logMessage := "HTTP请求处理完成"
		switch {
		case statusCode >= 500:
			logger.WithFields(logFields).Error(logMessage)
		case statusCode >= 400:
			logger.WithFields(logFields).Warn(logMessage)
		default:
			logger.WithFields(logFields).Info(logMessage)
		}
	}
}

// Recovery 恐慌恢复中间件
// 捕获并处理panic，防止服务器崩溃
// 记录详细的错误信息，并返回统一的错误响应
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录panic详情
				requestID := c.GetString("request_id")
				logger.WithFields(logrus.Fields{
					"request_id": requestID,
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"panic":      err,
				}).Error("HTTP请求处理发生panic")

				// 返回统一错误响应
				c.JSON(http.StatusInternalServerError, types.APIResponse{
					Success: false,
					Error: &types.APIError{
						Code:    types.ErrCodeInternal,
						Message: "服务器内部错误",
					},
					Timestamp: time.Now().Unix(),
					RequestID: requestID,
				})

				// 中止请求处理
				c.Abort()
			}
		}()

		// 继续处理请求
		c.Next()
	}
}

// CORS 跨域资源共享中间件
// 根据配置设置CORS头，支持跨域请求
// 遵循CORS规范，确保安全的跨域访问
func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否为允许的源
		allowedOrigin := "*"
		for _, allowedOrig := range cfg.Security.CORSAllowedOrigins {
			if origin == allowedOrig {
				allowedOrigin = origin
				break
			}
		}

		// 设置CORS头
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.Security.CORSAllowedHeaders, ", "))
		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.Security.CORSAllowedMethods, ", "))
		c.Header("Access-Control-Max-Age", "86400") // 24小时

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestID 请求ID中间件
// 为每个请求生成唯一ID，便于日志追踪和问题定位
// 支持从请求头中获取现有ID或生成新ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取请求ID
		requestID := c.GetHeader(types.HeaderRequestID)

		// 如果没有请求ID，生成新的
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 设置请求ID到上下文和响应头
		c.Set("request_id", requestID)
		c.Header(types.HeaderRequestID, requestID)

		c.Next()
	}
}

// RateLimit 限流中间件
// 基于IP地址进行简单的限流控制
// 生产环境建议使用Redis等外部存储实现分布式限流
func RateLimit(cfg *config.Config) gin.HandlerFunc {
	// 简单的内存限流实现
	// 生产环境应该使用Redis等外部存储
	var limiter = make(map[string][]time.Time)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// 清理过期记录
		if records, exists := limiter[clientIP]; exists {
			var validRecords []time.Time
			cutoff := now.Add(-cfg.Security.RateLimitDuration)

			for _, record := range records {
				if record.After(cutoff) {
					validRecords = append(validRecords, record)
				}
			}
			limiter[clientIP] = validRecords
		}

		// 检查是否超出限制
		if len(limiter[clientIP]) >= cfg.Security.RateLimitRequests {
			c.JSON(http.StatusTooManyRequests, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeRateLimit,
					Message: "请求频率过高，请稍后再试",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// 记录本次请求
		limiter[clientIP] = append(limiter[clientIP], now)

		c.Next()
	}
}

// Security 安全头中间件
// 设置安全相关的HTTP头，提高应用安全性
// 包含XSS保护、内容类型嗅探保护等
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防XSS攻击
		c.Header("X-XSS-Protection", "1; mode=block")

		// 防止内容类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// 防止页面被iframe嵌入
		c.Header("X-Frame-Options", "DENY")

		// 强制HTTPS（在生产环境中启用）
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 内容安全策略（根据需要配置）
		// c.Header("Content-Security-Policy", "default-src 'self'")

		// 推荐代理不缓存敏感内容
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Next()
	}
}

// JWT JWT认证中间件
// 验证JWT令牌，提取用户信息
// 支持Bearer Token格式，验证令牌有效性和过期时间
func JWT(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "缺少认证令牌",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// 检查Bearer格式
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "无效的认证令牌格式",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// 解析JWT令牌
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.SecretKey), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "无效的认证令牌",
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// 验证令牌有效性
		if !token.Valid {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "认证令牌已失效",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// 提取用户信息
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "无效的令牌声明",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// 验证必要的声明
		userIDClaim, exists := claims[types.JWTClaimUserID]
		if !exists {
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "令牌缺少用户信息",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// 转换用户ID类型
		var userID uint
		switch v := userIDClaim.(type) {
		case float64:
			userID = uint(v)
		case string:
			if id, err := strconv.ParseUint(v, 10, 32); err == nil {
				userID = uint(id)
			} else {
				c.JSON(http.StatusUnauthorized, types.APIResponse{
					Success: false,
					Error: &types.APIError{
						Code:    types.ErrCodeUnauthorized,
						Message: "无效的用户ID格式",
					},
					Timestamp: time.Now().Unix(),
					RequestID: c.GetString("request_id"),
				})
				c.Abort()
				return
			}
		default:
			c.JSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "无效的用户ID类型",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// 将用户信息设置到上下文
		c.Set("user_id", userID)
		if walletAddr, exists := claims[types.JWTClaimWalletAddr]; exists {
			c.Set("wallet_address", walletAddr)
		}
		if role, exists := claims[types.JWTClaimRole]; exists {
			c.Set("user_role", role)
		}

		c.Next()
	}
}

// Optional JWT 可选JWT认证中间件
// 如果提供了JWT令牌则验证，否则继续处理
// 适用于既支持认证用户又支持匿名用户的接口
func OptionalJWT(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// 如果没有认证头，继续处理（作为匿名用户）
		if authHeader == "" {
			c.Next()
			return
		}

		// 如果有认证头，则进行验证
		jwtMiddleware := JWT(cfg)
		jwtMiddleware(c)
	}
}

// Admin 管理员权限中间件
// 验证用户是否具有管理员权限
// 必须在JWT中间件之后使用
func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查用户角色
		role, exists := c.Get("user_role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeForbidden,
					Message: "需要管理员权限",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Timeout 请求超时中间件
// 为请求设置超时时间，防止长时间占用资源
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 替换请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 创建完成通道
		finished := make(chan struct{})

		go func() {
			defer close(finished)
			c.Next()
		}()

		select {
		case <-finished:
			// 请求正常完成
			return
		case <-ctx.Done():
			// 请求超时
			c.JSON(http.StatusRequestTimeout, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    "REQUEST_TIMEOUT",
					Message: "请求处理超时",
				},
				Timestamp: time.Now().Unix(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
		}
	}
}
