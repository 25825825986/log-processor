#!/usr/bin/env python3
"""
下载并校验 NASA 1995-07 访问日志数据集。
"""

from __future__ import annotations

import argparse
import gzip
import shutil
import sys
import urllib.request
from pathlib import Path

SOURCES = [
    "https://archive.org/download/NASA_access_log_Jul95/NASA_access_log_Jul95.gz",
    "http://ita.ee.lbl.gov/traces/NASA_access_log_Jul95.gz",
]

GZ_NAME = "NASA_access_log_Jul95.gz"
TXT_NAME = "NASA_access_log_Jul95.txt"


def download(url: str, dst: Path) -> bool:
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "LogProcessor/1.0"})
        with urllib.request.urlopen(req, timeout=180) as resp, dst.open("wb") as out:
            shutil.copyfileobj(resp, out)
        return True
    except Exception as exc:
        print(f"[WARN] 从 {url} 下载失败: {exc}")
        return False


def decompress(src: Path, dst: Path) -> None:
    with gzip.open(src, "rb") as fin, dst.open("wb") as fout:
        shutil.copyfileobj(fin, fout)


def verify(path: Path) -> int:
    if not path.exists():
        print(f"[FAIL] 文件不存在: {path}")
        return 1
    lines = 0
    with path.open("r", encoding="utf-8", errors="ignore") as f:
        for _ in f:
            lines += 1
    size_mb = path.stat().st_size / (1024 * 1024)
    print(f"[OK] {path}")
    print(f"     size:  {size_mb:.1f} MB")
    print(f"     lines: {lines:,}")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description="下载 NASA Jul95 访问日志。")
    parser.add_argument("-o", "--output", default="../data", help="输出目录")
    parser.add_argument("--verify", action="store_true", help="仅校验本地 txt 文件")
    parser.add_argument("--keep-gz", action="store_true", help="保留 gz 压缩包")
    args = parser.parse_args()

    out_dir = Path(args.output).resolve()
    out_dir.mkdir(parents=True, exist_ok=True)
    gz_path = out_dir / GZ_NAME
    txt_path = out_dir / TXT_NAME

    if args.verify:
        return verify(txt_path)

    if txt_path.exists():
        print(f"[INFO] 已存在 txt 文件: {txt_path}")
        return verify(txt_path)

    if not gz_path.exists():
        ok = False
        for url in SOURCES:
            print(f"[INFO] 尝试下载源: {url}")
            if download(url, gz_path):
                ok = True
                break
        if not ok:
            print("[FAIL] 所有下载源均失败")
            return 1

    print(f"[INFO] 正在解压 {gz_path} -> {txt_path}")
    decompress(gz_path, txt_path)

    if not args.keep_gz and gz_path.exists():
        gz_path.unlink()
    return verify(txt_path)


if __name__ == "__main__":
    sys.exit(main())
