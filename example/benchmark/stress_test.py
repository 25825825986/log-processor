#!/usr/bin/env python3
"""
日志处理器并发性能测试工具

使用示例:
    # TCP 测试 - 10万条，100并发
    python benchmark.py -protocol tcp -addr localhost:9000 -total 100000 -c 100
    
    # HTTP 测试 - 持续30秒，50并发
    python benchmark.py -protocol http -addr localhost:9002 -d 30 -c 50
    
    # UDP 测试
    python benchmark.py -protocol udp -addr localhost:9001 -total 50000 -c 50
"""

import argparse
import json
import socket
import time
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
import urllib.request
import urllib.error
import sys


class Stats:
    def __init__(self):
        self.sent = 0
        self.failed = 0
        self.lock = threading.Lock()
        self.start_time = time.time()

    def add_sent(self, count=1):
        with self.lock:
            self.sent += count

    def add_failed(self, count=1):
        with self.lock:
            self.failed += count

    def get_qps(self):
        elapsed = time.time() - self.start_time
        return self.sent / elapsed if elapsed > 0 else 0


def generate_log_line(seq):
    """生成 Nginx 格式日志"""
    timestamp = datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")
    path = f"/api/test{seq % 100}"
    size = 100 + (seq % 9900)
    return f'127.0.0.1 - - [{timestamp}] "GET {path} HTTP/1.1" 200 {size} "-" "Benchmark/{seq}"'


def tcp_sender(args, stats, worker_id):
    """TCP 发送器"""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect((args.addr.split(':')[0], int(args.addr.split(':')[1])))
        sock.settimeout(5)
        
        seq = worker_id
        while True:
            if args.duration > 0:
                if time.time() - stats.start_time > args.duration:
                    break
            else:
                if stats.sent >= args.total:
                    break
            
            # 限流控制
            if args.rate > 0:
                time.sleep(1.0 / args.rate)
            
            log_line = generate_log_line(seq)
            try:
                sock.sendall((log_line + '\n').encode())
                stats.add_sent()
            except Exception:
                stats.add_failed()
                try:
                    sock.close()
                    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                    sock.connect((args.addr.split(':')[0], int(args.addr.split(':')[1])))
                except Exception:
                    break
            
            seq += args.c
            
        sock.close()
    except Exception as e:
        print(f"[Worker {worker_id}] 错误: {e}")


def udp_sender(args, stats, worker_id):
    """UDP 发送器"""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        
        seq = worker_id
        while True:
            if args.duration > 0:
                if time.time() - stats.start_time > args.duration:
                    break
            else:
                if stats.sent >= args.total:
                    break
            
            # 限流控制
            if args.rate > 0:
                time.sleep(1.0 / args.rate)
            
            log_line = generate_log_line(seq)
            try:
                sock.sendto(log_line.encode(), 
                           (args.addr.split(':')[0], int(args.addr.split(':')[1])))
                stats.add_sent()
            except Exception:
                stats.add_failed()
            
            seq += args.c
            
        sock.close()
    except Exception as e:
        print(f"[Worker {worker_id}] 错误: {e}")


def http_sender(args, stats, worker_id):
    """HTTP 发送器"""
    url = f"http://{args.addr}/logs"
    seq = worker_id
    
    while True:
        if args.duration > 0:
            if time.time() - stats.start_time > args.duration:
                break
        else:
            if stats.sent >= args.total:
                break
        
        # 批量发送
        lines = []
        for _ in range(args.batch):
            lines.append(generate_log_line(seq))
            seq += args.c
        
        body = '\n'.join(lines).encode()
        
        try:
            req = urllib.request.Request(
                url, 
                data=body, 
                headers={'Content-Type': 'text/plain'},
                method='POST'
            )
            with urllib.request.urlopen(req, timeout=5) as resp:
                if resp.status == 200:
                    stats.add_sent(args.batch)
                else:
                    stats.add_failed(args.batch)
        except Exception:
            stats.add_failed(args.batch)


def progress_reporter(stats, stop_event):
    """进度报告"""
    last_sent = 0
    while not stop_event.is_set():
        time.sleep(1)
        sent = stats.sent
        qps = sent - last_sent
        avg_qps = stats.get_qps()
        elapsed = time.time() - stats.start_time
        
        sys.stdout.write(
            f"\r[QPS: {qps:6d}/s] [Total: {sent:8d}] [Avg: {avg_qps:6.0f}/s] "
            f"[Failed: {stats.failed}] [{elapsed:.1f}s]     "
        )
        sys.stdout.flush()
        last_sent = sent
    print()


def clear_server_logs():
    """清空服务端日志数据"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/logs', method='DELETE')
        urllib.request.urlopen(req, timeout=10)
        time.sleep(0.5)  # 等待清空完成
        return True
    except Exception as e:
        print(f"[WARN] 清空数据失败: {e}")
        return False

def get_server_count():
    """获取服务端日志数量"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/logs?limit=1', method='GET')
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            return data.get('total', 0)
    except Exception:
        return 0

def main():
    parser = argparse.ArgumentParser(description='日志处理器并发性能测试')
    parser.add_argument('-protocol', choices=['tcp', 'udp', 'http'], 
                       default='tcp', help='协议类型')
    parser.add_argument('-addr', default='localhost:9000', 
                       help='目标地址 (host:port)')
    parser.add_argument('-total', type=int, default=10000, 
                       help='总发送日志数')
    parser.add_argument('-c', type=int, default=100, 
                       help='并发连接数/协程数')
    parser.add_argument('-d', dest='duration', type=int, default=0, 
                       help='测试持续时间(秒)，0表示按total发送')
    parser.add_argument('-batch', type=int, default=1, 
                       help='每批发送条数(仅HTTP有效)')
    parser.add_argument('-rate', type=int, default=0, 
                       help='每连接限流速率(条/秒)，0为不限流')
    parser.add_argument('-no-clear', action='store_true',
                       help='测试前不清空服务端数据')
    args = parser.parse_args()

    print("=" * 50)
    print("日志处理器并发性能测试")
    print("=" * 50)
    print(f"协议: {args.protocol.upper()}")
    print(f"目标: {args.addr}")
    print(f"并发: {args.c}")
    if args.duration > 0:
        print(f"持续时间: {args.d} 秒")
    else:
        print(f"总量: {args.total}")
    print()

    # 获取初始数量
    initial_count = get_server_count()
    print(f"[INFO] 测试前服务端已有: {initial_count:,} 条日志")
    
    # 清空数据（除非指定 -no-clear）
    if not args.no_clear:
        print("[INFO] 正在清空服务端数据...")
        if clear_server_logs():
            initial_count = 0
            print("[INFO] 数据已清空")
        else:
            print("[WARN] 清空失败，将使用增量计算")
    else:
        print("[INFO] 保留已有数据，使用增量计算")
    
    print()
    stats = Stats()
    stop_event = threading.Event()

    # 启动进度报告
    reporter = threading.Thread(target=progress_reporter, args=(stats, stop_event))
    reporter.daemon = True
    reporter.start()

    # 选择发送器
    sender_func = {
        'tcp': tcp_sender,
        'udp': udp_sender,
        'http': http_sender
    }[args.protocol]

    # 启动工作线程
    threads = []
    for i in range(args.c):
        t = threading.Thread(target=sender_func, args=(args, stats, i))
        t.start()
        threads.append(t)

    # 等待完成
    for t in threads:
        t.join()

    stop_event.set()
    reporter.join()

    # 最终报告
    duration = time.time() - stats.start_time
    print()
    print("=" * 50)
    print("测试结果")
    print("=" * 50)
    print(f"总用时: {duration:.2f} 秒")
    print(f"成功发送: {stats.sent:,} 条")
    print(f"失败: {stats.failed:,} 条")
    print(f"平均 QPS: {stats.sent/duration:,.0f} 条/秒")
    print(f"吞吐量: {stats.sent*100/duration/1024/1024:.2f} MB/秒")
    
    # 验证服务端实际存储数量
    print()
    print("=" * 50)
    print("服务端验证")
    print("=" * 50)
    try:
        # 等待数据处理完成
        print("等待 2 秒让服务端处理完队列...")
        time.sleep(2)
        
        # 查询服务端存储数量
        final_count = get_server_count()
        received = stats.sent
        
        # 计算增量（排除测试前已有的数据）
        stored = final_count - initial_count
        
        print(f"客户端发送: {received:,} 条")
        print(f"服务端原有: {initial_count:,} 条")
        print(f"服务端现有: {final_count:,} 条")
        print(f"本次新增: {stored:,} 条")
        
        if received > 0:
            ratio = stored / received * 100
            print(f"处理成功率: {ratio:.1f}%")
            
            if ratio < 90:
                print("⚠️  警告: 大量日志可能因队列满被丢弃")
            elif ratio < 100:
                print("ℹ️  部分日志仍在处理队列中，属正常现象")
            else:
                print("✅ 所有日志已存储")
    except Exception as e:
        print(f"无法验证服务端状态: {e}")
        print("提示: 请手动访问 http://localhost:8080/api/logs?limit=1 查看")


if __name__ == '__main__':
    main()
