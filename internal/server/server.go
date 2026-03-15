// server/server.go - Web服务器
package server

import (
	"encoding/json"
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
	configPath     string
	router         *gin.Engine
	storage        storage.Storage
	parser         *parser.LogParser
	processor      *processor.Processor
	receiver       *receiver.Manager
	exportManager  *exporter.ExportManager
}

// NewServer 创建新服务器
func NewServer(cfg *config.Config, store storage.Storage, proc *processor.Processor, recv *receiver.Manager, logFile *os.File, configPath string) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	// 使用自定义 Logger，同时输出到终端和文件，并添加描述
	loggerConfig := customLoggerConfig(io.MultiWriter(os.Stdout, logFile))
	router.Use(gin.LoggerWithConfig(loggerConfig))

	s := &Server{
		config:        cfg,
		configPath:    configPath,
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

		// 存储管理
		api.GET("/storage/info", s.getStorageInfo)
		api.POST("/storage/compact", s.compactStorage)
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
			"tcp_enabled":         cfg.Receiver.TCPEnabled,
			"tcp_port":            cfg.Receiver.TCPPort,
			"udp_enabled":         cfg.Receiver.UDPEnabled,
			"udp_port":            cfg.Receiver.UDPPort,
			"http_enabled":        cfg.Receiver.HTTPEnabled,
			"http_port":           cfg.Receiver.HTTPPort,
			"http_auth_token":     cfg.Receiver.HTTPAuthToken,  // 返回实际值（为空则不启用）
			"http_allowed_ips":    cfg.Receiver.HTTPAllowedIPs,
			"http_rate_limit":     cfg.Receiver.HTTPRateLimit,
			"http_max_body_size":  cfg.Receiver.HTTPMaxBodySize,
			"buffer_size":         cfg.Receiver.BufferSize,
			"file_watcher_enabled": cfg.Receiver.FileWatcherEnabled,
			"watch_paths":         cfg.Receiver.WatchPaths,
			"max_connections":     cfg.Receiver.MaxConnections,
		},
	})
}

// updateConfig 更新配置
func (s *Server) updateConfig(c *gin.Context) {
	// 使用 map 接收 JSON，避免直接绑定到带有 sync.RWMutex 的 Config 结构体
	var jsonConfig map[string]interface{}
	if err := c.ShouldBindJSON(&jsonConfig); err != nil {
		log.Printf("[ERROR] 解析配置 JSON 失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "配置格式错误: " + err.Error()})
		return
	}
	
	// 将 map 转换为 JSON 再解析到 Config 结构体
	jsonData, err := json.Marshal(jsonConfig)
	if err != nil {
		log.Printf("[ERROR] 序列化配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "配置处理失败"})
		return
	}
	
	var newConfig config.Config
	if err := json.Unmarshal(jsonData, &newConfig); err != nil {
		log.Printf("[ERROR] 解析配置结构失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "配置字段错误: " + err.Error()})
		return
	}

	if err := s.config.Update(&newConfig); err != nil {
		log.Printf("[ERROR] 更新配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "配置更新失败: " + err.Error()})
		return
	}

	// 保存配置到文件
	if err := s.config.SaveToFile(s.configPath); err != nil {
		log.Printf("[ERROR] 保存配置到文件失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内存中配置已更新，但保存到文件失败: " + err.Error()})
		return
	}
	log.Printf("[INFO] 配置已保存到: %s", s.configPath)

	// 在更新配置前，先获取旧的接收器配置
	oldRecvCfg := s.config.Get().Receiver
	newRecvCfg := newConfig.Receiver
	receiverChanged := !compareReceiverConfig(oldRecvCfg, newRecvCfg)
	
	log.Printf("[DEBUG] 旧接收器配置: TCP=%v:%d UDP=%v:%d HTTP=%v:%d", 
		oldRecvCfg.TCPEnabled, oldRecvCfg.TCPPort,
		oldRecvCfg.UDPEnabled, oldRecvCfg.UDPPort,
		oldRecvCfg.HTTPEnabled, oldRecvCfg.HTTPPort)
	log.Printf("[DEBUG] 新接收器配置: TCP=%v:%d UDP=%v:%d HTTP=%v:%d", 
		newRecvCfg.TCPEnabled, newRecvCfg.TCPPort,
		newRecvCfg.UDPEnabled, newRecvCfg.UDPPort,
		newRecvCfg.HTTPEnabled, newRecvCfg.HTTPPort)
	log.Printf("[DEBUG] 接收器配置是否变更: %v", receiverChanged)
	
	// 更新解析器配置
	s.parser.SetConfig(newConfig.Parser)

	// 更新处理器配置
	s.processor.UpdateConfig(newConfig.Processor)
	
	// 更新处理器的解析器（因为解析器配置已改变）
	s.processor.SetParser(s.parser)

	// 如果接收器配置变更，重启接收器
	if receiverChanged {
		log.Printf("[INFO] 接收器配置已变更，正在重启接收器...")
		if err := s.restartReceivers(newRecvCfg); err != nil {
			log.Printf("[ERROR] 重启接收器失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "partial",
				"message": "配置已更新，但接收器重启动失败: " + err.Error(),
			})
			return
		}
		log.Printf("[OK] 接收器已重新启动")
	}

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

	// 获取导入前的日志总数
	statsBefore, _ := s.storage.Statistics(models.FilterCondition{})
	countBefore := int64(0)
	if statsBefore != nil {
		countBefore = statsBefore.TotalCount
	}
	
	// 再提交到处理理器（跳过注释行和空行）
	successCount := 0
	droppedCount := 0
	batchSize := 1000
	batchInterval := 100 * time.Millisecond // 每批1小时休息100ms
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过空行和注释行
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}
		
		// 尝试提交，如果失败（队列满）则等待重试
		retryCount := 0
		submitted := false
		for retryCount < 3 {
			if s.processor.Submit(line) {
				successCount++
				submitted = true
				break
			}
			retryCount++
			time.Sleep(50 * time.Millisecond) // 短暂等待后重试
		}
		
		if !submitted {
			droppedCount++
		}
		
		// 每批处理完后休息一下，避免塞满队列
		if i > 0 && i%batchSize == 0 {
			time.Sleep(batchInterval)
		}
	}

	if droppedCount > 0 {
		log.Printf("[IMPORT] 警告: 丢弃 %d 条日志（队列满）", droppedCount)
	}
	log.Printf("[IMPORT] 成功提交 %d 行到处理器", successCount)
	
	// 等待处理器处理完成（根据数据量计算等待时间）
	waitTime := time.Duration(successCount/500+2) * time.Second
	log.Printf("[IMPORT] 等待 %v 让处理器完成处理...", waitTime)
	time.Sleep(waitTime)
	
	// 获取导入后的实际日志总数
	statsAfter, _ := s.storage.Statistics(models.FilterCondition{})
	countAfter := int64(0)
	if statsAfter != nil {
		countAfter = statsAfter.TotalCount
	}
	actualImported := countAfter - countBefore
	
	if actualImported < int64(successCount) {
		log.Printf("[IMPORT] 警告: 提交 %d 条，实际导入 %d 条（可能有 %d 条解析失败）", 
			successCount, actualImported, successCount-int(actualImported))
	}
	
	// 确定响应状态
	responseStatus := "ok"
	warningMsg := ""
	
	if droppedCount > 0 {
		responseStatus = "partial"
		warningMsg = fmt.Sprintf("提交 %d 条，丢弃 %d 条（队列满）", successCount, droppedCount)
	}
	
	if actualImported < int64(successCount) {
		responseStatus = "partial"
		if warningMsg != "" {
			warningMsg += "；"
		}
		warningMsg += fmt.Sprintf("实际导入 %d 条，% d 条可能格式不匹配", 
			actualImported, successCount-int(actualImported))
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":     responseStatus,
		"lines":      len(lines),
		"accepted":   successCount,
		"imported":   actualImported,
		"dropped":    droppedCount,
		"file":       file.Filename,
		"warning":    warningMsg,
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
	// 空配置或 auto 模式自动识别所有格式
	if configFormat == "" || configFormat == "auto" {
		return true
	}

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

	outputPath := filepath.Join("./exports", filename+getExtension(format))
	contentType, err := s.exportManager.Export(entries, format, outputPath, &exporter.ExportOptions{
		TimeFormat: time.RFC3339,
	})
	if err != nil {
		log.Printf("[EXPORT] 导出失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = contentType

	c.FileAttachment(outputPath, filename+getExtension(format))
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

// compareReceiverConfig 比较两个接收器配置是否相同
func compareReceiverConfig(a, b config.ReceiverConfig) bool {
	if a.TCPEnabled != b.TCPEnabled || a.TCPPort != b.TCPPort {
		return false
	}
	if a.UDPEnabled != b.UDPEnabled || a.UDPPort != b.UDPPort {
		return false
	}
	if a.HTTPEnabled != b.HTTPEnabled || a.HTTPPort != b.HTTPPort {
		return false
	}
	if a.HTTPAuthToken != b.HTTPAuthToken {
		return false
	}
	if a.HTTPRateLimit != b.HTTPRateLimit {
		return false
	}
	if a.BufferSize != b.BufferSize {
		return false
	}
	// 比较IP白名单
	if len(a.HTTPAllowedIPs) != len(b.HTTPAllowedIPs) {
		return false
	}
	for i, ip := range a.HTTPAllowedIPs {
		if ip != b.HTTPAllowedIPs[i] {
			return false
		}
	}
	return true
}

// restartReceivers 重启接收器
func (s *Server) restartReceivers(newCfg config.ReceiverConfig) error {
	// 停止当前接收器
	if err := s.receiver.Stop(); err != nil {
		log.Printf("[WARN] 停止接收器时出现错误: %v", err)
		// 继续，尝试启动新的接收器
	}
	
	// 创建新的接收器管理器
	s.receiver = receiver.NewManager(newCfg)
	
	// 启动新的接收器
	err := s.receiver.Start(func(line string) bool {
		if !s.processor.Submit(line) {
			log.Printf("处理器队列已满，丢弃日志: %s", line[:min(50, len(line))])
			return false
		}
		return true
	})
	
	if err != nil {
		return fmt.Errorf("启动接收器失败: %v", err)
	}
	
	return nil
}

// getExtension 获取文件扩展名
func getExtension(format string) string {
	switch format {
	case "excel":
		return ".xlsx"
	case "csv":
		return ".csv"
	case "json":
		return ".json"
	default:
		return ".xlsx"
	}
}

// getStorageInfo 获取存储信息
func (s *Server) getStorageInfo(c *gin.Context) {
	info := gin.H{
		"type": s.config.Get().Storage.Type,
		"db_path": s.config.Get().Storage.DBPath,
	}

	// 获取数据库文件大小
	if s.config.Get().Storage.Type == "sqlite" {
		dbPath := s.config.Get().Storage.DBPath
		if stat, err := os.Stat(dbPath); err == nil {
			info["size_bytes"] = stat.Size()
		} else {
			info["size_bytes"] = 0
		}
	}

	c.JSON(http.StatusOK, info)
}

// compactStorage 压缩数据库（释放未使用空间）
func (s *Server) compactStorage(c *gin.Context) {
	if s.config.Get().Storage.Type != "sqlite" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only SQLite supports compact"})
		return
	}

	// 获取压缩前大小
	dbPath := s.config.Get().Storage.DBPath
	var sizeBefore int64
	if stat, err := os.Stat(dbPath); err == nil {
		sizeBefore = stat.Size()
	}

	// 执行VACUUM命令压缩数据库
	sqliteStorage, ok := s.storage.(*storage.SQLiteStorage)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage type mismatch"})
		return
	}

	if err := sqliteStorage.Vacuum(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Vacuum failed: " + err.Error()})
		return
	}

	// 获取压缩后大小
	var sizeAfter int64
	if stat, err := os.Stat(dbPath); err == nil {
		sizeAfter = stat.Size()
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"size_before_bytes": sizeBefore,
		"size_after_bytes": sizeAfter,
		"freed_bytes": sizeBefore - sizeAfter,
	})
}
