#!/usr/bin/env python3
"""
一键压测脚本：
1. 调用系统内置压测接口 /api/benchmark/run
2. 生成 JSON + Markdown 报告
"""

from __future__ import annotations

import argparse
import json
import urllib.request
from datetime import datetime
from pathlib import Path


def post_json(url: str, payload: dict) -> dict:
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=data,
        method="POST",
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=600) as resp:
        return json.loads(resp.read().decode("utf-8"))


def build_markdown(report: dict) -> str:
    processor = report.get("processor_delta", {})
    lines = [
        "# 压测报告",
        "",
        f"- 开始时间: `{report.get('started_at', '-')}`",
        f"- 结束时间: `{report.get('finished_at', '-')}`",
        f"- 持续时间: `{report.get('duration_seconds', '-')}` 秒",
        f"- 并发协程: `{report.get('workers', '-')}`",
        f"- 目标QPS: `{report.get('target_qps', '-')}`",
        "",
        "## 结果总览",
        "",
        f"- 提交总数: `{report.get('submitted', 0)}`",
        f"- 拒绝总数: `{report.get('rejected', 0)}`",
        f"- 接收率: `{report.get('accept_rate', 0):.2f}%`",
        f"- 提交QPS: `{report.get('submit_qps', 0):.2f}`",
        f"- 入库新增: `{report.get('stored_added', 0)}`",
        f"- 入库QPS: `{report.get('stored_qps', 0):.2f}`",
        "",
        "## 处理器增量",
        "",
        f"- received_delta: `{processor.get('received_delta', 0)}`",
        f"- processed_delta: `{processor.get('processed_delta', 0)}`",
        f"- dropped_delta: `{processor.get('dropped_delta', 0)}`",
        f"- parse_error_delta: `{processor.get('parse_error_delta', 0)}`",
        f"- spill_delta: `{processor.get('spill_delta', 0)}`",
        f"- overflow_recovered_delta: `{processor.get('overflow_recovered_delta', 0)}`",
        f"- overflow_pending: `{processor.get('overflow_pending', 0)}`",
        f"- overflow_dropped_delta: `{processor.get('overflow_dropped_delta', 0)}`",
        f"- overflow_write_err_delta: `{processor.get('overflow_write_err_delta', 0)}`",
        "",
    ]
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description="调用系统一键压测接口并生成报告")
    parser.add_argument("--base", default="http://localhost:8080", help="系统地址")
    parser.add_argument("--duration", type=int, default=10, help="压测持续秒数")
    parser.add_argument("--workers", type=int, default=20, help="并发协程数")
    parser.add_argument("--target-qps", type=int, default=5000, help="目标QPS，0为不限速")
    parser.add_argument(
        "--output-dir",
        default="example/benchmark/reports",
        help="报告输出目录",
    )
    args = parser.parse_args()

    payload = {
        "duration_seconds": args.duration,
        "workers": args.workers,
        "target_qps": args.target_qps,
    }

    print("=" * 60)
    print("执行一键压测")
    print("=" * 60)
    print(f"base={args.base}")
    print(
        f"duration={args.duration}s workers={args.workers} target_qps={args.target_qps}"
    )

    report = post_json(f"{args.base}/api/benchmark/run", payload)

    out_dir = Path(args.output_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")

    json_path = out_dir / f"benchmark_{ts}.json"
    md_path = out_dir / f"benchmark_{ts}.md"

    json_path.write_text(
        json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    md_path.write_text(build_markdown(report), encoding="utf-8")

    print("\n压测完成")
    print(f"- JSON 报告: {json_path}")
    print(f"- Markdown 报告: {md_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
