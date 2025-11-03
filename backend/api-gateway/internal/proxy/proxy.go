// Package proxy 反向代理核心实现
// 提供高性能的反向代理功能，支持负载均衡、健康检查、熔断等
// 实现企业级API网关的核心代理逻辑
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"defi-aggregator/api-gateway/internal/types"
	"defi-aggregator/api-gateway/pkg/balancer"

	"github.com/sirupsen/logrus"
)

// ReverseProxy 反向代理管理器
// 封装反向代理功能，提供统一的代理接口
type ReverseProxy struct {
	config   *types.Config                     // 网关配置
	balancer balancer.LoadBalancer             // 负载均衡器
	logger   *logrus.Logger                    // 日志记录器
	proxies  map[string]*httputil.ReverseProxy // 服务代理映射
	stats    *ProxyStats                       // 代理统计
}

// ProxyStats 代理统计信息
type ProxyStats struct {
	TotalRequests   int64                   `json:"total_requests"`
	SuccessRequests int64                   `json:"success_requests"`
	FailedRequests  int64                   `json:"failed_requests"`
	ServiceStats    map[string]*ServiceStat `json:"service_stats"`
	StartTime       time.Time               `json:"start_time"`
}

// ServiceStat 单个服务的统计信息
type ServiceStat struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessRequests int64         `json:"success_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	LastRequestTime time.Time     `json:"last_request_time"`
}

// NewReverseProxy 创建反向代理实例
// 初始化负载均衡器和服务代理
func NewReverseProxy(config *types.Config, lb balancer.LoadBalancer, logger *logrus.Logger) *ReverseProxy {
	proxy := &ReverseProxy{
		config:   config,
		balancer: lb,
		logger:   logger,
		proxies:  make(map[string]*httputil.ReverseProxy),
		stats: &ProxyStats{
			ServiceStats: make(map[string]*ServiceStat),
			StartTime:    time.Now(),
		},
	}

	// 初始化各服务的代理
	proxy.initializeProxies()

	return proxy
}

// ========================================
// 核心代理功能
// ========================================

// ProxyRequest 代理HTTP请求
// 根据路由规则将请求转发到适当的后端服务
// 参数:
//   - w: HTTP响应写入器
//   - r: HTTP请求
//   - serviceName: 目标服务名称
//
// 返回:
//   - error: 代理过程中的错误
func (p *ReverseProxy) ProxyRequest(w http.ResponseWriter, r *http.Request, serviceName string) error {
	startTime := time.Now()
	requestID := r.Header.Get(types.HeaderRequestID)

	p.logger.Debugf("[%s] 开始代理请求: service=%s, path=%s", requestID, serviceName, r.URL.Path)

	// 1. 选择目标实例
	target, err := p.balancer.SelectTarget(serviceName)
	if err != nil {
		p.updateStats(serviceName, false, time.Since(startTime))
		return fmt.Errorf("选择目标实例失败: %w", err)
	}

	// 2. 获取或创建服务代理
	proxy, err := p.getOrCreateProxy(serviceName, target)
	if err != nil {
		p.updateStats(serviceName, false, time.Since(startTime))
		return fmt.Errorf("获取服务代理失败: %w", err)
	}

	// 3. 设置代理请求头
	p.setupProxyHeaders(r, target)

	// 4. 执行代理请求
	responseWriter := &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	// 执行代理
	proxy.ServeHTTP(responseWriter, r)

	// 5. 更新统计信息
	success := responseWriter.statusCode < 400
	p.updateStats(serviceName, success, time.Since(startTime))

	p.logger.Debugf("[%s] 代理请求完成: service=%s, target=%s, status=%d, duration=%v",
		requestID, serviceName, target.URL.String(), responseWriter.statusCode, time.Since(startTime))

	return nil
}

// ========================================
// 代理初始化和管理
// ========================================

// initializeProxies 初始化各服务的代理
func (p *ReverseProxy) initializeProxies() {
	for _, service := range p.config.Routing.Services {
		// 为每个服务初始化统计
		p.stats.ServiceStats[service.Name] = &ServiceStat{}

		p.logger.Infof("初始化服务代理: %s, targets=%d", service.Name, len(service.Targets))
	}
}

// getOrCreateProxy 获取或创建服务代理
func (p *ReverseProxy) getOrCreateProxy(serviceName string, target *types.Target) (*httputil.ReverseProxy, error) {
	// 使用目标URL作为键，支持多实例
	proxyKey := fmt.Sprintf("%s_%s", serviceName, target.URL.String())

	if proxy, exists := p.proxies[proxyKey]; exists {
		return proxy, nil
	}

	// 创建新的反向代理
	proxy := httputil.NewSingleHostReverseProxy(target.URL)

	// 自定义Director函数
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// 移除API前缀（如果需要）
		p.rewritePath(req, serviceName)

		// 设置Host头
		req.Host = target.URL.Host

		// 添加代理标识头
		req.Header.Set("X-Forwarded-By", "defi-aggregator-gateway")
	}

	// 自定义错误处理
	proxy.ErrorHandler = p.createErrorHandler(serviceName, target)

	// 缓存代理实例
	p.proxies[proxyKey] = proxy

	p.logger.Debugf("创建新代理: service=%s, target=%s", serviceName, target.URL.String())
	return proxy, nil
}

// setupProxyHeaders 设置代理请求头
// 添加必要的代理头信息，保持请求链路的完整性
func (p *ReverseProxy) setupProxyHeaders(r *http.Request, target *types.Target) {
	// 设置X-Forwarded-*头
	if clientIP := p.getClientIP(r); clientIP != "" {
		r.Header.Set(types.HeaderForwardedFor, clientIP)
		r.Header.Set(types.HeaderRealIP, clientIP)
	}

	r.Header.Set(types.HeaderForwardedHost, r.Host)
	r.Header.Set(types.HeaderForwardedProto, "http") // TODO: 根据实际协议设置

	// 确保请求ID存在
	if r.Header.Get(types.HeaderRequestID) == "" {
		r.Header.Set(types.HeaderRequestID, generateRequestID())
	}
}

// rewritePath 重写请求路径
// 根据服务配置调整请求路径，适配后端服务的路由结构
func (p *ReverseProxy) rewritePath(req *http.Request, serviceName string) {
	originalPath := req.URL.Path

	switch serviceName {
	case types.ServiceSmartRouter:
		// 智能路由服务：移除 /api/v1/router 前缀
		if strings.HasPrefix(originalPath, "/api/v1/router") {
			req.URL.Path = strings.TrimPrefix(originalPath, "/api/v1/router")
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
	case types.ServiceBusinessLogic:
		// 业务逻辑服务：保持原路径
		// 不需要修改，直接透传
	}

	if originalPath != req.URL.Path {
		p.logger.Debugf("路径重写: %s -> %s (service: %s)", originalPath, req.URL.Path, serviceName)
	}
}

// createErrorHandler 创建错误处理器
func (p *ReverseProxy) createErrorHandler(serviceName string, target *types.Target) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		requestID := r.Header.Get(types.HeaderRequestID)

		p.logger.Errorf("[%s] 代理请求失败: service=%s, target=%s, error=%v",
			requestID, serviceName, target.URL.String(), err)

		// 标记目标为不健康
		target.Health.Healthy = false
		target.Health.Error = err.Error()
		target.Health.LastChecked = time.Now()

		// 返回统一的错误响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)

		errorResponse := types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    types.ErrCodeBadGateway,
				Message: "后端服务不可用",
				Details: map[string]interface{}{
					"service": serviceName,
					"target":  target.URL.String(),
				},
			},
			Timestamp: time.Now().Unix(),
			RequestID: requestID,
		}

		if jsonData, err := json.Marshal(errorResponse); err == nil {
			w.Write(jsonData)
		}
	}
}

// ========================================
// 辅助功能
// ========================================

// getClientIP 获取客户端真实IP
func (p *ReverseProxy) getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 检查X-Real-IP头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

// updateStats 更新代理统计信息
func (p *ReverseProxy) updateStats(serviceName string, success bool, duration time.Duration) {
	p.stats.TotalRequests++

	if success {
		p.stats.SuccessRequests++
	} else {
		p.stats.FailedRequests++
	}

	// 更新服务统计
	if serviceStat, exists := p.stats.ServiceStats[serviceName]; exists {
		serviceStat.TotalRequests++
		serviceStat.LastRequestTime = time.Now()

		if success {
			serviceStat.SuccessRequests++
		} else {
			serviceStat.FailedRequests++
		}

		// 更新平均响应时间（简化计算）
		if serviceStat.TotalRequests == 1 {
			serviceStat.AvgResponseTime = duration
		} else {
			alpha := 0.1
			serviceStat.AvgResponseTime = time.Duration(
				float64(serviceStat.AvgResponseTime)*(1-alpha) + float64(duration)*alpha,
			)
		}
	}
}

// GetStats 获取代理统计信息
func (p *ReverseProxy) GetStats() *ProxyStats {
	return p.stats
}

// ========================================
// 响应包装器
// ========================================

// responseWriterWrapper 响应写入器包装器
// 用于捕获响应状态码和其他信息
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

// ========================================
// 工具函数
// ========================================

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("gw_%d", time.Now().UnixNano())
}
