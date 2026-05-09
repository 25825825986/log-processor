# 轻量级高并发日志数据处理系统

基于 Go 语言开发的轻量级高并发日志数据处理系统，支持实时日志接收、解析、清洗、存储和导出。

## 特性

- **高并发处理**：基于 Goroutine + Channel，单机可处理数万 QPS
- **可视化配置**：Web 界面实时配置解析规则、并发参数和告警阈值
- **多源接收**：支持 TCP、UDP、HTTP 协议及文件导入
- **智能解析**：自动识别 Nginx、Apache、JSON、CSV、Syslog 等格式
- **数据导出**：支持 Excel、CSV、JSON 格式
- **轻量级存储**：基于 SQLite，支持数据保留策略和自动压缩
- **溢出保护**：内存队列满时自动溢写到磁盘，防止数据丢失
- **配置热更新**：通过 Web 或 API 修改后即时生效

## 快速开始

```bash
# 安装 Go 1.21+，克隆项目并下载依赖
go mod download

# 启动服务（基础配置）
go run cmd/server/main.go -config ./config.example.json

# 或高性能配置
go run cmd/server/main.go -config ./config.optimized.json
```

访问 http://localhost:8080 打开 Web 界面。

**生成测试数据：**

```bash
cd example/tools
python generate_test_logs.py -n 10000 -f nginx -o ../../temp/test.log
```

更多工具见 [example/README.md](example/README.md)。

## 系统架构

```
日志数据源 -> 接收器层 -> 解析器层 -> 处理器层 -> 存储层 -> API/Web层
```

## 配置说明

### 默认端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Web 界面 | 8080 | 可视化配置和数据查询 |
| TCP 接收器 | 9000 | 接收 TCP 日志流 |
| UDP 接收器 | 9001 | 接收 UDP 日志流 |
| HTTP 接收器 | 9002 | 接收 HTTP POST 日志 |

### HTTP 接收器安全配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `http_auth_token` | 访问认证 Token | 空 |
| `http_allowed_ips` | IP 白名单 | 空 |
| `http_max_body_size` | 最大请求体 | 10MB |
| `http_rate_limit` | 每 IP 每分钟限制 | 0 |

### 配置文件示例

```json
{
  "server": { "host": "0.0.0.0", "port": 8080 },
  "parser": { "format": "auto" },
  "processor": {
    "worker_count": 20, "batch_size": 2000, "batch_timeout": 500,
    "overflow_enabled": true, "overflow_dir": "./data/overflow",
    "overflow_max_disk_mb": 512, "clean_rules": [], "filter_rules": []
  },
  "alert": { "slow_threshold": 1000, "error_rate_threshold": 5 },
  "display": { "page_size": 20, "refresh_interval": 30 },
  "import": { "concurrency": 1, "max_lines": 100000 },
  "storage": { "type": "sqlite", "db_path": "./data/logs.db", "retention_hours": 168 },
  "receiver": {
    "tcp_enabled": true, "tcp_port": 9000,
    "udp_enabled": true, "udp_port": 9001,
    "http_enabled": true, "http_port": 9002,
    "buffer_size": 8192
  }
}
```

## API 接口

### 配置与状态
- `GET /api/config` - 获取配置
- `POST /api/config` - 更新配置（热生效）
- `GET /api/status` - 系统状态
- `GET /metrics` - Prometheus 指标

### 日志查询与管理
- `GET /api/logs` - 查询日志（支持时间、方法、路径、状态码、关键字等筛选）
- `POST /api/logs/import` - 导入日志文件
- `GET /api/logs/import/progress?id={id}` - 导入进度
- `DELETE /api/logs/:id` - 删除单条日志
- `DELETE /api/logs` - 清空所有日志

### 统计与导出
- `GET /api/statistics` - 统计数据
- `GET /api/export/formats` - 支持的导出格式
- `POST /api/export` - 导出数据

### 接收器与存储
- `POST /api/receiver/start` - 启动接收器
- `POST /api/receiver/stop` - 停止接收器
- `GET /api/storage/info` - 存储信息
- `POST /api/storage/compact` - 压缩数据库

### 压测
- `POST /api/benchmark/run` - 运行压测
- `GET /api/benchmark/report` - 压测报告

## 日志发送示例

### TCP
```bash
echo '127.0.0.1 - - [04/Mar/2024:10:30:00 +0800] "GET /api/users HTTP/1.1" 200 1234' | nc localhost 9000
```

### UDP
```bash
echo '<14>Mar  4 10:30:00 server app[1234]: {"level":"info","msg":"user login"}' | nc -u localhost 9001
```

### HTTP
```bash
curl -X POST http://localhost:9002/logs \
  -H "Content-Type: text/plain" \
  -d '127.0.0.1 - - [04/Mar/2024:10:30:00 +0800] "POST /api/login HTTP/1.1" 200 256'
```

## 性能指标

| 指标 | 同步模式 | 异步模式 (v2.0) |
|------|----------|-----------------|
| 持续吞吐量 | ~300 QPS | **1,500+ QPS** |
| 突发吞吐量 | 8,000 QPS | **20,000+ QPS** |
| 写入延迟 | 3-5ms | < 1ms |
| 内存缓冲 | 100,000条 | 200,000条 |

- 单节点处理能力: **1,500+ QPS 持续 / 20,000+ QPS 突发**
- 内存占用: < 200MB (默认配置)

## 项目结构

```
Log_processor/
├── cmd/server/main.go         # 主程序入口
├── internal/                  # 内部包（config/exporter/models/parser/processor/receiver/server/storage）
├── web/                       # 前端静态资源
├── example/                   # 测试工具和数据
├── data/                      # 运行时数据（数据库、溢写文件）
├── logs/                      # 应用日志
├── exports/                   # 数据导出目录
├── config.example.json        # 基础配置
└── config.optimized.json      # 高性能配置
```

## 存储占用说明

| 目录 | 用途 | 可清理 |
|------|------|--------|
| `data/` | SQLite 数据库和溢写文件 | 生产数据，谨慎清理 |
| `logs/` | 应用运行日志 | 建议保留最近 2 个文件 |
| `exports/` | 导出的报表文件 | 可随时清理 |
| `temp/` | 上传文件临时缓存 | 可随时清理 |

## 开发计划

- [x] 格式自动识别、磁盘溢写、配置热更新、数据保留、导入进度、实时告警
- [ ] 支持分布式部署
- [ ] 添加更多告警通道（邮件、Webhook）
- [ ] 支持日志压缩存储

## 许可证

MIT License
