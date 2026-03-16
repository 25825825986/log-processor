package storage

import (
	"log-processor/internal/models"
	"sync"
	"testing"
	"time"
)

type slowMockStorage struct {
	mu    sync.Mutex
	saved int
	delay time.Duration
}

func (m *slowMockStorage) SaveBatch(entries []*models.LogEntry) error {
	time.Sleep(m.delay)
	m.mu.Lock()
	m.saved += len(entries)
	m.mu.Unlock()
	return nil
}

func (m *slowMockStorage) Query(filter models.FilterCondition, limit, offset int) ([]*models.LogEntry, error) {
	return nil, nil
}

func (m *slowMockStorage) Count(filter models.FilterCondition) (int64, error) {
	return 0, nil
}

func (m *slowMockStorage) Statistics(filter models.FilterCondition) (*models.Statistics, error) {
	return &models.Statistics{}, nil
}

func (m *slowMockStorage) Delete(id string) error {
	return nil
}

func (m *slowMockStorage) Clear() error {
	return nil
}

func (m *slowMockStorage) Close() error {
	return nil
}

func (m *slowMockStorage) SavedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saved
}

func TestAsyncStorageSaveBatchBackpressureNoDrop(t *testing.T) {
	mock := &slowMockStorage{delay: 2 * time.Millisecond}
	as := NewAsyncStorage(mock, 2, 1, time.Millisecond)
	defer as.Close()

	total := 200
	entries := make([]*models.LogEntry, 0, total)
	for i := 0; i < total; i++ {
		entries = append(entries, models.NewLogEntry())
	}

	if err := as.SaveBatch(entries); err != nil {
		t.Fatalf("SaveBatch returned error: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if mock.SavedCount() == total {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := mock.SavedCount(); got != total {
		t.Fatalf("saved count mismatch: got=%d want=%d", got, total)
	}

	stats := as.GetStats()
	if stats.DroppedCount != 0 {
		t.Fatalf("dropped count mismatch: got=%d want=0", stats.DroppedCount)
	}
}

func TestAsyncStorageClearDropsBufferedEntries(t *testing.T) {
	mock := &slowMockStorage{delay: 50 * time.Millisecond}
	as := NewAsyncStorage(mock, 16, 16, time.Second)
	defer as.Close()

	entries := make([]*models.LogEntry, 0, 8)
	for i := 0; i < 8; i++ {
		entries = append(entries, models.NewLogEntry())
	}

	if err := as.SaveBatch(entries); err != nil {
		t.Fatalf("SaveBatch returned error: %v", err)
	}

	if err := as.Clear(); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if got := mock.SavedCount(); got != 0 {
		t.Fatalf("saved count mismatch after clear: got=%d want=0", got)
	}

	stats := as.GetStats()
	if stats.BufferedCount != 0 {
		t.Fatalf("buffered count mismatch after clear: got=%d want=0", stats.BufferedCount)
	}
}
