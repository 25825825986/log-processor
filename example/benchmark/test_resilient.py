#!/usr/bin/env python3
"""
韧性专项测试：
1) 以高于可持续吞吐的速率发送日志
2) 通过 /api/status 观察队列压力
3) 输出最终入库比例
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import time
import urllib.request

BASE = "http://localhost:8080"


def get_json(path: str):
    req = urllib.request.Request(f"{BASE}{path}", method="GET")
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))


def monitor(wait_s: int, interval: int) -> None:
    print(f"\n[MONITOR] 监控 {wait_s}s")
    end = time.time() + wait_s
    while time.time() < end:
        try:
            status = get_json("/api/status")
            proc = status.get("processor", {})
            print(
                f"  in_q={proc.get('input_queue_size', 0):>6} "
                f"out_q={proc.get('output_queue_size', 0):>6} "
                f"dropped={proc.get('dropped_count', 0):>8}"
            )
        except Exception as exc:
            print(f"  [WARN] 读取状态失败: {exc}")
        time.sleep(interval)


def main() -> int:
    parser = argparse.ArgumentParser(description="执行韧性场景测试。")
    parser.add_argument("--quick", action="store_true", help="快速模式")
    parser.add_argument("--count", type=int, default=50000, help="总发送日志数")
    parser.add_argument("--rate", type=int, default=100, help="每个 worker 发送速率")
    parser.add_argument("--concurrency", type=int, default=40, help="发送 worker 数")
    parser.add_argument("--wait", type=int, default=30, help="发送后监控秒数")
    args = parser.parse_args()

    if args.quick:
        args.count = 10000
        args.rate = 40
        args.concurrency = 20
        args.wait = 15

    cmd = [
        sys.executable,
        "stress_test.py",
        "-protocol",
        "tcp",
        "-addr",
        "localhost:9000",
        "-total",
        str(args.count),
        "-c",
        str(args.concurrency),
        "-rate",
        str(args.rate),
    ]

    print("=" * 60)
    print("韧性测试")
    print("=" * 60)
    print("执行命令:", " ".join(cmd))

    proc = subprocess.Popen(cmd, cwd=".", stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    assert proc.stdout is not None
    for line in proc.stdout:
        print(line.rstrip())

    rc = proc.wait()
    if rc != 0:
        print(f"[FAIL] stress_test.py 退出码: {rc}")
        return rc

    monitor(args.wait, interval=2)
    print("[DONE] 韧性场景测试完成")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
