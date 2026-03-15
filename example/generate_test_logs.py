#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
生成各种格式的测试日志文件
覆盖系统支持的所有格式：nginx, apache, json, csv, tsv, syslog, pipe, semicolon, plain
"""

import json
import random
from datetime import datetime, timedelta

# 样本数据池
IPS = ["192.168.1.100", "10.0.0.50", "172.16.0.25", "127.0.0.1", "192.168.0.1"]
METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"]
PATHS = [
    "/api/users", "/api/login", "/api/products", "/home", "/about",
    "/api/orders", "/static/js/app.js", "/css/style.css", "/api/search",
    "/admin/dashboard", "/api/v2/items?page=1", "/favicon.ico"
]
STATUS_CODES = [200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503]
USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.0",
    "Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.0",
    "curl/7.68.0", "PostmanRuntime/7.26.8"
]
REFERERS = ["-", "https://example.com", "https://google.com", "https://bing.com"]
SERVICES = ["nginx", "app", "api-gateway", "auth-service", "database"]
LEVELS = ["INFO", "WARN", "ERROR", "DEBUG"]
MESSAGES = [
    "User login successful",
    "Database connection established",
    "Request processed",
    "Cache miss for key",
    "API rate limit exceeded",
    "Invalid token provided",
    "Resource not found",
    "Query executed in 45ms"
]

def random_time(days_ago=7):
    """生成随机时间"""
    base = datetime.now() - timedelta(days=days_ago)
    offset = timedelta(
        hours=random.randint(0, 23),
        minutes=random.randint(0, 59),
        seconds=random.randint(0, 59)
    )
    return base + offset

def generate_nginx_logs(count=100):
    """生成 Nginx 格式日志 (Combined Log Format)"""
    logs = []
    for _ in range(count):
        time = random_time()
        time_str = time.strftime("%d/%b/%Y:%H:%M:%S +0800")
        ip = random.choice(IPS)
        method = random.choice(METHODS)
        path = random.choice(PATHS)
        status = random.choice(STATUS_CODES)
        size = random.randint(100, 10000)
        referer = random.choice(REFERERS)
        ua = random.choice(USER_AGENTS)
        response_time = round(random.uniform(0.01, 2.5), 3)
        
        log = f'{ip} - - [{time_str}] "{method} {path} HTTP/1.1" {status} {size} "{referer}" "{ua}" "{response_time}"'
        logs.append(log)
    return logs

def generate_apache_logs(count=100):
    """生成 Apache 格式日志 (带响应时间)"""
    logs = []
    for _ in range(count):
        time = random_time()
        time_str = time.strftime("%d/%b/%Y:%H:%M:%S +0800")
        ip = random.choice(IPS)
        method = random.choice(METHODS)
        path = random.choice(PATHS)
        status = random.choice(STATUS_CODES)
        size = random.randint(100, 10000)
        response_time = round(random.uniform(0.01, 2.5), 3)  # 响应时间（秒）
        
        log = f'{ip} - - [{time_str}] "{method} {path} HTTP/1.1" {status} {size} {response_time}'
        logs.append(log)
    return logs

def generate_json_logs(count=100):
    """生成 JSON 格式日志"""
    logs = []
    for _ in range(count):
        log_entry = {
            "timestamp": random_time().isoformat(),
            "level": random.choice(LEVELS),
            "service": random.choice(SERVICES),
            "client_ip": random.choice(IPS),
            "method": random.choice(METHODS),
            "path": random.choice(PATHS),
            "status_code": random.choice(STATUS_CODES),
            "response_time": random.randint(10, 2500),
            "response_size": random.randint(100, 10000),
            "message": random.choice(MESSAGES),
            "user_agent": random.choice(USER_AGENTS),
            "request_id": f"req_{random.randint(10000, 99999)}"
        }
        logs.append(json.dumps(log_entry, ensure_ascii=False))
    return logs

def generate_csv_logs(count=100):
    """生成 CSV 格式日志"""
    logs = []
    for _ in range(count):
        time = random_time().strftime("%Y-%m-%d %H:%M:%S")
        response_time = random.randint(10, 2500)  # 响应时间（毫秒）
        log = f"{random.choice(IPS)},GET,{random.choice(PATHS)},{random.choice(STATUS_CODES)},{random.randint(100,10000)},{response_time},{time}"
        logs.append(log)
    return logs

def generate_tsv_logs(count=100):
    """生成 TSV (Tab分隔) 格式日志"""
    logs = []
    for _ in range(count):
        time = random_time().strftime("%Y-%m-%d %H:%M:%S")
        response_time = random.randint(10, 2500)  # 响应时间（毫秒）
        log = f"{random.choice(IPS)}\t{random.choice(METHODS)}\t{random.choice(PATHS)}\t{random.choice(STATUS_CODES)}\t{random.randint(100,10000)}\t{response_time}\t{time}"
        logs.append(log)
    return logs

def generate_syslog_logs(count=100):
    """生成 Syslog 格式日志 (包含 HTTP 访问信息)"""
    logs = []
    months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
    for _ in range(count):
        time = random_time()
        month = months[time.month - 1]
        day = time.day
        time_str = time.strftime("%H:%M:%S")
        host = f"server-{random.randint(1,5)}"
        service = random.choice(["nginx", "apache", "api-gateway"])
        pid = random.randint(1000, 9999)
        ip = random.choice(IPS)
        method = random.choice(METHODS)
        path = random.choice(PATHS)
        status = random.choice(STATUS_CODES)
        size = random.randint(100, 10000)
        
        # 生成包含 HTTP 访问信息的 syslog 消息
        message = f"{ip} {method} {path} {status} {size}"
        log = f"{month}  {day} {time_str} {host} {service}[{pid}]: {message}"
        logs.append(log)
    return logs

def generate_pipe_logs(count=100):
    """生成 Pipe (管道符) 分隔格式日志"""
    logs = []
    for _ in range(count):
        time = random_time().strftime("%Y-%m-%d %H:%M:%S")
        response_time = random.randint(10, 2500)  # 响应时间（毫秒）
        log = f"{random.choice(IPS)}|{random.choice(METHODS)}|{random.choice(PATHS)}|{random.choice(STATUS_CODES)}|{random.randint(100,10000)}|{response_time}|{time}"
        logs.append(log)
    return logs

def generate_semicolon_logs(count=100):
    """生成 Semicolon (分号) 分隔格式日志"""
    logs = []
    for _ in range(count):
        time = random_time().strftime("%Y-%m-%d %H:%M:%S")
        response_time = random.randint(10, 2500)  # 响应时间（毫秒）
        log = f"{random.choice(IPS)};{random.choice(METHODS)};{random.choice(PATHS)};{random.choice(STATUS_CODES)};{random.randint(100,10000)};{response_time};{time}"
        logs.append(log)
    return logs

def generate_plain_logs(count=100):
    """生成 Plain (纯文本/非结构化) 格式日志 (包含 HTTP 访问信息)"""
    logs = []
    for _ in range(count):
        time = random_time().strftime("%Y-%m-%d %H:%M:%S")
        ip = random.choice(IPS)
        method = random.choice(METHODS)
        path = random.choice(PATHS)
        status = random.choice(STATUS_CODES)
        size = random.randint(100, 10000)
        response_time = random.randint(10, 2500)
        
        # 所有模板都包含 HTTP 访问信息，便于解析器提取
        templates = [
            f"[{time}] {ip} {method} {path} {status} {size} {response_time}ms",
            f"{time} - {ip} - {method} {path} - Status: {status} - Size: {size} - Time: {response_time}ms",
            f"[{time}] {ip} - {method} {path} - {status} - {size} - {response_time}ms",
            f"{ip} [{time}] \"{method} {path}\" {status} {size} {response_time}",
            f"Request from {ip} at {time}: {method} {path} -> {status} ({size} bytes, {response_time}ms)"
        ]
        logs.append(random.choice(templates))
    return logs

def main():
    output_dir = "data"
    
    formats = {
        "test_nginx.log": (generate_nginx_logs, "Nginx Combined Log Format"),
        "test_apache.log": (generate_apache_logs, "Apache Standard Format"),
        "test_json.log": (generate_json_logs, "JSON Format"),
        "test_csv.log": (generate_csv_logs, "CSV Format (comma-separated)"),
        "test_tsv.log": (generate_tsv_logs, "TSV Format (tab-separated)"),
        "test_syslog.log": (generate_syslog_logs, "Syslog Format"),
        "test_pipe.log": (generate_pipe_logs, "Pipe-delimited Format"),
        "test_semicolon.log": (generate_semicolon_logs, "Semicolon-delimited Format"),
        "test_plain.log": (generate_plain_logs, "Plain Text Format"),
    }
    
    print("=" * 60)
    print("生成测试日志文件")
    print("=" * 60)
    
    for filename, (generator, description) in formats.items():
        filepath = f"{output_dir}/{filename}"
        logs = generator(100)  # 每个文件生成100条
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write('\n'.join(logs))
        print(f"✓ {filename:<25} ({description})")
    
    print("=" * 60)
    print("所有测试日志文件生成完成！")
    print(f"文件位置: {output_dir}/")
    print("=" * 60)

if __name__ == "__main__":
    main()
