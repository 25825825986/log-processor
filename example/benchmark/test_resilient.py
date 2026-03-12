#!/usr/bin/env python3
"""
容错机制测试脚本 (v2.0)

测试系统的背压机制和溢出队列功能

使用示例:
    python test_resilient.py                    # 运行完整测试
    python test_resilient.py --quick            # 快速测试（1万条）
    python test_resilient.py --rate 200         # 自定义发送速率

测试流程:
    1. 清空数据，开始监控
    2. 高速发送（超过处理能力）
    3. 观察背压级别变化和溢出情况
    4. 等待回填完成
    5. 验证最终数据完整性
"""

import argparse
import json
import sys
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor
import socket


def fetch_status():
    """获取系统状态"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/status', method='GET')
        with urllib.request.urlopen(req, timeout=5) as resp:
            return json.loads(resp.read().decode())
    except Exception as e:
        print(f"[ERROR] 无法获取状态: {e}")
        return None


def fetch_log_count():
    """获取日志数量"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/logs?limit=1', method='GET')
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            return data.get('total', 0)
    except Exception:
        return 0


def clear_logs():
    """清空日志"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/logs', method='DELETE')
        urllib.request.urlopen(req, timeout=10)
        time.sleep(1)
        return True
    except Exception as e:
        print(f"[WARN] 清空日志失败: {e}")
        return False


def send_logs_batch(count, rate):
    """批量发送日志"""
    host, port = 'localhost', 9000
    
    def sender(start, end):
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.connect((host, port))
            sock.settimeout(10)
            
            for i in range(start, end):
                if rate > 0:
                    time.sleep(1.0 / rate)
                
                log_line = f'127.0.0.1 - - [{time.strftime("%d/%b/%Y:%H:%M:%S %z")}] "GET /api/test{i} HTTP/1.1" 200 {100+i%9900}'
                sock.sendall((log_line + '\n').encode())
            
            sock.close()
            return end - start
        except Exception as e:
            print(f"[ERROR] 发送失败: {e}")
            return 0
    
    # 使用10个并发连接
    batch_size = count // 10
    with ThreadPoolExecutor(max_workers=10) as executor:
        futures = []
        for i in range(10):
            start = i * batch_size
            end = start + batch_size if i < 9 else count
            futures.append(executor.submit(sender, start, end))
        
        total_sent = sum(f.result() for f in futures)
    
    return total_sent


def monitor_resilient(duration=30):
    """监控容错机制"""
    print(f"\n📊 开始监控（{duration}秒）...")
    print("-" * 80)
    print(f"{'时间':<10} {'日志数':>10} {'背压':>6} {'溢出':>8} {'回填':>8} {'QPS':>8}")
    print("-" * 80)
    
    prev_count = 0
    start_time = time.time()
    
    try:
        while time.time() - start_time < duration:
            count = fetch_log_count()
            elapsed = time.time() - start_time
            
            status = fetch_status()
            if status:
                resilient = status.get('resilient', {})
                bp_level = resilient.get('backpressure_level', 0)
                overflow = resilient.get('overflow_count', 0)
                drain = resilient.get('drain_count', 0)
                
                level_names = ['-', 'L', 'M', 'H']
                bp_str = level_names[bp_level] if bp_level < len(level_names) else '?'
                
                qps = (count - prev_count)
                time_str = time.strftime('%H:%M:%S')
                
                print(f"\r{time_str:<10} {count:>10,} {bp_str:>6} {overflow:>8} {drain:>8} {qps:>8}", end='')
                
                prev_count = count
            
            time.sleep(1)
    except KeyboardInterrupt:
        pass
    
    print("\n" + "-" * 80)
    return prev_count


def run_test(args):
    """运行测试"""
    print("=" * 80)
    print("🧪 容错机制测试 (v2.0)")
    print("=" * 80)
    
    # 检查服务器状态
    print("\n1️⃣ 检查服务器状态...")
    status = fetch_status()
    if not status:
        print("❌ 无法连接到服务器，请确保服务已启动")
        return False
    
    resilient_enabled = status.get('resilient_enabled', False)
    if resilient_enabled:
        print("✅ 容错机制已启用")
    else:
        print("⚠️  容错机制未启用（使用传统模式）")
    
    # 清空数据
    print("\n2️⃣ 清空历史数据...")
    clear_logs()
    
    # 计算测试参数
    total = args.count
    target_qps = args.rate
    
    print(f"\n3️⃣ 测试参数:")
    print(f"   发送总数: {total:,} 条")
    print(f"   目标速率: {target_qps} QPS")
    print(f"   预计用时: {total/target_qps:.1f} 秒")
    
    # 开始监控
    print(f"\n4️⃣ 高速发送数据（超过处理能力）...")
    print(f"   同时启动监控协程...")
    
    # 在后台启动监控
    import threading
    monitor_stop = threading.Event()
    monitor_result = {'count': 0}
    
    def background_monitor():
        prev = 0
        while not monitor_stop.is_set():
            count = fetch_log_count()
            monitor_result['count'] = count
            time.sleep(1)
    
    monitor_thread = threading.Thread(target=background_monitor)
    monitor_thread.start()
    
    # 发送数据
    start_time = time.time()
    sent = send_logs_batch(total, target_qps)
    send_time = time.time() - start_time
    
    monitor_stop.set()
    monitor_thread.join()
    
    print(f"\n   实际发送: {sent:,} 条")
    print(f"   实际用时: {send_time:.1f} 秒")
    print(f"   实际速率: {sent/send_time:.0f} QPS")
    
    # 等待处理完成
    print(f"\n5️⃣ 等待系统处理队列...")
    print(f"   监控 {args.wait} 秒...")
    final_count = monitor_resilient(args.wait)
    
    # 获取最终状态
    print(f"\n6️⃣ 最终结果:")
    status = fetch_status()
    if status:
        resilient = status.get('resilient', {})
        
        print(f"\n   📊 数据统计:")
        print(f"   目标发送: {total:,} 条")
        print(f"   实际发送: {sent:,} 条")
        print(f"   最终存储: {final_count:,} 条")
        
        if sent > 0:
            success_rate = final_count / sent * 100
            print(f"   存储成功率: {success_rate:.1f}%")
        
        print(f"\n   🛡️ 容错统计:")
        print(f"   背压级别: {resilient.get('backpressure_level', 'N/A')}")
        print(f"   溢出总数: {resilient.get('overflow_count', 0)} 条")
        print(f"   已回填数: {resilient.get('drain_count', 0)} 条")
        
        overflow_files = resilient.get('overflow_file_count', 0)
        if overflow_files > 0:
            print(f"   溢出文件: {overflow_files} 个")
            print(f"   总大小: {resilient.get('overflow_total_size', 0) / 1024 / 1024:.2f} MB")
    
    print("\n" + "=" * 80)
    print("✅ 测试完成")
    print("=" * 80)
    
    return True


def main():
    parser = argparse.ArgumentParser(description='容错机制测试')
    parser.add_argument('--count', type=int, default=50000, help='发送日志总数 (默认: 50000)')
    parser.add_argument('--rate', type=int, default=100, help='发送速率 QPS (默认: 100)')
    parser.add_argument('--wait', type=int, default=30, help='等待回填时间秒 (默认: 30)')
    parser.add_argument('--quick', action='store_true', help='快速测试模式 (10000条)')
    args = parser.parse_args()
    
    if args.quick:
        args.count = 10000
        args.rate = 100
        args.wait = 15
    
    try:
        run_test(args)
    except KeyboardInterrupt:
        print("\n\n⚠️ 测试被中断")
        sys.exit(1)


if __name__ == "__main__":
    main()
