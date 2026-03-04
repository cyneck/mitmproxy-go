# mitmproxy-go

使用 Go 语言实现的轻量级 HTTP/HTTPS 流量劫持工具。

## 功能特性

- 指定端口的 HTTP/HTTPS 代理
- URL 路径匹配（支持正则表达式）
- 自定义 JSON 响应返回
- 多种代理模式支持（regular / transparent）
- 配置文件热加载

## 快速开始

```bash
# 编译
go build -o mitmproxy-go

# 运行
./mitmproxy-go --config config.json
```

## 配置说明

```json
{
  "listen_port": 8082,
  "proxy_mode": "regular",
  "intercept_paths": ["/api/users"],
  "responses": {
    "/api/users": "{\"code\": 0, \"data\": {\"name\": \"hacked\"}}"
  },
  "default_response": "{\"error\": \"not intercepted\"}",
  "verbose": true
}
```

| 字段 | 说明 |
|------|------|
| `listen_port` | 代理服务器监听端口 |
| `proxy_mode` | 代理模式：`regular` 或 `transparent` |
| `intercept_paths` | 需要拦截的路径（支持正则） |
| `responses` | 拦截后返回的自定义响应 |
| `default_response` | 未匹配时的默认响应 |

## 使用示例

### 1. 本地测试（单终端）

```bash
# 终端 1：启动目标服务器
go run ./testserver/server.go 8888

# 终端 2：启动代理服务器
go run main.go --config config-8888.json

# 终端 3：测试劫持
curl -x http://127.0.0.1:8082 http://127.0.0.1:8888/api/users
```

### 2. Docker 部署

```bash
# 构建镜像
docker build -t mitmproxy-go .

# 运行
docker run -d --name mitm-proxy -p 8082:8082 mitmproxy-go

# 测试
curl -x http://localhost:8082 http://example.com/api/users

# 清理
docker stop mitm-proxy && docker rm mitm-proxy
```

### 3. 透明代理（Linux）

```bash
sudo iptables -t nat -A PREROUTING -p tcp --dport 8888 -j REDIRECT --to-port 8082
./mitmproxy-go --mode transparent
```

## 项目结构

```
mitmproxy-go/
├── main.go              # 主入口
├── config.json          # 配置文件
├── config-8888.json     # 测试配置示例
├── config/              # 配置管理
├── interceptor/         # 流量拦截器
├── proxy/               # 代理服务器
└── testserver/          # 测试服务器
```

## 适用场景

- API Mock 测试
- 离线环境部署
- 安全测试
- 嵌入式系统