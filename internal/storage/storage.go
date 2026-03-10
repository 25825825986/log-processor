// storage/storage.go - 数据存储
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/models"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

// SQLiteStorage SQLite存储实现
type SQLiteStorage struct {
	db     *sql.DB
	config config.StorageConfig
	mu     sync.RWMutex
}

// NewSQLiteStorage 创建SQLite存储
func NewSQLiteStorage(cfg config.StorageConfig) (*SQLiteStorage, error) {
	// 确保目录存在
	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// 高性能 SQLite 配置
	db, err := sql.Open("sqlite3", cfg.DBPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-64000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	s := &SQLiteStorage{
		db:     db,
		config: cfg,
	}

	if err := s.initTable(); err != nil {
		return nil, err
	}

	// 启动清理协程
	go s.cleanupRoutine()

	return s, nil
}

// initTable 初始化表结构
func (s *SQLiteStorage) initTable() error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS logs (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		source TEXT,
		level TEXT,
		method TEXT,
		path TEXT,
		status_code INTEGER,
		response_time INTEGER,
		client_ip TEXT,
		user_agent TEXT,
		referer TEXT,
		request_size INTEGER,
		response_size INTEGER,
		extra_fields TEXT,
		raw_data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_timestamp ON logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_status_code ON logs(status_code);
	CREATE INDEX IF NOT EXISTS idx_method ON logs(method);
	CREATE INDEX IF NOT EXISTS idx_path ON logs(path);
	CREATE INDEX IF NOT EXISTS idx_client_ip ON logs(client_ip);
	CREATE INDEX IF NOT EXISTS idx_source ON logs(source);
	CREATE INDEX IF NOT EXISTS idx_level ON logs(level);
	`

	_, err := s.db.Exec(createTableSQL)
	return err
}

// SaveBatch 批量保存
func (s *SQLiteStorage) SaveBatch(entries []*models.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// 注意：SQLite 使用 WAL 模式支持并发读，不需要全局锁
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO logs 
		(id, timestamp, source, level, method, path, status_code, response_time, 
		 client_ip, user_agent, referer, request_size, response_size, extra_fields, raw_data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entry := range entries {
		extraFields, _ := json.Marshal(entry.ExtraFields)
		_, err := stmt.Exec(
			entry.ID,
			entry.Timestamp,
			entry.Source,
			entry.Level,
			entry.Method,
			entry.Path,
			entry.StatusCode,
			entry.ResponseTime,
			entry.ClientIP,
			entry.UserAgent,
			entry.Referer,
			entry.RequestSize,
			entry.ResponseSize,
			string(extraFields),
			entry.RawData,
			entry.CreatedAt,
		)
		if err != nil {
			log.Printf("Failed to insert log: %v", err)
		}
	}

	return tx.Commit()
}

// Query 查询日志
func (s *SQLiteStorage) Query(filter models.FilterCondition, limit, offset int) ([]*models.LogEntry, error) {
	where, args := s.buildWhereClause(filter)

	query := fmt.Sprintf(`SELECT id, timestamp, source, level, method, path, status_code, 
		response_time, client_ip, user_agent, referer, request_size, response_size, 
		extra_fields, raw_data, created_at 
		FROM logs %s ORDER BY timestamp DESC LIMIT ? OFFSET ?`, where)

	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

// Count 统计数量
func (s *SQLiteStorage) Count(filter models.FilterCondition) (int64, error) {
	where, args := s.buildWhereClause(filter)

	query := fmt.Sprintf("SELECT COUNT(*) FROM logs %s", where)

	var count int64
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// Statistics 统计分析
func (s *SQLiteStorage) Statistics(filter models.FilterCondition) (*models.Statistics, error) {
	stats := &models.Statistics{
		StatusCodeDist: make(map[int]int64),
		MethodDist:     make(map[string]int64),
	}

	where, args := s.buildWhereClause(filter)

	// 总数量
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM logs %s", where)
	if err := s.db.QueryRow(countQuery, args...).Scan(&stats.TotalCount); err != nil {
		return nil, err
	}

	// 错误数量 (status_code >= 400)
	var errorQuery string
	if where == "" {
		errorQuery = "SELECT COUNT(*) FROM logs WHERE status_code >= 400"
	} else {
		errorQuery = fmt.Sprintf("SELECT COUNT(*) FROM logs %s AND status_code >= 400", where)
	}
	if err := s.db.QueryRow(errorQuery, args...).Scan(&stats.ErrorCount); err != nil {
		return nil, err
	}

	// 平均响应时间
	avgQuery := fmt.Sprintf("SELECT AVG(response_time) FROM logs %s", where)
	if err := s.db.QueryRow(avgQuery, args...).Scan(&stats.AvgResponseTime); err != nil {
		stats.AvgResponseTime = 0
	}

	// 状态码分布
	statusQuery := fmt.Sprintf("SELECT status_code, COUNT(*) FROM logs %s GROUP BY status_code", where)
	rows, err := s.db.Query(statusQuery, args...)
	if err == nil {
		for rows.Next() {
			var code int
			var count int64
			if err := rows.Scan(&code, &count); err == nil {
				stats.StatusCodeDist[code] = count
			}
		}
		rows.Close()
	}

	// 方法分布
	methodQuery := fmt.Sprintf("SELECT method, COUNT(*) FROM logs %s GROUP BY method", where)
	rows, err = s.db.Query(methodQuery, args...)
	if err == nil {
		for rows.Next() {
			var method string
			var count int64
			if err := rows.Scan(&method, &count); err == nil {
				stats.MethodDist[method] = count
			}
		}
		rows.Close()
	}

	// Top路径
	topPathQuery := fmt.Sprintf("SELECT path, COUNT(*) as cnt FROM logs %s GROUP BY path ORDER BY cnt DESC LIMIT 10", where)
	rows, err = s.db.Query(topPathQuery, args...)
	if err == nil {
		for rows.Next() {
			var stat models.PathStat
			if err := rows.Scan(&stat.Path, &stat.Count); err == nil {
				stats.TopPaths = append(stats.TopPaths, stat)
			}
		}
		rows.Close()
	}

	// 时间序列（按 5 分钟区间）
	timeQuery := fmt.Sprintf(`SELECT strftime('%%Y-%%m-%%d %%H:%%M', timestamp) as time_bucket, COUNT(*) 
		FROM logs %s GROUP BY time_bucket ORDER BY time_bucket LIMIT 50`, where)
	rows, err = s.db.Query(timeQuery, args...)
	if err == nil {
		for rows.Next() {
			var point models.TimePoint
			if err := rows.Scan(&point.Time, &point.Count); err == nil {
				stats.TimeSeries = append(stats.TimeSeries, point)
			}
		}
		rows.Close()
	}

	return stats, nil
}

// buildWhereClause 构建WHERE子句
func (s *SQLiteStorage) buildWhereClause(filter models.FilterCondition) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.EndTime)
	}
	if len(filter.Methods) > 0 {
		placeholders := make([]string, len(filter.Methods))
		for i := range filter.Methods {
			placeholders[i] = "?"
			args = append(args, filter.Methods[i])
		}
		conditions = append(conditions, fmt.Sprintf("method IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(filter.Paths) > 0 {
		placeholders := make([]string, len(filter.Paths))
		for i := range filter.Paths {
			placeholders[i] = "?"
			args = append(args, filter.Paths[i])
		}
		conditions = append(conditions, fmt.Sprintf("path IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(filter.StatusCodes) > 0 {
		placeholders := make([]string, len(filter.StatusCodes))
		for i := range filter.StatusCodes {
			placeholders[i] = "?"
			args = append(args, filter.StatusCodes[i])
		}
		conditions = append(conditions, fmt.Sprintf("status_code IN (%s)", strings.Join(placeholders, ",")))
	}
	if filter.Level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, filter.Level)
	}
	if filter.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, filter.Source)
	}
	if filter.Keyword != "" {
		conditions = append(conditions, "(raw_data LIKE ? OR path LIKE ? OR client_ip LIKE ?)")
		keyword := "%" + filter.Keyword + "%"
		args = append(args, keyword, keyword, keyword)
	}

	if len(conditions) > 0 {
		return "WHERE " + strings.Join(conditions, " AND "), args
	}
	return "", args
}

// scanRows 扫描行
func (s *SQLiteStorage) scanRows(rows *sql.Rows) ([]*models.LogEntry, error) {
	var entries []*models.LogEntry

	for rows.Next() {
		entry := &models.LogEntry{ExtraFields: make(map[string]string)}
		var extraFieldsStr string

		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.Source,
			&entry.Level,
			&entry.Method,
			&entry.Path,
			&entry.StatusCode,
			&entry.ResponseTime,
			&entry.ClientIP,
			&entry.UserAgent,
			&entry.Referer,
			&entry.RequestSize,
			&entry.ResponseSize,
			&extraFieldsStr,
			&entry.RawData,
			&entry.CreatedAt,
		)
		if err != nil {
			continue
		}

		// 解析extra_fields
		if extraFieldsStr != "" {
			json.Unmarshal([]byte(extraFieldsStr), &entry.ExtraFields)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// cleanupRoutine 清理协程
func (s *SQLiteStorage) cleanupRoutine() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup 清理过期数据
func (s *SQLiteStorage) cleanup() {
	if s.config.RetentionHours <= 0 {
		return
	}

	cutoff := time.Now().Add(-time.Duration(s.config.RetentionHours) * time.Hour)
	_, err := s.db.Exec("DELETE FROM logs WHERE timestamp < ?", cutoff)
	if err != nil {
		log.Printf("Cleanup failed: %v", err)
	}

	// 优化数据库
	s.db.Exec("VACUUM")
}

// Delete 删除单条日志
func (s *SQLiteStorage) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM logs WHERE id = ?", id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("log not found: %s", id)
	}

	return nil
}

// Clear 清空所有日志
func (s *SQLiteStorage) Clear() error {
	_, err := s.db.Exec("DELETE FROM logs")
	if err != nil {
		return err
	}

	// 优化数据库
	_, err = s.db.Exec("VACUUM")
	return err
}

// Close 关闭存储
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}


