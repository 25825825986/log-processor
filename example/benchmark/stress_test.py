#!/usr/bin/env python3
"""
TCP/UDP/HTTP 接收器压测发送工具。
"""

from __future__ import annotations

import argparse
import json
import random
import socket
import threading
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from pathlib import Path

BASE = "http://localhost:8080"

PATHS = ["/", "/api/users", "/api/orders", "/api/login", "/health", "/metrics"]
METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"]
STATUS = [200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503]
IPS = ["127.0.0.1", "10.0.0.12", "192.168.1.10", "172.16.1.8", "203.0.113.5"]


class Stats:
    def __init__(self) -> None:
        self.start = time.time()
        self.sent = 0
        self.failed = 0
        self.lock = threading.Lock()

    def reserve(self, n: int, total_limit: int, duration: int) -> int:
        with self.lock:
            if duration > 0:
                if time.time() - self.start > duration:
                    return 0
                self.sent += n
                return n
            remaining = total_limit - self.sent
            if remaining <= 0:
                return 0
            take = min(remaining, n)
            self.sent += take
            return take

    def fail(self, n: int) -> None:
        with self.lock:
            self.failed += n
            self.sent -= n

    def snapshot(self) -> tuple[int, int]:
        with self.lock:
            return self.sent, self.failed


def clear_server_logs() -> None:
    req = urllib.request.Request(f"{BASE}/api/logs", method="DELETE")
    urllib.request.urlopen(req, timeout=8).read()


def get_server_total() -> int:
    req = urllib.request.Request(f"{BASE}/api/logs?limit=1", method="GET")
    with urllib.request.urlopen(req, timeout=5) as resp:
        data = json.loads(resp.read().decode("utf-8"))
        return int(data.get("total", 0))


def gen_log(i: int) -> str:
    ts = datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")
    ip = random.choice(IPS)
    method = random.choice(METHODS)
    path = random.choice(PATHS)
    code = random.choice(STATUS)
    size = random.randint(100, 10000)
    return f'{ip} - - [{ts}] "{method} {path} HTTP/1.1" {code} {size}'


class FileSource:
    def __init__(self, path: str) -> None:
        p = Path(path)
        if not p.exists():
            raise FileNotFoundError(path)
        self.lines = [ln.strip() for ln in p.read_text(encoding="utf-8", errors="ignore").splitlines() if ln.strip()]
        if not self.lines:
            raise ValueError("empty log file")
        self.idx = 0
        self.lock = threading.Lock()

    def take(self, n: int) -> list[str]:
        out: list[str] = []
        with self.lock:
            for _ in range(n):
                out.append(self.lines[self.idx])
                self.idx = (self.idx + 1) % len(self.lines)
        return out


def sender_tcp(args, stats: Stats, wid: int, source: FileSource | None):
    host, port = args.addr.split(":")
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((host, int(port)))
    try:
        while True:
            take = stats.reserve(1, args.total, args.duration)
            if take == 0:
                return
            if args.rate > 0:
                time.sleep(1.0 / args.rate)
            line = source.take(1)[0] if source else gen_log(wid)
            try:
                sock.sendall((line + "\n").encode("utf-8"))
            except Exception:
                stats.fail(1)
                return
    finally:
        sock.close()


def sender_udp(args, stats: Stats, wid: int, source: FileSource | None):
    host, port = args.addr.split(":")
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try:
        while True:
            take = stats.reserve(1, args.total, args.duration)
            if take == 0:
                return
            if args.rate > 0:
                time.sleep(1.0 / args.rate)
            line = source.take(1)[0] if source else gen_log(wid)
            try:
                sock.sendto(line.encode("utf-8"), (host, int(port)))
            except Exception:
                stats.fail(1)
    finally:
        sock.close()


def sender_http(args, stats: Stats, wid: int, source: FileSource | None):
    url = f"http://{args.addr}/logs"
    while True:
        batch = stats.reserve(args.batch, args.total, args.duration)
        if batch == 0:
            return
        if args.rate > 0:
            time.sleep(1.0 / args.rate)
        lines = source.take(batch) if source else [gen_log(wid) for _ in range(batch)]
        body = "\n".join(lines).encode("utf-8")
        req = urllib.request.Request(url, data=body, method="POST", headers={"Content-Type": "text/plain"})
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                if resp.status != 200:
                    stats.fail(batch)
        except Exception:
            stats.fail(batch)


def progress_loop(stats: Stats, stop: threading.Event):
    prev_sent = 0
    while not stop.is_set():
        time.sleep(1)
        sent, failed = stats.snapshot()
        qps = sent - prev_sent
        elapsed = max(time.time() - stats.start, 1e-6)
        avg = sent / elapsed
        print(
            f"\r[QPS:{qps:6d}] [Sent:{sent:8d}] [Avg:{avg:7.0f}/s] [Failed:{failed:6d}]",
            end="",
            flush=True,
        )
        prev_sent = sent
    print()


def main() -> int:
    parser = argparse.ArgumentParser(description="日志处理系统压测发送器")
    parser.add_argument("-protocol", choices=["tcp", "udp", "http"], default="tcp")
    parser.add_argument("-addr", default="localhost:9000")
    parser.add_argument("-total", type=int, default=10000)
    parser.add_argument("-c", type=int, default=10, help="并发数")
    parser.add_argument("-d", "-duration", dest="duration", type=int, default=0, help="持续秒数")
    parser.add_argument("-rate", type=int, default=0, help="每个 worker 每秒发送条数")
    parser.add_argument("-batch", type=int, default=100, help="HTTP 批量发送大小")
    parser.add_argument("-file", type=str, default=None, help="从文件循环读取日志")
    parser.add_argument("-no-clear", action="store_true")
    args = parser.parse_args()

    source = FileSource(args.file) if args.file else None

    print("=" * 60)
    print("压测执行")
    print("=" * 60)
    print(f"protocol={args.protocol} addr={args.addr} concurrency={args.c}")
    if args.duration > 0:
        print(f"duration={args.duration}s")
    else:
        print(f"total={args.total}")
    if source:
        print(f"source_file={args.file} lines={len(source.lines)}")

    try:
        before = get_server_total()
    except Exception:
        before = 0

    if not args.no_clear:
        try:
            clear_server_logs()
            time.sleep(0.5)
            before = get_server_total()
            print("[INFO] 服务端日志已清空")
        except Exception as exc:
            print(f"[WARN] 清空服务端日志失败: {exc}")

    stats = Stats()
    reporter_stop = threading.Event()
    reporter = threading.Thread(target=progress_loop, args=(stats, reporter_stop), daemon=True)
    reporter.start()

    sender = {
        "tcp": sender_tcp,
        "udp": sender_udp,
        "http": sender_http,
    }[args.protocol]

    with ThreadPoolExecutor(max_workers=args.c) as pool:
        futures = [pool.submit(sender, args, stats, i, source) for i in range(args.c)]
        for fut in futures:
            fut.result()

    reporter_stop.set()
    reporter.join()

    elapsed = max(time.time() - stats.start, 1e-6)
    sent, failed = stats.snapshot()
    print("\n" + "=" * 60)
    print("客户端结果")
    print("=" * 60)
    print(f"elapsed:   {elapsed:.2f}s")
    print(f"sent:      {sent:,}")
    print(f"failed:    {failed:,}")
    print(f"avg_qps:   {sent / elapsed:.0f}")

    wait_s = max(5, min(60, int(sent / 500)))
    print(f"\n[INFO] 等待 {wait_s}s 让后端刷盘...")
    time.sleep(wait_s)
    try:
        after = get_server_total()
        stored = max(0, after - before)
        ratio = (stored / sent * 100.0) if sent > 0 else 0.0
        print("=" * 60)
        print("服务端校验")
        print("=" * 60)
        print(f"stored_added: {stored:,}")
        print(f"store_ratio:  {ratio:.1f}%")
    except Exception as exc:
        print(f"[WARN] 校验服务端计数失败: {exc}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
