// processor/processor.go - 数据处理器
package processor

import (
	"context"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/models"
	"math/rand"
	"sync"
	"time"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ProcessorStats 处理器统计
type ProcessorStats struct {
	ReceivedCount   int64 // 接收总数
	ProcessedCount  int64 // 处理成功数
	DroppedCount    int64 // 丢弃数（队列满）
	ParseErrorCount int64 // 解析错误数
}

// Processor 数据处理器
type Processor struct {
	config         config.ProcessorConfig
	inputChan      chan string
	outputChan     chan *models.LogEntry
	workerStopChan chan struct{}
	parser         Parser
	storage        Storage
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	stopped        bool
	stats          ProcessorStats
}

// Parser 解析器接口
type Parser interface {
	Parse(line string) (*models.LogEntry, error)
}

// Storage 存储接口
type Storage interface {
	SaveBatch(entries []*models.LogEntry) error
}

// NewProcessor 创建新的处理器
func NewProcessor(cfg config.ProcessorConfig, parser Parser, storage Storage) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	// 队列容量基于BatchSize计算，确保足够的缓冲空间应对突发流量
	// 容量 = BatchSize * 200，最小100,000，最大500,000
	queueSize := cfg.BatchSize * 200
	if queueSize < 100000 {
		queueSize = 100000
	}
	if queueSize > 500000 {
		queueSize = 500000
	}

	p := &Processor{
		config:         cfg,
		inputChan:      make(chan string, queueSize),
		outputChan:     make(chan *models.LogEntry, queueSize),
		workerStopChan: make(chan struct{}, 1024),
		parser:         parser,
		storage:        storage,
		ctx:            ctx,
		cancel:         cancel,
	}

	return p
}

// Start 启动处理器
func (p *Processor) Start() {
	cfg := p.getConfigSnapshot()

	// 启动工作协程
	for i := 0; i < cfg.WorkerCount; i++ {
		p.startWorker(i)
	}

	// 启动批处理协程
	p.wg.Add(1)
	go p.batchProcessor()

	log.Printf("Processor started with %d workers", cfg.WorkerCount)
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
		// 队列满，记录警告日志（每1000条丢弃记录一次，避免日志风暴）
		if rand.Intn(1000) == 0 {
			p.mu.RLock()
			dropped := p.stats.DroppedCount
			p.mu.RUnlock()
			log.Printf("[WARN] Processor input queue full (%d/%d), total dropped ~%d logs",
				len(p.inputChan), cap(p.inputChan), dropped)
		}
		p.mu.Lock()
		p.stats.DroppedCount++
		p.mu.Unlock()
		return false
	}
}

// worker 工作协程
func (p *Processor) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.workerStopChan:
			log.Printf("Processor worker %d stopped by config update", id)
			return
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
	p.mu.Lock()
	p.stats.ReceivedCount++
	p.mu.Unlock()

	// 解析
	parser := p.getParser()
	entry, err := parser.Parse(line)
	if err != nil {
		p.mu.Lock()
		p.stats.ParseErrorCount++
		p.mu.Unlock()
		log.Printf("[PROCESSOR] Parse error: %v, line: %s", err, line[:min(50, len(line))])
		return
	}

	// 输出
	select {
	case p.outputChan <- entry:
		p.mu.Lock()
		p.stats.ProcessedCount++
		p.mu.Unlock()
	case <-p.ctx.Done():
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

	cfg := p.getConfigSnapshot()
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}
	batchTimeout := cfg.BatchTimeout
	if batchTimeout <= 0 {
		batchTimeout = 1000
	}

	batch := make([]*models.LogEntry, 0, batchSize)
	ticker := time.NewTicker(time.Duration(batchTimeout) * time.Millisecond)
	defer ticker.Stop()
	currentTimeout := batchTimeout

	for {
		cfg = p.getConfigSnapshot()
		batchLimit := cfg.BatchSize
		if batchLimit <= 0 {
			batchLimit = 1
		}
		timeout := cfg.BatchTimeout
		if timeout <= 0 {
			timeout = 1000
		}
		if timeout != currentTimeout {
			ticker.Reset(time.Duration(timeout) * time.Millisecond)
			currentTimeout = timeout
		}

		select {
		case entry, ok := <-p.outputChan:
			if !ok {
				if len(batch) > 0 {
					p.saveBatch(batch)
				}
				return
			}
			batch = append(batch, entry)
			if len(batch) >= batchLimit {
				p.saveBatch(batch)
				batch = make([]*models.LogEntry, 0, batchLimit)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				p.saveBatch(batch)
				batch = make([]*models.LogEntry, 0, batchLimit)
			}
		case <-p.ctx.Done():
			// 处理剩余数据
			for entry := range p.outputChan {
				batch = append(batch, entry)
				if len(batch) >= batchLimit {
					p.saveBatch(batch)
					batch = make([]*models.LogEntry, 0, batchLimit)
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
	if cfg.WorkerCount < 1 {
		cfg.WorkerCount = 1
	}
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 1
	}
	if cfg.BatchTimeout < 1 {
		cfg.BatchTimeout = 1
	}

	p.mu.Lock()
	if p.stopped {
		p.config = cfg
		p.mu.Unlock()
		return
	}
	oldWorkerCount := p.config.WorkerCount
	p.config = cfg
	p.mu.Unlock()

	diff := cfg.WorkerCount - oldWorkerCount
	switch {
	case diff > 0:
		for i := 0; i < diff; i++ {
			p.startWorker(oldWorkerCount + i)
		}
		log.Printf("Processor worker count scaled up: %d -> %d", oldWorkerCount, cfg.WorkerCount)
	case diff < 0:
		for i := 0; i < -diff; i++ {
			p.workerStopChan <- struct{}{}
		}
		log.Printf("Processor worker count scaled down: %d -> %d", oldWorkerCount, cfg.WorkerCount)
	}
}

// SetParser 设置解析器
func (p *Processor) SetParser(parser Parser) {
	p.mu.Lock()
	p.parser = parser
	p.mu.Unlock()
}

func (p *Processor) startWorker(id int) {
	p.wg.Add(1)
	go p.worker(id)
}

func (p *Processor) getConfigSnapshot() config.ProcessorConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

func (p *Processor) getParser() Parser {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.parser
}

// GetStats 获取处理统计
func (p *Processor) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return map[string]interface{}{
		"input_queue_size":  len(p.inputChan),
		"output_queue_size": len(p.outputChan),
		"worker_count":      p.config.WorkerCount,
		"batch_size":        p.config.BatchSize,
		"received_count":    p.stats.ReceivedCount,
		"processed_count":   p.stats.ProcessedCount,
		"dropped_count":     p.stats.DroppedCount,
		"parse_error_count": p.stats.ParseErrorCount,
	}
}
