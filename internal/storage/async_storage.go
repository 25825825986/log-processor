// storage/async_storage.go - 异步存储包装器
package storage

import (
	"context"
	"log"
	"log-processor/internal/models"
	"sync"
	"time"
)

// AsyncStorage 异步存储包装器
// 通过缓冲队列和批量写入最大化 SQLite 单线程性能
type AsyncStorage struct {
	storage       Storage                    // 底层存储
	buffer        chan *models.LogEntry      // 写入缓冲队列
	batchSize     int                        // 批量大小
	flushInterval time.Duration              // 强制刷新间隔
	wg            sync.WaitGroup             
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	stats         AsyncStats
}

// AsyncStats 异步存储统计
type AsyncStats struct {
	BufferedCount   int64 // 缓冲中数量
	FlushedCount    int64 // 已刷新数量
	DroppedCount    int64 // 丢弃数量（队列满）
	LastFlushTime   time.Time
	AvgFlushLatency int64 // 平均刷新延迟(ms)
}

// NewAsyncStorage 创建异步存储
func NewAsyncStorage(storage Storage, bufferSize int, batchSize int, flushInterval time.Duration) *AsyncStorage {
	ctx, cancel := context.WithCancel(context.Background())
	
	as := &AsyncStorage{
		storage:       storage,
		buffer:        make(chan *models.LogEntry, bufferSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		ctx:           ctx,
		cancel:        cancel,
	}
	
	// 启动写入协程
	as.wg.Add(1)
	go as.writeLoop()
	
	log.Printf("[AsyncStorage] 启动异步存储: buffer=%d, batch=%d, interval=%v",
		bufferSize, batchSize, flushInterval)
	
	return as
}

// Save 异步保存单条日志（非阻塞）
func (as *AsyncStorage) Save(entry *models.LogEntry) bool {
	select {
	case as.buffer <- entry:
		as.mu.Lock()
		as.stats.BufferedCount++
		as.mu.Unlock()
		return true
	default:
		// 队列满，丢弃日志（避免阻塞上游）
		as.mu.Lock()
		as.stats.DroppedCount++
		as.mu.Unlock()
		return false
	}
}

// SaveBatch 批量保存（兼容接口，实际转异步）
func (as *AsyncStorage) SaveBatch(entries []*models.LogEntry) error {
	dropped := 0
	for _, entry := range entries {
		if !as.Save(entry) {
			dropped++
		}
	}
	if dropped > 0 {
		log.Printf("[WARN] AsyncStorage 队列满，丢弃 %d 条日志", dropped)
	}
	return nil
}

// writeLoop 写入循环
func (as *AsyncStorage) writeLoop() {
	defer as.wg.Done()
	
	batch := make([]*models.LogEntry, 0, as.batchSize)
	ticker := time.NewTicker(as.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case entry, ok := <-as.buffer:
			if !ok {
				// 通道关闭，刷新剩余数据
				if len(batch) > 0 {
					as.flush(batch)
				}
				return
			}
			
			batch = append(batch, entry)
			
			// 达到批次大小立即刷新
			if len(batch) >= as.batchSize {
				as.flush(batch)
				batch = make([]*models.LogEntry, 0, as.batchSize)
				ticker.Reset(as.flushInterval)
			}
			
		case <-ticker.C:
			// 定时刷新，避免数据滞留
			if len(batch) > 0 {
				as.flush(batch)
				batch = make([]*models.LogEntry, 0, as.batchSize)
			}
			
		case <-as.ctx.Done():
			// 处理剩余数据
			for entry := range as.buffer {
				batch = append(batch, entry)
				if len(batch) >= as.batchSize {
					as.flush(batch)
					batch = make([]*models.LogEntry, 0, as.batchSize)
				}
			}
			if len(batch) > 0 {
				as.flush(batch)
			}
			return
		}
	}
}

// flush 刷新到存储
func (as *AsyncStorage) flush(batch []*models.LogEntry) {
	if len(batch) == 0 {
		return
	}
	
	start := time.Now()
	
	// 使用底层存储批量保存
	if err := as.storage.SaveBatch(batch); err != nil {
		log.Printf("[ERROR] AsyncStorage 批量保存失败: %v", err)
		return
	}
	
	latency := time.Since(start).Milliseconds()
	
	as.mu.Lock()
	as.stats.FlushedCount += int64(len(batch))
	as.stats.BufferedCount -= int64(len(batch))
	as.stats.LastFlushTime = time.Now()
	// 移动平均
	as.stats.AvgFlushLatency = (as.stats.AvgFlushLatency*9 + latency) / 10
	as.mu.Unlock()
}

// Query 查询（透传）
func (as *AsyncStorage) Query(filter models.FilterCondition, limit, offset int) ([]*models.LogEntry, error) {
	return as.storage.Query(filter, limit, offset)
}

// Count 统计（透传）
func (as *AsyncStorage) Count(filter models.FilterCondition) (int64, error) {
	return as.storage.Count(filter)
}

// Statistics 统计（透传）
func (as *AsyncStorage) Statistics(filter models.FilterCondition) (*models.Statistics, error) {
	return as.storage.Statistics(filter)
}

// Delete 删除（透传）
func (as *AsyncStorage) Delete(id string) error {
	return as.storage.Delete(id)
}

// Clear 清空（透传）
func (as *AsyncStorage) Clear() error {
	return as.storage.Clear()
}

// Close 关闭
func (as *AsyncStorage) Close() error {
	log.Println("[AsyncStorage] 正在关闭...")
	
	as.cancel()
	close(as.buffer)
	
	// 等待写入完成
	done := make(chan struct{})
	go func() {
		as.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("[AsyncStorage] 写入协程已退出")
	case <-time.After(10 * time.Second):
		log.Println("[WARN] AsyncStorage 关闭超时")
	}
	
	return as.storage.Close()
}

// GetStats 获取统计
func (as *AsyncStorage) GetStats() AsyncStats {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.stats
}
