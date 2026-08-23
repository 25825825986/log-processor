// test_mysql_connection.go - MySQL 连接测试工具
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Storage struct {
		Type  string `json:"type"`
		MySQL struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
			Database string `json:"database"`
		} `json:"mysql"`
	} `json:"storage"`
}

func main() {
	configFile := flag.String("config", "config.mysql.json", "Config file path")
	flag.Parse()

	fmt.Println("=== MySQL 连接测试工具 ===\n")

	// 1. 读取配置文件
	fmt.Printf("📄 读取配置文件: %s\n", *configFile)
	data, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("❌ 读取配置文件失败: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("❌ 解析配置文件失败: %v", err)
	}

	// 2. 构建 DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.Storage.MySQL.User,
		cfg.Storage.MySQL.Password,
		cfg.Storage.MySQL.Host,
		cfg.Storage.MySQL.Port,
		cfg.Storage.MySQL.Database,
	)

	fmt.Printf("🔗 连接信息:\n")
	fmt.Printf("   Host: %s:%d\n", cfg.Storage.MySQL.Host, cfg.Storage.MySQL.Port)
	fmt.Printf("   User: %s\n", cfg.Storage.MySQL.User)
	fmt.Printf("   Database: %s\n\n", cfg.Storage.MySQL.Database)

	// 3. 连接数据库
	fmt.Println("🔌 连接 MySQL...")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer db.Close()

	// 4. Ping 测试
	fmt.Println("🏓 测试连接...")
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Ping 失败: %v", err)
	}
	fmt.Println("✅ 连接成功！\n")

	// 5. 检查数据库
	fmt.Println("📊 检查数据库状态...")
	var dbName string
	err = db.QueryRow("SELECT DATABASE()").Scan(&dbName)
	if err != nil {
		log.Fatalf("❌ 查询数据库失败: %v", err)
	}
	fmt.Printf("✅ 当前数据库: %s\n\n", dbName)

	// 6. 检查表
	fmt.Println("📋 检查表...")
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		log.Fatalf("❌ 查询表失败: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		rows.Scan(&table)
		tables = append(tables, table)
	}

	if len(tables) == 0 {
		fmt.Println("⚠️  数据库为空，未找到任何表")
		fmt.Println("   请先执行初始化脚本: mysql -u root -p < scripts/init_mysql.sql")
		return
	}

	fmt.Printf("✅ 找到 %d 个表:\n", len(tables))
	for _, table := range tables {
		fmt.Printf("   - %s\n", table)
	}
	fmt.Println()

	// 7. 检查 logs 表
	fmt.Println("📝 检查 logs 表...")
	var count int64
	err = db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
	if err != nil {
		fmt.Printf("⚠️  logs 表不存在或无法访问: %v\n", err)
		return
	}

	fmt.Printf("✅ logs 表记录数: %d 条\n\n", count)

	if count > 0 {
		// 8. 显示最新的 5 条记录
		fmt.Println("📄 最新的 5 条日志:")
		rows, err = db.Query(`SELECT id, timestamp, method, path, status_code, client_ip 
			FROM logs ORDER BY timestamp DESC LIMIT 5`)
		if err != nil {
			fmt.Printf("⚠️  查询日志失败: %v\n", err)
			return
		}
		defer rows.Close()

		fmt.Println("   ID                                   | 时间                | 方法   | 路径              | 状态码 | IP")
		fmt.Println("   " + strings.Repeat("=", 120))

		for rows.Next() {
			var id, method, path, clientIP string
			var timestamp string
			var statusCode int
			rows.Scan(&id, &timestamp, &method, &path, &statusCode, &clientIP)
			fmt.Printf("   %-36s | %-19s | %-6s | %-17s | %-6d | %s\n",
				id, timestamp, method, path, statusCode, clientIP)
		}
		fmt.Println()
	}

	// 9. 检查索引
	fmt.Println("🔍 检查索引...")
	rows, err = db.Query("SHOW INDEX FROM logs")
	if err != nil {
		fmt.Printf("⚠️  查询索引失败: %v\n", err)
		return
	}
	defer rows.Close()

	var indexCount int
	indexes := make(map[string]bool)
	for rows.Next() {
		var table, nonUnique, keyName, seqInIndex, columnName, collation, cardinality, subPart, packed, null, indexType, comment, indexComment, visible, expression string
		rows.Scan(&table, &nonUnique, &keyName, &seqInIndex, &columnName, &collation, &cardinality, &subPart, &packed, &null, &indexType, &comment, &indexComment, &visible, &expression)
		if !indexes[keyName] {
			indexes[keyName] = true
			indexCount++
		}
	}
	fmt.Printf("✅ 索引数量: %d 个\n\n", indexCount)

	// 10. 检查表大小
	fmt.Println("💾 检查表大小...")
	var dataLength, indexLength int64
	err = db.QueryRow(`SELECT data_length, index_length 
		FROM information_schema.TABLES 
		WHERE table_schema = ? AND table_name = 'logs'`,
		cfg.Storage.MySQL.Database).Scan(&dataLength, &indexLength)
	if err == nil {
		totalSize := dataLength + indexLength
		fmt.Printf("   数据大小: %.2f MB\n", float64(dataLength)/1024/1024)
		fmt.Printf("   索引大小: %.2f MB\n", float64(indexLength)/1024/1024)
		fmt.Printf("   总大小:   %.2f MB\n\n", float64(totalSize)/1024/1024)
	}

	// 11. 总结
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("🎉 MySQL 数据库检查完成！")
	fmt.Println("✅ 连接正常")
	fmt.Printf("✅ 数据库: %s\n", dbName)
	fmt.Printf("✅ 表数量: %d\n", len(tables))
	fmt.Printf("✅ 日志数: %d 条\n", count)
	fmt.Println(strings.Repeat("=", 50))
}
