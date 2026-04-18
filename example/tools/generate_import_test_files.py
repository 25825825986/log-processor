#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Generate import test files for .log, .txt, .csv, .json formats.
Each file contains 10,000 log entries.
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
LEVELS = ['INFO', 'WARN', 'ERROR', 'DEBUG']
SERVICES = ['nginx', 'apache', 'api-gateway', 'auth-service', 'database', 'app']
MESSAGES = [
    'Request processed', 'User login successful', 'Invalid token provided',
    'Database connection established', 'API rate limit exceeded',
    'Resource not found', 'Cache miss for key', 'Query executed in 45ms'
]


def random_datetime():
    start = datetime(2026, 3, 8, 0, 0, 0)
    end = datetime(2026, 4, 16, 23, 59, 59)
    delta = end - start
    random_second = random.randrange(int(delta.total_seconds()))
    return start + timedelta(seconds=random_second)


def generate_log_file():
    """.log format: Apache combined-like log entries"""
    lines = []
    for _ in range(NUM_LINES):
        dt = random_datetime()
        dt_str = dt.strftime('%d/%b/%Y:%H:%M:%S +0800')
        line = f'{random.choice(IPS)} - - [{dt_str}] "{random.choice(METHODS)} {random.choice(PATHS)} HTTP/1.1" {random.choice(STATUSES)} {random.randint(100, 10000)}'
        lines.append(line)
    return '\n'.join(lines)


def generate_txt_file():
    """.txt format: plain text structured log entries"""
    lines = []
    for _ in range(NUM_LINES):
        dt = random_datetime()
        line = f'[{dt.strftime("%Y-%m-%d %H:%M:%S")}] [{random.choice(LEVELS)}] {random.choice(IPS)} - {random.choice(METHODS)} {random.choice(PATHS)} - Status: {random.choice(STATUSES)} - Size: {random.randint(100, 10000)} bytes - Time: {random.randint(10, 2500)}ms'
        lines.append(line)
    return '\n'.join(lines)


def generate_csv_file():
    """.csv format: comma-separated values with header"""
    lines = ['client_ip,method,path,status_code,response_size,response_time,timestamp']
    for _ in range(NUM_LINES):
        dt = random_datetime()
        line = f'{random.choice(IPS)},{random.choice(METHODS)},{random.choice(PATHS)},{random.choice(STATUSES)},{random.randint(100, 10000)},{random.randint(10, 2500)},{dt.strftime("%Y-%m-%d %H:%M:%S")}'
        lines.append(line)
    return '\n'.join(lines)


def generate_json_file():
    """.json format: JSON Lines (one JSON object per line)"""
    lines = []
    for _ in range(NUM_LINES):
        dt = random_datetime()
        record = {
            'timestamp': dt.strftime('%Y-%m-%dT%H:%M:%S'),
            'level': random.choice(LEVELS),
            'service': random.choice(SERVICES),
            'client_ip': random.choice(IPS),
            'method': random.choice(METHODS),
            'path': random.choice(PATHS),
            'status_code': random.choice(STATUSES),
            'response_time': random.randint(10, 2500),
            'response_size': random.randint(100, 10000),
            'message': random.choice(MESSAGES),
            'user_agent': random.choice(USER_AGENTS)
        }
        lines.append(json.dumps(record, ensure_ascii=False))
    return '\n'.join(lines)


def write_file(filename, content):
    filepath = os.path.join(OUTPUT_DIR, filename)
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)
    size_kb = os.path.getsize(filepath) / 1024
    print(f'  Written {filename} ({size_kb:.1f} KB, {len(content.splitlines())} lines)')


def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    print(f'Generating import test files ({NUM_LINES} lines each)...')

    print('Generating .log file...')
    write_file('test_import.log', generate_log_file())

    print('Generating .txt file...')
    write_file('test_import.txt', generate_txt_file())

    print('Generating .csv file...')
    write_file('test_import.csv', generate_csv_file())

    print('Generating .json file...')
    write_file('test_import.json', generate_json_file())

    print('Done!')


if __name__ == '__main__':
    main()
