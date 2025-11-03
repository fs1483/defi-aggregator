// Package balancer 负载均衡器实现
// 提供多种负载均衡算法，支持健康检查和故障转移
// 实现企业级负载均衡功能
package balancer

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"defi-aggregator/api-gateway/internal/types"

	"github.com/sirupsen/logrus"
)

// LoadBalancer 负载均衡器接口
// 定义负载均衡的标准接口
type LoadBalancer interface {
	// 核心功能
	SelectTarget(serviceName string) (*types.Target, error)         // 选择目标实例
	UpdateTargets(serviceName string, targets []types.Target) error // 更新目标列表

	// 健康检查
	StartHealthChecks() error // 启动健康检查
	StopHealthChecks()        // 停止健康检查

	// 统计信息
	GetStats() map[string]interface{}                   // 获取统计信息
	GetServiceHealth(serviceName string) []types.Target // 获取服务健康状态
}

// RoundRobinBalancer 轮询负载均衡器
// 实现轮询算法的负载均衡器
type RoundRobinBalancer struct {
	services      map[string]*ServicePool // 服务池映射
	config        *types.Config           // 网关配置
	logger        *logrus.Logger          // 日志记录器
	healthChecker *HealthChecker          // 健康检查器
	mutex         sync.RWMutex            // 读写锁
}

// ServicePool 服务池
// 管理单个服务的所有目标实例
type ServicePool struct {
	Name     string         `json:"name"`     // 服务名称
	Targets  []types.Target `json:"targets"`  // 目标实例列表
	Current  int            `json:"current"`  // 当前轮询位置
	Strategy string         `json:"strategy"` // 负载均衡策略
	mutex    sync.RWMutex   `json:"-"`        // 读写锁
}

// HealthChecker 健康检查器
type HealthChecker struct {
	balancer *RoundRobinBalancer
	stopChan chan struct{}
	logger   *logrus.Logger
}

// NewRoundRobinBalancer 创建轮询负载均衡器
// 初始化负载均衡器和服务池
func NewRoundRobinBalancer(config *types.Config, logger *logrus.Logger) LoadBalancer {
	balancer := &RoundRobinBalancer{
		services: make(map[string]*ServicePool),
		config:   config,
		logger:   logger,
	}

	// 初始化服务池
	balancer.initializeServicePools()

	// 创建健康检查器
	balancer.healthChecker = &HealthChecker{
		balancer: balancer,
		stopChan: make(chan struct{}),
		logger:   logger,
	}

	return balancer
}

// ========================================
// 核心负载均衡实现
// ========================================

// SelectTarget 选择目标实例
// 使用轮询算法选择健康的目标实例
// 参数:
//   - serviceName: 服务名称
//
// 返回:
//   - *types.Target: 选中的目标实例
//   - error: 选择失败的错误
func (b *RoundRobinBalancer) SelectTarget(serviceName string) (*types.Target, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// 获取服务池
	pool, exists := b.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("服务不存在: %s", serviceName)
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	// 获取健康的目标实例
	healthyTargets := b.getHealthyTargets(pool)
	if len(healthyTargets) == 0 {
		return nil, fmt.Errorf("服务 %s 没有健康的实例", serviceName)
	}

	// 根据策略选择目标
	var selectedTarget *types.Target
	switch pool.Strategy {
	case types.StrategyRoundRobin:
		selectedTarget = b.selectRoundRobin(pool, healthyTargets)
	case types.StrategyWeighted:
		selectedTarget = b.selectWeighted(healthyTargets)
	case types.StrategyRandom:
		selectedTarget = b.selectRandom(healthyTargets)
	default:
		selectedTarget = b.selectRoundRobin(pool, healthyTargets)
	}

	if selectedTarget == nil {
		return nil, fmt.Errorf("无法选择目标实例")
	}

	b.logger.Debugf("选择目标实例: service=%s, target=%s, strategy=%s",
		serviceName, selectedTarget.URL.String(), pool.Strategy)

	return selectedTarget, nil
}

// ========================================
// 负载均衡算法实现
// ========================================

// selectRoundRobin 轮询选择算法
func (b *RoundRobinBalancer) selectRoundRobin(pool *ServicePool, healthyTargets []*types.Target) *types.Target {
	if len(healthyTargets) == 0 {
		return nil
	}

	// 轮询选择
	target := healthyTargets[pool.Current%len(healthyTargets)]
	pool.Current++

	return target
}

// selectWeighted 加权选择算法
func (b *RoundRobinBalancer) selectWeighted(healthyTargets []*types.Target) *types.Target {
	if len(healthyTargets) == 0 {
		return nil
	}

	// 计算总权重
	totalWeight := 0
	for _, target := range healthyTargets {
		totalWeight += target.Weight
	}

	if totalWeight == 0 {
		// 如果所有权重都为0，使用轮询
		return healthyTargets[rand.Intn(len(healthyTargets))]
	}

	// 加权随机选择
	randomWeight := rand.Intn(totalWeight)
	currentWeight := 0

	for _, target := range healthyTargets {
		currentWeight += target.Weight
		if currentWeight > randomWeight {
			return target
		}
	}

	// 兜底返回第一个
	return healthyTargets[0]
}

// selectRandom 随机选择算法
func (b *RoundRobinBalancer) selectRandom(healthyTargets []*types.Target) *types.Target {
	if len(healthyTargets) == 0 {
		return nil
	}

	return healthyTargets[rand.Intn(len(healthyTargets))]
}

// ========================================
// 服务池管理
// ========================================

// initializeServicePools 初始化服务池
func (b *RoundRobinBalancer) initializeServicePools() {
	for _, service := range b.config.Routing.Services {
		pool := &ServicePool{
			Name:     service.Name,
			Targets:  service.Targets,
			Current:  0,
			Strategy: service.Strategy,
		}

		b.services[service.Name] = pool
		b.logger.Infof("初始化服务池: %s, targets=%d, strategy=%s",
			service.Name, len(service.Targets), service.Strategy)
	}
}

// UpdateTargets 更新目标列表
func (b *RoundRobinBalancer) UpdateTargets(serviceName string, targets []types.Target) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	pool, exists := b.services[serviceName]
	if !exists {
		return fmt.Errorf("服务不存在: %s", serviceName)
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	pool.Targets = targets
	pool.Current = 0 // 重置轮询位置

	b.logger.Infof("更新服务目标: service=%s, targets=%d", serviceName, len(targets))
	return nil
}

// getHealthyTargets 获取健康的目标实例
func (b *RoundRobinBalancer) getHealthyTargets(pool *ServicePool) []*types.Target {
	var healthyTargets []*types.Target

	for i := range pool.Targets {
		target := &pool.Targets[i]
		if target.Active && target.Health.Healthy {
			healthyTargets = append(healthyTargets, target)
		}
	}

	return healthyTargets
}

// ========================================
// 健康检查实现
// ========================================

// StartHealthChecks 启动健康检查
func (b *RoundRobinBalancer) StartHealthChecks() error {
	b.logger.Info("启动负载均衡器健康检查...")

	go b.healthChecker.start()

	b.logger.Info("健康检查已启动")
	return nil
}

// StopHealthChecks 停止健康检查
func (b *RoundRobinBalancer) StopHealthChecks() {
	b.logger.Info("停止负载均衡器健康检查...")

	close(b.healthChecker.stopChan)

	b.logger.Info("健康检查已停止")
}

// start 启动健康检查循环
func (hc *HealthChecker) start() {
	ticker := time.NewTicker(hc.balancer.config.LoadBalancer.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performHealthChecks()
		case <-hc.stopChan:
			return
		}
	}
}

// performHealthChecks 执行健康检查
func (hc *HealthChecker) performHealthChecks() {
	hc.balancer.mutex.RLock()
	services := make(map[string]*ServicePool)
	for name, pool := range hc.balancer.services {
		services[name] = pool
	}
	hc.balancer.mutex.RUnlock()

	// 并发检查所有服务
	var wg sync.WaitGroup
	for serviceName, pool := range services {
		wg.Add(1)
		go func(svcName string, p *ServicePool) {
			defer wg.Done()
			hc.checkServiceHealth(svcName, p)
		}(serviceName, pool)
	}

	wg.Wait()
}

// checkServiceHealth 检查单个服务的健康状态
func (hc *HealthChecker) checkServiceHealth(serviceName string, pool *ServicePool) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	for i := range pool.Targets {
		target := &pool.Targets[i]
		if !target.Active {
			continue
		}

		// 执行健康检查
		healthy := hc.isTargetHealthy(target)

		// 更新健康状态
		target.Health.Healthy = healthy
		target.Health.LastChecked = time.Now()

		if !healthy {
			target.Health.Error = "健康检查失败"
		} else {
			target.Health.Error = ""
		}

		hc.logger.Debugf("健康检查: service=%s, target=%s, healthy=%t",
			serviceName, target.URL.String(), healthy)
	}
}

// isTargetHealthy 检查目标实例是否健康
func (hc *HealthChecker) isTargetHealthy(target *types.Target) bool {
	// 构建健康检查URL
	healthURL := target.URL.ResolveReference(&url.URL{Path: "/health"})

	// 创建带超时的HTTP客户端
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 创建健康检查请求
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL.String(), nil)
	if err != nil {
		hc.logger.Debugf("创建健康检查请求失败: %v", err)
		return false
	}

	// 发送健康检查请求
	resp, err := client.Do(req)
	if err != nil {
		hc.logger.Debugf("健康检查请求失败: %v", err)
		return false
	}
	defer resp.Body.Close()

	// 检查响应状态
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// ========================================
// 统计信息
// ========================================

// GetStats 获取负载均衡器统计信息
func (b *RoundRobinBalancer) GetStats() map[string]interface{} {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	stats := make(map[string]interface{})

	for serviceName, pool := range b.services {
		pool.mutex.RLock()

		healthyCount := 0
		totalCount := len(pool.Targets)

		for _, target := range pool.Targets {
			if target.Active && target.Health.Healthy {
				healthyCount++
			}
		}

		stats[serviceName] = map[string]interface{}{
			"total_targets":   totalCount,
			"healthy_targets": healthyCount,
			"current_index":   pool.Current,
			"strategy":        pool.Strategy,
		}

		pool.mutex.RUnlock()
	}

	return stats
}

// GetServiceHealth 获取服务健康状态
func (b *RoundRobinBalancer) GetServiceHealth(serviceName string) []types.Target {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	pool, exists := b.services[serviceName]
	if !exists {
		return nil
	}

	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	// 返回目标列表的副本
	targets := make([]types.Target, len(pool.Targets))
	copy(targets, pool.Targets)

	return targets
}
