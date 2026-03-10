# 日志处理器 - 测试工具集

本目录包含用于测试、数据生成和辅助工具的脚本。

## 📁 目录结构

```
example/
├── README.md                      # 本说明文档
├── DATASETS.md                    # 开源数据集说明
├── benchmark/                     # 性能测试脚本
│   ├── stress_test.py            # 压力测试（Python）- 支持随机真实数据
│   ├── stress_test.go            # 压力测试（Go）
│   └── find_max_capacity.py      # 查找系统最大处理能力
├── data/                          # 测试数据文件
│   ├── test_log.txt              # 基础测试日志
│   ├── test_logs.txt             # Nginx格式测试日志
│   └── test_logs_json.txt        # JSON格式测试日志
└── tools/                         # 辅助工具
    ├── generate_test_logs.py     # 生成测试日志
    ├── convert_log_format.py     # 日志格式转换
    ├── send_logs_unix.sh         # Unix/Mac 发送日志脚本
    ├── send_logs_windows.ps1     # Windows 发送日志脚本
    ├── download_nasa_unix.sh     # Unix 下载 NASA 数据集
    └── download_nasa_windows.ps1 # Windows 下载 NASA 数据集
```

---

## 🚀 快速开始

### 1. 启动服务器

```bash
cd ..
go run cmd/server/main.go
# 或使用高性能配置
go run cmd/server/main.go -config config.extreme.json
```

### 2. 压力测试

**⚠️ 重要提示**: 系统处理能力上限约 **1,500 QPS**，超过此速率将导致日志被丢弃。

```bash
cd benchmark

# 推荐：限流测试 - 确保100%成功率
# 10连接，每连接15条/秒 = 总计1,500 QPS（系统上限）
python stress_test.py -protocol tcp -addr localhost:9000 -total 10000 -c 10 -rate 15

# 基础压力测试 - 发送1万条，100并发（可能超系统能力）
python stress_test.py -protocol tcp -addr localhost:9000 -total 10000 -c 100

# HTTP 测试（批量发送效率更高）
python stress_test.py -protocol http -addr localhost:9002 -total 50000 -c 50 -rate 30

# UDP 测试
python stress_test.py -protocol udp -addr localhost:9001 -total 50000 -c 50 -rate 30

# 保留已有数据测试（不清空）
python stress_test.py -protocol tcp -total 10000 -c 10 -rate 15 -no-clear
```

### 3. 查找系统极限

```bash
cd benchmark
python find_max_capacity.py -protocol tcp -addr localhost:9000
```

此脚本会逐步增加压力，找出系统的最大稳定处理能力。

---

## 📊 性能测试详解

### stress_test.py 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-protocol` | 协议类型: tcp/udp/http | tcp |
| `-addr` | 目标地址 | localhost:9000 |
| `-total` | 总发送日志数 | 10000 |
| `-c` | 并发连接数 | 10 |
| `-d` | 测试持续时间(秒) | 0（按total发送） |
| `-rate` | **每连接限流(条/秒)，建议15-20** | 0（不限流） |
| `-batch` | HTTP批量发送条数 | 100 |
| `-no-clear` | 不清空服务端数据 | 默认清空 |

### 测试数据说明

压力测试使用**完全随机**的真实分布数据：

| 字段 | 随机分布 |
|------|----------|
| Client IP | 6个IP段 × 254 = 1,524个随机IP |
| HTTP Method | GET(70%), POST(20%), PUT(5%), DELETE(3%), PATCH(2%) |
| Path | 24个真实API路径随机选择 |
| Status Code | 2xx(82%), 3xx(9%), 4xx(11%), 5xx(2%) |
| Response Size | 100-100,000 bytes 随机 |
| Referer | 7种来源随机（含搜索引擎） |
| User-Agent | 8种真实浏览器/工具随机 |
| Response Time | 0.001-5.0秒随机 |

这种随机性确保测试结果更接近生产环境真实性能（避免CPU缓存和分支预测优化导致的虚高数据）。

### 测试示例

**场景1: 推荐 - 测试稳定处理能力（限流）**
```bash
# 100并发 × 15条/秒 = 1,500 QPS（系统稳定上限）
python stress_test.py -protocol tcp -total 100000 -c 100 -rate 15
```

**场景2: 测试极限吞吐量（可能丢包）**
```bash
# 不限流，测试系统峰值（会超过1,500 QPS，导致部分丢弃）
python stress_test.py -protocol tcp -total 100000 -c 100
```

**场景3: HTTP 批量发送**
```bash
# 50并发 × 30条/秒 = 1,500 QPS
python stress_test.py -protocol http -total 100000 -c 50 -rate 30
```

---

## 🛠️ 辅助工具

### 生成测试日志

```bash
cd tools
python generate_test_logs.py -n 10000 -f nginx -o test_logs.txt
```

参数:
- `-n`: 生成日志条数
- `-f`: 格式 (nginx/apache/json/csv)
- `-o`: 输出文件

### 发送日志到服务器

**Windows:**
```powershell
cd tools
.\send_logs_windows.ps1
```

**Unix/Mac:**
```bash
cd tools
chmod +x send_logs_unix.sh
./send_logs_unix.sh
```

### 日志格式转换

```bash
cd tools
python convert_log_format.py input.csv output.txt --input-format csv --output-format nginx
```

---

## 📈 性能基准参考

基于 SQLite 存储的系统性能（单节点，SSD磁盘）：

| 配置 | 队列容量 | 稳定处理能力 | 成功率 |
|------|---------|-------------|--------|
| 默认 (4 worker, batch 100) | 1,000 | ~500 QPS | 100% |
| 高性能 (50 worker, batch 1000) | 100,000 | **~1,500 QPS** | **100%** |

### 重要发现

1. **SQLite 是瓶颈**: 单文件写入上限约 **1,500 QPS**，这是物理限制
2. **队列缓存**: 高性能配置提供 100,000 条日志的突发缓冲
3. **限流建议**: 生产环境建议控制在 **1,200-1,400 QPS** 以保持稳定

### 测试对比

| 测试条件 | 发送速率 | 成功率 | 说明 |
|---------|---------|--------|------|
| 不限流 | ~10,000 QPS | ~40% | 远超处理能力，大量丢弃 |
| 限流 15/连接 × 100并发 | 1,500 QPS | **100%** | 推荐配置 |

**注意**: 实际性能取决于磁盘 I/O（SSD vs HDD）和 CPU。

---

## 🔧 故障排查

### 测试时成功率低（大量丢弃）

**原因**: 发送速率超过了系统处理能力（~1,500 QPS）

```bash
# 1. 检查服务器状态
curl http://localhost:8080/api/status

# 2. 使用限流测试（推荐每连接15-20条/秒）
# 示例：100并发 × 15条/秒 = 1,500 QPS（系统上限）
python stress_test.py -total 10000 -c 100 -rate 15

# 3. 如需更高吞吐，考虑：
#    - 部署多个实例分片处理
#    - 切换到 PostgreSQL 等更强数据库
```

### 连接被拒绝

```bash
# 检查服务器是否启动
curl http://localhost:8080/api/status

# 检查端口是否被占用
netstat -an | findstr 9000
```

### 解析错误率高

查看服务器日志，确认时间格式配置正确：
```
# 日志中如果显示 "Parse error"，说明格式不匹配
# 访问配置页面 http://localhost:8080 调整解析格式
```

---

## 📚 数据集

查看 `DATASETS.md` 了解如何获取开源日志数据集进行测试：
- NASA 1995 年真实访问日志（130万条）

---

## 💡 最佳实践

1. **始终使用限流**: 建议使用 `-rate 15`（100并发时约1,500 QPS），确保100%成功率
2. **测试数据随机性**: 脚本使用完全随机的真实分布数据，结果更贴近生产环境
3. **渐进式加压**: 先用 `find_max_capacity.py` 找出极限，再决定生产配置
4. **监控资源**: 测试时观察 CPU、内存、磁盘 I/O
5. **测试前清空数据**: 默认会自动清空，使用 `-no-clear` 可以保留历史数据

---

## 📝 版本记录

- v1.0: 基础性能测试功能
- v1.1: 添加限流支持和增量计算
- v1.2: 添加系统极限查找工具
- v1.3: 测试数据完全随机化（IP、方法、状态码、路径、UA等真实分布）
- v1.4: 优化队列容量（100,000缓冲），明确系统上限1,500 QPS
