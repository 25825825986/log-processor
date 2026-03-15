#!/usr/bin/env python3
"""
日志处理服务运行状态诊断工具。
"""

from __future__ import annotations

import argparse
import json
import time
import urllib.request

BASE = "http://localhost:8080"


def get_json(path: str, timeout: int = 5):
    req = urllib.request.Request(f"{BASE}{path}", method="GET")
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8"))


def fetch_status():
    try:
        return get_json("/api/status")
    except Exception as exc:
        print(f"[ERROR] 获取 /api/status 失败: {exc}")
        return None


def fetch_total() -> int:
    try:
        data = get_json("/api/logs?limit=1")
        return int(data.get("total", 0))
    except Exception:
        return 0


def diagnose_once() -> int:
    status = fetch_status()
    if not status:
        print("[FAIL] 无法连接服务器: http://localhost:8080")
        return 1

    processor = status.get("processor", {})
    total = fetch_total()

    print("=" * 60)
    print("系统诊断")
    print("=" * 60)
    print(f"日志总数:            {total:,}")
    print(f"输入队列长度:        {processor.get('input_queue_size', 'N/A')}")
    print(f"输出队列长度:        {processor.get('output_queue_size', 'N/A')}")
    print(f"Worker 数量:         {processor.get('worker_count', 'N/A')}")
    print(f"批大小:              {processor.get('batch_size', 'N/A')}")
    print(f"接收计数:            {processor.get('received_count', 'N/A')}")
    print(f"处理计数:            {processor.get('processed_count', 'N/A')}")
    print(f"丢弃计数:            {processor.get('dropped_count', 'N/A')}")
    print(f"解析错误计数:        {processor.get('parse_error_count', 'N/A')}")
    return 0


def monitor_loop():
    print("每 1 秒监控一次（Ctrl+C 停止）")
    print(f"{'time':<10}{'total':>12}{'qps':>8}{'in_q':>8}{'out_q':>8}{'drop':>10}")

    prev_total = fetch_total()
    prev_ts = time.time()

    while True:
        now = time.time()
        total = fetch_total()
        delta = max(now - prev_ts, 1e-6)
        qps = (total - prev_total) / delta

        status = fetch_status() or {}
        processor = status.get("processor", {})

        print(
            f"{time.strftime('%H:%M:%S'):<10}"
            f"{total:>12,}"
            f"{qps:>8.0f}"
            f"{int(processor.get('input_queue_size', 0)):>8}"
            f"{int(processor.get('output_queue_size', 0)):>8}"
            f"{int(processor.get('dropped_count', 0)):>10}"
        )

        prev_total = total
        prev_ts = now
        time.sleep(1)


def main() -> int:
    parser = argparse.ArgumentParser(description="诊断运行中的日志处理服务。")
    parser.add_argument("--once", action="store_true", help="输出一次后退出")
    args = parser.parse_args()

    if args.once:
        return diagnose_once()

    try:
        monitor_loop()
    except KeyboardInterrupt:
        print("\n已停止。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
