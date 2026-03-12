# 日志处理器 - 测试工具集 (v2.0)

本目录包含用于测试、数据生成和辅助工具的脚本，支持 **v2.0 容错架构**（背压机制 + 磁盘溢出队列）。

## 目录结构

```
example/
├── README.md                      # 本说明文档
├── benchmark/                     # 性能测试脚本
│   ├── stress_test.py            # 压力测试（支持容错监控）
│   ├── diagnose_system.py        # 系统诊断工具（监控背压/溢出）
│   ├── test_resilient.py         # 容错机制专项测试
│   ├── download_nasa_logs.py     # NASA 数据集下载
│   ├── generate_nasa_like_logs.py# 生成 NASA 格式模拟数据
│   ├── find_max_capacity.py      # 查找系统最大处理能力
│   └── stress_test.go            # 压力测试（Go）
├── data/                          # 测试数据文件
│   ├── test_log.txt              # 基础测试日志
│   ├── test_logs.txt             # Nginx格式测试日志
│   ├── test_logs_json.txt        # JSON格式测试日志
│   └── NASA_access_log_Jul95_simulated.txt  # NASA 模拟数据(100万条)
└── tools/                         # 辅助工具
    ├── generate_test_logs.py     # 生成测试日志
    ├── convert_log_format.py     # 日志格式转换
    ├── send_logs_unix.sh         # Unix/Mac 发送日志脚本
    └── send_logs_windows.ps1     # Windows 发送日志脚本
```

---

## 快速开始

### 1. 启动服务器

```bash
cd ..
go run cmd/server/main.go
# 或使用高性能配置
go run cmd/server/main.go -config config.optimized.json
```

### 2. 压力测试 (v2.0+ 容错架构)

**系统能力（v2.0 容错架构）**:

| 能力类型 | 速率 | 适用场景 | 容错机制 |
|---------|------|---------|----------|
| **突发处理能力** | ~20,000 QPS | 日志文件导入、短时高峰 | 背压 + 溢出队列 |
| **持续处理能力** | ~1,500 QPS | 实时日志流、生产环境 | 异步批量写入 |

**容错机制说明**:
- **背压机制**: 队列满时自动降速（延迟 10ms-100ms）
- **溢出队列**: 数据暂存到磁盘 (`./temp/overflow/`)，空闲时回填
- **数据保证**: 至少一次 (At-Least-Once)，保证不丢

```bash
cd benchmark

# ========== 系统诊断（推荐先运行） ==========
# 查看当前系统状态、背压级别、溢出情况
python diagnose_system.py --once

# ========== 持续压力测试（推荐用于生产评估） ==========
# 测试长期稳定处理能力，发送速率 <= 1,500 QPS
# 30并发 × 40条/秒 = 1,200 QPS（可长期稳定）
python stress_test.py -protocol tcp -addr localhost:9000 -total 50000 -c 30 -rate 40

# ========== 突发压力测试（推荐用于峰值评估） ==========
# 短时发送大量日志，测试队列缓冲和溢出能力
# 10,000条 @ 2,000 QPS，测试容错机制
python stress_test.py -protocol tcp -addr localhost:9000 -total 10000 -c 50 -rate 40

# ========== 使用真实数据测试容错能力 ==========
# 生成 100万条 NASA 格式数据
python generate_nasa_like_logs.py -n 1000000
# 高速发送，观察溢出队列工作情况
python stress_test.py -file ../data/NASA_access_log_Jul95_simulated.txt -total 100000 -rate 100

# HTTP 测试（批量发送效率更高）
python stress_test.py -protocol http -addr localhost:9002 -total 50000 -c 50 -rate 30

# 实时监控测试过程
python diagnose_system.py

# 保留已有数据测试（不清空）
python stress_test.py -protocol tcp -total 10000 -c 10 -rate 40 -no-clear
```

### 3. 容错机制专项测试 (v2.0)

```bash
cd benchmark

# 运行完整容错测试（5万条，监控30秒）
python test_resilient.py

# 快速测试（1万条，监控15秒）
python test_resilient.py --quick

# 自定义参数
python test_resilient.py --count 100000 --rate 200 --wait 60
```

**测试流程**:
1. 检查服务器状态和容错机制是否启用
2. 清空历史数据
3. 高速发送数据（超过处理能力）
4. 实时监控背压级别、溢出队列、回填情况
5. 验证最终数据完整性

**输出示例**:
```
[容错机制测试] (v2.0)

[1] 检查服务器状态...
    [OK] 容错机制已启用

[2] 清空历史数据...

[3] 测试参数:
    发送总数: 50,000 条
    目标速率: 100 QPS

[4] 高速发送数据...
    时间       日志数     背压   溢出     回填     QPS
    --------------------------------------------------
    14:32:01      1,234      L        0        0      234
    14:32:02      2,567      M      100        0      333
    14:32:03      3,890      H      500       50      323
    ...

[6] 最终结果:
    [数据统计]
    目标发送: 50,000 条
    实际发送: 50,000 条
    最终存储: 49,850 条
    存储成功率: 99.7%

    [容错统计]
    背压级别: 2
    溢出总数: 2,500 条
    已回填数: 2,350 条
```

### 4. 查找系统极限

```bash
cd benchmark
python find_max_capacity.py -protocol tcp -addr localhost:9000
```

此脚本测试系统的**突发处理能力**（短时脉冲），通过 3 秒等待让队列消化，找出可承受的最大峰值速率。

**注意**: 该脚本测试的是**短脉冲**能力（每次只发 10,000 条），不是**持续**能力。如需测试长期稳定能力，请使用 `stress_test.py` 发送 50,000 条以上。

---

## 性能测试详解

### stress_test.py 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-protocol` | 协议类型: tcp/udp/http | tcp |
| `-addr` | 目标地址 | localhost:9000 |
| `-total` | 总发送日志数 | 10000 |
| `-c` | 并发连接数 | 10 |
| `-d` | 测试持续时间(秒) | 0（按total发送） |
| `-rate` | **每连接限流(条/秒)，建议30-50** | 0（不限流） |
| `-batch` | HTTP批量发送条数 | 100 |
| `-file` | 从文件读取日志数据 | 无 |
| `-no-clear` | 不清空服务端数据 | 默认清空 |

**v2.0 建议使用参数**:
```bash
# 测试容错能力（发送速率 > 处理能力）
python stress_test.py -c 50 -rate 50 -total 100000

# 观察溢出队列工作
# 1. 运行测试
# 2. 另开窗口: python diagnose_system.py
# 3. 查看溢出计数是否增加，以及后续是否回填
```

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

### 性能基准参考 (v2.0 容错架构)

基于 **异步存储 + 背压 + 溢出队列** 的系统性能（单节点，SSD磁盘）：

| 能力类型 | 速率 | 容错机制 | 数据保证 |
|---------|------|----------|----------|
| **突发处理能力** | ~20,000 QPS | 背压降速 + 磁盘溢出 | 至少一次 |
| **持续处理能力** | **~1,500 QPS** | 异步批量写入 | 至少一次 |

**对比 (v1.0 vs v2.0)**:

```
v1.0 (同步模式):
发送: ████████░░░░░░░░░░░░  (1,000 QPS)
队列: 满 -> [X] 直接丢弃
成功率: ~70%

v2.0 (容错模式):
发送: ████████░░░░░░░░░░░░  (1,000 QPS)
队列: 满 -> [DISK] 溢出到磁盘 -> 空闲时回填
成功率: ~98%
```

**容错工作流程**:

```
高负载场景:
输入 2,000 QPS -> 背压降速(延迟50ms) -> 溢出到磁盘 -> 队列空闲时回填 -> SQLite存储
                    |
              系统不会崩溃，数据不会丢失
```

**溢出队列位置**: `./temp/overflow/`
- 最多 5 个文件，每个 100MB
- 队列空闲时自动回填（每5秒检查）
- 超过24小时的文件自动清理

---

## 接收器配置指南

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
[x] TCP (9000)  [x] UDP (9001)  [x] HTTP (9002)
认证 Token: 留空
IP 白名单: 留空
速率限制: 0
```

**生产环境（安全模式）：**
```
[x] TCP (9000)  <- 内网应用
[x] HTTP (9002) <- 外网应用
认证 Token: [复杂随机字符串]
IP 白名单: [应用服务器IP列表]
速率限制: 600
```

---

## 开源数据集

### NASA HTTP 日志 [推荐]

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

## 辅助工具

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

## 常见问题

### Q: 为什么 `stress_test.py` 显示成功率不是 100%？

**A**: 这是**预期行为**，v2.0 容错机制的设计选择。

**系统行为**:
```
发送速率 > 处理能力 (1,500 QPS) 时：
1. 触发背压 - 接收端自动降速
2. 启用溢出队列 - 数据暂存到磁盘
3. 部分数据在传输层被丢弃（TCP缓冲区满）
```

**关键概念**:
- **客户端成功率**: 成功发送到服务器的比例
- **服务端存储率**: 最终成功存储的比例（含溢出回填）

**查看真实成功率**:
```bash
# 运行测试后等待 10 秒（让回填完成）
python stress_test.py -c 50 -rate 100 -total 100000
sleep 10

# 查看实际存储数量
curl http://localhost:8080/api/logs?limit=1
```

**v2.0 数据保证**:
| 模式 | 客户端成功率 | 服务端最终存储率 | 数据保证 |
|------|-------------|-----------------|---------|
| 无容错 | ~70% | ~70% | 最多一次 |
| **v2.0 容错** | ~90% | **~99%** | **至少一次** |

**溢出队列状态**:
```bash
python diagnose_system.py --once
# 查看 overflow_count 和 drain_count
```

### Q: 为什么 `find_max_capacity.py` 显示 8,000 QPS 成功，但 `stress_test.py` 1,500 QPS 就丢包？

**A**: 两个脚本测试的是不同的能力：
- `find_max_capacity.py`: 测试**突发能力**，每次只发 10,000 条，然后等待 3 秒让队列消化
- `stress_test.py`: 测试**持续能力**，长时间发送，队列填满后就会丢包

**建议**: 生产环境实时流控制在 **800 QPS** 以下，文件导入可以用 **8,000 QPS** 快速完成。

### 测试时成功率低（大量丢弃）

**原因**: 发送速率超过了系统的**持续处理能力**（~800 QPS）

```bash
# 使用限流测试，确保持续速率 <= 800 QPS
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

## 最佳实践

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
- **混合负载**: 平均 <= 800 QPS，峰值 <= 8,000 QPS

### 3. 其他建议

- **测试数据随机性**: 脚本使用完全随机的真实分布数据，结果更贴近生产环境
- **监控资源**: 测试时观察 CPU、内存、磁盘 I/O
- **测试前清空数据**: 默认会自动清空，使用 `-no-clear` 可以保留历史数据

---

## 版本记录

- v1.0: 基础性能测试功能
- v1.1: 添加限流支持和增量计算
- v1.2: 添加系统极限查找工具
- v1.3: 测试数据完全随机化（IP、方法、状态码、路径、UA等真实分布）
- v1.4: 优化队列容量（200,000缓冲）
- v1.5: 明确区分突发能力(~8,000 QPS)与持续能力(~800 QPS)，更新文档
- **v2.0**: 添加容错机制（背压 + 溢出队列），支持至少一次数据保证
