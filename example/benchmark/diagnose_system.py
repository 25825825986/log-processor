#!/usr/bin/env python3
"""
系统诊断工具 - 帮助分析性能瓶颈和丢包原因

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
        # 通过查询接口获取日志数量
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
        print("❌ 无法连接到服务器，请确保服务已启动")
        return
    
    total_logs = fetch_stats()
    
    print("=" * 60)
    print("系统诊断报告")
    print("=" * 60)
    
    # 系统基本信息
    print(f"\n📊 基本信息:")
    print(f"  运行时间: {status.get('uptime', 'N/A')}")
    print(f"  日志总数: {total_logs:,}")
    
    # 处理器状态
    processor = status.get('processor', {})
    print(f"\n⚙️  处理器状态:")
    print(f"  Worker 数量: {processor.get('worker_count', 'N/A')}")
    print(f"  批处理大小: {processor.get('batch_size', 'N/A')}")
    
    # 队列状态（从日志中解析）
    print(f"\n📥 队列状态 (需要查看服务器日志):")
    print(f"  Processor input queue: 查看日志中的 '[WARN] Processor input queue full'")
    print(f"  AsyncStorage buffer: 查看日志中的 '[WARN] AsyncStorage 队列满'")
    
    # 性能评估
    print(f"\n📈 性能评估:")
    if processor.get('batch_size', 100) < 1000:
        print(f"  ⚠️  batch_size 较小 ({processor.get('batch_size')})，建议调整到 1000-2000")
    else:
        print(f"  ✅ batch_size 设置合理 ({processor.get('batch_size')})")
    
    print(f"\n🔧 建议:")
    print(f"  1. 如果出现丢包，检查服务器日志中的 '[WARN]' 信息")
    print(f"  2. 使用 'go run cmd/server/main.go -config config.optimized.json' 启动")
    print(f"  3. 降低压测速率: python stress_test.py -rate 20 -c 10")


def monitor():
    """持续监控"""
    print("开始监控 (按 Ctrl+C 停止)...")
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
                print(f"\r日志: {count:,} | QPS: {qps:6.0f} | Workers: {proc.get('worker_count', 'N/A')} | Batch: {proc.get('batch_size', 'N/A')}", end='')
            
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
