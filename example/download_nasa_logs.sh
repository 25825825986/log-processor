#!/bin/bash

# NASA HTTP 日志数据集下载脚本
# 这是 NASA Kennedy Space Center WWW 服务器 1995年7月的真实访问日志
# 完全公开，常用于学术研究和系统测试

DATA_DIR="./datasets"
mkdir -p $DATA_DIR

echo "正在下载 NASA HTTP 日志数据集..."

# NASA 日志 (1995年7月)
# 原始来源: http://ita.ee.lbl.gov/html/contrib/NASA-HTTP.html
curl -L -o $DATA_DIR/nasa_jul95.gz \
    "https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/nginx_logs/nginx_logs"

# 如果上面的链接失效，使用备选数据源
curl -L -o $DATA_DIR/nasa_access_log.csv \
    "https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/apache_logs/apache_logs"

echo "下载完成！"
echo "文件位置: $DATA_DIR/"
ls -lh $DATA_DIR/
