#!/bin/bash

# DeFi聚合器开发环境启动脚本
# 确保环境变量正确加载并启动所有服务

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 检查基础设施服务状态的函数
check_infrastructure_status() {
    echo "检查基础设施服务状态..."
    
    # 使用docker命令直接检查容器状态，避免docker-compose配置问题
    POSTGRES_RUNNING=$(docker ps -q --filter "name=defi-postgres" 2>/dev/null)
    REDIS_RUNNING=$(docker ps -q --filter "name=defi-redis" 2>/dev/null)
    
    if [[ -n "$POSTGRES_RUNNING" ]] && [[ -n "$REDIS_RUNNING" ]]; then
        # 检查容器健康状态
        POSTGRES_STATUS=$(docker ps --filter "name=defi-postgres" --format "{{.Status}}" 2>/dev/null | grep -c "Up")
        REDIS_STATUS=$(docker ps --filter "name=defi-redis" --format "{{.Status}}" 2>/dev/null | grep -c "Up")
        
        if [[ $POSTGRES_STATUS -gt 0 ]] && [[ $REDIS_STATUS -gt 0 ]]; then
            # 获取详细状态信息
            POSTGRES_HEALTH=$(docker ps --filter "name=defi-postgres" --format "{{.Status}}" 2>/dev/null)
            REDIS_HEALTH=$(docker ps --filter "name=defi-redis" --format "{{.Status}}" 2>/dev/null)
            
            echo -e "${GREEN}✅ PostgreSQL: $POSTGRES_HEALTH${NC}"
            echo -e "${GREEN}✅ Redis: $REDIS_HEALTH${NC}"
            echo -e "${GREEN}🚀 基础设施已就绪，跳过启动${NC}"
            return 0  # 跳过启动
        else
            echo -e "${YELLOW}⚠️  基础设施容器存在但未运行，重新启动...${NC}"
            return 1  # 需要启动
        fi
    else
        echo -e "${BLUE}🔍 基础设施服务未运行，开始启动...${NC}"
        return 1  # 需要启动
    fi
}

# 启动基础设施服务的函数
start_infrastructure() {
    echo "启动数据库和缓存服务..."
    
    # 首先加载环境变量
    if [[ -f "env.global" ]]; then
        source env.global
        echo "已加载全局环境变量"
    fi
    
    # 启动基础设施服务
    docker-compose up -d postgres redis
    
    # 等待服务启动并检查健康状态
    echo "等待服务启动..."
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        # 使用docker命令直接检查容器状态
        POSTGRES_HEALTHY=$(docker ps --filter "name=defi-postgres" --format "{{.Status}}" 2>/dev/null | grep -c "Up")
        REDIS_HEALTHY=$(docker ps --filter "name=defi-redis" --format "{{.Status}}" 2>/dev/null | grep -c "Up")
        
        if [[ $POSTGRES_HEALTHY -gt 0 ]] && [[ $REDIS_HEALTHY -gt 0 ]]; then
            echo ""
            echo -e "${GREEN}✅ 基础设施服务启动完成${NC}"
            return 0
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    echo ""
    echo -e "${YELLOW}⚠️  基础设施服务启动可能需要更多时间，继续启动应用服务...${NC}"
}

echo -e "${BLUE}🚀 DeFi聚合器开发环境启动${NC}"
echo "=================================="
echo ""
echo "请选择启动模式："
echo "1) 🐳 Docker Compose模式 (所有服务容器化)"
echo "2) 🛠️  混合模式 - 后台 (数据库+Redis容器化，服务后台运行)"
echo "3) 🔍 混合模式 - 前台 (数据库+Redis容器化，服务前台多终端运行)"
echo ""
read -p "请输入选择 (1, 2 或 3): " START_MODE

case $START_MODE in
    1)
        echo -e "${GREEN}选择了Docker Compose模式${NC}"
        MODE="docker"
        ;;
    2)
        echo -e "${GREEN}选择了混合模式 - 后台运行${NC}"
        MODE="hybrid-background"
        ;;
    3)
        echo -e "${GREEN}选择了混合模式 - 前台多终端${NC}"
        MODE="hybrid-foreground"
        ;;
    *)
        echo -e "${RED}无效选择，默认使用Docker Compose模式${NC}"
        MODE="docker"
        ;;
esac
echo ""

# 加载全局环境变量
if [[ -f "env.global" ]]; then
    source env.global
    echo -e "${GREEN}✅ 已加载全局环境变量${NC}"
else
    echo -e "${RED}❌ 未找到env.global文件${NC}"
    exit 1
fi
echo ""

# 检查必要文件
echo -e "${YELLOW}📋 检查配置文件...${NC}"

if [[ ! -f "docker-compose.yml" ]]; then
    echo -e "${RED}❌ 缺少 docker-compose.yml 文件${NC}"
    exit 1
fi

# 检查配置文件是否存在
echo -e "${YELLOW}🔧 检查配置文件...${NC}"

# 检查必要的配置文件
MISSING_FILES=()

if [[ ! -f "backend/business-logic/.env" ]]; then
    MISSING_FILES+=("backend/business-logic/.env")
fi

if [[ ! -f "backend/smart-router/.env" ]]; then
    MISSING_FILES+=("backend/smart-router/.env")
fi

if [[ ! -f "backend/api-gateway/.env" ]]; then
    MISSING_FILES+=("backend/api-gateway/.env")
fi

if [[ ! -f "frontend/.env.local" ]]; then
    MISSING_FILES+=("frontend/.env.local")
fi

if [[ ${#MISSING_FILES[@]} -gt 0 ]]; then
    echo -e "${RED}❌ 缺少以下配置文件：${NC}"
    for file in "${MISSING_FILES[@]}"; do
        echo "  - $file"
    done
    echo ""
    echo -e "${YELLOW}请先创建配置文件：${NC}"
    echo "cp backend/business-logic/env.example backend/business-logic/.env"
    echo "cp backend/smart-router/env.example backend/smart-router/.env"
    echo "cp backend/api-gateway/env.example backend/api-gateway/.env"
    echo "cp frontend/env.example frontend/.env.local"
    exit 1
fi

echo -e "${GREEN}✅ 配置文件检查完成${NC}"

# 加载环境变量
echo -e "${YELLOW}🔧 加载环境变量...${NC}"
set -a  # 自动导出所有变量
source ./env.global
set +a

# 验证关键环境变量
echo -e "${YELLOW}🔍 验证关键配置...${NC}"

if [[ -z "$FRONTEND_PORT" ]]; then
    echo -e "${RED}❌ FRONTEND_PORT 未设置${NC}"
    exit 1
fi

if [[ -z "$API_GATEWAY_PORT" ]]; then
    echo -e "${RED}❌ API_GATEWAY_PORT 未设置${NC}"
    exit 1
fi

if [[ -z "$BUSINESS_LOGIC_PORT" ]]; then
    echo -e "${RED}❌ BUSINESS_LOGIC_PORT 未设置${NC}"
    exit 1
fi

if [[ -z "$SMART_ROUTER_PORT" ]]; then
    echo -e "${RED}❌ SMART_ROUTER_PORT 未设置${NC}"
    exit 1
fi

if [[ -z "$DB_NAME" || -z "$DB_USER" || -z "$DB_PASSWORD" ]]; then
    echo -e "${RED}❌ 数据库配置不完整${NC}"
    exit 1
fi

echo -e "${GREEN}✅ 环境变量验证通过${NC}"
echo "  - 前端端口: $FRONTEND_PORT"
echo "  - API网关端口: $API_GATEWAY_PORT" 
echo "  - 业务逻辑端口: $BUSINESS_LOGIC_PORT"
echo "  - 智能路由端口: $SMART_ROUTER_PORT"
echo "  - 数据库: $DB_NAME"

# 检查端口占用（仅用于信息展示）
echo -e "${YELLOW}🔍 检查端口占用...${NC}"
PORTS=($FRONTEND_PORT $API_GATEWAY_PORT $BUSINESS_LOGIC_PORT $SMART_ROUTER_PORT 5432 6379)

for port in "${PORTS[@]}"; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  端口 $port 已被占用${NC}"
    fi
done

# 根据模式决定是否停止现有服务
if [[ $MODE == "docker" ]]; then
    # Docker Compose模式：停止所有服务重新启动
    echo -e "${YELLOW}⏹️  停止现有服务...${NC}"
    docker-compose down --remove-orphans
elif [[ $MODE == "hybrid-background" ]] || [[ $MODE == "hybrid-foreground" ]]; then
    # 混合模式：智能检查，只在必要时停止服务
    echo -e "${BLUE}🔍 混合模式：智能检查基础设施状态${NC}"
    # 注意：不在这里停止服务，让后面的智能检查逻辑处理
fi

if [[ $MODE == "docker" ]]; then
    # Docker Compose模式：启动所有服务
    echo -e "${YELLOW}🐳 启动Docker Compose服务...${NC}"
    
    # 首先启动基础设施服务
    echo "启动数据库和缓存服务..."
    docker-compose up -d postgres redis
    
    # 等待数据库启动
    echo "等待数据库启动..."
    sleep 10
    
    # 启动后端服务
    echo "启动后端服务..."
    docker-compose up -d smart-router business-logic api-gateway
    
    # 等待后端服务启动
    echo "等待后端服务启动..."
    sleep 15
    
    # 启动前端服务
    echo "启动前端服务..."
    docker-compose up -d frontend

elif [[ $MODE == "hybrid-background" ]]; then
    # 混合模式：数据库+Redis用Docker，服务本地启动
    echo -e "${YELLOW}🛠️  启动混合模式服务...${NC}"
    
    # 检查并启动基础设施服务
    if check_infrastructure_status; then
        echo "基础设施已就绪，直接启动应用服务..."
        sleep 2
    else
        start_infrastructure
    fi
    
    # 检查Go和Node.js环境
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ 未找到Go环境，请先安装Go${NC}"
        exit 1
    fi
    
    if ! command -v node &> /dev/null; then
        echo -e "${RED}❌ 未找到Node.js环境，请先安装Node.js${NC}"
        exit 1
    fi
    
    # 启动后端服务（本地方式）
    echo -e "${YELLOW}🚀 本地启动后端服务...${NC}"
    
    # 启动智能路由服务
    echo "启动智能路由服务 (端口: $SMART_ROUTER_PORT)..."
    cd backend/smart-router
    nohup go run cmd/main.go > ../../logs/smart-router.log 2>&1 &
    SMART_ROUTER_PID=$!
    echo "智能路由服务 PID: $SMART_ROUTER_PID"
    cd ../..
    
    # 等待智能路由服务启动
    sleep 5
    
    # 启动业务逻辑服务
    echo "启动业务逻辑服务 (端口: $BUSINESS_LOGIC_PORT)..."
    cd backend/business-logic
    nohup go run cmd/main.go > ../../logs/business-logic.log 2>&1 &
    BUSINESS_LOGIC_PID=$!
    echo "业务逻辑服务 PID: $BUSINESS_LOGIC_PID"
    cd ../..
    
    # 等待业务逻辑服务启动
    sleep 5
    
    # 启动API网关服务
    echo "启动API网关服务 (端口: $API_GATEWAY_PORT)..."
    cd backend/api-gateway
    nohup go run cmd/main.go > ../../logs/api-gateway.log 2>&1 &
    API_GATEWAY_PID=$!
    echo "API网关服务 PID: $API_GATEWAY_PID"
    cd ../..
    
    # 等待API网关启动
    sleep 5
    
    # 启动前端服务
    echo "启动前端服务 (端口: $FRONTEND_PORT)..."
    cd frontend
    
    # 检查是否已安装依赖
    if [[ ! -d "node_modules" ]]; then
        echo "安装前端依赖..."
        npm install
    fi
    
    nohup npm run dev > ../logs/frontend.log 2>&1 &
    FRONTEND_PID=$!
    echo "前端服务 PID: $FRONTEND_PID"
    cd ..
    
    # 保存进程ID到文件
    mkdir -p logs
    echo "$SMART_ROUTER_PID" > logs/smart-router.pid
    echo "$BUSINESS_LOGIC_PID" > logs/business-logic.pid  
    echo "$API_GATEWAY_PID" > logs/api-gateway.pid
    echo "$FRONTEND_PID" > logs/frontend.pid
    
    echo -e "${GREEN}✅ 所有服务已启动，PID已保存到logs目录${NC}"
    echo ""
    echo -e "${YELLOW}📋 停止服务命令：${NC}"
    echo "  停止所有本地服务: ./stop-local-services.sh"
    echo "  或手动停止: kill \$(cat logs/*.pid)"
    echo "  停止数据库: docker-compose stop postgres redis"

elif [[ $MODE == "hybrid-foreground" ]]; then
    # 混合模式 - 前台多终端：数据库+Redis用Docker，服务在多个终端前台运行
    echo -e "${YELLOW}🔍 启动前台多终端模式...${NC}"
    
    # 检查并启动基础设施服务
    if check_infrastructure_status; then
        echo "基础设施已就绪，直接启动应用服务..."
        sleep 2
    else
        start_infrastructure
    fi
    
    # 检查Go和Node.js环境
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ 未找到Go环境，请先安装Go${NC}"
        exit 1
    fi
    
    if ! command -v node &> /dev/null; then
        echo -e "${RED}❌ 未找到Node.js环境，请先安装Node.js${NC}"
        exit 1
    fi
    
    # 获取项目根目录的绝对路径
    PROJECT_ROOT=$(pwd)
    
    # 检测终端类型
    echo "检测终端应用..."
    TERMINAL_APP=""
    if pgrep -f "iTerm" > /dev/null || [[ "$TERM_PROGRAM" == "iTerm.app" ]]; then
        TERMINAL_APP="iterm"
        echo "检测到 iTerm2 🚀"
    else
        TERMINAL_APP="terminal"
        echo "使用原生终端 📱"
    fi
    
    echo -e "${YELLOW}🚀 在多个终端启动服务...${NC}"
    
    if [[ $TERMINAL_APP == "iterm" ]]; then
        # iTerm2 模式 - 在同一窗口创建多个标签页
        echo "在iTerm2中创建服务标签页..."
        
        # 启动智能路由服务 - 新标签页
        echo "创建智能路由服务标签页 (端口: $SMART_ROUTER_PORT)..."
        osascript -e "
        tell application \"iTerm\"
            tell current window
                create tab with default profile
                tell current session
                    set name to \"🧠 Smart Router\"
                    write text \"cd '$PROJECT_ROOT/backend/smart-router'\"
                    write text \"source '$PROJECT_ROOT/env.global'\"
                    write text \"export PORT=$SMART_ROUTER_PORT\"
                    write text \"export APP_ENV=development\"
                    write text \"export LOG_LEVEL=info\"
                    write text \"export DEBUG=true\"
                    write text \"echo '🧠 智能路由服务启动中 (端口: $SMART_ROUTER_PORT)...'\"
                    write text \"go run cmd/main.go\"
                end tell
            end tell
        end tell"
        
        sleep 2
        
        # 启动业务逻辑服务 - 新标签页
        echo "创建业务逻辑服务标签页 (端口: $BUSINESS_LOGIC_PORT)..."
        osascript -e "
        tell application \"iTerm\"
            tell current window
                create tab with default profile
                tell current session
                    set name to \"💼 Business Logic\"
                    write text \"cd '$PROJECT_ROOT/backend/business-logic'\"
                    write text \"source '$PROJECT_ROOT/env.global'\"
                    write text \"export PORT=$BUSINESS_LOGIC_PORT\"
                    write text \"export APP_ENV=development\"
                    write text \"export LOG_LEVEL=info\"
                    write text \"export DEBUG=true\"
                    write text \"echo '💼 业务逻辑服务启动中 (端口: $BUSINESS_LOGIC_PORT)...'\"
                    write text \"go run cmd/main.go\"
                end tell
            end tell
        end tell"
        
        sleep 2
        
        # 启动API网关服务 - 新标签页
        echo "创建API网关服务标签页 (端口: $API_GATEWAY_PORT)..."
        osascript -e "
        tell application \"iTerm\"
            tell current window
                create tab with default profile
                tell current session
                    set name to \"🚪 API Gateway\"
                    write text \"cd '$PROJECT_ROOT/backend/api-gateway'\"
                    write text \"source '$PROJECT_ROOT/env.global'\"
                    write text \"export PORT=$API_GATEWAY_PORT\"
                    write text \"export APP_ENV=development\"
                    write text \"export LOG_LEVEL=info\"
                    write text \"export DEBUG=true\"
                    write text \"export JWT_SECRET_KEY=dev-jwt-secret-key-change-in-production\"
                    write text \"export CORS_ALLOWED_ORIGINS=http://localhost:5175,http://localhost:5176,http://localhost:5177,http://localhost:5178\"
                    write text \"export BUSINESS_LOGIC_TARGETS=http://localhost:5177\"
                    write text \"export SMART_ROUTER_TARGETS=http://localhost:5178\"
                    write text \"echo '🚪 API网关服务启动中 (端口: $API_GATEWAY_PORT)...'\"
                    write text \"go run cmd/main.go\"
                end tell
            end tell
        end tell"
        
        sleep 2
        
        # 启动前端服务 - 新标签页
        echo "创建前端服务标签页 (端口: $FRONTEND_PORT)..."
        osascript -e "
        tell application \"iTerm\"
            tell current window
                create tab with default profile
                tell current session
                    set name to \"🌐 Frontend\"
                    write text \"cd '$PROJECT_ROOT/frontend'\"
                    write text \"echo '🌐 前端服务启动中 (端口: $FRONTEND_PORT)...'\"
                    write text \"if [ ! -d node_modules ]; then echo '安装依赖中...' && npm install; fi\"
                    write text \"npm run dev\"
                end tell
            end tell
        end tell"
        
    else
        # 原生终端模式 - 创建多个窗口
        echo "在原生终端中创建服务窗口..."
        
        # 启动智能路由服务 - 新终端窗口
        echo "启动智能路由服务终端 (端口: $SMART_ROUTER_PORT)..."
        osascript -e "
        tell application \"Terminal\"
            do script \"cd '$PROJECT_ROOT/backend/smart-router' && source '$PROJECT_ROOT/env.global' && export PORT=$SMART_ROUTER_PORT && export APP_ENV=development && export LOG_LEVEL=info && export DEBUG=true && echo '🧠 智能路由服务启动中...' && go run cmd/main.go\"
        end tell"
        
        sleep 2
        
        # 启动业务逻辑服务 - 新终端窗口
        echo "启动业务逻辑服务终端 (端口: $BUSINESS_LOGIC_PORT)..."
        osascript -e "
        tell application \"Terminal\"
            do script \"cd '$PROJECT_ROOT/backend/business-logic' && source '$PROJECT_ROOT/env.global' && export PORT=$BUSINESS_LOGIC_PORT && export APP_ENV=development && export LOG_LEVEL=info && export DEBUG=true && echo '💼 业务逻辑服务启动中...' && go run cmd/main.go\"
        end tell"
        
        sleep 2
        
        # 启动API网关服务 - 新终端窗口
        echo "启动API网关服务终端 (端口: $API_GATEWAY_PORT)..."
        osascript -e "
        tell application \"Terminal\"
            do script \"cd '$PROJECT_ROOT/backend/api-gateway' && source '$PROJECT_ROOT/env.global' && export PORT=$API_GATEWAY_PORT && export APP_ENV=development && export LOG_LEVEL=info && export DEBUG=true && export JWT_SECRET_KEY=dev-jwt-secret-key-change-in-production && export CORS_ALLOWED_ORIGINS=http://localhost:5175,http://localhost:5176,http://localhost:5177,http://localhost:5178 && export BUSINESS_LOGIC_TARGETS=http://localhost:5177 && export SMART_ROUTER_TARGETS=http://localhost:5178 && echo '🚪 API网关服务启动中...' && go run cmd/main.go\"
        end tell"
        
        sleep 2
        
        # 启动前端服务 - 新终端窗口
        echo "启动前端服务终端 (端口: $FRONTEND_PORT)..."
        osascript -e "
        tell application \"Terminal\"
            do script \"cd '$PROJECT_ROOT/frontend' && echo '🌐 前端服务启动中...' && if [ ! -d node_modules ]; then echo '安装依赖中...' && npm install; fi && npm run dev\"
        end tell"
    fi
    
    if [[ $TERMINAL_APP == "iterm" ]]; then
        echo -e "${GREEN}✅ 所有服务已在iTerm2标签页中启动${NC}"
        echo ""
        echo -e "${BLUE}📋 iTerm2标签页模式说明：${NC}"
        echo "  - 每个服务在独立的标签页中运行"
        echo "  - 标签页已设置有意义的名称 (🧠 Smart Router, 💼 Business Logic...)"
        echo "  - 可以直接看到每个服务的实时日志"
        echo "  - 在对应标签页按 Ctrl+C 可停止单个服务"
        echo "  - 关闭标签页会停止对应的服务"
        echo ""
        echo -e "${YELLOW}🔧 iTerm2优势：${NC}"
        echo "  - 所有服务在同一窗口，便于管理"
        echo "  - 标签页名称清晰标识每个服务"
        echo "  - 可以快速在服务间切换 (Cmd+数字键)"
        echo "  - 支持分屏查看多个服务日志"
    else
        echo -e "${GREEN}✅ 所有服务已在独立终端窗口中启动${NC}"
        echo ""
        echo -e "${BLUE}📋 多窗口模式说明：${NC}"
        echo "  - 每个服务在独立的终端窗口中运行"
        echo "  - 可以直接看到每个服务的实时日志"
        echo "  - 在对应终端按 Ctrl+C 可停止单个服务"
        echo "  - 关闭终端窗口也会停止对应的服务"
    fi
    echo ""
    echo -e "${YELLOW}🔧 通用管理提示：${NC}"
    echo "  - 修改代码后，Go服务会自动重启"
    echo "  - 前端支持热重载，修改即生效"
    echo "  - 数据库和Redis仍在Docker容器中"
    echo "  - 停止数据库: docker-compose stop postgres redis"
    
    # 创建一个简单的停止脚本用于前台模式
    cat > stop-foreground-services.sh << 'EOF'
#!/bin/bash
echo "🛑 停止前台多终端服务..."
echo "请手动关闭各个服务终端窗口，或在对应终端按 Ctrl+C"
echo ""
read -p "是否停止数据库和Redis服务？(y/N): " STOP_DOCKER
if [[ $STOP_DOCKER == "y" || $STOP_DOCKER == "Y" ]]; then
    echo "停止数据库和Redis服务..."
    docker-compose stop postgres redis
    echo "✅ 数据库和Redis已停止"
fi
EOF
    chmod +x stop-foreground-services.sh
    
    echo "  - 创建了停止脚本: ./stop-foreground-services.sh"
fi

echo -e "${GREEN}🎉 所有服务启动完成！${NC}"
echo ""
echo "📱 访问地址:"
echo "  🌐 前端应用: http://localhost:$FRONTEND_PORT"
echo "  🚪 API网关: http://localhost:$API_GATEWAY_PORT"
echo "  💼 业务逻辑: http://localhost:$BUSINESS_LOGIC_PORT"
echo "  🧠 智能路由: http://localhost:$SMART_ROUTER_PORT"
echo ""
echo "🔍 管理命令:"
echo "  查看状态: docker-compose ps"
echo "  查看日志: docker-compose logs -f"
echo "  停止服务: docker-compose down"
echo ""
echo "⏱️  等待服务完全启动..."
sleep 5

# 健康检查
echo -e "${YELLOW}🏥 健康检查...${NC}"
if [[ $MODE == "docker" ]]; then
    sleep 10
elif [[ $MODE == "hybrid-background" ]]; then
    sleep 15  # 后台本地服务需要更多时间启动
elif [[ $MODE == "hybrid-foreground" ]]; then
    sleep 20  # 前台多终端服务需要更多时间启动
fi

HEALTH_CHECKS=(
    "http://localhost:$API_GATEWAY_PORT/health|API网关"
    "http://localhost:$BUSINESS_LOGIC_PORT/health|业务逻辑"
    "http://localhost:$SMART_ROUTER_PORT/health|智能路由"
)

for check in "${HEALTH_CHECKS[@]}"; do
    IFS='|' read -r url name <<< "$check"
    if curl -sf "$url" >/dev/null 2>&1; then
        echo -e "  $name: ${GREEN}🟢 健康${NC}"
    else
        echo -e "  $name: ${YELLOW}🟡 启动中...${NC}"
    fi
done

if [[ $MODE == "hybrid-background" ]]; then
    echo ""
    echo -e "${BLUE}📋 本地服务管理：${NC}"
    echo "  查看日志: tail -f logs/[service-name].log"
    echo "  查看进程: ps aux | grep 'go run'"
    echo "  停止服务: ./stop-local-services.sh"
    echo ""
    echo -e "${YELLOW}🔧 调试提示：${NC}"
    echo "  - 各服务日志保存在 logs/ 目录"
    echo "  - 可以直接修改代码，服务会自动重启"
    echo "  - 数据库和Redis仍然在Docker容器中运行"
elif [[ $MODE == "hybrid-foreground" ]]; then
    echo ""
    if [[ $TERMINAL_APP == "iterm" ]]; then
        echo -e "${BLUE}📋 iTerm2标签页管理：${NC}"
        echo "  查看服务: 每个服务在独立标签页中"
        echo "  切换服务: Cmd+数字键或点击标签页"
        echo "  停止单个服务: 在对应标签页按 Ctrl+C"
        echo "  停止所有服务: ./stop-foreground-services.sh"
        echo ""
        echo -e "${YELLOW}🔧 iTerm2调试优势：${NC}"
        echo "  - 实时查看每个服务的日志输出"
        echo "  - 标签页名称清晰标识服务状态"
        echo "  - 可以分屏同时查看多个服务"
        echo "  - 修改代码后服务自动重启"
        echo "  - 前端热重载，修改即生效"
    else
        echo -e "${BLUE}📋 多窗口模式管理：${NC}"
        echo "  查看服务: 每个服务在独立终端窗口中"
        echo "  停止单个服务: 在对应终端按 Ctrl+C"
        echo "  停止所有服务: ./stop-foreground-services.sh"
        echo ""
        echo -e "${YELLOW}🔧 调试优势：${NC}"
        echo "  - 实时查看每个服务的日志输出"
        echo "  - 修改代码后服务自动重启"
        echo "  - 可以独立调试和重启单个服务"
        echo "  - 前端热重载，修改即生效"
    fi
fi

echo -e "${GREEN}✨ 开发环境就绪！${NC}"
