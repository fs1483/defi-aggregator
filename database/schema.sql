-- DeFi聚合器数据库表结构设计
-- 基于PostgreSQL 15+ 
-- 设计原则：支持完整业务流程，考虑性能和扩展性

-- ========================================
-- 1. 用户管理相关表
-- ========================================

-- 用户基础信息表
CREATE TABLE users (
    id                  SERIAL PRIMARY KEY,
    wallet_address      VARCHAR(66) UNIQUE NOT NULL,        -- 钱包地址 (支持所有区块链地址格式)
    nonce              VARCHAR(64) NOT NULL DEFAULT '',     -- 登录随机数 (用于钱包签名验证)
    username           VARCHAR(50),                         -- 可选用户名
    email              VARCHAR(255),                        -- 可选邮箱
    avatar_url         VARCHAR(500),                        -- 头像URL
    preferred_language VARCHAR(10) DEFAULT 'en',            -- 首选语言
    timezone           VARCHAR(50) DEFAULT 'UTC',           -- 时区
    is_active          BOOLEAN DEFAULT true,                -- 账户状态
    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at      TIMESTAMP                            -- 最后登录时间
);

-- 用户偏好设置表
CREATE TABLE user_preferences (
    id                    SERIAL PRIMARY KEY,
    user_id              INTEGER REFERENCES users(id) ON DELETE CASCADE,
    default_slippage     DECIMAL(5,4) DEFAULT 0.005,       -- 默认滑点 0.5%
    preferred_gas_speed  VARCHAR(20) DEFAULT 'standard',    -- fast/standard/slow
    auto_approve_tokens  BOOLEAN DEFAULT false,             -- 是否自动批准代币授权
    show_test_tokens     BOOLEAN DEFAULT false,             -- 是否显示测试代币
    notification_email   BOOLEAN DEFAULT true,              -- 邮件通知
    notification_browser BOOLEAN DEFAULT true,              -- 浏览器通知
    privacy_analytics    BOOLEAN DEFAULT true,              -- 是否允许分析
    created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(user_id)
);

-- ========================================
-- 2. 代币和链信息管理
-- ========================================

-- 支持的区块链网络
CREATE TABLE chains (
    id              SERIAL PRIMARY KEY,
    chain_id        INTEGER UNIQUE NOT NULL,               -- 链ID (1=Ethereum, 137=Polygon等)
    name            VARCHAR(50) NOT NULL,                  -- 链名称
    display_name    VARCHAR(50) NOT NULL,                  -- 显示名称
    symbol          VARCHAR(10) NOT NULL,                  -- 原生代币符号 (ETH, MATIC等)
    rpc_url         VARCHAR(500) NOT NULL,                 -- RPC节点URL
    explorer_url    VARCHAR(500) NOT NULL,                 -- 区块浏览器URL
    is_testnet      BOOLEAN DEFAULT false,                 -- 是否为测试网
    is_active       BOOLEAN DEFAULT true,                  -- 是否启用
    gas_price_gwei  INTEGER DEFAULT 20,                    -- 建议Gas价格
    block_time_sec  INTEGER DEFAULT 15,                    -- 平均出块时间
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 支持的代币信息
CREATE TABLE tokens (
    id              SERIAL PRIMARY KEY,
    chain_id        INTEGER REFERENCES chains(id),
    contract_address VARCHAR(66) NOT NULL,                 -- 代币合约地址 (支持所有区块链地址格式)
    symbol          VARCHAR(20) NOT NULL,                  -- 代币符号
    name            VARCHAR(100) NOT NULL,                 -- 代币全名
    decimals        INTEGER NOT NULL,                      -- 小数位数
    logo_url        VARCHAR(500),                          -- 代币图标URL
    coingecko_id    VARCHAR(100),                          -- CoinGecko ID
    coinmarketcap_id INTEGER,                              -- CoinMarketCap ID
    is_native       BOOLEAN DEFAULT false,                 -- 是否为原生代币
    is_stable       BOOLEAN DEFAULT false,                 -- 是否为稳定币
    is_verified     BOOLEAN DEFAULT false,                 -- 是否已验证
    is_active       BOOLEAN DEFAULT true,                  -- 是否启用
    daily_volume_usd DECIMAL(20,2),                       -- 24小时交易量USD
    market_cap_usd  DECIMAL(20,2),                        -- 市值USD
    price_usd       DECIMAL(20,8),                        -- 当前价格USD
    price_updated_at TIMESTAMP,                           -- 价格更新时间
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(chain_id, contract_address)
);

-- ========================================
-- 3. 聚合器和报价相关表
-- ========================================

-- 第三方聚合器配置
CREATE TABLE aggregators (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(50) UNIQUE NOT NULL,           -- 聚合器名称 (1inch, paraswap等)
    display_name    VARCHAR(100) NOT NULL,                 -- 显示名称
    api_url         VARCHAR(500) NOT NULL,                 -- API基础URL
    api_key         VARCHAR(255),                          -- API密钥 (加密存储)
    logo_url        VARCHAR(500),                          -- Logo URL
    is_active       BOOLEAN DEFAULT true,                  -- 是否启用
    priority        INTEGER DEFAULT 1,                     -- 优先级 (1最高)
    timeout_ms      INTEGER DEFAULT 5000,                  -- 超时时间毫秒
    retry_count     INTEGER DEFAULT 3,                     -- 重试次数
    success_rate    DECIMAL(5,4) DEFAULT 1.0000,          -- 成功率统计
    avg_response_ms INTEGER DEFAULT 1000,                  -- 平均响应时间
    last_health_check TIMESTAMP,                          -- 最后健康检查时间
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 支持的聚合器和链的关系
CREATE TABLE aggregator_chains (
    id              SERIAL PRIMARY KEY,
    aggregator_id   INTEGER REFERENCES aggregators(id) ON DELETE CASCADE,
    chain_id        INTEGER REFERENCES chains(id) ON DELETE CASCADE,
    is_active       BOOLEAN DEFAULT true,
    gas_multiplier  DECIMAL(3,2) DEFAULT 1.0,              -- Gas费用乘数
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(aggregator_id, chain_id)
);

-- 报价请求记录表
CREATE TABLE quote_requests (
    id                  SERIAL PRIMARY KEY,
    request_id          VARCHAR(64) UNIQUE NOT NULL,       -- 唯一请求ID
    user_id             INTEGER REFERENCES users(id),      -- 用户ID (可为空，支持匿名)
    chain_id            INTEGER REFERENCES chains(id) NOT NULL,
    from_token_id       INTEGER REFERENCES tokens(id) NOT NULL,
    to_token_id         INTEGER REFERENCES tokens(id) NOT NULL,
    amount_in           DECIMAL(78,0) NOT NULL,            -- 输入数量 (wei格式)
    slippage            DECIMAL(5,4) NOT NULL,             -- 滑点设置
    user_address        VARCHAR(66),                        -- 用户钱包地址
    ip_address          INET,                               -- 用户IP
    user_agent          TEXT,                               -- 用户代理
    request_source      VARCHAR(20) DEFAULT 'web',         -- web/mobile/api
    aggregator_count    INTEGER DEFAULT 0,                 -- 调用的聚合器数量
    success_count       INTEGER DEFAULT 0,                 -- 成功响应数量
    best_aggregator_id  INTEGER REFERENCES aggregators(id), -- 最佳聚合器
    best_amount_out     DECIMAL(78,0),                     -- 最佳输出数量
    best_gas_estimate   BIGINT,                            -- 最佳Gas估算
    best_price_impact   DECIMAL(8,6),                      -- 最佳价格冲击
    total_duration_ms   INTEGER,                           -- 总耗时毫秒
    cache_hit           BOOLEAN DEFAULT false,             -- 是否命中缓存
    status              VARCHAR(20) DEFAULT 'pending',     -- pending/completed/failed
    error_message       TEXT,                              -- 错误信息
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at        TIMESTAMP                          -- 完成时间
);

-- 聚合器响应详情表
CREATE TABLE quote_responses (
    id                  SERIAL PRIMARY KEY,
    quote_request_id    INTEGER REFERENCES quote_requests(id) ON DELETE CASCADE,
    aggregator_id       INTEGER REFERENCES aggregators(id) NOT NULL,
    response_time_ms    INTEGER NOT NULL,                  -- 响应时间毫秒
    success             BOOLEAN NOT NULL,                  -- 是否成功
    amount_out          DECIMAL(78,0),                     -- 输出数量
    gas_estimate        BIGINT,                            -- Gas估算
    price_impact        DECIMAL(8,6),                      -- 价格冲击
    confidence_score    DECIMAL(3,2),                      -- 置信度分数
    price_rank          INTEGER,                           -- 价格排名
    route_data          JSONB,                             -- 交易路径数据
    raw_response        JSONB,                             -- 原始响应数据
    error_code          VARCHAR(50),                       -- 错误代码
    error_message       TEXT,                              -- 错误信息
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ========================================
-- 4. 交易相关表
-- ========================================

-- 交易记录表
CREATE TABLE transactions (
    id                  SERIAL PRIMARY KEY,
    user_id             INTEGER REFERENCES users(id),
    quote_request_id    INTEGER REFERENCES quote_requests(id), -- 关联的报价请求
    tx_hash             VARCHAR(66) UNIQUE,                -- 交易哈希
    chain_id            INTEGER REFERENCES chains(id) NOT NULL,
    from_token_id       INTEGER REFERENCES tokens(id) NOT NULL,
    to_token_id         INTEGER REFERENCES tokens(id) NOT NULL,
    aggregator_id       INTEGER REFERENCES aggregators(id) NOT NULL,
    
    -- 交易参数
    amount_in           DECIMAL(78,0) NOT NULL,            -- 实际输入数量
    amount_out_expected DECIMAL(78,0) NOT NULL,            -- 预期输出数量
    amount_out_actual   DECIMAL(78,0),                     -- 实际输出数量
    slippage_set        DECIMAL(5,4) NOT NULL,             -- 设置的滑点
    slippage_actual     DECIMAL(8,6),                      -- 实际滑点
    
    -- Gas相关
    gas_limit           BIGINT,                            -- Gas限制
    gas_used            BIGINT,                            -- 实际使用Gas
    gas_price           BIGINT,                            -- Gas价格 (wei)
    gas_fee_eth         DECIMAL(18,9),                     -- Gas费用 (ETH)
    gas_fee_usd         DECIMAL(10,2),                     -- Gas费用 (USD)
    
    -- 价格相关
    price_impact        DECIMAL(8,6),                      -- 价格冲击
    exchange_rate       DECIMAL(36,18),                    -- 汇率 (amount_out/amount_in)
    amount_in_usd       DECIMAL(12,2),                     -- 输入金额USD
    amount_out_usd      DECIMAL(12,2),                     -- 输出金额USD
    
    -- 交易状态和时间
    status              VARCHAR(20) DEFAULT 'pending',     -- pending/confirmed/failed/cancelled
    user_address        VARCHAR(66) NOT NULL,              -- 用户钱包地址
    to_address          VARCHAR(66),                       -- 目标合约地址
    block_number        BIGINT,                            -- 区块号
    block_timestamp     TIMESTAMP,                         -- 区块时间戳
    confirmation_count  INTEGER DEFAULT 0,                 -- 确认数
    
    -- 元数据
    route_data          JSONB,                             -- 交易路径详情
    error_reason        TEXT,                              -- 失败原因
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    confirmed_at        TIMESTAMP                          -- 确认时间
);

-- ========================================
-- 5. 系统监控和统计表
-- ========================================

-- 聚合器性能统计表 (按小时聚合)
CREATE TABLE aggregator_stats_hourly (
    id                  SERIAL PRIMARY KEY,
    aggregator_id       INTEGER REFERENCES aggregators(id) ON DELETE CASCADE,
    hour_timestamp      TIMESTAMP NOT NULL,                -- 小时时间戳
    total_requests      INTEGER DEFAULT 0,                 -- 总请求数
    successful_requests INTEGER DEFAULT 0,                 -- 成功请求数
    failed_requests     INTEGER DEFAULT 0,                 -- 失败请求数
    avg_response_time   INTEGER DEFAULT 0,                 -- 平均响应时间ms
    min_response_time   INTEGER DEFAULT 0,                 -- 最小响应时间ms
    max_response_time   INTEGER DEFAULT 0,                 -- 最大响应时间ms
    total_volume_usd    DECIMAL(15,2) DEFAULT 0,           -- 总交易量USD
    best_quotes_count   INTEGER DEFAULT 0,                 -- 最优报价次数
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(aggregator_id, hour_timestamp)
);

-- 代币对交易统计表 (按日聚合)
CREATE TABLE token_pair_stats_daily (
    id                  SERIAL PRIMARY KEY,
    from_token_id       INTEGER REFERENCES tokens(id) ON DELETE CASCADE,
    to_token_id         INTEGER REFERENCES tokens(id) ON DELETE CASCADE,
    chain_id            INTEGER REFERENCES chains(id) ON DELETE CASCADE,
    date                DATE NOT NULL,                     -- 日期
    transaction_count   INTEGER DEFAULT 0,                 -- 交易次数
    total_volume_from   DECIMAL(78,0) DEFAULT 0,           -- 总输入量
    total_volume_to     DECIMAL(78,0) DEFAULT 0,           -- 总输出量
    total_volume_usd    DECIMAL(15,2) DEFAULT 0,           -- 总交易量USD
    avg_price_impact    DECIMAL(8,6),                      -- 平均价格冲击
    avg_gas_fee_usd     DECIMAL(10,2),                     -- 平均Gas费用USD
    unique_users        INTEGER DEFAULT 0,                 -- 独立用户数
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(from_token_id, to_token_id, chain_id, date)
);

-- 系统性能监控表
CREATE TABLE system_metrics (
    id                  SERIAL PRIMARY KEY,
    metric_name         VARCHAR(100) NOT NULL,             -- 指标名称
    metric_value        DECIMAL(15,6) NOT NULL,            -- 指标值
    metric_unit         VARCHAR(20),                       -- 单位
    tags                JSONB,                             -- 标签数据
    timestamp           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX(metric_name, timestamp)
);

-- ========================================
-- 6. 缓存相关表 (可选，主要使用Redis)
-- ========================================

-- 报价缓存表 (作为Redis的备份)
CREATE TABLE quote_cache (
    id                  SERIAL PRIMARY KEY,
    cache_key           VARCHAR(255) UNIQUE NOT NULL,      -- 缓存键
    from_token_address  VARCHAR(66) NOT NULL,
    to_token_address    VARCHAR(66) NOT NULL,
    amount_in           DECIMAL(78,0) NOT NULL,
    chain_id            INTEGER NOT NULL,
    quote_data          JSONB NOT NULL,                    -- 报价数据
    hit_count           INTEGER DEFAULT 0,                 -- 命中次数
    expires_at          TIMESTAMP NOT NULL,                -- 过期时间
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ========================================
-- 7. 创建索引以优化查询性能
-- ========================================

-- 用户相关索引
CREATE INDEX idx_users_wallet_address ON users(wallet_address);
CREATE INDEX idx_users_created_at ON users(created_at DESC);
CREATE INDEX idx_users_last_login ON users(last_login_at DESC);

-- 代币相关索引
CREATE INDEX idx_tokens_chain_address ON tokens(chain_id, contract_address);
CREATE INDEX idx_tokens_symbol ON tokens(symbol);
CREATE INDEX idx_tokens_is_active ON tokens(is_active);
CREATE INDEX idx_tokens_price_updated ON tokens(price_updated_at DESC);

-- 报价请求相关索引
CREATE INDEX idx_quote_requests_user_id ON quote_requests(user_id);
CREATE INDEX idx_quote_requests_created_at ON quote_requests(created_at DESC);
CREATE INDEX idx_quote_requests_status ON quote_requests(status);
CREATE INDEX idx_quote_requests_token_pair ON quote_requests(from_token_id, to_token_id);
CREATE INDEX idx_quote_requests_chain ON quote_requests(chain_id);

-- 交易相关索引
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_tx_hash ON transactions(tx_hash);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_token_pair ON transactions(from_token_id, to_token_id);
CREATE INDEX idx_transactions_block_number ON transactions(block_number);

-- 聚合器性能索引
CREATE INDEX idx_aggregator_stats_time ON aggregator_stats_hourly(hour_timestamp DESC);
CREATE INDEX idx_aggregator_stats_agg_time ON aggregator_stats_hourly(aggregator_id, hour_timestamp);

-- 代币对统计索引
CREATE INDEX idx_token_pair_stats_date ON token_pair_stats_daily(date DESC);
CREATE INDEX idx_token_pair_stats_pair_date ON token_pair_stats_daily(from_token_id, to_token_id, date);

-- 系统指标索引
CREATE INDEX idx_system_metrics_name_time ON system_metrics(metric_name, timestamp DESC);

-- 缓存相关索引
CREATE INDEX idx_quote_cache_expires ON quote_cache(expires_at);
CREATE INDEX idx_quote_cache_tokens ON quote_cache(from_token_address, to_token_address);

-- ========================================
-- 8. 创建视图以简化常用查询
-- ========================================

-- 用户交易统计视图
CREATE VIEW user_trading_stats AS
SELECT 
    u.id as user_id,
    u.wallet_address,
    COUNT(t.id) as total_transactions,
    COUNT(CASE WHEN t.status = 'confirmed' THEN 1 END) as successful_transactions,
    SUM(CASE WHEN t.status = 'confirmed' THEN t.amount_in_usd ELSE 0 END) as total_volume_usd,
    SUM(CASE WHEN t.status = 'confirmed' THEN t.gas_fee_usd ELSE 0 END) as total_gas_fee_usd,
    AVG(CASE WHEN t.status = 'confirmed' THEN t.price_impact END) as avg_price_impact,
    MAX(t.created_at) as last_transaction_at
FROM users u
LEFT JOIN transactions t ON u.id = t.user_id
GROUP BY u.id, u.wallet_address;

-- 聚合器综合排名视图
CREATE VIEW aggregator_rankings AS
SELECT 
    a.id,
    a.name,
    a.display_name,
    COUNT(qr.id) as total_responses,
    COUNT(CASE WHEN qr.success = true THEN 1 END) as successful_responses,
    ROUND(COUNT(CASE WHEN qr.success = true THEN 1 END)::DECIMAL / NULLIF(COUNT(qr.id), 0) * 100, 2) as success_rate,
    AVG(CASE WHEN qr.success = true THEN qr.response_time_ms END) as avg_response_time,
    COUNT(CASE WHEN qr.price_rank = 1 THEN 1 END) as best_price_count,
    SUM(CASE WHEN t.status = 'confirmed' THEN t.amount_in_usd ELSE 0 END) as total_volume_usd
FROM aggregators a
LEFT JOIN quote_responses qr ON a.id = qr.aggregator_id
LEFT JOIN transactions t ON a.id = t.aggregator_id
WHERE a.is_active = true
GROUP BY a.id, a.name, a.display_name
ORDER BY success_rate DESC, avg_response_time ASC;

-- 热门代币对视图
CREATE VIEW popular_token_pairs AS
SELECT 
    ft.symbol as from_symbol,
    tt.symbol as to_symbol,
    ft.name as from_name,
    tt.name as to_name,
    c.name as chain_name,
    COUNT(qr.id) as quote_requests,
    COUNT(t.id) as transactions,
    SUM(CASE WHEN t.status = 'confirmed' THEN t.amount_in_usd ELSE 0 END) as total_volume_usd,
    AVG(CASE WHEN t.status = 'confirmed' THEN t.price_impact END) as avg_price_impact
FROM quote_requests qr
JOIN tokens ft ON qr.from_token_id = ft.id
JOIN tokens tt ON qr.to_token_id = tt.id
JOIN chains c ON qr.chain_id = c.id
LEFT JOIN transactions t ON qr.id = t.quote_request_id
WHERE qr.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY ft.symbol, tt.symbol, ft.name, tt.name, c.name
HAVING COUNT(qr.id) >= 10
ORDER BY total_volume_usd DESC, quote_requests DESC;

-- ========================================
-- 9. 添加数据库约束和触发器
-- ========================================

-- 确保交易金额为正数
ALTER TABLE quote_requests ADD CONSTRAINT chk_amount_positive CHECK (amount_in > 0);
ALTER TABLE transactions ADD CONSTRAINT chk_amount_in_positive CHECK (amount_in > 0);
ALTER TABLE transactions ADD CONSTRAINT chk_amount_out_positive CHECK (amount_out_expected > 0);

-- 确保滑点在合理范围内 (0% - 50%)
ALTER TABLE quote_requests ADD CONSTRAINT chk_slippage_range CHECK (slippage >= 0 AND slippage <= 0.5);
ALTER TABLE transactions ADD CONSTRAINT chk_slippage_range CHECK (slippage_set >= 0 AND slippage_set <= 0.5);

-- 确保价格冲击在合理范围内
ALTER TABLE quote_responses ADD CONSTRAINT chk_price_impact_range CHECK (price_impact >= 0 AND price_impact <= 1);
ALTER TABLE transactions ADD CONSTRAINT chk_price_impact_range CHECK (price_impact >= 0 AND price_impact <= 1);

-- 自动更新updated_at字段的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为相关表创建触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON user_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chains_updated_at BEFORE UPDATE ON chains
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tokens_updated_at BEFORE UPDATE ON tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_aggregators_updated_at BEFORE UPDATE ON aggregators
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ========================================
-- 数据库设计完成
-- 总计包含：
-- - 12个核心业务表
-- - 3个统计分析表  
-- - 1个缓存备份表
-- - 3个查询优化视图
-- - 完整的索引策略
-- - 数据约束和触发器
-- ========================================
