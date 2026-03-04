FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装 build 依赖
RUN apk add --no-cache git

# 复制源码
COPY go.mod go.sum ./
COPY config ./config
COPY interceptor ./interceptor
COPY proxy ./proxy
COPY main.go ./

# 下载依赖
RUN go mod download

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -o mitmproxy-go main.go

# 运行镜像
FROM alpine:latest

RUN apk add --no-cache curl

WORKDIR /app

# 从 builder 阶段复制二进制文件
COPY --from=builder /app/mitmproxy-go .
COPY config.json .
COPY config-8888.json .

EXPOSE 8082

CMD ["./mitmproxy-go", "--config", "config-8888.json"]