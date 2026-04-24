#!/usr/bin/env python3
"""
生成适合论文答辩PPT展示的高质量并发性能趋势图。

输出（白色背景、300 DPI、独立PNG）：
  1. ppt_throughput.png   — 吞吐量趋势（核心图）
  2. ppt_drain_time.png   — 后端排空时间
  3. ppt_resources.png    — CPU + 内存资源消耗
  4. ppt_backlog.png      — 队列最大积压
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

# ============================================================
# 全局样式设置（PPT友好：白底、大字体、清晰线条）
# ============================================================
plt.rcParams.update({
    'font.size': 14,
    'axes.labelsize': 15,
    'axes.titlesize': 17,
    'xtick.labelsize': 13,
    'ytick.labelsize': 13,
    'legend.fontsize': 13,
    'figure.facecolor': 'white',
    'axes.facecolor': 'white',
    'axes.edgecolor': '#333333',
    'axes.grid': True,
    'grid.alpha': 0.25,
    'grid.color': '#999999',
    'grid.linestyle': '--',
    'grid.linewidth': 0.8,
    'lines.linewidth': 2.5,
    'lines.markersize': 9,
    'figure.dpi': 300,
})

# 中文字体
plt.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'Arial Unicode MS']
plt.rcParams['axes.unicode_minus'] = False

# 统一配色方案（学术/答辩风格，色盲友好）
C_INJECT = "#4472C4"      # 理论注入 — 蓝
C_SUSTAINED = "#ED7D31"   # 实际吞吐 — 橙
C_PEAK = "#70AD47"        # 峰值 — 绿
C_CPU = "#C5504B"         # CPU — 红
C_MEM = "#5B9BD5"         # 内存 — 浅蓝
C_DRAIN = "#7030A0"       # 排空 — 紫
C_BACKLOG = "#ED7D31"     # 积压 — 橙
C_IDEAL = "#A5A5A5"       # 理想线 — 灰

CSV_PATH = Path(__file__).parent / "scaling.csv"
if not CSV_PATH.exists():
    CSV_PATH = Path("scaling.csv")

OUTDIR = Path(__file__).parent


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


def save(fig, name):
    path = OUTDIR / name
    fig.savefig(path, dpi=300, bbox_inches='tight', facecolor='white', edgecolor='none')
    print(f"[OK] 已保存: {path}")
    plt.close(fig)


# ============================================================
# 图1：吞吐量趋势（答辩核心图）
# ============================================================
def plot_throughput(data):
    x = np.array([r["concurrency"] for r in data])
    inject = np.array([r["inject_qps"] for r in data])
    sustained = np.array([r["sustained_qps"] for r in data])
    peak = np.array([r["peak_qps"] for r in data])

    fig, ax = plt.subplots(figsize=(10, 6))

    # 理想线性参考线（基于1并发的sustained）
    ideal = sustained[0] * x
    ax.plot(x, ideal, "--", linewidth=1.8, color=C_IDEAL, label="理想线性增长", zorder=1)

    ax.plot(x, inject, "o-", linewidth=2.5, markersize=9, color=C_INJECT,
            label="理论注入速率", markerfacecolor='white', markeredgewidth=2, zorder=3)
    ax.plot(x, sustained, "s-", linewidth=2.5, markersize=9, color=C_SUSTAINED,
            label="服务端持续吞吐量", markerfacecolor='white', markeredgewidth=2, zorder=4)
    ax.plot(x, peak, "^-", linewidth=2, markersize=9, color=C_PEAK,
            label="瞬时峰值吞吐量", markerfacecolor='white', markeredgewidth=2, zorder=3)

    # 数据标注（只标sustained）——错开布局+白色衬底避免被线条遮挡
    anno_offsets = {
        1: (0, 14),     # 167 放上方
        2: (0, -22),    # 333 放下方
        4: (0, -22),    # 667 放正下方（避开健康区标注框）
        8: (22, 0),     # 1200 放右方（避开健康区标注框左侧）
        16: (-18, 12),  # 1000 放左上方
        32: (18, 10),   # 837 放右上方
    }
    for xi, yi in zip(x, sustained):
        dx, dy = anno_offsets.get(int(xi), (0, 14))
        ax.annotate(f"{yi:.0f}", (xi, yi), textcoords="offset points",
                    xytext=(dx, dy), ha="center", fontsize=12, fontweight='bold',
                    color=C_SUSTAINED, zorder=5,
                    bbox=dict(boxstyle='round,pad=0.15', facecolor='white',
                              edgecolor='none', alpha=0.85))

    # 区域划分：健康区 / 过载区
    ax.axvspan(0.5, 8.5, alpha=0.06, color=C_PEAK, zorder=0)
    ax.axvspan(8.5, 36, alpha=0.06, color=C_CPU, zorder=0)

    # 区域文字标注——错开布局避免重叠
    ax.text(4, max(inject) * 0.18, "健康区：吞吐线性增长", ha="center",
            fontsize=11, color=C_PEAK, fontweight='bold',
            bbox=dict(boxstyle='round,pad=0.25', facecolor='white', edgecolor=C_PEAK, alpha=0.85))
    ax.text(28, max(inject) * 0.85, "过载区：吞吐下降\n积压激增、效率衰减", ha="center",
            fontsize=11, color=C_CPU, fontweight='bold',
            bbox=dict(boxstyle='round,pad=0.25', facecolor='white', edgecolor=C_CPU, alpha=0.85))

    ax.set_xlabel("并发客户端数", fontsize=15)
    ax.set_ylabel("吞吐量 (条/秒)", fontsize=15)
    ax.set_title("并发递增测试：系统吞吐量变化趋势", fontsize=18, fontweight='bold', pad=15)
    ax.legend(loc="upper left", framealpha=0.95, edgecolor='#cccccc')
    ax.set_xticks(x)
    ax.set_xlim(0, max(x) * 1.15)
    ax.set_ylim(0, max(inject) * 1.15)

    plt.tight_layout()
    save(fig, "ppt_throughput.png")


# ============================================================
# 图2：后端排空时间（指数增长，展示系统恶化）
# ============================================================
def plot_drain_time(data):
    x = np.array([r["concurrency"] for r in data])
    drain = np.array([r["drain_elapsed"] for r in data])

    fig, ax = plt.subplots(figsize=(10, 6))

    colors = [C_PEAK if d < 10 else "#ED7D31" if d < 60 else C_CPU for d in drain]
    bars = ax.bar(x.astype(str), drain, color=colors, edgecolor="white", width=0.55, zorder=3)

    for bar, val in zip(bars, drain):
        ax.annotate(f"{val:.1f}s",
                    xy=(bar.get_x() + bar.get_width() / 2, bar.get_height()),
                    xytext=(0, 5), textcoords="offset points",
                    ha="center", fontsize=12, fontweight='bold', zorder=4)

    # 阈值线
    ax.axhline(y=10, color=C_PEAK, linestyle='--', linewidth=1.5, alpha=0.7, zorder=2)
    ax.axhline(y=60, color=C_CPU, linestyle='--', linewidth=1.5, alpha=0.7, zorder=2)
    ax.text(len(x) - 0.3, 11, "健康阈值 10s", fontsize=11, color=C_PEAK, fontweight='bold')
    ax.text(len(x) - 0.3, 63, "过载阈值 60s", fontsize=11, color=C_CPU, fontweight='bold')

    ax.set_xlabel("并发客户端数", fontsize=15)
    ax.set_ylabel("后端排空等待时间 (秒)", fontsize=15)
    ax.set_title("系统恶化指标：队列排空时间随并发激增", fontsize=18, fontweight='bold', pad=15)

    from matplotlib.patches import Patch
    legend_elements = [
        Patch(facecolor=C_PEAK, edgecolor='white', label="健康 (<10s)"),
        Patch(facecolor="#ED7D31", edgecolor='white', label="过载 (10~60s)"),
        Patch(facecolor=C_CPU, edgecolor='white', label="严重过载 (>60s)"),
    ]
    ax.legend(handles=legend_elements, loc="upper left", framealpha=0.95, edgecolor='#cccccc')

    plt.tight_layout()
    save(fig, "ppt_drain_time.png")


# ============================================================
# 图3：CPU + 内存资源消耗（双Y轴）
# ============================================================
def plot_resources(data):
    x = np.array([r["concurrency"] for r in data])
    avg_cpu = np.array([r.get("avg_cpu", 0) for r in data])
    peak_cpu = np.array([r.get("peak_cpu", 0) for r in data])
    avg_mem = np.array([r.get("avg_mem", 0) for r in data])
    peak_mem = np.array([r.get("peak_mem", 0) for r in data])

    fig, ax1 = plt.subplots(figsize=(10, 6))

    # CPU 左轴
    ax1.plot(x, avg_cpu, "o-", linewidth=2.5, markersize=9, color=C_CPU,
             label="平均 CPU", markerfacecolor='white', markeredgewidth=2)
    ax1.plot(x, peak_cpu, "s--", linewidth=2, markersize=8, color=C_CPU,
             label="峰值 CPU", markerfacecolor='white', markeredgewidth=2, alpha=0.7)
    ax1.set_xlabel("并发客户端数", fontsize=15)
    ax1.set_ylabel("CPU 占用率 (%)", fontsize=15, color=C_CPU)
    ax1.tick_params(axis='y', labelcolor=C_CPU)
    ax1.set_ylim(0, max(peak_cpu) * 1.3 if max(peak_cpu) > 0 else 100)

    # 标注CPU数据点——上下交错+白色衬底避免被内存线遮挡
    cpu_offsets = {
        1: (0, -18),    # 5.2% 放下方
        2: (14, 6),     # 8.5% 放右上方
        4: (0, -18),    # 15.1% 放下方
        8: (-16, 0),    # 27.1% 放左方（避开右轴内存点）
        16: (0, -18),   # 22.5% 放下方
        32: (0, 12),    # 36.5% 放上方
    }
    for xi, yi in zip(x, avg_cpu):
        if yi > 0:
            dx, dy = cpu_offsets.get(int(xi), (0, 12))
            ax1.annotate(f"{yi:.1f}%", (xi, yi), textcoords="offset points",
                         xytext=(dx, dy), ha="center", fontsize=11, color=C_CPU, fontweight='bold',
                         bbox=dict(boxstyle='round,pad=0.15', facecolor='white',
                                   edgecolor='none', alpha=0.85))

    # 内存 右轴
    ax2 = ax1.twinx()
    ax2.plot(x, avg_mem, "D-", linewidth=2.5, markersize=9, color=C_MEM,
             label="平均内存", markerfacecolor='white', markeredgewidth=2)
    ax2.plot(x, peak_mem, "d--", linewidth=2, markersize=8, color=C_MEM,
             label="峰值内存", markerfacecolor='white', markeredgewidth=2, alpha=0.7)
    ax2.set_ylabel("内存占用 (MB)", fontsize=15, color=C_MEM)
    ax2.tick_params(axis='y', labelcolor=C_MEM)
    ax2.set_ylim(0, max(peak_mem) * 1.3 if max(peak_mem) > 0 else 500)

    # 合并图例
    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax1.legend(lines1 + lines2, labels1 + labels2, loc="upper left",
               framealpha=0.95, edgecolor='#cccccc', ncol=1, fontsize=11,
               bbox_to_anchor=(0.02, 0.98))

    ax1.set_title("资源消耗：CPU 与内存随并发变化", fontsize=18, fontweight='bold', pad=15)
    ax1.set_xticks(x)

    plt.tight_layout()
    save(fig, "ppt_resources.png")


# ============================================================
# 图4：队列最大积压
# ============================================================
def plot_backlog(data):
    x = np.array([r["concurrency"] for r in data])
    backlog = np.array([r["max_backlog"] for r in data])

    fig, ax = plt.subplots(figsize=(10, 6))

    colors = [C_PEAK if b == 0 else "#ED7D31" if b < 100000 else C_CPU for b in backlog]
    bars = ax.bar(x.astype(str), backlog, color=colors, edgecolor="white", width=0.55, zorder=3)

    for bar, val in zip(bars, backlog):
        label = f"{val:,.0f}" if val > 0 else "0 (无积压)"
        ax.annotate(label,
                    xy=(bar.get_x() + bar.get_width() / 2, bar.get_height()),
                    xytext=(0, 5), textcoords="offset points",
                    ha="center", fontsize=12, fontweight='bold', zorder=4)

    ax.set_xlabel("并发客户端数", fontsize=15)
    ax.set_ylabel("最大队列积压 (条)", fontsize=15)
    ax.set_title("内存队列积压：过载后积压数量级跃升", fontsize=18, fontweight='bold', pad=15)

    from matplotlib.patches import Patch
    legend_elements = [
        Patch(facecolor=C_PEAK, edgecolor='white', label="无积压"),
        Patch(facecolor="#ED7D31", edgecolor='white', label="轻度积压 (<10万)"),
        Patch(facecolor=C_CPU, edgecolor='white', label="严重积压 (≥10万)"),
    ]
    ax.legend(handles=legend_elements, loc="upper left", framealpha=0.95, edgecolor='#cccccc')

    plt.tight_layout()
    save(fig, "ppt_backlog.png")


# ============================================================
# 图5：并发效率衰减（单图）
# ============================================================
def plot_efficiency(data):
    x = np.array([r["concurrency"] for r in data])
    sustained = np.array([r["sustained_qps"] for r in data])
    efficiency = sustained / x

    fig, ax = plt.subplots(figsize=(10, 6))

    ax.plot(x, efficiency, "H-", linewidth=2.5, markersize=10, color=C_DRAIN,
            markerfacecolor='white', markeredgewidth=2, zorder=3)

    for xi, yi in zip(x, efficiency):
        offset = 14 if xi not in (1, 2) else 22
        ax.annotate(f"{yi:.0f}", (xi, yi), textcoords="offset points",
                    xytext=(0, offset), ha="center", fontsize=12, fontweight='bold',
                    color=C_DRAIN, zorder=4)

    ax.set_xlabel("并发客户端数", fontsize=15)
    ax.set_ylabel("每并发持续 QPS (条/秒/并发)", fontsize=15)
    ax.set_title("并发效率衰减：单位并发处理能力随规模递减", fontsize=18, fontweight='bold', pad=15)
    ax.set_xticks(x)

    plt.tight_layout()
    save(fig, "ppt_efficiency.png")


def main():
    data = load_data()
    if not data:
        print("[ERROR] 未读取到数据")
        return 1

    print("=" * 60)
    print("正在生成答辩PPT用图表...")
    print("=" * 60)

    plot_throughput(data)
    plot_drain_time(data)
    plot_resources(data)
    plot_backlog(data)
    plot_efficiency(data)

    print("=" * 60)
    print("全部完成！输出目录:", OUTDIR.absolute())
    print("=" * 60)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
