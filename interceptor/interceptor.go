package interceptor

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mitmproxy-go/config"
)

// Interceptor 流量拦截器
type Interceptor struct {
	cfgManager *config.ConfigManager
}

// New 创建拦截器
func New(cfgManager *config.ConfigManager) *Interceptor {
	return &Interceptor{
		cfgManager: cfgManager,
	}
}

// RequestHandler 请求处理器
func (i *Interceptor) RequestHandler(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	// 获取配置
	cfg := i.cfgManager.GetConfig()

	// 记录请求信息
	if cfg.Verbose {
		i.logRequest(req)
	}

	// 获取请求路径
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path = path + "?" + req.URL.RawQuery
	}

	// 检查是否匹配拦截规则
	matched, response := i.cfgManager.MatchPath(req.URL.Path)

	if matched {
		// 记录匹配信息
		if cfg.Verbose {
			log.Printf("[Interceptor] Path matched: %s -> %s", req.URL.Path, truncateString(response, 100))
		}

		// 发送自定义响应
		i.sendResponse(w, req, response, startTime)
		return
	}

	// 不匹配，继续代理到目标服务器
	i.proxyRequest(w, req, startTime)
}

// logRequest 记录请求信息
func (i *Interceptor) logRequest(req *http.Request) {
	host := req.Host
	if req.URL.Host != "" {
		host = req.URL.Host
	}
	log.Printf("[Request] %s %s%s | Host: %s | User-Agent: %s",
		req.Method,
		req.URL.Path,
		getQueryString(req.URL),
		host,
		truncateString(req.UserAgent(), 50),
	)
}

// getQueryString 获取查询字符串
func getQueryString(u *url.URL) string {
	if u.RawQuery != "" {
		return "?" + u.RawQuery
	}
	return ""
}

// sendResponse 发送自定义响应
func (i *Interceptor) sendResponse(w http.ResponseWriter, req *http.Request, responseBody string, startTime time.Time) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Intercepted", "true")

	// 写入响应状态码
	w.WriteHeader(http.StatusOK)

	// 写入响应体
	if _, err := w.Write([]byte(responseBody)); err != nil {
		log.Printf("[Error] Failed to write response: %v", err)
	}

	// 记录响应信息
	duration := time.Since(startTime)
	cfg := i.cfgManager.GetConfig()
	if cfg.Verbose {
		log.Printf("[Response] %d | %s | %s | %s",
			http.StatusOK,
			req.URL.Path,
			truncateString(responseBody, 100),
			duration,
		)
	}
}

// proxyRequest 代理请求到目标服务器
func (i *Interceptor) proxyRequest(w http.ResponseWriter, req *http.Request, startTime time.Time) {
	// 获取目标服务器地址
	targetURL := i.getTargetURL(req)
	if targetURL == "" {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// 创建代理请求
	proxyReq, err := http.NewRequest(req.Method, targetURL, req.Body)
	if err != nil {
		log.Printf("[Error] Failed to create proxy request: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// 复制请求头
	for key, values := range req.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// 移除 hop-by-hop 头
	removeHopByHopHeaders(proxyReq.Header)

	// 设置原始请求的 Host
	if req.Host != "" {
		proxyReq.Host = req.Host
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("[Error] Failed to proxy request to %s: %v", targetURL, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头到客户端
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 写入响应状态码
	w.WriteHeader(resp.StatusCode)

	// 复制响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Error] Failed to read response body: %v", err)
	} else {
		if _, err := w.Write(body); err != nil {
			log.Printf("[Error] Failed to write response: %v", err)
		}
	}

	// 记录响应信息
	duration := time.Since(startTime)
	cfg := i.cfgManager.GetConfig()
	if cfg.Verbose {
		log.Printf("[Proxied] %d | %s -> %s | %s",
			resp.StatusCode,
			req.URL.Path,
			targetURL,
			duration,
		)
	}
}

// getTargetURL 获取目标服务器 URL
func (i *Interceptor) getTargetURL(req *http.Request) string {
	// 如果请求有直接的主机名，使用它
	if req.URL.Host != "" {
		scheme := "http"
		if req.TLS != nil || strings.HasPrefix(req.URL.Path, "https") {
			scheme = "https"
		}
		// 检查是否已经是完整 URL
		if req.URL.Scheme != "" {
			scheme = req.URL.Scheme
		}
		return fmt.Sprintf("%s://%s%s%s", scheme, req.URL.Host, req.URL.Path, getQueryString(req.URL))
	}

	// 对于 CONNECT 请求，返回空让代理处理
	if req.Method == http.MethodConnect {
		return ""
	}

	// 从 Header 中获取 Host
	host := req.Header.Get("Host")
	if host == "" {
		host = req.Host
	}

	if host == "" {
		return ""
	}

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s%s", scheme, host, req.URL.Path, getQueryString(req.URL))
}

// removeHopByHopHeaders 移除 hop-by-hop 头
func removeHopByHopHeaders(header http.Header) {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, h := range hopByHopHeaders {
		header.Del(h)
	}
}

// HandleConnect 处理 CONNECT 请求（用于 HTTPS 代理）
func (i *Interceptor) HandleConnect(w http.ResponseWriter, req *http.Request) {
	host := req.Host

	if i.cfgManager.IsVerbose() {
		log.Printf("[Connect] %s", host)
	}

	// 告诉客户端连接已建立
	w.WriteHeader(http.StatusOK)

	// 获取 Hijacker
	hij, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("[Error] HTTP server does not support hijacking")
		return
	}

	conn, _, err := hij.Hijack()
	if err != nil {
		log.Printf("[Error] Failed to hijack connection: %v", err)
		return
	}

	// 连接到目标服务器
	targetConn, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("[Error] Failed to connect to target %s: %v", host, err)
		conn.Close()
		return
	}

	// 这里可以添加对 HTTPS 流量的处理
	// 由于 HTTPS 是加密的，无法直接修改内容
	// 只能作为透明代理转发流量

	_ = targetConn
	_ = conn
}

// ProxyHandler 返回代理请求处理器
func (i *Interceptor) ProxyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// 处理 CONNECT 方法（用于 HTTPS 代理）
		if req.Method == http.MethodConnect {
			i.HandleConnect(w, req)
			return
		}

		// 处理普通请求
		i.RequestHandler(w, req)
	})
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// HandleHTTP 处理 HTTP 请求
func (i *Interceptor) HandleHTTP(w http.ResponseWriter, req *http.Request) {
	// 清理 URL
	req.URL.Scheme = "http"
	if req.TLS != nil {
		req.URL.Scheme = "https"
	}

	// 处理请求
	i.RequestHandler(w, req)
}

// CopyBody 复制请求体
func CopyBody(body io.Reader) ([]byte, error) {
	if body == nil {
		return []byte{}, nil
	}
	return io.ReadAll(body)
}

// CreateBuffer 创建缓冲区
func CreateBuffer(data []byte) *bytes.Buffer {
	return bytes.NewBuffer(data)
}
