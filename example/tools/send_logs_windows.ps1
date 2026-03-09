# 日志发送示例脚本 (PowerShell)

Write-Host "发送测试日志到日志处理系统..."

# TCP 方式发送 Nginx 格式日志
try {
    $tcpClient = New-Object System.Net.Sockets.TcpClient
    $tcpClient.Connect("localhost", 9000)
    $stream = $tcpClient.GetStream()
    $writer = New-Object System.IO.StreamWriter($stream)

    for ($i = 1; $i -le 100; $i++) {
        $timestamp = Get-Date -Format "dd/MMM/yyyy:HH:mm:ss zzz"
        $size = Get-Random -Minimum 100 -Maximum 1000
        $log = "127.0.0.1 - - [$timestamp] `"GET /api/users/$i HTTP/1.1`" 200 $size `"`" `"Mozilla/5.0`""
        $writer.WriteLine($log)
        $writer.Flush()
    }

    $writer.Close()
    $tcpClient.Close()
    Write-Host "TCP 日志发送完成 (100条)"
} catch {
    Write-Host "TCP 发送失败: $_"
}

# HTTP 方式发送
try {
    $body = @"
127.0.0.1 - - [$(Get-Date -Format "dd/MMM/yyyy:HH:mm:ss zzz")] `"GET /api/test HTTP/1.1`" 200 512 `"`" `"PowerShell`"
"@

    Invoke-RestMethod -Uri "http://localhost:9002/logs" -Method Post -Body $body -ContentType "text/plain"
    Write-Host "HTTP 日志发送完成"
} catch {
    Write-Host "HTTP 发送失败: $_"
}

Write-Host "所有测试日志发送完成！"
