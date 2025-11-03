// Package controllers 提供HTTP控制器
// 统一管理所有HTTP控制器实例，提供RESTful API接口
package controllers

import (
	"time"

	"defi-aggregator/business-logic/internal/services"
	"defi-aggregator/business-logic/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Controllers 控制器集合
type Controllers struct {
	Auth        *AuthController        // 认证控制器
	User        *UserController        // 用户控制器
	Token       *TokenController       // 代币控制器
	Chain       *ChainController       // 区块链控制器
	Quote       *QuoteController       // 报价控制器
	Swap        *SwapController        // 交易控制器
	Transaction *TransactionController // 交易历史控制器
	Stats       *StatsController       // 统计控制器
	Health      *HealthController      // 健康检查控制器
}

// New 创建控制器集合
// 初始化所有控制器实例，注入必要的服务依赖
func New(srvs *services.Services, cfg *config.Config, logger *logrus.Logger) *Controllers {
	return &Controllers{
		Auth:        NewAuthController(srvs.Auth, cfg, logger),
		User:        NewUserController(srvs.User, cfg, logger),
		Token:       NewTokenController(srvs.Token, srvs.Chain, cfg, logger),
		Chain:       NewChainController(srvs.Chain, cfg, logger),
		Quote:       NewQuoteController(srvs.Quote, cfg, logger),
		Swap:        &SwapController{},        // TODO: 实现
		Transaction: &TransactionController{}, // TODO: 实现
		Stats:       &StatsController{},       // TODO: 实现
		Health:      &HealthController{},      // TODO: 实现
	}
}

// 临时控制器结构体（待实现）
type SwapController struct{}
type TransactionController struct{}
type StatsController struct{}
type HealthController struct{}

// 临时方法（待实现）

func (c *SwapController) CreateSwap(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Swap controller not implemented yet"})
}
func (c *SwapController) GetSwapStatus(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Swap controller not implemented yet"})
}

func (c *TransactionController) GetTransactions(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Transaction controller not implemented yet"})
}
func (c *TransactionController) GetTransaction(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Transaction controller not implemented yet"})
}

func (c *StatsController) GetSystemStats(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Stats controller not implemented yet"})
}
func (c *StatsController) GetAggregatorStats(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Stats controller not implemented yet"})
}
func (c *StatsController) GetTokenPairStats(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Stats controller not implemented yet"})
}

func (c *HealthController) HealthCheck(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}
func (c *HealthController) Metrics(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Metrics not implemented yet"})
}
