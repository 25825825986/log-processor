# Example 测试目录审计（2026-03-15）

## 1. 可用性结论

### 可直接使用
- `example/tools/send_logs_windows.ps1`
- `example/tools/send_logs_unix.sh`
- `example/tools/generate_test_logs.py`（已修复）
- `example/tools/convert_log_format.py`（已修复）
- `example/benchmark/stress_test.py`（已修复）
- `example/benchmark/find_max_capacity.py`（已修复）
- `example/benchmark/diagnose_system.py`（已修复）
- `example/benchmark/download_nasa_logs.py`（已修复）
- `example/benchmark/generate_nasa_like_logs.py`（已修复）
- `example/benchmark/test_resilient.py`（补充新增）
- `example/benchmark/stress_test.go`
- `example/benchmark/api_smoke_test_test.go`（补充新增）

### 说明
- 原有多个 Python 脚本存在字符串损坏导致的语法错误，已统一重写为可执行版本。
- `api_smoke_test_test.go` 是“接口冒烟测试”，需要服务启动后运行；服务未启动时会 `Skip`，不会导致 `go test` 失败。

## 2. 发现并补齐的缺失测试

- 缺失 `test_resilient.py`：已补充，覆盖高压下队列/丢弃观测流程。
- 缺少自动化接口可达性检查：已补充 `api_smoke_test_test.go`，覆盖：
  - `/api/status`
  - `/api/config`
  - `/api/storage/info`
  - `/api/export/formats`
  - `/api/logs?limit=1`

## 3. 仍建议后续补充

- 配置变更回归测试（`POST /api/config` 的局部更新与运行时生效验证）。
- 接收器启停行为测试（`/api/receiver/start`、`/api/receiver/stop`）。
- 导入文件格式兼容性回归测试（nginx/json/csv/syslog 等）。

