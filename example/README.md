# 日志处理器 - 测试工具集

本目录包含用于测试、数据生成和辅助工具的脚本，以及详细的配置指南和开源数据集说明。

## 📁 目录结构

```
example/
├── README.md                      # 本说明文档
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
go run cmd/server/main.go -config config.optimized.json
```

### 2. 压力测试

**⚠️ 重要提示**: 系统有两种不同的处理能力，请根据场景选择测试方式：

| 能力类型 | 速率 | 适用场景 | 说明 |
|---------|------|---------|------|
| **突发处理能力** | ~8,000 QPS | 日志文件导入、短时高峰 | 靠队列缓冲，不能持续 |
| **持续处理能力** | ~800 QPS | 实时日志流、生产环境 | SQLite写入上限，可长期稳定 |

```bash
cd benchmark

# ========== 持续压力测试（推荐用于生产评估） ==========
# 测试长期稳定处理能力，确保发送速率 <= 800 QPS
# 50并发 × 15条/秒 = 750 QPS（可长期稳定）
python stress_test.py -protocol tcp -addr localhost:9000 -total 50000 -c 50 -rate 15

# ========== 突发压力测试（推荐用于峰值评估） ==========
# 短时发送大量日志，测试队列缓冲能力
# 10,000条 @ 8,000 QPS，约1秒完成，队列缓冲后消化
python stress_test.py -protocol tcp -addr localhost:9000 -total 10000 -c 50 -rate 160

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

此脚本测试系统的**突发处理能力**（短时脉冲），通过 3 秒等待让队列消化，找出可承受的最大峰值速率。

**注意**: 该脚本测试的是**短脉冲**能力（每次只发 10,000 条），不是**持续**能力。如需测试长期稳定能力，请使用 `stress_test.py` 发送 50,000 条以上。

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

### 性能基准参考

基于 **异步存储架构** 的系统性能（单节点，SSD磁盘）：

| 能力类型 | 优化配置 (20 worker) | 说明 |
|---------|---------------------|------|
| **突发处理能力** | ~20,000 QPS | 短时峰值，依赖 200,000 队列缓冲 |
| **持续处理能力** | **~1,500 QPS** | 异步批量写入，5倍于同步模式 |

**使用异步存储 (v2.0+)**:
```bash
go run cmd/server/main.go -config config.optimized.json
```

**为什么会有差异？**

```
突发场景（10,000条 @ 8,000 QPS）:
发送: ████████░░░░░░░░░░░░  (1.25秒发完)
队列: [████████░░░░░░░░░░░░]  队列装得下
处理: ░░░░░░░░████████████  3秒后消化完
结果: ✅ 100% 成功

持续场景（100,000条 @ 1,500 QPS）:
发送: ████████████████████  (持续66秒)
队列: [████████████████████]  队列填满后溢出
处理: ░░░░░░░░░░░░░░░░░░░░  持续处理中
结果: ❌ 只处理了 ~40%，其余丢弃
```

---

## 📡 接收器配置指南

### 概述

系统支持三种日志接收方式：
- **TCP** - 长连接，适合高吞吐量的持续日志流
- **UDP** - 无连接，轻量级，适合大量设备上报
- **HTTP** - REST API，适合应用程序主动推送

### TCP/UDP 接收器

| 属性 | TCP | UDP |
|------|-----|-----|
| **默认端口** | 9000 | 9001 |
| **适用场景** | 长连接、高吞吐量、可靠传输 | 大量设备上报、容忍丢包、低延迟 |
| **典型用户** | Nginx、Apache、Filebeat | Syslog、IoT 设备、Docker |

**使用示例：**
```bash
# Nginx 配置
access_log syslog:server=localhost:9000 main;

# Syslog 发送
echo '<14>Test message' | nc -u localhost 9001
```

### HTTP 接收器

| 属性 | 说明 |
|------|------|
| **默认端口** | 9002 |
| **认证 Token** | 防止未授权访问，留空允许匿名（不推荐生产环境） |
| **IP 白名单** | 限制可访问的 IP 地址，逗号分隔 |
| **速率限制** | 每 IP 每分钟最大请求数，0为不限制 |

**使用示例：**
```bash
curl -X POST http://localhost:9002/logs \
  -H "X-Auth-Token: your-secret-token" \
  -d '127.0.0.1 - - [01/Jan/2024:00:00:00 +0800] "GET /api/test HTTP/1.1" 200 123'
```

### 配置建议

**开发环境：**
```
☑️ TCP (9000)  ☑️ UDP (9001)  ☑️ HTTP (9002)
认证 Token: 留空
IP 白名单: 留空
速率限制: 0
```

**生产环境（安全模式）：**
```
☑️ TCP (9000)  ← 内网应用
☑️ HTTP (9002) ← 外网应用
认证 Token: [复杂随机字符串]
IP 白名单: [应用服务器IP列表]
速率限制: 600
```

---

## 📚 开源数据集

### NASA HTTP 日志 ⭐推荐

- **来源**: NASA Kennedy Space Center WWW 服务器
- **时间**: 1995年7月
- **记录数**: 约 190 万条
- **格式**: Apache/Nginx Combined Log Format
- **大小**: 压缩 20MB / 解压后 200MB

**获取方式（Python 脚本）：**
```bash
cd example/benchmark
python download_nasa_logs.py              # 下载到默认目录
python download_nasa_logs.py --verify     # 验证文件完整性
```

**压测使用示例：**
```bash
# 使用 NASA 真实日志进行压测
cd example/benchmark
python stress_test.py -file ../data/NASA_access_log_Jul95.txt -total 100000 -rate 50

# 持续压测 60 秒
python stress_test.py -file ../data/NASA_access_log_Jul95.txt -duration 60 -rate 100 -c 10
```

**格式示例：**
```
199.72.81.55 - - [01/Jul/1995:00:00:01 -0400] "GET /history/apollo/ HTTP/1.0" 200 6245
```

**导入系统：**
1. 系统配置保持默认（Nginx 格式）
2. 直接导入下载的文件
3. 即可分析 1995 年 NASA 网站的访问情况

### 其他数据集

| 数据集 | 记录数 | 用途 |
|--------|--------|------|
| Apache 官方示例 | - | 格式兼容性测试 |
| SecRepo 安全日志 | 数百万条 | 安全分析、入侵检测测试 |
| test_logs.txt (自带) | 10条 | 功能测试 |

---

## 🛠️ 辅助工具

### 生成测试日志

```bash
cd tools
python generate_test_logs.py -n 10000 -f nginx -o test_logs.txt
```

### 日志格式转换

```bash
cd tools
python convert_log_format.py input.csv output.txt --input-format csv --output-format nginx
```

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

---

## ❓ 常见问题

### Q: 为什么 `stress_test.py` 500 QPS 就丢包？

**A**: 如果使用的是 **v1.x 同步存储版本**，这是预期行为。

**根本原因**: SQLite 单线程写入 (~300 QPS) 无法匹配输入速度。

**✅ 已解决（v2.0 异步存储）**:
```bash
# 新版本启用异步存储，持续吞吐量提升至 1,500+ QPS
go run cmd/server/main.go  # 默认启用异步存储

# 测试验证
python stress_test.py -c 20 -rate 100 -total 50000  # 2,000 QPS
```

**异步存储架构**:
```
v1.x 同步: 输入 → 处理 → [阻塞SQLite] → 响应
              ↓
v2.0 异步: 输入 → 处理 → [内存队列] → 立即响应
                              ↓
                        后台批量写入SQLite
```

**性能对比**:
| 模式 | 持续 QPS | 突发 QPS | 丢包率 @500QPS |
|------|----------|----------|----------------|
| v1.x 同步 | ~300 | 8,000 | 40% |
| v2.0 异步 | **1,500+** | **20,000+** | **0%** |

**如果仍需降级到同步模式**（不推荐）:
```bash
# 修改 cmd/server/main.go，注释掉 AsyncStorage 包装
store := sqliteStore  // 直接使用 SQLiteStorage
```

### Q: 为什么 `find_max_capacity.py` 显示 8,000 QPS 成功，但 `stress_test.py` 1,500 QPS 就丢包？

**A**: 两个脚本测试的是不同的能力：
- `find_max_capacity.py`: 测试**突发能力**，每次只发 10,000 条，然后等待 3 秒让队列消化
- `stress_test.py`: 测试**持续能力**，长时间发送，队列填满后就会丢包

**建议**: 生产环境实时流控制在 **800 QPS** 以下，文件导入可以用 **8,000 QPS** 快速完成。

### 测试时成功率低（大量丢弃）

**原因**: 发送速率超过了系统的**持续处理能力**（~800 QPS）

```bash
# 使用限流测试，确保持续速率 ≤ 800 QPS
python stress_test.py -total 100000 -c 50 -rate 15
```

### 连接被拒绝

```bash
# 检查服务器状态
curl http://localhost:8080/api/status

# 检查端口是否被占用
netstat -an | findstr 9000
```

### HTTP 返回 401/403/429

- **401 Unauthorized**: 检查 Token 是否正确
- **403 Forbidden**: 检查客户端 IP 是否在白名单中
- **429 Too Many Requests**: 增加速率限制值或设置为 0

---

## 💡 最佳实践

### 1. 区分测试目的

**测试突发能力**（评估峰值承载）:
```bash
python find_max_capacity.py -protocol tcp -addr localhost:9000
```

**测试持续能力**（评估生产配置）:
```bash
python stress_test.py -protocol tcp -total 100000 -c 50 -rate 15
```

### 2. 生产环境限流建议

- **实时日志流**: 控制在 **800 QPS** 以下（长期稳定）
- **文件导入**: 可接受 **8,000 QPS** 脉冲（短时完成）
- **混合负载**: 平均 ≤ 800 QPS，峰值 ≤ 8,000 QPS

### 3. 其他建议

- **测试数据随机性**: 脚本使用完全随机的真实分布数据，结果更贴近生产环境
- **监控资源**: 测试时观察 CPU、内存、磁盘 I/O
- **测试前清空数据**: 默认会自动清空，使用 `-no-clear` 可以保留历史数据

---

## 📝 版本记录

- v1.0: 基础性能测试功能
- v1.1: 添加限流支持和增量计算
- v1.2: 添加系统极限查找工具
- v1.3: 测试数据完全随机化（IP、方法、状态码、路径、UA等真实分布）
- v1.4: 优化队列容量（200,000缓冲）
- v1.5: 明确区分突发能力(~8,000 QPS)与持续能力(~800 QPS)，更新文档
