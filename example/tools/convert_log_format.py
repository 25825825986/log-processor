#!/usr/bin/env python3
"""
将常见日志格式转换为 nginx 或 json 行格式。
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any

NASA_RE = re.compile(
    r'^(\S+)\s+-\s+-\s+\[([^\]]+)\]\s+"([^"]+)"\s+(\d+)\s+(\S+)'
)


def parse_nasa_line(line: str) -> dict[str, Any] | None:
    m = NASA_RE.match(line)
    if not m:
        return None
    ip, ts, request, status, size = m.groups()
    req_parts = request.split()
    return {
        "client_ip": ip,
        "timestamp": ts,
        "method": req_parts[0] if len(req_parts) > 0 else "GET",
        "path": req_parts[1] if len(req_parts) > 1 else "/",
        "protocol": req_parts[2] if len(req_parts) > 2 else "HTTP/1.0",
        "status_code": int(status),
        "response_size": int(size) if size.isdigit() else 0,
    }


def parse_csv_line(line: str, delimiter: str = ",") -> dict[str, Any] | None:
    parts = [p.strip() for p in line.split(delimiter)]
    if len(parts) < 5:
        return None
    try:
        status = int(parts[4])
    except ValueError:
        return None
    return {
        "client_ip": parts[0],
        "timestamp": parts[1],
        "method": parts[2],
        "path": parts[3],
        "status_code": status,
        "response_size": int(parts[5]) if len(parts) > 5 and parts[5].isdigit() else 0,
    }


def detect_input_format(first_line: str) -> str:
    if not first_line:
        return "nginx"
    if first_line.startswith("{"):
        return "json"
    if NASA_RE.match(first_line):
        return "nasa"
    if "," in first_line and len(first_line.split(",")) >= 5:
        return "csv"
    return "nginx"


def to_nginx(log: dict[str, Any]) -> str:
    ts = str(log.get("timestamp", "")).strip()
    if ts and " " not in ts and "T" in ts:
        # 保持 ISO 时间戳原样，便于解析器自动识别
        pass
    elif ts and not ts.endswith(tuple(["+0000", "+0800", "-0400"])):
        ts = f"{ts} +0800"
    method = str(log.get("method", "GET"))
    path = str(log.get("path", "/"))
    proto = str(log.get("protocol", "HTTP/1.1"))
    status = int(log.get("status_code", 200))
    size = int(log.get("response_size", 0))
    ip = str(log.get("client_ip", "127.0.0.1"))
    return f'{ip} - - [{ts}] "{method} {path} {proto}" {status} {size}'


def convert_file(
    input_file: Path,
    output_file: Path,
    input_format: str,
    output_format: str,
) -> tuple[int, int]:
    lines_total = 0
    lines_ok = 0

    first = ""
    with input_file.open("r", encoding="utf-8", errors="ignore") as f:
        for raw in f:
            first = raw.strip()
            if first:
                break

    if input_format == "auto":
        input_format = detect_input_format(first)
        print(f"[INFO] detected input format: {input_format}")

    with input_file.open("r", encoding="utf-8", errors="ignore") as fin, output_file.open(
        "w", encoding="utf-8"
    ) as fout:
        for raw in fin:
            line = raw.strip()
            if not line:
                continue
            lines_total += 1

            record: dict[str, Any] | None = None
            try:
                if input_format == "json":
                    value = json.loads(line)
                    if isinstance(value, dict):
                        record = value
                elif input_format == "nasa":
                    record = parse_nasa_line(line)
                elif input_format == "csv":
                    record = parse_csv_line(line)
                elif input_format == "nginx":
                    # 已是 nginx 行格式，直接透传
                    fout.write(line + "\n")
                    lines_ok += 1
                    continue
            except Exception:
                record = None

            if not record:
                continue

            if output_format == "json":
                fout.write(json.dumps(record, ensure_ascii=False) + "\n")
            else:
                fout.write(to_nginx(record) + "\n")
            lines_ok += 1

    return lines_total, lines_ok


def main() -> int:
    parser = argparse.ArgumentParser(description="日志格式转换工具。")
    parser.add_argument("input", help="输入文件路径")
    parser.add_argument("output", help="输出文件路径")
    parser.add_argument(
        "--input-format",
        choices=["auto", "nasa", "json", "csv", "nginx"],
        default="auto",
    )
    parser.add_argument("--output-format", choices=["nginx", "json"], default="nginx")
    args = parser.parse_args()

    input_file = Path(args.input)
    output_file = Path(args.output)
    if not input_file.exists():
        print(f"[ERROR] 输入文件不存在: {input_file}")
        return 1

    total, ok = convert_file(input_file, output_file, args.input_format, args.output_format)
    print(f"[DONE] 已转换 {ok}/{total} 行 -> {output_file}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
