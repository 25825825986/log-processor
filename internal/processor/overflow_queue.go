// processor/overflow_queue.go - 溢出队列（多级队列实现）
package processor

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxOverflowFiles    = 5               // 最大溢出文件数
	maxOverflowFileSize = 100 * 1024 * 1024 // 单个溢出文件最大 100MB
)

// OverflowQueue 溢出队列
type OverflowQueue struct {
	dir           string
	overflowCount int64
	drainCount    int64
	mu            sync.RWMutex
	currentFile   *os.File
	writer        *bufio.Writer
	fileIndex     int
	draining      bool
}

// NewOverflowQueue 创建溢出队列
func NewOverflowQueue(dir string) (*OverflowQueue, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建溢出目录失败: %w", err)
	}
	
	oq := &OverflowQueue{
		dir: dir,
	}
	
	// 启动清理协程
	go oq.cleanupRoutine()
	
	return oq, nil
}

// Write 写入溢出数据
func (oq *OverflowQueue) Write(data string) bool {
	oq.mu.Lock()
	defer oq.mu.Unlock()
	
	// 如果正在排空，直接返回失败
	if oq.draining {
		return false
	}
	
	// 检查是否需要创建新文件
	if oq.currentFile == nil {
		if err := oq.rotateFile(); err != nil {
			log.Printf("[OverflowQueue] 创建溢出文件失败: %v", err)
			return false
		}
	}
	
	// 检查文件大小
	info, err := oq.currentFile.Stat()
	if err == nil && info.Size() >= maxOverflowFileSize {
		oq.rotateFile()
	}
	
	// 写入数据
	_, err = oq.writer.WriteString(data + "\n")
	if err != nil {
		log.Printf("[OverflowQueue] 写入失败: %v", err)
		return false
	}
	
	oq.writer.Flush()
	atomic.AddInt64(&oq.overflowCount, 1)
	
	return true
}

// ReadBatch 批量读取溢出数据（用于回填）
func (oq *OverflowQueue) ReadBatch(batchSize int) ([]string, bool) {
	oq.mu.Lock()
	oq.draining = true
	oq.mu.Unlock()
	
	defer func() {
		oq.mu.Lock()
		oq.draining = false
		oq.mu.Unlock()
	}()
	
	// 获取所有溢出文件
	files, err := filepath.Glob(filepath.Join(oq.dir, "overflow_*.log"))
	if err != nil || len(files) == 0 {
		return nil, false
	}
	
	// 读取第一个文件
	file := files[0]
	data, err := oq.readFile(file, batchSize)
	if err != nil {
		log.Printf("[OverflowQueue] 读取文件失败: %v", err)
		return nil, false
	}
	
	// 如果文件已读完，删除它
	if len(data) < batchSize {
		os.Remove(file)
	}
	
	atomic.AddInt64(&oq.drainCount, int64(len(data)))
	return data, len(data) > 0
}

// HasOverflow 是否有溢出数据
func (oq *OverflowQueue) HasOverflow() bool {
	files, _ := filepath.Glob(filepath.Join(oq.dir, "overflow_*.log"))
	return len(files) > 0
}

// GetStats 获取统计
func (oq *OverflowQueue) GetStats() map[string]interface{} {
	files, _ := filepath.Glob(filepath.Join(oq.dir, "overflow_*.log"))
	
	var totalSize int64
	for _, f := range files {
		info, err := os.Stat(f)
		if err == nil {
			totalSize += info.Size()
		}
	}
	
	return map[string]interface{}{
		"overflow_count": atomic.LoadInt64(&oq.overflowCount),
		"drain_count":    atomic.LoadInt64(&oq.drainCount),
		"file_count":     len(files),
		"total_size":     totalSize,
	}
}

// rotateFile 轮转文件
func (oq *OverflowQueue) rotateFile() error {
	// 关闭旧文件
	if oq.currentFile != nil {
		oq.writer.Flush()
		oq.currentFile.Close()
	}
	
	// 创建新文件
	oq.fileIndex = (oq.fileIndex + 1) % maxOverflowFiles
	filename := filepath.Join(oq.dir, fmt.Sprintf("overflow_%d.log", oq.fileIndex))
	
	// 如果文件已存在，删除它
	os.Remove(filename)
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	
	oq.currentFile = file
	oq.writer = bufio.NewWriter(file)
	
	log.Printf("[OverflowQueue] 创建新溢出文件: %s", filename)
	return nil
}

// readFile 读取文件
func (oq *OverflowQueue) readFile(filename string, maxLines int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() && len(lines) < maxLines {
		lines = append(lines, scanner.Text())
	}
	
	return lines, scanner.Err()
}

// cleanupRoutine 清理协程
func (oq *OverflowQueue) cleanupRoutine() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		// 清理过期文件（超过24小时）
		oq.cleanupOldFiles()
	}
}

// cleanupOldFiles 清理旧文件
func (oq *OverflowQueue) cleanupOldFiles() {
	files, err := filepath.Glob(filepath.Join(oq.dir, "overflow_*.log"))
	if err != nil {
		return
	}
	
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			os.Remove(file)
			log.Printf("[OverflowQueue] 清理过期文件: %s", file)
		}
	}
}

// Close 关闭
func (oq *OverflowQueue) Close() error {
	oq.mu.Lock()
	defer oq.mu.Unlock()
	
	if oq.currentFile != nil {
		oq.writer.Flush()
		return oq.currentFile.Close()
	}
	return nil
}
