#!/usr/bin/env python3
"""
生成 NASA 格式的模拟日志数据

当无法下载真实的 NASA 日志时，使用此脚本生成格式相似的测试数据。

格式示例:
    199.72.81.55 - - [01/Jul/1995:00:00:01 -0400] "GET /history/apollo/ HTTP/1.0" 200 6245

使用方法:
    python generate_nasa_like_logs.py              # 生成 10万条 (默认)
    python generate_nasa_like_logs.py -n 1000000   # 生成 100万条
    python generate_nasa_like_logs.py -o ../data/my_nasa_logs.txt
"""

import argparse
import random
from datetime import datetime, timedelta
from pathlib import Path

# NASA 日志特征
NASA_IPS = [
    "199.72.81.55", "burger.letters.com", "199.120.110.21",
    "205.212.115.106", "130.110.74.81", "143.167.2.10",
    "163.205.16.75", "206.27.239.151", "130.61.130.40",
    "147.154.150.184", "204.62.245.32", "131.102.120.17",
    "134.153.50.9", "152.163.192.5", "198.133.29.18",
    "204.130.242.2", "163.206.104.34", "128.159.122.250",
    "141.102.80.151", "163.205.1.18", "128.217.62.1",
    "152.163.192.6", "163.205.16.23", "139.169.174.5",
]

NASA_PATHS = [
    "/history/apollo/", "/shuttle/countdown/", "/shuttle/missions/sts-71/mission-sts-71.html",
    "/shuttle/countdown/liftoff.html", "/history/skylab/skylab.html",
    "/history/apollo/apollo.html", "/history/apollo/apollo-13/apollo-13.html",
    "/shuttle/missions/sts-71/sts-71-patch-small.gif",
    "/images/NASA-logosmall.gif", "/images/WORLD-logosmall.gif",
    "/images/USA-logosmall.gif", "/images/MOSAIC-logosmall.gif",
    "/images/ksclogosmall.gif", "/history/apollo/images/apollo-logo1.gif",
    "/shuttle/countdown/video/livevideo.gif", "/shuttle/resources/orbiters/endeavour.html",
    "/shuttle/countdown/countdown.html", "/elv/DELTA/uncons.htm",
    "/icons/menu.xbm", "/icons/blank.xbm", "/icons/image.xbm",
    "/shuttle/technology/sts-newsref/stsref-toc.html", "/htbin/cdt_main.pl",
    "/shuttle/missions/sts-71/movies/movies.html", "/shuttle/missions/sts-71/images/images.html",
    "/history/apollo/apollo-11/apollo-11.html", "/history/gemini/gemini.html",
    "/history/mercury/mercury.html", "/www/faq.html", "/elv/elvpage.htm",
]

NASA_STATUS_CODES = [200, 304, 302, 404, 403, 500]
NASA_STATUS_WEIGHTS = [75, 15, 5, 3, 1, 1]  # NASA 日志中 200 占绝大多数

# HTTP 方法分布（模拟真实场景）
NASA_METHODS = ["GET", "POST", "HEAD", "PUT", "DELETE"]
NASA_METHOD_WEIGHTS = [85, 10, 3, 1, 1]  # GET 占绝大多数，但也有少量其他方法

# 用户代理（模拟 1995 年的浏览器）
NASA_USER_AGENTS = [
    "Mozilla/1.0 (Windows 3.1)",
    "Mozilla/1.1 (Windows 95)",
    "Mozilla/1.22 (Windows 95)",
    "NCSA Mosaic/2.0 (Windows 3.1)",
    "NCSA Mosaic/2.6 (X11; SunOS 5.3 sun4m)",
    "Lynx/2.4 (Unix)",
    "libwww/2.14",
    "-"
]


def generate_nasa_log_line(timestamp, seq):
    """生成一条 NASA 格式的日志（扩展版，支持系统解析）"""
    # 随机选择 IP
    host = random.choice(NASA_IPS)
    
    # 时间戳格式: [01/Jul/1995:00:00:01 -0400]
    time_str = timestamp.strftime("%d/%b/%Y:%H:%M:%S -0400")
    
    # 随机请求方法（不再只是 GET）
    method = random.choices(NASA_METHODS, weights=NASA_METHOD_WEIGHTS)[0]
    path = random.choice(NASA_PATHS)
    protocol = "HTTP/1.0"
    
    # 随机状态码和大小
    status = random.choices(NASA_STATUS_CODES, weights=NASA_STATUS_WEIGHTS)[0]
    
    # 根据状态码决定大小
    if status == 200:
        size = random.randint(100, 10000)
    elif status == 304:
        size = 0  # Not Modified
    elif status == 404:
        size = random.randint(100, 500)
    else:
        size = random.randint(100, 2000)
    
    # 添加 referer 和 user_agent（部分请求有）
    referer = "-"
    if random.random() > 0.3:  # 70% 的请求有 referer
        referer = random.choice(["-", "https://www.nasa.gov/", "https://www.google.com/"])
    
    user_agent = random.choice(NASA_USER_AGENTS)
    
    # 添加响应时间（毫秒，转换为秒）
    response_time = round(random.uniform(0.001, 2.0), 3)
    
    # 返回完整格式（兼容 Nginx 解析）
    return f'{host} - - [{time_str}] "{method} {path} {protocol}" {status} {size} "{referer}" "{user_agent}" "{response_time}"'


def generate_logs(count, output_path):
    """生成指定数量的 NASA 格式日志"""
    print(f"[INFO] 正在生成 {count:,} 条 NASA 格式日志...")
    print(f"[INFO] 输出文件: {output_path}")
    
    # 从 1995年7月1日开始
    start_time = datetime(1995, 7, 1, 0, 0, 0)
    
    output_path = Path(output_path)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    
    with open(output_path, 'w', encoding='utf-8') as f:
        for i in range(count):
            # 时间均匀分布在整个7月
            seconds_offset = random.randint(0, 31 * 24 * 3600 - 1)
            timestamp = start_time + timedelta(seconds=seconds_offset)
            
            line = generate_nasa_log_line(timestamp, i)
            f.write(line + '\n')
            
            # 进度显示
            if (i + 1) % 100000 == 0:
                print(f"  已生成: {(i + 1):,} 条")
    
    file_size_mb = output_path.stat().st_size / (1024 * 1024)
    print(f"\n✅ 生成完成!")
    print(f"   文件: {output_path}")
    print(f"   大小: {file_size_mb:.1f} MB")
    print(f"   条数: {count:,}")


def show_sample(output_path, n=5):
    """显示样例数据"""
    print(f"\n[样例数据 - 前 {n} 行]")
    with open(output_path, 'r', encoding='utf-8') as f:
        for i, line in enumerate(f):
            if i >= n:
                break
            print(f"  {line.rstrip()}")


def main():
    parser = argparse.ArgumentParser(
        description='生成 NASA 格式的模拟日志数据',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
    python generate_nasa_like_logs.py              # 生成 10万条
    python generate_nasa_like_logs.py -n 1000000   # 生成 100万条 (~100MB)
    python generate_nasa_like_logs.py -n 2000000 -o ../data/nasa_2m.txt
        """
    )
    parser.add_argument('-n', '--count', type=int, default=100000,
                       help='生成日志条数 (默认: 100000)')
    parser.add_argument('-o', '--output', type=str, default='../data/NASA_access_log_Jul95_simulated.txt',
                       help='输出文件路径 (默认: ../data/NASA_access_log_Jul95_simulated.txt)')
    args = parser.parse_args()

    print("=" * 60)
    print("NASA 格式模拟日志生成器")
    print("=" * 60)
    print(f"目标条数: {args.count:,}")
    print(f"输出路径: {args.output}")
    print()

    generate_logs(args.count, args.output)
    show_sample(args.output)
    
    print("\n" + "=" * 60)
    print("使用方式:")
    print("=" * 60)
    print(f"\n  python stress_test.py -file {args.output} -total 100000 -rate 50")
    print("\n注意: 这是模拟数据，格式与真实 NASA 日志一致，")
    print("      但内容基于随机选择，用于测试系统性能。")


if __name__ == "__main__":
    main()
