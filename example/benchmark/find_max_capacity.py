#!/usr/bin/env python3
"""
估算 TCP/UDP 接收器可稳定承载的最大速率。
"""

from __future__ import annotations

import argparse
import json
import socket
import threading
import time
import urllib.request

BASE = "http://localhost:8080"


def clear_logs() -> None:
    try:
        req = urllib.request.Request(f"{BASE}/api/logs", method="DELETE")
        urllib.request.urlopen(req, timeout=10).read()
    except Exception:
        pass


def get_total() -> int:
    try:
        req = urllib.request.Request(f"{BASE}/api/logs?limit=1", method="GET")
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return int(data.get("total", 0))
    except Exception:
        return 0


def build_log_line(i: int) -> str:
    ts = time.strftime("%d/%b/%Y:%H:%M:%S %z")
    return f'127.0.0.1 - - [{ts}] "GET /api/capacity/{i % 1000} HTTP/1.1" 200 {100 + (i % 8000)}'


def run_sender(protocol: str, addr: str, total: int, concurrency: int, rate_per_conn: int):
    sent = 0
    failed = 0
    sent_lock = threading.Lock()
    host, port = addr.split(":")
    port_i = int(port)
    stop = threading.Event()

    def worker(worker_id: int):
        nonlocal sent, failed
        sock = None
        try:
            if protocol == "tcp":
                sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                sock.connect((host, port_i))
            else:
                sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

            while not stop.is_set():
                with sent_lock:
                    if sent >= total:
                        break
                    idx = sent
                    sent += 1
                if rate_per_conn > 0:
                    time.sleep(1.0 / rate_per_conn)
                line = build_log_line(idx).encode("utf-8")
                try:
                    if protocol == "tcp":
                        assert sock is not None
                        sock.sendall(line + b"\n")
                    else:
                        assert sock is not None
                        sock.sendto(line, (host, port_i))
                except Exception:
                    with sent_lock:
                        failed += 1
        finally:
            if sock is not None:
                sock.close()

    threads = [threading.Thread(target=worker, args=(i,), daemon=True) for i in range(concurrency)]
    start = time.time()
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    elapsed = max(time.time() - start, 1e-6)
    stop.set()
    return sent, failed, elapsed


def main() -> int:
    parser = argparse.ArgumentParser(description="探测接收器稳定容量。")
    parser.add_argument("-protocol", choices=["tcp", "udp"], default="tcp")
    parser.add_argument("-addr", default="localhost:9000")
    parser.add_argument("-c", type=int, default=30, help="并发数")
    parser.add_argument("-test_each", type=int, default=20000, help="每档速率发送日志数")
    parser.add_argument("-max_rate", type=int, default=10000)
    args = parser.parse_args()

    rates = [500, 1000, 1500, 2000, 3000, 5000, 8000, 10000]
    best = 0
    best_store_rate = 0.0

    print("=" * 60)
    print("容量探测")
    print("=" * 60)
    print(f"protocol={args.protocol} addr={args.addr} concurrency={args.c}")

    for rate in rates:
        if rate > args.max_rate:
            break
        rate_per_conn = max(1, rate // max(args.c, 1))
        print(f"\n[TEST] target={rate} qps, per_conn={rate_per_conn}")

        clear_logs()
        before = get_total()
        sent, failed, elapsed = run_sender(
            protocol=args.protocol,
            addr=args.addr,
            total=args.test_each,
            concurrency=args.c,
            rate_per_conn=rate_per_conn,
        )

        # 等待后端刷盘和批处理完成
        time.sleep(5)
        after = get_total()

        stored = max(0, after - before)
        send_rate = sent / elapsed
        store_rate = stored / elapsed
        success = (stored / sent * 100.0) if sent else 0.0

        print(f"sent={sent} failed={failed} elapsed={elapsed:.2f}s")
        print(f"send_rate={send_rate:.0f} qps stored={stored} success={success:.1f}%")

        if success >= 95.0:
            best = rate
            best_store_rate = store_rate
        else:
            print("[STOP] 成功率低于 95%，停止探测。")
            break

    print("\n" + "=" * 60)
    if best > 0:
        print(f"稳定目标速率: {best} qps")
        print(f"观测存储速率: {best_store_rate:.0f} qps")
    else:
        print("在测试范围内未找到稳定速率。")
    print("=" * 60)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
