#!/usr/bin/env python3
"""
日志格式转换脚本
将各种开源数据集转换为系统可解析的格式
"""

import argparse
import json
import re
from datetime import datetime
from pathlib import Path


def parse_nasa_log(line):
    """解析 NASA HTTP 日志格式"""
    # 格式: host - - [timestamp] "request" status bytes
    pattern = r'^(\S+)\s+-\s+-\s+\[([^\]]+)\]\s+"([^"]+)"\s+(\d+)\s+(\S+)'
    match = re.match(pattern, line)
    
    if not match:
        return None
    
    ip, timestamp, request, status, size = match.groups()
    
    # 解析请求行: GET /path HTTP/1.0
    req_parts = request.split()
    method = req_parts[0] if len(req_parts) > 0 else "GET"
    path = req_parts[1] if len(req_parts) > 1 else "/"
    
    # 转换时间格式
    # 原始: 01/Jul/1995:00:00:01 -0400
    # 目标: 01/Jul/1995:00:00:01 -0400 (Nginx 格式兼容)
    
    return {
        "client_ip": ip,
        "timestamp": timestamp,
        "method": method,
        "path": path,
        "status_code": int(status),
        "response_size": int(size) if size.isdigit() else 0,
        "protocol": req_parts[2] if len(req_parts) > 2 else "HTTP/1.0"
    }


def parse_csv_log(line, delimiter=","):
    """解析 CSV 格式日志"""
    parts = line.split(delimiter)
    if len(parts) < 5:
        return None
    
    # 假设格式: ip,time,method,path,status
    return {
        "client_ip": parts[0].strip(),
        "timestamp": parts[1].strip(),
        "method": parts[2].strip(),
        "path": parts[3].strip(),
        "status_code": int(parts[4].strip())
    }


def to_nginx_format(log_dict):
    """转换为 Nginx 格式"""
    # Nginx 格式: ip - - [time] "method path protocol" status size
    timestamp = log_dict.get("timestamp", "")
    # 确保时间格式符合 Nginx 格式
    if " " in timestamp and "-" in timestamp.split()[-1]:
        # 已经有 timezone，保持原样
        pass
    else:
        # 添加默认 timezone
        timestamp = f"{timestamp} +0800"
    
    return '{ip} - - [{time}] "{method} {path} {protocol}" {status} {size}'.format(
        ip=log_dict.get("client_ip", "-"),
        time=timestamp,
        method=log_dict.get("method", "GET"),
        path=log_dict.get("path", "/"),
        protocol=log_dict.get("protocol", "HTTP/1.1"),
        status=log_dict.get("status_code", 200),
        size=log_dict.get("response_size", 0)
    )


def to_json_format(log_dict):
    """转换为 JSON 格式"""
    return json.dumps(log_dict, ensure_ascii=False)


def convert_file(input_file, output_file, input_format="auto", output_format="nginx"):
    """转换日志文件"""
    input_path = Path(input_file)
    if not input_path.exists():
        print(f"错误: 文件 {input_file} 不存在")
        return False
    
    # 自动检测格式
    if input_format == "auto":
        first_line = input_path.read_text().split('\n')[0]
        if first_line.startswith('{'):
            input_format = "json"
        elif ',' in first_line and len(first_line.split(',')) > 3:
            input_format = "csv"
        elif '[' in first_line and ']' in first_line:
            input_format = "nasa"
        else:
            input_format = "nginx"
        print(f"自动检测到格式: {input_format}")
    
    # 转换
    with open(input_file, 'r', encoding='utf-8', errors='ignore') as f_in, \
         open(output_file, 'w', encoding='utf-8') as f_out:
        
        line_count = 0
        success_count = 0
        
        for line in f_in:
            line = line.strip()
            if not line:
                continue
            
            line_count += 1
            log_dict = None
            
            # 解析输入
            try:
                if input_format == "nasa":
                    log_dict = parse_nasa_log(line)
                elif input_format == "json":
                    log_dict = json.loads(line)
                elif input_format == "csv":
                    log_dict = parse_csv_log(line)
                else:
                    # 已经是 nginx 格式，直接复制
                    f_out.write(line + '\n')
                    success_count += 1
                    continue
            except Exception as e:
                print(f"解析第 {line_count} 行失败: {e}")
                continue
            
            if log_dict:
                # 输出转换后的格式
                if output_format == "nginx":
                    f_out.write(to_nginx_format(log_dict) + '\n')
                elif output_format == "json":
                    f_out.write(to_json_format(log_dict) + '\n')
                success_count += 1
        
        print(f"转换完成: {success_count}/{line_count} 行成功")
        return True


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='日志格式转换工具')
    parser.add_argument('input', help='输入文件路径')
    parser.add_argument('output', help='输出文件路径')
    parser.add_argument('--input-format', choices=['auto', 'nasa', 'json', 'csv', 'nginx'], 
                        default='auto', help='输入格式 (默认: 自动检测)')
    parser.add_argument('--output-format', choices=['nginx', 'json'], 
                        default='nginx', help='输出格式 (默认: nginx)')
    
    args = parser.parse_args()
    convert_file(args.input, args.output, args.input_format, args.output_format)
