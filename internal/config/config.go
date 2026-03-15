// config/config.go - 配置管理
package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// Config 系统配置
type Config struct {
	mu sync.RWMutex

	// 服务器配置
	Server ServerConfig `json:"server"`

	// 解析配置
	Parser ParserConfig `json:"parser"`

	// 处理器配置
	Processor ProcessorConfig `json:"processor"`

	// 告警配置
	Alert AlertConfig `json:"alert"`

	// 显示配置
	Display DisplayConfig `json:"display"`

	// 导入配置
	Import ImportConfig `json:"import"`

	// 存储配置
	Storage StorageConfig `json:"storage"`

	// 接收器配置
	Receiver ReceiverConfig `json:"receiver"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ParserConfig 解析配置 - 系统自动识别格式，无需手动配置
type ParserConfig struct {
	// 格式自动识别，固定为 auto
	Format string `json:"format"`
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	// 工作协程数
	WorkerCount int `json:"worker_count"`

	// 批处理大小
	BatchSize int `json:"batch_size"`

	// 批处理超时（毫秒）
	BatchTimeout int `json:"batch_timeout"`
}

// AlertConfig 告警配置
type AlertConfig struct {
	// 慢请求阈值（毫秒）
	SlowThreshold int `json:"slow_threshold"`

	// 错误率阈值（百分比）
	ErrorRateThreshold int `json:"error_rate_threshold"`
}

// DisplayConfig 显示配置
type DisplayConfig struct {
	// 每页显示条数
	PageSize int `json:"page_size"`

	// 自动刷新间隔（秒，0表示关闭）
	RefreshInterval int `json:"refresh_interval"`

	// 显示的列
	Columns []string `json:"columns"`
}

// ImportConfig 导入配置
type ImportConfig struct {
	// 并发数
	Concurrency int `json:"concurrency"`

	// 单文件最大行数
	MaxLines int `json:"max_lines"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	// 存储类型：memory, sqlite
	Type string `json:"type"`

	// SQLite数据库路径
	DBPath string `json:"db_path,omitempty"`

	// 内存存储最大条目数
	MaxMemoryItems int `json:"max_memory_items,omitempty"`

	// 数据保留时间（小时）
	RetentionHours int `json:"retention_hours"`
}

// ReceiverConfig 接收器配置
type ReceiverConfig struct {
	// TCP接收配置
	TCPEnabled bool `json:"tcp_enabled"`
	TCPPort    int  `json:"tcp_port"`

	// UDP接收配置
	UDPEnabled bool `json:"udp_enabled"`
	UDPPort    int  `json:"udp_port"`

	// HTTP接收配置
	HTTPEnabled bool `json:"http_enabled"`
	HTTPPort    int  `json:"http_port"`

	// HTTP安全认证配置
	HTTPAuthToken    string   `json:"http_auth_token,omitempty"`    // 认证Token，为空则不启用
	HTTPAllowedIPs   []string `json:"http_allowed_ips,omitempty"`   // IP白名单，为空则不限制
	HTTPMaxBodySize  int64    `json:"http_max_body_size,omitempty"` // 最大请求体大小(字节)，默认10MB
	HTTPRateLimit    int      `json:"http_rate_limit,omitempty"`    // 每分钟最大请求数，0为不限制

	// 文件监控配置
	FileWatcherEnabled bool     `json:"file_watcher_enabled"`
	WatchPaths         []string `json:"watch_paths,omitempty"`

	// 最大连接数
	MaxConnections int `json:"max_connections"`

	// 接收缓冲区大小
	BufferSize int `json:"buffer_size"`
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig 获取配置单例
func GetConfig() *Config {
	once.Do(func() {
		instance = loadDefaultConfig()
	})
	return instance
}

// loadDefaultConfig 加载默认配置
func loadDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Parser: ParserConfig{
			Format: "auto", // 系统自动识别日志格式
		},
		Processor: ProcessorConfig{
			WorkerCount:  10,
			BatchSize:    500,
			BatchTimeout: 1000,
		},
		Alert: AlertConfig{
			SlowThreshold:      1000,
			ErrorRateThreshold: 5,
		},
		Display: DisplayConfig{
			PageSize:        50,
			RefreshInterval: 10,
			Columns:         []string{"timestamp", "method", "path", "status_code", "response_time", "client_ip"},
		},
		Import: ImportConfig{
			Concurrency: 5,
			MaxLines:    100000,
		},
		Storage: StorageConfig{
			Type:           "sqlite",
			DBPath:         "./data/logs.db",
			MaxMemoryItems: 100000,
			RetentionHours: 168, // 7天
		},
		Receiver: ReceiverConfig{
			TCPEnabled:         true,
			TCPPort:            9000,
			UDPEnabled:         true,
			UDPPort:            9001,
			HTTPEnabled:        true,
			HTTPPort:           9002,
			FileWatcherEnabled: false,
			WatchPaths:         []string{},
			MaxConnections:     1000,
			BufferSize:         8192,
		},
	}
}



// Update 更新配置
func (c *Config) Update(newConfig *Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Server = newConfig.Server
	c.Parser = newConfig.Parser
	c.Processor = newConfig.Processor
	c.Alert = newConfig.Alert
	c.Display = newConfig.Display
	c.Import = newConfig.Import
	c.Storage = newConfig.Storage
	c.Receiver = newConfig.Receiver

	return nil
}

// Get 获取配置副本
func (c *Config) Get() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Config{
		Server:    c.Server,
		Parser:    c.Parser,
		Processor: c.Processor,
		Alert:     c.Alert,
		Display:   c.Display,
		Import:    c.Import,
		Storage:   c.Storage,
		Receiver:  c.Receiver,
	}
}

// SaveToFile 保存配置到文件
func (c *Config) SaveToFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadFromFile 从文件加载配置
func (c *Config) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file not found: %s, using default config", path)
			return nil
		}
		return err
	}

	var newConfig Config
	if err := json.Unmarshal(data, &newConfig); err != nil {
		return err
	}

	return c.Update(&newConfig)
}

// GetParserConfig 获取解析配置
func (c *Config) GetParserConfig() ParserConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Parser
}

// GetProcessorConfig 获取处理器配置
func (c *Config) GetProcessorConfig() ProcessorConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Processor
}

// GetReceiverConfig 获取接收器配置
func (c *Config) GetReceiverConfig() ReceiverConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Receiver
}

// GetAlertConfig 获取告警配置
func (c *Config) GetAlertConfig() AlertConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Alert
}

// GetDisplayConfig 获取显示配置
func (c *Config) GetDisplayConfig() DisplayConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Display
}

// GetImportConfig 获取导入配置
func (c *Config) GetImportConfig() ImportConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Import
}
