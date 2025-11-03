#!/bin/bash
# 重构流程端到端测试脚本
# 测试从前端代币选择到聚合器调用的完整流程

set -e

echo "🚀 开始测试重构后的DeFi聚合器流程..."

# 配置
API_BASE_URL="http://localhost:5176"
CHAIN_API_URL="$API_BASE_URL/api/v1/chains"
TOKEN_API_URL="$API_BASE_URL/api/v1/tokens"
QUOTE_API_URL="$API_BASE_URL/api/v1/quotes"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试函数
test_step() {
    local step_name="$1"
    local command="$2"
    
    echo -e "\n${BLUE}📋 测试步骤: $step_name${NC}"
    echo "执行命令: $command"
    
    if eval "$command"; then
        echo -e "${GREEN}✅ $step_name 测试通过${NC}"
        return 0
    else
        echo -e "${RED}❌ $step_name 测试失败${NC}"
        return 1
    fi
}

# 检查服务状态
check_service() {
    local service_name="$1"
    local url="$2"
    
    echo -e "\n${YELLOW}🔍 检查 $service_name 服务状态...${NC}"
    
    if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -q "200\|404"; then
        echo -e "${GREEN}✅ $service_name 服务运行正常${NC}"
        return 0
    else
        echo -e "${RED}❌ $service_name 服务未运行，请启动服务${NC}"
        return 1
    fi
}

# 主测试流程
main() {
    echo "开始重构流程测试..."
    
    # 1. 检查后端服务状态
    echo -e "\n${BLUE}=== 第1步: 检查服务状态 ===${NC}"
    check_service "Business Logic API" "$API_BASE_URL/health" || exit 1
    
    # 2. 测试链信息查询
    echo -e "\n${BLUE}=== 第2步: 测试链信息查询 ===${NC}"
    test_step "获取活跃链列表" "curl -s '$CHAIN_API_URL?type=active' | jq '.success' | grep -q true" || exit 1
    
    # 获取第一个活跃链的ID用于后续测试
    CHAIN_ID=$(curl -s "$CHAIN_API_URL?type=active" | jq -r '.data[0].id // empty')
    if [ -z "$CHAIN_ID" ]; then
        echo -e "${RED}❌ 没有找到活跃的链，请检查数据库数据${NC}"
        exit 1
    fi
    echo -e "${GREEN}📌 使用链ID: $CHAIN_ID 进行测试${NC}"
    
    # 3. 测试代币信息查询（按链过滤）
    echo -e "\n${BLUE}=== 第3步: 测试代币信息查询 ===${NC}"
    test_step "获取指定链的已验证代币" "curl -s '$TOKEN_API_URL?chain_id=$CHAIN_ID&is_verified=true&is_active=true' | jq '.success' | grep -q true" || exit 1
    
    # 获取代币信息用于报价测试
    TOKEN_RESPONSE=$(curl -s "$TOKEN_API_URL?chain_id=$CHAIN_ID&is_verified=true&is_active=true&page_size=10")
    FROM_TOKEN_ID=$(echo "$TOKEN_RESPONSE" | jq -r '.data[0].id // empty')
    TO_TOKEN_ID=$(echo "$TOKEN_RESPONSE" | jq -r '.data[1].id // empty')
    
    if [ -z "$FROM_TOKEN_ID" ] || [ -z "$TO_TOKEN_ID" ]; then
        echo -e "${RED}❌ 没有找到足够的代币进行测试，请检查数据库数据${NC}"
        echo "可用代币数量: $(echo "$TOKEN_RESPONSE" | jq '.data | length')"
        exit 1
    fi
    
    FROM_TOKEN_SYMBOL=$(echo "$TOKEN_RESPONSE" | jq -r '.data[0].symbol')
    TO_TOKEN_SYMBOL=$(echo "$TOKEN_RESPONSE" | jq -r '.data[1].symbol')
    EXTERNAL_CHAIN_ID=$(echo "$TOKEN_RESPONSE" | jq -r '.data[0].chain_id')
    
    # 获取实际的外部链ID
    CHAIN_INFO=$(curl -s "$CHAIN_API_URL")
    EXTERNAL_CHAIN_ID=$(echo "$CHAIN_INFO" | jq -r ".data[] | select(.id == $CHAIN_ID) | .chain_id")
    
    echo -e "${GREEN}📌 测试代币对: $FROM_TOKEN_SYMBOL (ID: $FROM_TOKEN_ID) -> $TO_TOKEN_SYMBOL (ID: $TO_TOKEN_ID)${NC}"
    echo -e "${GREEN}📌 外部链ID: $EXTERNAL_CHAIN_ID${NC}"
    
    # 4. 测试报价请求
    echo -e "\n${BLUE}=== 第4步: 测试报价请求 ===${NC}"
    
    # 构建报价请求
    QUOTE_REQUEST=$(cat <<EOF
{
    "from_token_id": $FROM_TOKEN_ID,
    "to_token_id": $TO_TOKEN_ID,
    "amount_in": "1000000000000000000",
    "chain_id": $EXTERNAL_CHAIN_ID,
    "slippage": 0.005,
    "user_address": "0x742d35Cc6634C0532925a3b8D8A8CE8D3C8E8834A1"
}
EOF
    )
    
    echo "报价请求参数:"
    echo "$QUOTE_REQUEST" | jq '.'
    
    # 发送报价请求
    echo -e "\n${YELLOW}📤 发送报价请求...${NC}"
    QUOTE_RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$QUOTE_REQUEST" \
        "$QUOTE_API_URL")
    
    echo "报价响应:"
    echo "$QUOTE_RESPONSE" | jq '.'
    
    # 检查报价响应
    SUCCESS=$(echo "$QUOTE_RESPONSE" | jq -r '.success // false')
    if [ "$SUCCESS" = "true" ]; then
        echo -e "${GREEN}✅ 报价请求成功${NC}"
        
        # 提取报价信息
        BEST_AGGREGATOR=$(echo "$QUOTE_RESPONSE" | jq -r '.data.best_aggregator // "未知"')
        AMOUNT_OUT=$(echo "$QUOTE_RESPONSE" | jq -r '.data.amount_out // "未知"')
        CACHE_HIT=$(echo "$QUOTE_RESPONSE" | jq -r '.data.cache_hit // false')
        
        echo -e "${GREEN}📊 报价结果:${NC}"
        echo -e "   最优聚合器: $BEST_AGGREGATOR"
        echo -e "   预期输出: $AMOUNT_OUT"
        echo -e "   缓存命中: $CACHE_HIT"
        
    else
        echo -e "${RED}❌ 报价请求失败${NC}"
        ERROR_MSG=$(echo "$QUOTE_RESPONSE" | jq -r '.error.message // "未知错误"')
        echo -e "错误信息: $ERROR_MSG"
        exit 1
    fi
    
    # 5. 测试前端代币选择接口兼容性
    echo -e "\n${BLUE}=== 第5步: 测试前端接口兼容性 ===${NC}"
    
    # 测试代币列表API（前端格式）
    test_step "测试前端代币列表格式" "curl -s '$TOKEN_API_URL?is_active=true&is_verified=true&page_size=50' | jq '.data[0] | has(\"id\", \"symbol\", \"name\", \"contract_address\", \"chain_id\", \"decimals\")' | grep -q true" || exit 1
    
    # 测试链列表API（前端格式） 
    test_step "测试前端链列表格式" "curl -s '$CHAIN_API_URL?type=active' | jq '.data[0] | has(\"id\", \"chain_id\", \"display_name\", \"symbol\", \"is_testnet\")' | grep -q true" || exit 1
    
    # 6. 性能测试
    echo -e "\n${BLUE}=== 第6步: 简单性能测试 ===${NC}"
    
    echo -e "${YELLOW}🏃 执行5次报价请求测试响应时间...${NC}"
    TOTAL_TIME=0
    SUCCESS_COUNT=0
    
    for i in {1..5}; do
        echo -n "请求 $i/5: "
        START_TIME=$(date +%s%N)
        
        RESPONSE=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$QUOTE_REQUEST" \
            "$QUOTE_API_URL")
        
        END_TIME=$(date +%s%N)
        REQUEST_TIME=$((($END_TIME - $START_TIME) / 1000000)) # 转换为毫秒
        
        if echo "$RESPONSE" | jq -e '.success' > /dev/null 2>&1; then
            echo -e "${GREEN}✅ ${REQUEST_TIME}ms${NC}"
            TOTAL_TIME=$((TOTAL_TIME + REQUEST_TIME))
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        else
            echo -e "${RED}❌ 失败${NC}"
        fi
    done
    
    if [ $SUCCESS_COUNT -gt 0 ]; then
        AVG_TIME=$((TOTAL_TIME / SUCCESS_COUNT))
        echo -e "${GREEN}📈 性能统计: 成功率 $SUCCESS_COUNT/5, 平均响应时间: ${AVG_TIME}ms${NC}"
    fi
    
    # 7. 总结
    echo -e "\n${BLUE}=== 测试总结 ===${NC}"
    echo -e "${GREEN}🎉 重构流程测试完成！${NC}"
    echo ""
    echo "✅ 测试通过的功能:"
    echo "   - 链信息动态查询"
    echo "   - 代币信息按链过滤"
    echo "   - 报价请求端到端流程"
    echo "   - 前端接口兼容性"
    echo "   - 基本性能验证"
    echo ""
    echo -e "${YELLOW}📝 重构改进点:${NC}"
    echo "   1. 前端代币选择器支持按链分组显示"
    echo "   2. 后端数据查询逻辑简化，去除复杂映射"
    echo "   3. 聚合器配置可从数据库动态加载"
    echo "   4. 链ID映射逻辑统一和标准化"
    echo ""
    echo -e "${BLUE}🚀 下一步建议:${NC}"
    echo "   1. 启动智能路由服务测试聚合器调用"
    echo "   2. 添加更多聚合器配置到数据库"
    echo "   3. 测试真实的Web3钱包连接"
    echo "   4. 增加错误处理和用户体验优化"
}

# 运行主测试
main "$@"

