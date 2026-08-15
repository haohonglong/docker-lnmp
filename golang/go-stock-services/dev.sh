#!/bin/bash

# Go Stock Services 开发环境脚本
# 用法: ./dev.sh [command]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Go Stock Services 开发环境${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}➜ $1${NC}"
}

# 检查 Docker 是否运行
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker 未运行，请先启动 Docker"
        exit 1
    fi
}

# 检查网络是否存在
check_network() {
    if ! docker network ls | grep -q dnmp_default; then
        print_error "dnmp_default 网络不存在，请先启动 dnmp 服务"
        print_info "请先运行: cd /workspace/docker/dnmp && docker-compose up -d"
        exit 1
    fi
}

# 启动开发环境
start() {
    print_header
    print_info "启动 Go 股票服务开发环境..."
    
    check_docker
    check_network
    
    # 启动容器
    docker-compose -f docker-compose.dev.yml up -d
    
    if [ $? -eq 0 ]; then
        print_success "开发容器启动成功"
        print_info "容器名称: go-stock-dev"
        print_info "HTTP API: http://localhost:8080"
        print_info "调试端口: 2345"
        echo ""
        print_info "常用命令:"
        print_info "  ./dev.sh shell    - 进入容器"
        print_info "  ./dev.sh logs     - 查看日志"
        print_info "  ./dev.sh stop     - 停止容器"
        print_info "  ./dev.sh restart  - 重启容器"
    else
        print_error "启动失败"
        exit 1
    fi
}

# 停止开发环境
stop() {
    print_info "停止开发容器..."
    docker-compose -f docker-compose.dev.yml down
    print_success "容器已停止"
}

# 重启开发环境
restart() {
    stop
    sleep 2
    start
}

# 进入容器
shell() {
    print_info "进入开发容器..."
    docker-compose -f docker-compose.dev.yml exec go-stock-dev sh
}

# 查看日志
logs() {
    print_info "查看容器日志..."
    docker-compose -f docker-compose.dev.yml logs -f
}

# 构建镜像
build() {
    print_info "构建 Docker 镜像..."
    docker-compose -f docker-compose.dev.yml build
    print_success "镜像构建完成"
}

# 清理环境
clean() {
    print_info "清理开发环境..."
    
    # 停止容器
    docker-compose -f docker-compose.dev.yml down
    
    # 删除卷
    docker volume rm -f go-stock-services_go-modules 2>/dev/null || true
    docker volume rm -f go-stock-services_go-build 2>/dev/null || true
    
    # 清理本地编译文件
    rm -rf bin/ coverage.out
    
    print_success "环境清理完成"
}

# 健康检查
health() {
    print_info "检查服务健康状态..."
    
    if curl -s http://localhost:8080/health > /dev/null; then
        print_success "服务运行正常"
    else
        print_error "服务异常"
    fi
}

# 初始化项目
init() {
    print_info "初始化 Go 项目..."
    
    # 进入容器初始化
    docker-compose -f docker-compose.dev.yml run --rm go-stock-dev sh -c "\
        cd /app && \
        go mod init go-stock-services && \
        go mod tidy && \
        echo '项目初始化完成'"
    
    print_success "项目初始化完成"
}

# 显示帮助
help() {
    print_header
    echo "可用命令:"
    echo "  start     - 启动开发环境"
    echo "  stop      - 停止开发环境"
    echo "  restart   - 重启开发环境"
    echo "  shell     - 进入容器终端"
    echo "  logs      - 查看容器日志"
    echo "  build     - 构建 Docker 镜像"
    echo "  clean     - 清理开发环境"
    echo "  health    - 检查服务健康状态"
    echo "  init      - 初始化 Go 项目"
    echo "  help      - 显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  ./dev.sh start   # 启动开发环境"
    echo "  ./dev.sh shell   # 进入容器"
    echo "  ./dev.sh logs    # 查看日志"
}

# 主函数
main() {
    case "$1" in
        start)
            start
            ;;
        stop)
            stop
            ;;
        restart)
            restart
            ;;
        shell)
            shell
            ;;
        logs)
            logs
            ;;
        build)
            build
            ;;
        clean)
            clean
            ;;
        health)
            health
            ;;
        init)
            init
            ;;
        help|--help|-h)
            help
            ;;
        *)
            if [ -z "$1" ]; then
                start
            else
                print_error "未知命令: $1"
                echo ""
                help
                exit 1
            fi
            ;;
    esac
}

# 执行主函数
main "$@"