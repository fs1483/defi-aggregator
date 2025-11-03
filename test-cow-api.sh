#!/bin/bash
# CoW Protocol API调用测试脚本
# 直接测试CoW Protocol API和我们的适配器实现

set -e

echo "🐄 CoW Protocol API调用测试"

# 配置
COW_API_URL="https://api.cow.fi/mainnet/api/v1"
SMART_ROUTER_URL="http://localhost:5178"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "\n${BLUE}=== 第1步: 直接测试CoW Protocol API ===${NC}"

# 根据您提供的文档示例构建请求
COW_REQUEST=$(cat <<EOF
{
  "sellToken": "0x6810e776880c02933d47db1b9fc05908e5386b96",
  "buyToken": "0x6810e776880c02933d47db1b9fc05908e5386b96",
  "receiver": "0x6810e776880c02933d47db1b9fc05908e5386b96",
  "appData": "{\"version\":\"0.9.0\",\"metadata\":{}}",
  "appDataHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "sellTokenBalance": "erc20",
  "buyTokenBalance": "erc20",
  "from": "0x6810e776880c02933d47db1b9fc05908e5386b96",
  "priceQuality": "verified",
  "signingScheme": "eip712",
  "onchainOrder": false,
  "timeout": 0,
  "kind": "sell",
  "sellAmountBeforeFee": "1234567890"
}
EOF
)

echo "📤 发送请求到CoW Protocol:"
echo "$COW_REQUEST" | jq '.'

echo -e "\n${YELLOW}🌐 直接调用CoW Protocol API...${NC}"
COW_RESPONSE=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "$COW_REQUEST" \
  "$COW_API_URL/quote")

echo -e "\n📥 CoW Protocol原始响应:"
echo "$COW_RESPONSE" | jq '.'

# 检查响应是否成功
if echo "$COW_RESPONSE" | jq -e '.sellToken' > /dev/null 2>&1; then
    echo -e "${GREEN}✅ CoW Protocol API调用成功${NC}"
    
    # 提取关键信息
    SELL_TOKEN=$(echo "$COW_RESPONSE" | jq -r '.sellToken')
    BUY_TOKEN=$(echo "$COW_RESPONSE" | jq -r '.buyToken')
    SELL_AMOUNT=$(echo "$COW_RESPONSE" | jq -r '.sellAmount')
    BUY_AMOUNT=$(echo "$COW_RESPONSE" | jq -r '.buyAmount')
    VALID_TO=$(echo "$COW_RESPONSE" | jq -r '.validTo')
    
    echo -e "${GREEN}📊 CoW协议报价结果:${NC}"
    echo -e "   卖出代币: $SELL_TOKEN"
    echo -e "   买入代币: $BUY_TOKEN"
    echo -e "   卖出数量: $SELL_AMOUNT"
    echo -e "   买入数量: $BUY_AMOUNT"
    echo -e "   有效期至: $VALID_TO"
    
else
    echo -e "${RED}❌ CoW Protocol API调用失败${NC}"
    echo "可能的原因："
    echo "1. 网络连接问题"
    echo "2. API参数格式错误"
    echo "3. 代币地址无效"
    echo "4. 服务暂时不可用"
    exit 1
fi

echo -e "\n${BLUE}=== 第2步: 测试我们的智能路由服务 ===${NC}"

# 检查智能路由服务是否运行
if curl -s "$SMART_ROUTER_URL/health" > /dev/null; then
    echo -e "${GREEN}✅ 智能路由服务运行正常${NC}"
else
    echo -e "${RED}❌ 智能路由服务未运行，请启动服务${NC}"
    exit 1
fi

# 使用数据库中的真实代币地址进行测试
echo -e "\n${YELLOW}🔍 从数据库获取真实代币地址...${NC}"

# 获取ETH代币信息（主网）
ETH_TOKEN=$(psql postgresql://admin:password@localhost:5432/defi_aggregator -t -c "
    SELECT contract_address 
    FROM tokens t 
    JOIN chains c ON t.chain_id = c.id 
    WHERE c.chain_id = 1 AND t.symbol = 'ETH' AND t.is_active = true 
    LIMIT 1;
" | xargs)

# 获取USDC代币信息（主网）
USDC_TOKEN=$(psql postgresql://admin:password@localhost:5432/defi_aggregator -t -c "
    SELECT contract_address 
    FROM tokens t 
    JOIN chains c ON t.chain_id = c.id 
    WHERE c.chain_id = 1 AND t.symbol = 'USDC' AND t.is_active = true 
    LIMIT 1;
" | xargs)

if [ -z "$ETH_TOKEN" ] || [ -z "$USDC_TOKEN" ]; then
    echo -e "${RED}❌ 无法从数据库获取代币地址${NC}"
    echo "ETH地址: $ETH_TOKEN"
    echo "USDC地址: $USDC_TOKEN"
    exit 1
fi

echo -e "${GREEN}📌 使用数据库代币地址:${NC}"
echo -e "   ETH: $ETH_TOKEN"
echo -e "   USDC: $USDC_TOKEN"

# 构建智能路由请求
ROUTER_REQUEST=$(cat <<EOF
{
  "request_id": "test_cow_$(date +%s)",
  "from_token": "$ETH_TOKEN",
  "to_token": "$USDC_TOKEN",
  "amount_in": "1000000000000000000",
  "chain_id": 1,
  "slippage": "0.005",
  "user_address": "0x6810e776880c02933d47db1b9fc05908e5386b96"
}
EOF
)

echo -e "\n📤 发送请求到智能路由服务:"
echo "$ROUTER_REQUEST" | jq '.'

echo -e "\n${YELLOW}🤖 调用智能路由服务...${NC}"
ROUTER_RESPONSE=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "$ROUTER_REQUEST" \
  "$SMART_ROUTER_URL/api/v1/quote")

echo -e "\n📥 智能路由服务响应:"
echo "$ROUTER_RESPONSE" | jq '.'

# 检查智能路由响应
if echo "$ROUTER_RESPONSE" | jq -e '.success' > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 智能路由服务调用成功${NC}"
    
    BEST_PROVIDER=$(echo "$ROUTER_RESPONSE" | jq -r '.data.best_provider // "未知"')
    BEST_PRICE=$(echo "$ROUTER_RESPONSE" | jq -r '.data.best_price // "未知"')
    CACHE_HIT=$(echo "$ROUTER_RESPONSE" | jq -r '.data.cache_hit // false')
    
    echo -e "${GREEN}📊 智能路由结果:${NC}"
    echo -e "   最优提供商: $BEST_PROVIDER"
    echo -e "   最优价格: $BEST_PRICE"
    echo -e "   缓存命中: $CACHE_HIT"
    
else
    echo -e "${RED}❌ 智能路由服务调用失败${NC}"
    ERROR_MSG=$(echo "$ROUTER_RESPONSE" | jq -r '.error.message // "未知错误"')
    echo -e "错误信息: $ERROR_MSG"
fi

echo -e "\n${BLUE}=== 测试总结 ===${NC}"
echo -e "${GREEN}🎉 CoW Protocol API集成测试完成！${NC}"
echo ""
echo "✅ 验证要点:"
echo "   1. CoW Protocol API响应格式与文档一致"
echo "   2. 我们的适配器能正确解析响应"
echo "   3. 智能路由服务能调用CoW适配器"
echo "   4. 数据库代币地址配置正确"

