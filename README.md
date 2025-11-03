# DeFi聚合器 - 企业级Web3聚合交易平台

## 📋 项目概述

这是一个企业级的DeFi聚合器项目，采用微服务架构，为用户提供最优的代币交换价格。通过聚合多个去中心化交易所(DEX)的报价，帮助用户找到最佳交易路径。

### 🎯 项目特色

- **🏗️ 企业级微服务架构**: Go后端 + React前端
- **⚡ 高性能并发聚合**: 同时调用多个聚合器API
- **🔒 Web3原生安全**: 钱包签名认证，用户完全控制资金
- **📊 智能路由算法**: 基于价格、Gas费用、可靠性的最优选择
- **🚀 现代化技术栈**: Go 1.21 + React 18 + TypeScript + PostgreSQL + Redis

## 🏗️ 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   React前端     │    │   API网关       │    │  业务逻辑服务   │
│   :5175         │───▶│   :5176         │───▶│   :5177         │
│                 │    │                 │    │                 │
│ • 钱包集成      │    │ • 反向代理      │    │ • 用户管理      │
│ • 交易界面      │    │ • 负载均衡      │    │ • 代币管理      │
│ • 实时报价      │    │ • 安全防护      │    │ • 报价协调      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  智能路由服务   │    │   PostgreSQL    │    │     Redis       │
│   :5178         │◀───│   :5432         │    │   :6379         │
│                 │    │                 │    │                 │
│ • 并发聚合      │    │ • 用户数据      │    │ • 报价缓存      │
│ • 最优选择      │    │ • 交易记录      │    │ • 会话存储      │
│ • 缓存管理      │    │ • 统计分析      │    │ • 性能指标      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    第三方聚合器API                              │
│  1inch API  │  ParaSwap API  │  0x Protocol  │  CoW Protocol   │
└─────────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 前置要求

- Docker & Docker Compose
- Node.js 18+ (本地开发)
- Go 1.21+ (本地开发)

### 一键启动（推荐）

```bash
# 1. 克隆项目
git clone <repository-url>
cd defi-aggregator

# 2. 启动完整系统
docker-compose up -d

# 3. 检查服务状态
docker-compose ps

# 4. 访问应用
echo "🌐 前端应用: http://localhost:5175"
echo "🚪 API网关: http://localhost:5176" 
echo "📊 业务逻辑: http://localhost:5177"
echo "🧠 智能路由: http://localhost:5178"
```

### 本地开发模式

```bash
# 1. 启动基础服务（数据库）
docker-compose up -d postgres redis

# 2. 启动后端服务
# 终端1: 智能路由服务
cd backend/smart-router && go run cmd/main.go

# 终端2: 业务逻辑服务
cd backend/business-logic && go run cmd/main.go

# 终端3: API网关
cd backend/api-gateway && go run cmd/main.go

# 终端4: 前端应用
cd frontend && npm run dev
```

## 🔧 技术栈详情

### 后端服务

#### API网关 (Go + Gin)
- **端口**: 5176
- **功能**: 反向代理、负载均衡、安全防护、监控
- **特性**: 中间件栈、健康检查、限流、CORS

#### 业务逻辑服务 (Go + Gin + GORM)
- **端口**: 5177
- **功能**: 用户管理、代币管理、报价协调、数据持久化
- **特性**: JWT认证、Web3签名验证、RESTful API

#### 智能路由服务 (Go + Gin + Redis)
- **端口**: 5178
- **功能**: 并发聚合、最优选择、缓存管理、性能监控
- **特性**: Goroutine并发、智能算法、Redis缓存

### 前端应用

#### React应用 (React 18 + TypeScript + Vite)
- **端口**: 5175 (开发和生产)
- **功能**: 钱包连接、交易界面、实时报价、用户管理
- **特性**: Web3集成、响应式设计、类型安全

### 数据存储

#### PostgreSQL 15
- **端口**: 5432
- **用途**: 用户数据、交易记录、代币信息、统计分析
- **特性**: ACID事务、复杂查询、JSON支持

#### Redis 7.0
- **端口**: 6379
- **用途**: 报价缓存、会话存储、性能指标
- **特性**: 内存存储、TTL过期、高性能

## 📚 API文档

### 认证接口
```bash
POST /api/v1/auth/nonce     # 获取登录随机数
POST /api/v1/auth/login     # 钱包签名登录
POST /api/v1/auth/refresh   # 刷新JWT令牌
POST /api/v1/auth/logout    # 用户登出
```

### 代币接口
```bash
GET  /api/v1/tokens         # 获取代币列表
GET  /api/v1/tokens/:id     # 获取代币详情
GET  /api/v1/chains         # 获取区块链列表
```

### 报价接口
```bash
POST /api/v1/quotes         # 获取最优报价
GET  /api/v1/quotes/history # 获取报价历史
```

### 智能路由接口
```bash
POST /api/v1/router/quote   # 直接调用智能路由
GET  /api/v1/router/metrics # 路由性能指标
```

### 监控接口
```bash
GET  /health                # 健康检查
GET  /metrics               # 性能指标
```

## 🧪 测试指南

### API测试

```bash
# 1. 健康检查
curl http://localhost:5176/health

# 2. 获取代币列表
curl http://localhost:5176/api/v1/tokens

# 3. 获取区块链列表
curl http://localhost:5176/api/v1/chains

# 4. 测试智能路由（需要真实代币地址）
curl -X POST http://localhost:5176/api/v1/router/quote \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": "test_001",
    "from_token": "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
    "to_token": "0xA0b86a33E6441E0d1b7C8d9b3c4e5f6a7b8c9d0e1f",
    "amount_in": "1000000000000000000",
    "chain_id": 1,
    "slippage": "0.005"
  }'
```

### 前端测试

1. 访问 http://localhost:5175
2. 点击"连接钱包"按钮
3. 使用MetaMask连接
4. 测试API调用功能
5. 查看浏览器控制台获取详细日志

## 📊 性能指标

### 系统性能
- **报价聚合时间**: < 500ms (目标)
- **API响应时间**: < 100ms (目标)
- **缓存命中率**: > 80% (目标)
- **系统可用性**: 99.9% (目标)

### 支持规模
- **并发用户**: 1000+ (设计目标)
- **每秒请求**: 500+ QPS (设计目标)
- **支持代币**: 多种主流代币
- **支持链**: 4+ 主流区块链

## 🛠️ 开发指南

### 环境配置

1. **复制环境配置文件**
```bash
# 后端服务配置文件 (使用.env)
cp backend/business-logic/env.example backend/business-logic/.env
cp backend/smart-router/env.example backend/smart-router/.env
cp backend/api-gateway/env.example backend/api-gateway/.env

# 前端配置文件 (使用.env.local，优先级更高且不会被Git跟踪)
cp frontend/env.example frontend/.env.local
```

> **💡 为什么前端使用 `.env.local`？**
> - 🔒 **安全性**: `.env.local` 自动被Git忽略，避免泄露API密钥
> - 👤 **个人化**: 每个开发者可以有自己的配置（如不同的RPC端点）
> - 🔄 **覆盖性**: 优先级高于 `.env`，可以覆盖默认配置

2. **配置数据库连接**
```bash
# 修改 backend/business-logic/.env
DB_HOST=localhost
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=password
DB_NAME=defi_aggregator
```

3. **配置第三方API密钥**
```bash
# 修改 backend/smart-router/.env
ONEINCH_API_KEY=your-1inch-api-key
PARASWAP_API_KEY=your-paraswap-api-key
```

### 数据库初始化

```bash
# 1. 启动PostgreSQL
docker-compose up -d postgres

# 2. 执行数据库迁移
psql -h localhost -U admin -d defi_aggregator -f database/migrations/001_initial_schema.sql

# 3. 插入种子数据
psql -h localhost -U admin -d defi_aggregator -f database/seed_data.sql
```

## 📈 扩展计划

### 短期优化
- [ ] 完善前端交易功能
- [ ] 添加更多聚合器支持
- [ ] 实现交易历史和统计
- [ ] 优化缓存策略

### 长期规划
- [ ] 支持更多区块链网络
- [ ] 实现高级交易功能（限价单等）
- [ ] 添加流动性挖矿功能
- [ ] 集成DeFi协议（借贷、质押等）

## 🤝 贡献指南

### 代码规范
- Go代码遵循官方格式化标准
- TypeScript使用严格模式
- 提交信息使用约定式提交格式
- 所有API需要完整的文档注释

### 测试要求
- 单元测试覆盖率 > 80%
- 集成测试覆盖核心业务流程
- API接口需要完整的测试用例

## 📞 联系信息

- **项目作者**: [sfan]
- **邮箱**: [sfanonwork@gmail.com]
- **技术栈**: Go + React + PostgreSQL + Redis
- **开发时间**: 2024年

---

