// scripts/migrate_sqlite_to_mysql.go - SQLite 迁移到 MySQL
package main

import (
	"database/sql"
	"flag"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	sqlitePath := flag.String("sqlite", "./data/logs.db", "SQLite database path")
	mysqlDSN := flag.String("mysql", "log_processor:password@tcp(localhost:3306)/log_processor?parseTime=true", "MySQL DSN")
	batchSize := flag.Int("batch", 1000, "Batch size")
	flag.Parse()

	log.Printf("Starting migration: SQLite -> MySQL")
	log.Printf("Batch size: %d", *batchSize)

	sqliteDB, err := sql.Open("sqlite3", *sqlitePath)
	if err != nil {
		log.Fatalf("Failed to open SQLite: %v", err)
	}
	defer sqliteDB.Close()

	mysqlDB, err := sql.Open("mysql", *mysqlDSN)
	if err != nil {
		log.Fatalf("Failed to open MySQL: %v", err)
	}
	defer mysqlDB.Close()

	if err := mysqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping MySQL: %v", err)
	}

	var total int64
	sqliteDB.QueryRow("SELECT COUNT(*) FROM logs").Scan(&total)
	log.Printf("Total logs: %d", total)

	offset := 0
	migrated := 0

	for {
		rows, err := sqliteDB.Query(`SELECT 
			id, timestamp, project_id, project_name, environment, service_name,
			source, level, log_type, method, path, status_code, response_time,
			client_ip, user_agent, referer, request_size, response_size,
			error_message, error_code, stack_trace, request_id, user_id, session_id,
			ai_analyzed, ai_analysis, ai_suggestions, ai_analyzed_at,
			extra_fields, raw_data, created_at
			FROM logs LIMIT ? OFFSET ?`, *batchSize, offset)
		if err != nil {
			log.Fatalf("Query error: %v", err)
		}

		count := 0
		tx, _ := mysqlDB.Begin()
		stmt, _ := tx.Prepare(`INSERT IGNORE INTO logs 
			(id, timestamp, project_id, project_name, environment, service_name,
			 source, level, log_type, method, path, status_code, response_time,
			 client_ip, user_agent, referer, request_size, response_size,
			 error_message, error_code, stack_trace, request_id, user_id, session_id,
			 ai_analyzed, ai_analysis, ai_suggestions, ai_analyzed_at,
			 extra_fields, raw_data, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)

		for rows.Next() {
			var id, projectID, projectName, env, service, source, level, logType string
			var method, path, clientIP, userAgent, referer, errorMsg, errorCode string
			var stackTrace, reqID, userID, sessID, aiAnalysis, aiSugg, extraFields, rawData string
			var timestamp, createdAt time.Time
			var statusCode int
			var responseTime, reqSize, respSize int64
			var aiAnalyzed bool
			var aiAnalyzedAt sql.NullTime

			rows.Scan(&id, &timestamp, &projectID, &projectName, &env, &service,
				&source, &level, &logType, &method, &path, &statusCode, &responseTime,
				&clientIP, &userAgent, &referer, &reqSize, &respSize,
				&errorMsg, &errorCode, &stackTrace, &reqID, &userID, &sessID,
				&aiAnalyzed, &aiAnalysis, &aiSugg, &aiAnalyzedAt,
				&extraFields, &rawData, &createdAt)

			stmt.Exec(id, timestamp, projectID, projectName, env, service,
				source, level, logType, method, path, statusCode, responseTime,
				clientIP, userAgent, referer, reqSize, respSize,
				errorMsg, errorCode, stackTrace, reqID, userID, sessID,
				aiAnalyzed, aiAnalysis, aiSugg, aiAnalyzedAt,
				extraFields, rawData, createdAt)
			count++
			migrated++
		}
		rows.Close()
		stmt.Close()
		tx.Commit()

		if count == 0 {
			break
		}

		offset += *batchSize
		log.Printf("Progress: %d/%d (%.1f%%)", migrated, total, float64(migrated)/float64(total)*100)
	}

	log.Printf("Migration completed! Total: %d", migrated)
}
