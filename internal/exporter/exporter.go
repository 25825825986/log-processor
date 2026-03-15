// exporter/exporter.go - 数据导出器
package exporter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log-processor/internal/models"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportOptions 导出选项
type ExportOptions struct {
	TimeFormat string // 时间格式，如 "2006-01-02 15:04:05"
}

// Exporter 导出器接口
type Exporter interface {
	Export(entries []*models.LogEntry, outputPath string, opts *ExportOptions) error
	GetContentType() string
	GetExtension() string
}

// ExcelExporter Excel导出器
type ExcelExporter struct{}

// NewExcelExporter 创建Excel导出器
func NewExcelExporter() *ExcelExporter {
	return &ExcelExporter{}
}

// Export 导出为Excel
func (e *ExcelExporter) Export(entries []*models.LogEntry, outputPath string, opts *ExportOptions) error {
	f := excelize.NewFile()
	sheetName := "Logs"
	f.SetSheetName("Sheet1", sheetName)

	// 设置表头
	headers := []string{
		"ID", "Timestamp", "Source", "Level", "Method", "Path",
		"Status Code", "Response Time (ms)", "Client IP", "User Agent",
		"Referer", "Request Size", "Response Size", "Raw Data",
	}

	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 设置表头样式
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
	})
	f.SetRowStyle(sheetName, 1, 1, style)

	// 确定时间格式
	timeFormat := time.RFC3339
	if opts != nil && opts.TimeFormat != "" {
		timeFormat = opts.TimeFormat
	}

	// 写入数据
	for i, entry := range entries {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), entry.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entry.Timestamp.Format(timeFormat))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), entry.Source)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), entry.Level)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), entry.Method)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), entry.Path)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), entry.StatusCode)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), entry.ResponseTime)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), entry.ClientIP)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), entry.UserAgent)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), entry.Referer)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), entry.RequestSize)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), entry.ResponseSize)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), entry.RawData)
	}

	// 自动调整列宽
	for col := 1; col <= len(headers); col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 1)
		f.SetColWidth(sheetName, cell[:1], cell[:1], 18)
	}

	return f.SaveAs(outputPath)
}

// GetContentType 返回Content-Type
func (e *ExcelExporter) GetContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// GetExtension 返回文件扩展名
func (e *ExcelExporter) GetExtension() string {
	return ".xlsx"
}

// CSVExporter CSV导出器
type CSVExporter struct{}

// NewCSVExporter 创建CSV导出器
func NewCSVExporter() *CSVExporter {
	return &CSVExporter{}
}

// Export 导出为CSV
func (e *CSVExporter) Export(entries []*models.LogEntry, outputPath string, opts *ExportOptions) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{
		"ID", "Timestamp", "Source", "Level", "Method", "Path",
		"Status Code", "Response Time (ms)", "Client IP", "User Agent",
		"Referer", "Request Size", "Response Size", "Raw Data",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// 确定时间格式
	timeFormat := time.RFC3339
	if opts != nil && opts.TimeFormat != "" {
		timeFormat = opts.TimeFormat
	}

	// 写入数据
	for _, entry := range entries {
		record := []string{
			entry.ID,
			entry.Timestamp.Format(timeFormat),
			entry.Source,
			entry.Level,
			entry.Method,
			entry.Path,
			strconv.Itoa(entry.StatusCode),
			strconv.FormatInt(entry.ResponseTime, 10),
			entry.ClientIP,
			entry.UserAgent,
			entry.Referer,
			strconv.FormatInt(entry.RequestSize, 10),
			strconv.FormatInt(entry.ResponseSize, 10),
			entry.RawData,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// GetContentType 返回Content-Type
func (e *CSVExporter) GetContentType() string {
	return "text/csv"
}

// GetExtension 返回文件扩展名
func (e *CSVExporter) GetExtension() string {
	return ".csv"
}

// JSONExporter JSON导出器
type JSONExporter struct{}

// JSONEntry 带格式化时间的日志条目
type JSONEntry struct {
	ID           string            `json:"id"`
	Timestamp    string            `json:"timestamp"`
	Source       string            `json:"source"`
	Level        string            `json:"level"`
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	StatusCode   int               `json:"status_code"`
	ResponseTime int64             `json:"response_time"`
	ClientIP     string            `json:"client_ip"`
	UserAgent    string            `json:"user_agent"`
	Referer      string            `json:"referer"`
	RequestSize  int64             `json:"request_size"`
	ResponseSize int64             `json:"response_size"`
	ExtraFields  map[string]string `json:"extra_fields,omitempty"`
	RawData      string            `json:"raw_data"`
	CreatedAt    string            `json:"created_at"`
}

// NewJSONExporter 创建JSON导出器
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

// Export 导出为JSON
func (e *JSONExporter) Export(entries []*models.LogEntry, outputPath string, opts *ExportOptions) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 确定时间格式
	timeFormat := time.RFC3339
	if opts != nil && opts.TimeFormat != "" {
		timeFormat = opts.TimeFormat
	}

	// 转换为带格式化时间的结构
	jsonEntries := make([]JSONEntry, len(entries))
	for i, entry := range entries {
		jsonEntries[i] = JSONEntry{
			ID:           entry.ID,
			Timestamp:    entry.Timestamp.Format(timeFormat),
			Source:       entry.Source,
			Level:        entry.Level,
			Method:       entry.Method,
			Path:         entry.Path,
			StatusCode:   entry.StatusCode,
			ResponseTime: entry.ResponseTime,
			ClientIP:     entry.ClientIP,
			UserAgent:    entry.UserAgent,
			Referer:      entry.Referer,
			RequestSize:  entry.RequestSize,
			ResponseSize: entry.ResponseSize,
			ExtraFields:  entry.ExtraFields,
			RawData:      entry.RawData,
			CreatedAt:    entry.CreatedAt.Format(timeFormat),
		}
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonEntries)
}

// GetContentType 返回Content-Type
func (e *JSONExporter) GetContentType() string {
	return "application/json"
}

// GetExtension 返回文件扩展名
func (e *JSONExporter) GetExtension() string {
	return ".json"
}

// ExportManager 导出管理器
type ExportManager struct{}

// NewExportManager 创建导出管理器
func NewExportManager() *ExportManager {
	return &ExportManager{}
}

// Export 根据格式导出数据
func (m *ExportManager) Export(entries []*models.LogEntry, format, outputPath string, opts *ExportOptions) (string, error) {
	var exporter Exporter

	switch format {
	case "excel":
		exporter = NewExcelExporter()
	case "csv":
		exporter = NewCSVExporter()
	case "json":
		exporter = NewJSONExporter()
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	if err := exporter.Export(entries, outputPath, opts); err != nil {
		return "", err
	}

	return exporter.GetContentType(), nil
}

// GetSupportedFormats 返回支持的导出格式
func (m *ExportManager) GetSupportedFormats() []string {
	return []string{"excel", "csv", "json"}
}
