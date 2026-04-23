// server/server.go - Web服务器
package server

import (
	"bufio"
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
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type directBatchStorage interface {
	SaveBatchDirect(entries []*models.LogEntry) error
}

type importProgressState struct {
	ID             string    `json:"id"`
	FileName       string    `json:"file_name"`
	DetectedFormat string    `json:"detected_format,omitempty"`
	Phase          string    `json:"phase"`
	ParsedLines    int64     `json:"parsed_lines"`
	WrittenLines   int64     `json:"written_lines"`
	SkippedLines   int64     `json:"skipped_lines"`
	ScannedLines   int64     `json:"scanned_lines"`
	TargetLines    int64     `json:"target_lines"`
	Percent        float64   `json:"percent"`
	LimitReached   bool      `json:"limit_reached"`
	StartedAt      time.Time `json:"started_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Message        string    `json:"message,omitempty"`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Server Web服务器
type Server struct {
	config           *config.Config
	configPath       string
	router           *gin.Engine
	storage          storage.Storage
	parser           *parser.LogParser
	processor        *processor.Processor
	receiver         *receiver.Manager
	exportManager    *exporter.ExportManager
	runtimeMu        sync.Mutex
	receiverRunning  bool
	benchmarkMu      sync.Mutex
	benchmarkRunning bool
	lastBenchmark    map[string]interface{}
	importProgressMu sync.RWMutex
	importProgress   map[string]*importProgressState
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
		config:          cfg,
		configPath:      configPath,
		router:          router,
		storage:         store,
		parser:          parser.NewLogParser(cfg.GetParserConfig()),
		processor:       proc,
		receiver:        recv,
		exportManager:   exporter.NewExportManager(),
		receiverRunning: true,
		importProgress:  make(map[string]*importProgressState),
	}

	s.setupRoutes()
	return s
}

// 算法2-7：路由注册与分组
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
	s.router.GET("/metrics", s.getMetrics)

	// API路由组
	api := s.router.Group("/api")
	{
		// 配置管理
		api.GET("/config", s.getConfig)
		api.POST("/config", s.updateConfig)

		// 日志查询
		api.GET("/logs", s.queryLogs)
		api.POST("/logs/import", s.importLogsFast)
		api.GET("/logs/import/progress", s.getImportProgress)
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
		api.POST("/benchmark/run", s.runBenchmark)
		api.GET("/benchmark/report", s.getBenchmarkReport)
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
		"server":    cfg.Server,
		"parser":    cfg.Parser,
		"processor": cfg.Processor,
		"alert":     cfg.Alert,
		"display":   cfg.Display,
		"import":    cfg.Import,
		"storage":   cfg.Storage,
		"receiver": gin.H{
			"tcp_enabled":          cfg.Receiver.TCPEnabled,
			"tcp_port":             cfg.Receiver.TCPPort,
			"udp_enabled":          cfg.Receiver.UDPEnabled,
			"udp_port":             cfg.Receiver.UDPPort,
			"http_enabled":         cfg.Receiver.HTTPEnabled,
			"http_port":            cfg.Receiver.HTTPPort,
			"http_auth_token":      cfg.Receiver.HTTPAuthToken, // 返回实际值（为空则不启用）
			"http_allowed_ips":     cfg.Receiver.HTTPAllowedIPs,
			"http_rate_limit":      cfg.Receiver.HTTPRateLimit,
			"http_max_body_size":   cfg.Receiver.HTTPMaxBodySize,
			"buffer_size":          cfg.Receiver.BufferSize,
			"file_watcher_enabled": cfg.Receiver.FileWatcherEnabled,
			"watch_paths":          cfg.Receiver.WatchPaths,
			"max_connections":      cfg.Receiver.MaxConnections,
		},
	})
}

// 算法2-8：配置热更新流程
// updateConfig 更新配置
func (s *Server) updateConfig(c *gin.Context) {
	var jsonConfig map[string]interface{}
	if err := c.ShouldBindJSON(&jsonConfig); err != nil {
		log.Printf("[ERROR] failed to parse config json: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config format: " + err.Error()})
		return
	}

	oldConfig := s.config.Get()
	mergedConfig := s.config.Get()

	if err := mergeConfigSection(jsonConfig, "server", &mergedConfig.Server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mergeConfigSection(jsonConfig, "parser", &mergedConfig.Parser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mergeConfigSection(jsonConfig, "processor", &mergedConfig.Processor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mergeConfigSection(jsonConfig, "alert", &mergedConfig.Alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mergeConfigSection(jsonConfig, "display", &mergedConfig.Display); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mergeConfigSection(jsonConfig, "import", &mergedConfig.Import); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mergeConfigSection(jsonConfig, "storage", &mergedConfig.Storage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mergeConfigSection(jsonConfig, "receiver", &mergedConfig.Receiver); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.config.Update(&mergedConfig); err != nil {
		log.Printf("[ERROR] failed to update config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config: " + err.Error()})
		return
	}

	if err := s.config.SaveToFile(s.configPath); err != nil {
		log.Printf("[ERROR] failed to save config file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "config updated in memory but failed to save file: " + err.Error()})
		return
	}

	oldRecvCfg := oldConfig.Receiver
	newRecvCfg := mergedConfig.Receiver
	receiverChanged := !compareReceiverConfig(oldRecvCfg, newRecvCfg)
	storageChanged := !compareStorageConfig(oldConfig.Storage, mergedConfig.Storage)

	s.parser.SetConfig(mergedConfig.Parser)
	s.processor.UpdateConfig(mergedConfig.Processor)
	s.processor.SetParser(s.parser)

	if storageChanged {
		if err := s.applyStorageConfig(mergedConfig.Storage); err != nil {
			log.Printf("[ERROR] failed to apply storage runtime config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "partial",
				"message": "config saved but failed to apply storage config at runtime: " + err.Error(),
			})
			return
		}
	}

	if receiverChanged {
		log.Printf("[INFO] receiver config changed, restarting receiver")
		if err := s.restartReceivers(newRecvCfg); err != nil {
			log.Printf("[ERROR] failed to restart receivers: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "partial",
				"message": "config saved but failed to restart receiver: " + err.Error(),
			})
			return
		}
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
	if ranges := c.QueryArray("status_code_ranges"); len(ranges) > 0 {
		filter.StatusCodeRanges = ranges
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
		"data":   entries,
		"total":  count,
		"limit":  limit,
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
			"status":          "warning",
			"lines":           len(lines),
			"accepted":        0,
			"file":            file.Filename,
			"warning":         fmt.Sprintf("文件格式为 [%s]，但当前配置为 [%s]。请前往「配置」页面修改解析格式后再导入。", detectedFormat, currentFormat),
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

	// 确定响应状态
	responseStatus := "ok"
	warningMsg := ""

	if droppedCount > 0 {
		responseStatus = "partial"
		warningMsg = fmt.Sprintf("提交 %d 条，丢弃 %d 条（队列满）", successCount, droppedCount)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   responseStatus,
		"lines":    len(lines),
		"accepted": successCount,
		"imported": actualImported,
		"dropped":  droppedCount,
		"file":     file.Filename,
		"warning":  warningMsg,
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
// 直接复用 parser 包的 DetectFormat，确保导入检测与解析层逻辑一致
func detectLogFormat(line string) string {
	return parser.DetectFormat(line)
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
	s.processor.ClearPendingData()

	if err := s.storage.Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "All logs and pending backlog cleared",
	})
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

	alerts := buildAlertSummaries(stats, s.config.Get().Alert)
	log.Printf("[API] Statistics: total=%d, errors=%d, avg_response=%.2fms",
		stats.TotalCount, stats.ErrorCount, stats.AvgResponseTime)
	c.JSON(http.StatusOK, gin.H{
		"total_count":       stats.TotalCount,
		"error_count":       stats.ErrorCount,
		"avg_response_time": stats.AvgResponseTime,
		"status_code_dist":  stats.StatusCodeDist,
		"method_dist":       stats.MethodDist,
		"top_paths":         stats.TopPaths,
		"time_series":       stats.TimeSeries,
		"alerts":            alerts,
	})
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
	storageStats, _ := s.storage.Statistics(models.FilterCondition{})
	alerts := buildAlertSummaries(storageStats, cfg.Alert)

	// 只返回基本配置信息，过滤敏感字段
	c.JSON(http.StatusOK, gin.H{
		"config": gin.H{
			"server":    cfg.Server,
			"parser":    cfg.Parser,
			"processor": cfg.Processor,
			"alert":     cfg.Alert,
			"display":   cfg.Display,
			"import":    cfg.Import,
			"receiver": gin.H{
				"tcp_enabled":          cfg.Receiver.TCPEnabled,
				"tcp_port":             cfg.Receiver.TCPPort,
				"udp_enabled":          cfg.Receiver.UDPEnabled,
				"udp_port":             cfg.Receiver.UDPPort,
				"http_enabled":         cfg.Receiver.HTTPEnabled,
				"http_port":            cfg.Receiver.HTTPPort,
				"http_rate_limit":      cfg.Receiver.HTTPRateLimit,
				"http_max_body_size":   cfg.Receiver.HTTPMaxBodySize,
					// 高级接收参数与文件监控已隐藏（答辩演示不需要）
			},
			"storage": cfg.Storage,
		},
		"processor":  stats,
		"alerts":     alerts,
		"statistics": storageStats,
		"timestamp":  time.Now(),
	})
}

// startReceiver 启动接收器
func (s *Server) startReceiver(c *gin.Context) {
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()

	if s.receiverRunning {
		c.JSON(http.StatusOK, gin.H{"status": "already running"})
		return
	}

	if err := s.startReceiverLocked(s.config.Get().Receiver); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start receiver: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// stopReceiver 停止接收器
func (s *Server) stopReceiver(c *gin.Context) {
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()

	if !s.receiverRunning {
		c.JSON(http.StatusOK, gin.H{"status": "already stopped"})
		return
	}

	if err := s.stopReceiverLocked(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop receiver: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func buildAlertSummaries(stats *models.Statistics, cfg config.AlertConfig) []gin.H {
	if stats == nil {
		return []gin.H{}
	}

	alerts := make([]gin.H, 0, 2)
	if cfg.SlowThreshold > 0 && stats.AvgResponseTime >= float64(cfg.SlowThreshold) {
		alerts = append(alerts, gin.H{
			"level":   "warning",
			"title":   "平均响应时间过高",
			"message": fmt.Sprintf("当前平均响应时间 %.0fms，已达到告警阈值 %dms。", stats.AvgResponseTime, cfg.SlowThreshold),
		})
	}

	if cfg.ErrorRateThreshold > 0 && stats.TotalCount > 0 {
		errorRate := float64(stats.ErrorCount) / float64(stats.TotalCount) * 100
		if errorRate >= float64(cfg.ErrorRateThreshold) {
			alerts = append(alerts, gin.H{
				"level":   "error",
				"title":   "错误率过高",
				"message": fmt.Sprintf("当前错误率 %.1f%%，已达到告警阈值 %d%%。", errorRate, cfg.ErrorRateThreshold),
			})
		}
	}

	return alerts
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
	if a.HTTPMaxBodySize != b.HTTPMaxBodySize {
		return false
	}
	// 高级接收参数与文件监控不参与差异化重启判断（答辩演示不需要）
	if !compareStringSlices(a.HTTPAllowedIPs, b.HTTPAllowedIPs) {
		return false
	}
	return true
}

// restartReceivers 重启接收器
func (s *Server) restartReceivers(newCfg config.ReceiverConfig) error {
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()

	wasRunning := s.receiverRunning
	if wasRunning {
		if err := s.stopReceiverLocked(); err != nil {
			log.Printf("[WARN] failed to stop receiver before restart: %v", err)
		}
	}

	if !wasRunning {
		s.receiver = receiver.NewManager(newCfg)
		return nil
	}

	if err := s.startReceiverLocked(newCfg); err != nil {
		return fmt.Errorf("failed to start receiver: %w", err)
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
		"type":    s.config.Get().Storage.Type,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "only SQLite supports compact"})
		return
	}

	dbPath := s.config.Get().Storage.DBPath
	var sizeBefore int64
	if stat, err := os.Stat(dbPath); err == nil {
		sizeBefore = stat.Size()
	}

	vacuumer, ok := s.storage.(interface {
		Vacuum() error
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage does not support compact"})
		return
	}

	if err := vacuumer.Vacuum(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vacuum failed: " + err.Error()})
		return
	}

	var sizeAfter int64
	if stat, err := os.Stat(dbPath); err == nil {
		sizeAfter = stat.Size()
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            "ok",
		"size_before_bytes": sizeBefore,
		"size_after_bytes":  sizeAfter,
		"freed_bytes":       sizeBefore - sizeAfter,
	})
}

// getMetrics 导出 Prometheus 文本指标
func (s *Server) getMetrics(c *gin.Context) {
	stats := s.processor.GetStats()
	cfg := s.config.Get()

	var b strings.Builder
	writePromMetric(&b, "log_processor_input_queue_size", float64(asInt64(stats["input_queue_size"])))
	writePromMetric(&b, "log_processor_output_queue_size", float64(asInt64(stats["output_queue_size"])))
	writePromMetric(&b, "log_processor_worker_count", float64(asInt64(stats["worker_count"])))
	writePromMetric(&b, "log_processor_batch_size", float64(asInt64(stats["batch_size"])))
	writePromMetric(&b, "log_processor_received_total", float64(asInt64(stats["received_count"])))
	writePromMetric(&b, "log_processor_processed_total", float64(asInt64(stats["processed_count"])))
	writePromMetric(&b, "log_processor_dropped_total", float64(asInt64(stats["dropped_count"])))
	writePromMetric(&b, "log_processor_parse_error_total", float64(asInt64(stats["parse_error_count"])))
	writePromMetric(&b, "log_processor_spill_total", float64(asInt64(stats["spill_count"])))
	writePromMetric(&b, "log_processor_overflow_enabled", boolToFloat64(cfg.Processor.OverflowEnabled))
	writePromMetric(&b, "log_processor_overflow_pending", float64(asInt64(stats["overflow_queue_pending"])))
	writePromMetric(&b, "log_processor_overflow_file_bytes", float64(asInt64(stats["overflow_file_size_bytes"])))
	writePromMetric(&b, "log_processor_overflow_enqueued_total", float64(asInt64(stats["overflow_enqueued_count"])))
	writePromMetric(&b, "log_processor_overflow_recovered_total", float64(asInt64(stats["overflow_recovered_count"])))
	writePromMetric(&b, "log_processor_overflow_dropped_total", float64(asInt64(stats["overflow_dropped_count"])))
	writePromMetric(&b, "log_processor_overflow_write_error_total", float64(asInt64(stats["overflow_write_error_count"])))

	s.runtimeMu.Lock()
	receiverRunning := s.receiverRunning
	s.runtimeMu.Unlock()
	writePromMetric(&b, "log_receiver_running", boolToFloat64(receiverRunning))

	if asyncStore, ok := s.storage.(interface {
		GetStats() storage.AsyncStats
	}); ok {
		asyncStats := asyncStore.GetStats()
		writePromMetric(&b, "log_storage_buffered", float64(asyncStats.BufferedCount))
		writePromMetric(&b, "log_storage_flushed_total", float64(asyncStats.FlushedCount))
		writePromMetric(&b, "log_storage_dropped_total", float64(asyncStats.DroppedCount))
		writePromMetric(&b, "log_storage_avg_flush_latency_ms", float64(asyncStats.AvgFlushLatency))
	}

	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(b.String()))
}

type benchmarkRunRequest struct {
	DurationSeconds int `json:"duration_seconds"`
	Workers         int `json:"workers"`
	TargetQPS       int `json:"target_qps"`
}

// runBenchmark 执行一键压测并返回报告
// 算法2-9：压测执行流程
func (s *Server) runBenchmark(c *gin.Context) {
	req := benchmarkRunRequest{
		DurationSeconds: 10,
		Workers:         20,
		TargetQPS:       5000,
	}

	if c.Request.ContentLength > 0 {
		var payload benchmarkRunRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid benchmark request: " + err.Error()})
			return
		}
		if payload.DurationSeconds > 0 {
			req.DurationSeconds = payload.DurationSeconds
		}
		if payload.Workers > 0 {
			req.Workers = payload.Workers
		}
		if payload.TargetQPS >= 0 {
			req.TargetQPS = payload.TargetQPS
		}
	}

	if req.DurationSeconds < 3 {
		req.DurationSeconds = 3
	}
	if req.DurationSeconds > 300 {
		req.DurationSeconds = 300
	}
	if req.Workers < 1 {
		req.Workers = 1
	}
	if req.Workers > 200 {
		req.Workers = 200
	}
	if req.TargetQPS < 0 {
		req.TargetQPS = 0
	}
	if req.TargetQPS > 1000000 {
		req.TargetQPS = 1000000
	}

	s.benchmarkMu.Lock()
	if s.benchmarkRunning {
		s.benchmarkMu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "benchmark is already running"})
		return
	}
	s.benchmarkRunning = true
	s.benchmarkMu.Unlock()

	defer func() {
		s.benchmarkMu.Lock()
		s.benchmarkRunning = false
		s.benchmarkMu.Unlock()
	}()

	beforeTotal, _ := s.storage.Count(models.FilterCondition{})
	beforeStats := s.processor.GetStats()
	start := time.Now()
	deadline := start.Add(time.Duration(req.DurationSeconds) * time.Second)

	var submitted int64
	var rejected int64
	var sequence int64
	var wg sync.WaitGroup

	for workerID := 0; workerID < req.Workers; workerID++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()

			interval := time.Duration(0)
			if req.TargetQPS > 0 {
				perWorkerQPS := float64(req.TargetQPS) / float64(req.Workers)
				if perWorkerQPS < 1 {
					perWorkerQPS = 1
				}
				interval = time.Duration(float64(time.Second) / perWorkerQPS)
			}
			nextTick := time.Now()

			for time.Now().Before(deadline) {
				if interval > 0 {
					now := time.Now()
					if nextTick.After(now) {
						time.Sleep(nextTick.Sub(now))
					}
					nextTick = nextTick.Add(interval)
				}

				id := atomic.AddInt64(&sequence, 1)
				line := buildBenchmarkLogLine(id, wid)
				if s.processor.Submit(line) {
					atomic.AddInt64(&submitted, 1)
				} else {
					atomic.AddInt64(&rejected, 1)
				}
			}
		}(workerID)
	}

	wg.Wait()
	sendFinished := time.Now()
	drainTimeout := time.Duration(req.DurationSeconds) * time.Second
	if drainTimeout < 5*time.Second {
		drainTimeout = 5 * time.Second
	}
	if drainTimeout > 30*time.Second {
		drainTimeout = 30 * time.Second
	}
	drainElapsed, drainCompleted, backlogSnapshot := s.waitForBenchmarkDrain(drainTimeout)

	afterTotal, _ := s.storage.Count(models.FilterCondition{})
	afterStats := s.processor.GetStats()

	storedAdded := afterTotal - beforeTotal
	if storedAdded < 0 {
		storedAdded = 0
	}

	sendDuration := float64(req.DurationSeconds)
	if sendDuration <= 0 {
		sendDuration = 1
	}
	sendElapsed := sendFinished.Sub(start).Seconds()
	if sendElapsed <= 0 {
		sendElapsed = sendDuration
	}
	totalElapsed := time.Since(start).Seconds()

	totalSubmitted := atomic.LoadInt64(&submitted)
	totalRejected := atomic.LoadInt64(&rejected)
	acceptRate := 0.0
	if totalSubmitted+totalRejected > 0 {
		acceptRate = float64(totalSubmitted) / float64(totalSubmitted+totalRejected) * 100
	}

	report := map[string]interface{}{
		"started_at":            start.Format(time.RFC3339),
		"send_finished_at":      sendFinished.Format(time.RFC3339),
		"finished_at":           time.Now().Format(time.RFC3339),
		"duration_seconds":      req.DurationSeconds,
		"send_elapsed_seconds":  sendElapsed,
		"drain_elapsed_seconds": drainElapsed,
		"total_elapsed_seconds": totalElapsed,
		"drain_completed":       drainCompleted,
		"drain_timeout_seconds": drainTimeout.Seconds(),
		"workers":               req.Workers,
		"target_qps":            req.TargetQPS,
		"submitted":             totalSubmitted,
		"rejected":              totalRejected,
		"accept_rate":           acceptRate,
		"submit_qps":            float64(totalSubmitted) / sendDuration,
		"stored_added":          storedAdded,
		"stored_qps":            float64(storedAdded) / sendDuration,
		"queue_backlog":         backlogSnapshot,
		"processor_delta": map[string]interface{}{
			"received_delta":           metricDelta(afterStats, beforeStats, "received_count"),
			"processed_delta":          metricDelta(afterStats, beforeStats, "processed_count"),
			"dropped_delta":            metricDelta(afterStats, beforeStats, "dropped_count"),
			"parse_error_delta":        metricDelta(afterStats, beforeStats, "parse_error_count"),
			"spill_delta":              metricDelta(afterStats, beforeStats, "spill_count"),
			"overflow_recovered_delta": metricDelta(afterStats, beforeStats, "overflow_recovered_count"),
			"overflow_pending":         asInt64(afterStats["overflow_queue_pending"]),
			"overflow_dropped_delta":   metricDelta(afterStats, beforeStats, "overflow_dropped_count"),
			"overflow_write_err_delta": metricDelta(afterStats, beforeStats, "overflow_write_error_count"),
		},
	}

	s.benchmarkMu.Lock()
	s.lastBenchmark = report
	s.benchmarkMu.Unlock()

	c.JSON(http.StatusOK, report)
}

// getBenchmarkReport 获取最近一次压测报告
func (s *Server) getBenchmarkReport(c *gin.Context) {
	s.benchmarkMu.Lock()
	report := s.lastBenchmark
	running := s.benchmarkRunning
	s.benchmarkMu.Unlock()

	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"running": running,
			"error":   "no benchmark report available",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"running": running,
		"report":  report,
	})
}

func buildBenchmarkLogLine(sequence int64, workerID int) string {
	timestamp := time.Now().Format("02/Jan/2006:15:04:05 -0700")

	paths := []string{
		"/",
		"/index.html",
		"/favicon.ico",
		"/static/app.js",
		"/static/style.css",
		"/api/login",
		"/api/logout",
		"/api/user/profile",
		"/api/user/list",
		"/api/orders",
		"/api/orders/create",
		"/api/orders/detail",
		"/api/products",
		"/api/reports/daily",
		"/api/reports/error",
		"/admin/dashboard",
		"/admin/system/status",
		"/search?q=log",
		"/docs/help",
	}
	methods := []string{"GET", "GET", "GET", "GET", "POST", "PUT", "DELETE"}
	referers := []string{
		"-",
		"https://portal.example.com/",
		"https://portal.example.com/dashboard",
		"https://portal.example.com/orders",
		"https://portal.example.com/reports",
		"https://m.example.com/home",
	}
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/132.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/17.0 Safari/605.1.15",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 Chrome/131.0 Mobile Safari/537.36",
		"curl/8.5.0",
		"PostmanRuntime/7.43.0",
	}
	statusBuckets := []int{200, 200, 200, 200, 200, 200, 201, 204, 301, 302, 400, 401, 403, 404, 429, 500, 502, 503}

	idx := int((sequence + int64(workerID)*7) % int64(len(paths)))
	path := paths[idx]
	method := methods[int((sequence+int64(workerID)*3)%int64(len(methods)))]
	referer := referers[int((sequence+int64(workerID))%int64(len(referers)))]
	userAgent := userAgents[int((sequence+int64(workerID)*5)%int64(len(userAgents)))]
	statusCode := statusBuckets[int((sequence*3+int64(workerID))%int64(len(statusBuckets)))]

	if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".ico") {
		method = "GET"
	}
	if strings.Contains(path, "/create") {
		method = "POST"
	}
	if strings.Contains(path, "/logout") {
		method = "POST"
	}
	if strings.Contains(path, "/profile") && statusCode >= 500 {
		statusCode = 200
	}

	ipA := 10 + int((sequence+int64(workerID))%10)
	ipB := 20 + workerID%30
	ipC := 1 + int(sequence%200)
	clientIP := fmt.Sprintf("192.168.%d.%d", ipA, (ipB+ipC)%254+1)

	responseSize := 256 + (sequence*73+int64(workerID)*131)%16384
	responseTime := 20 + (sequence*17+int64(workerID)*29)%2400

	if statusCode >= 500 {
		responseTime += 1500
		responseSize = 128 + sequence%2048
	}
	if statusCode == 404 {
		responseSize = 512 + sequence%1024
	}
	if statusCode == 301 || statusCode == 302 {
		responseTime = 10 + sequence%200
		responseSize = 64 + sequence%512
	}
	if referer == "-" && strings.HasPrefix(path, "/api/") {
		referer = "https://portal.example.com/api-console"
	}

	return fmt.Sprintf(
		`%s - - [%s] "%s %s HTTP/1.1" %d %d "%s" "%s" "%d"`,
		clientIP,
		timestamp,
		method,
		path,
		statusCode,
		responseSize,
		referer,
		userAgent,
		responseTime,
	)
}

func writePromMetric(builder *strings.Builder, name string, value float64) {
	builder.WriteString(name)
	builder.WriteByte(' ')
	builder.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
	builder.WriteByte('\n')
}

func metricDelta(after map[string]interface{}, before map[string]interface{}, key string) int64 {
	return asInt64(after[key]) - asInt64(before[key])
}

func (s *Server) getBenchmarkBacklogSnapshot() map[string]int64 {
	stats := s.processor.GetStats()
	snapshot := map[string]int64{
		"input_queue":      asInt64(stats["input_queue_size"]),
		"output_queue":     asInt64(stats["output_queue_size"]),
		"overflow_pending": asInt64(stats["overflow_queue_pending"]),
		"async_buffered":   0,
	}

	if asyncStore, ok := s.storage.(interface{ GetStats() storage.AsyncStats }); ok {
		snapshot["async_buffered"] = asyncStore.GetStats().BufferedCount
	}

	return snapshot
}

func (s *Server) waitForBenchmarkDrain(timeout time.Duration) (float64, bool, map[string]int64) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	start := time.Now()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	lastSnapshot := s.getBenchmarkBacklogSnapshot()
	for {
		if lastSnapshot["input_queue"] == 0 &&
			lastSnapshot["output_queue"] == 0 &&
			lastSnapshot["overflow_pending"] == 0 &&
			lastSnapshot["async_buffered"] == 0 {
			return time.Since(start).Seconds(), true, lastSnapshot
		}

		if time.Since(start) >= timeout {
			return time.Since(start).Seconds(), false, lastSnapshot
		}

		<-ticker.C
		lastSnapshot = s.getBenchmarkBacklogSnapshot()
	}
}

func asInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		if v > uint64(^uint64(0)>>1) {
			return int64(^uint64(0) >> 1)
		}
		return int64(v)
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func boolToFloat64(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

// importLogsFast 走离线导入专用链路，绕过实时处理队列和异步存储缓冲。
// 算法2-6：文件导入流程
func (s *Server) importLogsFast(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	importID := c.PostForm("import_id")
	if importID == "" {
		importID = fmt.Sprintf("import_%d", time.Now().UnixNano())
	}
	s.createImportProgress(importID, file.Filename)
	defer s.scheduleImportProgressCleanup(importID, 2*time.Minute)

	tempPath := filepath.Join("./temp", file.Filename)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		s.finishImportProgress(importID, "error", "保存上传文件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.Remove(tempPath)

	currentFormat := s.config.Get().Parser.Format
	maxImportLines := s.config.Get().Import.MaxLines
	if maxImportLines <= 0 || maxImportLines > 100000 {
		maxImportLines = 100000
	}
	log.Printf("[IMPORT] 开始快速导入文件 %s, 当前解析格式: %s", file.Filename, currentFormat)

	statsBefore, _ := s.storage.Statistics(models.FilterCondition{})
	countBefore := int64(0)
	if statsBefore != nil {
		countBefore = statsBefore.TotalCount
	}

	sourceFile, err := os.Open(tempPath)
	if err != nil {
		s.finishImportProgress(importID, "error", "打开临时文件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	scanner := bufio.NewScanner(sourceFile)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	targetLines := int64(0)
	limitReached := false
	detectedFormat := "unknown"
	formatChecked := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !formatChecked {
			detectedFormat = detectLogFormat(trimmed)
			if !isFormatCompatible(detectedFormat, currentFormat) {
				sourceFile.Close()
				s.finishImportProgress(importID, "warning", "文件格式与当前配置不匹配")
				c.JSON(http.StatusOK, gin.H{
					"status":          "warning",
					"lines":           1,
					"accepted":        0,
					"file":            file.Filename,
					"warning":         fmt.Sprintf("文件格式为 [%s]，但当前配置为 [%s]。请前往「配置」页面修改解析格式后再导入。", detectedFormat, currentFormat),
					"detected_format": detectedFormat,
					"current_format":  currentFormat,
				})
				return
			}
			formatChecked = true
		}

		targetLines++
		if targetLines >= int64(maxImportLines) {
			limitReached = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		sourceFile.Close()
		s.finishImportProgress(importID, "error", "预扫描导入文件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sourceFile.Close()

	if !formatChecked {
		s.finishImportProgress(importID, "completed", "文件为空")
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"lines":    0,
			"accepted": 0,
			"file":     file.Filename,
			"warning":  "文件为空",
		})
		return
	}

	s.updateImportProgress(importID, 0, 0, 0, 0, targetLines, limitReached, "processing", "")
	s.setImportProgressDetectedFormat(importID, detectedFormat)
	log.Printf("[IMPORT] 预扫描完成: target=%d, limit_reached=%v, format=%s", targetLines, limitReached, detectedFormat)

	sourceFile, err = os.Open(tempPath)
	if err != nil {
		s.finishImportProgress(importID, "error", "打开临时文件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer sourceFile.Close()

	scanner = bufio.NewScanner(sourceFile)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	const importBatchSize = 5000
	batch := make([]*models.LogEntry, 0, importBatchSize)
	totalLines := 0
	acceptedCount := 0
	droppedCount := 0
	writtenCount := int64(0)
	failureReasons := make(map[string]int)
	failureSamples := make([]string, 0, 3)

	for scanner.Scan() {
		totalLines++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if acceptedCount+droppedCount >= maxImportLines {
			limitReached = true
			break
		}

		entry, parseErr := s.parser.ParseWithFormat(line, detectedFormat)
		if parseErr != nil {
			droppedCount++
			reason := parseErr.Error()
			failureReasons[reason]++
			if len(failureSamples) < 3 {
				// 记录失败样本（截断过长行）
				sample := trimmed
				if len(sample) > 80 {
					sample = sample[:80] + "..."
				}
				failureSamples = append(failureSamples, fmt.Sprintf("[%s] %s", reason, sample))
			}
			s.updateImportProgress(importID, int64(acceptedCount), writtenCount, int64(droppedCount), int64(totalLines), targetLines, limitReached, "processing", "")
			continue
		}

		batch = append(batch, entry)
		acceptedCount++
		if acceptedCount%500 == 0 {
			s.updateImportProgress(importID, int64(acceptedCount), writtenCount, int64(droppedCount), int64(totalLines), targetLines, limitReached, "processing", "")
		}

		if len(batch) >= importBatchSize {
			if err := s.saveImportBatch(batch); err != nil {
				s.finishImportProgress(importID, "error", "批量写入失败")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save imported logs: " + err.Error()})
				return
			}
			writtenCount += int64(len(batch))
			s.updateImportProgress(importID, int64(acceptedCount), writtenCount, int64(droppedCount), int64(totalLines), targetLines, limitReached, "processing", "")
			batch = batch[:0]
		}
	}

	if err := scanner.Err(); err != nil {
		s.finishImportProgress(importID, "error", "读取导入文件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(batch) > 0 {
		if err := s.saveImportBatch(batch); err != nil {
			s.finishImportProgress(importID, "error", "批量写入失败")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save imported logs: " + err.Error()})
			return
		}
		writtenCount += int64(len(batch))
		s.updateImportProgress(importID, int64(acceptedCount), writtenCount, int64(droppedCount), int64(totalLines), targetLines, limitReached, "processing", "")
	}

	statsAfter, _ := s.storage.Statistics(models.FilterCondition{})
	countAfter := int64(0)
	if statsAfter != nil {
		countAfter = statsAfter.TotalCount
	}
	// 使用本请求实际写入的 writtenCount，避免并发导入时 countAfter-countBefore 包含其他请求的写入量
	actualImported := writtenCount
	if actualImported <= 0 && countAfter > countBefore {
		actualImported = countAfter - countBefore
	}

	// 生成失败原因摘要
	failureReasonSummary := ""
	if len(failureReasons) > 0 {
		// 取出现次数最多的前 3 个原因
		type reasonCount struct {
			reason string
			count  int
		}
		reasons := make([]reasonCount, 0, len(failureReasons))
		for r, c := range failureReasons {
			reasons = append(reasons, reasonCount{r, c})
		}
		// 简单冒泡排序取前3
		for i := 0; i < len(reasons)-1; i++ {
			for j := i + 1; j < len(reasons); j++ {
				if reasons[j].count > reasons[i].count {
					reasons[i], reasons[j] = reasons[j], reasons[i]
				}
			}
		}
		parts := make([]string, 0, 3)
		for i := 0; i < len(reasons) && i < 3; i++ {
			parts = append(parts, fmt.Sprintf("%s（%d 行）", reasons[i].reason, reasons[i].count))
		}
		failureReasonSummary = strings.Join(parts, "；")
	}

	responseStatus := "ok"
	warningMsg := ""
	missedCount := int64(droppedCount)
	if actualImported < int64(acceptedCount) {
		missedCount = int64(acceptedCount) - actualImported
	}
	if missedCount > 0 {
		responseStatus = "partial"
		warningMsg = fmt.Sprintf("解析成功 %d 条，未导入 %d 条", acceptedCount, missedCount)
	}

	// 解析质量检查：如果解析失败率过高，给出明确提示
	if targetLines > 0 {
		failureRate := float64(droppedCount) / float64(targetLines)
		if actualImported == 0 && droppedCount > 0 {
			// 全部解析失败，大概率不是日志文件
			responseStatus = "error"
			warningMsg = fmt.Sprintf("未能解析任何有效日志（共 %d 行），该文件似乎不是标准的日志格式。", targetLines)
		} else if failureRate > 0.8 {
			// 失败率超过 80%，给出警告
			responseStatus = "warning"
			warningMsg = fmt.Sprintf("解析失败率过高（%.0f%%），文件内容可能不符合当前解析格式。", failureRate*100)
		}
	}

	if limitReached {
		limitWarning := fmt.Sprintf("单次最多导入 %d 条，已自动截取前 %d 条有效日志", maxImportLines, maxImportLines)
		if warningMsg == "" {
			warningMsg = limitWarning
		} else if responseStatus != "error" {
			warningMsg = warningMsg + "；" + limitWarning
		}
		if responseStatus == "ok" {
			responseStatus = "partial"
		}
	}

	finalPhase := "completed"
	if responseStatus == "partial" || responseStatus == "warning" {
		finalPhase = "partial"
	} else if responseStatus == "error" {
		finalPhase = "error"
	}
	s.finishImportProgress(importID, finalPhase, warningMsg)
	s.updateImportProgress(importID, int64(acceptedCount), actualImported, int64(droppedCount), int64(totalLines), targetLines, limitReached, finalPhase, warningMsg)

	log.Printf("[IMPORT] 文件 %s 快速导入完成: lines=%d accepted=%d imported=%d dropped=%d format=%s",
		file.Filename, totalLines, acceptedCount, actualImported, droppedCount, detectedFormat)

	c.JSON(http.StatusOK, gin.H{
		"import_id":        importID,
		"status":           responseStatus,
		"lines":            totalLines,
		"accepted":         acceptedCount,
		"imported":         actualImported,
		"dropped":          droppedCount,
		"max_lines":        maxImportLines,
		"limit_reached":    limitReached,
		"file":             file.Filename,
		"detected_format":  detectedFormat,
		"warning":          warningMsg,
		"failure_reasons":  failureReasonSummary,
		"failure_samples":  failureSamples,
	})
}

func (s *Server) saveImportBatch(entries []*models.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	if directStore, ok := s.storage.(directBatchStorage); ok {
		return directStore.SaveBatchDirect(entries)
	}
	return s.storage.SaveBatch(entries)
}

func (s *Server) createImportProgress(id, fileName string) {
	s.importProgressMu.Lock()
	defer s.importProgressMu.Unlock()

	now := time.Now()
	s.importProgress[id] = &importProgressState{
		ID:        id,
		FileName:  fileName,
		Phase:     "preparing",
		StartedAt: now,
		UpdatedAt: now,
		Message:   "正在准备导入任务",
	}
}

func (s *Server) updateImportProgress(id string, parsed, written, skipped, scanned, target int64, limitReached bool, phase, message string) {
	s.importProgressMu.Lock()
	defer s.importProgressMu.Unlock()

	progress, ok := s.importProgress[id]
	if !ok {
		return
	}

	progress.ParsedLines = parsed
	progress.WrittenLines = written
	progress.SkippedLines = skipped
	progress.ScannedLines = scanned
	progress.TargetLines = target
	progress.LimitReached = limitReached
	handledLines := parsed + skipped
	if target > 0 {
		progress.Percent = minFloat64(100, (float64(handledLines)/float64(target))*100)
	} else if phase == "completed" || phase == "partial" {
		progress.Percent = 100
	} else {
		progress.Percent = 0
	}
	if phase != "" {
		progress.Phase = phase
	}
	if message != "" {
		progress.Message = message
	}
	progress.UpdatedAt = time.Now()
}

func (s *Server) setImportProgressDetectedFormat(id, detectedFormat string) {
	s.importProgressMu.Lock()
	defer s.importProgressMu.Unlock()

	progress, ok := s.importProgress[id]
	if !ok {
		return
	}

	progress.DetectedFormat = detectedFormat
	progress.UpdatedAt = time.Now()
}

func (s *Server) finishImportProgress(id, phase, message string) {
	s.importProgressMu.Lock()
	defer s.importProgressMu.Unlock()

	progress, ok := s.importProgress[id]
	if !ok {
		return
	}

	if phase != "" {
		progress.Phase = phase
	}
	progress.Message = message
	if progress.TargetLines > 0 && (progress.Phase == "completed" || progress.Phase == "partial") {
		progress.Percent = 100
	}
	progress.UpdatedAt = time.Now()
}

func (s *Server) scheduleImportProgressCleanup(id string, delay time.Duration) {
	time.AfterFunc(delay, func() {
		s.importProgressMu.Lock()
		delete(s.importProgress, id)
		s.importProgressMu.Unlock()
	})
}

func (s *Server) getImportProgress(c *gin.Context) {
	importID := c.Query("id")
	if importID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing import id"})
		return
	}

	s.importProgressMu.RLock()
	progress, ok := s.importProgress[importID]
	if !ok {
		s.importProgressMu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "import progress not found"})
		return
	}

	copyProgress := *progress
	s.importProgressMu.RUnlock()

	c.JSON(http.StatusOK, copyProgress)
}

func mergeConfigSection(payload map[string]interface{}, key string, target interface{}) error {
	raw, ok := payload[key]
	if !ok {
		return nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to encode %s config: %w", key, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("invalid %s config: %w", key, err)
	}

	return nil
}

func compareStorageConfig(a, b config.StorageConfig) bool {
	return a.Type == b.Type &&
		a.DBPath == b.DBPath &&
		a.MaxMemoryItems == b.MaxMemoryItems &&
		a.RetentionHours == b.RetentionHours
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *Server) applyStorageConfig(cfg config.StorageConfig) error {
	updater, ok := s.storage.(interface {
		UpdateConfig(config.StorageConfig) error
	})
	if !ok {
		return fmt.Errorf("storage backend does not support runtime update")
	}
	return updater.UpdateConfig(cfg)
}

func (s *Server) startReceiverLocked(cfg config.ReceiverConfig) error {
	s.receiver = receiver.NewManager(cfg)
	if err := s.receiver.Start(func(line string) bool {
		if !s.processor.Submit(line) {
			log.Printf("processor queue full, dropped log: %s", line[:min(50, len(line))])
			return false
		}
		return true
	}); err != nil {
		return err
	}
	s.receiverRunning = true
	return nil
}

func (s *Server) stopReceiverLocked() error {
	if s.receiver == nil {
		s.receiverRunning = false
		return nil
	}

	if err := s.receiver.Stop(); err != nil {
		return err
	}

	s.receiverRunning = false
	return nil
}
