// storage/storage.go - 数据存储接口
package storage

import (
	"fmt"
	"log-processor/internal/config"
	"log-processor/internal/models"
)

// Storage 存储接口
type Storage interface {
	SaveBatch(entries []*models.LogEntry) error
	Query(filter models.FilterCondition, limit, offset int) ([]*models.LogEntry, error)
	Count(filter models.FilterCondition) (int64, error)
	Statistics(filter models.FilterCondition) (*models.Statistics, error)
	Delete(id string) error
	Clear() error
	Close() error
}

// NewStorage 创建存储实例（仅支持 MySQL）
func NewStorage(cfg config.StorageConfig) (Storage, error) {
	if cfg.Type != "mysql" && cfg.Type != "" {
		return nil, fmt.Errorf("unsupported storage type: %s (only 'mysql' is supported)", cfg.Type)
	}
	
	return NewMySQLStorage(cfg)
}
