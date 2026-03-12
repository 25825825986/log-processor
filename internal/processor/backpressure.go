// processor/backpressure.go - 背压管理机制
package processor

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// BackpressureLevel 背压级别
type BackpressureLevel int32

const (
	BackpressureNone     BackpressureLevel = 0 // 无背压
	BackpressureLight    BackpressureLevel = 1 // 轻度背压（队列 60%）
	BackpressureModerate BackpressureLevel = 2 // 中度背压（队列 80%）
	BackpressureSevere   BackpressureLevel = 3 // 重度背压（队列 95%）
)

// BackpressureManager 背压管理器
type BackpressureManager struct {
	level      int32 // 原子操作
	onLevelChange func(BackpressureLevel)
	mu         sync.RWMutex
	history    []backpressureSample
}

type backpressureSample struct {
	level     BackpressureLevel
	timestamp time.Time
}

// NewBackpressureManager 创建背压管理器
func NewBackpressureManager(onChange func(BackpressureLevel)) *BackpressureManager {
	return &BackpressureManager{
		onLevelChange: onChange,
		history:       make([]backpressureSample, 0, 100),
	}
}

// UpdateLevel 更新背压级别
func (bp *BackpressureManager) UpdateLevel(queueSize, queueCap int) {
	ratio := float64(queueSize) / float64(queueCap)
	
	var newLevel BackpressureLevel
	switch {
	case ratio >= 0.95:
		newLevel = BackpressureSevere
	case ratio >= 0.80:
		newLevel = BackpressureModerate
	case ratio >= 0.60:
		newLevel = BackpressureLight
	default:
		newLevel = BackpressureNone
	}
	
	oldLevel := BackpressureLevel(atomic.LoadInt32(&bp.level))
	if oldLevel != newLevel {
		atomic.StoreInt32(&bp.level, int32(newLevel))
		
		bp.mu.Lock()
		bp.history = append(bp.history, backpressureSample{
			level:     newLevel,
			timestamp: time.Now(),
		})
		// 只保留最近100个样本
		if len(bp.history) > 100 {
			bp.history = bp.history[len(bp.history)-100:]
		}
		bp.mu.Unlock()
		
		if bp.onLevelChange != nil {
			bp.onLevelChange(newLevel)
		}
		
		log.Printf("[Backpressure] 级别变化: %s -> %s (队列: %d/%d, %.1f%%)",
			bp.levelString(oldLevel), bp.levelString(newLevel), queueSize, queueCap, ratio*100)
	}
}

// GetLevel 获取当前背压级别
func (bp *BackpressureManager) GetLevel() BackpressureLevel {
	return BackpressureLevel(atomic.LoadInt32(&bp.level))
}

// GetDelay 根据背压级别获取建议延迟
func (bp *BackpressureManager) GetDelay() time.Duration {
	switch bp.GetLevel() {
	case BackpressureSevere:
		return 100 * time.Millisecond // 严重背压，大幅降速
	case BackpressureModerate:
		return 50 * time.Millisecond // 中度背压，明显降速
	case BackpressureLight:
		return 10 * time.Millisecond // 轻度背压，轻微降速
	default:
		return 0 // 无背压，全速
	}
}

// ShouldDrop 是否建议丢弃
func (bp *BackpressureManager) ShouldDrop() bool {
	return bp.GetLevel() == BackpressureSevere
}

// levelString 级别转字符串
func (bp *BackpressureManager) levelString(level BackpressureLevel) string {
	switch level {
	case BackpressureNone:
		return "无"
	case BackpressureLight:
		return "轻度"
	case BackpressureModerate:
		return "中度"
	case BackpressureSevere:
		return "严重"
	default:
		return "未知"
	}
}
