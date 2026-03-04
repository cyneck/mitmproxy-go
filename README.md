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
git clone https://github.com/your-repo/mitmproxy-go.git
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
    "/DescribeLicense",
    "/api/v1/.*"
  ],
  "responses": {
    "/DescribeLicense": "{\"Response\": {\"License\": {\"LicenseId\": \"license-001\"}}}",
    "/api/v1/user": "{\"code\": 0, \"data\": {\"name\": \"test\"}}"
  },
  "default_response": "{\"error\": \"not found\"}",
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

```bash
# 设置代理环境变量
export http_proxy=http://127.0.0.1:8082
export https_proxy=http://127.0.0.1:8082

# 或者配置iptables进行路由重定向

# 测试劫持
curl -x http://127.0.0.1:8082/DescribeLicense
```

## 项目结构

```
mitmproxy-go/
├── main.go              # 主入口
├── go.mod               # Go 模块
├── config.json          # 配置文件示例
├── config/
│   └── config.go        # 配置管理
├── interceptor/
│   └── interceptor.go  # 流量拦截器
└── proxy/
    └── proxy.go         # 代理服务器
```

## 使用示例

### 场景 1: 劫持特定 API 响应

```yaml
intercept_paths:
  - /api/license

responses:
  /api/license: '{"status": "valid", "expires": "2125-01-01"}'
```

### 场景 2: 使用正则表达式

```yaml
intercept_paths:
  - /api/v[0-9]+/.*
  - /user/.*/info
```

### 场景 3: 透明代理模式

```bash
# 需要 root 权限
sudo ./mitmproxy-go --mode transparent --port 8080
```

1. **离线环境部署** - 单二进制文件，无外部依赖
2. **轻量级需求** - 只需基本的流量劫持功能
3. **嵌入式系统** - 资源受限的环境

## 许可证

MIT License