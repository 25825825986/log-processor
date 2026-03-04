// server/server.go - Web服务器
package server

import (
	"fmt"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/exporter"
	"log-processor/internal/models"
	"log-processor/internal/parser"
	"log-processor/internal/processor"
	"log-processor/internal/receiver"
	"log-processor/internal/storage"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

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
func NewServer(cfg *config.Config, store storage.Storage, proc *processor.Processor, recv *receiver.Manager) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

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

// getConfig 获取配置
func (s *Server) getConfig(c *gin.Context) {
	cfg := s.config.Get()
	c.JSON(http.StatusOK, cfg)
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

	// 导入文件
	importer := receiver.NewFileImporter()
	lineCount := 0
	err = importer.ImportFile(tempPath, func(line string) {
		s.processor.Submit(line)
		lineCount++
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"lines":      lineCount,
		"file":       file.Filename,
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// exportLogs 导出日志
func (s *Server) exportLogs(c *gin.Context) {
	var req models.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查询数据
	entries, err := s.storage.Query(req.Filter, 10000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

// getStatus 获取系统状态
func (s *Server) getStatus(c *gin.Context) {
	cfg := s.config.Get()
	stats := s.processor.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"config":    cfg,
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
