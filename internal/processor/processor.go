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

const (
	defaultOverflowDrainBatch      = 1000
	defaultOverflowDrainIntervalMS = 200
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
	DroppedCount    int64 // 丢弃数（内存队列满且溢写失败）
	ParseErrorCount int64 // 解析错误数
	SpillCount      int64 // 成功溢写到磁盘的条数
}

// Processor 数据处理器
type Processor struct {
	config         config.ProcessorConfig
	inputChan      chan string
	outputChan     chan *models.LogEntry
	workerStopChan chan struct{}
	overflow       *DiskOverflowQueue
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

	// 队列容量基于 BatchSize 计算，确保突发流量下有足够缓冲
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

	p.initOverflowQueue(cfg)
	return p
}

// Start 启动处理器
func (p *Processor) Start() {
	cfg := p.getConfigSnapshot()

	for i := 0; i < cfg.WorkerCount; i++ {
		p.startWorker(i)
	}

	p.wg.Add(1)
	go p.batchProcessor()

	// 启动磁盘溢写回灌协程（只有开启时才会实际回灌）
	p.wg.Add(1)
	go p.overflowDrainer()

	log.Printf("Processor started with %d workers", cfg.WorkerCount)
}

// Stop 停止处理器
func (p *Processor) Stop() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true
	p.mu.Unlock()

	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Println("[WARN] 处理器停止超时，部分后台协程可能仍在退出中")
	}

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
		// 内存队列满时优先尝试磁盘溢写
		if p.trySpill(line) {
			return true
		}

		if rand.Intn(1000) == 0 {
			p.mu.RLock()
			dropped := p.stats.DroppedCount
			p.mu.RUnlock()
			log.Printf("[WARN] Processor input queue full (%d/%d), dropped ~%d",
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
		case <-p.ctx.Done():
			// 退出前尽量消费输入队列中已经到达的数据
			for {
				select {
				case line, ok := <-p.inputChan:
					if !ok {
						return
					}
					p.processLine(line)
				default:
					return
				}
			}
		case line, ok := <-p.inputChan:
			if !ok {
				return
			}
			p.processLine(line)
		}
	}
}

// processLine 处理单行日志
func (p *Processor) processLine(line string) {
	p.mu.Lock()
	p.stats.ReceivedCount++
	p.mu.Unlock()

	entry, err := p.getParser().Parse(line)
	if err != nil {
		p.mu.Lock()
		p.stats.ParseErrorCount++
		p.mu.Unlock()
		log.Printf("[PROCESSOR] Parse error: %v, line: %s", err, line[:min(50, len(line))])
		return
	}

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
		limit := cfg.BatchSize
		if limit <= 0 {
			limit = 1
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
		case <-p.ctx.Done():
			// 退出前尽量把输出队列中已经产生的数据刷盘
			for {
				select {
				case entry := <-p.outputChan:
					if entry == nil {
						continue
					}
					batch = append(batch, entry)
					if len(batch) >= limit {
						p.saveBatch(batch)
						batch = make([]*models.LogEntry, 0, limit)
					}
				default:
					if len(batch) > 0 {
						p.saveBatch(batch)
					}
					return
				}
			}
		case entry := <-p.outputChan:
			if entry == nil {
				continue
			}
			batch = append(batch, entry)
			if len(batch) >= limit {
				p.saveBatch(batch)
				batch = make([]*models.LogEntry, 0, limit)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				p.saveBatch(batch)
				batch = make([]*models.LogEntry, 0, limit)
			}
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
	if cfg.OverflowDrainBatch < 1 {
		cfg.OverflowDrainBatch = defaultOverflowDrainBatch
	}
	if cfg.OverflowDrainIntervalMS < 1 {
		cfg.OverflowDrainIntervalMS = defaultOverflowDrainIntervalMS
	}
	if cfg.OverflowDir == "" {
		cfg.OverflowDir = defaultOverflowDir
	}
	if cfg.OverflowMaxDiskMB < 1 {
		cfg.OverflowMaxDiskMB = defaultOverflowMaxDiskM
	}

	var createdOverflow *DiskOverflowQueue
	var createErr error

	if cfg.OverflowEnabled {
		p.mu.RLock()
		needCreate := p.overflow == nil
		p.mu.RUnlock()
		if needCreate {
			createdOverflow, createErr = NewDiskOverflowQueue(cfg.OverflowDir, cfg.OverflowMaxDiskMB)
		}
	}

	p.mu.Lock()
	if p.stopped {
		p.config = cfg
		p.mu.Unlock()
		return
	}

	if cfg.OverflowEnabled && p.overflow == nil {
		if createErr != nil {
			log.Printf("[WARN] failed to create overflow queue: %v", createErr)
			cfg.OverflowEnabled = false
		} else if createdOverflow != nil {
			p.overflow = createdOverflow
		}
	}

	if p.overflow != nil {
		p.overflow.SetMaxDiskMB(cfg.OverflowMaxDiskMB)
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

func (p *Processor) getOverflow() *DiskOverflowQueue {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.overflow
}

func (p *Processor) initOverflowQueue(cfg config.ProcessorConfig) {
	if !cfg.OverflowEnabled {
		return
	}

	overflow, err := NewDiskOverflowQueue(cfg.OverflowDir, cfg.OverflowMaxDiskMB)
	if err != nil {
		log.Printf("[WARN] overflow queue disabled: %v", err)
		return
	}
	p.overflow = overflow
}

func (p *Processor) trySpill(line string) bool {
	cfg := p.getConfigSnapshot()
	if !cfg.OverflowEnabled {
		return false
	}

	overflow := p.getOverflow()
	if overflow == nil {
		return false
	}

	if overflow.Enqueue(line) {
		p.mu.Lock()
		p.stats.SpillCount++
		p.mu.Unlock()
		return true
	}
	return false
}

func (p *Processor) overflowDrainer() {
	defer p.wg.Done()

	cfg := p.getConfigSnapshot()
	intervalMS := cfg.OverflowDrainIntervalMS
	if intervalMS < 1 {
		intervalMS = defaultOverflowDrainIntervalMS
	}
	ticker := time.NewTicker(time.Duration(intervalMS) * time.Millisecond)
	defer ticker.Stop()

	currentInterval := intervalMS
	for {
		cfg = p.getConfigSnapshot()
		intervalMS = cfg.OverflowDrainIntervalMS
		if intervalMS < 1 {
			intervalMS = defaultOverflowDrainIntervalMS
		}
		if intervalMS != currentInterval {
			ticker.Reset(time.Duration(intervalMS) * time.Millisecond)
			currentInterval = intervalMS
		}

		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if !cfg.OverflowEnabled {
				continue
			}
			overflow := p.getOverflow()
			if overflow == nil {
				continue
			}

			batch := cfg.OverflowDrainBatch
			if batch < 1 {
				batch = defaultOverflowDrainBatch
			}

			recovered := overflow.Drain(batch, func(line string) bool {
				select {
				case p.inputChan <- line:
					return true
				default:
					return false
				}
			})

			if recovered > 0 && rand.Intn(200) == 0 {
				stats := overflow.Stats()
				log.Printf("[INFO] overflow drained=%d pending=%d file=%dB",
					recovered, stats.PendingCount, stats.FileSizeBytes)
			}
		}
	}
}

// GetStats 获取处理统计
func (p *Processor) GetStats() map[string]interface{} {
	p.mu.RLock()
	stats := p.stats
	cfg := p.config
	p.mu.RUnlock()

	result := map[string]interface{}{
		"input_queue_size":  len(p.inputChan),
		"output_queue_size": len(p.outputChan),
		"worker_count":      cfg.WorkerCount,
		"batch_size":        cfg.BatchSize,
		"received_count":    stats.ReceivedCount,
		"processed_count":   stats.ProcessedCount,
		"dropped_count":     stats.DroppedCount,
		"parse_error_count": stats.ParseErrorCount,
		"spill_count":       stats.SpillCount,
		"overflow_enabled":  cfg.OverflowEnabled,
	}

	overflow := p.getOverflow()
	if overflow == nil {
		result["overflow_queue_pending"] = int64(0)
		result["overflow_file_size_bytes"] = int64(0)
		result["overflow_enqueued_count"] = int64(0)
		result["overflow_recovered_count"] = int64(0)
		result["overflow_dropped_count"] = int64(0)
		result["overflow_write_error_count"] = int64(0)
		return result
	}

	ov := overflow.Stats()
	result["overflow_queue_pending"] = ov.PendingCount
	result["overflow_file_size_bytes"] = ov.FileSizeBytes
	result["overflow_enqueued_count"] = ov.EnqueuedCount
	result["overflow_recovered_count"] = ov.RecoveredCount
	result["overflow_dropped_count"] = ov.DroppedCount
	result["overflow_write_error_count"] = ov.WriteErrorCount
	return result
}
