# Example 测试工具目录

本目录用于本项目的手工测试、压测和数据准备。

## 目录结构

- `benchmark/`
  - `stress_test.py`：实时导入最大压力测试（TCP/UDP/HTTP 压测发送器）
  - `file_import_stress_test.py`：文件导入最大压力测试（通过 `/api/logs/import` 并发上传大文件）
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

1. 实时导入最大压力测试（TCP）

```bash
cd example/benchmark
python stress_test.py -protocol tcp -addr localhost:9000 -total 20000 -c 20 -rate 40
```

2. 文件导入最大压力测试

```bash
cd example/benchmark
python file_import_stress_test.py -lines 50000 -files 5 -clear
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
