// models/log.go - 数据模型定义
package models

import (
	"encoding/json"
	"time"
)

// LogEntry 统一的日志条目结构
type LogEntry struct {
	ID            string            `json:"id" db:"id"`
	Timestamp     time.Time         `json:"timestamp" db:"timestamp"`
	
	// 项目维度（新增）
	ProjectID     string            `json:"project_id" db:"project_id"`
	ProjectName   string            `json:"project_name" db:"project_name"`
	Environment   string            `json:"environment" db:"environment"` // dev/test/staging/prod
	ServiceName   string            `json:"service_name" db:"service_name"` // 微服务名称
	
	// 日志分类（扩展）
	Source        string            `json:"source" db:"source"`
	Level         string            `json:"level" db:"level"` // DEBUG/INFO/WARN/ERROR/FATAL
	LogType       string            `json:"log_type" db:"log_type"` // api_call/system/middleware/error
	
	// 请求信息
	Method        string            `json:"method" db:"method"`
	Path          string            `json:"path" db:"path"`
	StatusCode    int               `json:"status_code" db:"status_code"`
	ResponseTime  int64             `json:"response_time" db:"response_time"`
	ClientIP      string            `json:"client_ip" db:"client_ip"`
	UserAgent     string            `json:"user_agent" db:"user_agent"`
	Referer       string            `json:"referer" db:"referer"`
	RequestSize   int64             `json:"request_size" db:"request_size"`
	ResponseSize  int64             `json:"response_size" db:"response_size"`
	
	// 错误追踪（新增）
	ErrorMessage  string            `json:"error_message" db:"error_message"`
	ErrorCode     string            `json:"error_code" db:"error_code"`
	StackTrace    string            `json:"stack_trace" db:"stack_trace"`
	
	// 上下文信息（新增）
	RequestID     string            `json:"request_id" db:"request_id"`
	UserID        string            `json:"user_id" db:"user_id"`
	SessionID     string            `json:"session_id" db:"session_id"`
	
	// AI 分析（新增）
	AIAnalyzed    bool              `json:"ai_analyzed" db:"ai_analyzed"`
	AIAnalysis    string            `json:"ai_analysis" db:"ai_analysis"` // JSON 格式
	AISuggestions string            `json:"ai_suggestions" db:"ai_suggestions"`
	AIAnalyzedAt  *time.Time        `json:"ai_analyzed_at" db:"ai_analyzed_at"`
	
	// 原有字段
	ExtraFields   map[string]string `json:"extra_fields" db:"extra_fields"`
	RawData       string            `json:"raw_data" db:"raw_data"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
}

// NewLogEntry 创建新的日志条目
func NewLogEntry() *LogEntry {
	return &LogEntry{
		ID:          NewUUID(),
		ExtraFields: make(map[string]string),
		CreatedAt:   time.Now(),
	}
}

// TableName 返回表名
func (l *LogEntry) TableName() string {
	return "logs"
}

// ToJSON 转换为JSON字符串
func (l *LogEntry) ToJSON() string {
	data, _ := json.Marshal(l)
	return string(data)
}

// FilterCondition 筛选条件
type FilterCondition struct {
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	Methods      []string   `json:"methods,omitempty"`
	Paths        []string   `json:"paths,omitempty"`
	StatusCodes      []int      `json:"status_codes,omitempty"`
	StatusCodeRanges []string   `json:"status_code_ranges,omitempty"`
	ClientIPs    []string   `json:"client_ips,omitempty"`
	MinResponseTime int64   `json:"min_response_time,omitempty"`
	MaxResponseTime int64   `json:"max_response_time,omitempty"`
	Level        string     `json:"level,omitempty"`
	Source       string     `json:"source,omitempty"`
	Keyword      string     `json:"keyword,omitempty"`
}

// ExportRequest 导出请求
type ExportRequest struct {
	Filter   FilterCondition `json:"filter"`
	Format   string          `json:"format"` // excel, csv, json
	FileName string          `json:"file_name,omitempty"`
}

// Statistics 统计信息
type Statistics struct {
	TotalCount      int64            `json:"total_count"`
	ErrorCount      int64            `json:"error_count"`
	AvgResponseTime float64          `json:"avg_response_time"`
	StatusCodeDist  map[int]int64    `json:"status_code_dist"`
	MethodDist      map[string]int64 `json:"method_dist"`
	TopPaths        []PathStat       `json:"top_paths"`
	TimeSeries      []TimePoint      `json:"time_series"`
}

type PathStat struct {
	Path  string `json:"path"`
	Count int64  `json:"count"`
}

type TimePoint struct {
	Time  string `json:"time"`
	Count int64  `json:"count"`
}

// ParseResult 解析结果
type ParseResult struct {
	Entry   *LogEntry
	Success bool
	Error   error
}
