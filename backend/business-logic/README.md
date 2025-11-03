


## 用户认证服

### Web3认证系统

1、钱包签名认证
   // 核心认证流程
   POST /api/v1/auth/nonce        // 获取登录随机数
   POST /api/v1/auth/login        // 钱包签名登录
   POST /api/v1/auth/refresh      // 刷新JWT令牌
   POST /api/v1/auth/logout       // 用户登出

2、用户资料管理
   // 用户管理接口
   GET  /api/v1/users/profile     // 获取用户资料
   PUT  /api/v1/users/profile     // 更新用户资料
   GET  /api/v1/users/preferences // 获取偏好设置
   PUT  /api/v1/users/preferences // 更新偏好设置
   GET  /api/v1/users/stats       // 获取用户统计

当前项目结构

backend/business-logic/
├── cmd/
│   └── main.go                    # ✅ 完整的主程序
├── internal/
│   ├── controllers/
│   │   ├── controllers.go         # ✅ 控制器集合
│   │   ├── auth_controller.go     # ✅ 认证控制器实现
│   │   └── user_controller.go     # ✅ 用户控制器实现
│   ├── services/
│   │   ├── services.go           # ✅ 服务接口定义
│   │   ├── auth_service.go       # ✅ 认证服务实现
│   │   ├── user_service.go       # ✅ 用户服务实现
│   │   └── temp_implementations.go # ✅ 临时实现
│   ├── models/
│   │   └── models.go             # ✅ 完整GORM模型
│   ├── repository/
│   │   ├── repository.go         # ✅ Repository接口
│   │   ├── user_repository.go    # ✅ 用户Repository实现
│   │   └── implementations.go    # ✅ 其他Repository实现
│   └── types/
│       └── types.go              # ✅ 完整类型定义
├── pkg/
│   ├── config/
│   │   └── config.go             # ✅ 配置管理
│   ├── database/
│   │   └── database.go           # ✅ 数据库管理
│   ├── middleware/
│   │   └── middleware.go         # ✅ HTTP中间件
│   └── utils/
│       └── crypto.go             # ✅ 加密工具
└── 配置文件...                   # ✅ 完整配置

### 运行：
# 1. 启动服务
cd defi-aggregator/backend/business-logic
go run cmd/main.go

# 2. 测试认证接口
# 获取nonce
curl -X POST http://localhost:3000/api/v1/auth/nonce \
  -H "Content-Type: application/json" \
  -d '{"wallet_address": "0x742d35Cc6634C0532925a3b8D8A8CE8D3C8E8834A1"}'

# 健康检查
curl http://localhost:3000/health




## 代币管理系统

代币服务，包括多链代币管理、价格更新、搜索等功能：

1、代币信息管理

   // 代币查询接口
   GET  /api/v1/tokens              // 分页获取代币列表（支持筛选）
   GET  /api/v1/tokens/:id          // 获取代币详情
   GET  /api/v1/tokens/search?q=ETH // 搜索代币
   GET  /api/v1/tokens/popular      // 获取热门代币

2、区块链网络管理

   // 代币查询接口
   GET  /api/v1/tokens              // 分页获取代币列表（支持筛选）
   GET  /api/v1/tokens/:id          // 获取代币详情
   GET  /api/v1/tokens/search?q=ETH // 搜索代币
   GET  /api/v1/tokens/popular      // 获取热门代币

3、企业级特性

✅ 多链支持: 以太坊、Polygon、Arbitrum等
✅ 价格管理: 自动价格更新和历史记录
✅ 搜索筛选: 智能搜索和多维度筛选
✅ 缓存策略: 基于TTL的价格缓存
✅ 管理员功能: 代币验证、停用等管理操作


## 测试这些功能：

# 启动服务
cd defi-aggregator/backend/business-logic  
go run cmd/main.go

# 测试认证接口
curl -X POST http://localhost:3000/api/v1/auth/nonce \
  -H "Content-Type: application/json" \
  -d '{"wallet_address": "0x742d35Cc6634C0532925a3b8D8A8CE8D3C8E8834A1"}'

# 测试代币接口
curl http://localhost:3000/api/v1/tokens
curl http://localhost:3000/api/v1/tokens?page=1&page_size=10
curl "http://localhost:3000/api/v1/tokens/search?q=ETH"

# 测试区块链接口
curl http://localhost:3000/api/v1/chains
curl "http://localhost:3000/api/v1/chains?type=mainnet"