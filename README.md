# 轻量级高并发日志数据处理系统

基于 Go 语言开发的轻量级高并发日志数据处理系统，支持实时日志接收、解析、清洗、存储和导出。

## 特性

- 🚀 **高并发处理**：基于 Goroutine + Channel 实现，单机可处理数万 QPS
- 📊 **可视化配置**：Web 界面支持实时配置解析规则、清洗规则和并发参数
- 📡 **多源数据接收**：支持 TCP、UDP、HTTP 协议接收，同时支持文件导入
- 🔍 **灵活的数据查询**：支持按时间、接口、状态码等多维度筛选
- 📤 **数据导出**：支持导出为 Excel、CSV、JSON 格式
- 💾 **轻量级存储**：基于 SQLite，无需外部数据库依赖
- 🔧 **可扩展解析**：支持 Nginx、Apache、JSON、CSV 及自定义格式

## 快速开始

### 1. 安装依赖

```bash
# 安装 Go 1.21+
# https://golang.org/dl/

# 克隆项目
git clone <repository>
cd log-processor

# 下载依赖
go mod download
```

### 2. 启动服务

```bash
go run cmd/server/main.go
```

或使用配置文件：

```bash
go run cmd/server/main.go -config ./config.json
```

### 3. 访问 Web 界面

打开浏览器访问：http://localhost:8080

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      日志数据源                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ TCP端口  │  │ UDP端口  │  │ HTTP端口 │  │ 文件导入 │    │
│  │  :9000   │  │  :9001   │  │  :9002   │  │          │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
└───────┼─────────────┼─────────────┼─────────────┼──────────┘
        │             │             │             │
        └─────────────┴─────────────┴─────────────┘
                          │
                    ┌─────┴─────┐
                    │ 接收器层   │  Receiver (TCP/UDP/HTTP/File)
                    └─────┬─────┘
                          │
                    ┌─────┴─────┐
                    │ 解析器层   │  Parser (Nginx/Apache/JSON/CSV/Custom)
                    └─────┬─────┘
                          │
                    ┌─────┴─────┐
                    │ 处理器层   │  Processor (清洗、过滤、批处理)
                    └─────┬─────┘
                          │
                    ┌─────┴─────┐
                    │ 存储层     │  Storage (SQLite)
                    └─────┬─────┘
                          │
                    ┌─────┴─────┐
                    │ API/Web层 │  REST API + 可视化界面
                    └───────────┘
```

## 配置说明

### 默认端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Web 界面 | 8080 | 可视化配置和数据查询 |
| TCP 接收器 | 9000 | 接收 TCP 日志流 |
| UDP 接收器 | 9001 | 接收 UDP 日志流 |
| HTTP 接收器 | 9002 | 接收 HTTP POST 日志 |

### 配置文件示例

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "parser": {
    "format": "nginx",
    "delimiter": " ",
    "time_format": "02/Jan/2006:15:04:05 -0700",
    "field_mapping": {
      "0": "client_ip",
      "3": "timestamp",
      "4": "method",
      "5": "path"
    }
  },
  "processor": {
    "worker_count": 10,
    "batch_size": 100,
    "batch_timeout": 1000,
    "clean_rules": [],
    "filter_rules": []
  },
  "receiver": {
    "tcp_enabled": true,
    "tcp_port": 9000,
    "udp_enabled": true,
    "udp_port": 9001,
    "http_enabled": true,
    "http_port": 9002,
    "buffer_size": 8192
  },
  "storage": {
    "type": "sqlite",
    "db_path": "./data/logs.db",
    "retention_hours": 168
  }
}
```

## API 接口

### 配置管理

- `GET /api/config` - 获取当前配置
- `POST /api/config` - 更新配置

### 日志查询

- `GET /api/logs` - 查询日志
  - 参数: `start_time`, `end_time`, `methods`, `status_codes`, `keyword`, `limit`, `offset`
- `POST /api/logs/import` - 导入日志文件

### 统计分析

- `GET /api/statistics` - 获取统计数据

### 数据导出

- `GET /api/export/formats` - 获取支持的导出格式
- `POST /api/export` - 导出数据

### 系统状态

- `GET /api/status` - 获取系统状态

## 日志发送示例

### TCP 发送

```bash
# 使用 nc 发送日志
echo '127.0.0.1 - - [04/Mar/2024:10:30:00 +0800] "GET /api/users HTTP/1.1" 200 1234' | nc localhost 9000
```

### UDP 发送

```bash
# 使用 nc 发送 UDP 日志
echo '<14>Mar  4 10:30:00 server app[1234]: {"level":"info","msg":"user login"}' | nc -u localhost 9001
```

### HTTP 发送

```bash
# 使用 curl 发送日志
curl -X POST http://localhost:9002/logs \
  -H "Content-Type: text/plain" \
  -d '127.0.0.1 - - [04/Mar/2024:10:30:00 +0800] "POST /api/login HTTP/1.1" 200 256'
```

## 性能指标

- 单节点处理能力: 10,000+ 条/秒
- 内存占用: < 200MB (默认配置)
- 并发连接数: 1000+
- 数据存储: 取决于磁盘容量

## 开发计划

- [ ] 支持更多日志格式 (Syslog, LTSV)
- [ ] 添加告警规则配置
- [ ] 支持分布式部署
- [ ] 添加实时监控仪表盘
- [ ] 支持日志压缩存储

## 许可证

MIT License
