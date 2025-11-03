// Package config 聚合器配置管理器
// 优雅地管理聚合器配置：数据库基本信息 + 环境变量敏感信息
// 实现数据库驱动的动态配置，支持热更新和优雅降级
package config

import (
	"fmt"
	"strings"
	"time"

	"defi-aggregator/smart-router/internal/types"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// AggregatorConfigManager 聚合器配置管理器
// 负责从数据库和环境变量加载聚合器配置，确保数据一致性
type AggregatorConfigManager struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// DatabaseAggregator 数据库聚合器模型
type DatabaseAggregator struct {
	ID            uint    `gorm:"primaryKey"`
	Name          string  `gorm:"column:name"`
	DisplayName   string  `gorm:"column:display_name"`
	APIURL        string  `gorm:"column:api_url"`
	APIKey        string  `gorm:"column:api_key"`   // 通常为空，从环境变量读取
	IsActive      bool    `gorm:"column:is_active"` // 关键：控制聚合器是否启用
	Priority      int     `gorm:"column:priority"`
	TimeoutMS     int     `gorm:"column:timeout_ms"`
	RetryCount    int     `gorm:"column:retry_count"`
	SuccessRate   float64 `gorm:"column:success_rate"`
	AvgResponseMS int     `gorm:"column:avg_response_ms"`
}

func (DatabaseAggregator) TableName() string { return "aggregators" }

// DatabaseChain 数据库链模型
type DatabaseChain struct {
	ID       uint   `gorm:"primaryKey"`
	ChainID  uint   `gorm:"column:chain_id"`
	Name     string `gorm:"column:name"`
	IsActive bool   `gorm:"column:is_active"`
}

func (DatabaseChain) TableName() string { return "chains" }

// DatabaseAggregatorChain 聚合器支持链关系
type DatabaseAggregatorChain struct {
	ID            uint    `gorm:"primaryKey"`
	AggregatorID  uint    `gorm:"column:aggregator_id"`
	ChainID       uint    `gorm:"column:chain_id"`
	IsActive      bool    `gorm:"column:is_active"`
	GasMultiplier float64 `gorm:"column:gas_multiplier"`
}

func (DatabaseAggregatorChain) TableName() string { return "aggregator_chains" }

// NewAggregatorConfigManager 创建聚合器配置管理器
func NewAggregatorConfigManager(dbURL string, logger *logrus.Logger) (*AggregatorConfigManager, error) {
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: nil, // 使用默认日志
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return &AggregatorConfigManager{
		db:     db,
		logger: logger,
	}, nil
}

// LoadActiveProviders 加载活跃的聚合器配置
// 数据库控制启用状态，环境变量提供敏感信息
func (mgr *AggregatorConfigManager) LoadActiveProviders() ([]types.ProviderConfig, error) {
	mgr.logger.Info("🔄 从数据库加载活跃聚合器配置...")

	// 1. 查询数据库中启用的聚合器 (is_active = true)
	var dbAggregators []DatabaseAggregator
	if err := mgr.db.Where("is_active = ?", true).Order("priority ASC").Find(&dbAggregators).Error; err != nil {
		return nil, fmt.Errorf("查询活跃聚合器失败: %w", err)
	}

	mgr.logger.Infof("📋 数据库中找到 %d 个活跃聚合器", len(dbAggregators))

	var providers []types.ProviderConfig
	for i, dbAgg := range dbAggregators {
		// 创建数据库记录的副本，避免引用问题
		aggregator := DatabaseAggregator{
			ID:            dbAgg.ID,
			Name:          dbAgg.Name,
			DisplayName:   dbAgg.DisplayName,
			APIURL:        dbAgg.APIURL,
			APIKey:        dbAgg.APIKey,
			IsActive:      dbAgg.IsActive,
			Priority:      dbAgg.Priority,
			TimeoutMS:     dbAgg.TimeoutMS,
			RetryCount:    dbAgg.RetryCount,
			SuccessRate:   dbAgg.SuccessRate,
			AvgResponseMS: dbAgg.AvgResponseMS,
		}

		mgr.logger.Infof("📦 处理聚合器 %d/%d: ID=%d, Name=%s, DisplayName=%s, URL=%s",
			i+1, len(dbAggregators), aggregator.ID, aggregator.Name, aggregator.DisplayName, aggregator.APIURL)

		// 2. 查询支持的链（使用明确的ID）
		supportedChains, err := mgr.loadSupportedChains(aggregator.ID, aggregator.Name)
		if err != nil {
			mgr.logger.Warnf("⚠️ 跳过聚合器 %s (ID=%d): 加载支持链失败 - %v", aggregator.Name, aggregator.ID, err)
			continue
		}

		// 3. 从环境变量加载敏感配置
		envConfig := mgr.loadEnvironmentConfig(aggregator.Name)

		// 4. 合并数据库配置 + 环境变量配置（使用独立的变量）
		provider := types.ProviderConfig{
			Name:            aggregator.Name,                                                       // 数据库：确保使用正确的名称
			DisplayName:     aggregator.DisplayName,                                                // 数据库：确保使用正确的显示名
			BaseURL:         aggregator.APIURL,                                                     // 数据库：确保使用正确的URL
			APIKey:          mgr.selectAPIKey(aggregator.APIKey, envConfig.APIKey),                 // 优先环境变量
			Timeout:         mgr.selectTimeout(aggregator.TimeoutMS, envConfig.TimeoutMS),          // 优先环境变量
			RetryCount:      mgr.selectRetryCount(aggregator.RetryCount, envConfig.RetryCount),     // 优先环境变量
			Priority:        aggregator.Priority,                                                   // 数据库
			Weight:          mgr.calculateWeight(aggregator.SuccessRate, aggregator.AvgResponseMS), // 数据库计算
			IsActive:        aggregator.IsActive,                                                   // 数据库控制
			SupportedChains: append([]uint{}, supportedChains...),                                  // 深拷贝，避免slice引用问题
		}

		providers = append(providers, provider)

		mgr.logger.Infof("✅ 聚合器配置完成: ID=%d, %s", aggregator.ID, mgr.formatProviderSummary(provider))
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("没有找到可用的活跃聚合器")
	}

	mgr.logger.Infof("🎉 聚合器配置加载完成: %d 个活跃聚合器", len(providers))
	return providers, nil
}

// loadSupportedChains 加载聚合器支持的链
func (mgr *AggregatorConfigManager) loadSupportedChains(aggregatorID uint, aggregatorName string) ([]uint, error) {
	var chainRelations []DatabaseAggregatorChain
	if err := mgr.db.Where("aggregator_id = ? AND is_active = ?", aggregatorID, true).Find(&chainRelations).Error; err != nil {
		return nil, fmt.Errorf("查询聚合器链关系失败: %w", err)
	}

	if len(chainRelations) == 0 {
		return nil, fmt.Errorf("聚合器 %s 没有配置支持的链", aggregatorName)
	}

	// 获取链的外部ChainID
	var chainIDs []uint
	for _, relation := range chainRelations {
		chainIDs = append(chainIDs, relation.ChainID)
	}

	var chains []DatabaseChain
	if err := mgr.db.Where("id IN ? AND is_active = ?", chainIDs, true).Find(&chains).Error; err != nil {
		return nil, fmt.Errorf("查询链信息失败: %w", err)
	}

	var supportedChains []uint
	for _, chain := range chains {
		supportedChains = append(supportedChains, chain.ChainID) // 使用外部ChainID
	}

	mgr.logger.Debugf("📊 聚合器 %s 支持 %d 条链: %v", aggregatorName, len(supportedChains), supportedChains)
	return supportedChains, nil
}

// EnvironmentConfig 环境变量配置
type EnvironmentConfig struct {
	APIKey     string
	TimeoutMS  int
	RetryCount int
	Enabled    bool
}

// loadEnvironmentConfig 从环境变量加载聚合器配置
func (mgr *AggregatorConfigManager) loadEnvironmentConfig(aggregatorName string) EnvironmentConfig {
	// 根据聚合器名称确定环境变量前缀
	var envPrefix string
	switch aggregatorName {
	case "0x":
		envPrefix = "ZRX" // 0x Protocol使用ZRX前缀
	case "cowswap":
		envPrefix = "COW"
	case "1inch":
		envPrefix = "ONEINCH"
	case "paraswap":
		envPrefix = "PARASWAP"
	default:
		envPrefix = strings.ToUpper(aggregatorName)
	}

	config := EnvironmentConfig{
		APIKey:     getEnv(envPrefix+"_API_KEY", ""),
		TimeoutMS:  getEnvAsInt(envPrefix+"_TIMEOUT_MS", 0),
		RetryCount: getEnvAsInt(envPrefix+"_RETRY_COUNT", 0),
		Enabled:    getEnvAsBool(envPrefix+"_ENABLED", false),
	}

	mgr.logger.Debugf("🔧 环境变量配置 %s: APIKey=%s, Timeout=%dms, Retry=%d, Enabled=%t",
		aggregatorName,
		func() string {
			if config.APIKey != "" {
				return "已配置"
			}
			return "未配置"
		}(),
		config.TimeoutMS, config.RetryCount, config.Enabled)

	return config
}

// 配置选择器：优先使用环境变量，回退到数据库
func (mgr *AggregatorConfigManager) selectAPIKey(dbKey, envKey string) string {
	if envKey != "" {
		return envKey
	}
	return dbKey
}

func (mgr *AggregatorConfigManager) selectTimeout(dbTimeoutMS, envTimeoutMS int) time.Duration {
	if envTimeoutMS > 0 {
		return time.Duration(envTimeoutMS) * time.Millisecond
	}
	return time.Duration(dbTimeoutMS) * time.Millisecond
}

func (mgr *AggregatorConfigManager) selectRetryCount(dbRetry, envRetry int) int {
	if envRetry > 0 {
		return envRetry
	}
	return dbRetry
}

// calculateWeight 根据历史性能计算权重
func (mgr *AggregatorConfigManager) calculateWeight(successRate float64, avgResponseMS int) decimal.Decimal {
	baseWeight := decimal.NewFromFloat(1.0)
	successFactor := decimal.NewFromFloat(successRate)

	var timeFactor decimal.Decimal
	if avgResponseMS <= 500 {
		timeFactor = decimal.NewFromFloat(1.0)
	} else if avgResponseMS <= 1000 {
		timeFactor = decimal.NewFromFloat(0.9)
	} else if avgResponseMS <= 2000 {
		timeFactor = decimal.NewFromFloat(0.8)
	} else {
		timeFactor = decimal.NewFromFloat(0.7)
	}

	weight := baseWeight.Mul(successFactor.Mul(decimal.NewFromFloat(0.6))).
		Add(timeFactor.Mul(decimal.NewFromFloat(0.4)))

	if weight.LessThan(decimal.NewFromFloat(0.1)) {
		weight = decimal.NewFromFloat(0.1)
	}
	if weight.GreaterThan(decimal.NewFromFloat(1.0)) {
		weight = decimal.NewFromFloat(1.0)
	}

	return weight
}

// formatProviderSummary 格式化聚合器配置摘要
func (mgr *AggregatorConfigManager) formatProviderSummary(provider types.ProviderConfig) string {
	apiKeyStatus := "未配置"
	if provider.APIKey != "" {
		apiKeyStatus = "已配置"
	}

	return fmt.Sprintf("%s(%s) | URL: %s | API Key: %s | 支持链: %d条 | 权重: %.2f",
		provider.DisplayName, provider.Name, provider.BaseURL, apiKeyStatus,
		len(provider.SupportedChains), provider.Weight.InexactFloat64())
}

// Close 关闭数据库连接
func (mgr *AggregatorConfigManager) Close() error {
	if sqlDB, err := mgr.db.DB(); err == nil {
		return sqlDB.Close()
	}
	return nil
}
