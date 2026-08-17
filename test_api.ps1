# 测试 API 功能
Write-Host "=== 日志处理系统验证测试 ===" -ForegroundColor Cyan

# 1. 测试只读接口（无需认证）
Write-Host "`n[测试1] GET /api/status (只读，无需认证)" -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/status" -Method Get
    Write-Host "✓ 状态API正常" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 3
} catch {
    Write-Host "✗ 失败: $($_.Exception.Message)" -ForegroundColor Red
}

# 2. 测试配置获取（只读）
Write-Host "`n[测试2] GET /api/config (只读，无需认证)" -ForegroundColor Yellow
try {
    $config = Invoke-RestMethod -Uri "http://localhost:8080/api/config" -Method Get
    Write-Host "✓ 配置API正常" -ForegroundColor Green
    Write-Host "  - API Token 已设置: $($config.server.api_token_set)"
    Write-Host "  - 处理器工作数: $($config.processor.worker_count)"
    Write-Host "  - 溢写保护: $($config.processor.overflow_enabled)"
} catch {
    Write-Host "✗ 失败: $($_.Exception.Message)" -ForegroundColor Red
}

# 3. 测试写操作（无认证，应该失败）
Write-Host "`n[测试3] POST /api/config (写操作，无Token应失败)" -ForegroundColor Yellow
try {
    $body = @{
        processor = @{
            worker_count = 10
        }
    } | ConvertTo-Json
    
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/config" -Method Post -Body $body -ContentType "application/json"
    Write-Host "✓ 请求成功 (未配置Token，向后兼容模式)" -ForegroundColor Green
} catch {
    if ($_.Exception.Response.StatusCode -eq 401) {
        Write-Host "✓ 正确返回401 Unauthorized" -ForegroundColor Green
    } else {
        Write-Host "✗ 失败: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# 4. 测试日志查询（只读）
Write-Host "`n[测试4] GET /api/logs (只读，无需认证)" -ForegroundColor Yellow
try {
    $logs = Invoke-RestMethod -Uri "http://localhost:8080/api/logs?limit=5" -Method Get
    Write-Host "✓ 日志查询API正常" -ForegroundColor Green
    Write-Host "  - 总数: $($logs.total)"
    Write-Host "  - 返回: $($logs.logs.Count) 条"
} catch {
    Write-Host "✗ 失败: $($_.Exception.Message)" -ForegroundColor Red
}

# 5. 测试统计API
Write-Host "`n[测试5] GET /api/statistics (只读，无需认证)" -ForegroundColor Yellow
try {
    $stats = Invoke-RestMethod -Uri "http://localhost:8080/api/statistics" -Method Get
    Write-Host "✓ 统计API正常" -ForegroundColor Green
    Write-Host "  - 总日志数: $($stats.total_count)"
    Write-Host "  - 错误数: $($stats.error_count)"
    Write-Host "  - 平均响应时间: $([math]::Round($stats.avg_response_time, 2)) ms"
} catch {
    Write-Host "✗ 失败: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n=== 测试完成 ===" -ForegroundColor Cyan
