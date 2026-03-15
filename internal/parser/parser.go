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
	// 格式自动识别，无需手动配置特定格式
	// 所有格式在 Parse 方法中通过自动检测处理
}

// SetConfig 更新配置
func (p *LogParser) SetConfig(cfg config.ParserConfig) {
	p.config = cfg
	p.init()
}

// Parse 解析日志行（自动识别格式）
func (p *LogParser) Parse(line string) (*models.LogEntry, error) {
	entry := models.NewLogEntry()
	entry.RawData = line

	// 自动检测格式
	format := DetectFormat(line)
	
	switch format {
	case "nginx", "apache":
		return p.parseNginxApache(line, entry)
	case "json":
		return p.parseJSON(line, entry)
	case "csv", "tsv", "pipe", "semicolon":
		return p.parseAutoDelimited(line, entry, format)
	case "syslog":
		return p.parseSyslog(line, entry)
	default:
		// 未知格式，尝试通用解析
		return p.parseGeneric(line, entry)
	}
}

// parseNginxApache 解析Nginx/Apache格式
func (p *LogParser) parseNginxApache(line string, entry *models.LogEntry) (*models.LogEntry, error) {
	data, ok := parseNginxLog(line)
	if !ok {
		return entry, fmt.Errorf("failed to parse nginx/apache format")
	}
	
	for key, value := range data {
		p.setField(entry, key, value)
	}
	
	return entry, nil
}

// parseAutoDelimited 自动解析分隔符格式
func (p *LogParser) parseAutoDelimited(line string, entry *models.LogEntry, format string) (*models.LogEntry, error) {
	var delimiter string
	switch format {
	case "csv":
		delimiter = ","
	case "tsv":
		delimiter = "\t"
	case "pipe":
		delimiter = "|"
	case "semicolon":
		delimiter = ";"
	default:
		delimiter = ","
	}
	
	fields := strings.Split(line, delimiter)
	
	// 尝试自动推断字段映射
	for i, field := range fields {
		field = strings.TrimSpace(field)
		fieldName := inferFieldName(field, i, len(fields))
		if fieldName != "" {
			p.setField(entry, fieldName, field)
		} else {
			entry.ExtraFields[fmt.Sprintf("field_%d", i)] = field
		}
	}
	
	return entry, nil
}

// parseSyslog 解析Syslog格式
func (p *LogParser) parseSyslog(line string, entry *models.LogEntry) (*models.LogEntry, error) {
	// Syslog格式: 月 日 时间 主机 进程[PID]: 消息
	pattern := regexp.MustCompile(`^(?P<month>\w{3})\s+(?P<day>\d+)\s+(?P<time>\d{2}:\d{2}:\d{2})\s+(?P<host>\S+)\s+(?P<process>[^\[:]+)(?:\[(?P<pid>\d+)\])?:\s+(?P<message>.+)$`)
	
	matches := pattern.FindStringSubmatch(line)
	if matches == nil {
		return entry, fmt.Errorf("failed to parse syslog format")
	}
	
	names := pattern.SubexpNames()
	for i, name := range names {
		if i > 0 && i < len(matches) && name != "" {
			p.setField(entry, name, matches[i])
		}
	}
	
	// 尝试从 message 字段提取 HTTP 访问信息
	// 格式: IP METHOD PATH STATUS SIZE
	message := ""
	for i, name := range names {
		if name == "message" && i < len(matches) {
			message = matches[i]
			break
		}
	}
	
	if message != "" {
		messagePattern := regexp.MustCompile(`^(?P<client_ip>\S+)\s+(?P<method>\S+)\s+(?P<path>\S+)\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)`)
		if msgMatches := messagePattern.FindStringSubmatch(message); msgMatches != nil {
			msgNames := messagePattern.SubexpNames()
			for i, name := range msgNames {
				if i > 0 && i < len(msgMatches) && name != "" {
					p.setField(entry, name, msgMatches[i])
				}
			}
		}
	}
	
	return entry, nil
}

// parseGeneric 通用解析（未知格式）
func (p *LogParser) parseGeneric(line string, entry *models.LogEntry) (*models.LogEntry, error) {
	// 尝试多种通用格式模式
	
	// 模式1: [time] IP METHOD PATH STATUS SIZE TIMEms
	pattern1 := regexp.MustCompile(`\[(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\]\s+(?P<client_ip>\S+)\s+(?P<method>\S+)\s+(?P<path>\S+)\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)\s+(?P<response_time>\d+)ms`)
	if matches := pattern1.FindStringSubmatch(line); matches != nil {
		names := pattern1.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}
	
	// 模式2: time - IP - METHOD PATH - Status: STATUS - Size: SIZE - Time: TIMEms
	pattern2 := regexp.MustCompile(`(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\s+-\s+(?P<client_ip>\S+)\s+-\s+(?P<method>\S+)\s+(?P<path>\S+)\s+-\s+Status:\s+(?P<status_code>\d+)\s+-\s+Size:\s+(?P<response_size>\d+)\s+-\s+Time:\s+(?P<response_time>\d+)ms`)
	if matches := pattern2.FindStringSubmatch(line); matches != nil {
		names := pattern2.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}
	
	// 模式3: [time] IP - METHOD PATH - STATUS - SIZE - TIMEms
	pattern3 := regexp.MustCompile(`\[(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\]\s+(?P<client_ip>\S+)\s+-\s+(?P<method>\S+)\s+(?P<path>\S+)\s+-\s+(?P<status_code>\d+)\s+-\s+(?P<response_size>\d+)\s+-\s+(?P<response_time>\d+)ms`)
	if matches := pattern3.FindStringSubmatch(line); matches != nil {
		names := pattern3.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}
	
	// 模式4: IP [time] "METHOD PATH" STATUS SIZE TIME
	pattern4 := regexp.MustCompile(`(?P<client_ip>\S+)\s+\[(?P<timestamp>[^\]]+)\]\s+"(?P<method>\S+)\s+(?P<path>\S+)"\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)\s+(?P<response_time>\d+)`)
	if matches := pattern4.FindStringSubmatch(line); matches != nil {
		names := pattern4.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}
	
	// 模式5: Request from IP at time: METHOD PATH -> STATUS (SIZE bytes, TIMEms)
	// 注意：timestamp 格式是 2026-03-09 10:00:04
	pattern5 := regexp.MustCompile(`Request from (?P<client_ip>\S+) at (?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}):\s+(?P<method>\S+)\s+(?P<path>\S+)\s+->\s+(?P<status_code>\d+)\s+\((?P<response_size>\d+) bytes, (?P<response_time>\d+)ms\)`)
	if matches := pattern5.FindStringSubmatch(line); matches != nil {
		names := pattern5.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}
	
	// 兜底：尝试提取可能的字段
	// 1. 查找IP地址
	ipPattern := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	if ip := ipPattern.FindString(line); ip != "" {
		entry.ClientIP = ip
	}
	
	// 2. 查找时间戳
	timePatterns := []string{
		`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`,
		`\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2}`,
	}
	for _, pattern := range timePatterns {
		if t := regexp.MustCompile(pattern).FindString(line); t != "" {
			entry.Timestamp = p.parseTime(t)
			break
		}
	}
	
	// 3. 查找HTTP方法
	methodPattern := regexp.MustCompile(`\b(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b`)
	if method := methodPattern.FindString(line); method != "" {
		entry.Method = method
	}
	
	// 4. 查找状态码
	statusPattern := regexp.MustCompile(`"\s+(\d{3})\s+`)
	if matches := statusPattern.FindStringSubmatch(line); len(matches) > 1 {
		if code, _ := strconv.Atoi(matches[1]); code > 0 {
			entry.StatusCode = code
		}
	}
	
	return entry, nil
}

// inferFieldName 根据字段内容和位置推断字段名
func inferFieldName(field string, index, total int) string {
	field = strings.TrimSpace(field)
	
	// IP地址
	if regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`).MatchString(field) {
		return "client_ip"
	}
	
	// HTTP方法
	if regexp.MustCompile(`^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)$`).MatchString(field) {
		return "method"
	}
	
	// 时间戳
	if regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}`).MatchString(field) {
		return "timestamp"
	}
	
	// 路径
	if strings.HasPrefix(field, "/") {
		return "path"
	}
	
	// 数字字段（响应大小、响应时间或状态码）
	if regexp.MustCompile(`^\d+$`).MatchString(field) {
		val, _ := strconv.ParseInt(field, 10, 64)
		
		// 根据字段位置和值综合判断
		// 典型格式：ip,method,path,status_code,response_size,response_time,timestamp
		// 位置：    0    1      2   3           4             5             6
		
		if total >= 7 {
			// 标准7字段格式：优先按位置判断
			switch index {
			case 3:
				// 第4个字段：3位数字通常是状态码
				if val >= 100 && val <= 599 {
					return "status_code"
				}
			case 4:
				return "response_size"  // 第5个字段是响应大小
			case 5:
				return "response_time"  // 第6个字段是响应时间（即使是3位数字如495ms）
			}
		}
		
		// 根据字段特征判断（非标准格式或位置不匹配时）
		// 3位数字且在HTTP状态码范围内
		if val >= 100 && val <= 999 && len(field) == 3 {
			return "status_code"
		}
		
		// 倒数第2个字段：通常是 response_size（如果值较大）或 response_time（如果值较小）
		if index == total-2 {
			if val > 10000 {
				return "response_size"
			}
			return "response_time"
		}
		
		// 倒数第3个字段
		if index == total-3 {
			if val < 30000 {
				return "response_time"
			}
			return "response_size"
		}
		
		// 默认：小数值是响应时间，大数值是响应大小
		if val < 30000 {
			return "response_time"
		}
		return "response_size"
	}
	
	return ""
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
		// 判断是秒（通常带小数点，如 "2.319"）还是毫秒（整数，如 "2319"）
		if strings.Contains(value, ".") {
			// 包含小数点，认为是秒，转换为毫秒
			if rt, err := strconv.ParseFloat(value, 64); err == nil {
				entry.ResponseTime = int64(rt * 1000)
			}
		} else {
			// 整数，认为是毫秒，直接使用
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
