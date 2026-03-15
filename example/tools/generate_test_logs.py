#!/usr/bin/env python3
"""
生成用于快速手工验证的模拟日志。
"""

from __future__ import annotations

import argparse
import json
import random
from datetime import datetime, timedelta
from pathlib import Path

IPS = [
    "127.0.0.1",
    "192.168.1.10",
    "10.0.0.12",
    "172.16.2.8",
]
METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"]
PATHS = [
    "/",
    "/api/users",
    "/api/orders",
    "/api/login",
    "/health",
    "/metrics",
]
STATUS_CODES = [200, 201, 204, 301, 400, 401, 403, 404, 500, 502]


def random_time(days: int = 30) -> datetime:
    base = datetime.now() - timedelta(days=days)
    return base + timedelta(seconds=random.randint(0, days * 24 * 3600))


def gen_nginx() -> str:
    ts = random_time().strftime("%d/%b/%Y:%H:%M:%S +0800")
    ip = random.choice(IPS)
    method = random.choice(METHODS)
    path = random.choice(PATHS)
    code = random.choice(STATUS_CODES)
    size = random.randint(100, 10000)
    return f'{ip} - - [{ts}] "{method} {path} HTTP/1.1" {code} {size}'


def gen_json() -> str:
    data = {
        "timestamp": random_time().isoformat(),
        "client_ip": random.choice(IPS),
        "method": random.choice(METHODS),
        "path": random.choice(PATHS),
        "status_code": random.choice(STATUS_CODES),
        "response_time": random.randint(5, 2000),
    }
    return json.dumps(data, ensure_ascii=False)


def gen_csv() -> str:
    ts = random_time().strftime("%Y-%m-%d %H:%M:%S")
    return ",".join(
        [
            random.choice(IPS),
            ts,
            random.choice(METHODS),
            random.choice(PATHS),
            str(random.choice(STATUS_CODES)),
            str(random.randint(100, 10000)),
            str(random.randint(5, 2000)),
        ]
    )


def gen_syslog() -> str:
    ts = random_time().strftime("%b %d %H:%M:%S")
    host = f"srv-{random.randint(1, 5)}"
    ip = random.choice(IPS)
    method = random.choice(METHODS)
    path = random.choice(PATHS)
    code = random.choice(STATUS_CODES)
    size = random.randint(100, 10000)
    return f"{ts} {host} app[{random.randint(1000, 9999)}]: {ip} {method} {path} {code} {size}"


def main() -> int:
    parser = argparse.ArgumentParser(description="生成模拟测试日志。")
    parser.add_argument("-n", "--count", type=int, default=1000)
    parser.add_argument(
        "-f",
        "--format",
        choices=["nginx", "json", "csv", "syslog"],
        default="nginx",
    )
    parser.add_argument("-o", "--output", required=True, help="输出文件路径")
    args = parser.parse_args()

    factory = {
        "nginx": gen_nginx,
        "json": gen_json,
        "csv": gen_csv,
        "syslog": gen_syslog,
    }[args.format]

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    with output.open("w", encoding="utf-8") as f:
        for _ in range(args.count):
            f.write(factory() + "\n")

    print(f"[DONE] 已生成 {args.count} 条 {args.format} 日志 -> {output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
