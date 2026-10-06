# Go 股票服务 - 开发环境

将股票价格监听、委托检查等并发任务从 PHP 迁移到 Go 的开发和测试环境。

## 项目结构
```
go-stock-services/
├── cmd/                 # 应用入口
├── internal/           # 内部包（不对外暴露）
│   ├── config/        # 配置管理
│   ├── database/      # 数据库连接
│   ├── services/      # 业务服务
│   ├── api/          # HTTP API
│   └── models/       # 数据模型
├── pkg/               # 可复用的包
├── scripts/           # 脚本文件
├── tests/             # 测试文件
├── go.mod
├── go.sum
├── .env.example
├── docker-compose.dev.yml
├── Dockerfile
├── Makefile
└── README.md
```

## 快速开始

### 1. 启动开发环境
```bash
# 进入项目目录
cd /workspace/docker/go-stock-services

# 启动开发容器
make up

# 或者直接使用脚本
./dev.sh
```

### 2. 进入容器开发
```bash
# 进入容器
make exec
# 重新编译
docker exec go-stock-dev sh -c "cd /app && go build ./... && echo BUILD_OK" 2>&1 | tail -20
# 编译 + 重启服务（最常用：改完代码 → 编译 → 容器重启自动 reload）
docker exec go-stock-dev sh -c "cd /app && go build ./... && echo BUILD_OK" && docker restart go-stock-dev
# 或者
docker exec -it go-stock-dev sh
```

### 3. 容器内操作
```bash
# 初始化 Go modules（如果未初始化）
go mod init go-stock-services
go mod tidy

# 运行应用
go run cmd/main.go

# 或者使用热重载
make watch
```

### 4. 测试
```bash
# 运行单元测试
make test

# 运行集成测试
make test-integration

# 检查代码质量
make lint
```

## 开发工作流

### 环境配置
1. 复制环境变量文件：
   ```bash
   cp .env.example .env
   ```

2. 编辑 `.env` 文件，配置数据库连接：
   ```
   MYSQL_HOST=mysql
   MYSQL_PORT=3306
   MYSQL_DATABASE=blog
   MYSQL_USER=lam
   MYSQL_PASSWORD=wanelslbmy
   
   REDIS_HOST=redis
   REDIS_PORT=6379
   REDIS_DB=0
   
   APP_PORT=8080
   APP_ENV=development
   ```

### 常用命令
```bash
# 启动开发环境
make up

# 停止开发环境
make down

# 查看日志
make logs

# 重新构建镜像
make rebuild

# 清理缓存
make clean

# 格式化代码
make format

# 代码检查
make lint
```

## 服务架构

### 核心服务
1. **委托监听服务** - 30秒轮询检查委托匹配
2. **价格获取服务** - 定时获取股票价格
3. **委托清理服务** - 每日清理过期委托

### 数据流
```
前端 → Nginx → Go API → Redis/MySQL
                    ↓
                调用 PHP API（执行交易）
```

### API 兼容性
- 前端 API 保持不变
- Nginx 根据路径路由到 Go 或 PHP
- Go 服务只处理高频并发任务

## 网络配置

Go 容器加入 `dnmp_default` 网络，可以访问：
- MySQL: `mysql:3306`
- Redis: `redis:6379`
- PHP: `php:9000`

## 监控和调试

### 健康检查
```bash
curl http://localhost:8080/health
```

### 指标监控
```bash
curl http://localhost:8080/metrics
```

### 调试端口
- Delve 调试器: `2345` 端口
- HTTP API: `8080` 端口

## 部署说明

### 开发环境
使用 `docker-compose.dev.yml` 启动开发容器，代码实时挂载。

### 生产环境
1. 构建生产镜像：
   ```bash
   make build
   ```

2. 推送到镜像仓库：
   ```bash
   make push
   ```

3. 部署到生产环境（使用生产环境的 docker-compose.yml）

## 注意事项

1. **数据一致性**：Go 服务只读 Redis，写操作通过 API 调用 PHP
2. **错误处理**：所有错误需要记录日志并告警
3. **监控**：需要监控服务健康状态和性能指标
4. **回滚**：Nginx 配置回滚即可恢复 PHP 服务

## 开发进度

### 已完成
- [x] 基础项目结构
- [x] Docker 开发环境配置
- [x] 数据库连接配置
- [x] 数据模型定义

### 待完成
- [ ] 委托监听服务实现
- [ ] 价格获取服务实现
- [ ] HTTP API 接口
- [ ] 与 PHP API 集成
- [ ] 监控和告警
- [ ] 性能测试
