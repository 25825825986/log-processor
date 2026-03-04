#!/bin/bash

# 日志发送示例脚本

echo "发送测试日志到日志处理系统..."

# TCP 方式发送 Nginx 格式日志
for i in {1..100}; do
    echo "127.0.0.1 - - [$(date '+%d/%b/%Y:%H:%M:%S %z')] \"GET /api/users/$i HTTP/1.1\" 200 $((RANDOM % 1000 + 100)) \"-\" \"Mozilla/5.0\"" | nc localhost 9000
done

echo "TCP 日志发送完成"

# UDP 方式发送
for i in {1..50}; do
    echo "192.168.1.$i - - [$(date '+%d/%b/%Y:%H:%M:%S %z')] \"POST /api/login HTTP/1.1\" 200 256 \"-\" \"curl/7.68.0\"" | nc -u localhost 9001
done

echo "UDP 日志发送完成"

# HTTP 方式发送 JSON 日志
curl -X POST http://localhost:9002/logs \
  -H "Content-Type: text/plain" \
  -d '{
    "timestamp": "'$(date -Iseconds)'",
    "level": "info",
    "source": "test-app",
    "method": "GET",
    "path": "/health",
    "status_code": 200,
    "response_time": 12,
    "client_ip": "127.0.0.1"
}'

echo "HTTP 日志发送完成"
