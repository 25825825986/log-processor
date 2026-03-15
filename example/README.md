# Example 测试工具目录

本目录用于本项目的手工测试、压测和数据准备。

## 目录结构

- `benchmark/`
  - `stress_test.py`：TCP/UDP/HTTP 压测发送器
  - `find_max_capacity.py`：逐级探测系统可承受速率
  - `diagnose_system.py`：读取接口状态并监控队列
  - `download_nasa_logs.py`：下载 NASA 日志样本
  - `generate_nasa_like_logs.py`：生成 NASA 格式模拟日志
  - `test_resilient.py`：高压场景下的韧性测试
  - `stress_test.go`：Go 版发送器
  - `api_smoke_test_test.go`：Go 接口冒烟测试
- `tools/`
  - `generate_test_logs.py`：生成测试日志
  - `convert_log_format.py`：日志格式转换
  - `send_logs_unix.sh`：Unix 发送示例
  - `send_logs_windows.ps1`：Windows 发送示例
- `data/`
  - 示例测试数据文件

## 前置条件

1. 启动主服务（仓库根目录）：

```bash
go run cmd/server/main.go
```

2. 默认端口：
- Web/API: `8080`
- TCP Receiver: `9000`
- UDP Receiver: `9001`
- HTTP Receiver: `9002`

## 推荐测试顺序

1. 诊断当前状态

```bash
cd example/benchmark
python diagnose_system.py --once
```

2. 运行压测（TCP）

```bash
python stress_test.py -protocol tcp -addr localhost:9000 -total 20000 -c 20 -rate 40
```

3. 探测容量

```bash
python find_max_capacity.py -protocol tcp -addr localhost:9000 -c 30
```

4. 韧性测试（高压）

```bash
python test_resilient.py --quick
```

5. 接口冒烟测试（Go）

```bash
go test ./example/benchmark -run TestStatusEndpoint -v
go test ./example/benchmark -run TestConfigEndpoint -v
```

## 数据生成/转换

生成测试日志：

```bash
cd example/tools
python generate_test_logs.py -n 5000 -f nginx -o ../data/test_nginx.log
```

转换日志格式：

```bash
python convert_log_format.py ../data/test_nginx.log ../data/test_json.log --input-format nginx --output-format json
```

## 可用性审计

详见 `example/TEST_AUDIT.md`。

