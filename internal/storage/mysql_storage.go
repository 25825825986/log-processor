// storage/mysql_storage.go - MySQL存储实现
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log-processor/internal/config"
	"log-processor/internal/models"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLStorage MySQL存储实现
type MySQLStorage struct {
	db     *sql.DB
	config config.StorageConfig
	mu     sync.RWMutex
}

// NewMySQLStorage 创建MySQL存储
func NewMySQLStorage(cfg config.StorageConfig) (*MySQLStorage, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		cfg.MySQL.User,
		cfg.MySQL.Password,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.MySQL.ConnMaxLifetime) * time.Minute)

	s := &MySQLStorage{
		db:     db,
		config: cfg,
	}

	if err := s.initTable(); err != nil {
		return nil, err
	}

	go s.cleanupRoutine()

	return s, nil
}

// initTable 初始化表结构
func (s *MySQLStorage) initTable() error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS logs (
		id VARCHAR(36) PRIMARY KEY COMMENT '日志ID',
		timestamp DATETIME(3) NOT NULL COMMENT '日志时间',
		
		project_id VARCHAR(64) DEFAULT '' COMMENT '项目ID',
		project_name VARCHAR(128) DEFAULT '' COMMENT '项目名称',
		environment VARCHAR(32) DEFAULT 'prod' COMMENT '环境',
		service_name VARCHAR(128) DEFAULT '' COMMENT '服务名称',
		
		source VARCHAR(64) DEFAULT '' COMMENT '日志来源',
		level VARCHAR(16) DEFAULT 'INFO' COMMENT '日志级别',
		log_type VARCHAR(32) DEFAULT '' COMMENT '日志类型',
		
		method VARCHAR(16) DEFAULT '' COMMENT 'HTTP方法',
		path TEXT COMMENT '请求路径',
		status_code INT DEFAULT 0 COMMENT 'HTTP状态码',
		response_time BIGINT DEFAULT 0 COMMENT '响应时间(ms)',
		client_ip VARCHAR(64) DEFAULT '' COMMENT '客户端IP',
		user_agent TEXT COMMENT 'User-Agent',
		referer TEXT COMMENT 'Referer',
		request_size BIGINT DEFAULT 0 COMMENT '请求大小',
		response_size BIGINT DEFAULT 0 COMMENT '响应大小',
		
		error_message TEXT COMMENT '错误信息',
		error_code VARCHAR(64) DEFAULT '' COMMENT '错误码',
		stack_trace TEXT COMMENT '堆栈跟踪',
		
		request_id VARCHAR(64) DEFAULT '' COMMENT '请求ID',
		user_id VARCHAR(64) DEFAULT '' COMMENT '用户ID',
		session_id VARCHAR(64) DEFAULT '' COMMENT '会话ID',
		
		ai_analyzed BOOLEAN DEFAULT FALSE COMMENT '是否已AI分析',
		ai_analysis TEXT COMMENT 'AI分析结果',
		ai_suggestions TEXT COMMENT 'AI建议',
		ai_analyzed_at DATETIME(3) NULL COMMENT 'AI分析时间',
		
		extra_fields JSON COMMENT '额外字段',
		raw_data TEXT COMMENT '原始日志',
		created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
		
		INDEX idx_timestamp (timestamp),
		INDEX idx_project_env (project_id, environment),
		INDEX idx_status_code (status_code),
		INDEX idx_level (level),
		INDEX idx_log_type (log_type),
		INDEX idx_method (method),
		INDEX idx_client_ip (client_ip),
		INDEX idx_ai_analyzed (ai_analyzed, level),
		INDEX idx_created_at (created_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='日志表';`

	_, err := s.db.Exec(createTableSQL)
	return err
}

// SaveBatch 批量保存
func (s *MySQLStorage) SaveBatch(entries []*models.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT IGNORE INTO logs 
		(id, timestamp, project_id, project_name, environment, service_name,
		 source, level, log_type, method, path, status_code, response_time,
		 client_ip, user_agent, referer, request_size, response_size,
		 error_message, error_code, stack_trace, request_id, user_id, session_id,
		 ai_analyzed, ai_analysis, ai_suggestions, ai_analyzed_at,
		 extra_fields, raw_data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var failCount int
	for _, entry := range entries {
		extraFields, _ := json.Marshal(entry.ExtraFields)
		
		_, err := stmt.Exec(
			entry.ID, entry.Timestamp, entry.ProjectID, entry.ProjectName,
			entry.Environment, entry.ServiceName, entry.Source, entry.Level,
			entry.LogType, entry.Method, entry.Path, entry.StatusCode,
			entry.ResponseTime, entry.ClientIP, entry.UserAgent, entry.Referer,
			entry.RequestSize, entry.ResponseSize, entry.ErrorMessage,
			entry.ErrorCode, entry.StackTrace, entry.RequestID, entry.UserID,
			entry.SessionID, entry.AIAnalyzed, entry.AIAnalysis,
			entry.AISuggestions, entry.AIAnalyzedAt, string(extraFields),
			entry.RawData, entry.CreatedAt,
		)
		if err != nil {
			failCount++
			if failCount <= 3 {
				log.Printf("Failed to insert log: %v", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("batch commit failed (%d entries): %w", len(entries), err)
	}

	if failCount > 0 {
		log.Printf("[WARN] SaveBatch: %d/%d entries failed", failCount, len(entries))
	}
	return nil
}

// Query 查询日志
func (s *MySQLStorage) Query(filter models.FilterCondition, limit, offset int) ([]*models.LogEntry, error) {
	where, args := s.buildWhereClause(filter)

	query := fmt.Sprintf(`SELECT id, timestamp, project_id, project_name, environment, service_name,
		source, level, log_type, method, path, status_code, response_time,
		client_ip, user_agent, referer, request_size, response_size,
		error_message, error_code, stack_trace, request_id, user_id, session_id,
		ai_analyzed, ai_analysis, ai_suggestions, ai_analyzed_at,
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
func (s *MySQLStorage) Count(filter models.FilterCondition) (int64, error) {
	where, args := s.buildWhereClause(filter)
	query := fmt.Sprintf("SELECT COUNT(*) FROM logs %s", where)
	var count int64
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// Statistics 统计分析
func (s *MySQLStorage) Statistics(filter models.FilterCondition) (*models.Statistics, error) {
	stats := &models.Statistics{
		StatusCodeDist: make(map[int]int64),
		MethodDist:     make(map[string]int64),
	}

	where, args := s.buildWhereClause(filter)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM logs %s", where)
	if err := s.db.QueryRow(countQuery, args...).Scan(&stats.TotalCount); err != nil {
		return nil, err
	}

	var errorQuery string
	if where == "" {
		errorQuery = "SELECT COUNT(*) FROM logs WHERE status_code >= 400"
	} else {
		errorQuery = fmt.Sprintf("SELECT COUNT(*) FROM logs %s AND status_code >= 400", where)
	}
	s.db.QueryRow(errorQuery, args...).Scan(&stats.ErrorCount)

	avgQuery := fmt.Sprintf("SELECT COALESCE(AVG(response_time), 0) FROM logs %s", where)
	s.db.QueryRow(avgQuery, args...).Scan(&stats.AvgResponseTime)

	statusQuery := fmt.Sprintf("SELECT status_code, COUNT(*) FROM logs %s GROUP BY status_code", where)
	if rows, err := s.db.Query(statusQuery, args...); err == nil {
		for rows.Next() {
			var code int
			var count int64
			if err := rows.Scan(&code, &count); err == nil {
				stats.StatusCodeDist[code] = count
			}
		}
		rows.Close()
	}

	methodQuery := fmt.Sprintf("SELECT method, COUNT(*) FROM logs %s GROUP BY method", where)
	if rows, err := s.db.Query(methodQuery, args...); err == nil {
		for rows.Next() {
			var method string
			var count int64
			if err := rows.Scan(&method, &count); err == nil {
				stats.MethodDist[method] = count
			}
		}
		rows.Close()
	}

	topPathQuery := fmt.Sprintf(`SELECT path, COUNT(*) as cnt FROM logs %s 
		GROUP BY path ORDER BY cnt DESC LIMIT 10`, where)
	if rows, err := s.db.Query(topPathQuery, args...); err == nil {
		for rows.Next() {
			var stat models.PathStat
			if err := rows.Scan(&stat.Path, &stat.Count); err == nil {
				stats.TopPaths = append(stats.TopPaths, stat)
			}
		}
		rows.Close()
	}

	timeQuery := fmt.Sprintf(`SELECT DATE_FORMAT(timestamp, '%%Y-%%m-%%d %%H:%%i:00') as time_bucket, COUNT(*) 
		FROM logs %s GROUP BY time_bucket ORDER BY time_bucket DESC LIMIT 50`, where)
	if rows, err := s.db.Query(timeQuery, args...); err == nil {
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
func (s *MySQLStorage) buildWhereClause(filter models.FilterCondition) (string, []interface{}) {
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
	if len(filter.StatusCodeRanges) > 0 {
		rangeConditions := make([]string, 0, len(filter.StatusCodeRanges))
		for _, r := range filter.StatusCodeRanges {
			parts := strings.Split(r, "-")
			if len(parts) == 2 {
				minVal, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				maxVal, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil {
					rangeConditions = append(rangeConditions, "(status_code >= ? AND status_code <= ?)")
					args = append(args, minVal, maxVal)
				}
			}
		}
		if len(rangeConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(rangeConditions, " OR ")+")")
		}
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

// CONTINUATION_MARKER_6
