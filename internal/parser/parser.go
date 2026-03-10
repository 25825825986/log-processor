// parser/parser.go - 日志解析器
package parser

import (
	"encoding/json"
	"fmt"
	"log-processor/internal/config"
	"log-processor/internal/models"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Parser 日志解析器接口
type Parser interface {
	Parse(line string) (*models.LogEntry, error)
	SetConfig(cfg config.ParserConfig)
}

// LogParser 日志解析器实现
type LogParser struct {
	config      config.ParserConfig
	regex       *regexp.Regexp
	jsonParser  *JSONParser
}

// JSONParser JSON格式解析器
type JSONParser struct{}

// NewLogParser 创建新的日志解析器
func NewLogParser(cfg config.ParserConfig) *LogParser {
	p := &LogParser{
		config: cfg,
	}
	p.init()
	return p
}

// init 初始化解析器
func (p *LogParser) init() {
	switch p.config.Format {
	case "nginx":
		// 支持简化格式和完整格式，包含可选的 referer、user_agent 和 response_time
		p.regex = regexp.MustCompile(`^(?P<client_ip>\S+)\s+\S+\s+\S+\s+\[(?P<timestamp>[^\]]+)\]\s+"(?P<method>\S+)\s+(?P<path>\S+)\s+(?P<protocol>[^"]+)"\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)(?:\s+"(?P<referer>[^"]*)"\s+"(?P<user_agent>[^"]*)"(?:\s+"(?P<response_time>[^"]*)")?)?`)
	case "apache":
		p.regex = regexp.MustCompile(`^(?P<client_ip>\S+)\s+\S+\s+\S+\s+\[(?P<timestamp>[^\]]+)\]\s+"(?P<method>\S+)\s+(?P<path>\S+)\s+(?P<protocol>[^"]+)"\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)`)
	case "custom":
		if p.config.CustomRegex != "" {
			p.regex = regexp.MustCompile(p.config.CustomRegex)
		}
	case "json":
		p.jsonParser = &JSONParser{}
	}
}

// SetConfig 更新配置
func (p *LogParser) SetConfig(cfg config.ParserConfig) {
	p.config = cfg
	p.init()
}

// Parse 解析日志行
func (p *LogParser) Parse(line string) (*models.LogEntry, error) {
	entry := models.NewLogEntry()
	entry.RawData = line

	switch p.config.Format {
	case "nginx", "apache":
		return p.parseWithRegex(line, entry)
	case "json":
		return p.parseJSON(line, entry)
	case "csv", "tsv":
		return p.parseDelimited(line, entry)
	case "custom":
		if p.regex != nil {
			return p.parseWithRegex(line, entry)
		}
		return p.parseDelimited(line, entry)
	default:
		return p.parseDelimited(line, entry)
	}
}

// parseWithRegex 使用正则表达式解析
func (p *LogParser) parseWithRegex(line string, entry *models.LogEntry) (*models.LogEntry, error) {
	if p.regex == nil {
		return nil, fmt.Errorf("regex parser not initialized")
	}

	matches := p.regex.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("line does not match pattern")
	}

	names := p.regex.SubexpNames()
	for i, name := range names {
		if i == 0 || name == "" {
			continue
		}
		if i >= len(matches) {
			continue
		}
		p.setField(entry, name, matches[i])
	}

	return entry, nil
}

// parseJSON 解析JSON格式
func (p *LogParser) parseJSON(line string, entry *models.LogEntry) (*models.LogEntry, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return nil, err
	}

	for key, value := range data {
		strValue := fmt.Sprintf("%v", value)
		p.setField(entry, key, strValue)
	}

	return entry, nil
}

// parseDelimited 解析分隔符格式
func (p *LogParser) parseDelimited(line string, entry *models.LogEntry) (*models.LogEntry, error) {
	fields := strings.Split(line, p.config.Delimiter)

	for pos, fieldName := range p.config.FieldMapping {
		if pos >= 0 && pos < len(fields) {
			value := strings.TrimSpace(fields[pos])
			p.setField(entry, fieldName, value)
		}
	}

	// 存储所有额外字段
	for i, field := range fields {
		if _, exists := p.config.FieldMapping[i]; !exists {
			entry.ExtraFields[fmt.Sprintf("field_%d", i)] = strings.TrimSpace(field)
		}
	}

	return entry, nil
}

// setField 设置字段值
func (p *LogParser) setField(entry *models.LogEntry, field, value string) {
	switch field {
	case "client_ip", "ip", "remote_addr":
		entry.ClientIP = value
	case "timestamp", "time", "date":
		entry.Timestamp = p.parseTime(value)
	case "method", "request_method":
		entry.Method = strings.ToUpper(value)
	case "path", "request_uri", "uri", "url":
		entry.Path = value
	case "status_code", "status":
		if code, err := strconv.Atoi(value); err == nil {
			entry.StatusCode = code
		}
	case "response_time", "request_time", "duration":
		if rt, err := strconv.ParseFloat(value, 64); err == nil {
			entry.ResponseTime = int64(rt * 1000) // 转换为毫秒
		} else {
			entry.ResponseTime, _ = strconv.ParseInt(value, 10, 64)
		}
	case "response_size", "bytes_sent", "body_bytes_sent":
		if size, err := strconv.ParseInt(value, 10, 64); err == nil {
			entry.ResponseSize = size
		}
	case "request_size", "bytes_received":
		if size, err := strconv.ParseInt(value, 10, 64); err == nil {
			entry.RequestSize = size
		}
	case "user_agent", "http_user_agent":
		entry.UserAgent = value
	case "referer", "referrer", "http_referer":
		entry.Referer = value
	case "level", "log_level":
		entry.Level = value
	case "source", "app", "service":
		entry.Source = value
	default:
		entry.ExtraFields[field] = value
	}
}

// parseTime 解析时间字符串
func (p *LogParser) parseTime(value string) time.Time {
	// 尝试多种时间格式
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"02/Jan/2006:15:04:05 -0700",
		"02/Jan/2006:15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006/01/02 15:04:05",
		"01/02/2006 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t
		}
	}

	// 尝试Unix时间戳
	if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
		if ts > 1e12 {
			return time.Unix(ts/1000, 0) // 毫秒时间戳
		}
		return time.Unix(ts, 0)
	}

	return time.Now()
}

// ParserPool 解析器池
type ParserPool struct {
	parsers chan *LogParser
	config  config.ParserConfig
}

// NewParserPool 创建解析器池
func NewParserPool(size int, cfg config.ParserConfig) *ParserPool {
	pool := &ParserPool{
		parsers: make(chan *LogParser, size),
		config:  cfg,
	}
	for i := 0; i < size; i++ {
		pool.parsers <- NewLogParser(cfg)
	}
	return pool
}

// Get 获取解析器
func (p *ParserPool) Get() *LogParser {
	select {
	case parser := <-p.parsers:
		return parser
	default:
		return NewLogParser(p.config)
	}
}

// Put 归还解析器
func (p *ParserPool) Put(parser *LogParser) {
	select {
	case p.parsers <- parser:
	default:
		// 池已满，丢弃
	}
}

// UpdateConfig 更新配置
func (p *ParserPool) UpdateConfig(cfg config.ParserConfig) {
	p.config = cfg
	// 清空池并重新创建
	for len(p.parsers) > 0 {
		<-p.parsers
	}
	for i := 0; i < cap(p.parsers); i++ {
		p.parsers <- NewLogParser(cfg)
	}
}
