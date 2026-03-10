# 日志处理器 - 测试工具集

本目录包含用于测试、数据生成和辅助工具的脚本。

## 📁 目录结构

```
example/
├── README.md                      # 本说明文档
├── DATASETS.md                    # 开源数据集说明
├── benchmark/                     # 性能测试脚本
│   ├── stress_test.py            # 压力测试（Python）
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

```bash
cd benchmark

# 基础压力测试 - 发送 1万条，100并发
python stress_test.py -protocol tcp -addr localhost:9000 -total 10000 -c 100

# 限流测试 - 10连接，每连接80条/秒（总800 QPS）
python stress_test.py -protocol tcp -addr localhost:9000 -total 10000 -c 10 -rate 80

# HTTP 测试
python stress_test.py -protocol http -addr localhost:9002 -total 50000 -c 50

# UDP 测试
python stress_test.py -protocol udp -addr localhost:9001 -total 50000 -c 50

# 保留已有数据测试（不清空）
python stress_test.py -protocol tcp -total 10000 -c 10 -no-clear
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
| `-c` | 并发连接数 | 100 |
| `-d` | 测试持续时间(秒) | 0（按total发送） |
| `-rate` | 每连接限流(条/秒) | 0（不限流） |
| `-batch` | HTTP批量发送条数 | 1 |
| `-no-clear` | 不清空服务端数据 | 默认清空 |

### 测试示例

**场景1: 测试极限吞吐量（不限流）**
```bash
python stress_test.py -protocol tcp -total 100000 -c 100
```

**场景2: 测试稳定性（限流）**
```bash
# 模拟 1000 QPS 持续压力
python stress_test.py -protocol tcp -total 100000 -c 20 -rate 50
```

**场景3: 测试 HTTP 批量发送**
```bash
python stress_test.py -protocol http -total 100000 -c 50 -batch 100
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

基于 SQLite 存储的系统性能（单节点）：

| 配置 | 稳定处理能力 | 峰值处理能力 | 内存占用 |
|------|-------------|-------------|---------|
| 默认 (10 worker, batch 100) | ~500 QPS | ~800 QPS | ~100MB |
| 高性能 (20 worker, batch 2000) | ~1,000 QPS | ~1,500 QPS | ~200MB |
| 极端 (20 worker, batch 5000) | ~1,200 QPS | ~2,000 QPS | ~300MB |

**注意**: 实际性能取决于磁盘 I/O（SSD vs HDD）和 CPU。

---

## 🔧 故障排查

### 测试时成功率低（大量丢弃）

```bash
# 1. 检查服务器状态
curl http://localhost:8080/api/status

# 2. 降低发送速率
python stress_test.py -total 10000 -c 10 -rate 50

# 3. 增加服务器 worker_count
# 修改 config.json 中的 processor.worker_count 为更大的值
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

1. **测试前清空数据**: 默认会自动清空，使用 `-no-clear` 可以保留历史数据
2. **渐进式加压**: 先用 `find_max_capacity.py` 找出极限，再决定生产配置
3. **监控资源**: 测试时观察 CPU、内存、磁盘 I/O
4. **使用限流**: 生产环境建议用 `-rate` 控制发送速率，避免压垮服务器

---

## 📝 版本记录

- v1.0: 基础性能测试功能
- v1.1: 添加限流支持和增量计算
- v1.2: 添加系统极限查找工具
