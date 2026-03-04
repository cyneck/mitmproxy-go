# mitmproxy-go

使用 Go 语言实现的流量劫持工具，核心功能与 mitmproxy 相同，但更轻量、部署更简单。

## 功能特性

- ✅ 指定端口的 HTTP/HTTPS 代理
- ✅ 精确 URL 路径匹配（支持正则表达式）
- ✅ 自定义 JSON 响应返回
- ✅ 多种代理模式支持（regular / transparent）
- ✅ 详细日志记录
- ✅ 配置文件热加载
- ✅ 跨平台编译（Linux/macOS/Windows）

## 快速开始

### 1. 安装

```bash
# 克隆项目
git clone https://github.com/cyneck/mitmproxy-go
cd mitmproxy-go

# 下载依赖
go mod tidy

# 编译
go build -o mitmproxy-go

# 或者直接运行
go run main.go --port 8082
```

### 2. 配置

编辑 `config.json` 文件：

```json
{
  "listen_port": 8082,
  "proxy_mode": "regular",
  "intercept_paths": [
    "/api/users"
  ],
  "responses": {
    "/api/users": "{\"code\": 0, \"data\": [{\"id\": 1, \"name\": \"hacked\"}]}"
  },
  "default_response": "{\"error\": \"not intercepted\"}",
  "verbose": true
}
```

### 3. 运行

```bash
# 使用默认配置
./mitmproxy-go

# 指定端口
./mitmproxy-go --port 8082

# 指定配置文件
./mitmproxy-go --config my-config.json

# 启用详细日志
./mitmproxy-go -v
```

### 4. 测试

启动测试服务器：

```bash
go run ./testserver/server.go 8888
```

**方式一：系统代理配置（推荐用于开发测试）**

```bash
# 设置代理环境变量
export http_proxy=http://127.0.0.1:8082
export https_proxy=http://127.0.0.1:8082

# 测试 - 请求会被代理拦截
curl http://127.0.0.1:8888/api/users
# 返回: {"code": 0, "data": [{"id": 1, "name": "hacked"}]}
```

**方式二：curl 直接指定代理**

```bash
# 使用 -x 参数指定代理
curl -x http://127.0.0.1:8082 http://127.0.0.1:8888/api/users
# 返回: {"code": 0, "data": [{"id": 1, "name": "hacked"}]}
```

**方式三：iptables 透明代理（Linux 生产环境）**

```bash
# 需要 root 权限，将所有 80/443 流量重定向到代理
sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8082
sudo iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-port 8082

# 启动透明代理模式
./mitmproxy-go --mode transparent --port 8082

# 测试 - 无需设置代理，流量自动被劫持
curl http://127.0.0.1:8888/api/users
```

## 代理方式对比

| 方式 | 适用场景 | 优点 | 缺点 |
|------|---------|------|------|
| **系统代理配置** | 开发测试、单个应用 | 简单灵活、即配即用 | 需应用支持代理设置 |
| **iptables 透明代理** | 生产环境、全网流量 | 无感知、全局生效 | 需 root、配置复杂、仅 Linux |

**效果是否一样？**

- **拦截效果**：基本一致，都能劫持匹配的 URL 并返回自定义响应
- **实现原理**：不同
  - 系统代理：应用主动将请求发送到代理服务器
  - iptables：操作系统强制重定向流量到代理服务器
- **适用范围**：
  - 系统代理：只影响配置了代理的应用
  - iptables：影响整个系统的所有流量

## 项目结构

```
mitmproxy-go/
├── main.go              # 主入口
├── go.mod               # Go 模块
├── config.json          # 配置文件示例
├── config/
│   └── config.go        # 配置管理
├── interceptor/
│   └── interceptor.go   # 流量拦截器
├── proxy/
│   └── proxy.go         # 代理服务器
└── testserver/
    └── server.go        # 测试服务器
```

## 使用示例

### 场景 1: 劫持特定 API 响应

```json
{
  "intercept_paths": ["/api/users"],
  "responses": {
    "/api/users": "{\"code\": 0, \"data\": [{\"id\": 1, \"name\": \"hacked\"}]}"
  }
}
```

### 场景 2: 使用正则表达式匹配多个路径

```json
{
  "intercept_paths": [
    "/api/v[0-9]+/.*",
    "/user/.*/info"
  ]
}
```

### 场景 3: 透明代理模式（Linux）

```bash
# 配置 iptables 规则
sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8082

# 启动透明代理
sudo ./mitmproxy-go --mode transparent --port 8082
```

## 适用场景

1. **API  Mock 测试** - 拦截特定接口返回模拟数据
2. **离线环境部署** - 单二进制文件，无外部依赖
3. **安全测试** - 分析和修改 HTTP/HTTPS 流量
4. **嵌入式系统** - 资源受限的环境

## 许可证

MIT License
