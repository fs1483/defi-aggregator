// Package middleware API网关中间件
// 提供认证、限流、CORS、日志等企业级中间件功能
// 实现统一的安全策略和请求处理流水线
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"defi-aggregator/api-gateway/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// ========================================
// 请求日志中间件
// ========================================

// RequestLogger 请求日志中间件
// 记录所有通过网关的请求详细信息
func RequestLogger(logger *logrus.Logger, config *types.MonitoringConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// 获取或生成请求ID
		requestID := c.GetHeader(types.HeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header(types.HeaderRequestID, requestID)
		}
		c.Set("request_id", requestID)

		// 记录请求开始
		if config.LogRequests {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"query":      c.Request.URL.RawQuery,
				"client_ip":  c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			}).Info("API网关请求开始")
		}

		// 处理请求
		c.Next()

		// 计算处理时间
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		// 记录请求完成
		logLevel := logrus.InfoLevel
		if statusCode >= 400 {
			logLevel = logrus.WarnLevel
		}
		if statusCode >= 500 {
			logLevel = logrus.ErrorLevel
		}

		logger.WithFields(logrus.Fields{
			"request_id":    requestID,
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"status_code":   statusCode,
			"duration_ms":   duration.Milliseconds(),
			"client_ip":     c.ClientIP(),
			"response_size": c.Writer.Size(),
		}).Log(logLevel, "API网关请求完成")

		// 记录慢请求
		if duration.Milliseconds() > int64(config.SlowRequestMs) {
			logger.WithFields(logrus.Fields{
				"request_id":  requestID,
				"duration_ms": duration.Milliseconds(),
				"threshold":   config.SlowRequestMs,
			}).Warn("检测到慢请求")
		}
	}
}

// ========================================
// CORS中间件
// ========================================

// CORS 跨域资源共享中间件
// 根据配置处理CORS请求
func CORS(config *types.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否为允许的源
		allowedOrigin := ""
		for _, allowedOrig := range config.AllowedOrigins {
			if allowedOrig == "*" || allowedOrig == origin {
				allowedOrigin = allowedOrig
				break
			}
		}

		if allowedOrigin == "" && len(config.AllowedOrigins) > 0 && origin != "" {
			// 只有当有Origin头且不在允许列表中时才拒绝
			c.AbortWithStatusJSON(http.StatusForbidden, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeForbidden,
					Message: "跨域请求被拒绝",
				},
				Timestamp: time.Now().Unix(),
			})
			return
		}

		// 设置CORS头
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
		} else if len(config.AllowedOrigins) > 0 {
			// 如果有配置但不匹配，设置第一个允许的源
			c.Header("Access-Control-Allow-Origin", config.AllowedOrigins[0])
		}

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
		c.Header("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ========================================
// JWT认证中间件
// ========================================

// JWTAuth JWT认证中间件
// 验证JWT令牌的有效性
func JWTAuth(config *types.JWTConfig, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		// 获取Authorization头
		authHeader := c.GetHeader(types.HeaderAuthorization)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "缺少认证令牌",
				},
				Timestamp: time.Now().Unix(),
				RequestID: requestID,
			})
			return
		}

		// 检查Bearer格式
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "无效的认证令牌格式",
				},
				Timestamp: time.Now().Unix(),
				RequestID: requestID,
			})
			return
		}

		tokenString := tokenParts[1]

		// 解析JWT令牌
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.SecretKey), nil
		})

		if err != nil || !token.Valid {
			logger.Warnf("[%s] JWT令牌验证失败: %v", requestID, err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeUnauthorized,
					Message: "认证令牌无效",
				},
				Timestamp: time.Now().Unix(),
				RequestID: requestID,
			})
			return
		}

		// 提取用户信息
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, exists := claims["user_id"]; exists {
				c.Set("user_id", userID)
			}
			if walletAddr, exists := claims["wallet_address"]; exists {
				c.Set("wallet_address", walletAddr)
			}
		}

		c.Next()
	}
}

// ========================================
// 限流中间件
// ========================================

// RateLimiter 限流中间件
// 实现基于IP和用户的限流控制
type RateLimiter struct {
	config   *types.RateLimitConfig
	limiters map[string]*rate.Limiter
	mutex    sync.RWMutex
	logger   *logrus.Logger
}

// NewRateLimiter 创建限流中间件
func NewRateLimiter(config *types.RateLimitConfig, logger *logrus.Logger) *RateLimiter {
	return &RateLimiter{
		config:   config,
		limiters: make(map[string]*rate.Limiter),
		logger:   logger,
	}
}

// RateLimit 限流中间件函数
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.config.Enabled {
			c.Next()
			return
		}

		requestID := c.GetString("request_id")
		clientIP := c.ClientIP()

		// 检查IP限流
		if !rl.checkIPRateLimit(clientIP) {
			rl.logger.Warnf("[%s] IP限流触发: %s", requestID, clientIP)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeRateLimitExceeded,
					Message: "请求频率过高，请稍后再试",
				},
				Timestamp: time.Now().Unix(),
				RequestID: requestID,
			})
			return
		}

		c.Next()
	}
}

// checkIPRateLimit 检查IP限流
func (rl *RateLimiter) checkIPRateLimit(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		// 创建新的限流器
		limiter = rate.NewLimiter(
			rate.Every(rl.config.PerIPRate.Duration/time.Duration(rl.config.PerIPRate.Requests)),
			rl.config.PerIPRate.Requests,
		)
		rl.limiters[ip] = limiter
	}

	return limiter.Allow()
}

// ========================================
// 安全中间件
// ========================================

// Security 安全头中间件
// 设置安全相关的HTTP头
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防XSS攻击
		c.Header("X-XSS-Protection", "1; mode=block")

		// 防止内容类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// 防止页面被iframe嵌入
		c.Header("X-Frame-Options", "DENY")

		// API网关标识
		c.Header("X-Gateway", "DeFi-Aggregator-Gateway/1.0")

		// 推荐不缓存敏感API响应
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// ========================================
// 请求ID中间件
// ========================================

// RequestID 请求ID中间件
// 为每个请求生成或传递唯一ID，便于链路追踪
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

// ========================================
// 恢复中间件
// ========================================

// Recovery 恐慌恢复中间件
// 捕获panic并返回统一的错误响应
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := c.GetString("request_id")

				logger.WithFields(logrus.Fields{
					"request_id": requestID,
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"panic":      err,
				}).Error("API网关发生panic")

				c.JSON(http.StatusInternalServerError, types.APIResponse{
					Success: false,
					Error: &types.APIError{
						Code:    types.ErrCodeInternalError,
						Message: "网关内部错误",
					},
					Timestamp: time.Now().Unix(),
					RequestID: requestID,
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
