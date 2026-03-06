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

// ParserConfig 解析配置
type ParserConfig struct {
	// 字段分隔符，如 " ", "\t", "|" 等
	Delimiter string `json:"delimiter"`

	// 字段映射规则：位置 -> 字段名
	FieldMapping map[int]string `json:"field_mapping"`

	// 预定义格式：nginx, apache, json, custom
	Format string `json:"format"`

	// 时间字段格式
	TimeFormat string `json:"time_format"`

	// 是否解析User-Agent
	ParseUserAgent bool `json:"parse_user_agent"`

	// 自定义解析规则（正则表达式）
	CustomRegex string `json:"custom_regex,omitempty"`
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	// 工作协程数
	WorkerCount int `json:"worker_count"`

	// 批处理大小
	BatchSize int `json:"batch_size"`

	// 批处理超时（毫秒）
	BatchTimeout int `json:"batch_timeout"`

	// 清洗规则
	CleanRules []CleanRule `json:"clean_rules"`

	// 过滤规则
	FilterRules []FilterRule `json:"filter_rules"`
}

// CleanRule 清洗规则
type CleanRule struct {
	Field     string `json:"field"`     // 目标字段
	Operation string `json:"operation"` // trim, remove, replace, regex
	Value     string `json:"value"`     // 替换值或正则表达式
}

// FilterRule 过滤规则
type FilterRule struct {
	Field     string `json:"field"`
	Operator  string `json:"operator"` // eq, ne, gt, lt, contains, regex
	Value     string `json:"value"`
	Condition string `json:"condition"` // and, or
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
			Delimiter:      " ",
			FieldMapping:   getDefaultFieldMapping(),
			Format:         "nginx",
			TimeFormat:     "02/Jan/2006:15:04:05 -0700",
			ParseUserAgent: false,
		},
		Processor: ProcessorConfig{
			WorkerCount:    10,
			BatchSize:      100,
			BatchTimeout:   1000,
			CleanRules:     []CleanRule{},
			FilterRules:    []FilterRule{},
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

// getDefaultFieldMapping 获取默认字段映射（Nginx格式）
func getDefaultFieldMapping() map[int]string {
	return map[int]string{
		0:  "client_ip",
		3:  "timestamp",
		4:  "method",
		5:  "path",
		6:  "protocol",
		8:  "status_code",
		9:  "response_size",
		10: "referer",
		11: "user_agent",
	}
}

// Update 更新配置
func (c *Config) Update(newConfig *Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Server = newConfig.Server
	c.Parser = newConfig.Parser
	c.Processor = newConfig.Processor
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
