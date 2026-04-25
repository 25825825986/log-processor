#!/usr/bin/env python3
"""
Go vs Java vs Python 资源占用对比图（论文答辩 PPT 用）。

数据来源说明：
  - Go: 本系统实测数据（8并发 1200 QPS 场景）
  - Java: 基于 Spring Boot + Logback 同类系统的典型公开数据
  - Python: 基于 Python asyncio / multiprocessing 同类系统的典型公开数据
"""

import matplotlib.pyplot as plt
import numpy as np

# ============================================================
# 全局样式（PPT 友好）
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
    'figure.dpi': 300,
})

plt.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'Arial Unicode MS']
plt.rcParams['axes.unicode_minus'] = False

# 统一配色
C_GO = "#00ADD8"      # Go 蓝
C_JAVA = "#E76F00"    # Java 橙
C_PYTHON = "#3776AB"  # Python 蓝

OUTDIR = "."


def save(fig, name):
    path = f"{OUTDIR}/{name}"
    fig.savefig(path, dpi=300, bbox_inches='tight', facecolor='white', edgecolor='none')
    print(f"[OK] 已保存: {path}")
    plt.close(fig)


# ============================================================
# 图1：内存占用对比（分组柱状图）
# ============================================================
def plot_memory():
    categories = ["空载内存", "1000 QPS\n满载内存"]
    go_vals = [80, 129]
    java_vals = [350, 900]
    py_vals = [50, 550]

    x = np.arange(len(categories))
    width = 0.22

    fig, ax = plt.subplots(figsize=(10, 6))

    bars1 = ax.bar(x - width, go_vals, width, label="Go（本系统）", color=C_GO, edgecolor='white', zorder=3)
    bars2 = ax.bar(x, java_vals, width, label="Java（典型值）", color=C_JAVA, edgecolor='white', zorder=3)
    bars3 = ax.bar(x + width, py_vals, width, label="Python（典型值）", color=C_PYTHON, edgecolor='white', zorder=3)

    # 数值标注
    for bars in [bars1, bars2, bars3]:
        for bar in bars:
            height = bar.get_height()
            ax.annotate(f"{height:.0f}MB",
                        xy=(bar.get_x() + bar.get_width() / 2, height),
                        xytext=(0, 4), textcoords="offset points",
                        ha="center", fontsize=11, fontweight='bold', zorder=4)

    ax.set_ylabel("内存占用 (MB)", fontsize=15)
    ax.set_title("同性能下内存占用对比（1000 QPS  sustained）", fontsize=18, fontweight='bold', pad=15)
    ax.set_xticks(x)
    ax.set_xticklabels(categories, fontsize=14)
    ax.legend(framealpha=0.95, edgecolor='#cccccc', loc='upper left')
    ax.set_ylim(0, 1100)

    # 添加优势倍数标注
    ax.annotate("Go 内存仅为 Java 的 14%", xy=(1, 129), xytext=(0.65, 350),
                fontsize=12, color=C_GO, fontweight='bold',
                arrowprops=dict(arrowstyle='->', color=C_GO, lw=1.5),
                bbox=dict(boxstyle='round,pad=0.3', facecolor='#F0F8FB', edgecolor=C_GO, alpha=0.9))

    plt.tight_layout()
    save(fig, "ppt_lang_compare_memory.png")


# ============================================================
# 图2：CPU 利用率对比
# ============================================================
def plot_cpu():
    fig, ax = plt.subplots(figsize=(10, 6))

    langs = ["Go\n（本系统）", "Java\n（典型值）", "Python\n（典型值）"]
    cpu_vals = [27, 50, 95]
    colors = [C_GO, C_JAVA, C_PYTHON]

    bars = ax.bar(langs, cpu_vals, color=colors, edgecolor='white', width=0.5, zorder=3)

    for bar, val in zip(bars, cpu_vals):
        ax.annotate(f"{val:.0f}%",
                    xy=(bar.get_x() + bar.get_width() / 2, bar.get_height()),
                    xytext=(0, 5), textcoords="offset points",
                    ha="center", fontsize=13, fontweight='bold', color='#333333',
                    zorder=4,
                    bbox=dict(boxstyle='round,pad=0.15', facecolor='white',
                              edgecolor='none', alpha=0.9))

    ax.set_ylabel("CPU 占用率 (%)", fontsize=15)
    ax.set_title("同性能下 CPU 利用率对比（1000 QPS  sustained）", fontsize=18, fontweight='bold', pad=15)
    ax.set_ylim(0, 120)

    # 添加说明文字
    ax.text(0, 105, "编译型语言\n无解释器/JVM 开销", ha="center", fontsize=11, color=C_GO,
            bbox=dict(boxstyle='round,pad=0.3', facecolor='#F0F8FB', edgecolor=C_GO, alpha=0.85))
    ax.text(1, 105, "JIT 编译 + GC\n有一定运行时开销", ha="center", fontsize=11, color=C_JAVA,
            bbox=dict(boxstyle='round,pad=0.3', facecolor='#FFF5F0', edgecolor=C_JAVA, alpha=0.85))
    ax.text(2, 105, "GIL 限制 + 解释执行\n多进程通信开销大", ha="center", fontsize=11, color=C_PYTHON,
            bbox=dict(boxstyle='round,pad=0.3', facecolor='#F0F5FA', edgecolor=C_PYTHON, alpha=0.85))

    plt.tight_layout()
    save(fig, "ppt_lang_compare_cpu.png")


# ============================================================
# 图3：并发模型效率对比（单并发内存开销）
# ============================================================
def plot_concurrency():
    fig, ax = plt.subplots(figsize=(10, 6))

    # 使用对数坐标展示数量级差异
    langs = ["Go\ngoroutine", "Java\nThread", "Python\nProcess"]
    mem_per_unit = [0.002, 1, 50]  # MB
    colors = [C_GO, C_JAVA, C_PYTHON]

    bars = ax.bar(langs, mem_per_unit, color=colors, edgecolor='white', width=0.5, zorder=3)

    for bar, val in zip(bars, mem_per_unit):
        label = f"{val*1000:.0f}KB" if val < 1 else f"{val:.0f}MB"
        ax.annotate(label,
                    xy=(bar.get_x() + bar.get_width() / 2, bar.get_height()),
                    xytext=(0, 5), textcoords="offset points",
                    ha="center", fontsize=13, fontweight='bold', zorder=4)

    ax.set_ylabel("单并发内存开销 (MB)", fontsize=15)
    ax.set_title("并发模型效率：单并发内存占用对比", fontsize=18, fontweight='bold', pad=15)
    ax.set_ylim(0, 65)

    # 添加倍数对比线
    ax.annotate("", xy=(2, 50), xytext=(0, 0.002),
                arrowprops=dict(arrowstyle='<->', color='#666666', lw=1.5, ls='--'))
    ax.text(1.5, 28, "相差 25,000 倍", fontsize=12, color='#666666', fontweight='bold',
            ha='center', bbox=dict(boxstyle='round,pad=0.3', facecolor='white', edgecolor='#999999', alpha=0.9))

    plt.tight_layout()
    save(fig, "ppt_lang_compare_concurrency.png")


# ============================================================
# 图4：综合雷达图（可选）
# ============================================================
def plot_radar():
    from math import pi

    categories = ["内存效率", "CPU 效率", "并发效率", "启动速度", "部署便捷性"]
    N = len(categories)

    # 评分（5分制，主观但合理）
    go_scores = [5, 4.5, 5, 5, 5]
    java_scores = [2.5, 3.5, 3, 2.5, 3]
    py_scores = [2, 2, 2.5, 3.5, 3.5]

    angles = [n / float(N) * 2 * pi for n in range(N)]
    angles += angles[:1]

    go_scores += go_scores[:1]
    java_scores += java_scores[:1]
    py_scores += py_scores[:1]

    fig, ax = plt.subplots(figsize=(8, 8), subplot_kw=dict(polar=True))

    ax.plot(angles, go_scores, 'o-', linewidth=2.5, color=C_GO, label="Go")
    ax.fill(angles, go_scores, alpha=0.15, color=C_GO)

    ax.plot(angles, java_scores, 's-', linewidth=2.5, color=C_JAVA, label="Java")
    ax.fill(angles, java_scores, alpha=0.15, color=C_JAVA)

    ax.plot(angles, py_scores, '^-', linewidth=2.5, color=C_PYTHON, label="Python")
    ax.fill(angles, py_scores, alpha=0.15, color=C_PYTHON)

    ax.set_xticks(angles[:-1])
    ax.set_xticklabels(categories, fontsize=13)
    ax.set_ylim(0, 5)
    ax.set_title("语言选型综合评估", fontsize=18, fontweight='bold', pad=20)
    ax.legend(loc='upper right', bbox_to_anchor=(1.15, 1.1), framealpha=0.95, edgecolor='#cccccc')

    plt.tight_layout()
    save(fig, "ppt_lang_compare_radar.png")


def main():
    print("=" * 60)
    print("正在生成 Go vs Java vs Python 资源占用对比图...")
    print("=" * 60)

    plot_memory()
    plot_cpu()
    plot_concurrency()
    plot_radar()

    print("=" * 60)
    print("全部完成！输出文件:")
    print("  1. ppt_lang_compare_memory.png     — 内存占用对比")
    print("  2. ppt_lang_compare_cpu.png        — CPU 利用率对比")
    print("  3. ppt_lang_compare_concurrency.png — 并发模型效率对比")
    print("  4. ppt_lang_compare_radar.png      — 综合评估雷达图")
    print("=" * 60)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
