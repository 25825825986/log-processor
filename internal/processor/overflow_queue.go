package processor

import (
	"bufio"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strconv"
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

// DiskOverflowQueue 是一个轻量的磁盘溢写队列。
//
// 设计要点：
//   - 数据以 base64+'\n' 写入队列文件，readOffset 追踪消费位置
//   - readOffset 持久化到 .offset 辅助文件，进程重启后不会重复回放
//   - Enqueue 的容量检查基于 pending bytes（文件大小 - readOffset），而非全文件大小
//   - compactFile 先原子写入临时文件再 rename，确保写失败时数据不丢失
type DiskOverflowQueue struct {
	mu sync.Mutex

	filePath   string // 队列主文件路径
	offsetPath string // readOffset 持久化辅助文件路径
	maxBytes   int64
	readOffset int64 // 当前已消费到的字节偏移（同步持久化到 offsetPath）

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
	offsetPath := filepath.Join(dir, "overflow.offset")

	// 确保队列文件存在
	f, err := os.OpenFile(filePath, os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	_ = f.Close()

	// Bug 1 fix: 从持久化文件读取 readOffset，避免重启后重复回放
	readOffset := loadOffset(offsetPath)

	// 如果 offset 超过实际文件大小（文件被外部清理），重置
	if fi, err := os.Stat(filePath); err == nil && readOffset > fi.Size() {
		readOffset = 0
		_ = saveOffset(offsetPath, 0)
	}

	pending, err := countQueueLines(filePath, readOffset)
	if err != nil {
		return nil, err
	}

	return &DiskOverflowQueue{
		filePath:   filePath,
		offsetPath: offsetPath,
		maxBytes:   int64(maxDiskMB) * 1024 * 1024,
		readOffset: readOffset,
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

	// Bug 2 fix: 容量检查只统计未消费的字节（文件大小 - readOffset），
	// 已被 Drain 消费但尚未压缩的前缀不占用有效配额。
	pendingBytes := stat.Size() - q.readOffset
	if pendingBytes < 0 {
		pendingBytes = 0
	}
	if pendingBytes+int64(len(encoded)) > q.maxBytes {
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

	f, err := os.Open(q.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			q.writeErr++
		}
		return 0
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		q.writeErr++
		return 0
	}

	// offset 超过文件大小时，文件已被外部清理，重置
	if q.readOffset > stat.Size() {
		q.readOffset = 0
		q.pending = 0
		_ = saveOffset(q.offsetPath, 0)
	}

	if _, err := f.Seek(q.readOffset, io.SeekStart); err != nil {
		q.writeErr++
		return 0
	}

	reader := bufio.NewReader(f)
	drained := 0
	skipped := 0

	for drained+skipped < maxBatch {
		lineStartOffset := q.readOffset

		raw, readErr := reader.ReadString('\n')
		if readErr == io.EOF {
			if len(raw) > 0 {
				q.readOffset += int64(len(raw))
				if !q.processLine(raw, submit, &drained, &skipped) {
					q.readOffset = lineStartOffset
				}
			}
			break
		}
		if readErr != nil {
			q.writeErr++
			break
		}

		q.readOffset += int64(len(raw))
		if !q.processLine(raw, submit, &drained, &skipped) {
			q.readOffset = lineStartOffset
			break
		}
	}

	// Bug 1 fix: 每次 Drain 后持久化 readOffset，重启后从此处续读
	if drained+skipped > 0 {
		if err := saveOffset(q.offsetPath, q.readOffset); err != nil {
			q.writeErr++
		}
	}

	removed := int64(drained + skipped)
	if removed >= q.pending {
		q.pending = 0
	} else {
		q.pending -= removed
	}
	q.recovered += int64(drained)

	// 触发延迟压缩：全部读完，或已消费超过一半且 pending 较少
	fileSize := stat.Size()
	if (q.readOffset >= fileSize && q.pending == 0) ||
		(q.readOffset > fileSize/2 && q.pending < 100) {
		q.compactFile()
	}

	return drained
}

// processLine 解码并提交单行；返回 false 表示下游队列满，需回退 offset
func (q *DiskOverflowQueue) processLine(raw string, submit func(string) bool, drained, skipped *int) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		*skipped++
		q.dropped++
		return true
	}

	data, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		*skipped++
		q.dropped++
		return true
	}

	if !submit(string(data)) {
		return false
	}

	*drained++
	return true
}

// compactFile 将未消费的数据移到文件头部。
// Bug 3 fix: 先原子写入临时文件再 rename，避免 truncate 后写失败导致数据丢失。
func (q *DiskOverflowQueue) compactFile() {
	if q.readOffset == 0 {
		return
	}

	stat, err := os.Stat(q.filePath)
	if err != nil {
		return
	}

	// 全部已读：直接 truncate，无需保留任何数据
	if q.readOffset >= stat.Size() {
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
		_ = saveOffset(q.offsetPath, 0)
		return
	}

	// 部分已读：读取未消费数据
	src, err := os.Open(q.filePath)
	if err != nil {
		q.writeErr++
		return
	}
	if _, err := src.Seek(q.readOffset, io.SeekStart); err != nil {
		src.Close()
		q.writeErr++
		return
	}
	remaining, err := io.ReadAll(src)
	src.Close()
	if err != nil {
		q.writeErr++
		return
	}

	// Bug 3 fix: 先写临时文件，成功后再 rename 到队列文件路径，
	// 确保写失败时原队列文件完整保留。
	tmpPath := q.filePath + ".tmp"
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		q.writeErr++
		return
	}

	if _, err := tmp.Write(remaining); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		q.writeErr++
		return
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		q.writeErr++
		return
	}
	tmp.Close()

	// 原子替换：rename 在同一文件系统上是原子的
	if err := os.Rename(tmpPath, q.filePath); err != nil {
		os.Remove(tmpPath)
		q.writeErr++
		return
	}

	q.readOffset = 0
	_ = saveOffset(q.offsetPath, 0)
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
	_ = saveOffset(q.offsetPath, 0)
	return nil
}

// loadOffset 从辅助文件中读取持久化的 readOffset
func loadOffset(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

// saveOffset 将 readOffset 持久化到辅助文件
func saveOffset(path string, offset int64) error {
	return os.WriteFile(path, []byte(strconv.FormatInt(offset, 10)), 0o644)
}

// countQueueLines 统计从 startOffset 开始的未读行数
func countQueueLines(path string, startOffset int64) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	if startOffset > 0 {
		if _, err := f.Seek(startOffset, io.SeekStart); err != nil {
			return 0, err
		}
	}

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
