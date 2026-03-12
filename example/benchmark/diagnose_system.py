#!/usr/bin/env python3
"""
系统诊断工具 (v2.0 - 支持容错机制监控)

使用方法:
    python diagnose_system.py              # 实时监控系统状态
    python diagnose_system.py --once       # 只显示一次状态

容错机制说明 (v2.0+):
    背压级别: 0=无, 1=轻度(延迟10ms), 2=中度(延迟50ms), 3=严重(延迟100ms+溢出)
    溢出队列: 当队列满时，数据暂存到磁盘，空闲时自动回填
    数据保证: 至少一次 (At-Least-Once)
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
    print("系统诊断报告 (v2.0 容错架构)")
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
    print(f"  批处理超时: {processor.get('batch_timeout', 'N/A')} ms")
    
    # 容错机制状态
    resilient = status.get('resilient', {})
    resilient_enabled = status.get('resilient_enabled', False)
    
    print(f"\n🛡️  容错机制状态:")
    if resilient_enabled:
        backpressure_level = resilient.get('backpressure_level', 0)
        overflow_count = resilient.get('overflow_count', 0)
        drain_count = resilient.get('drain_count', 0)
        overflow_files = resilient.get('overflow_file_count', 0)
        
        level_names = ['无', '轻度', '中度', '严重']
        level_emojis = ['✅', '⚡', '⚠️', '🔴']
        
        print(f"  状态: ✅ 已启用")
        print(f"  背压级别: {level_emojis[backpressure_level]} {backpressure_level} ({level_names[backpressure_level]})")
        print(f"  建议延迟: {resilient.get('backpressure_delay_ms', 0)} ms")
        
        if overflow_count > 0:
            print(f"\n  💾 磁盘溢出队列:")
            print(f"    溢出总数: {overflow_count} 条")
            print(f"    已回填数: {drain_count} 条")
            print(f"    等待回填: {overflow_count - drain_count} 条")
            print(f"    溢出文件: {overflow_files} 个")
            print(f"    总大小: {resilient.get('overflow_total_size', 0) / 1024 / 1024:.2f} MB")
    else:
        print(f"  状态: ❌ 未启用 (使用传统模式)")
    
    # 队列状态
    print(f"\n📥 队列状态:")
    print(f"  Input Queue: {processor.get('input_queue_size', 'N/A')} / {processor.get('input_queue_capacity', 'N/A')}")
    print(f"  Output Queue: {processor.get('output_queue_size', 'N/A')} / {processor.get('output_queue_capacity', 'N/A')}")
    
    if processor.get('dropped_count', 0) > 0:
        print(f"  ⚠️  丢弃总数: {processor.get('dropped_count')} 条")
    
    # 性能评估
    print(f"\n📈 性能评估:")
    if processor.get('batch_size', 100) < 1000:
        print(f"  ⚠️  batch_size 较小 ({processor.get('batch_size')})，建议调整到 1000-2000")
    else:
        print(f"  ✅ batch_size 设置合理 ({processor.get('batch_size')})")
    
    if resilient_enabled and overflow_count > 1000:
        print(f"  ⚠️  溢出数据较多，建议降低发送速率或增大队列容量")
    
    print(f"\n🔧 建议:")
    if resilient_enabled:
        print(f"  1. 容错机制已启用，系统会自动处理队列满的情况")
        print(f"  2. 如果出现大量溢出，检查 ./temp/overflow/ 目录")
        print(f"  3. 溢出数据会在队列空闲时自动回填")
    else:
        print(f"  1. 建议使用容错模式: go run cmd/server/main.go")
        print(f"  2. 如果出现丢包，降低压测速率: python stress_test.py -rate 20 -c 10")


def monitor():
    """持续监控"""
    print("开始监控 (按 Ctrl+C 停止)...")
    print("=" * 80)
    print(f"{'时间':<10} {'日志总数':>10} {'QPS':>8} {'背压':>6} {'溢出':>8} {'回填':>8}")
    print("=" * 80)
    
    prev_count = 0
    prev_time = time.time()
    
    try:
        while True:
            count = fetch_stats()
            now = time.time()
            
            qps = (count - prev_count) / (now - prev_time) if now > prev_time else 0
            
            status = fetch_status()
            if status:
                resilient = status.get('resilient', {})
                backpressure_level = resilient.get('backpressure_level', 0)
                overflow_count = resilient.get('overflow_count', 0)
                drain_count = resilient.get('drain_count', 0)
                
                level_names = ['-', 'L', 'M', 'H']
                bp_indicator = level_names[backpressure_level] if backpressure_level < len(level_names) else '?'
                
                time_str = time.strftime('%H:%M:%S')
                print(f"\r{time_str:<10} {count:>10,} {qps:>8.0f} {bp_indicator:>6} {overflow_count:>8} {drain_count:>8}", end='')
            
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
