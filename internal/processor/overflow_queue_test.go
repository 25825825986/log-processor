package processor

import (
	"strings"
	"testing"
)

func TestDiskOverflowQueueEnqueueAndDrain(t *testing.T) {
	t.Parallel()

	q, err := NewDiskOverflowQueue(t.TempDir(), 10)
	if err != nil {
		t.Fatalf("create queue failed: %v", err)
	}

	if !q.Enqueue("line-1") || !q.Enqueue("line-2") || !q.Enqueue("line-3") {
		t.Fatalf("enqueue should succeed")
	}

	collected := make([]string, 0, 3)
	drained := q.Drain(10, func(line string) bool {
		collected = append(collected, line)
		return true
	})
	if drained != 3 {
		t.Fatalf("expected drained=3, got %d", drained)
	}

	stats := q.Stats()
	if stats.PendingCount != 0 {
		t.Fatalf("expected pending=0, got %d", stats.PendingCount)
	}
	if len(collected) != 3 {
		t.Fatalf("expected collected=3, got %d", len(collected))
	}
}

func TestDiskOverflowQueueRejectsWhenDiskLimitReached(t *testing.T) {
	t.Parallel()

	q, err := NewDiskOverflowQueue(t.TempDir(), 1)
	if err != nil {
		t.Fatalf("create queue failed: %v", err)
	}

	// 1MB 上限下，2MB 的日志必然写不进去。
	huge := strings.Repeat("x", 2*1024*1024)
	if q.Enqueue(huge) {
		t.Fatalf("enqueue should fail when disk limit exceeded")
	}

	stats := q.Stats()
	if stats.DroppedCount == 0 {
		t.Fatalf("expected dropped count > 0")
	}
}

func TestDiskOverflowQueueClear(t *testing.T) {
	t.Parallel()

	q, err := NewDiskOverflowQueue(t.TempDir(), 10)
	if err != nil {
		t.Fatalf("create queue failed: %v", err)
	}

	if !q.Enqueue("line-1") || !q.Enqueue("line-2") {
		t.Fatalf("enqueue should succeed")
	}

	if err := q.Clear(); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	stats := q.Stats()
	if stats.PendingCount != 0 {
		t.Fatalf("expected pending=0, got %d", stats.PendingCount)
	}

	drained := q.Drain(10, func(line string) bool {
		t.Fatalf("no line should be drained after clear, got %q", line)
		return true
	})
	if drained != 0 {
		t.Fatalf("expected drained=0, got %d", drained)
	}
}
