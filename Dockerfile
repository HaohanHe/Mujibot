# Mujibot Dockerfile
# 多阶段构建，最小化镜像大小

# 构建阶段
FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /build

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建（静态链接）
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags=-static" \
    -trimpath \
    -o mujibot \
    ./cmd/mujibot

# 运行阶段（使用scratch最小化镜像）
FROM scratch

# 从builder复制证书（用于HTTPS请求）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 复制二进制文件
COPY --from=builder /build/mujibot /mujibot

# 复制默认配置
COPY config.json5.example /config.json5

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/mujibot", "--health-check"] || exit 1

# 入口点
ENTRYPOINT ["/mujibot"]
CMD ["--config", "/config.json5"]
