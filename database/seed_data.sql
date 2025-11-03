-- DeFi聚合器初始数据种子文件
-- 包含开发和测试环境所需的基础数据

-- ========================================
-- 1. 支持的区块链网络数据
-- ========================================

INSERT INTO chains (chain_id, name, display_name, symbol, rpc_url, explorer_url, is_testnet, is_active, gas_price_gwei, block_time_sec) VALUES
-- 主网
(1, 'ethereum', 'Ethereum', 'ETH', 'https://mainnet.infura.io/v3/YOUR_PROJECT_ID', 'https://etherscan.io', false, true, 20, 15),
(137, 'polygon', 'Polygon', 'MATIC', 'https://polygon-mainnet.infura.io/v3/YOUR_PROJECT_ID', 'https://polygonscan.com', false, true, 30, 2),
(42161, 'arbitrum', 'Arbitrum One', 'ETH', 'https://arbitrum-mainnet.infura.io/v3/YOUR_PROJECT_ID', 'https://arbiscan.io', false, true, 1, 1),
(10, 'optimism', 'Optimism', 'ETH', 'https://optimism-mainnet.infura.io/v3/YOUR_PROJECT_ID', 'https://optimistic.etherscan.io', false, true, 1, 2),

-- 测试网 (开发环境使用)
(11155111, 'sepolia', 'Sepolia Testnet', 'SepoliaETH', 'https://sepolia.infura.io/v3/YOUR_PROJECT_ID', 'https://sepolia.etherscan.io', true, true, 20, 15),
(80001, 'mumbai', 'Mumbai Testnet', 'MATIC', 'https://polygon-mumbai.infura.io/v3/YOUR_PROJECT_ID', 'https://mumbai.polygonscan.com', true, true, 30, 2);

-- ========================================
-- 2. 第三方聚合器配置
-- ========================================

INSERT INTO aggregators (name, display_name, api_url, logo_url, is_active, priority, timeout_ms, retry_count) VALUES
('1inch', '1inch', 'https://api.1inch.io/v5.0', 'https://app.1inch.io/assets/images/1inch_logo.svg', true, 1, 3000, 3),
('paraswap', 'ParaSwap', 'https://apiv5.paraswap.io', 'https://paraswap.io/paraswap.svg', true, 2, 4000, 3),
('0x', '0x Protocol', 'https://api.0x.org', 'https://0x.org/images/favicon.png', true, 3, 5000, 2),
('cowswap', 'CoW Protocol', 'https://api.cow.fi/mainnet/api/v1', 'https://cow.fi/favicon.ico', true, 4, 6000, 2);

-- ========================================
-- 3. 聚合器支持的链配置
-- ========================================

-- 1inch 支持的链
INSERT INTO aggregator_chains (aggregator_id, chain_id, is_active, gas_multiplier) VALUES
((SELECT id FROM aggregators WHERE name = '1inch'), (SELECT id FROM chains WHERE chain_id = 1), true, 1.0),
((SELECT id FROM aggregators WHERE name = '1inch'), (SELECT id FROM chains WHERE chain_id = 137), true, 1.0),
((SELECT id FROM aggregators WHERE name = '1inch'), (SELECT id FROM chains WHERE chain_id = 42161), true, 1.0),
((SELECT id FROM aggregators WHERE name = '1inch'), (SELECT id FROM chains WHERE chain_id = 10), true, 1.0);

-- ParaSwap 支持的链
INSERT INTO aggregator_chains (aggregator_id, chain_id, is_active, gas_multiplier) VALUES
((SELECT id FROM aggregators WHERE name = 'paraswap'), (SELECT id FROM chains WHERE chain_id = 1), true, 1.0),
((SELECT id FROM aggregators WHERE name = 'paraswap'), (SELECT id FROM chains WHERE chain_id = 137), true, 1.0),
((SELECT id FROM aggregators WHERE name = 'paraswap'), (SELECT id FROM chains WHERE chain_id = 42161), true, 1.0);

-- 0x Protocol 支持的链
INSERT INTO aggregator_chains (aggregator_id, chain_id, is_active, gas_multiplier) VALUES
((SELECT id FROM aggregators WHERE name = '0x'), (SELECT id FROM chains WHERE chain_id = 1), true, 1.0),
((SELECT id FROM aggregators WHERE name = '0x'), (SELECT id FROM chains WHERE chain_id = 137), true, 1.0);

-- CoW Protocol 支持的链 (主要是以太坊)
INSERT INTO aggregator_chains (aggregator_id, chain_id, is_active, gas_multiplier) VALUES
((SELECT id FROM aggregators WHERE name = 'cowswap'), (SELECT id FROM chains WHERE chain_id = 1), true, 1.0);

-- ========================================
-- 4. 主要代币信息 (以太坊主网)
-- ========================================

INSERT INTO tokens (chain_id, contract_address, symbol, name, decimals, is_native, is_stable, is_verified, is_active, coingecko_id) VALUES
-- 以太坊主网代币
((SELECT id FROM chains WHERE chain_id = 1), '0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE', 'ETH', 'Ethereum', 18, true, false, true, true, 'ethereum'),
((SELECT id FROM chains WHERE chain_id = 1), '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', 'USDC', 'USD Coin', 6, false, true, true, true, 'usd-coin'),
((SELECT id FROM chains WHERE chain_id = 1), '0xdAC17F958D2ee523a2206206994597C13D831ec7', 'USDT', 'Tether USD', 6, false, true, true, true, 'tether'),
((SELECT id FROM chains WHERE chain_id = 1), '0x6B175474E89094C44Da98b954EedeAC495271d0F', 'DAI', 'Dai Stablecoin', 18, false, true, true, true, 'dai'),
((SELECT id FROM chains WHERE chain_id = 1), '0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599', 'WBTC', 'Wrapped Bitcoin', 8, false, false, true, true, 'wrapped-bitcoin'),
((SELECT id FROM chains WHERE chain_id = 1), '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2', 'WETH', 'Wrapped Ether', 18, false, false, true, true, 'weth'),
((SELECT id FROM chains WHERE chain_id = 1), '0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984', 'UNI', 'Uniswap', 18, false, false, true, true, 'uniswap'),
((SELECT id FROM chains WHERE chain_id = 1), '0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9', 'AAVE', 'Aave Token', 18, false, false, true, true, 'aave'),
((SELECT id FROM chains WHERE chain_id = 1), '0x514910771AF9Ca656af840dff83E8264EcF986CA', 'LINK', 'ChainLink Token', 18, false, false, true, true, 'chainlink'),
((SELECT id FROM chains WHERE chain_id = 1), '0x6B3595068778DD592e39A122f4f5a5cF09C90fE2', 'SUSHI', 'SushiToken', 18, false, false, true, true, 'sushi');

-- Polygon 主要代币
INSERT INTO tokens (chain_id, contract_address, symbol, name, decimals, is_native, is_stable, is_verified, is_active, coingecko_id) VALUES
((SELECT id FROM chains WHERE chain_id = 137), '0x0000000000000000000000000000000000001010', 'MATIC', 'Polygon', 18, true, false, true, true, 'matic-network'),
((SELECT id FROM chains WHERE chain_id = 137), '0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174', 'USDC', 'USD Coin (PoS)', 6, false, true, true, true, 'usd-coin'),
((SELECT id FROM chains WHERE chain_id = 137), '0xc2132D05D31c914a87C6611C10748AEb04B58e8F', 'USDT', 'Tether USD (PoS)', 6, false, true, true, true, 'tether'),
((SELECT id FROM chains WHERE chain_id = 137), '0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063', 'DAI', 'Dai Stablecoin (PoS)', 18, false, true, true, true, 'dai'),
((SELECT id FROM chains WHERE chain_id = 137), '0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619', 'WETH', 'Wrapped Ether', 18, false, false, true, true, 'weth'),
((SELECT id FROM chains WHERE chain_id = 137), '0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270', 'WMATIC', 'Wrapped Matic', 18, false, false, true, true, 'wmatic');

-- 测试网代币 (Sepolia)
INSERT INTO tokens (chain_id, contract_address, symbol, name, decimals, is_native, is_stable, is_verified, is_active) VALUES
((SELECT id FROM chains WHERE chain_id = 11155111), '0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE', 'SepoliaETH', 'Sepolia Ether', 18, true, false, true, true),
((SELECT id FROM chains WHERE chain_id = 11155111), '0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238', 'USDC', 'USD Coin (Sepolia)', 6, false, true, true, true),
((SELECT id FROM chains WHERE chain_id = 11155111), '0xaA8E23Fb1079EA71e0a56F48a2aA51851D8433D0', 'USDT', 'Tether USD (Sepolia)', 6, false, true, true, true);

-- ========================================
-- 5. 测试用户数据 (开发环境)
-- ========================================

-- 插入测试用户
INSERT INTO users (wallet_address, username, email, preferred_language, is_active) VALUES
('0x742d35Cc6634C0532925a3b8D8A8CE8D3C8E8834A1', 'test_user_1', 'test1@example.com', 'en', true),
('0x8ba1f109551bD432803012645Hac136c1c7AD8B8A', 'test_user_2', 'test2@example.com', 'zh', true),
('0x1234567890123456789012345678901234567890', 'dev_user', 'dev@example.com', 'en', true);

-- 插入用户偏好设置
INSERT INTO user_preferences (user_id, default_slippage, preferred_gas_speed, auto_approve_tokens, show_test_tokens) VALUES
((SELECT id FROM users WHERE wallet_address = '0x742d35Cc6634C0532925a3b8D8A8CE8D3C8E8834A1'), 0.005, 'standard', false, true),
((SELECT id FROM users WHERE wallet_address = '0x8ba1f109551bD432803012645Hac136c1c7AD8B8A'), 0.010, 'fast', true, false),
((SELECT id FROM users WHERE wallet_address = '0x1234567890123456789012345678901234567890'), 0.003, 'slow', false, true);

-- ========================================
-- 6. 常用代币对的示例报价请求 (用于测试)
-- ========================================

-- 生成一些测试报价请求数据
INSERT INTO quote_requests (
    request_id, 
    user_id, 
    chain_id, 
    from_token_id, 
    to_token_id, 
    amount_in, 
    slippage, 
    user_address, 
    request_source,
    status,
    completed_at
) VALUES
-- ETH to USDC 交易
(
    'test_req_001', 
    (SELECT id FROM users WHERE wallet_address = '0x742d35Cc6634C0532925a3b8D8A8CE8D3C8E8834A1'),
    (SELECT id FROM chains WHERE chain_id = 1),
    (SELECT id FROM tokens WHERE symbol = 'ETH' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1)),
    (SELECT id FROM tokens WHERE symbol = 'USDC' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1)),
    '1000000000000000000',  -- 1 ETH
    0.005,
    '0x742d35Cc6634C0532925a3b8D8A8CE8D3C8E8834A1',
    'web',
    'completed',
    CURRENT_TIMESTAMP - INTERVAL '1 hour'
),
-- USDC to DAI 交易
(
    'test_req_002',
    (SELECT id FROM users WHERE wallet_address = '0x8ba1f109551bD432803012645Hac136c1c7AD8B8A'),
    (SELECT id FROM chains WHERE chain_id = 1),
    (SELECT id FROM tokens WHERE symbol = 'USDC' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1)),
    (SELECT id FROM tokens WHERE symbol = 'DAI' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1)),
    '1000000000',  -- 1000 USDC
    0.001,
    '0x8ba1f109551bD432803012645Hac136c1c7AD8B8A',
    'web',
    'completed',
    CURRENT_TIMESTAMP - INTERVAL '30 minutes'
);

-- ========================================
-- 7. 系统初始监控指标
-- ========================================

-- 插入一些初始系统指标
INSERT INTO system_metrics (metric_name, metric_value, metric_unit, tags) VALUES
('api_requests_total', 0, 'count', '{"service": "api-gateway"}'),
('quote_requests_total', 0, 'count', '{"service": "smart-router"}'),
('successful_transactions', 0, 'count', '{"service": "business-logic"}'),
('average_response_time', 0, 'ms', '{"service": "smart-router"}'),
('cache_hit_rate', 0, 'percent', '{"service": "redis"}'),
('active_users_daily', 0, 'count', '{"period": "24h"}');

-- ========================================
-- 8. 更新代币价格 (示例数据)
-- ========================================

-- 更新主要代币的示例价格 (实际环境中应该从CoinGecko等API获取)
UPDATE tokens SET 
    price_usd = 2000.00,
    price_updated_at = CURRENT_TIMESTAMP,
    daily_volume_usd = 15000000000.00,
    market_cap_usd = 240000000000.00
WHERE symbol = 'ETH' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1);

UPDATE tokens SET 
    price_usd = 1.00,
    price_updated_at = CURRENT_TIMESTAMP,
    daily_volume_usd = 5000000000.00,
    market_cap_usd = 32000000000.00
WHERE symbol = 'USDC' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1);

UPDATE tokens SET 
    price_usd = 1.00,
    price_updated_at = CURRENT_TIMESTAMP,
    daily_volume_usd = 4000000000.00,
    market_cap_usd = 83000000000.00
WHERE symbol = 'USDT' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1);

UPDATE tokens SET 
    price_usd = 1.00,
    price_updated_at = CURRENT_TIMESTAMP,
    daily_volume_usd = 500000000.00,
    market_cap_usd = 5300000000.00
WHERE symbol = 'DAI' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1);

UPDATE tokens SET 
    price_usd = 43000.00,
    price_updated_at = CURRENT_TIMESTAMP,
    daily_volume_usd = 400000000.00,
    market_cap_usd = 845000000000.00
WHERE symbol = 'WBTC' AND chain_id = (SELECT id FROM chains WHERE chain_id = 1);

-- Polygon 代币价格
UPDATE tokens SET 
    price_usd = 0.85,
    price_updated_at = CURRENT_TIMESTAMP,
    daily_volume_usd = 200000000.00,
    market_cap_usd = 8500000000.00
WHERE symbol = 'MATIC' AND chain_id = (SELECT id FROM chains WHERE chain_id = 137);

-- ========================================
-- 数据种子完成
-- ========================================

-- 输出完成信息
SELECT 'Seed data inserted successfully' as status,
       COUNT(DISTINCT c.id) as chains_count,
       COUNT(DISTINCT a.id) as aggregators_count,
       COUNT(DISTINCT t.id) as tokens_count,
       COUNT(DISTINCT u.id) as users_count
FROM chains c
CROSS JOIN aggregators a  
CROSS JOIN tokens t
CROSS JOIN users u;
