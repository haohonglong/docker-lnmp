# Go Stock Services c

.PHONY: help build run test clean dev debug migrate lint

# 帮助信息
help:
	@echo "Go Stock Services 开发命令"
	@echo ""
	@echo "用法:"
	@echo "  make <command>"
	@echo ""
	@echo "命令:"
	@echo "  build      - 编译应用"
	@echo "  run        - 运行应用"
	@echo "  dev        - 启动开发容器"
	@echo "  debug      - 启动调试模式"
	@echo "  test       - 运行测试"
	@echo "  clean      - 清理编译文件"
	@echo "  lint       - 代码检查"
	@echo "  migrate    - 数据库迁移"
	@echo "  docker-up  - 启动开发容器"
	@echo "  docker-down - 停止开发容器"
	@echo "  docker-logs - 查看容器日志"

# 编译应用
build:
	go build -o bin/stock-services ./cmd/main.go

# 运行应用
run: build
	./bin/stock-services

# 启动开发容器
dev:
	docker-compose -f docker-compose.dev.yml up -d

# 进入开发容器
dev-shell:
	docker-compose -f docker-compose.dev.yml exec go-stock-dev sh

# 启动调试模式
debug:
	dlv debug ./cmd/main.go

# 运行测试
test:
	go test -v ./...

# 清理编译文件
clean:
	rm -rf bin/
	rm -rf coverage.out
	go clean -cache

# 代码检查
lint:
	go vet ./...
	staticcheck ./...

# 数据库迁移
migrate:
	go run scripts/migrate.go

# 格式化代码
fmt:
	go fmt ./...

# Docker 相关命令
docker-up:
	docker-compose -f docker-compose.dev.yml up -d

docker-down:
	docker-compose -f docker-compose.dev.yml down

docker-logs:
	docker-compose -f docker-compose.dev.yml logs -f

docker-restart:
	docker-compose -f docker-compose.dev.yml restart

# 安装依赖
deps:
	go mod download
	go mod tidy

# 生成 swagger 文档（如果需要）
swagger:
	swag init -g cmd/main.go -o docs/swagger