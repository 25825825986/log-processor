#!/usr/bin/env python3
"""
系统诊断工具

使用方法:
    python diagnose_system.py              # 实时监控系统状态
    python diagnose_system.py --once       # 只显示一次状态
"""

import argparse
import json
import sys
import time
import urllib.request


def fetch_status():
    """获取系统状态"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/status', method='GET')
        with urllib.request.urlopen(req, timeout=5) as resp:
            return json.loads(resp.read().decode())
    except Exception as e:
        print(f"[ERROR] 无法获取状态: {e}")
        return None


def fetch_stats():
    """获取处理器统计"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/logs?limit=1', method='GET')
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            return data.get('total', 0)
    except Exception as e:
        print(f"[ERROR] 无法获取统计: {e}")
        return 0


def diagnose_once():
    """诊断一次"""
    status = fetch_status()
    if not status:
        print("[X] 无法连接到服务器，请确保服务已启动")
        return
    
    total_logs = fetch_stats()
    
    print("=" * 60)
    print("系统诊断报告")
    print("=" * 60)
    
    # 系统基本信息
    print(f"\n[基本信息]")
    print(f"  运行时间: {status.get('uptime', 'N/A')}")
    print(f"  日志总数: {total_logs:,}")
    
    # 处理器状态
    processor = status.get('processor', {})
    print(f"\n[处理器状态]")
    print(f"  Worker 数量: {processor.get('worker_count', 'N/A')}")
    print(f"  批处理大小: {processor.get('batch_size', 'N/A')}")
    print(f"  批处理超时: {processor.get('batch_timeout', 'N/A')} ms")
    
    # 队列状态
    print(f"\n[队列状态]")
    print(f"  Input Queue: {processor.get('input_queue_size', 'N/A')}")
    print(f"  Output Queue: {processor.get('output_queue_size', 'N/A')}")
    
    if processor.get('dropped_count', 0) > 0:
        print(f"  [警告] 丢弃总数: {processor.get('dropped_count')} 条")
    
    # 性能评估
    print(f"\n[性能评估]")
    if processor.get('batch_size', 100) < 1000:
        print(f"  [警告] batch_size 较小 ({processor.get('batch_size')})，建议调整到 1000-2000")
    else:
        print(f"  [OK] batch_size 设置合理 ({processor.get('batch_size')})")
    
    print(f"\n[建议]")
    print(f"  1. 如果出现丢包，降低压测速率: python stress_test.py -rate 20 -c 10")
    print(f"  2. SQLite 单线程写入上限约 500-800 QPS")


def monitor():
    """持续监控"""
    print("开始监控 (按 Ctrl+C 停止)...")
    print("=" * 60)
    print(f"{'时间':<10} {'日志总数':>10} {'QPS':>8} {'Input':>8} {'Output':>8}")
    print("=" * 60)
    
    prev_count = 0
    prev_time = time.time()
    
    try:
        while True:
            count = fetch_stats()
            now = time.time()
            
            qps = (count - prev_count) / (now - prev_time) if now > prev_time else 0
            
            status = fetch_status()
            if status:
                proc = status.get('processor', {})
                time_str = time.strftime('%H:%M:%S')
                input_qsize = proc.get('input_queue_size', 0)
                output_qsize = proc.get('output_queue_size', 0)
                print(f"\r{time_str:<10} {count:>10,} {qps:>8.0f} {input_qsize:>8} {output_qsize:>8}", end='')
            
            prev_count = count
            prev_time = now
            time.sleep(1)
    except KeyboardInterrupt:
        print("\n\n监控已停止")


def main():
    parser = argparse.ArgumentParser(description='系统诊断工具')
    parser.add_argument('--once', action='store_true', help='只显示一次状态')
    args = parser.parse_args()
    
    if args.once:
        diagnose_once()
    else:
        monitor()


if __name__ == "__main__":
    main()
