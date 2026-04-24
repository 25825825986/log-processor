#!/usr/bin/env python3
"""
突发吞吐量（Burst QPS）测试脚本。

突发吞吐量定义：
    系统在短时间内（通常 1~5 秒）吸收并处理流量峰值的能力，
    反映的是队列缓冲 + 批量写入加速后的瞬时最高速率。

与持续吞吐量的区别：
    - 持续吞吐量：长期稳定运行时的平均处理速率（受限于 SQLite 单线程写入）
    - 突发吞吐量：短时间脉冲注入时，利用队列缓冲和批量攒批达到的峰值速率

测试原理：
    1. 清空服务端日志。
    2. 以极高并发（如 50~100）在短时间内（如 3 秒）脉冲式注入日志，
       让 inputChan / outputChan 堆积大量数据。
    3. 发送阶段内高频采样（每 200~500 ms）服务端入库数量。
    4. 计算每个采样间隔的瞬时 QPS，最高值即为突发吞吐量。

注意：
    由于后端采用批量提交（batch_size=1000），total_count 的更新呈"阶梯式"跳变。
    因此突发吞吐量的测量精度受限于 batch 大小。如需更高精度，可临时调小
    batch_timeout（如改为 100ms），测试完成后再改回。

使用示例：
    python burst_qps_test.py -protocol tcp -addr localhost:9000 -c 50 -d 3
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


def high_freq_sampler(stop: threading.Event, interval: float, samples: list):
    """高频采样服务端实际入库数量"""
    while not stop.is_set():
        total = get_server_total()
        samples.append((time.time(), total))
        time.sleep(interval)


def main() -> int:
    parser = argparse.ArgumentParser(description="突发吞吐量（Burst QPS）测试")
    parser.add_argument("-protocol", choices=["tcp", "udp", "http"], default="tcp")
    parser.add_argument("-addr", default="localhost:9000")
    parser.add_argument("-c", type=int, default=50, help="并发数（建议 30~100）")
    parser.add_argument("-d", type=int, default=3, help="脉冲发送秒数（建议 2~5）")
    parser.add_argument("-batch", type=int, default=100, help="HTTP 批量大小")
    parser.add_argument("-interval", type=float, default=0.2, help="采样间隔（秒，默认0.2）")
    args = parser.parse_args()

    print("=" * 60)
    print("突发吞吐量（Burst QPS）测试")
    print("=" * 60)
    print(f"协议: {args.protocol}, 地址: {args.addr}")
    print(f"并发: {args.c}, 脉冲时长: {args.d}s, 采样间隔: {args.interval}s")

    # 1. 清空日志
    clear_logs()
    time.sleep(1)
    before = get_server_total()
    print(f"[INFO] 服务端日志已清空，初始数量: {before}")

    # 2. 启动高频采样
    sampler_stop = threading.Event()
    samples = []
    sampler_thread = threading.Thread(
        target=high_freq_sampler,
        args=(sampler_stop, args.interval, samples),
        daemon=True,
    )
    sampler_thread.start()

    # 3. 启动高并发脉冲发送
    send_start = time.time()
    sender_map = {"tcp": sender_tcp, "udp": sender_udp, "http": sender_http}
    sender = sender_map[args.protocol]

    with ThreadPoolExecutor(max_workers=args.c) as pool:
        futures = [pool.submit(sender, args, i) for i in range(args.c)]
        for f in futures:
            f.result()

    send_elapsed = time.time() - send_start
    print(f"[INFO] 脉冲发送结束，耗时: {send_elapsed:.2f}s")

    # 发送结束后继续采样一段时间（捕捉排空阶段的峰值）
    time.sleep(args.d * 2)
    sampler_stop.set()
    sampler_thread.join(timeout=5)

    # 4. 等待完全排空（确保最终总数准确）
    print("[INFO] 等待后端完全排空...")
    prev = get_server_total()
    stable = 0
    while stable < 3:
        time.sleep(1)
        curr = get_server_total()
        if curr == prev:
            stable += 1
        else:
            stable = 0
            print(f"  [刷盘中] total={curr} (+{curr - prev})")
        prev = curr

    final_total = prev
    stored = max(0, final_total - before)

    # 5. 计算突发吞吐量
    # 从采样数据中计算每个间隔的瞬时 QPS
    qps_list = []
    for i in range(1, len(samples)):
        dt = samples[i][0] - samples[i - 1][0]
        dq = samples[i][1] - samples[i - 1][1]
        if dt > 0.05:  # 忽略异常间隔
            qps = dq / dt
            qps_list.append((samples[i][0], qps))

    burst_qps = 0
    burst_time = 0
    if qps_list:
        burst_qps, burst_time = max(qps_list, key=lambda x: x[1])

    # 计算发送阶段内的平均速率
    send_phase_samples = [s for s in samples if s[0] <= send_start + send_elapsed + 0.5]
    if len(send_phase_samples) >= 2:
        send_phase_qps = (send_phase_samples[-1][1] - send_phase_samples[0][1]) / (
            send_phase_samples[-1][0] - send_phase_samples[0][0]
        )
    else:
        send_phase_qps = 0

    # 6. 输出报告
    print("\n" + "=" * 60)
    print("测试结果")
    print("=" * 60)
    print(f"  脉冲发送时长:     {send_elapsed:.2f}s")
    print(f"  服务端入库总数:   {stored:,} 条")
    print(f"  发送阶段平均速率: {send_phase_qps:,.0f} 条/秒")
    print(f"  突发吞吐量:       {burst_qps:,.0f} 条/秒  <-- 峰值，发生在 t={burst_time - send_start:.1f}s")
    print("=" * 60)
    print("\n说明:")
    print("  - 突发吞吐量是高频采样中观察到的最高瞬时入库速率")
    print("  - 该值通常高于持续吞吐量，因为批量写入在队列积压时更高效")
    print("  - 若数值呈'阶梯式'跳变（如 0→1000→2000），说明受 batch_size 限制")
    print(f"  - 当前采样点数: {len(samples)}, 有效 QPS 数据点: {len(qps_list)}")

    # 可选：打印前10个最高 QPS 数据点
    if qps_list:
        print("\n  前10个最高瞬时 QPS 采样点:")
        top10 = sorted(qps_list, key=lambda x: x[1], reverse=True)[:10]
        for t, q in top10:
            print(f"    t={t - send_start:5.1f}s  QPS={q:8.0f}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
