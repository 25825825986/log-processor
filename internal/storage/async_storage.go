// storage/async_storage.go - 异步存储包装器
package storage

import (
	"context"
	"fmt"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/models"
	"sync"
	"time"
)

// AsyncStorage 通过缓冲队列和批量写入提升 SQLite 吞吐。
type AsyncStorage struct {
	storage       Storage
	buffer        chan *models.LogEntry
	batchSize     int
	flushInterval time.Duration
	clearCh       chan chan struct{}
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	stats         AsyncStats
}

// AsyncStats 异步存储统计信息。
type AsyncStats struct {
	BufferedCount   int64
	FlushedCount    int64
	DroppedCount    int64
	LastFlushTime   time.Time
	AvgFlushLatency int64
}

// NewAsyncStorage 创建异步存储包装。
func NewAsyncStorage(storage Storage, bufferSize int, batchSize int, flushInterval time.Duration) *AsyncStorage {
	ctx, cancel := context.WithCancel(context.Background())

	as := &AsyncStorage{
		storage:       storage,
		buffer:        make(chan *models.LogEntry, bufferSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		clearCh:       make(chan chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}

	as.wg.Add(1)
	go as.writeLoop()

	log.Printf("[AsyncStorage] 启动异步存储: buffer=%d, batch=%d, interval=%v",
		bufferSize, batchSize, flushInterval)

	return as
}

// Save 异步保存单条日志（非阻塞）。
func (as *AsyncStorage) Save(entry *models.LogEntry) bool {
	select {
	case as.buffer <- entry:
		as.mu.Lock()
		as.stats.BufferedCount++
		as.mu.Unlock()
		return true
	default:
		as.mu.Lock()
		as.stats.DroppedCount++
		as.mu.Unlock()
		return false
	}
}

// SaveBatch 批量保存（通过背压避免队列满时直接丢日志）。
func (as *AsyncStorage) SaveBatch(entries []*models.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	for i, entry := range entries {
		select {
		case <-as.ctx.Done():
			remaining := len(entries) - i
			if remaining > 0 {
				as.mu.Lock()
				as.stats.DroppedCount += int64(remaining)
				as.mu.Unlock()
				log.Printf("[WARN] AsyncStorage 已关闭，丢弃 %d 条待写入日志", remaining)
			}
			return context.Canceled
		case as.buffer <- entry:
			as.mu.Lock()
			as.stats.BufferedCount++
			as.mu.Unlock()
		}
	}

	return nil
}

// SaveBatchDirect 直接写入底层存储，适合离线导入场景绕过异步队列。
func (as *AsyncStorage) SaveBatchDirect(entries []*models.LogEntry) error {
	return as.storage.SaveBatch(entries)
}

func (as *AsyncStorage) writeLoop() {
	defer as.wg.Done()

	batch := make([]*models.LogEntry, 0, as.batchSize)
	ticker := time.NewTicker(as.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-as.buffer:
			if !ok {
				if len(batch) > 0 {
					as.flush(batch)
				}
				return
			}

			batch = append(batch, entry)
			if len(batch) >= as.batchSize {
				as.flush(batch)
				batch = make([]*models.LogEntry, 0, as.batchSize)
				ticker.Reset(as.flushInterval)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				as.flush(batch)
				batch = make([]*models.LogEntry, 0, as.batchSize)
			}

		case done := <-as.clearCh:
			discarded := int64(len(batch))
			batch = make([]*models.LogEntry, 0, as.batchSize)

			for {
				select {
				case <-as.buffer:
					discarded++
				default:
					as.mu.Lock()
					if discarded >= as.stats.BufferedCount {
						as.stats.BufferedCount = 0
					} else {
						as.stats.BufferedCount -= discarded
					}
					as.mu.Unlock()
					close(done)
					goto nextLoop
				}
			}

		case <-as.ctx.Done():
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

	nextLoop:
	}
}

func (as *AsyncStorage) flush(batch []*models.LogEntry) {
	if len(batch) == 0 {
		return
	}

	start := time.Now()
	if err := as.storage.SaveBatch(batch); err != nil {
		log.Printf("[ERROR] AsyncStorage 批量保存失败: %v", err)
		return
	}

	latency := time.Since(start).Milliseconds()
	as.mu.Lock()
	as.stats.FlushedCount += int64(len(batch))
	as.stats.BufferedCount -= int64(len(batch))
	as.stats.LastFlushTime = time.Now()
	as.stats.AvgFlushLatency = (as.stats.AvgFlushLatency*9 + latency) / 10
	as.mu.Unlock()
}

func (as *AsyncStorage) Query(filter models.FilterCondition, limit, offset int) ([]*models.LogEntry, error) {
	return as.storage.Query(filter, limit, offset)
}

func (as *AsyncStorage) Count(filter models.FilterCondition) (int64, error) {
	return as.storage.Count(filter)
}

func (as *AsyncStorage) Statistics(filter models.FilterCondition) (*models.Statistics, error) {
	return as.storage.Statistics(filter)
}

func (as *AsyncStorage) Delete(id string) error {
	return as.storage.Delete(id)
}

func (as *AsyncStorage) Clear() error {
	done := make(chan struct{})
	select {
	case as.clearCh <- done:
		<-done
	case <-as.ctx.Done():
		return context.Canceled
	}

	return as.storage.Clear()
}

func (as *AsyncStorage) UpdateConfig(cfg config.StorageConfig) error {
	updater, ok := as.storage.(interface {
		UpdateConfig(config.StorageConfig) error
	})
	if !ok {
		return fmt.Errorf("underlying storage does not support runtime config update")
	}
	return updater.UpdateConfig(cfg)
}

func (as *AsyncStorage) Vacuum() error {
	vacuumer, ok := as.storage.(interface {
		Vacuum() error
	})
	if !ok {
		return fmt.Errorf("underlying storage does not support vacuum")
	}
	return vacuumer.Vacuum()
}

// Close 关闭异步写入并等待落盘完成。
func (as *AsyncStorage) Close() error {
	log.Println("[AsyncStorage] 正在关闭...")

	as.cancel()
	close(as.buffer)

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

func (as *AsyncStorage) GetStats() AsyncStats {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.stats
}
