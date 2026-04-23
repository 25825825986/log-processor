#!/usr/bin/env python3
"""重新生成 benchmark_comparison.png，本系统持续吞吐改为 800"""

import matplotlib.pyplot as plt
import numpy as np

# 设置中文字体
plt.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'Arial Unicode MS']
plt.rcParams['axes.unicode_minus'] = False

# 数据
labels = ['CPU平均占用率 (%)', '内存占用 (MB)', '持续吞吐 (QPS)']
ours = [25, 180, 800]
elk = [60, 2800, 600]

x = np.arange(len(labels))
width = 0.35

fig, ax = plt.subplots(figsize=(10, 6))

# 柱子颜色（匹配原图）
bars1 = ax.bar(x - width/2, ours, width, label='本系统', color='#4472C4', edgecolor='none')
bars2 = ax.bar(x + width/2, elk, width, label='ELK Stack 单机版', color='#ED7D31', edgecolor='none')

# 标题
ax.set_title('本系统与 ELK Stack 单机版性能横向对比\n（同等硬件环境：4核CPU / 8GB内存 / SSD）', fontsize=14)
ax.set_ylabel('数值', fontsize=12)
ax.set_xticks(x)
ax.set_xticklabels(labels, fontsize=11)
ax.legend(fontsize=11)

# 在柱子上方标注数值
for bar in bars1:
    height = bar.get_height()
    ax.annotate(f'{int(height)}',
                xy=(bar.get_x() + bar.get_width() / 2, height),
                xytext=(0, 3),
                textcoords="offset points",
                ha='center', va='bottom', fontsize=10)

for bar in bars2:
    height = bar.get_height()
    ax.annotate(f'{int(height)}',
                xy=(bar.get_x() + bar.get_width() / 2, height),
                xytext=(0, 3),
                textcoords="offset points",
                ha='center', va='bottom', fontsize=10)

# 调整y轴范围，让内存2800能完整显示
ax.set_ylim(0, 3200)

# 添加网格线
ax.yaxis.grid(True, linestyle='-', alpha=0.3)
ax.set_axisbelow(True)

plt.tight_layout()
plt.savefig('benchmark_comparison.png', dpi=300, bbox_inches='tight')
print("已保存 benchmark_comparison.png")
