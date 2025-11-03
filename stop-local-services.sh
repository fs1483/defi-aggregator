#!/bin/bash

# 停止本地运行的DeFi聚合器服务
# 用于混合模式启动的服务停止

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🛑 停止DeFi聚合器本地服务${NC}"
echo "=================================="

# 检查logs目录是否存在
if [[ ! -d "logs" ]]; then
    echo -e "${YELLOW}⚠️  未找到logs目录，可能没有本地服务在运行${NC}"
    exit 0
fi

# 停止本地服务
SERVICES=("smart-router" "business-logic" "api-gateway" "frontend")

for service in "${SERVICES[@]}"; do
    PID_FILE="logs/${service}.pid"
    
    if [[ -f "$PID_FILE" ]]; then
        PID=$(cat "$PID_FILE")
        
        if ps -p $PID > /dev/null 2>&1; then
            echo -e "${YELLOW}停止 $service 服务 (PID: $PID)...${NC}"
            kill $PID
            
            # 等待进程结束
            sleep 2
            
            if ps -p $PID > /dev/null 2>&1; then
                echo -e "${RED}强制停止 $service 服务...${NC}"
                kill -9 $PID
            fi
            
            echo -e "${GREEN}✅ $service 服务已停止${NC}"
        else
            echo -e "${YELLOW}⚠️  $service 服务进程不存在 (PID: $PID)${NC}"
        fi
        
        # 删除PID文件
        rm -f "$PID_FILE"
    else
        echo -e "${YELLOW}⚠️  未找到 $service 的PID文件${NC}"
    fi
done

# 清理日志文件（可选）
read -p "是否清理日志文件？(y/N): " CLEAN_LOGS
if [[ $CLEAN_LOGS == "y" || $CLEAN_LOGS == "Y" ]]; then
    echo -e "${YELLOW}清理日志文件...${NC}"
    rm -f logs/*.log
    echo -e "${GREEN}✅ 日志文件已清理${NC}"
fi

# 停止Docker服务（可选）
read -p "是否同时停止数据库和Redis服务？(y/N): " STOP_DOCKER
if [[ $STOP_DOCKER == "y" || $STOP_DOCKER == "Y" ]]; then
    echo -e "${YELLOW}停止数据库和Redis服务...${NC}"
    docker-compose stop postgres redis
    echo -e "${GREEN}✅ 数据库和Redis服务已停止${NC}"
fi

echo -e "${GREEN}🎉 所有本地服务已停止！${NC}"
echo ""
echo -e "${BLUE}💡 提示：${NC}"
echo "  重新启动: ./start-dev.sh"
echo "  查看Docker状态: docker-compose ps"
