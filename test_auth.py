#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""测试 API Token 认证功能"""

import requests
import json

BASE_URL = "http://localhost:8080"
TEST_TOKEN = "test-secret-token-12345"

def test_with_token():
    """测试配置API Token后的认证流程"""
    print("\n" + "="*60)
    print("  测试 API Token 认证功能")
    print("="*60)
    
    # 步骤1: 先配置一个测试token（当前未设置token，可以直接修改）
    print("\n[步骤1] 设置 API Token")
    try:
        payload = {
            "server": {
                "host": "0.0.0.0",
                "port": 8080,
                "api_token": TEST_TOKEN,
                "cors_origins": []
            }
        }
        response = requests.post(f"{BASE_URL}/api/config", json=payload, timeout=3)
        if response.status_code == 200:
            print(f"✓ Token设置成功")
        else:
            print(f"✗ 设置失败: {response.status_code}")
            print(f"  响应: {response.text}")
            return
    except Exception as e:
        print(f"✗ 异常: {e}")
        return
    
    # 步骤2: 验证配置API不暴露token明文
    print("\n[步骤2] 验证配置API不暴露Token明文")
    try:
        response = requests.get(f"{BASE_URL}/api/config", timeout=3)
        config = response.json()
        server_config = config.get('server', {})
        
        if 'api_token' not in server_config:
            print("✓ 配置中不包含api_token字段")
        else:
            print("✗ 警告: 配置中暴露了api_token字段")
        
        if server_config.get('api_token_set') == True:
            print("✓ api_token_set 正确显示为 True")
        else:
            print("✗ api_token_set 值不正确")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    # 步骤3: 测试无Token访问写接口（应该被拒绝）
    print("\n[步骤3] 无Token访问写接口（应返回401）")
    try:
        payload = {"processor": {"worker_count": 25}}
        response = requests.post(f"{BASE_URL}/api/config", json=payload, timeout=3)
        
        if response.status_code == 401:
            print(f"✓ 正确拒绝: HTTP 401")
            print(f"  响应: {response.json()}")
        else:
            print(f"✗ 未正确拒绝: HTTP {response.status_code}")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    # 步骤4: 使用错误Token访问（应该被拒绝）
    print("\n[步骤4] 使用错误Token访问（应返回401）")
    try:
        headers = {"Authorization": "Bearer wrong-token"}
        payload = {"processor": {"worker_count": 25}}
        response = requests.post(
            f"{BASE_URL}/api/config",
            json=payload,
            headers=headers,
            timeout=3
        )
        
        if response.status_code == 401:
            print(f"✓ 正确拒绝: HTTP 401")
        else:
            print(f"✗ 未正确拒绝: HTTP {response.status_code}")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    # 步骤5: 使用正确Token通过Authorization header访问
    print("\n[步骤5] 使用正确Token (Authorization header)")
    try:
        headers = {"Authorization": f"Bearer {TEST_TOKEN}"}
        payload = {"processor": {"worker_count": 30}}
        response = requests.post(
            f"{BASE_URL}/api/config",
            json=payload,
            headers=headers,
            timeout=3
        )
        
        if response.status_code == 200:
            print(f"✓ 认证成功: HTTP 200")
        else:
            print(f"✗ 失败: HTTP {response.status_code}")
            print(f"  响应: {response.text}")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    # 步骤6: 使用X-API-Token header
    print("\n[步骤6] 使用正确Token (X-API-Token header)")
    try:
        headers = {"X-API-Token": TEST_TOKEN}
        payload = {"processor": {"worker_count": 35}}
        response = requests.post(
            f"{BASE_URL}/api/config",
            json=payload,
            headers=headers,
            timeout=3
        )
        
        if response.status_code == 200:
            print(f"✓ 认证成功: HTTP 200")
        else:
            print(f"✗ 失败: HTTP {response.status_code}")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    # 步骤7: 使用查询参数token
    print("\n[步骤7] 使用正确Token (查询参数)")
    try:
        payload = {"processor": {"worker_count": 40}}
        response = requests.post(
            f"{BASE_URL}/api/config?token={TEST_TOKEN}",
            json=payload,
            timeout=3
        )
        
        if response.status_code == 200:
            print(f"✓ 认证成功: HTTP 200")
        else:
            print(f"✗ 失败: HTTP {response.status_code}")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    # 步骤8: 验证只读接口仍然无需认证
    print("\n[步骤8] 验证只读接口无需Token")
    try:
        response = requests.get(f"{BASE_URL}/api/status", timeout=3)
        if response.status_code == 200:
            print(f"✓ 只读接口无需认证，正常访问")
        else:
            print(f"✗ 只读接口访问失败: HTTP {response.status_code}")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    # 步骤9: 测试DELETE操作的认证
    print("\n[步骤9] 测试DELETE操作认证")
    try:
        # 先测试无Token删除（应失败）
        response = requests.delete(f"{BASE_URL}/api/logs", timeout=3)
        if response.status_code == 401:
            print(f"✓ 无Token正确拒绝: HTTP 401")
        else:
            print(f"? 响应: HTTP {response.status_code}")
        
        # 再测试有Token删除（会成功但我们不执行，只测试认证）
        headers = {"Authorization": f"Bearer {TEST_TOKEN}"}
        # 使用一个不存在的ID，避免真删数据
        response = requests.delete(
            f"{BASE_URL}/api/logs/test-non-exist-id",
            headers=headers,
            timeout=3
        )
        if response.status_code in [200, 404, 500]:
            print(f"✓ 有Token可访问删除接口 (HTTP {response.status_code})")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    # 步骤10: 恢复原配置（移除token）
    print("\n[步骤10] 恢复原配置（移除Token）")
    try:
        headers = {"Authorization": f"Bearer {TEST_TOKEN}"}
        payload = {
            "server": {
                "host": "0.0.0.0",
                "port": 8080,
                "api_token": "",
                "cors_origins": []
            }
        }
        response = requests.post(
            f"{BASE_URL}/api/config",
            json=payload,
            headers=headers,
            timeout=3
        )
        if response.status_code == 200:
            print(f"✓ 配置恢复成功")
        else:
            print(f"✗ 恢复失败: {response.status_code}")
    except Exception as e:
        print(f"✗ 异常: {e}")
    
    print("\n" + "="*60)
    print("  API Token 认证测试完成")
    print("="*60)

if __name__ == "__main__":
    test_with_token()
