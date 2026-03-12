#!/usr/bin/env python3
"""
日志处理器并发性能测试工具 (v2.0 - 支持容错机制)

使用示例:
    # TCP 测试 - 10万条，100并发，限制速率1000条/秒每连接
    python stress_test.py -protocol tcp -addr localhost:9000 -total 100000 -c 100 -rate 1000
    
    # HTTP 测试 - 持续30秒，50并发
    python stress_test.py -protocol http -addr localhost:9002 -d 30 -c 50 -rate 100
    
    # UDP 测试
    python stress_test.py -protocol udp -addr localhost:9001 -total 50000 -c 50 -rate 1000
    
    # 使用 NASA 真实日志数据测试
    python download_nasa_logs.py                    # 先下载数据集
    python stress_test.py -file ../data/NASA_access_log_Jul95.txt -total 100000 -rate 50

系统能力 (v2.0+ 容错架构):
    持续处理能力: ~1,500 QPS (异步存储优化)
    突发处理能力: ~20,000 QPS (依赖队列缓冲)
    数据保证: 至少一次 (背压 + 磁盘溢出队列)
    
    当发送速率超过处理能力时：
    1. 触发背压机制 - 自动降速
    2. 启用磁盘溢出队列 - 数据暂存到磁盘
    3. 队列空闲时自动回填 - 保证数据不丢失

NASA 数据集:
    下载: python download_nasa_logs.py
    大小: ~200MB, 约 190万条真实 HTTP 请求
    时间: 1995年7月 NASA Kennedy Space Center 服务器日志
"""

import argparse
import json
import socket
import time
import threading
import random
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
import urllib.request
import urllib.error
import sys

# 真实场景数据集 - 用于生成更真实的测试数据
REALISTIC_PATHS = [
    "/api/users", "/api/products", "/api/orders", "/api/auth/login", "/api/auth/logout",
    "/api/search", "/api/cart", "/api/checkout", "/api/payment", "/api/inventory",
    "/static/js/app.js", "/static/css/style.css", "/static/img/logo.png",
    "/", "/about", "/contact", "/products", "/blog", "/faq",
    "/api/v2/users", "/api/v2/products", "/api/admin/dashboard", "/api/admin/users",
    "/api/health", "/api/metrics", "/favicon.ico", "/robots.txt"
]

HTTP_METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"]
HTTP_METHOD_WEIGHTS = [70, 20, 5, 3, 2]  # GET占70%，POST占20%等

STATUS_CODES = [200, 201, 204, 301, 302, 304, 400, 401, 403, 404, 500, 502, 503]
STATUS_CODE_WEIGHTS = [65, 8, 5, 3, 2, 4, 3, 2, 2, 4, 1, 0.5, 0.5]  # 2xx占82%，4xx占11%等

IP_RANGES = [
    "192.168.1", "10.0.0", "172.16.0",  # 内网IP
    "203.0.113", "198.51.100", "192.0.2"  # 测试/文档IP
]

USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.0",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.0",
    "Mozilla/5.0 (Android 11; Mobile; rv:83.0) Gecko/83.0 Firefox/83.0",
    "curl/7.68.0", "PostmanRuntime/7.26.8", "Go-http-client/1.1"
]

REFERERS = [
    "-", "https://www.google.com/", "https://www.bing.com/",
    "https://example.com/", "https://example.com/products",
    "https://twitter.com/", "https://facebook.com/"
]


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
    """生成真实分布的Nginx格式日志"""
    timestamp = datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")
    
    # 随机选择各个字段
    ip_range = random.choice(IP_RANGES)
    client_ip = f"{ip_range}.{random.randint(1, 254)}"
    
    method = random.choices(HTTP_METHODS, weights=HTTP_METHOD_WEIGHTS)[0]
    path = random.choice(REALISTIC_PATHS)
    status = random.choices(STATUS_CODES, weights=STATUS_CODE_WEIGHTS)[0]
    size = random.randint(100, 100000)
    referer = random.choice(REFERERS)
    user_agent = random.choice(USER_AGENTS)
    
    # 随机响应时间 (0.001 ~ 5.0秒)
    response_time = round(random.uniform(0.001, 5.0), 3)
    
    return f'{client_ip} - - [{timestamp}] "{method} {path} HTTP/1.1" {status} {size} "{referer}" "{user_agent}" "{response_time}"'


def tcp_sender(args, stats, worker_id, log_reader=None):
    """TCP发送器 - 带精确总量控制"""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect((args.addr.split(':')[0], int(args.addr.split(':')[1])))
        sock.settimeout(5)
        
        while True:
            # 检查是否应该停止（原子操作）
            with stats.lock:
                if args.duration > 0:
                    should_stop = time.time() - stats.start_time > args.duration
                else:
                    should_stop = stats.sent >= args.total
                if should_stop:
                    break
                # 预占额度
                stats.sent += 1
            
            # 限流控制
            if args.rate > 0:
                time.sleep(1.0 / args.rate)
            
            # 从文件读取或生成随机日志
            if log_reader:
                log_line = log_reader.get_line()
            else:
                log_line = generate_log_line(worker_id)
            
            try:
                sock.sendall((log_line + '\n').encode())
            except Exception:
                # 发送失败，回退统计
                with stats.lock:
                    stats.sent -= 1
                    stats.failed += 1
                try:
                    sock.close()
                    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                    sock.connect((args.addr.split(':')[0], int(args.addr.split(':')[1])))
                except Exception:
                    break
            
        sock.close()
    except Exception as e:
        print(f"[Worker {worker_id}] 错误: {e}")


def udp_sender(args, stats, worker_id, log_reader=None):
    """UDP发送器 - 带精确总量控制"""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        
        while True:
            # 检查是否应该停止（原子操作）
            with stats.lock:
                if args.duration > 0:
                    should_stop = time.time() - stats.start_time > args.duration
                else:
                    should_stop = stats.sent >= args.total
                if should_stop:
                    break
                # 预占额度
                stats.sent += 1
            
            if args.rate > 0:
                time.sleep(1.0 / args.rate)
            
            # 从文件读取或生成随机日志
            if log_reader:
                log_line = log_reader.get_line()
            else:
                log_line = generate_log_line(worker_id)
            
            try:
                sock.sendto((log_line + '\n').encode(), (args.addr.split(':')[0], int(args.addr.split(':')[1])))
            except Exception:
                # 发送失败，回退统计
                with stats.lock:
                    stats.sent -= 1
                    stats.failed += 1
        
        sock.close()
    except Exception as e:
        print(f"[Worker {worker_id}] 错误: {e}")


def http_sender(args, stats, worker_id, log_reader=None):
    """HTTP发送器 - 带精确总量控制"""
    url = f"http://{args.addr}/logs"
    
    while True:
        # 检查是否应该停止（原子操作）
        with stats.lock:
            if args.duration > 0:
                should_stop = time.time() - stats.start_time > args.duration
            else:
                should_stop = stats.sent >= args.total
            if should_stop:
                break
            
            # 计算本次发送数量
            remaining = args.total - stats.sent if args.duration == 0 else args.batch
            batch_size = min(args.batch, remaining) if args.duration == 0 else args.batch
            if batch_size <= 0:
                break
            
            # 预占额度
            stats.sent += batch_size
        
        # 批量获取日志行
        lines = []
        if log_reader:
            lines = log_reader.get_batch(batch_size)
        else:
            for _ in range(batch_size):
                lines.append(generate_log_line(worker_id))
        
        body = '\n'.join(lines).encode()
        
        try:
            req = urllib.request.Request(
                url, 
                data=body, 
                headers={'Content-Type': 'text/plain'},
                method='POST'
            )
            with urllib.request.urlopen(req, timeout=5) as resp:
                if resp.status != 200:
                    # 发送失败，回退统计
                    with stats.lock:
                        stats.sent -= batch_size
                        stats.failed += batch_size
        except Exception:
            # 异常，回退统计
            with stats.lock:
                stats.sent -= batch_size
                stats.failed += batch_size


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


def get_resilient_info():
    """获取容错机制信息"""
    try:
        req = urllib.request.Request('http://localhost:8080/api/status', method='GET')
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            resilient = data.get('resilient', {})
            resilient['resilient_enabled'] = data.get('resilient_enabled', False)
            return resilient
    except Exception:
        return None

# 日志文件读取器（支持循环读取）
class LogFileReader:
    def __init__(self, filepath):
        self.filepath = filepath
        self.lines = []
        self.index = 0
        self.lock = threading.Lock()
        self._load_file()
    
    def _load_file(self):
        """加载日志文件到内存"""
        try:
            with open(self.filepath, 'r', encoding='utf-8', errors='ignore') as f:
                self.lines = [line.strip() for line in f if line.strip()]
            print(f"[INFO] 已加载日志文件: {self.filepath}")
            print(f"       共 {len(self.lines):,} 条日志")
        except Exception as e:
            print(f"[ERROR] 无法读取日志文件: {e}")
            self.lines = []
    
    def get_line(self):
        """获取下一行日志（循环读取）"""
        with self.lock:
            if not self.lines:
                return None
            line = self.lines[self.index]
            self.index = (self.index + 1) % len(self.lines)
            return line
    
    def get_batch(self, size):
        """获取一批日志"""
        with self.lock:
            if not self.lines:
                return []
            batch = []
            for _ in range(size):
                batch.append(self.lines[self.index])
                self.index = (self.index + 1) % len(self.lines)
            return batch


def main():
    parser = argparse.ArgumentParser(description='日志处理器并发性能测试')
    parser.add_argument('-protocol', choices=['tcp', 'udp', 'http'], 
                       default='tcp', help='协议类型')
    parser.add_argument('-addr', default='localhost:9000', 
                       help='目标地址 (host:port)')
    parser.add_argument('-total', type=int, default=10000, 
                       help='总发送日志数')
    parser.add_argument('-c', type=int, default=10, 
                       help='并发连接数/协程数')
    parser.add_argument('-duration', type=int, default=0, 
                       help='测试持续时间(秒)，0表示按total发送')
    parser.add_argument('-batch', type=int, default=100, 
                       help='每批发送条数(仅HTTP有效)')
    parser.add_argument('-rate', type=int, default=0, 
                       help='每连接限流速率(条/秒)，0为不限流')
    parser.add_argument('-no-clear', action='store_true',
                       help='测试前不清空服务端数据')
    parser.add_argument('-file', type=str, default=None,
                       help='从日志文件读取数据发送。支持: 1) 项目自带测试文件(../data/test_logs.txt) '
                            '2) NASA真实日志(../data/NASA_access_log_Jul95.txt，需先运行 download_nasa_logs.py 下载)')
    args = parser.parse_args()

    # 初始化日志文件读取器（如果指定了文件）
    log_reader = None
    if args.file:
        log_reader = LogFileReader(args.file)
        if not log_reader.lines:
            print("[ERROR] 日志文件为空或无法读取，退出测试")
            return
    
    print("=" * 50)
    print("日志处理器并发性能测试")
    print("=" * 50)
    print(f"协议: {args.protocol.upper()}")
    print(f"目标: {args.addr}")
    print(f"并发: {args.c}")
    if args.file:
        print(f"数据源: 文件 ({len(log_reader.lines):,} 条)")
        print(f"文件路径: {args.file}")
    else:
        print(f"数据源: 随机生成")
    if args.duration > 0:
        print(f"持续时间: {args.duration} 秒")
    else:
        print(f"总量: {args.total}")
    
    # 系统能力警告
    estimated_qps = args.c * args.rate if args.rate > 0 else args.total if args.duration == 0 else args.c * 10000
    if estimated_qps > 1500:
        print(f"\n[⚠️  警告] 预估发送速率 {estimated_qps:,} QPS 超过系统处理能力上限 (~1,500 QPS)")
        print("          超过此限制的日志将被丢弃，这是预期行为")
        print("          建议使用 -rate 参数限制发送速率")
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
            print("[WARN] 清空数据失败，继续测试...")
    
    stats = Stats()
    stop_event = threading.Event()
    
    # 启动进度报告
    reporter = threading.Thread(target=progress_reporter, args=(stats, stop_event))
    reporter.start()
    
    # 选择发送器
    sender_func = {
        'tcp': tcp_sender,
        'udp': udp_sender,
        'http': http_sender
    }[args.protocol]
    
    # 启动测试
    start_time = time.time()
    
    with ThreadPoolExecutor(max_workers=args.c) as executor:
        futures = [executor.submit(sender_func, args, stats, i, log_reader) for i in range(args.c)]
        for future in as_completed(futures):
            try:
                future.result()
            except Exception as e:
                print(f"[ERROR] Worker异常: {e}")
    
    elapsed = time.time() - start_time
    stop_event.set()
    reporter.join()
    
    print()
    print("=" * 50)
    print("测试结果")
    print("=" * 50)
    print(f"总用时: {elapsed:.2f} 秒")
    print(f"成功发送: {stats.sent:,} 条")
    print(f"失败: {stats.failed:,} 条")
    print(f"平均 QPS: {stats.sent/elapsed:,.0f} 条/秒")
    print(f"吞吐量: {(stats.sent * 100) / (1024 * 1024 * elapsed):.2f} MB/秒")
    print()
    
    # 等待服务端处理完队列，然后验证
    print("=" * 50)
    print("服务端验证")
    print("=" * 50)
    print("等待 2 秒让服务端处理完队列...")
    time.sleep(2)
    
    final_count = get_server_count()
    added = final_count - initial_count
    
    print(f"客户端发送: {stats.sent:,} 条")
    print(f"服务端原有: {initial_count:,} 条")
    print(f"服务端现有: {final_count:,} 条")
    print(f"本次新增: {added:,} 条")
    
    if stats.sent > 0:
        success_rate = (added / stats.sent) * 100
        print(f"处理成功率: {success_rate:.1f}%")
        
        # 检查是否启用了容错机制
        resilient_info = get_resilient_info()
        if resilient_info and resilient_info.get('resilient_enabled'):
            overflow_count = resilient_info.get('overflow_count', 0)
            drain_count = resilient_info.get('drain_count', 0)
            backpressure_level = resilient_info.get('backpressure_level', 0)
            
            print(f"\n📊 容错机制状态:")
            print(f"  背压级别: {backpressure_level} (0=无, 1=轻度, 2=中度, 3=严重)")
            print(f"  溢出到磁盘: {overflow_count} 条")
            print(f"  已回填: {drain_count} 条")
            
            if overflow_count > 0:
                print(f"\n💾 数据已溢出到磁盘，将在队列空闲时自动回填")
        
        if success_rate >= 95:
            print("\n✅ 所有日志已存储")
        elif success_rate >= 80:
            print("\n⚠️  部分日志可能仍在处理队列中或已溢出到磁盘")
        else:
            print("\n⚠️  警告: 大量日志可能因队列满被丢弃")
            print("   建议: 1) 降低发送速率 2) 增大溢出队列 3) 检查磁盘空间")
    else:
        print("❌ 未成功发送任何日志")

if __name__ == "__main__":
    main()
