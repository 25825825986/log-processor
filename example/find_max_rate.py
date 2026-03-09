#!/usr/bin/env python3
"""
测试系统最大处理能力 - 渐进式增加压力

使用示例:
    python find_max_rate.py -protocol tcp -addr localhost:9000
"""

import argparse
import json
import socket
import time
import threading
import urllib.request
import sys


def test_with_rate(protocol, addr, total, concurrency, rate_per_conn):
    """以指定速率测试"""
    stats = {'sent': 0, 'failed': 0, 'start_time': time.time()}
    lock = threading.Lock()
    stop_event = threading.Event()
    
    def sender(worker_id):
        try:
            if protocol == 'tcp':
                conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                conn.connect((addr.split(':')[0], int(addr.split(':')[1])))
            elif protocol == 'udp':
                conn = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            else:
                conn = None
                
            seq = worker_id
            last_send_time = time.time()
            
            while not stop_event.is_set():
                if stats['sent'] >= total:
                    break
                
                # 速率限制
                if rate_per_conn > 0:
                    expected_time = last_send_time + (1.0 / rate_per_conn)
                    sleep_time = expected_time - time.time()
                    if sleep_time > 0:
                        time.sleep(sleep_time)
                    last_send_time = time.time()
                
                log_line = f'127.0.0.1 - - [{time.strftime("%d/%b/%Y:%H:%M:%S %z")}] "GET /api/test{seq%100} HTTP/1.1" 200 {100+(seq%9900)}'
                
                try:
                    if protocol == 'tcp':
                        conn.sendall((log_line + '\n').encode())
                    elif protocol == 'udp':
                        conn.sendto(log_line.encode(), (addr.split(':')[0], int(addr.split(':')[1])))
                    
                    with lock:
                        stats['sent'] += 1
                except Exception:
                    with lock:
                        stats['failed'] += 1
                
                seq += concurrency
                
            if conn:
                conn.close()
        except Exception as e:
            print(f"[Worker {worker_id}] 错误: {e}")
    
    # 启动工作线程
    threads = []
    for i in range(concurrency):
        t = threading.Thread(target=sender, args=(i,))
        t.start()
        threads.append(t)
    
    # 进度显示
    last_sent = 0
    while stats['sent'] < total and not stop_event.is_set():
        time.sleep(1)
        with lock:
            sent = stats['sent']
        qps = sent - last_sent
        last_sent = sent
        print(f"\r  发送进度: {sent:,} / {total:,} ({qps:,}/s)", end='', flush=True)
    
    stop_event.set()
    for t in threads:
        t.join()
    
    print()
    return stats['sent'], stats['failed']


def get_server_count():
    """获取服务端存储数量"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/logs?limit=1', method='GET')
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            return data.get('total', 0)
    except Exception:
        return 0


def main():
    parser = argparse.ArgumentParser(description='测试系统最大处理能力')
    parser.add_argument('-protocol', choices=['tcp', 'udp'], default='tcp')
    parser.add_argument('-addr', default='localhost:9000')
    parser.add_argument('-c', type=int, default=50, help='并发连接数')
    parser.add_argument('-test_each', type=int, default=10000, help='每个速率测试的条数')
    parser.add_argument('-max_rate', type=int, default=10000, help='最大测试速率')
    args = parser.parse_args()
    
    print("=" * 60)
    print("系统最大处理能力测试")
    print("=" * 60)
    print(f"协议: {args.protocol.upper()}")
    print(f"地址: {args.addr}")
    print(f"并发: {args.c}")
    print()
    
    # 初始清空
    print("正在清空已有数据...")
    try:
        req = urllib.request.Request('http://localhost:8080/api/logs', method='DELETE')
        urllib.request.urlopen(req, timeout=10)
        time.sleep(1)
    except Exception:
        pass
    
    rates = [1000, 2000, 3000, 4000, 5000, 6000, 8000]
    best_rate = 0
    best_success_rate = 0
    
    for rate in rates:
        if rate > args.max_rate:
            break
            
        print(f"\n📊 测试速率: {rate:,} QPS")
        print("-" * 40)
        
        # 清空数据
        try:
            req = urllib.request.Request('http://localhost:8080/api/logs', method='DELETE')
            urllib.request.urlopen(req, timeout=5)
            time.sleep(0.5)
        except Exception:
            pass
        
        # 计算每个连接的速率
        rate_per_conn = rate // args.c
        if rate_per_conn < 1:
            rate_per_conn = 1
        
        # 测试
        sent, failed = test_with_rate(args.protocol, args.addr, args.test_each, args.c, rate_per_conn)
        
        # 等待处理完成
        print("  等待 3 秒让服务端处理...")
        time.sleep(3)
        
        # 检查存储数量
        stored = get_server_count()
        success_rate = (stored / sent * 100) if sent > 0 else 0
        
        print(f"  发送: {sent:,} 条")
        print(f"  存储: {stored:,} 条")
        print(f"  成功率: {success_rate:.1f}%")
        
        if success_rate >= 95:
            best_rate = rate
            best_success_rate = success_rate
            print("  ✅ 通过")
        else:
            print("  ❌ 未通过 (大量丢弃)")
            break
    
    print()
    print("=" * 60)
    print("测试结果")
    print("=" * 60)
    if best_rate > 0:
        print(f"🎉 系统最大稳定处理能力: ~{best_rate:,} QPS")
        print(f"   成功率: {best_success_rate:.1f}%")
        print()
        print("💡 建议配置:")
        print(f"   - 如果持续接收 {best_rate//2:,} QPS 以下，可以稳定处理")
        print(f"   - 如果超过 {best_rate:,} QPS，考虑降低发送速率或使用多个实例")
    else:
        print("⚠️  即使在最低速率下也有大量丢弃")
        print("   建议: 大幅降低并发或增加 worker_count")


if __name__ == '__main__':
    main()
