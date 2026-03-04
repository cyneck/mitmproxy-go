package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mitmproxy-go/config"
	"mitmproxy-go/proxy"
)

// 存储信号处理器引用
var (
	cfgManager *config.ConfigManager
	server     *proxy.Server
	sigChan    chan os.Signal
)

func main() {
	// 显示版本信息
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	configFile := flag.String("config", "config.json", "Config file path")
	port := flag.Int("port", 0, "Listen port (0 means use config file)")
	mode := flag.String("mode", "", "Proxy mode: regular or transparent")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	if *showVersion {
		fmt.Println("mitmproxy-go version 1.0.0")
		fmt.Println("A lightweight HTTP/HTTPS proxy with traffic interception capabilities")
		return
	}

	if *showHelp {
		showHelpMessage()
		return
	}

	// 初始化配置管理器
	cfgManager = config.GetInstance()

	// 加载配置
	if err := cfgManager.Load(*configFile, *port, *mode, *verbose); err != nil {
		log.Fatalf("[Error] Failed to load configuration: %v", err)
	}

	// 显示启动信息
	cfg := cfgManager.GetConfig()
	log.Printf("[Info] Starting mitmproxy-go")
	log.Printf("[Info] Listen port: %d", cfg.ListenPort)
	log.Printf("[Info] Proxy mode: %s", cfg.ProxyMode)
	log.Printf("[Info] Config file: %s", *configFile)

	// 创建代理服务器
	server = proxy.New(cfgManager)

	// 设置信号处理器
	setupSignalHandler(*configFile)

	// 根据代理模式启动服务器
	if cfg.ProxyMode == "transparent" {
		log.Println("[Info] Starting in transparent proxy mode")
		log.Println("[Warning] Transparent mode requires root privileges and iptables configuration")
		if err := server.StartTransparent(); err != nil {
			log.Fatalf("[Error] Failed to start transparent proxy: %v", err)
		}
	} else {
		// 启动服务器
		log.Println("[Info] Starting in regular proxy mode")
		if err := server.StartWithGracefulShutdown(); err != nil {
			log.Fatalf("[Error] Failed to start server: %v", err)
		}
	}
}

// setupSignalHandler 设置信号处理器
func setupSignalHandler(configFile string) {
	sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		for {
			sig := <-sigChan
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Printf("[Info] Received signal %v, shutting down...", sig)
				// 优雅关闭由 server.StartWithGracefulShutdown 处理
				os.Exit(0)
			case syscall.SIGHUP:
				log.Printf("[Info] Received SIGHUP, reloading configuration...")
				if err := cfgManager.Reload(); err != nil {
					log.Printf("[Error] Failed to reload configuration: %v", err)
				} else {
					log.Printf("[Info] Configuration reloaded successfully")
				}
			}
		}
	}()
}

// showHelpMessage 显示帮助信息
func showHelpMessage() {
	fmt.Println("mitmproxy-go - A lightweight HTTP/HTTPS proxy with traffic interception")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mitmproxy-go [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --port PORT     Listen port (default: 8082)")
	fmt.Println("  --mode MODE     Proxy mode: regular or transparent (default: regular)")
	fmt.Println("  --config FILE   Config file path (default: config.json)")
	fmt.Println("  --verbose       Enable verbose logging")
	fmt.Println("  -v              Enable verbose logging (short)")
	fmt.Println("  --version       Show version information")
	fmt.Println("  --help          Show this help information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mitmproxy-go                      # Run with default config")
	fmt.Println("  mitmproxy-go --port 8080          # Run on port 8080")
	fmt.Println("  mitmproxy-go --config my.json     # Use custom config file")
	fmt.Println("  mitmproxy-go --verbose            # Enable verbose logging")
	fmt.Println()
	fmt.Println("For more information, see: https://github.com/your-repo/mitmproxy-go")
}