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

	Server    ServerConfig    `json:"server"`
	Parser    ParserConfig    `json:"parser"`
	Processor ProcessorConfig `json:"processor"`
	Alert     AlertConfig     `json:"alert"`
	Display   DisplayConfig   `json:"display"`
	Import    ImportConfig    `json:"import"`
	Storage   StorageConfig   `json:"storage"`
	Receiver  ReceiverConfig  `json:"receiver"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ParserConfig 解析配置
type ParserConfig struct {
	// 格式自动识别，固定为 auto
	Format string `json:"format"`
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	WorkerCount  int `json:"worker_count"`
	BatchSize    int `json:"batch_size"`
	BatchTimeout int `json:"batch_timeout"`

	// 磁盘溢写配置
	OverflowEnabled         bool   `json:"overflow_enabled"`
	OverflowDir             string `json:"overflow_dir,omitempty"`
	OverflowMaxDiskMB       int    `json:"overflow_max_disk_mb,omitempty"`
	OverflowDrainBatch      int    `json:"overflow_drain_batch,omitempty"`
	OverflowDrainIntervalMS int    `json:"overflow_drain_interval_ms,omitempty"`
}

// AlertConfig 告警配置
type AlertConfig struct {
	SlowThreshold      int `json:"slow_threshold"`
	ErrorRateThreshold int `json:"error_rate_threshold"`
}

// DisplayConfig 显示配置
type DisplayConfig struct {
	PageSize        int      `json:"page_size"`
	RefreshInterval int      `json:"refresh_interval"`
	Columns         []string `json:"columns"`
}

// ImportConfig 导入配置
type ImportConfig struct {
	Concurrency int `json:"concurrency"`
	MaxLines    int `json:"max_lines"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type           string `json:"type"`
	DBPath         string `json:"db_path,omitempty"`
	MaxMemoryItems int    `json:"max_memory_items,omitempty"`
	RetentionHours int    `json:"retention_hours"`
}

// ReceiverConfig 接收器配置
type ReceiverConfig struct {
	TCPEnabled bool `json:"tcp_enabled"`
	TCPPort    int  `json:"tcp_port"`

	UDPEnabled bool `json:"udp_enabled"`
	UDPPort    int  `json:"udp_port"`

	HTTPEnabled bool `json:"http_enabled"`
	HTTPPort    int  `json:"http_port"`

	HTTPAuthToken   string   `json:"http_auth_token,omitempty"`
	HTTPAllowedIPs  []string `json:"http_allowed_ips,omitempty"`
	HTTPMaxBodySize int64    `json:"http_max_body_size,omitempty"`
	HTTPRateLimit   int      `json:"http_rate_limit,omitempty"`

	FileWatcherEnabled bool     `json:"file_watcher_enabled"`
	WatchPaths         []string `json:"watch_paths,omitempty"`
	MaxConnections     int      `json:"max_connections"`
	BufferSize         int      `json:"buffer_size"`
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
			Format: "auto",
		},
		Processor: ProcessorConfig{
			WorkerCount:             10,
			BatchSize:               500,
			BatchTimeout:            1000,
			OverflowEnabled:         true,
			OverflowDir:             "./data/overflow",
			OverflowMaxDiskMB:       512,
			OverflowDrainBatch:      1000,
			OverflowDrainIntervalMS: 200,
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
			RetentionHours: 168,
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

	normalizeParserConfig(&newConfig.Parser)
	normalizeProcessorConfig(&newConfig.Processor)
	normalizeAlertConfig(&newConfig.Alert)
	normalizeDisplayConfig(&newConfig.Display)
	normalizeImportConfig(&newConfig.Import)
	normalizeStorageConfig(&newConfig.Storage)
	normalizeReceiverConfig(&newConfig.Receiver)

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
	return os.WriteFile(path, data, 0o644)
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

	// 兼容旧配置：如果没有该字段，默认开启溢写。
	if !hasProcessorField(data, "overflow_enabled") {
		newConfig.Processor.OverflowEnabled = true
	}
	normalizeParserConfig(&newConfig.Parser)
	normalizeProcessorConfig(&newConfig.Processor)
	normalizeAlertConfig(&newConfig.Alert)
	normalizeDisplayConfig(&newConfig.Display)
	normalizeImportConfig(&newConfig.Import)
	normalizeStorageConfig(&newConfig.Storage)
	normalizeReceiverConfig(&newConfig.Receiver)

	return c.Update(&newConfig)
}

func normalizeParserConfig(cfg *ParserConfig) {
	if cfg.Format == "" {
		cfg.Format = "auto"
	}
}

func normalizeProcessorConfig(cfg *ProcessorConfig) {
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 10
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 1000
	}
	if cfg.OverflowDir == "" {
		cfg.OverflowDir = "./data/overflow"
	}
	if cfg.OverflowMaxDiskMB <= 0 {
		cfg.OverflowMaxDiskMB = 512
	}
	if cfg.OverflowDrainBatch <= 0 {
		cfg.OverflowDrainBatch = 1000
	}
	if cfg.OverflowDrainIntervalMS <= 0 {
		cfg.OverflowDrainIntervalMS = 200
	}
}

func normalizeAlertConfig(cfg *AlertConfig) {
	if cfg.SlowThreshold < 0 {
		cfg.SlowThreshold = 1000
	}
	if cfg.ErrorRateThreshold < 0 {
		cfg.ErrorRateThreshold = 5
	}
}

func normalizeDisplayConfig(cfg *DisplayConfig) {
	if cfg.PageSize <= 0 {
		cfg.PageSize = 50
	}
	if cfg.RefreshInterval < 0 {
		cfg.RefreshInterval = 10
	}
	if len(cfg.Columns) == 0 {
		cfg.Columns = []string{"timestamp", "method", "path", "status_code", "response_time", "client_ip"}
	}
}

func normalizeImportConfig(cfg *ImportConfig) {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.MaxLines <= 0 || cfg.MaxLines > 100000 {
		cfg.MaxLines = 100000
	}
}

func normalizeStorageConfig(cfg *StorageConfig) {
	if cfg.Type == "" {
		cfg.Type = "sqlite"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "./data/logs.db"
	}
	if cfg.RetentionHours <= 0 {
		cfg.RetentionHours = 168
	}
	if cfg.MaxMemoryItems <= 0 {
		cfg.MaxMemoryItems = 100000
	}
}

func normalizeReceiverConfig(cfg *ReceiverConfig) {
	if cfg.TCPPort <= 0 {
		cfg.TCPPort = 9000
	}
	if cfg.UDPPort <= 0 {
		cfg.UDPPort = 9001
	}
	if cfg.HTTPPort <= 0 {
		cfg.HTTPPort = 9002
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 8192
	}
	if cfg.MaxConnections <= 0 {
		cfg.MaxConnections = 1000
	}
}

func hasProcessorField(raw []byte, field string) bool {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return false
	}

	processorRaw, ok := root["processor"]
	if !ok {
		return false
	}

	var processorMap map[string]json.RawMessage
	if err := json.Unmarshal(processorRaw, &processorMap); err != nil {
		return false
	}

	_, exists := processorMap[field]
	return exists
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
