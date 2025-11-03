// Package adapters 聚合器适配器接口定义
// 定义所有聚合器适配器的标准接口
package adapters

import (
	"context"

	"defi-aggregator/smart-router/internal/types"
)

// ProviderAdapter 聚合器适配器接口
// 定义所有第三方聚合器必须实现的标准接口
type ProviderAdapter interface {
	// 基础信息
	GetName() string               // 获取聚合器名称
	GetDisplayName() string        // 获取显示名称
	IsSupported(chainID uint) bool // 检查是否支持指定链

	// 核心功能
	GetQuote(ctx context.Context, req *types.QuoteRequest) (*types.ProviderQuote, error) // 获取报价
	HealthCheck(ctx context.Context) error                                               // 健康检查

	// 配置管理
	UpdateConfig(config *types.ProviderConfig) error // 更新配置
	GetConfig() *types.ProviderConfig                // 获取当前配置
}
