# 开发阶段
FROM golang:1.21-alpine AS development

# 安装必要的工具
RUN apk add --no-cache \
    git \
    curl \
    bash \
    make \
    gcc \
    musl-dev

# 安装热重载工具 air
RUN go install github.com/cosmtrek/air@latest

# 安装调试器
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# 设置工作目录
WORKDIR /app

# 默认命令：使用 air 进行热重载开发
CMD ["air"]

# ============================================

# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o stock-services ./cmd/main.go

# ============================================

# 生产阶段
FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/stock-services .

# 复制配置文件
COPY .env.example .env

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["/app/stock-services"]