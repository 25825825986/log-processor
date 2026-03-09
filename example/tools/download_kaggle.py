#!/usr/bin/env python3
"""
Kaggle 日志数据集下载脚本
需要先安装 kaggle API: pip install kaggle
并在 https://www.kaggle.com/account 获取 API Token
"""

import os
import subprocess

def download_dataset(dataset_name, output_dir="./datasets"):
    """从 Kaggle 下载数据集"""
    os.makedirs(output_dir, exist_ok=True)
    
    cmd = f"kaggle datasets download -d {dataset_name} -p {output_dir}"
    print(f"Downloading {dataset_name}...")
    subprocess.run(cmd, shell=True, check=True)
    print(f"Downloaded to {output_dir}")

if __name__ == "__main__":
    # 推荐的数据集（无需注册即可浏览，下载需要 Kaggle 账号）
    
    # 1. Web 服务器日志（含攻击样本）
    # 包含正常请求和攻击请求，适合测试安全分析功能
    download_dataset("defconnoob/web-server-log-samples")
    
    # 2. 电子商务网站日志
    # 包含用户行为数据，适合测试用户行为分析
    # download_dataset("omarsobhy14/ecommerce-logs")
    
    # 3. 系统日志（Linux系统日志）
    # download_dataset("therohk/million-headlines")
