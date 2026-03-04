package proxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mitmproxy-go/config"
	"mitmproxy-go/interceptor"
)

// Server 代理服务器
type Server struct {
	cfgManager *config.ConfigManager
	interceptor *interceptor.Interceptor
	httpServer  *http.Server
}

// New 创建代理服务器
func New(cfgManager *config.ConfigManager) *Server {
	return &Server{
		cfgManager: cfgManager,
		interceptor: interceptor.New(cfgManager),
	}
}

// Start 启动代理服务器
func (s *Server) Start() error {
	cfg := s.cfgManager.GetConfig()

	// 创建 HTTP 服务器
	addr := fmt.Sprintf(":%d", cfg.ListenPort)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.createHandler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("[Proxy] Starting server on %s", addr)
	log.Printf("[Proxy] Proxy mode: %s", cfg.ProxyMode)

	// 启动服务器
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// createHandler 创建处理器
func (s *Server) createHandler() http.Handler {
	mux := http.NewServeMux()

	// 注册拦截处理器
	mux.HandleFunc("/", s.handleRequest)

	return mux
}

// handleRequest 处理请求
func (s *Server) handleRequest(w http.ResponseWriter, req *http.Request) {
	cfg := s.cfgManager.GetConfig()

	// 记录原始请求信息
	if cfg.Verbose {
		log.Printf("[Proxy] %s %s from %s", req.Method, req.URL.String(), req.RemoteAddr)
	}

	// 统一使用拦截器处理
	s.interceptor.RequestHandler(w, req)
}

// StartWithGracefulShutdown 启动服务器并支持优雅关闭
func (s *Server) StartWithGracefulShutdown() error {
	cfg := s.cfgManager.GetConfig()

	addr := fmt.Sprintf(":%d", cfg.ListenPort)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.createHandler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器
	go func() {
		log.Printf("[Proxy] Starting server on %s", addr)
		log.Printf("[Proxy] Proxy mode: %s", cfg.ProxyMode)

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Error] Server error: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Proxy] Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	log.Println("[Proxy] Server stopped")
	return nil
}

// StartTransparent 启动透明代理模式
func (s *Server) StartTransparent() error {
	cfg := s.cfgManager.GetConfig()

	log.Printf("[Proxy] Starting transparent proxy on port %d", cfg.ListenPort)

	// 透明代理需要iptables配置，这里先实现基础功能
	// 透明代理模式需要root权限和iptables规则
	addr := fmt.Sprintf(":%d", cfg.ListenPort)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.createTransparentHandler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s.httpServer.ListenAndServe()
}

// createTransparentHandler 创建透明代理处理器
func (s *Server) createTransparentHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleTransparentRequest)
	return mux
}

// handleTransparentRequest 处理透明代理请求
func (s *Server) handleTransparentRequest(w http.ResponseWriter, req *http.Request) {
	cfg := s.cfgManager.GetConfig()

	if cfg.Verbose {
		log.Printf("[Transparent] %s %s from %s", req.Method, req.URL.String(), req.RemoteAddr)
	}

	// 从 X-Forwarded-For 或 RemoteAddr 获取原始客户端 IP
	clientIP := getClientIP(req)

	// 尝试从请求中提取目标地址
	targetHost := s.extractTargetHost(req)

	if targetHost != "" {
		if cfg.Verbose {
			log.Printf("[Transparent] Target: %s, Client: %s", targetHost, clientIP)
		}
	}

	// 使用拦截器处理
	s.interceptor.RequestHandler(w, req)
}

// extractTargetHost 从请求中提取目标主机
func (s *Server) extractTargetHost(req *http.Request) string {
	// 尝试从不同的 Header 获取目标主机
	hosts := []string{
		req.Header.Get("X-Forwarded-Host"),
		req.Header.Get("X-Real-IP"),
		req.Header.Get("Host"),
		req.Host,
	}

	for _, host := range hosts {
		if host != "" {
			return host
		}
	}

	// 尝试从请求URL获取
	if req.URL.Host != "" {
		return req.URL.Host
	}

	// 尝试从 RemoteAddr 解析
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		return host
	}

	return ""
}

// getClientIP 获取客户端IP
func getClientIP(req *http.Request) string {
	// 优先从 X-Forwarded-For 获取
	xff := req.Header.Get("X-Forwarded-For")
	if xff != "" {
		// 取第一个IP（原始客户端）
		ips := commaSplit(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// 从 X-Real-IP 获取
	xri := req.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// 从 RemoteAddr 解析
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

// commaSplit 分割逗号分隔的字符串
func commaSplit(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}