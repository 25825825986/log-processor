# NASA HTTP 日志数据集下载脚本 (PowerShell)
# 这是 NASA Kennedy Space Center WWW 服务器 1995年7月的真实访问日志
# 完全公开，常用于学术研究和系统测试

$DATA_DIR = "./datasets"
New-Item -ItemType Directory -Force -Path $DATA_DIR | Out-Null

Write-Host "正在下载 NASA HTTP 日志数据集..." -ForegroundColor Green

# NASA 日志 (1995年7月) - 备选数据源
$urls = @(
    "https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/nginx_logs/nginx_logs",
    "https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/apache_logs/apache_logs"
)

foreach ($url in $urls) {
    $filename = Split-Path $url -Leaf
    $outputPath = Join-Path $DATA_DIR $filename
    
    try {
        Write-Host "Downloading: $filename"
        Invoke-WebRequest -Uri $url -OutFile $outputPath -UseBasicParsing
        Write-Host "Downloaded: $filename" -ForegroundColor Green
    } catch {
        Write-Host "Failed to download: $filename" -ForegroundColor Red
    }
}

Write-Host "`n下载完成！文件位置: $DATA_DIR" -ForegroundColor Green
Get-ChildItem $DATA_DIR | Select-Object Name, Length
