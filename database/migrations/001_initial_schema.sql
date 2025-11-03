-- Migration: 001_initial_schema.sql
-- Description: 创建DeFi聚合器的初始数据库表结构
-- Created: 2024年
-- Version: 1.0.0

-- ========================================
-- 执行此迁移前的检查
-- ========================================

-- 检查PostgreSQL版本 (需要13+)
SELECT version();

-- 检查是否已安装必要的扩展
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- UUID生成函数
-- CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- 文本相似性搜索
-- CREATE EXTENSION IF NOT EXISTS "btree_gin";  -- GIN索引支持

-- ========================================
-- 开始迁移事务
-- ========================================

BEGIN;

-- ========================================
-- 1. 基础配置表（系统级别）
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

-- ========================================
-- 2. 代币信息表
-- ========================================

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
-- 3. 用户管理表
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
-- 4. 报价相关表
-- ========================================

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
-- 5. 交易相关表
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
-- 6. 统计和监控表
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
    timestamp           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ========================================
-- 7. 创建索引
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

-- 报价响应索引
CREATE INDEX idx_quote_responses_request_id ON quote_responses(quote_request_id);
CREATE INDEX idx_quote_responses_aggregator ON quote_responses(aggregator_id);
CREATE INDEX idx_quote_responses_success ON quote_responses(success);

-- 交易相关索引
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_tx_hash ON transactions(tx_hash);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_token_pair ON transactions(from_token_id, to_token_id);
CREATE INDEX idx_transactions_block_number ON transactions(block_number);

-- 统计相关索引
CREATE INDEX idx_aggregator_stats_time ON aggregator_stats_hourly(hour_timestamp DESC);
CREATE INDEX idx_aggregator_stats_agg_time ON aggregator_stats_hourly(aggregator_id, hour_timestamp);
CREATE INDEX idx_token_pair_stats_date ON token_pair_stats_daily(date DESC);
CREATE INDEX idx_token_pair_stats_pair_date ON token_pair_stats_daily(from_token_id, to_token_id, date);
CREATE INDEX idx_system_metrics_name_time ON system_metrics(metric_name, timestamp DESC);

-- ========================================
-- 8. 添加数据约束
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

-- 确保代币精度在合理范围内
ALTER TABLE tokens ADD CONSTRAINT chk_decimals_range CHECK (decimals >= 0 AND decimals <= 30);

-- 确保链ID为正数
ALTER TABLE chains ADD CONSTRAINT chk_chain_id_positive CHECK (chain_id > 0);

-- ========================================
-- 9. 创建触发器函数
-- ========================================

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
-- 提交事务
-- ========================================

COMMIT;

-- ========================================
-- 迁移完成
-- ========================================

-- 输出完成信息
SELECT 'Migration 001_initial_schema.sql completed successfully' as status;
