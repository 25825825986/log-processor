# 轻量级高并发日志数据处理系统

基于 Go 语言开发的轻量级高并发日志数据处理系统，支持实时日志接收、自动格式识别、解析、存储和导出，附带可视化 Web 管理界面。

## 特性

- **高并发处理**：Goroutine + Channel 流水线，多 Worker 并行解析，单机可达 1,500+ QPS 持续 / 20,000+ QPS 突发
- **溢出保护**：内存队列满时自动溢写到磁盘；readOffset 持久化，进程重启后不重放已消费数据
- **多协议接收**：TCP / UDP / HTTP 三路并行接收，支持文件批量导入
- **智能解析**：自动识别 Nginx、Apache、JSON、CSV、TSV、Syslog 等格式；预编译正则，热路径零 GC 压力
- **异步存储**：内存批次缓冲 + 定时刷盘，SQLite WAL 模式，写入延迟 < 1ms
- **配置热更新**：通过 Web 界面或 REST API 修改配置，即时生效无需重启
- **API 认证**：可选 Bearer Token 认证，写操作与只读操作分离；CORS 白名单模式
- **数据导出**：支持 Excel / CSV / JSON 格式
- **可观测性**：`GET /metrics` 暴露 Prometheus 文本格式指标；内置一键压测

## 快速开始

```bash
# 1. 安装 Go 1.21+ 并获取依赖
go mod download

# 2. 编译
go build -o bin/log-processor ./cmd/server/

# 3. 启动（基础配置）
./bin/log-processor -config config.example.json

# 4. 打开 Web 界面
# 浏览器访问 http://localhost:8080
```

或直接运行（无需预编译）：

```bash
go run ./cmd/server/ -config config.example.json
```

**生成测试日志数据：**

```bash
cd example/tools
python generate_test_logs.py -n 10000 -f nginx -o ../../temp/test.log
```

更多工具见 [`example/README.md`](example/README.md)。

## 项目结构

```
Log_processor/
├── cmd/server/
│   └── main.go                 # 程序入口：初始化各层并启动服务
├── internal/
│   ├── config/config.go        # 全局配置（单例，支持热更新）
│   ├── models/                 # 数据模型：LogEntry、FilterCondition、Statistics
│   ├── parser/
│   │   ├── parser.go           # 多格式解析器（预编译正则）
│   │   └── auto_detect.go      # 格式自动识别
│   ├── processor/
│   │   ├── processor.go        # 多 Worker 流水线处理器
│   │   └── overflow_queue.go   # 磁盘溢写队列（offset 持久化，原子压缩）
│   ├── receiver/receiver.go    # TCP / UDP / HTTP / 文件接收器
│   ├── storage/
│   │   ├── storage.go          # SQLite 存储（WAL 模式）
│   │   └── async_storage.go    # 异步写入包装器
│   ├── exporter/exporter.go    # Excel / CSV / JSON 导出
│   └── server/
│       ├── server.go           # Server 结构体、路由、中间件、所有 handler
│       └── helpers.go          # 辅助函数
├── web/
│   ├── index.html              # 单页面 Web 界面
│   ├── css/style.css
│   └── js/app.js
├── example/
│   ├── tools/                  # 测试数据生成、格式转换、发送脚本
│   └── benchmark/              # QPS / 并发 / 突发压测脚本
├── config.example.json         # 基础配置（开发 / 演示）
├── config.optimized.json       # 高性能配置（生产）
├── Makefile                    # 构建、测试、压测快捷命令
├── go.mod
└── go.sum
```

运行时生成（已 gitignore）：

```
data/          # SQLite 数据库文件、溢写队列文件及 offset 辅助文件
logs/          # 程序运行日志
exports/       # 数据导出文件
temp/          # 文件导入临时目录
bin/           # 编译产物
```

## 系统架构

```
日志数据源
    │
    ▼
接收器层 (TCP :9000 / UDP :9001 / HTTP :9002 / 文件导入)
    │  内存队列满时旁路溢写到磁盘（overflow.queue）
    ▼
处理器层 (多 Worker 并行解析)
    │
    ▼
存储层 (AsyncStorage 缓冲 → SQLite WAL 批量写入)
    │
    ▼
API / Web 层 (Gin, :8080)
```

## 配置说明

### 默认端口

| 服务        | 端口 | 说明                   |
|------------|------|----------------------|
| Web 界面    | 8080 | 可视化管理、查询、导出    |
| TCP 接收器  | 9000 | 可靠传输，逐行读取        |
| UDP 接收器  | 9001 | 高吞吐，允许少量丢包      |
| HTTP 接收器 | 9002 | POST 批量提交，支持认证  |

### 完整配置项

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "api_token": "",         // 非空时启用写操作 Bearer Token 认证
    "cors_origins": []       // 为空时仅允许 localhost；["*"] 放通所有来源
  },
  "parser": {
    "format": "auto"         // auto | nginx | apache | json | csv | tsv | syslog | ...
  },
  "processor": {
    "worker_count": 20,
    "batch_size": 2000,
    "batch_timeout": 500,    // ms，批次超时强制刷盘
    "overflow_enabled": true,
    "overflow_dir": "./data/overflow",
    "overflow_max_disk_mb": 512,
    "overflow_drain_batch": 1000,
    "overflow_drain_interval_ms": 200
  },
  "receiver": {
    "tcp_enabled": true,  "tcp_port": 9000,
    "udp_enabled": true,  "udp_port": 9001,
    "http_enabled": true, "http_port": 9002,
    "http_auth_token": "",       // HTTP 接收器独立 Token
    "http_allowed_ips": [],      // IP 白名单，空表示不限
    "http_max_body_size": 0,     // 字节，0 = 默认 10MB
    "http_rate_limit": 0,        // 每 IP 每分钟请求数，0 = 不限
    "buffer_size": 8192,
    "max_connections": 1000
  },
  "storage": {
    "type": "sqlite",
    "db_path": "./data/logs.db",
    "retention_hours": 168       // 数据保留时长，0 = 永不清理
  },
  "alert": {
    "slow_threshold": 1000,      // ms，平均响应时间告警阈值
    "error_rate_threshold": 5    // %，错误率告警阈值
  },
  "import": {
    "max_lines": 100000
  }
}
```

### API 认证

设置 `server.api_token` 后，所有**写操作**（POST / DELETE）需要携带令牌；**只读接口**（GET）无需认证。

令牌传递方式（任选其一）：

```bash
# Authorization header
curl -H "Authorization: Bearer <token>" -X POST http://localhost:8080/api/config ...

# 自定义 header
curl -H "X-API-Token: <token>" -X POST http://localhost:8080/api/config ...

# 查询参数（调试用）
curl -X POST "http://localhost:8080/api/config?token=<token>" ...
```

## REST API

### 只读接口（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/config` | 获取当前配置（api_token 字段不返回明文） |
| GET | `/api/status` | 系统状态、处理器统计、告警摘要 |
| GET | `/api/logs` | 查询日志，支持时间范围、方法、路径、状态码、关键字等多维筛选 |
| GET | `/api/logs/import/progress?id=` | 文件导入进度 |
| GET | `/api/statistics` | 汇总统计：总量、错误数、平均响应时间、状态码分布、Top 路径 |
| GET | `/api/export/formats` | 支持的导出格式列表 |
| GET | `/api/storage/info` | 存储引擎信息及数据库大小 |
| GET | `/api/benchmark/report` | 最近一次压测报告 |
| GET | `/metrics` | Prometheus 文本格式指标 |

### 写操作接口（需认证，未配置 token 时向后兼容）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/config` | 热更新配置，支持局部更新 |
| POST | `/api/logs/import` | 上传文件批量导入 |
| DELETE | `/api/logs/:id` | 删除单条日志 |
| DELETE | `/api/logs` | 清空所有日志 |
| POST | `/api/export` | 导出数据（Excel / CSV / JSON） |
| POST | `/api/receiver/start` | 启动接收器 |
| POST | `/api/receiver/stop` | 停止接收器 |
| POST | `/api/storage/compact` | 压缩数据库（VACUUM） |
| POST | `/api/benchmark/run` | 执行一键压测 |

### 日志查询参数

```
GET /api/logs?page=1&limit=20
    &start_time=2024-01-01T00:00:00Z
    &end_time=2024-01-02T00:00:00Z
    &methods=GET&methods=POST
    &status_codes=200&status_codes=404
    &status_code_ranges=400-499
    &client_ips=192.168.1.1
    &min_response_time=100
    &max_response_time=5000
    &keyword=login
```

## 日志发送示例

### TCP

```bash
echo '192.168.1.1 - - [04/Mar/2024:10:30:00 +0800] "GET /api/users HTTP/1.1" 200 1234 "-" "curl/8.5.0"' \
  | nc localhost 9000
```

### UDP

```bash
echo '192.168.1.1 - - [04/Mar/2024:10:30:00 +0800] "POST /api/login HTTP/1.1" 200 256' \
  | nc -u localhost 9001
```

### HTTP

```bash
curl -X POST http://localhost:9002/logs \
  -H "Content-Type: text/plain" \
  -d '192.168.1.1 - - [04/Mar/2024:10:30:00 +0800] "GET /api/users HTTP/1.1" 200 1234'
```

### Python 批量发送

```python
import socket, time

lines = [
    f'10.0.0.{i%254+1} - - [04/Mar/2024:10:30:00 +0800] "GET /api/item/{i} HTTP/1.1" 200 512'
    for i in range(10000)
]

with socket.create_connection(("localhost", 9000)) as s:
    for line in lines:
        s.sendall((line + "\n").encode())
        time.sleep(0.0001)
```

## 性能参考

测试环境：4 核 / 8GB / SSD，SQLite WAL 模式，异步存储缓冲 50,000 条。

| 指标 | 基础配置 | 高性能配置 |
|------|----------|-----------|
| 持续吞吐量 | ~1,500 QPS | ~3,000+ QPS |
| 突发吞吐量 | ~20,000 QPS | ~50,000+ QPS |
| 写入延迟 | < 1ms | < 1ms |
| 内存占用 | < 150MB | < 300MB |

运行内置压测：

```bash
# 通过 Web 界面：「配置」→「压测」→「运行」
# 或通过 API：
curl -X POST http://localhost:8080/api/benchmark/run \
  -H "Content-Type: application/json" \
  -d '{"duration_seconds": 10, "workers": 20, "target_qps": 5000}'
```

## 常用 Make 命令

```bash
make build          # 编译到 build/ 目录
make run            # go run 直接运行
make test           # 运行所有测试
make fmt            # 格式化代码
make deps           # 下载并整理依赖
make build-all      # 交叉编译 Linux / macOS / Windows
make clean          # 清理编译产物和运行时数据目录
make benchmark-tcp  # TCP 压测（需先启动服务）
make benchmark-http # HTTP 压测
```

## 溢写队列说明

当处理器内存队列满时，新到达的日志行会被追加写入 `data/overflow/overflow.queue`（base64 编码，每行一条）。空闲时定期回灌到内存队列继续处理。

关键行为：

- **不重放**：读取偏移量持久化到 `overflow.offset`，进程重启后从上次位置续读，不产生重复日志。
- **容量控制**：磁盘上限基于*待处理字节数*（文件大小 - 已消费偏移），而非文件总大小。
- **原子压缩**：清理已消费前缀时先写临时文件再 rename，写失败不丢数据。

## 许可证

MIT License
