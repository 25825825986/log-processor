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

var (
	// parseSyslog
	reSyslog    = regexp.MustCompile(`^(?P<month>\w{3})\s+(?P<day>\d+)\s+(?P<time>\d{2}:\d{2}:\d{2})\s+(?P<host>\S+)\s+(?P<process>[^\[:]+)(?:\[(?P<pid>\d+)\])?:\s+(?P<message>.+)$`)
	reSyslogMsg = regexp.MustCompile(`^(?P<client_ip>\S+)\s+(?P<method>\S+)\s+(?P<path>\S+)\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)`)

	// parseGeneric patterns
	reGeneric1 = regexp.MustCompile(`\[(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\]\s+(?P<client_ip>\S+)\s+(?P<method>\S+)\s+(?P<path>\S+)\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)\s+(?P<response_time>\d+)ms`)
	reGeneric2 = regexp.MustCompile(`(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\s+-\s+(?P<client_ip>\S+)\s+-\s+(?P<method>\S+)\s+(?P<path>\S+)\s+-\s+Status:\s+(?P<status_code>\d+)\s+-\s+Size:\s+(?P<response_size>\d+)\s+-\s+Time:\s+(?P<response_time>\d+)ms`)
	reGeneric3 = regexp.MustCompile(`\[(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\]\s+(?P<client_ip>\S+)\s+-\s+(?P<method>\S+)\s+(?P<path>\S+)\s+-\s+(?P<status_code>\d+)\s+-\s+(?P<response_size>\d+)\s+-\s+(?P<response_time>\d+)ms`)
	reGeneric4 = regexp.MustCompile(`(?P<client_ip>\S+)\s+\[(?P<timestamp>[^\]]+)\]\s+"(?P<method>\S+)\s+(?P<path>\S+)"\s+(?P<status_code>\d+)\s+(?P<response_size>\d+)\s+(?P<response_time>\d+)`)
	reGeneric5 = regexp.MustCompile(`Request from (?P<client_ip>\S+) at (?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}):\s+(?P<method>\S+)\s+(?P<path>\S+)\s+->\s+(?P<status_code>\d+)\s+\((?P<response_size>\d+) bytes, (?P<response_time>\d+)ms\)`)

	// parseGeneric fallback extractors
	reIP          = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	reTimestamp1  = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)
	reTimestamp2  = regexp.MustCompile(`\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2}`)
	reMethod      = regexp.MustCompile(`\b(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b`)
	reStatusCode  = regexp.MustCompile(`"\s+(\d{3})\s+`)
	reStatusCode2 = regexp.MustCompile(`(?i)(?:status[:\s]+)(\d{3})`)
	reSize        = regexp.MustCompile(`(?i)(?:size[:\s]+)(\d+)`)
	reTime        = regexp.MustCompile(`(?i)(?:time[:\s]+)(\d+)`)
	rePath        = regexp.MustCompile(`\s(/[\w/._~%&?=-]*)`)

	// inferFieldName patterns
	reInferIP        = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	reInferMethod    = regexp.MustCompile(`^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)$`)
	reInferTimestamp = regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}`)
	reInferDigit     = regexp.MustCompile(`^\d+$`)
)

// Parser 日志解析器接口
type Parser interface {
	Parse(line string) (*models.LogEntry, error)
	SetConfig(cfg config.ParserConfig)
}

// LogParser 日志解析器实现
type LogParser struct {
	config     config.ParserConfig
	regex      *regexp.Regexp
	jsonParser *JSONParser
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
	format := strings.TrimSpace(strings.ToLower(p.config.Format))
	if format == "" || format == "auto" {
		format = DetectFormat(line)
	}
	return p.ParseWithFormat(line, format)
}

// ParseWithFormat 使用指定格式解析日志行，适合批量导入时复用已检测格式。
func (p *LogParser) ParseWithFormat(line string, format string) (*models.LogEntry, error) {
	entry := models.NewLogEntry()
	entry.RawData = line

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

	// 有效性校验：如果一条分隔符日志完全无法提取任何有效字段，
	// 大概率是表头行或格式不符，视为解析失败
	if entry.Timestamp.IsZero() && entry.ClientIP == "" && entry.Method == "" && entry.StatusCode == 0 {
		return entry, fmt.Errorf("无法识别分隔符日志格式，可能是表头行或字段内容不匹配")
	}

	return entry, nil
}

// parseSyslog 解析Syslog格式
func (p *LogParser) parseSyslog(line string, entry *models.LogEntry) (*models.LogEntry, error) {
	matches := reSyslog.FindStringSubmatch(line)
	if matches == nil {
		return entry, fmt.Errorf("failed to parse syslog format")
	}

	names := reSyslog.SubexpNames()
	for i, name := range names {
		if i > 0 && i < len(matches) && name != "" {
			p.setField(entry, name, matches[i])
		}
	}

	message := ""
	for i, name := range names {
		if name == "message" && i < len(matches) {
			message = matches[i]
			break
		}
	}

	if message != "" {
		if msgMatches := reSyslogMsg.FindStringSubmatch(message); msgMatches != nil {
			msgNames := reSyslogMsg.SubexpNames()
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
	if matches := reGeneric1.FindStringSubmatch(line); matches != nil {
		names := reGeneric1.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}

	// 模式2: time - IP - METHOD PATH - Status: STATUS - Size: SIZE - Time: TIMEms
	if matches := reGeneric2.FindStringSubmatch(line); matches != nil {
		names := reGeneric2.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}

	// 模式3: [time] IP - METHOD PATH - STATUS - SIZE - TIMEms
	if matches := reGeneric3.FindStringSubmatch(line); matches != nil {
		names := reGeneric3.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}

	// 模式4: IP [time] "METHOD PATH" STATUS SIZE TIME
	if matches := reGeneric4.FindStringSubmatch(line); matches != nil {
		names := reGeneric4.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}

	// 模式5: Request from IP at time: METHOD PATH -> STATUS (SIZE bytes, TIMEms)
	if matches := reGeneric5.FindStringSubmatch(line); matches != nil {
		names := reGeneric5.SubexpNames()
		for i, name := range names {
			if i > 0 && i < len(matches) && name != "" {
				p.setField(entry, name, matches[i])
			}
		}
		return entry, nil
	}

	// 兜底：尝试提取可能的字段
	if ip := reIP.FindString(line); ip != "" {
		entry.ClientIP = ip
	}

	timestampPatterns := []*regexp.Regexp{reTimestamp1, reTimestamp2}
	for _, re := range timestampPatterns {
		if t := re.FindString(line); t != "" {
			entry.Timestamp = p.parseTime(t)
			break
		}
	}

	if method := reMethod.FindString(line); method != "" {
		entry.Method = method
	}

	if matches := reStatusCode.FindStringSubmatch(line); len(matches) > 1 {
		if code, _ := strconv.Atoi(matches[1]); code > 0 {
			entry.StatusCode = code
		}
	}

	if entry.StatusCode == 0 {
		if matches := reStatusCode2.FindStringSubmatch(line); len(matches) > 1 {
			if code, _ := strconv.Atoi(matches[1]); code > 0 {
				entry.StatusCode = code
			}
		}
	}

	if entry.ResponseSize == 0 {
		if matches := reSize.FindStringSubmatch(line); len(matches) > 1 {
			if size, _ := strconv.ParseInt(matches[1], 10, 64); size > 0 {
				entry.ResponseSize = size
			}
		}
	}

	if entry.ResponseTime == 0 {
		if matches := reTime.FindStringSubmatch(line); len(matches) > 1 {
			if rt, _ := strconv.ParseInt(matches[1], 10, 64); rt > 0 {
				entry.ResponseTime = rt
			}
		}
	}

	if entry.Path == "" {
		if matches := rePath.FindStringSubmatch(line); len(matches) > 1 {
			entry.Path = matches[1]
		}
	}

	if entry.Timestamp.IsZero() && entry.ClientIP == "" && entry.Method == "" && entry.StatusCode == 0 {
		return entry, fmt.Errorf("无法识别日志格式，未提取到有效字段")
	}

	return entry, nil
}

// inferFieldName 根据字段内容和位置推断字段名
func inferFieldName(field string, index, total int) string {
	field = strings.TrimSpace(field)

	// IP地址
	if reInferIP.MatchString(field) {
		return "client_ip"
	}

	// HTTP方法
	if reInferMethod.MatchString(field) {
		return "method"
	}

	// 时间戳
	if reInferTimestamp.MatchString(field) {
		return "timestamp"
	}

	// 路径
	if strings.HasPrefix(field, "/") {
		return "path"
	}

	// 数字字段（响应大小、响应时间或状态码）
	if reInferDigit.MatchString(field) {
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
				return "response_size" // 第5个字段是响应大小
			case 5:
				return "response_time" // 第6个字段是响应时间（即使是3位数字如495ms）
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
