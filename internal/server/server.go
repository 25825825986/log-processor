// server/server.go - Web服务器
package server

import (
	"fmt"
	"io"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/exporter"
	"log-processor/internal/models"
	"log-processor/internal/parser"
	"log-processor/internal/processor"
	"log-processor/internal/receiver"
	"log-processor/internal/storage"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Server Web服务器
type Server struct {
	config         *config.Config
	router         *gin.Engine
	storage        storage.Storage
	parser         *parser.LogParser
	processor      *processor.Processor
	receiver       *receiver.Manager
	exportManager  *exporter.ExportManager
}

// NewServer 创建新服务器
func NewServer(cfg *config.Config, store storage.Storage, proc *processor.Processor, recv *receiver.Manager, logFile *os.File) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	// 使用自定义 Logger，同时输出到终端和文件，并添加描述
	loggerConfig := customLoggerConfig(io.MultiWriter(os.Stdout, logFile))
	router.Use(gin.LoggerWithConfig(loggerConfig))

	s := &Server{
		config:        cfg,
		router:        router,
		storage:       store,
		parser:        parser.NewLogParser(cfg.GetParserConfig()),
		processor:     proc,
		receiver:      recv,
		exportManager: exporter.NewExportManager(),
	}

	s.setupRoutes()
	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// CORS 中间件
	s.router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
	
	// 静态文件
	s.router.Static("/static", "./web")
	s.router.LoadHTMLFiles("./web/index.html")

	// 页面路由
	s.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// API路由组
	api := s.router.Group("/api")
	{
		// 配置管理
		api.GET("/config", s.getConfig)
		api.POST("/config", s.updateConfig)

		// 日志查询
		api.GET("/logs", s.queryLogs)
		api.POST("/logs/import", s.importLogs)
		api.DELETE("/logs/:id", s.deleteLog)
		api.DELETE("/logs", s.clearLogs)

		// 统计分析
		api.GET("/statistics", s.getStatistics)

		// 导出
		api.POST("/export", s.exportLogs)
		api.GET("/export/formats", s.getExportFormats)

		// 系统状态
		api.GET("/status", s.getStatus)

		// 接收器控制
		api.POST("/receiver/start", s.startReceiver)
		api.POST("/receiver/stop", s.stopReceiver)
	}
}

// Run 启动服务器
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	log.Printf("Web server starting on http://%s", addr)
	return s.router.Run(addr)
}

// getConfig 获取配置（过滤敏感信息）
func (s *Server) getConfig(c *gin.Context) {
	cfg := s.config.Get()
	
	// 出于安全考虑，不返回敏感配置（如认证Token）
	c.JSON(http.StatusOK, gin.H{
		"server": cfg.Server,
		"parser": cfg.Parser,
		"processor": cfg.Processor,
		"storage": cfg.Storage,
		"receiver": gin.H{
			"tcp_enabled":       cfg.Receiver.TCPEnabled,
			"tcp_port":          cfg.Receiver.TCPPort,
			"udp_enabled":       cfg.Receiver.UDPEnabled,
			"udp_port":          cfg.Receiver.UDPPort,
			"http_enabled":      cfg.Receiver.HTTPEnabled,
			"http_port":         cfg.Receiver.HTTPPort,
			"http_auth_token":   cfg.Receiver.HTTPAuthToken != "", // 只返回是否启用，不返回值
			"http_allowed_ips":  cfg.Receiver.HTTPAllowedIPs,
			"http_rate_limit":   cfg.Receiver.HTTPRateLimit,
			"buffer_size":       cfg.Receiver.BufferSize,
		},
	})
}

// updateConfig 更新配置
func (s *Server) updateConfig(c *gin.Context) {
	var newConfig config.Config
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.config.Update(&newConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新解析器配置
	s.parser.SetConfig(newConfig.Parser)

	// 更新处理器配置
	s.processor.UpdateConfig(newConfig.Processor)
	
	// 更新处理器的解析器（因为解析器配置已改变）
	s.processor.SetParser(s.parser)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// queryLogs 查询日志
func (s *Server) queryLogs(c *gin.Context) {
	var filter models.FilterCondition

	// 解析查询参数
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = &t
		}
	}
	if methods := c.QueryArray("methods"); len(methods) > 0 {
		filter.Methods = methods
	}
	if paths := c.QueryArray("paths"); len(paths) > 0 {
		filter.Paths = paths
	}
	if codes := c.QueryArray("status_codes"); len(codes) > 0 {
		for _, code := range codes {
			if i, err := strconv.Atoi(code); err == nil {
				filter.StatusCodes = append(filter.StatusCodes, i)
			}
		}
	}
	filter.Keyword = c.Query("keyword")
	filter.Level = c.Query("level")
	filter.Source = c.Query("source")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	entries, err := s.storage.Query(filter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	count, _ := s.storage.Count(filter)

	c.JSON(http.StatusOK, gin.H{
		"data":  entries,
		"total": count,
		"limit": limit,
		"offset": offset,
	})
}

// importLogs 导入日志文件
func (s *Server) importLogs(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// 保存临时文件
	tempPath := filepath.Join("./temp", file.Filename)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	currentFormat := s.config.Get().Parser.Format
	log.Printf("[IMPORT] 开始导入文件: %s, 当前解析格式: %s", file.Filename, currentFormat)

	// 导入文件 - 使用同步处理避免 channel panic
	importer := receiver.NewFileImporter()
	lines := make([]string, 0)
	
	// 先读取所有行
	_, err = importer.ImportFile(tempPath, func(line string) bool {
		lines = append(lines, line)
		return true
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(lines) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"lines":    0,
			"accepted": 0,
			"file":     file.Filename,
			"warning":  "文件为空",
		})
		return
	}

	log.Printf("[IMPORT] 读取到 %d 行数据", len(lines))

	// 检测文件格式（跳过注释行和空行）
	detectedFormat := detectFileFormat(lines)
	log.Printf("[IMPORT] 检测到文件格式: %s, 当前配置: %s", detectedFormat, currentFormat)

	// 检查格式是否匹配
	if !isFormatCompatible(detectedFormat, currentFormat) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "warning",
			"lines":    len(lines),
			"accepted": 0,
			"file":     file.Filename,
			"warning":  fmt.Sprintf("文件格式为 [%s]，但当前配置为 [%s]。请前往「配置」页面修改解析格式后再导入。", detectedFormat, currentFormat),
			"detected_format": detectedFormat,
			"current_format":  currentFormat,
		})
		return
	}

	// 再提交到处理器（跳过注释行和空行）
	successCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过空行和注释行
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if s.processor.Submit(line) {
			successCount++
		}
	}

	log.Printf("[IMPORT] 成功提交 %d 行到处理器", successCount)
	// 等待处理器处理完成（简单等待1秒）
	time.Sleep(1 * time.Second)

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"lines":      len(lines),
		"accepted":   successCount,
		"file":       file.Filename,
	})
}

// detectFileFormat 检测文件格式（跳过注释行和空行）
func detectFileFormat(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过空行和注释行
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return detectLogFormat(trimmed)
	}
	return "unknown"
}

// detectLogFormat 检测单行日志格式
func detectLogFormat(line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) == 0 {
		return "unknown"
	}

	// 检测 JSON 格式
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		return "json"
	}

	// 检测 CSV 格式（简单判断是否有多个逗号分隔）
	if strings.Count(trimmed, ",") > 2 && !strings.Contains(trimmed, " ") {
		return "csv"
	}

	// 检测 Nginx/Apache 格式（包含IP地址和时间戳格式）
	// 典型特征: IP地址 + - - + [时间]
	if strings.Contains(trimmed, " - - [") && strings.Contains(trimmed, "\"") {
		return "nginx"
	}

	// 检测是否包含常见日志字段
	if strings.Contains(trimmed, "GET ") || strings.Contains(trimmed, "POST ") {
		if strings.Contains(trimmed, "HTTP/1.") {
			return "nginx"
		}
	}

	// 检测 Syslog 格式
	if strings.Contains(trimmed, "]: ") && (strings.HasPrefix(trimmed, "<") || strings.Contains(trimmed, ": ")) {
		return "syslog"
	}

	return "unknown"
}

// isFormatCompatible 检查文件格式与配置是否兼容
func isFormatCompatible(fileFormat, configFormat string) bool {
	// 完全匹配
	if fileFormat == configFormat {
		return true
	}

	// 特殊兼容规则
	switch configFormat {
	case "custom":
		// custom 格式可以处理多种格式
		return true
	case "nginx", "apache":
		// nginx 和 apache 格式相似，可以互相兼容
		return fileFormat == "nginx" || fileFormat == "apache"
	}

	return false
}

// deleteLog 删除单条日志
func (s *Server) deleteLog(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	if err := s.storage.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Log deleted"})
}

// clearLogs 清空所有日志
func (s *Server) clearLogs(c *gin.Context) {
	if err := s.storage.Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "All logs cleared"})
}

// getStatistics 获取统计信息
func (s *Server) getStatistics(c *gin.Context) {
	var filter models.FilterCondition

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = &t
		}
	}

	stats, err := s.storage.Statistics(filter)
	if err != nil {
		log.Printf("[API] Statistics query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[API] Statistics: total=%d, errors=%d, avg_response=%.2fms",
		stats.TotalCount, stats.ErrorCount, stats.AvgResponseTime)
	c.JSON(http.StatusOK, stats)
}

// exportLogs 导出日志
func (s *Server) exportLogs(c *gin.Context) {
	var req models.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[EXPORT] 解析请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[EXPORT] 筛选条件: StartTime=%v, EndTime=%v, StatusCodes=%v", 
		req.Filter.StartTime, req.Filter.EndTime, req.Filter.StatusCodes)

	// 查询数据
	entries, err := s.storage.Query(req.Filter, 10000, 0)
	if err != nil {
		log.Printf("[EXPORT] 查询失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[EXPORT] 查询到 %d 条记录", len(entries))

	if len(entries) == 0 {
		c.JSON(http.StatusOK, gin.H{"error": "没有符合条件的数据"})
		return
	}

	// 生成文件名
	format := req.Format
	if format == "" {
		format = "excel"
	}
	filename := req.FileName
	if filename == "" {
		filename = fmt.Sprintf("logs_%s", time.Now().Format("20060102_150405"))
	}

	exporter, ok := s.exportManager.GetExporter(format)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported format"})
		return
	}

	outputPath := filepath.Join("./exports", filename+exporter.GetExtension())
	if err := exporter.Export(entries, outputPath); err != nil {
		log.Printf("[EXPORT] 导出失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.FileAttachment(outputPath, filename+exporter.GetExtension())
}

// getExportFormats 获取支持的导出格式
func (s *Server) getExportFormats(c *gin.Context) {
	formats := s.exportManager.GetSupportedFormats()
	c.JSON(http.StatusOK, gin.H{"formats": formats})
}

// getStatus 获取系统状态（过滤敏感信息）
func (s *Server) getStatus(c *gin.Context) {
	cfg := s.config.Get()
	stats := s.processor.GetStats()

	// 只返回基本配置信息，过滤敏感字段
	c.JSON(http.StatusOK, gin.H{
		"config": gin.H{
			"server":    cfg.Server,
			"parser":    cfg.Parser,
			"processor": cfg.Processor,
			"receiver": gin.H{
				"tcp_enabled":  cfg.Receiver.TCPEnabled,
				"tcp_port":     cfg.Receiver.TCPPort,
				"udp_enabled":  cfg.Receiver.UDPEnabled,
				"udp_port":     cfg.Receiver.UDPPort,
				"http_enabled": cfg.Receiver.HTTPEnabled,
				"http_port":    cfg.Receiver.HTTPPort,
			},
			"storage": cfg.Storage,
		},
		"processor": stats,
		"timestamp": time.Now(),
	})
}

// startReceiver 启动接收器
func (s *Server) startReceiver(c *gin.Context) {
	// 接收器已在启动时运行，这里可以添加额外的控制逻辑
	c.JSON(http.StatusOK, gin.H{"status": "already running"})
}

// stopReceiver 停止接收器
func (s *Server) stopReceiver(c *gin.Context) {
	// 接收器控制逻辑
	c.JSON(http.StatusOK, gin.H{"status": "not implemented"})
}

// customLoggerConfig 返回自定义的 Gin Logger 配置，添加简短描述
func customLoggerConfig(writer io.Writer) gin.LoggerConfig {
	return gin.LoggerConfig{
		Output: writer,
		Formatter: func(param gin.LogFormatterParams) string {
			// 根据状态码和方法生成简短描述
			desc := getAccessDescription(param.StatusCode, param.Method, param.Path)
			
			return fmt.Sprintf("[ACCESS] %s | %3d | %13v | %15s | %-7s %s | %s\n",
				param.TimeStamp.Format("2006/01/02 15:04:05"),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
				desc,
			)
		},
	}
}

// getAccessDescription 根据状态码和方法返回简短描述
func getAccessDescription(statusCode int, method, path string) string {
	// 首先根据状态码判断
	switch {
	case statusCode >= 500:
		return "[服务器错误]"
	case statusCode == 404:
		return "[资源未找到]"
	case statusCode == 403:
		return "[禁止访问]"
	case statusCode == 401:
		return "[未授权]"
	case statusCode >= 400:
		return "[请求错误]"
	case statusCode >= 300:
		return "[重定向]"
	}
	
	// 200/201 成功状态，根据路径和方法进一步描述
	if statusCode >= 200 && statusCode < 300 {
		// 静态资源
		if path == "/" || path == "/index.html" || path == "/favicon.ico" {
			return "[访问首页]"
		}
		if path == "/static/css/style.css" || path == "/static/js/app.js" {
			return "[加载资源]"
		}
		
		// API 接口
		switch path {
		case "/api/config":
			return "[获取配置]"
		case "/api/statistics":
			return "[获取统计]"
		case "/api/logs":
			if method == "GET" {
				return "[查询日志]"
			}
			return "[清空日志]"
		case "/api/logs/import":
			return "[导入日志]"
		case "/api/export":
			return "[导出数据]"
		case "/api/status":
			return "[获取状态]"
		default:
			// 处理带参数的日志删除
			if len(path) > 10 && path[:10] == "/api/logs/" {
				return "[删除日志]"
			}
			return "[接口调用]"
		}
	}
	
	return "[未知操作]"
}
