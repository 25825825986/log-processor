# 日志发送示例脚本 (PowerShell)
# 使用 UTF-8 with BOM 编码

Write-Host "Sending test logs to log processor..."

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
    Write-Host "TCP: Sent 100 logs"
} catch {
    Write-Host "TCP Error: $_"
}

# HTTP 方式发送
try {
    $body = @"
127.0.0.1 - - [$(Get-Date -Format "dd/MMM/yyyy:HH:mm:ss zzz")] `"GET /api/test HTTP/1.1`" 200 512 `"`" `"PowerShell`"
"@

    Invoke-RestMethod -Uri "http://localhost:9002/logs" -Method Post -Body $body -ContentType "text/plain"
    Write-Host "HTTP: Sent 1 log"
} catch {
    Write-Host "HTTP Error: $_"
}

Write-Host "Done!"
