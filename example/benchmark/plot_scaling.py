#!/usr/bin/env python3
"""
根据 scaling.csv 生成并发递增性能测试的可视化图表。

生成图表：
1. 吞吐量对比折线图（注入 QPS vs 持续 QPS vs 峰值 QPS）
2. 系统开销增长图（后端排空等待时间）
3. 队列积压增长图（最大积压）
4. CPU 占用率折线图（平均 / 峰值）
5. 内存占用折线图（平均 / 峰值）
6. 并发效率衰减图（每并发实际获得的 QPS）
"""

import csv
from pathlib import Path

try:
    import matplotlib.pyplot as plt
    import numpy as np
except ImportError:
    print("[ERROR] 需要安装 matplotlib 和 numpy")
    print("  pip install matplotlib numpy")
    raise SystemExit(1)

# 设置中文字体
plt.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'Arial Unicode MS']
plt.rcParams['axes.unicode_minus'] = False

CSV_PATH = Path(__file__).parent / "scaling.csv"
if not CSV_PATH.exists():
    CSV_PATH = Path("scaling.csv")


def load_data():
    rows = []
    with open(CSV_PATH, "r", encoding="utf-8", newline="") as f:
        reader = csv.DictReader(f)
        for r in reader:
            row = {}
            for k, v in r.items():
                if v == "":
                    row[k] = 0
                elif '.' in v:
                    row[k] = float(v)
                else:
                    row[k] = int(v)
            rows.append(row)
    return rows


def plot_throughput(data, ax):
    x = [r["concurrency"] for r in data]
    inject = [r["inject_qps"] for r in data]
    sustained = [r["sustained_qps"] for r in data]
    peak = [r["peak_qps"] for r in data]

    ax.plot(x, inject, "o-", linewidth=2, markersize=8, label="理论注入速率", color="#4472C4")
    ax.plot(x, sustained, "s-", linewidth=2, markersize=8, label="服务端持续QPS", color="#ED7D31")
    ax.plot(x, peak, "^-", linewidth=2, markersize=8, label="峰值QPS", color="#70AD47")

    for xi, yi in zip(x, sustained):
        ax.annotate(f"{yi:.0f}", (xi, yi), textcoords="offset points", xytext=(0, 10), ha="center", fontsize=9)

    ax.set_xlabel("并发数", fontsize=12)
    ax.set_ylabel("吞吐量 (条/秒)", fontsize=12)
    ax.set_title("并发量 vs 吞吐量", fontsize=14, fontweight="bold")
    ax.legend(fontsize=10, loc="upper left")
    ax.grid(True, alpha=0.3)
    ax.set_xticks(x)

    ax.axvspan(8, 16, alpha=0.1, color="red")
    ax.text(12, max(inject) * 0.5, "性能拐点区\n(系统开始过载)", ha="center", fontsize=10, color="red")


def plot_overhead(data, ax):
    x = [r["concurrency"] for r in data]
    drain = [r["drain_elapsed"] for r in data]

    colors = ["#70AD47" if d < 10 else "#ED7D31" if d < 60 else "#C5504B" for d in drain]
    bars = ax.bar([str(v) for v in x], drain, color=colors, edgecolor="white", width=0.6)

    for bar, val in zip(bars, drain):
        height = bar.get_height()
        ax.annotate(f"{val:.1f}s",
                    xy=(bar.get_x() + bar.get_width() / 2, height),
                    xytext=(0, 3),
                    textcoords="offset points",
                    ha="center", fontsize=9)

    ax.set_xlabel("并发数", fontsize=12)
    ax.set_ylabel("后端排空等待时间 (秒)", fontsize=12)
    ax.set_title("系统开销：队列积压消化时间", fontsize=14, fontweight="bold")
    ax.grid(True, alpha=0.3, axis="y")

    from matplotlib.patches import Patch
    legend_elements = [
        Patch(facecolor="#70AD47", label="健康 (<10s)"),
        Patch(facecolor="#ED7D31", label="过载 (10~60s)"),
        Patch(facecolor="#C5504B", label="严重过载 (>60s)"),
    ]
    ax.legend(handles=legend_elements, fontsize=9, loc="upper left")


def plot_backlog(data, ax):
    x = [r["concurrency"] for r in data]
    backlog = [r["max_backlog"] for r in data]

    colors = ["#70AD47" if b == 0 else "#ED7D31" if b < 100000 else "#C5504B" for b in backlog]
    bars = ax.bar([str(v) for v in x], backlog, color=colors, edgecolor="white", width=0.6)

    for bar, val in zip(bars, backlog):
        height = bar.get_height()
        label = f"{val:,.0f}" if val > 0 else "0"
        ax.annotate(label,
                    xy=(bar.get_x() + bar.get_width() / 2, height),
                    xytext=(0, 3),
                    textcoords="offset points",
                    ha="center", fontsize=9)

    ax.set_xlabel("并发数", fontsize=12)
    ax.set_ylabel("最大队列积压 (条)", fontsize=12)
    ax.set_title("系统开销：内存队列最大积压", fontsize=14, fontweight="bold")
    ax.grid(True, alpha=0.3, axis="y")


def plot_cpu(data, ax):
    x = [r["concurrency"] for r in data]
    avg_cpu = [r.get("avg_cpu", 0) for r in data]
    peak_cpu = [r.get("peak_cpu", 0) for r in data]

    ax.plot(x, avg_cpu, "o-", linewidth=2, markersize=8, label="平均CPU", color="#4472C4")
    ax.plot(x, peak_cpu, "s--", linewidth=2, markersize=8, label="峰值CPU", color="#C5504B")

    for xi, yi in zip(x, avg_cpu):
        if yi > 0:
            ax.annotate(f"{yi:.1f}%", (xi, yi), textcoords="offset points", xytext=(0, 10), ha="center", fontsize=9)

    ax.set_xlabel("并发数", fontsize=12)
    ax.set_ylabel("CPU 占用率 (%)", fontsize=12)
    ax.set_title("系统开销：CPU 占用率", fontsize=14, fontweight="bold")
    ax.legend(fontsize=10)
    ax.grid(True, alpha=0.3)
    ax.set_xticks(x)


def plot_memory(data, ax):
    x = [r["concurrency"] for r in data]
    avg_mem = [r.get("avg_mem", 0) for r in data]
    peak_mem = [r.get("peak_mem", 0) for r in data]

    ax.plot(x, avg_mem, "o-", linewidth=2, markersize=8, label="平均内存", color="#4472C4")
    ax.plot(x, peak_mem, "s--", linewidth=2, markersize=8, label="峰值内存", color="#C5504B")

    for xi, yi in zip(x, avg_mem):
        if yi > 0:
            ax.annotate(f"{yi:.0f}MB", (xi, yi), textcoords="offset points", xytext=(0, 10), ha="center", fontsize=9)

    ax.set_xlabel("并发数", fontsize=12)
    ax.set_ylabel("内存占用 (MB)", fontsize=12)
    ax.set_title("系统开销：内存占用", fontsize=14, fontweight="bold")
    ax.legend(fontsize=10)
    ax.grid(True, alpha=0.3)
    ax.set_xticks(x)


def plot_efficiency(data, ax):
    x = [r["concurrency"] for r in data]
    sustained = [r["sustained_qps"] for r in data]
    efficiency = [s / c for s, c in zip(sustained, x)]

    ax.plot(x, efficiency, "D-", linewidth=2, markersize=8, color="#7030A0")

    for xi, yi in zip(x, efficiency):
        ax.annotate(f"{yi:.0f}", (xi, yi), textcoords="offset points", xytext=(0, 10), ha="center", fontsize=9)

    ax.set_xlabel("并发数", fontsize=12)
    ax.set_ylabel("每并发持续 QPS (条/秒/并发)", fontsize=12)
    ax.set_title("并发效率衰减：单位并发的处理能力", fontsize=14, fontweight="bold")
    ax.grid(True, alpha=0.3)
    ax.set_xticks(x)


def plot_overhead_ratio(data, ax):
    x = [r["concurrency"] for r in data]
    drain = [r["drain_elapsed"] for r in data]
    base = drain[0] if drain else 1
    ratios = [d / base if base > 0 else 0 for d in drain]

    ax.plot(x, ratios, "H-", linewidth=2, markersize=10, color="#C5504B")

    for xi, yi in zip(x, ratios):
        ax.annotate(f"{yi:.1f}x", (xi, yi), textcoords="offset points", xytext=(0, 10), ha="center", fontsize=9)

    ax.set_xlabel("并发数", fontsize=12)
    ax.set_ylabel("开销倍数 (相对于 1 并发)", fontsize=12)
    ax.set_title("系统开销增长倍数", fontsize=14, fontweight="bold")
    ax.grid(True, alpha=0.3)
    ax.set_xticks(x)


def print_analysis(data):
    print("\n" + "=" * 80)
    print("并发递增性能测试分析")
    print("=" * 80)

    has_resource = any(r.get("avg_cpu", 0) > 0 for r in data)
    base_drain = data[0]["drain_elapsed"]
    for r in data:
        c = r["concurrency"]
        drain = r["drain_elapsed"]
        ratio = drain / base_drain if base_drain > 0 else 0
        backlog = r["max_backlog"]
        sustained = r["sustained_qps"]
        inject = r["inject_qps"]
        efficiency = sustained / c
        status = "健康" if backlog == 0 and drain < 10 else "轻微过载" if drain < 60 else "严重过载"

        print(f"\n并发={c:>2}:")
        print(f"  注入速率: {inject:>6,.0f} QPS  |  持续吞吐: {sustained:>6,.0f} QPS")
        print(f"  排空等待: {drain:>6.1f}s    |  相对基准: {ratio:>5.1f}x")
        print(f"  最大积压: {backlog:>7,.0f}   |  每并发效率: {efficiency:>6.0f} QPS/并发")
        if has_resource:
            print(f"  平均CPU:  {r['avg_cpu']:>6.1f}%     |  平均内存: {r['avg_mem']:>6.1f} MB")
        print(f"  状态判定: {status}")

    print("\n" + "-" * 80)
    print("性能拐点分析:")
    for i in range(1, len(data)):
        prev = data[i - 1]
        curr = data[i]
        if curr["drain_elapsed"] > prev["drain_elapsed"] * 5 and curr["drain_elapsed"] > 10:
            print(f"  并发从 {prev['concurrency']} 增加到 {curr['concurrency']} 时:")
            print(f"    - 后端排空等待从 {prev['drain_elapsed']:.1f}s 暴增至 {curr['drain_elapsed']:.1f}s")
            print(f"    - 系统从 '{('健康' if prev['max_backlog']==0 else '过载')}' 进入 '严重过载' 状态")
            print(f"    - 建议系统最佳工作并发 ≤ {prev['concurrency']}")
            break
    else:
        print("  未观察到明显性能拐点，系统在测试范围内持续扩展")

    print("=" * 80)


def main():
    data = load_data()
    if not data:
        print("[ERROR] 未读取到数据")
        return 1

    print_analysis(data)

    has_resource = any(r.get("avg_cpu", 0) > 0 for r in data)

    if has_resource:
        # 有资源数据：2x3 布局
        fig, axes = plt.subplots(2, 3, figsize=(16, 10))
        fig.suptitle("并发量递增性能测试分析", fontsize=16, fontweight="bold", y=0.98)

        plot_throughput(data, axes[0, 0])
        plot_overhead(data, axes[0, 1])
        plot_backlog(data, axes[0, 2])
        plot_cpu(data, axes[1, 0])
        plot_memory(data, axes[1, 1])
        plot_efficiency(data, axes[1, 2])
    else:
        # 无资源数据：2x2 布局
        fig, axes = plt.subplots(2, 2, figsize=(14, 10))
        fig.suptitle("并发量递增性能测试分析", fontsize=16, fontweight="bold", y=0.98)

        plot_throughput(data, axes[0, 0])
        plot_overhead(data, axes[0, 1])
        plot_backlog(data, axes[1, 0])
        plot_efficiency(data, axes[1, 1])

    plt.tight_layout(rect=[0, 0, 1, 0.96])
    output_path = Path(__file__).parent / "scaling_analysis.png"
    plt.savefig(output_path, dpi=300)
    print(f"\n[INFO] 图表已保存: {output_path}")

    # 额外生成一张开销增长倍数图
    fig2, ax2 = plt.subplots(figsize=(8, 5))
    plot_overhead_ratio(data, ax2)
    plt.tight_layout()
    output_path2 = Path(__file__).parent / "scaling_overhead_ratio.png"
    plt.savefig(output_path2, dpi=300)
    print(f"[INFO] 开销倍数图已保存: {output_path2}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
