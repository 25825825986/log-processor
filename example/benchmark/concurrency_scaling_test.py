#!/usr/bin/env python3
"""
并发量递增性能测试脚本（Concurrency Scaling Test）。

测试目的：
    逐步增加注入并发量，测量系统在不同负载下的持续吞吐量和队列积压情况，
    找到系统性能拐点（即吞吐量不再随并发增加而提升的临界点）。

测试流程：
    1. 清空服务端日志。
    2. 从起始并发开始，按指定步长逐步增加到最大并发。
    3. 每个并发级别下持续发送 N 秒，高频采样服务端入库数。
    4. 发送结束后等待队列完全排空（通过 /api/status 检测）。
    5. 记录该并发级别下的各项指标。
    6. 全部阶段完成后输出汇总表格，并可选择绘制性能曲线图。

输出指标（每并发级别）：
    - 客户端发送速率（条/秒）
    - 服务端持续 QPS（条/秒）
    - 峰值 QPS（条/秒）
    - 后端排空等待时间（秒）
    - 最终入库总数
    - 最大队列积压（input + output）

使用示例：
    # 指数递增：1, 2, 4, 8, 16, 32, 64 并发，每阶段 20 秒
    python concurrency_scaling_test.py -protocol tcp -addr localhost:9000 -start 1 -max 64 -mode exp -d 20

    # 线性递增：5, 10, 15, ... 50 并发，每阶段 30 秒
    python concurrency_scaling_test.py -protocol tcp -start 5 -max 50 -mode linear -step 5 -d 30

    # 生成图表并保存 CSV
    python concurrency_scaling_test.py -max 64 -d 20 -plot -output results.csv
"""

from __future__ import annotations

import argparse
import csv
import json
import random
import socket
import threading
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from pathlib import Path

try:
    import psutil
    HAS_PSUTIL = True
except ImportError:
    HAS_PSUTIL = False

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
    try:
        req = urllib.request.Request(f"{BASE}/api/status", method="GET")
        with urllib.request.urlopen(req, timeout=5) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except Exception:
        return {}


def sender_tcp(args, wid: int, duration: int):
    host, port = args.addr.split(":")
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((host, int(port)))
    end_time = time.time() + duration
    interval = 1.0 / args.rate if args.rate > 0 else 0
    next_send = time.time()
    try:
        while time.time() < end_time:
            if interval > 0:
                now = time.time()
                if next_send > now:
                    time.sleep(next_send - now)
                next_send += interval
            line = gen_log(wid)
            try:
                sock.sendall((line + "\n").encode("utf-8"))
            except Exception:
                break
    finally:
        sock.close()


def sender_udp(args, wid: int, duration: int):
    host, port = args.addr.split(":")
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    end_time = time.time() + duration
    interval = 1.0 / args.rate if args.rate > 0 else 0
    next_send = time.time()
    try:
        while time.time() < end_time:
            if interval > 0:
                now = time.time()
                if next_send > now:
                    time.sleep(next_send - now)
                next_send += interval
            line = gen_log(wid)
            try:
                sock.sendto(line.encode("utf-8"), (host, int(port)))
            except Exception:
                break
    finally:
        sock.close()


def sender_http(args, wid: int, duration: int):
    url = f"http://{args.addr}/logs"
    end_time = time.time() + duration
    interval = 1.0 / args.rate if args.rate > 0 else 0
    next_send = time.time()
    while time.time() < end_time:
        if interval > 0:
            now = time.time()
            if next_send > now:
                time.sleep(next_send - now)
            next_send += interval
        lines = [gen_log(wid) for _ in range(args.batch)]
        body = "\n".join(lines).encode("utf-8")
        req = urllib.request.Request(url, data=body, method="POST", headers={"Content-Type": "text/plain"})
        try:
            urllib.request.urlopen(req, timeout=5)
        except Exception:
            pass


def high_freq_sampler(stop: threading.Event, interval: float, samples: list):
    while not stop.is_set():
        total = get_server_total()
        samples.append((time.time(), total))
        time.sleep(interval)


def find_server_process():
    """查找 log-processor 服务端进程（优先按端口，再按进程名）"""
    if not HAS_PSUTIL:
        return None
    # 要排除的进程名（避免把 Python/Kimi 自身或无关进程匹配进来）
    EXCLUDE_NAMES = {'python.exe', 'python', 'python3', 'kimi.exe', 'kimi',
                     'cmd.exe', 'powershell.exe', 'pwsh.exe', 'explorer.exe'}
    # 方法1：通过 HTTP 端口 8080 查找（最可靠）
    for conn in psutil.net_connections():
        if conn.laddr.port == 8080 and conn.pid:
            try:
                p = psutil.Process(conn.pid)
                if p.name().lower() not in EXCLUDE_NAMES:
                    return p
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                continue
    # 方法2：进程名匹配（fallback）
    for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
        try:
            name = (proc.info['name'] or '').lower()
            if name in EXCLUDE_NAMES:
                continue
            cmdline = ' '.join(proc.info['cmdline'] or []).lower()
            if any(k in name or k in cmdline for k in ['log-processor', 'log_processor']):
                return psutil.Process(proc.info['pid'])
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            continue
    return None


def resource_sampler(stop: threading.Event, process, interval: float, samples: list):
    """采样服务端进程的 CPU 和内存占用"""
    try:
        process.cpu_percent(interval=None)  # 初始化基准
    except Exception:
        pass
    while not stop.is_set():
        time.sleep(interval)
        try:
            cpu = process.cpu_percent(interval=None)
            mem = process.memory_info().rss / (1024 * 1024)  # MB
            samples.append((time.time(), cpu, mem))
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            break
        except Exception:
            pass


def queue_depth_sampler(stop: threading.Event, interval: float, samples: list):
    """高频采样队列深度（发送阶段专用）"""
    while not stop.is_set():
        status = get_server_status()
        proc = status.get("processor", {})
        input_q = proc.get("input_queue_size", 0)
        output_q = proc.get("output_queue_size", 0)
        overflow = proc.get("overflow_queue_pending", 0)
        samples.append((time.time(), input_q, output_q, overflow))
        time.sleep(interval)


def wait_for_drain(timeout: int = 180) -> tuple[int, float, int]:
    """等待队列排空，返回 (最终总数, 排空耗时, 最大队列积压)

    关键改进：队列深度检测间隔从 0.5s 缩短到 50ms，避免错过 batch commit 间隙的峰值
    """
    drain_start = time.time()
    prev_total = get_server_total()
    stable_count = 0
    max_backlog = 0

    while time.time() - drain_start < timeout:
        status = get_server_status()
        proc = status.get("processor", {})
        input_q = proc.get("input_queue_size", 0)
        output_q = proc.get("output_queue_size", 0)
        overflow = proc.get("overflow_queue_pending", 0)
        backlog = input_q + output_q + overflow

        if backlog > max_backlog:
            max_backlog = backlog

        curr_total = get_server_total()

        if backlog > 0:
            time.sleep(0.05)  # 50ms 高频采样，捕捉 batch commit 间隙的峰值
            prev_total = curr_total
            stable_count = 0
            continue

        if curr_total == prev_total:
            stable_count += 1
            if stable_count >= 3:
                return curr_total, time.time() - drain_start, max_backlog
        else:
            stable_count = 0

        prev_total = curr_total
        time.sleep(1)

    return curr_total, time.time() - drain_start, max_backlog


def run_stage(args, concurrency: int, duration: int) -> dict:
    """执行单个并发级别的测试，返回该阶段的结果"""
    print(f"\n{'='*60}")
    print(f"阶段: 并发={concurrency}, 持续={duration}s")
    print(f"{'='*60}")

    # 清空日志
    clear_logs()
    time.sleep(1)
    before = get_server_total()
    print(f"[INFO] 已清空，初始数量: {before}")

    # 启动高频采样（入库数）
    sampler_stop = threading.Event()
    samples = []
    sampler_thread = threading.Thread(
        target=high_freq_sampler,
        args=(sampler_stop, args.interval, samples),
        daemon=True,
    )
    sampler_thread.start()

    # 启动队列深度高频采样（发送阶段专用，100ms 间隔）
    queue_stop = threading.Event()
    queue_samples = []
    queue_thread = threading.Thread(
        target=queue_depth_sampler,
        args=(queue_stop, 0.1, queue_samples),
        daemon=True,
    )
    queue_thread.start()

    # 启动资源采样（CPU / 内存）
    server_proc = find_server_process()
    resource_stop = threading.Event()
    resource_samples = []
    resource_thread = None
    if server_proc:
        resource_thread = threading.Thread(
            target=resource_sampler,
            args=(resource_stop, server_proc, 2.0, resource_samples),
            daemon=True,
        )
        resource_thread.start()
        print(f"[INFO] 资源监控已启动，目标进程 PID={server_proc.pid}")
    else:
        print("[WARN] 未找到服务端进程，跳过 CPU/内存监控")

    # 发送阶段
    send_start = time.time()
    sender_map = {"tcp": sender_tcp, "udp": sender_udp, "http": sender_http}
    sender = sender_map[args.protocol]

    with ThreadPoolExecutor(max_workers=concurrency) as pool:
        futures = [pool.submit(sender, args, i, duration) for i in range(concurrency)]
        for f in futures:
            f.result()

    send_elapsed = time.time() - send_start
    print(f"[INFO] 发送结束，耗时: {send_elapsed:.2f}s")

    # 发送结束后立即停止队列深度采样
    queue_stop.set()
    queue_thread.join(timeout=2)

    # 计算发送阶段队列深度峰值（从高频采样数据中获取）
    send_phase_max_input = 0
    send_phase_max_output = 0
    if queue_samples:
        send_phase_max_input = max(s[1] for s in queue_samples)
        send_phase_max_output = max(s[2] for s in queue_samples)

    # 发送结束后立即开始等待排空，入库采样线程在此期间继续运行
    print("[INFO] 等待后端排空...")
    final_total, drain_elapsed, drain_max_backlog = wait_for_drain()

    # 取发送阶段峰值和排空阶段峰值的较大者
    max_backlog = max(send_phase_max_input + send_phase_max_output, drain_max_backlog)

    # 排空完成后再停止其他采样线程
    sampler_stop.set()
    sampler_thread.join(timeout=5)

    if resource_thread:
        resource_stop.set()
        resource_thread.join(timeout=5)

    total_elapsed = time.time() - send_start
    stored = max(0, final_total - before)

    # 计算资源指标
    avg_cpu = 0.0
    peak_cpu = 0.0
    avg_mem = 0.0
    peak_mem = 0.0
    if resource_samples:
        cpus = [s[1] for s in resource_samples]
        mems = [s[2] for s in resource_samples]
        avg_cpu = sum(cpus) / len(cpus)
        peak_cpu = max(cpus)
        avg_mem = sum(mems) / len(mems)
        peak_mem = max(mems)

    # 计算指标
    client_qps = stored / send_elapsed if send_elapsed > 0 else 0
    sustained_qps = stored / total_elapsed if total_elapsed > 0 else 0

    # 计算峰值 QPS：使用滑动窗口平均（3 个采样点）避免 batch commit 跳变导致的异常峰值
    peak_qps = 0
    qps_list = []
    for i in range(1, len(samples)):
        dt = samples[i][0] - samples[i - 1][0]
        dq = samples[i][1] - samples[i - 1][1]
        if dt > 0.05:
            qps_list.append(dq / dt)

    if qps_list:
        # 过滤异常值：并发查询 SQLite WAL 快照不一致可能导致 total_count 跳变
        max_reasonable = max(client_qps * 3, 20000)
        filtered = [q for q in qps_list if 0 <= q <= max_reasonable]

        # 方法：取最高的 3 个连续采样点的平均速率，过滤单点跳变
        window_qps = []
        for i in range(len(filtered) - 2):
            window_avg = sum(filtered[i:i+3]) / 3
            window_qps.append(window_avg)
        if window_qps:
            peak_qps = max(window_qps)
        elif filtered:
            peak_qps = max(filtered)
        else:
            peak_qps = 0

    inject_qps = concurrency * args.rate if args.rate > 0 else round(client_qps, 0)
    result = {
        "concurrency": concurrency,
        "send_elapsed": round(send_elapsed, 2),
        "drain_elapsed": round(drain_elapsed, 2),
        "total_elapsed": round(total_elapsed, 2),
        "stored": stored,
        "inject_qps": round(inject_qps, 0),
        "client_qps": round(client_qps, 0),
        "sustained_qps": round(sustained_qps, 0),
        "peak_qps": round(peak_qps, 0),
        "max_backlog": max_backlog,
        "avg_cpu": round(avg_cpu, 1),
        "peak_cpu": round(peak_cpu, 1),
        "avg_mem": round(avg_mem, 1),
        "peak_mem": round(peak_mem, 1),
    }

    inject_label = f"inject={result['inject_qps']:.0f}/s" if args.rate > 0 else f"client={result['client_qps']:.0f}/s"
    cpu_label = f" CPU={avg_cpu:.1f}%" if resource_samples else ""
    mem_label = f" MEM={avg_mem:.1f}MB" if resource_samples else ""
    print(f"[INFO] 阶段完成: {inject_label}, sustained={result['sustained_qps']:.0f}/s, peak={result['peak_qps']:.0f}/s, backlog={max_backlog}{cpu_label}{mem_label}")
    return result


def generate_concurrency_levels(start: int, max_c: int, mode: str, step: int) -> list[int]:
    levels = []
    c = start
    while c <= max_c:
        levels.append(c)
        if mode == "exp":
            c *= 2
        else:
            c += step
    if levels[-1] != max_c and max_c not in levels:
        levels.append(max_c)
    return levels


def print_summary(results: list[dict]):
    has_resource = any(r.get("avg_cpu", 0) > 0 for r in results)
    print("\n" + "=" * 120)
    print("并发量递增性能测试汇总结果")
    print("=" * 120)
    print(f"{'并发':>5} | {'发送':>5} | {'排空':>5} | {'入库数':>8} | {'注入QPS':>8} | {'持续QPS':>8} | {'峰值QPS':>8} | {'队列积压':>8} | {'CPU':>6} | {'内存':>8}")
    print("-" * 110)
    for r in results:
        inject_val = r['inject_qps'] if r.get('inject_qps', 0) > 0 else r['client_qps']
        print(
            f"{r['concurrency']:>5} | {r['send_elapsed']:>4.1f}s | {r['drain_elapsed']:>4.1f}s | "
            f"{r['stored']:>8,} | {inject_val:>8,.0f} | {r['sustained_qps']:>8,.0f} | "
            f"{r['peak_qps']:>8,.0f} | {r['max_backlog']:>8,} | "
            f"{r['avg_cpu']:>5.1f}% | {r['avg_mem']:>7.1f}MB"
        )
    print("=" * 110)


def save_csv(results: list[dict], path: str):
    with open(path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=results[0].keys())
        writer.writeheader()
        writer.writerows(results)
    print(f"[INFO] 结果已保存到 {path}")


def plot_results(results: list[dict]):
    try:
        import matplotlib.pyplot as plt
    except ImportError:
        print("[WARN] matplotlib 未安装，跳过绘图。运行: pip install matplotlib")
        return

    concurrencies = [r["concurrency"] for r in results]
    has_resource = any(r.get("avg_cpu", 0) > 0 for r in results)

    fig, axes = plt.subplots(2, 2, figsize=(14, 10))
    fig.suptitle("并发量递增性能测试分析", fontsize=16, fontweight="bold")

    # 图1: QPS 曲线
    ax1 = axes[0, 0]
    client_qps = [r["client_qps"] for r in results]
    sustained_qps = [r["sustained_qps"] for r in results]
    peak_qps = [r["peak_qps"] for r in results]
    ax1.plot(concurrencies, client_qps, "o-", label="客户端发送速率" if args.rate == 0 else "理论注入速率", color="#4472C4")
    ax1.plot(concurrencies, sustained_qps, "s-", label="服务端持续QPS", color="#ED7D31")
    ax1.plot(concurrencies, peak_qps, "^-", label="峰值QPS", color="#70AD47")
    ax1.set_xlabel("并发数")
    ax1.set_ylabel("QPS (条/秒)")
    ax1.set_title("并发量 vs 吞吐量")
    ax1.legend()
    ax1.grid(True, alpha=0.3)
    ax1.set_xticks(concurrencies)

    # 图2: 排空等待时间
    ax2 = axes[0, 1]
    drain_times = [r["drain_elapsed"] for r in results]
    colors = ["#70AD47" if d < 10 else "#ED7D31" if d < 60 else "#C5504B" for d in drain_times]
    ax2.bar([str(c) for c in concurrencies], drain_times, color=colors)
    ax2.set_xlabel("并发数")
    ax2.set_ylabel("后端排空等待时间 (秒)")
    ax2.set_title("系统开销：队列积压消化时间")
    ax2.grid(True, alpha=0.3, axis="y")

    # 图3: 队列积压
    ax3 = axes[1, 0]
    backlogs = [r["max_backlog"] for r in results]
    colors3 = ["#70AD47" if b == 0 else "#ED7D31" if b < 100000 else "#C5504B" for b in backlogs]
    ax3.bar([str(c) for c in concurrencies], backlogs, color=colors3)
    ax3.set_xlabel("并发数")
    ax3.set_ylabel("最大队列积压 (条)")
    ax3.set_title("系统开销：内存队列最大积压")
    ax3.grid(True, alpha=0.3, axis="y")

    if has_resource:
        # 图4: CPU + 内存双轴
        ax4 = axes[1, 1]
        avg_cpus = [r["avg_cpu"] for r in results]
        avg_mems = [r["avg_mem"] for r in results]
        ax4_twin = ax4.twinx()
        ax4.plot(concurrencies, avg_cpus, "o-", label="平均CPU", color="#C5504B")
        ax4_twin.plot(concurrencies, avg_mems, "s--", label="平均内存", color="#4472C4")
        ax4.set_xlabel("并发数")
        ax4.set_ylabel("CPU (%)", color="#C5504B")
        ax4_twin.set_ylabel("内存 (MB)", color="#4472C4")
        ax4.set_title("系统开销：CPU / 内存")
        ax4.tick_params(axis='y', labelcolor="#C5504B")
        ax4_twin.tick_params(axis='y', labelcolor="#4472C4")
        ax4.grid(True, alpha=0.3)
        ax4.set_xticks(concurrencies)
        # 合并图例
        lines1, labels1 = ax4.get_legend_handles_labels()
        lines2, labels2 = ax4_twin.get_legend_handles_labels()
        ax4.legend(lines1 + lines2, labels1 + labels2, loc="upper left", fontsize=9)
    else:
        # 图4: 并发效率
        ax4 = axes[1, 1]
        efficiency = [r["sustained_qps"] / r["concurrency"] for r in results]
        ax4.plot(concurrencies, efficiency, "D-", color="#7030A0")
        ax4.set_xlabel("并发数")
        ax4.set_ylabel("每并发持续 QPS (条/秒/并发)")
        ax4.set_title("并发效率衰减")
        ax4.grid(True, alpha=0.3)
        ax4.set_xticks(concurrencies)

    plt.tight_layout(rect=[0, 0, 1, 0.96])
    plt.savefig("concurrency_scaling_result.png", dpi=300)
    print("[INFO] 图表已保存到 concurrency_scaling_result.png")


def main() -> int:
    parser = argparse.ArgumentParser(description="并发量递增性能测试")
    parser.add_argument("-protocol", choices=["tcp", "udp", "http"], default="tcp")
    parser.add_argument("-addr", default="localhost:9000")
    parser.add_argument("-start", type=int, default=1, help="起始并发数")
    parser.add_argument("-max", type=int, default=64, help="最大并发数")
    parser.add_argument("-mode", choices=["linear", "exp"], default="exp", help="递增模式: linear=线性, exp=指数")
    parser.add_argument("-step", type=int, default=5, help="线性模式下的递增步长")
    parser.add_argument("-d", type=int, default=20, help="每个并发级别的持续发送秒数")
    parser.add_argument("-interval", type=float, default=0.5, help="采样间隔（秒）")
    parser.add_argument("-rate", type=int, default=0, help="每个 worker 的限速（条/秒），0=不限速")
    parser.add_argument("-output", type=str, default="", help="保存 CSV 结果文件路径")
    parser.add_argument("-plot", action="store_true", help="生成性能曲线图")
    args = parser.parse_args()

    levels = generate_concurrency_levels(args.start, args.max, args.mode, args.step)
    print("=" * 60)
    print("并发量递增性能测试")
    print("=" * 60)
    print(f"协议: {args.protocol}, 地址: {args.addr}")
    print(f"递增模式: {args.mode}, 并发级别: {levels}")
    print(f"每阶段持续: {args.d}s, 采样间隔: {args.interval}s")
    if args.rate > 0:
        print(f"每个 worker 限速: {args.rate} 条/秒")
    else:
        print("限速: 无限制（每个 worker 尽可能快发送）")
    print(f"预计总耗时: ~{len(levels) * (args.d + 30)}s（含排空等待）")

    results = []
    for concurrency in levels:
        result = run_stage(args, concurrency, args.d)
        results.append(result)

    # 输出汇总
    print_summary(results)

    # 保存 CSV
    if args.output:
        save_csv(results, args.output)

    # 绘制图表
    if args.plot:
        plot_results(results)

    # 性能拐点提示
    print("\n分析提示:")
    max_sustained = max(results, key=lambda x: x["sustained_qps"])
    print(f"  - 最高持续吞吐量出现在并发={max_sustained['concurrency']}，QPS={max_sustained['sustained_qps']:.0f}")

    # 找拐点：持续QPS开始明显下降的点
    for i in range(1, len(results)):
        if results[i]["sustained_qps"] < results[i - 1]["sustained_qps"] * 0.85:
            print(f"  - 性能拐点约出现在并发={results[i]['concurrency']}（持续QPS开始下降）")
            break
    else:
        print(f"  - 未观察到明显性能拐点，系统在测试范围内持续扩展")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
