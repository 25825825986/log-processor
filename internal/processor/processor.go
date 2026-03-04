// processor/processor.go - 数据处理器
package processor

import (
	"context"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/models"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Processor 数据处理器
type Processor struct {
	config      config.ProcessorConfig
	inputChan   chan string
	outputChan  chan *models.LogEntry
	parser      Parser
	storage     Storage
	cleanRules  []CleanRule
	filterRules []FilterRule
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	stopped     bool
}

// Parser 解析器接口
type Parser interface {
	Parse(line string) (*models.LogEntry, error)
}

// Storage 存储接口
type Storage interface {
	SaveBatch(entries []*models.LogEntry) error
}

// CleanRule 清洗规则
type CleanRule struct {
	Field     string
	Operation string // trim, remove, replace, regex
	Value     string
	Regex     *regexp.Regexp
}

// FilterRule 过滤规则
type FilterRule struct {
	Field    string
	Operator string // eq, ne, gt, lt, contains, regex
	Value    string
	Regex    *regexp.Regexp
}

// NewProcessor 创建新的处理器
func NewProcessor(cfg config.ProcessorConfig, parser Parser, storage Storage) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	p := &Processor{
		config:      cfg,
		inputChan:   make(chan string, cfg.BatchSize*2),
		outputChan:  make(chan *models.LogEntry, cfg.BatchSize*2),
		parser:      parser,
		storage:     storage,
		cleanRules:  make([]CleanRule, 0),
		filterRules: make([]FilterRule, 0),
		ctx:         ctx,
		cancel:      cancel,
	}

	p.initRules(cfg)
	return p
}

// initRules 初始化规则
func (p *Processor) initRules(cfg config.ProcessorConfig) {
	// 转换清洗规则
	for _, rule := range cfg.CleanRules {
		cleanRule := CleanRule{
			Field:     rule.Field,
			Operation: rule.Operation,
			Value:     rule.Value,
		}
		if rule.Operation == "regex" && rule.Value != "" {
			cleanRule.Regex = regexp.MustCompile(rule.Value)
		}
		p.cleanRules = append(p.cleanRules, cleanRule)
	}

	// 转换过滤规则
	for _, rule := range cfg.FilterRules {
		filterRule := FilterRule{
			Field:    rule.Field,
			Operator: rule.Operator,
			Value:    rule.Value,
		}
		if rule.Operator == "regex" && rule.Value != "" {
			filterRule.Regex = regexp.MustCompile(rule.Value)
		}
		p.filterRules = append(p.filterRules, filterRule)
	}
}

// Start 启动处理器
func (p *Processor) Start() {
	// 启动工作协程
	for i := 0; i < p.config.WorkerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	// 启动批处理协程
	p.wg.Add(1)
	go p.batchProcessor()

	log.Printf("Processor started with %d workers", p.config.WorkerCount)
}

// Stop 停止处理器
func (p *Processor) Stop() {
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
	
	p.cancel()
	close(p.inputChan)
	
	// 使用超时等待，避免永久卡住
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// 正常完成
	case <-time.After(5 * time.Second):
		log.Println("[WARN] 处理器停止超时，强制关闭")
	}
	
	close(p.outputChan)
	log.Println("Processor stopped")
}

// Submit 提交日志行
func (p *Processor) Submit(line string) bool {
	p.mu.RLock()
	if p.stopped {
		p.mu.RUnlock()
		return false
	}
	p.mu.RUnlock()

	select {
	case <-p.ctx.Done():
		return false
	case p.inputChan <- line:
		return true
	default:
		return false
	}
}

// worker 工作协程
func (p *Processor) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case line, ok := <-p.inputChan:
			if !ok {
				return
			}
			p.processLine(line)
		case <-p.ctx.Done():
			// 处理剩余数据
			for line := range p.inputChan {
				p.processLine(line)
			}
			return
		}
	}
}

// processLine 处理单行日志
func (p *Processor) processLine(line string) {
	// 解析
	entry, err := p.parser.Parse(line)
	if err != nil {
		log.Printf("Parse error: %v, line: %s", err, line)
		return
	}

	// 清洗
	p.clean(entry)

	// 过滤
	if !p.filter(entry) {
		return
	}

	// 输出
	select {
	case p.outputChan <- entry:
	case <-p.ctx.Done():
	}
}

// clean 清洗数据
func (p *Processor) clean(entry *models.LogEntry) {
	for _, rule := range p.cleanRules {
		p.applyCleanRule(entry, rule)
	}
}

// applyCleanRule 应用清洗规则
func (p *Processor) applyCleanRule(entry *models.LogEntry, rule CleanRule) {
	value := p.getFieldValue(entry, rule.Field)
	if value == "" {
		return
	}

	var newValue string
	switch rule.Operation {
	case "trim":
		newValue = strings.TrimSpace(value)
	case "remove":
		newValue = strings.ReplaceAll(value, rule.Value, "")
	case "replace":
		parts := strings.SplitN(rule.Value, "|", 2)
		if len(parts) == 2 {
			newValue = strings.ReplaceAll(value, parts[0], parts[1])
		}
	case "regex":
		if rule.Regex != nil {
			newValue = rule.Regex.ReplaceAllString(value, rule.Value)
		}
	case "lowercase":
		newValue = strings.ToLower(value)
	case "uppercase":
		newValue = strings.ToUpper(value)
	default:
		return
	}

	p.setFieldValue(entry, rule.Field, newValue)
}

// filter 过滤数据
func (p *Processor) filter(entry *models.LogEntry) bool {
	if len(p.filterRules) == 0 {
		return true
	}

	for _, rule := range p.filterRules {
		if !p.applyFilterRule(entry, rule) {
			return false
		}
	}
	return true
}

// applyFilterRule 应用过滤规则
func (p *Processor) applyFilterRule(entry *models.LogEntry, rule FilterRule) bool {
	value := p.getFieldValue(entry, rule.Field)

	switch rule.Operator {
	case "eq":
		return value == rule.Value
	case "ne":
		return value != rule.Value
	case "gt":
		return value > rule.Value
	case "lt":
		return value < rule.Value
	case "contains":
		return strings.Contains(value, rule.Value)
	case "not_contains":
		return !strings.Contains(value, rule.Value)
	case "regex":
		if rule.Regex != nil {
			return rule.Regex.MatchString(value)
		}
		return false
	case "empty":
		return value == ""
	case "not_empty":
		return value != ""
	default:
		return true
	}
}

// getFieldValue 获取字段值
func (p *Processor) getFieldValue(entry *models.LogEntry, field string) string {
	switch field {
	case "client_ip":
		return entry.ClientIP
	case "method":
		return entry.Method
	case "path":
		return entry.Path
	case "status_code":
		return string(rune(entry.StatusCode))
	case "user_agent":
		return entry.UserAgent
	case "referer":
		return entry.Referer
	case "level":
		return entry.Level
	case "source":
		return entry.Source
	default:
		if v, ok := entry.ExtraFields[field]; ok {
			return v
		}
		return ""
	}
}

// setFieldValue 设置字段值
func (p *Processor) setFieldValue(entry *models.LogEntry, field, value string) {
	switch field {
	case "client_ip":
		entry.ClientIP = value
	case "method":
		entry.Method = value
	case "path":
		entry.Path = value
	case "user_agent":
		entry.UserAgent = value
	case "referer":
		entry.Referer = value
	case "level":
		entry.Level = value
	case "source":
		entry.Source = value
	default:
		entry.ExtraFields[field] = value
	}
}

// batchProcessor 批处理协程
func (p *Processor) batchProcessor() {
	defer p.wg.Done()

	batch := make([]*models.LogEntry, 0, p.config.BatchSize)
	ticker := time.NewTicker(time.Duration(p.config.BatchTimeout) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-p.outputChan:
			if !ok {
				if len(batch) > 0 {
					p.saveBatch(batch)
				}
				return
			}
			batch = append(batch, entry)
			if len(batch) >= p.config.BatchSize {
				p.saveBatch(batch)
				batch = make([]*models.LogEntry, 0, p.config.BatchSize)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				p.saveBatch(batch)
				batch = make([]*models.LogEntry, 0, p.config.BatchSize)
			}
		case <-p.ctx.Done():
			// 处理剩余数据
			for entry := range p.outputChan {
				batch = append(batch, entry)
				if len(batch) >= p.config.BatchSize {
					p.saveBatch(batch)
					batch = make([]*models.LogEntry, 0, p.config.BatchSize)
				}
			}
			if len(batch) > 0 {
				p.saveBatch(batch)
			}
			return
		}
	}
}

// saveBatch 批量保存
func (p *Processor) saveBatch(batch []*models.LogEntry) {
	if err := p.storage.SaveBatch(batch); err != nil {
		log.Printf("Failed to save batch: %v", err)
	}
}

// UpdateConfig 更新配置
func (p *Processor) UpdateConfig(cfg config.ProcessorConfig) {
	p.config = cfg
	p.cleanRules = make([]CleanRule, 0)
	p.filterRules = make([]FilterRule, 0)
	p.initRules(cfg)
}

// GetStats 获取处理统计
func (p *Processor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"input_queue_size":  len(p.inputChan),
		"output_queue_size": len(p.outputChan),
		"worker_count":      p.config.WorkerCount,
		"batch_size":        p.config.BatchSize,
	}
}
