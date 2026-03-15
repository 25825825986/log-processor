#!/usr/bin/env python3
"""
生成 NASA 风格访问日志，用于本地压测。
"""

from __future__ import annotations

import argparse
import random
from datetime import datetime, timedelta
from pathlib import Path

HOSTS = [
    "199.72.81.55",
    "205.212.115.106",
    "130.110.74.81",
    "128.159.122.250",
    "198.133.29.18",
]
PATHS = [
    "/history/apollo/",
    "/shuttle/countdown/",
    "/images/NASA-logosmall.gif",
    "/images/WORLD-logosmall.gif",
    "/shuttle/missions/sts-71/mission-sts-71.html",
    "/icons/menu.xbm",
    "/robots.txt",
]
METHODS = ["GET", "POST", "HEAD"]
STATUS = [200, 200, 200, 200, 200, 304, 302, 404, 500]


def gen_line(ts: datetime) -> str:
    host = random.choice(HOSTS)
    path = random.choice(PATHS)
    method = random.choice(METHODS)
    code = random.choice(STATUS)
    size = 0 if code == 304 else random.randint(80, 12000)
    stamp = ts.strftime("%d/%b/%Y:%H:%M:%S -0400")
    return f'{host} - - [{stamp}] "{method} {path} HTTP/1.0" {code} {size}'


def main() -> int:
    parser = argparse.ArgumentParser(description="生成 NASA 风格日志。")
    parser.add_argument("-n", "--count", type=int, default=100000, help="日志条数")
    parser.add_argument(
        "-o",
        "--output",
        default="../data/NASA_access_log_Jul95_simulated.txt",
        help="输出路径",
    )
    args = parser.parse_args()

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    start = datetime(1995, 7, 1, 0, 0, 0)
    total_seconds = 31 * 24 * 3600

    with output.open("w", encoding="utf-8") as f:
        for i in range(args.count):
            ts = start + timedelta(seconds=random.randint(0, total_seconds - 1))
            f.write(gen_line(ts) + "\n")
            if (i + 1) % 100000 == 0:
                print(f"[INFO] 已生成 {i + 1:,} 条")

    size_mb = output.stat().st_size / (1024 * 1024)
    print(f"[DONE] 已生成 {args.count:,} 条 -> {output} ({size_mb:.1f} MB)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
