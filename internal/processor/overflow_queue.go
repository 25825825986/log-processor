package processor

import (
	"bufio"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	defaultOverflowDir      = "./data/overflow"
	defaultOverflowMaxDiskM = 512
)

// OverflowQueueStats 表示磁盘溢写队列统计
type OverflowQueueStats struct {
	PendingCount    int64 `json:"pending_count"`
	FileSizeBytes   int64 `json:"file_size_bytes"`
	EnqueuedCount   int64 `json:"enqueued_count"`
	RecoveredCount  int64 `json:"recovered_count"`
	DroppedCount    int64 `json:"dropped_count"`
	WriteErrorCount int64 `json:"write_error_count"`
}

// DiskOverflowQueue 是一个轻量的磁盘溢写队列（优化版）
// 使用 offset 追踪避免大文件全量读写
type DiskOverflowQueue struct {
	mu sync.Mutex

	filePath   string
	maxBytes   int64
	readOffset int64 // 当前读取位置

	pending   int64
	enqueued  int64
	recovered int64
	dropped   int64
	writeErr  int64
}

// NewDiskOverflowQueue 创建磁盘溢写队列
func NewDiskOverflowQueue(dir string, maxDiskMB int) (*DiskOverflowQueue, error) {
	if dir == "" {
		dir = defaultOverflowDir
	}
	if maxDiskMB <= 0 {
		maxDiskMB = defaultOverflowMaxDiskM
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(dir, "overflow.queue")
	f, err := os.OpenFile(filePath, os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	_ = f.Close()

	pending, err := countQueueLines(filePath)
	if err != nil {
		return nil, err
	}

	return &DiskOverflowQueue{
		filePath:   filePath,
		maxBytes:   int64(maxDiskMB) * 1024 * 1024,
		readOffset: 0,
		pending:    pending,
	}, nil
}

// SetMaxDiskMB 动态更新磁盘上限
func (q *DiskOverflowQueue) SetMaxDiskMB(maxDiskMB int) {
	if maxDiskMB <= 0 {
		maxDiskMB = defaultOverflowMaxDiskM
	}

	q.mu.Lock()
	q.maxBytes = int64(maxDiskMB) * 1024 * 1024
	q.mu.Unlock()
}

// Enqueue 将日志写入溢写队列
func (q *DiskOverflowQueue) Enqueue(line string) bool {
	encoded := base64.StdEncoding.EncodeToString([]byte(line)) + "\n"

	q.mu.Lock()
	defer q.mu.Unlock()

	stat, err := os.Stat(q.filePath)
	if err != nil {
		q.writeErr++
		q.dropped++
		return false
	}

	if stat.Size()+int64(len(encoded)) > q.maxBytes {
		q.dropped++
		return false
	}

	f, err := os.OpenFile(q.filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		q.writeErr++
		q.dropped++
		return false
	}
	defer f.Close()

	if _, err := f.WriteString(encoded); err != nil {
		q.writeErr++
		q.dropped++
		return false
	}

	q.enqueued++
	q.pending++
	return true
}

// Drain 从磁盘队列回灌到内存处理队列（优化版：使用 offset 追踪）
func (q *DiskOverflowQueue) Drain(maxBatch int, submit func(string) bool) int {
	if maxBatch <= 0 {
		maxBatch = 1
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// 打开文件用于读取
	f, err := os.Open(q.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			q.writeErr++
		}
		return 0
	}
	defer f.Close()

	// 获取文件大小
	stat, err := f.Stat()
	if err != nil {
		q.writeErr++
		return 0
	}

	// 如果 offset 超过文件大小，说明文件已被外部清理，重置 offset
	if q.readOffset > stat.Size() {
		q.readOffset = 0
		q.pending = 0
	}

	// 定位到上次读取的位置
	if _, err := f.Seek(q.readOffset, io.SeekStart); err != nil {
		q.writeErr++
		return 0
	}

	reader := bufio.NewReader(f)
	drained := 0
	skipped := 0

	for drained+skipped < maxBatch {
		// 记录当前行开始位置
		lineStartOffset := q.readOffset

		raw, readErr := reader.ReadString('\n')
		if readErr == io.EOF {
			// 到达文件末尾
			if len(raw) > 0 {
				// 处理最后一行（没有换行符）
				q.readOffset += int64(len(raw))
				if processed := q.processLine(raw, submit, &drained, &skipped); !processed {
					// 如果处理失败（队列满），回退 offset
					q.readOffset = lineStartOffset
				}
			}
			break
		}
		if readErr != nil {
			q.writeErr++
			break
		}

		// 更新 offset
		q.readOffset += int64(len(raw))

		// 处理当前行
		if processed := q.processLine(raw, submit, &drained, &skipped); !processed {
			// 如果处理失败（队列满），回退 offset 并退出
			q.readOffset = lineStartOffset
			break
		}
	}

	// 更新统计
	removed := int64(drained + skipped)
	if removed >= q.pending {
		q.pending = 0
	} else {
		q.pending -= removed
	}
	q.recovered += int64(drained)

	// 如果文件已完全读取，且没有待处理数据，压缩文件
	if q.readOffset >= stat.Size() && q.pending == 0 {
		q.compactFile()
	}

	// 如果 offset 已超过文件一半且待处理数据较少，考虑压缩
	if q.readOffset > stat.Size()/2 && q.pending < 100 {
		q.compactFile()
	}

	return drained
}

// processLine 处理单行数据
func (q *DiskOverflowQueue) processLine(raw string, submit func(string) bool, drained, skipped *int) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		*skipped++
		q.dropped++
		return true
	}

	data, decodeErr := base64.StdEncoding.DecodeString(trimmed)
	if decodeErr != nil {
		*skipped++
		q.dropped++
		return true
	}

	if !submit(string(data)) {
		// 队列满，无法提交
		return false
	}

	*drained++
	return true
}

// compactFile 压缩文件：删除已消费的数据
func (q *DiskOverflowQueue) compactFile() {
	// 如果 offset 为 0 或文件为空，无需压缩
	if q.readOffset == 0 {
		return
	}

	// 如果文件已完全读取，直接清空
	stat, err := os.Stat(q.filePath)
	if err != nil {
		return
	}

	if q.readOffset >= stat.Size() {
		// 完全读取，直接 truncate
		f, err := os.OpenFile(q.filePath, os.O_RDWR, 0o644)
		if err != nil {
			q.writeErr++
			return
		}
		defer f.Close()

		if err := f.Truncate(0); err != nil {
			q.writeErr++
			return
		}
		q.readOffset = 0
		q.pending = 0
		return
	}

	// 部分读取：将未读数据移到文件头部
	// 读取未消费的数据
	f, err := os.Open(q.filePath)
	if err != nil {
		q.writeErr++
		return
	}

	if _, err := f.Seek(q.readOffset, io.SeekStart); err != nil {
		f.Close()
		q.writeErr++
		return
	}

	remaining, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		q.writeErr++
		return
	}

	// 重写文件
	f, err = os.OpenFile(q.filePath, os.O_RDWR, 0o644)
	if err != nil {
		q.writeErr++
		return
	}
	defer f.Close()

	if err := f.Truncate(0); err != nil {
		q.writeErr++
		return
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		q.writeErr++
		return
	}

	if _, err := f.Write(remaining); err != nil {
		q.writeErr++
		return
	}

	// 重置 offset
	q.readOffset = 0
}

// Stats 返回当前队列统计
func (q *DiskOverflowQueue) Stats() OverflowQueueStats {
	q.mu.Lock()
	defer q.mu.Unlock()

	var size int64
	if stat, err := os.Stat(q.filePath); err == nil {
		size = stat.Size()
	}

	return OverflowQueueStats{
		PendingCount:    q.pending,
		FileSizeBytes:   size,
		EnqueuedCount:   q.enqueued,
		RecoveredCount:  q.recovered,
		DroppedCount:    q.dropped,
		WriteErrorCount: q.writeErr,
	}
}

// Clear 清空磁盘溢出队列中的所有待处理数据
func (q *DiskOverflowQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	f, err := os.OpenFile(q.filePath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		q.writeErr++
		return err
	}
	defer f.Close()

	if err := f.Truncate(0); err != nil {
		q.writeErr++
		return err
	}

	q.readOffset = 0
	q.pending = 0
	return nil
}

// countQueueLines 统计队列文件的行数
func countQueueLines(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, 64*1024)
	var count int64
	var lastByte byte
	var hasData bool

	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			hasData = true
			lastByte = buf[n-1]
			for _, b := range buf[:n] {
				if b == '\n' {
					count++
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, readErr
		}
	}

	if hasData && lastByte != '\n' {
		count++
	}
	return count, nil
}
