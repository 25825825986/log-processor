#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Generate test log files for the log-processor import feature.
Each file contains 10,000 log entries in various formats.
"""

import random
import json
import os
from datetime import datetime, timedelta

OUTPUT_DIR = os.path.join(os.path.dirname(__file__), '..', 'data')
NUM_LINES = 10000

IPS = ['127.0.0.1', '192.168.0.1', '192.168.1.100', '10.0.0.50', '172.16.0.25']
METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD']
PATHS = [
    '/', '/home', '/about', '/api/login', '/api/search',
    '/api/users', '/api/orders', '/api/products', '/admin/dashboard',
    '/static/js/app.js', '/css/style.css', '/favicon.ico',
    '/api/v2/items?page=1', '/api/v2/items?page=2'
]
STATUSES = [200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503]
USER_AGENTS = [
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
    'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15',
    'Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36',
    'curl/7.68.0',
    'PostmanRuntime/7.26.8',
]
REFERRERS = ['https://google.com', 'https://bing.com', 'https://example.com', '-']
SERVICES = ['nginx', 'apache', 'api-gateway', 'auth-service', 'database', 'app']
HOSTS = [f'server-{i}' for i in range(1, 6)]
LEVELS = ['INFO', 'WARN', 'ERROR', 'DEBUG']
MESSAGES = [
    'Request processed', 'User login successful', 'Invalid token provided',
    'Database connection established', 'API rate limit exceeded',
    'Resource not found', 'Cache miss for key', 'Query executed in 45ms'
]


def random_datetime(start=None, end=None):
    if start is None:
        start = datetime(2026, 3, 8, 0, 0, 0)
    if end is None:
        end = datetime(2026, 4, 16, 23, 59, 59)
    delta = end - start
    int_delta = int(delta.total_seconds())
    random_second = random.randrange(int_delta)
    return start + timedelta(seconds=random_second)


def generate_apache_logs(n):
    lines = []
    for _ in range(n):
        dt = random_datetime()
        dt_str = dt.strftime('%d/%b/%Y:%H:%M:%S +0800')
        line = f'{random.choice(IPS)} - - [{dt_str}] "{random.choice(METHODS)} {random.choice(PATHS)} HTTP/1.1" {random.choice(STATUSES)} {random.randint(100, 10000)} {round(random.uniform(0.01, 2.5), 3)}'
        lines.append(line)
    return '\n'.join(lines)


def generate_nginx_logs(n):
    lines = []
    for _ in range(n):
        dt = random_datetime()
        dt_str = dt.strftime('%d/%b/%Y:%H:%M:%S +0800')
        line = f'{random.choice(IPS)} - - [{dt_str}] "{random.choice(METHODS)} {random.choice(PATHS)} HTTP/1.1" {random.choice(STATUSES)} {random.randint(100, 10000)} "{random.choice(REFERRERS)}" "{random.choice(USER_AGENTS)}" "{round(random.uniform(0.01, 2.5), 3)}"'
        lines.append(line)
    return '\n'.join(lines)


def generate_json_logs(n):
    lines = []
    for i in range(n):
        dt = random_datetime()
        record = {
            'timestamp': dt.strftime('%Y-%m-%dT%H:%M:%S.') + f'{random.randint(100000, 999999)}',
            'level': random.choice(LEVELS),
            'service': random.choice(SERVICES),
            'client_ip': random.choice(IPS),
            'method': random.choice(METHODS),
            'path': random.choice(PATHS),
            'status_code': random.choice(STATUSES),
            'response_time': random.randint(10, 2500),
            'response_size': random.randint(100, 10000),
            'message': random.choice(MESSAGES),
            'user_agent': random.choice(USER_AGENTS),
            'request_id': f'req_{random.randint(10000, 99999)}'
        }
        lines.append(json.dumps(record, ensure_ascii=False))
    return '\n'.join(lines)


def generate_csv_logs(n):
    lines = []
    for _ in range(n):
        dt = random_datetime()
        line = f'{random.choice(IPS)},{random.choice(METHODS)},{random.choice(PATHS)},{random.choice(STATUSES)},{random.randint(100, 10000)},{random.randint(10, 2500)},{dt.strftime("%Y-%m-%d %H:%M:%S")}'
        lines.append(line)
    return '\n'.join(lines)


def generate_pipe_logs(n):
    lines = []
    for _ in range(n):
        dt = random_datetime()
        line = f'{random.choice(IPS)}|{random.choice(METHODS)}|{random.choice(PATHS)}|{random.choice(STATUSES)}|{random.randint(100, 10000)}|{random.randint(10, 2500)}|{dt.strftime("%Y-%m-%d %H:%M:%S")}'
        lines.append(line)
    return '\n'.join(lines)


def generate_tsv_logs(n):
    lines = []
    for _ in range(n):
        dt = random_datetime()
        line = f'{random.choice(IPS)}\t{random.choice(METHODS)}\t{random.choice(PATHS)}\t{random.choice(STATUSES)}\t{random.randint(100, 10000)}\t{random.randint(10, 2500)}\t{dt.strftime("%Y-%m-%d %H:%M:%S")}'
        lines.append(line)
    return '\n'.join(lines)


def generate_semicolon_logs(n):
    lines = []
    for _ in range(n):
        dt = random_datetime()
        line = f'{random.choice(IPS)};{random.choice(METHODS)};{random.choice(PATHS)};{random.choice(STATUSES)};{random.randint(100, 10000)};{random.randint(10, 2500)};{dt.strftime("%Y-%m-%d %H:%M:%S")}'
        lines.append(line)
    return '\n'.join(lines)


def generate_plain_logs(n):
    """Generate plain text logs with a few consistent simple formats."""
    lines = []
    formats = [
        lambda dt, ip, method, path, status, size, time:
            f'[{dt.strftime("%Y-%m-%d %H:%M:%S")}] {ip} - {method} {path} - {status} - {size} - {time}ms',
        lambda dt, ip, method, path, status, size, time:
            f'{dt.strftime("%Y-%m-%d %H:%M:%S")} - {ip} - {method} {path} - Status: {status} - Size: {size} - Time: {time}ms',
        lambda dt, ip, method, path, status, size, time:
            f'{ip} [{dt.strftime("%Y-%m-%d %H:%M:%S")}] "{method} {path}" {status} {size} {time}',
        lambda dt, ip, method, path, status, size, time:
            f'Request from {ip} at {dt.strftime("%Y-%m-%d %H:%M:%S")}: {method} {path} -> {status} ({size} bytes, {time}ms)',
    ]
    for _ in range(n):
        dt = random_datetime()
        fmt = random.choice(formats)
        line = fmt(dt, random.choice(IPS), random.choice(METHODS), random.choice(PATHS),
                   random.choice(STATUSES), random.randint(100, 10000), random.randint(10, 2500))
        lines.append(line)
    return '\n'.join(lines)


def generate_syslog_logs(n):
    lines = []
    months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
    for _ in range(n):
        dt = random_datetime()
        mon = months[dt.month - 1]
        day = dt.day
        time_str = dt.strftime('%H:%M:%S')
        host = random.choice(HOSTS)
        service = random.choice(SERVICES)
        pid = random.randint(1000, 9999)
        line = f'{mon} {day:2d} {time_str} {host} {service}[{pid}]: {random.choice(IPS)} {random.choice(METHODS)} {random.choice(PATHS)} {random.choice(STATUSES)} {random.randint(100, 10000)}'
        lines.append(line)
    return '\n'.join(lines)


def write_file(filename, content):
    filepath = os.path.join(OUTPUT_DIR, filename)
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)
    size_kb = os.path.getsize(filepath) / 1024
    print(f'  Written {filepath} ({size_kb:.1f} KB, {len(content.splitlines())} lines)')


def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    print(f'Generating {NUM_LINES} lines per format into {OUTPUT_DIR}...')

    print('Generating Apache logs...')
    write_file('test_apache.log', generate_apache_logs(NUM_LINES))

    print('Generating Nginx logs...')
    write_file('test_nginx.log', generate_nginx_logs(NUM_LINES))

    print('Generating JSON logs...')
    write_file('test_json.log', generate_json_logs(NUM_LINES))

    print('Generating CSV logs...')
    write_file('test_csv.log', generate_csv_logs(NUM_LINES))

    print('Generating Pipe-delimited logs...')
    write_file('test_pipe.log', generate_pipe_logs(NUM_LINES))

    print('Generating TSV logs...')
    write_file('test_tsv.log', generate_tsv_logs(NUM_LINES))

    print('Generating Semicolon-delimited logs...')
    write_file('test_semicolon.log', generate_semicolon_logs(NUM_LINES))

    print('Generating Plain text logs...')
    write_file('test_plain.log', generate_plain_logs(NUM_LINES))

    print('Generating Syslog logs...')
    write_file('test_syslog.log', generate_syslog_logs(NUM_LINES))

    print('Done!')


if __name__ == '__main__':
    main()
