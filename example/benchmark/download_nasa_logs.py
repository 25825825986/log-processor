#!/usr/bin/env python3
"""
NASA HTTP Logs 下载工具

下载来源:
    - Internet Archive (推荐): 稳定、快速
    - 原始FTP: ftp://ita.ee.lbl.gov/traces/

数据集信息:
    - NASA Kennedy Space Center WWW server
    - 时间: 1995年7月1日 - 7月31日
    - 原始大小: 约 200MB (压缩后 20MB)
    - 记录数: 约 190万条 HTTP 请求
    - 格式: Apache/Nginx Combined Log Format

使用示例:
    python download_nasa_logs.py              # 下载到默认位置
    python download_nasa_logs.py -o ./data    # 下载到指定目录
    python download_nasa_logs.py --verify     # 验证本地文件完整性
"""

import argparse
import gzip
import hashlib
import os
import shutil
import sys
import urllib.request
from pathlib import Path

# NASA 日志下载地址（多镜像源）
MIRROR_SOURCES = [
    # Internet Archive (国际)
    "https://archive.org/download/NASA_access_log_Jul95/NASA_access_log_Jul95.gz",
    # 官方 FTP (国际)
    "http://ita.ee.lbl.gov/traces/NASA_access_log_Jul95.gz",
    # GitHub Raw (相对稳定)
    "https://raw.githubusercontent.com/logpai/loghub/master/NASA/NASA_access_log_Jul95.gz",
]

# 文件校验信息（基于官方数据集）
FILE_INFO = {
    "filename": "NASA_access_log_Jul95.gz",
    "decompressed": "NASA_access_log_Jul95.txt",
    "compressed_size_mb": 20.7,
    "decompressed_size_mb": 200.0,
    "expected_lines": 1891711,
}


def print_progress(block_num, block_size, total_size):
    """下载进度回调"""
    downloaded = block_num * block_size
    percent = min(100, downloaded * 100 / total_size)
    mb = downloaded / (1024 * 1024)
    total_mb = total_size / (1024 * 1024)
    sys.stdout.write(f"\r[下载] {percent:.1f}% ({mb:.1f}/{total_mb:.1f} MB)")
    sys.stdout.flush()


def download_file(url, output_path, timeout=120):
    """下载文件到指定路径"""
    print(f"[INFO] 正在下载: {url}")
    try:
        # 设置超时和 User-Agent
        headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
        }
        request = urllib.request.Request(url, headers=headers)
        
        with urllib.request.urlopen(request, timeout=timeout) as response:
            total_size = int(response.headers.get('Content-Length', 0))
            downloaded = 0
            block_size = 8192
            
            with open(output_path, 'wb') as f:
                while True:
                    chunk = response.read(block_size)
                    if not chunk:
                        break
                    f.write(chunk)
                    downloaded += len(chunk)
                    if total_size > 0:
                        print_progress(downloaded // block_size, block_size, total_size)
        
        print()  # 换行
        return True
    except Exception as e:
        print(f"\n[ERROR] 下载失败: {e}")
        return False


def decompress_gz(gz_path, output_path):
    """解压 gzip 文件"""
    print(f"[INFO] 正在解压: {gz_path}")
    try:
        with gzip.open(gz_path, 'rb') as f_in:
            with open(output_path, 'wb') as f_out:
                shutil.copyfileobj(f_in, f_out)
        
        # 获取解压后大小
        size_mb = os.path.getsize(output_path) / (1024 * 1024)
        print(f"[INFO] 解压完成: {size_mb:.1f} MB")
        return True
    except Exception as e:
        print(f"[ERROR] 解压失败: {e}")
        return False


def verify_file(filepath):
    """验证文件完整性"""
    if not os.path.exists(filepath):
        print(f"[ERROR] 文件不存在: {filepath}")
        return False
    
    size_mb = os.path.getsize(filepath) / (1024 * 1024)
    lines = 0
    
    print(f"[INFO] 正在验证文件: {filepath}")
    print(f"       文件大小: {size_mb:.1f} MB")
    
    # 统计行数
    try:
        with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
            for _ in f:
                lines += 1
        print(f"       日志条数: {lines:,}")
        
        # 显示前3行样例
        print("\n[样例数据]")
        with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
            for i, line in enumerate(f):
                if i >= 3:
                    break
                print(f"  {i+1}. {line[:100]}...")
        
        return True
    except Exception as e:
        print(f"[ERROR] 验证失败: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(
        description='下载 NASA HTTP Logs 数据集',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
    python download_nasa_logs.py
    python download_nasa_logs.py -o ../data
    python download_nasa_logs.py --verify
        """
    )
    parser.add_argument('-o', '--output', 
                       default='../data',
                       help='输出目录 (默认: ../data)')
    parser.add_argument('--verify', 
                       action='store_true',
                       help='验证已下载文件的完整性')
    parser.add_argument('--keep-gz', 
                       action='store_true',
                       help='保留压缩文件')
    args = parser.parse_args()

    # 确保输出目录存在
    output_dir = Path(args.output).resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    
    gz_path = output_dir / FILE_INFO["filename"]
    txt_path = output_dir / FILE_INFO["decompressed"]

    print("=" * 60)
    print("NASA HTTP Logs 下载工具")
    print("=" * 60)
    print(f"输出目录: {output_dir}")
    print()

    # 验证模式
    if args.verify:
        if verify_file(txt_path):
            print("\n✅ 文件验证通过")
            return 0
        else:
            print("\n❌ 文件验证失败")
            return 1

    # 检查是否已存在
    if txt_path.exists():
        print(f"[INFO] 文件已存在: {txt_path}")
        print("[INFO] 使用 --verify 验证文件完整性")
        print("[INFO] 如需重新下载，请先删除现有文件")
        return 0

    # 下载压缩文件
    if not gz_path.exists():
        downloaded = False
        for i, url in enumerate(MIRROR_SOURCES):
            print(f"\n[{i+1}/{len(MIRROR_SOURCES)}] 尝试镜像源...")
            if download_file(url, gz_path, timeout=180):
                downloaded = True
                break
            else:
                # 删除失败的临时文件
                if gz_path.exists():
                    gz_path.unlink()
        
        if not downloaded:
            print("\n" + "=" * 60)
            print("❌ 所有下载源均失败")
            print("=" * 60)
            print("\n可能原因：")
            print("  1. 网络连接问题（国际网站访问受限）")
            print("  2. 防火墙/代理限制")
            print("\n解决方案：")
            print("  1. 设置代理环境变量: set HTTP_PROXY=http://proxy:port")
            print("  2. 手动下载后放置到:", output_dir)
            print("     下载地址:", MIRROR_SOURCES[0])
            print("  3. 使用项目自带测试数据: ../data/test_logs.txt")
            return 1
    else:
        print(f"[INFO] 使用已存在的压缩文件: {gz_path}")

    # 解压
    if not decompress_gz(gz_path, txt_path):
        return 1

    # 清理压缩文件
    if not args.keep_gz:
        print(f"[INFO] 删除压缩文件: {gz_path}")
        gz_path.unlink()

    # 验证
    print()
    if verify_file(txt_path):
        print("\n" + "=" * 60)
        print("✅ 下载并验证成功!")
        print("=" * 60)
        print(f"\n文件位置: {txt_path}")
        print(f"\n使用方法:")
        print(f"  python stress_test.py -file {txt_path} -total 100000")
        return 0
    else:
        print("\n⚠️  文件下载完成但验证失败")
        return 1


if __name__ == "__main__":
    sys.exit(main())
