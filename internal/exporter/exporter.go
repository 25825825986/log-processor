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

// Exporter 导出器接口
type Exporter interface {
	Export(entries []*models.LogEntry, outputPath string) error
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
func (e *ExcelExporter) Export(entries []*models.LogEntry, outputPath string) error {
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

	// 写入数据
	for i, entry := range entries {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), entry.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entry.Timestamp.Format(time.RFC3339))
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

	// 调整列宽
	for col := 1; col <= len(headers); col++ {
		colName, _ := excelize.ColumnNumberToName(col)
		f.SetColWidth(sheetName, colName, colName, 20)
	}

	// 确保目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
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
func (e *CSVExporter) Export(entries []*models.LogEntry, outputPath string) error {
	// 确保目录存在
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

	// 写入数据
	for _, entry := range entries {
		record := []string{
			entry.ID,
			entry.Timestamp.Format(time.RFC3339),
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

// NewJSONExporter 创建JSON导出器
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

// Export 导出为JSON
func (e *JSONExporter) Export(entries []*models.LogEntry, outputPath string) error {
	// 确保目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
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
type ExportManager struct {
	exporters map[string]Exporter
}

// NewExportManager 创建导出管理器
func NewExportManager() *ExportManager {
	m := &ExportManager{
		exporters: make(map[string]Exporter),
	}
	m.Register("excel", NewExcelExporter())
	m.Register("csv", NewCSVExporter())
	m.Register("json", NewJSONExporter())
	return m
}

// Register 注册导出器
func (m *ExportManager) Register(format string, exporter Exporter) {
	m.exporters[format] = exporter
}

// GetExporter 获取导出器
func (m *ExportManager) GetExporter(format string) (Exporter, bool) {
	exporter, ok := m.exporters[format]
	return exporter, ok
}

// Export 导出数据
func (m *ExportManager) Export(format string, entries []*models.LogEntry, outputPath string) error {
	exporter, ok := m.exporters[format]
	if !ok {
		return fmt.Errorf("unsupported format: %s", format)
	}
	return exporter.Export(entries, outputPath)
}

// GetSupportedFormats 获取支持的格式列表
func (m *ExportManager) GetSupportedFormats() []string {
	formats := make([]string, 0, len(m.exporters))
	for format := range m.exporters {
		formats = append(formats, format)
	}
	return formats
}
