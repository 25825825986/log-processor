# Example 测试目录审计

## 保留的测试

- `example/benchmark/stress_test.py`：实时导入最大压力测试（TCP/UDP/HTTP 压测发送器）
- `example/benchmark/file_import_stress_test.py`：文件导入最大压力测试（通过 `/api/logs/import` 并发上传大文件）

## 说明

根据系统代码结构，仅保留与实时导入和文件导入相关的最大压力测试，其余测试已清理。
