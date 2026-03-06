#!/usr/bin/env python3
"""
日志数据生成器
生成指定数量和格式的模拟日志数据
"""

import argparse
import random
from datetime import datetime, timedelta
from pathlib import Path


# 模拟数据池
IPS = [
    "127.0.0.1",
    "192.168.1.100", "192.168.1.101", "192.168.1.102",
    "10.0.0.1", "10.0.0.2", "10.0.0.3",
    "172.16.0.1", "172.16.0.2"
]

METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"]
METHODS_WEIGHTS = [70, 20, 5, 3, 2]  # GET 请求更多

PATHS = [
    "/api/users", "/api/users/login", "/api/users/1", "/api/users/2",
    "/api/products", "/api/products/1", "/api/products/2", "/api/products/search",
    "/api/orders", "/api/orders/1", "/api/orders/create",
    "/api/cart", "/api/cart/add", "/api/cart/remove",
    "/static/js/app.js", "/static/css/style.css", "/static/img/logo.png",
    "/", "/home", "/about", "/contact",
    "/health", "/metrics"
]

STATUS_CODES = [200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503]
STATUS_WEIGHTS = [60, 10, 5, 3, 3, 5, 2, 2, 8, 1, 0.5, 0.5]

USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15",
    "Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X) AppleWebKit/605.1.15",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
    "curl/7.68.0",
    "PostmanRuntime/7.28.4"
]


def generate_timestamp(start_time=None, days=30):
    """生成随机时间戳"""
    if start_time is None:
        start_time = datetime.now() - timedelta(days=days)
    
    # 在指定天数范围内随机
    random_offset = random.randint(0, days * 24 * 3600)
    ts = start_time + timedelta(seconds=random_offset)
    return ts


def generate_nginx_log(timestamp=None):
    """生成单条 Nginx 格式日志"""
    if timestamp is None:
        ts = generate_timestamp()
        timestamp = ts.strftime("%d/%b/%Y:%H:%M:%S +0800")
    
    ip = random.choice(IPS)
    method = random.choices(METHODS, weights=METHODS_WEIGHTS)[0]
    path = random.choice(PATHS)
    status = random.choices(STATUS_CODES, weights=STATUS_WEIGHTS)[0]
    size = random.randint(100, 10000)
    user_agent = random.choice(USER_AGENTS)
    
    # Nginx Combined Log Format
    return f'{ip} - - [{timestamp}] "{method} {path} HTTP/1.1" {status} {size} "-" "{user_agent}"'


def generate_json_log(timestamp=None):
    """生成单条 JSON 格式日志"""
    import json
    
    if timestamp is None:
        ts = generate_timestamp()
        timestamp = ts.isoformat()
    
    log_entry = {
        "timestamp": timestamp,
        "level": "info" if random.random() > 0.1 else "error",
        "client_ip": random.choice(IPS),
        "method": random.choices(METHODS, weights=METHODS_WEIGHTS)[0],
        "path": random.choice(PATHS),
        "status_code": random.choices(STATUS_CODES, weights=STATUS_WEIGHTS)[0],
        "response_time": random.randint(10, 2000),
        "user_agent": random.choice(USER_AGENTS)
    }
    
    return json.dumps(log_entry, ensure_ascii=False)


def generate_logs(count=1000, format="nginx", output_file=None, days=30):
    """生成日志文件"""
    logs = []
    
    # 生成时间序列（确保按时间顺序）
    base_time = datetime.now() - timedelta(days=days)
    timestamps = [
        base_time + timedelta(seconds=random.randint(0, days * 24 * 3600))
        for _ in range(count)
    ]
    timestamps.sort()
    
    for ts in timestamps:
        if format == "nginx":
            ts_str = ts.strftime("%d/%b/%Y:%H:%M:%S +0800")
            logs.append(generate_nginx_log(ts_str))
        elif format == "json":
            ts_str = ts.isoformat()
            logs.append(generate_json_log(ts_str))
    
    output = '\n'.join(logs)
    
    if output_file:
        Path(output_file).write_text(output, encoding='utf-8')
        print(f"已生成 {count} 条 {format} 格式日志到 {output_file}")
        print(f"时间范围: {timestamps[0]} ~ {timestamps[-1]}")
    else:
        print(output)
    
    return output


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='生成模拟日志数据')
    parser.add_argument('-n', '--count', type=int, default=1000, 
                        help='生成日志条数 (默认: 1000)')
    parser.add_argument('-f', '--format', choices=['nginx', 'json'], 
                        default='nginx', help='日志格式 (默认: nginx)')
    parser.add_argument('-o', '--output', type=str, 
                        help='输出文件路径 (默认: 输出到控制台)')
    parser.add_argument('-d', '--days', type=int, default=30,
                        help='时间范围跨度(天) (默认: 30)')
    
    args = parser.parse_args()
    
    generate_logs(args.count, args.format, args.output, args.days)
