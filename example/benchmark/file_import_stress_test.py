#!/usr/bin/env python3
"""
文件导入最大压力测试：
通过 /api/logs/import 上传大文件，测试文件导入链路的极限吞吐与稳定性。
"""

from __future__ import annotations

import argparse
import json
import os
import tempfile
import threading
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from pathlib import Path

BASE = "http://localhost:8080"


def gen_log_line(i: int) -> str:
    ts = datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")
    return f'127.0.0.1 - - [{ts}] "GET /api/stress/{i % 10000} HTTP/1.1" 200 {100 + (i % 9900)}'


def build_log_file(path: str, lines: int) -> None:
    with open(path, "w", encoding="utf-8") as f:
        for i in range(lines):
            f.write(gen_log_line(i) + "\n")


def get_total() -> int:
    try:
        req = urllib.request.Request(f"{BASE}/api/logs?limit=1", method="GET")
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return int(data.get("total", 0))
    except Exception:
        return 0


def clear_logs() -> None:
    try:
        req = urllib.request.Request(f"{BASE}/api/logs", method="DELETE")
        urllib.request.urlopen(req, timeout=10).read()
    except Exception:
        pass


def upload_file(file_path: str, import_id: str) -> dict:
    boundary = "----WebKitFormBoundary" + hex(int(time.time() * 1000))[2:]
    with open(file_path, "rb") as f:
        file_data = f.read()

    file_name = os.path.basename(file_path)
    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="file"; filename="{file_name}"\r\n'
        f"Content-Type: text/plain\r\n\r\n"
    ).encode("utf-8")
    body += file_data
    body += f"\r\n--{boundary}\r\n".encode("utf-8")
    body += (
        f'Content-Disposition: form-data; name="import_id"\r\n\r\n'
        f"{import_id}\r\n"
    ).encode("utf-8")
    body += f"--{boundary}--\r\n".encode("utf-8")

    req = urllib.request.Request(
        f"{BASE}/api/logs/import",
        data=body,
        method="POST",
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
    )
    with urllib.request.urlopen(req, timeout=300) as resp:
        return json.loads(resp.read().decode("utf-8"))


def monitor_import(import_id: str, stop: threading.Event) -> None:
    while not stop.is_set():
        try:
            req = urllib.request.Request(f"{BASE}/api/logs/import/progress?id={import_id}", method="GET")
            with urllib.request.urlopen(req, timeout=5) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                phase = data.get("phase", "unknown")
                parsed = data.get("parsed_count", 0)
                written = data.get("written_count", 0)
                target = data.get("target_count", 0)
                print(f"  [{import_id}] phase={phase} parsed={parsed} written={written} target={target}")
        except Exception as exc:
            print(f"  [{import_id}] monitor error: {exc}")
        time.sleep(1)


class ImportStats:
    def __init__(self):
        self.ok = 0
        self.fail = 0
        self.total_lines = 0
        self.total_imported = 0
        self.lock = threading.Lock()
        self.start = time.time()

    def add(self, ok: bool, lines: int, imported: int):
        with self.lock:
            if ok:
                self.ok += 1
            else:
                self.fail += 1
            self.total_lines += lines
            self.total_imported += imported

    def snapshot(self) -> tuple:
        with self.lock:
            return self.ok, self.fail, self.total_lines, self.total_imported


def worker(args, file_path: str, wid: int, stats: ImportStats):
    import_id = f"stress_file_{wid}_{int(time.time()*1000)}"
    stop_monitor = threading.Event()
    monitor_thread = threading.Thread(target=monitor_import, args=(import_id, stop_monitor), daemon=True)
    monitor_thread.start()

    try:
        result = upload_file(file_path, import_id)
        ok = result.get("status") in ("ok", "partial", "warning")
        lines = result.get("lines", 0)
        imported = result.get("imported", 0)
        stats.add(ok, lines, imported)
        print(f"[Worker {wid}] result: {result.get('status')} lines={lines} imported={imported}")
    except Exception as exc:
        stats.add(False, 0, 0)
        print(f"[Worker {wid}] upload failed: {exc}")
    finally:
        time.sleep(2)
        stop_monitor.set()
        monitor_thread.join(timeout=3)


def main() -> int:
    parser = argparse.ArgumentParser(description="文件导入最大压力测试")
    parser.add_argument("-lines", type=int, default=50000, help="每个文件行数")
    parser.add_argument("-files", type=int, default=5, help="并发上传文件数")
    parser.add_argument("-clear", action="store_true", help="测试前清空日志")
    args = parser.parse_args()

    print("=" * 60)
    print("文件导入最大压力测试")
    print("=" * 60)
    print(f"每文件行数: {args.lines}, 并发文件数: {args.files}")

    if args.clear:
        print("[INFO] 清空服务端日志...")
        clear_logs()
        time.sleep(1)

    before = get_total()

    with tempfile.TemporaryDirectory() as tmpdir:
        file_path = os.path.join(tmpdir, "stress_import.log")
        print(f"[INFO] 生成测试文件: {file_path} ({args.lines} 行)")
        build_log_file(file_path, args.lines)
        file_size = os.path.getsize(file_path)
        print(f"[INFO] 文件大小: {file_size / 1024 / 1024:.2f} MB")

        stats = ImportStats()

        print("[INFO] 开始并发上传...")
        with ThreadPoolExecutor(max_workers=args.files) as pool:
            futures = [pool.submit(worker, args, file_path, i, stats) for i in range(args.files)]
            for fut in futures:
                fut.result()

    elapsed = max(time.time() - stats.start, 1e-6)
    ok, fail, total_lines, total_imported = stats.snapshot()

    print("\n" + "=" * 60)
    print("客户端结果")
    print("=" * 60)
    print(f"elapsed:       {elapsed:.2f}s")
    print(f"success:       {ok}")
    print(f"failed:        {fail}")
    print(f"total_lines:   {total_lines}")
    print(f"total_imported:{total_imported}")

    wait_s = max(5, min(60, int(total_imported / 1000)))
    print(f"\n[INFO] 等待 {wait_s}s 让后端完成刷盘...")
    time.sleep(wait_s)

    after = get_total()
    stored = max(0, after - before)
    ratio = (stored / total_imported * 100.0) if total_imported > 0 else 0.0

    print("=" * 60)
    print("服务端校验")
    print("=" * 60)
    print(f"stored_added:  {stored}")
    print(f"store_ratio:   {ratio:.1f}%")
    print(f"import_tps:    {total_imported / elapsed:.0f} 条/秒")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
