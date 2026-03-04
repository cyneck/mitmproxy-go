package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
)

// Config 配置结构
type Config struct {
	ListenPort      int               `json:"listen_port"`
	ProxyMode       string            `json:"proxy_mode"`
	InterceptPaths  []string          `json:"intercept_paths"`
	Responses       map[string]string `json:"responses"`
	DefaultResponse string            `json:"default_response"`
	Verbose         bool              `json:"verbose"`
	ConfigFile      string            // 配置文件路径（运行时使用）

	mu            sync.RWMutex
	compiledPaths []*regexp.Regexp // 编译后的正则表达式
}

// ConfigManager 配置管理器
type ConfigManager struct {
	config     *Config
	configFile string
	verbose    bool
}

var (
	defaultConfig = &Config{
		ListenPort:      8082,
		ProxyMode:       "regular",
		InterceptPaths:  []string{},
		Responses:       map[string]string{},
		DefaultResponse: `{"error": "not found"}`,
		Verbose:         false,
	}

	instance *ConfigManager
	once     sync.Once
)

// GetInstance 获取配置管理器单例
func GetInstance() *ConfigManager {
	once.Do(func() {
		instance = &ConfigManager{
			config: defaultConfig,
		}
	})
	return instance
}

// Load 加载配置
func (cm *ConfigManager) Load(configFile string, port int, proxyMode string, verbose bool) error {
	cm.configFile = configFile

	// 如果配置文件存在，读取它
	if configFile != "" {
		if err := cm.loadFromFile(configFile); err != nil {
			return fmt.Errorf("failed to load config from %s: %w", configFile, err)
		}
	}

	// 命令行参数覆盖
	if port > 0 {
		cm.config.ListenPort = port
	}
	if proxyMode != "" {
		cm.config.ProxyMode = proxyMode
	}
	if verbose {
		cm.config.Verbose = verbose
	}

	// 编译正则表达式
	if err := cm.compilePaths(); err != nil {
		return fmt.Errorf("failed to compile regex paths: %w", err)
	}

	cm.verbose = cm.config.Verbose
	return nil
}

// loadFromFile 从文件加载配置
func (cm *ConfigManager) loadFromFile(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	if cfg.ListenPort == 0 {
		cfg.ListenPort = defaultConfig.ListenPort
	}
	if cfg.ProxyMode == "" {
		cfg.ProxyMode = defaultConfig.ProxyMode
	}
	if cfg.Responses == nil {
		cfg.Responses = map[string]string{}
	}
	if cfg.DefaultResponse == "" {
		cfg.DefaultResponse = defaultConfig.DefaultResponse
	}

	cm.config = cfg
	return nil
}

// compilePaths 编译正则表达式路径
func (cm *ConfigManager) compilePaths() error {
	cm.config.mu.Lock()
	defer cm.config.mu.Unlock()

	cm.config.compiledPaths = make([]*regexp.Regexp, 0, len(cm.config.InterceptPaths))
	for _, path := range cm.config.InterceptPaths {
		re, err := regexp.Compile(path)
		if err != nil {
			return fmt.Errorf("invalid regex pattern %q: %w", path, err)
		}
		cm.config.compiledPaths = append(cm.config.compiledPaths, re)
	}
	return nil
}

// Reload 重新加载配置
func (cm *ConfigManager) Reload() error {
	log.Println("[Config] Reloading configuration...")
	if err := cm.loadFromFile(cm.configFile); err != nil {
		return err
	}
	if err := cm.compilePaths(); err != nil {
		return err
	}
	log.Println("[Config] Configuration reloaded successfully")
	return nil
}

// GetConfig 获取当前配置（线程安全）
func (cm *ConfigManager) GetConfig() *Config {
	cm.config.mu.RLock()
	defer cm.config.mu.RUnlock()

	// 返回副本以避免数据竞争
	cfg := *cm.config
	cfg.Responses = make(map[string]string, len(cm.config.Responses))
	for k, v := range cm.config.Responses {
		cfg.Responses[k] = v
	}
	cfg.compiledPaths = cm.config.compiledPaths
	return &cfg
}

// GetPort 获取监听端口
func (cm *ConfigManager) GetPort() int {
	cm.config.mu.RLock()
	defer cm.config.mu.RUnlock()
	return cm.config.ListenPort
}

// GetProxyMode 获取代理模式
func (cm *ConfigManager) GetProxyMode() string {
	cm.config.mu.RLock()
	defer cm.config.mu.RUnlock()
	return cm.config.ProxyMode
}

// IsVerbose 是否详细日志
func (cm *ConfigManager) IsVerbose() bool {
	cm.config.mu.RLock()
	defer cm.config.mu.RUnlock()
	return cm.config.Verbose
}

// MatchPath 检查路径是否匹配拦截规则
func (cm *ConfigManager) MatchPath(path string) (matched bool, response string) {
	cm.config.mu.RLock()
	defer cm.config.mu.RUnlock()

	// 检查是否匹配任何拦截路径
	for _, re := range cm.config.compiledPaths {
		if re.MatchString(path) {
			// 查找对应的响应
			for interceptPath, resp := range cm.config.Responses {
				// 精确匹配或正则匹配
				if interceptPath == path {
					return true, resp
				}
				// 检查是否是前缀匹配
				if len(interceptPath) > 0 && interceptPath[len(interceptPath)-1] == '*' {
					prefix := interceptPath[:len(interceptPath)-1]
					if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
						return true, resp
					}
				}
				// 检查正则匹配
				if matched, _ := regexp.MatchString(interceptPath, path); matched {
					return true, resp
				}
			}
			// 匹配路径但没有自定义响应，使用默认响应
			return true, cm.config.DefaultResponse
		}
	}

	return false, ""
}

// GetDefaultResponse 获取默认响应
func (cm *ConfigManager) GetDefaultResponse() string {
	cm.config.mu.RLock()
	defer cm.config.mu.RUnlock()
	return cm.config.DefaultResponse
}
