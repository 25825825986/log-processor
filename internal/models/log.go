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
	Source        string            `json:"source" db:"source"`
	Level         string            `json:"level" db:"level"`
	Method        string            `json:"method" db:"method"`
	Path          string            `json:"path" db:"path"`
	StatusCode    int               `json:"status_code" db:"status_code"`
	ResponseTime  int64             `json:"response_time" db:"response_time"`
	ClientIP      string            `json:"client_ip" db:"client_ip"`
	UserAgent     string            `json:"user_agent" db:"user_agent"`
	Referer       string            `json:"referer" db:"referer"`
	RequestSize   int64             `json:"request_size" db:"request_size"`
	ResponseSize  int64             `json:"response_size" db:"response_size"`
	ExtraFields   map[string]string `json:"extra_fields" db:"extra_fields"`
	RawData       string            `json:"raw_data" db:"raw_data"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
}

// NewLogEntry 创建新的日志条目
func NewLogEntry() *LogEntry {
	return &LogEntry{
		ID:          uuid.New().String(),
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
	StatusCodes  []int      `json:"status_codes,omitempty"`
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
