// processor/resilient_processor.go - 弹性处理器（带容错机制）
package processor

import (
	"log"
	"log-processor/internal/config"
	"time"
)

// ResilientProcessor 弹性处理器
type ResilientProcessor struct {
	*Processor
	backpressure *BackpressureManager
	overflow     *OverflowQueue
	stopDrain    chan struct{}
}

// NewResilientProcessor 创建弹性处理器
func NewResilientProcessor(cfg config.ProcessorConfig, parser Parser, storage Storage, overflowDir string) (*ResilientProcessor, error) {
	// 创建基础处理器
	baseProcessor := NewProcessor(cfg, parser, storage)
	
	// 创建背压管理器
	backpressure := NewBackpressureManager(func(level BackpressureLevel) {
		log.Printf("[ResilientProcessor] 背压级别: %d", level)
	})
	
	// 创建溢出队列
	overflow, err := NewOverflowQueue(overflowDir)
	if err != nil {
		log.Printf("[WARN] 溢出队列初始化失败: %v, 将使用无溢出模式", err)
		overflow = nil
	}
	
	rp := &ResilientProcessor{
		Processor:    baseProcessor,
		backpressure: backpressure,
		overflow:     overflow,
		stopDrain:    make(chan struct{}),
	}
	
	// 启动背压监控
	go rp.monitorBackpressure()
	
	// 启动溢出回填协程
	if overflow != nil {
		go rp.drainOverflow()
	}
	
	log.Println("[ResilientProcessor] 弹性处理器初始化完成")
	return rp, nil
}

// Submit 提交日志（带容错）
func (rp *ResilientProcessor) Submit(line string) bool {
	rp.mu.RLock()
	if rp.stopped {
		rp.mu.RUnlock()
		return false
	}
	rp.mu.RUnlock()
	
	// 检查背压级别
	if rp.backpressure.GetLevel() == BackpressureSevere {
		// 严重背压，尝试溢出到磁盘
		if rp.overflow != nil && rp.overflow.Write(line) {
			log.Printf("[ResilientProcessor] 数据已溢出到磁盘 (当前溢出: %d)",
				rp.overflow.GetStats()["overflow_count"])
			return true
		}
		// 溢出失败，记录统计
		rp.mu.Lock()
		rp.stats.DroppedCount++
		rp.mu.Unlock()
		return false
	}
	
	// 尝试写入队列
	select {
	case <-rp.ctx.Done():
		return false
	case rp.inputChan <- line:
		return true
	default:
		// 队列满，根据背压级别处理
		delay := rp.backpressure.GetDelay()
		if delay > 0 {
			time.Sleep(delay)
			// 再次尝试
			select {
			case rp.inputChan <- line:
				return true
			default:
			}
		}
		
		// 还是满，尝试溢出
		if rp.overflow != nil && rp.overflow.Write(line) {
			return true
		}
		
		// 最终失败
		rp.mu.Lock()
		rp.stats.DroppedCount++
		rp.mu.Unlock()
		return false
	}
}

// monitorBackpressure 监控背压
func (rp *ResilientProcessor) monitorBackpressure() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			queueSize := len(rp.inputChan)
			queueCap := cap(rp.inputChan)
			rp.backpressure.UpdateLevel(queueSize, queueCap)
		case <-rp.ctx.Done():
			return
		}
	}
}

// drainOverflow 回填溢出数据
func (rp *ResilientProcessor) drainOverflow() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if !rp.overflow.HasOverflow() {
				continue
			}
			
			// 队列空闲时才回填
			if len(rp.inputChan) < cap(rp.inputChan)/2 {
				batch, ok := rp.overflow.ReadBatch(100)
				if !ok {
					continue
				}
				
				for _, line := range batch {
					select {
					case rp.inputChan <- line:
					default:
						// 队列又满了，放回溢出队列
						rp.overflow.Write(line)
					}
				}
				
				log.Printf("[ResilientProcessor] 回填 %d 条溢出数据", len(batch))
			}
		case <-rp.stopDrain:
			return
		case <-rp.ctx.Done():
			return
		}
	}
}

// GetResilientStats 获取弹性处理统计
func (rp *ResilientProcessor) GetResilientStats() map[string]interface{} {
	stats := rp.GetStats()
	
	// 添加背压统计
	stats["backpressure_level"] = rp.backpressure.GetLevel()
	stats["backpressure_delay_ms"] = rp.backpressure.GetDelay().Milliseconds()
	
	// 添加溢出统计
	if rp.overflow != nil {
		overflowStats := rp.overflow.GetStats()
		for k, v := range overflowStats {
			stats["overflow_"+k] = v
		}
	}
	
	return stats
}

// Stop 停止
func (rp *ResilientProcessor) Stop() {
	close(rp.stopDrain)
	if rp.overflow != nil {
		rp.overflow.Close()
	}
	rp.Processor.Stop()
}
