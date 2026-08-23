# MySQL 数据库迁移指南

## 快速开始

### 1. 安装 MySQL

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install mysql-server
sudo mysql_secure_installation
```

**CentOS/RHEL:**
```bash
sudo yum install mysql-server
sudo systemctl start mysqld
sudo mysql_secure_installation
```

**macOS:**
```bash
brew install mysql
brew services start mysql
```

**Windows:**
下载并安装 MySQL Community Server: https://dev.mysql.com/downloads/mysql/

### 2. 初始化数据库

```bash
# 登录 MySQL
mysql -u root -p

# 执行初始化脚本
mysql -u root -p < scripts/init_mysql.sql
```

或者手动创建：

```sql
CREATE DATABASE log_processor DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'log_processor'@'localhost' IDENTIFIED BY 'your_secure_password';
GRANT ALL PRIVILEGES ON log_processor.* TO 'log_processor'@'localhost';
FLUSH PRIVILEGES;
```

### 3. 配置系统使用 MySQL

编辑配置文件（或使用 `config.mysql.json`）：

```json
{
  "storage": {
    "type": "mysql",
    "retention_hours": 168,
    "mysql": {
      "host": "localhost",
      "port": 3306,
      "user": "log_processor",
      "password": "your_secure_password",
      "database": "log_processor",
      "max_open_conns": 100,
      "max_idle_conns": 10,
      "conn_max_lifetime": 60
    }
  }
}
```

### 4. 启动系统

```bash
# 使用 MySQL 配置启动
./bin/log-processor -config config.mysql.json

# 或使用环境变量
export DB_TYPE=mysql
export DB_HOST=localhost
export DB_USER=log_processor
export DB_PASSWORD=your_secure_password
./bin/log-processor
```

## 从 SQLite 迁移到 MySQL

### 方法一：使用迁移脚本（推荐）

```bash
# 1. 编译迁移工具
cd scripts
go build -o migrate migrate_sqlite_to_mysql.go

# 2. 执行迁移
./migrate \
  -sqlite ../data/logs.db \
  -mysql "log_processor:password@tcp(localhost:3306)/log_processor?parseTime=true" \
  -batch 1000

# 3. 验证数据
mysql -u log_processor -p log_processor -e "SELECT COUNT(*) FROM logs;"
```

### 方法二：手动导出导入

```bash
# 1. 从 SQLite 导出为 CSV
sqlite3 data/logs.db <<EOF
.headers on
.mode csv
.output /tmp/logs.csv
SELECT * FROM logs;
.quit
EOF

# 2. 导入到 MySQL
mysql -u log_processor -p log_processor <<EOF
LOAD DATA LOCAL INFILE '/tmp/logs.csv'
INTO TABLE logs
FIELDS TERMINATED BY ','
ENCLOSED BY '"'
LINES TERMINATED BY '\n'
IGNORE 1 ROWS;
EOF
```

## 性能优化

### MySQL 配置优化

编辑 `/etc/mysql/my.cnf` 或 `/etc/my.cnf`：

```ini
[mysqld]
# InnoDB 设置
innodb_buffer_pool_size = 2G          # 设置为物理内存的 50-70%
innodb_log_file_size = 256M
innodb_flush_log_at_trx_commit = 2    # 性能优化，可能损失1秒数据
innodb_flush_method = O_DIRECT

# 连接设置
max_connections = 500
max_connect_errors = 1000

# 查询缓存
query_cache_size = 64M
query_cache_type = 1

# 慢查询日志
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow-query.log
long_query_time = 2
```

重启 MySQL：

```bash
sudo systemctl restart mysql
```

### 索引优化

```sql
-- 查看索引使用情况
SHOW INDEX FROM logs;

-- 分析查询计划
EXPLAIN SELECT * FROM logs WHERE project_id = 'xxx' AND environment = 'prod';

-- 添加复合索引（根据实际查询需求）
CREATE INDEX idx_project_time ON logs(project_id, timestamp);
CREATE INDEX idx_error_time ON logs(level, timestamp) WHERE level IN ('ERROR', 'FATAL');
```

### 定期维护

```sql
-- 分析表
ANALYZE TABLE logs;

-- 优化表
OPTIMIZE TABLE logs;

-- 查看表大小
SELECT 
    table_name AS 'Table',
    ROUND(((data_length + index_length) / 1024 / 1024), 2) AS 'Size (MB)'
FROM information_schema.TABLES
WHERE table_schema = 'log_processor'
ORDER BY (data_length + index_length) DESC;
```

## 备份与恢复

### 备份

```bash
# 完整备份
mysqldump -u log_processor -p log_processor > backup_$(date +%Y%m%d).sql

# 仅备份表结构
mysqldump -u log_processor -p --no-data log_processor > schema.sql

# 仅备份数据
mysqldump -u log_processor -p --no-create-info log_processor > data.sql

# 压缩备份
mysqldump -u log_processor -p log_processor | gzip > backup_$(date +%Y%m%d).sql.gz
```

### 恢复

```bash
# 恢复完整备份
mysql -u log_processor -p log_processor < backup_20240304.sql

# 从压缩文件恢复
gunzip < backup_20240304.sql.gz | mysql -u log_processor -p log_processor
```

### 自动备份脚本

创建 `/etc/cron.daily/mysql-backup`：

```bash
#!/bin/bash
BACKUP_DIR="/var/backups/mysql"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

mysqldump -u log_processor -pYOUR_PASSWORD log_processor | gzip > $BACKUP_DIR/log_processor_$DATE.sql.gz

# 保留最近7天的备份
find $BACKUP_DIR -name "log_processor_*.sql.gz" -mtime +7 -delete
```

## 监控与告警

### 连接数监控

```sql
-- 查看当前连接数
SHOW STATUS LIKE 'Threads_connected';

-- 查看最大连接数
SHOW VARIABLES LIKE 'max_connections';

-- 查看活跃连接
SHOW PROCESSLIST;
```

### 性能监控

```sql
-- 查看慢查询数量
SHOW STATUS LIKE 'Slow_queries';

-- 查看 InnoDB 状态
SHOW ENGINE INNODB STATUS;

-- 查看表锁等待
SHOW STATUS LIKE 'Table_locks_waited';
```

## 故障排查

### 常见问题

**1. 连接失败**

```bash
# 检查 MySQL 是否运行
sudo systemctl status mysql

# 检查端口监听
netstat -tulnp | grep 3306

# 检查防火墙
sudo ufw allow 3306/tcp
```

**2. 权限问题**

```sql
-- 重新授权
GRANT ALL PRIVILEGES ON log_processor.* TO 'log_processor'@'localhost';
FLUSH PRIVILEGES;

-- 查看用户权限
SHOW GRANTS FOR 'log_processor'@'localhost';
```

**3. 性能慢**

```sql
-- 查看慢查询
SELECT * FROM mysql.slow_log ORDER BY start_time DESC LIMIT 10;

-- 查看表锁
SHOW OPEN TABLES WHERE In_use > 0;

-- 优化表
OPTIMIZE TABLE logs;
```

## SQLite vs MySQL 对比

| 特性 | SQLite | MySQL |
|------|--------|-------|
| 并发写入 | 低（单写锁） | 高（行级锁） |
| 适合规模 | < 10GB | > 10GB |
| 查询性能 | 简单查询快 | 复杂查询快 |
| 网络访问 | 不支持 | 支持 |
| 资源占用 | 极低 | 中等 |
| 运维成本 | 零 | 需要维护 |
| 推荐场景 | 单机、轻量 | 多机、大规模 |

## 建议

- **日志量 < 1万条/天**：使用 SQLite
- **日志量 1万-10万条/天**：SQLite 或 MySQL 均可
- **日志量 > 10万条/天**：推荐 MySQL
- **多服务器部署**：必须使用 MySQL
- **需要 AI 分析**：推荐 MySQL（更好的并发支持）

## 下一步

完成 MySQL 迁移后，可以继续实施：

1. ✅ 多项目隔离管理
2. ✅ AI 错误分析引擎
3. ✅ 问题跟踪系统
4. ✅ 高级监控告警

参考主项目 README.md 获取更多信息。
