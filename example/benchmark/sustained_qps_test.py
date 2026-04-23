#!/usr/bin/env python3
"""
持续吞吐率（Sustained QPS）测试脚本。

测试原理：
1. 清空服务端日志。
2. 以固定并发持续发送日志 N 秒。
3. 发送过程中每秒采样服务端实际入库数量，绘制实时 QPS 曲线。
4. 发送停止后，持续轮询服务端直到入库数不再增长（确认后端已处理完毕）。
5. 计算并输出明确的持续 QPS（服务端实际稳定入库速率）。

使用示例：
    python sustained_qps_test.py -protocol tcp -addr localhost:9000 -c 20 -d 60
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


def gen_log(i: int) -> str:
    ts = datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")
    ip = random.choice(IPS)
    method = random.choice(METHODS)
    path = random.choice(PATHS)
    code = random.choice(STATUS)
    size = random.randint(100, 10000)
    return f'{ip} - - [{ts}] "{method} {path} HTTP/1.1" {code} {size}'


def clear_logs() -> None:
    try:
        req = urllib.request.Request(f"{BASE}/api/logs", method="DELETE")
        urllib.request.urlopen(req, timeout=8).read()
    except Exception:
        pass


def get_server_total() -> int:
    try:
        req = urllib.request.Request(f"{BASE}/api/logs?limit=1", method="GET")
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return int(data.get("total", 0))
    except Exception:
        return 0


def get_server_status() -> dict:
    """获取服务端处理器和队列状态"""
    try:
        req = urllib.request.Request(f"{BASE}/api/status", method="GET")
        with urllib.request.urlopen(req, timeout=5) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except Exception:
        return {}


def sender_tcp(args, wid: int):
    host, port = args.addr.split(":")
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((host, int(port)))
    end_time = time.time() + args.d
    try:
        while time.time() < end_time:
            line = gen_log(wid)
            try:
                sock.sendall((line + "\n").encode("utf-8"))
            except Exception:
                break
    finally:
        sock.close()


def sender_udp(args, wid: int):
    host, port = args.addr.split(":")
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    end_time = time.time() + args.d
    try:
        while time.time() < end_time:
            line = gen_log(wid)
            try:
                sock.sendto(line.encode("utf-8"), (host, int(port)))
            except Exception:
                break
    finally:
        sock.close()


def sender_http(args, wid: int):
    url = f"http://{args.addr}/logs"
    end_time = time.time() + args.d
    while time.time() < end_time:
        lines = [gen_log(wid) for _ in range(args.batch)]
        body = "\n".join(lines).encode("utf-8")
        req = urllib.request.Request(url, data=body, method="POST", headers={"Content-Type": "text/plain"})
        try:
            urllib.request.urlopen(req, timeout=5)
        except Exception:
            pass


def sampler(stop: threading.Event, samples: list):
    """每秒采样服务端实际入库数量"""
    while not stop.is_set():
        total = get_server_total()
        samples.append((time.time(), total))
        time.sleep(1)


def wait_for_stability(timeout: int = 180) -> int:
    """等待服务端所有队列排空且入库数稳定，返回最终总数。

    检测逻辑：
    1. 先等待 input_queue / output_queue / overflow_pending 全部为空
    2. 再等待 total_count 连续稳定（防止数据库异步写入的尾延迟）
    """
    start = time.time()
    prev_total = get_server_total()
    stable_count = 0

    while time.time() - start < timeout:
        status = get_server_status()
        proc = status.get("processor", {})
        input_q = proc.get("input_queue_size", 0)
        output_q = proc.get("output_queue_size", 0)
        overflow = proc.get("overflow_queue_pending", 0)
        curr_total = get_server_total()

        # 第一阶段：等待所有队列排空
        if input_q > 0 or output_q > 0 or overflow > 0:
            print(
                f"  [等待排空] input={input_q}, output={output_q}, overflow={overflow}, total={curr_total}"
            )
            time.sleep(0.5)
            prev_total = curr_total
            stable_count = 0
            continue

        # 第二阶段：队列已空，检查 total_count 是否稳定
        if curr_total == prev_total:
            stable_count += 1
            if stable_count >= 3:
                print(f"  [排空完成] 队列全空，total={curr_total} 连续稳定")
                return curr_total
        else:
            if curr_total > prev_total:
                print(f"  [刷盘中] total={curr_total} (+{curr_total - prev_total})")
            stable_count = 0

        prev_total = curr_total
        time.sleep(1)

    print(f"  [警告] 排空超时，返回当前 total={curr_total}")
    return curr_total


def main() -> int:
    parser = argparse.ArgumentParser(description="持续吞吐率（Sustained QPS）测试")
    parser.add_argument("-protocol", choices=["tcp", "udp", "http"], default="tcp")
    parser.add_argument("-addr", default="localhost:9000")
    parser.add_argument("-c", type=int, default=20, help="并发数")
    parser.add_argument("-d", type=int, default=60, help="持续发送秒数")
    parser.add_argument("-batch", type=int, default=100, help="HTTP 批量大小")
    args = parser.parse_args()

    print("=" * 60)
    print("持续吞吐率（Sustained QPS）测试")
    print("=" * 60)
    print(f"协议: {args.protocol}, 地址: {args.addr}, 并发: {args.c}, 持续: {args.d}s")

    # 1. 清空日志
    clear_logs()
    time.sleep(1)
    before = get_server_total()
    print(f"[INFO] 服务端日志已清空，初始数量: {before}")

    # 2. 启动并发发送
    send_start = time.time()
    sender_map = {"tcp": sender_tcp, "udp": sender_udp, "http": sender_http}
    sender = sender_map[args.protocol]

    sampler_stop = threading.Event()
    samples = []
    sampler_thread = threading.Thread(target=sampler, args=(sampler_stop, samples), daemon=True)
    sampler_thread.start()

    with ThreadPoolExecutor(max_workers=args.c) as pool:
        futures = [pool.submit(sender, args, i) for i in range(args.c)]
        for f in futures:
            f.result()

    send_elapsed = time.time() - send_start
    print(f"[INFO] 发送阶段结束，耗时: {send_elapsed:.2f}s")

    # 发送结束后立即停止采样，只统计发送阶段的数据
    sampler_stop.set()
    sampler_thread.join(timeout=5)

    # 3. 等待后端完全处理
    print("[INFO] 等待后端完成刷盘和处理...")
    after_send = get_server_total()
    final_total = wait_for_stability()
    total_elapsed = time.time() - send_start
    drain_elapsed = total_elapsed - send_elapsed

    stored = max(0, final_total - before)
    backlog_at_send_end = max(0, final_total - after_send)

    # 4. 计算各项指标
    # 客户端发送速率（理论值）
    client_qps = stored / send_elapsed if send_elapsed > 0 else 0
    # 持续 QPS（包含处理等待时间的真实稳定速率）
    sustained_qps = stored / total_elapsed if total_elapsed > 0 else 0

    # 从采样数据计算峰值 QPS 和稳定期 QPS（仅基于发送阶段内的采样）
    peak_qps = 0
    stable_qps = 0
    if len(samples) >= 3:
        qps_list = []
        for i in range(1, len(samples)):
            dt = samples[i][0] - samples[i - 1][0]
            dq = samples[i][1] - samples[i - 1][1]
            if dt > 0:
                qps_list.append(dq / dt)
        if qps_list:
            # 去掉异常值（负数和过大值）
            filtered = [q for q in qps_list if 0 <= q <= max(qps_list) * 1.5]
            if filtered:
                peak_qps = max(filtered)
                # 稳定期 = 发送后半段的平均 QPS（排除开头的预热阶段）
                stable_start = max(1, len(filtered) * 2 // 3)
                stable_qps = sum(filtered[stable_start:]) / max(1, len(filtered) - stable_start)

    # 5. 输出报告
    print("\n" + "=" * 60)
    print("测试结果")
    print("=" * 60)
    print(f"  发送时长:           {send_elapsed:.2f}s")
    print(f"  后端排空等待:       {drain_elapsed:.2f}s")
    print(f"  总耗时:             {total_elapsed:.2f}s")
    print(f"  发送结束时积压数:   {backlog_at_send_end:,} 条")
    print(f"  服务端入库总数:     {stored:,} 条")
    print(f"  客户端发送速率:     {client_qps:,.0f} 条/秒")
    print(f"  峰值 QPS:           {peak_qps:,.0f} 条/秒")
    print(f"  稳定期 QPS:         {stable_qps:,.0f} 条/秒")
    print(f"  持续 QPS:           {sustained_qps:,.0f} 条/秒  <-- 论文推荐引用值")
    print("=" * 60)
    print("\n说明:")
    print("  - 客户端发送速率: 仅计算发送阶段的平均速率")
    print("  - 峰值 QPS:       发送过程中服务端每秒最高入库速率")
    print("  - 稳定期 QPS:     发送后半段的平均入库速率（反映系统稳定处理能力）")
    print("  - 持续 QPS:       总入库数 / 总耗时，反映系统端到端持续处理能力")
    print("  - 后端排空等待:   发送结束后到所有日志入库完成的等待时间")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
