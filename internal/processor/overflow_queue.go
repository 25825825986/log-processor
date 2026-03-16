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

// DiskOverflowQueue 是一个轻量的磁盘溢写队列
// 数据以 base64 + '\n' 方式写入，避免原始日志换行导致解析歧义。
type DiskOverflowQueue struct {
	mu sync.Mutex

	filePath string
	maxBytes int64

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
		filePath: filePath,
		maxBytes: int64(maxDiskMB) * 1024 * 1024,
		pending:  pending,
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

// Drain 从磁盘队列回灌到内存处理队列，返回成功回灌条数
func (q *DiskOverflowQueue) Drain(maxBatch int, submit func(string) bool) int {
	if maxBatch <= 0 {
		maxBatch = 1
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	f, err := os.OpenFile(q.filePath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		q.writeErr++
		return 0
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	drained := 0
	skipped := 0
	var consumedBytes int64
	var pushBackLine []byte

	for drained+skipped < maxBatch {
		raw, readErr := reader.ReadString('\n')
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			q.writeErr++
			break
		}

		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			consumedBytes += int64(len(raw))
			skipped++
			q.dropped++
			continue
		}

		data, decodeErr := base64.StdEncoding.DecodeString(trimmed)
		if decodeErr != nil {
			consumedBytes += int64(len(raw))
			skipped++
			q.dropped++
			continue
		}

		if !submit(string(data)) {
			pushBackLine = []byte(raw)
			break
		}

		consumedBytes += int64(len(raw))
		drained++
	}

	if consumedBytes == 0 {
		return drained
	}

	// 将未消费的数据回写到文件头部，保证文件始终只保留“待处理”的记录
	tail, err := io.ReadAll(reader)
	if err != nil {
		q.writeErr++
		return drained
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		q.writeErr++
		return drained
	}
	if err := f.Truncate(0); err != nil {
		q.writeErr++
		return drained
	}

	if len(pushBackLine) > 0 {
		if _, err := f.Write(pushBackLine); err != nil {
			q.writeErr++
			return drained
		}
	}
	if len(tail) > 0 {
		if _, err := f.Write(tail); err != nil {
			q.writeErr++
			return drained
		}
	}

	removed := int64(drained + skipped)
	if removed >= q.pending {
		q.pending = 0
	} else {
		q.pending -= removed
	}
	q.recovered += int64(drained)

	return drained
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

// Clear 清空磁盘溢出队列中的所有待处理数据。
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
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		q.writeErr++
		return err
	}

	q.pending = 0
	return nil
}

func countQueueLines(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
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
