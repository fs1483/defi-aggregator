// Package handlers API网关HTTP处理器
// 提供路由分发、代理转发、监控等功能
// 实现智能路由规则和服务发现
package handlers

import (
	"net/http"
	"strings"
	"time"

	"defi-aggregator/api-gateway/internal/proxy"
	"defi-aggregator/api-gateway/internal/types"
	"defi-aggregator/api-gateway/pkg/balancer"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GatewayHandler API网关处理器
// 核心的请求处理和路由分发器
type GatewayHandler struct {
	config   *types.Config         // 网关配置
	proxy    *proxy.ReverseProxy   // 反向代理
	balancer balancer.LoadBalancer // 负载均衡器
	logger   *logrus.Logger        // 日志记录器
}

// NewGatewayHandler 创建网关处理器实例
func NewGatewayHandler(config *types.Config, reverseProxy *proxy.ReverseProxy, lb balancer.LoadBalancer, logger *logrus.Logger) *GatewayHandler {
	return &GatewayHandler{
		config:   config,
		proxy:    reverseProxy,
		balancer: lb,
		logger:   logger,
	}
}

// ========================================
// 核心路由处理
// ========================================

// HandleRequest 处理所有API请求
// 智能路由分发器，根据路径规则转发到相应的后端服务
func (h *GatewayHandler) HandleRequest(c *gin.Context) {
	requestID := c.GetString("request_id")
	path := c.Request.URL.Path

	h.logger.Debugf("[%s] 网关路由处理: %s %s", requestID, c.Request.Method, path)

	// 1. 确定目标服务
	serviceName := h.determineTargetService(path)
	if serviceName == "" {
		h.logger.Warnf("[%s] 无法确定目标服务: path=%s", requestID, path)
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeNotFound,
				Message: "请求的API路径不存在",
				Details: map[string]interface{}{
					"path": path,
				},
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		})
		return
	}

	h.logger.Debugf("[%s] 路由到服务: %s", requestID, serviceName)

	// 2. 执行代理转发
	if err := h.proxy.ProxyRequest(c.Writer, c.Request, serviceName); err != nil {
		h.logger.Errorf("[%s] 代理请求失败: service=%s, error=%v", requestID, serviceName, err)

		// 如果还没有写入响应，返回错误
		if !c.Writer.Written() {
			c.JSON(http.StatusBadGateway, types.APIResponse{
				Success: false,
				Error: &types.APIError{
					Code:    types.ErrCodeBadGateway,
					Message: "后端服务不可用",
					Details: map[string]interface{}{
						"service": serviceName,
						"error":   err.Error(),
					},
				},
				Timestamp: time.Now().Unix(),
				RequestID: requestID,
			})
		}
		return
	}

	h.logger.Debugf("[%s] 代理请求成功: service=%s", requestID, serviceName)
}

// ========================================
// 路由决策逻辑
// ========================================

// determineTargetService 确定目标服务
// 根据请求路径和配置规则确定应该转发到哪个后端服务
func (h *GatewayHandler) determineTargetService(path string) string {
	// 标准化路径
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}

	// 特殊路由规则（按优先级排序）
	switch {
	// 网关管理接口（最高优先级）
	case strings.HasPrefix(path, "/api/v1/gateway/"):
		return "gateway" // 网关自身处理

	// 智能路由服务路径
	case strings.HasPrefix(path, "/api/v1/router/"):
		return types.ServiceSmartRouter

	// 业务逻辑服务路径（特定前缀）
	case strings.HasPrefix(path, "/api/v1/business/"):
		return types.ServiceBusinessLogic

	// 健康检查和监控接口
	case path == "/health", path == "/metrics":
		return "gateway" // 网关自身处理

	// 静态资源
	case strings.HasPrefix(path, "/static/"):
		return types.ServiceBusinessLogic

	// 根路径和常见文件
	case path == "/", path == "/favicon.ico", path == "/robots.txt":
		return types.ServiceBusinessLogic

	// 其他未匹配的路径（可能是老的API路径）
	case strings.HasPrefix(path, "/api/"):
		h.logger.Warnf("未知的API路径: %s，路由到业务逻辑服务", path)
		return types.ServiceBusinessLogic
	}

	// 默认路由到业务逻辑服务
	return types.ServiceBusinessLogic
}

// ========================================
// 监控和管理接口
// ========================================

// HealthCheck 网关健康检查
// GET /health
// 检查网关自身和所有后端服务的健康状态
func (h *GatewayHandler) HealthCheck(c *gin.Context) {
	requestID := c.GetString("request_id")

	h.logger.Debugf("[%s] 执行网关健康检查", requestID)

	// 检查后端服务健康状态
	serviceHealth := make(map[string]interface{})
	overallHealthy := true

	for _, service := range h.config.Routing.Services {
		targets := h.balancer.GetServiceHealth(service.Name)
		healthyCount := 0
		totalCount := len(targets)

		for _, target := range targets {
			if target.Active && target.Health.Healthy {
				healthyCount++
			}
		}

		serviceStatus := "healthy"
		if healthyCount == 0 {
			serviceStatus = "unhealthy"
			overallHealthy = false
		} else if healthyCount < totalCount {
			serviceStatus = "degraded"
		}

		serviceHealth[service.Name] = map[string]interface{}{
			"status":          serviceStatus,
			"healthy_count":   healthyCount,
			"total_count":     totalCount,
			"healthy_targets": healthyCount,
		}
	}

	// 确定整体状态
	status := types.ServiceStatusHealthy
	if !overallHealthy {
		status = types.ServiceStatusDegraded
	}

	// 返回健康检查结果
	healthResponse := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"services":  serviceHealth,
		"gateway": map[string]interface{}{
			"status": "healthy",
			"uptime": time.Since(time.Now()).String(), // TODO: 计算真实运行时间
		},
	}

	statusCode := http.StatusOK
	if !overallHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, healthResponse)
	h.logger.Debugf("[%s] 健康检查完成: status=%s", requestID, status)
}

// GetMetrics 获取网关指标
// GET /metrics
// 返回网关和后端服务的详细性能指标
func (h *GatewayHandler) GetMetrics(c *gin.Context) {
	requestID := c.GetString("request_id")

	h.logger.Debugf("[%s] 获取网关指标", requestID)

	// 获取代理统计
	proxyStats := h.proxy.GetStats()

	// 获取负载均衡器统计
	balancerStats := h.balancer.GetStats()

	// 构建指标响应
	metrics := map[string]interface{}{
		"proxy":         proxyStats,
		"load_balancer": balancerStats,
		"timestamp":     time.Now().Unix(),
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      metrics,
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	h.logger.Debugf("[%s] 指标获取完成", requestID)
}

// GetServiceStatus 获取服务状态
// GET /api/v1/gateway/services/status
// 返回所有后端服务的详细状态信息
func (h *GatewayHandler) GetServiceStatus(c *gin.Context) {
	requestID := c.GetString("request_id")

	h.logger.Debugf("[%s] 获取服务状态", requestID)

	serviceStatus := make(map[string]interface{})

	for _, service := range h.config.Routing.Services {
		targets := h.balancer.GetServiceHealth(service.Name)

		var targetStatus []map[string]interface{}
		for _, target := range targets {
			targetStatus = append(targetStatus, map[string]interface{}{
				"url":          target.URL.String(),
				"weight":       target.Weight,
				"active":       target.Active,
				"healthy":      target.Health.Healthy,
				"last_checked": target.Health.LastChecked,
				"error":        target.Health.Error,
			})
		}

		serviceStatus[service.Name] = map[string]interface{}{
			"targets":     targetStatus,
			"strategy":    service.Strategy,
			"timeout":     service.Timeout.String(),
			"retry_count": service.RetryCount,
		}
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success:   true,
		Data:      serviceStatus,
		Timestamp: time.Now().Unix(),
		RequestID: requestID,
	})

	h.logger.Debugf("[%s] 服务状态获取完成", requestID)
}
