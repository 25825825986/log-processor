# 开源日志数据集使用指南

## 推荐数据集

### 1. NASA HTTP 日志 ⭐推荐
- **来源**: NASA Kennedy Space Center WWW 服务器
- **时间**: 1995年7月
- **记录数**: 约 130 万条
- **格式**: 类 Nginx 格式
- **用途**: 系统功能测试、流量分析
- **获取方式**:

```bash
# 方法1: 直接下载（约 200MB）
wget ftp://ita.ee.lbl.gov/traces/NASA_access_log_Jul95.gz

# 方法2: 使用提供的脚本
cd example
./download_nasa_logs.sh
```

**格式示例**:
```
199.72.81.55 - - [01/Jul/1995:00:00:01 -0400] "GET /history/apollo/ HTTP/1.0" 200 6245
```

**导入系统**:
1. 系统配置保持默认（Nginx 格式）
2. 直接导入下载的文件
3. 即可分析 1995 年 NASA 网站的访问情况

---

### 2. Apache 官方示例日志
- **来源**: Apache 官方文档
- **格式**: 标准 Apache/Nginx Combined Log Format
- **用途**: 格式兼容性测试
- **获取**:

```bash
# 使用系统自带的测试数据
cat example/test_logs.txt

# 或使用 Apache 官方示例
curl https://raw.githubusercontent.com/elastic/examples/master/Common%20Data%20Formats/apache_logs/apache_logs
```

---

### 3. SecRepo 安全日志数据集
- **来源**: http://www.secrepo.com/
- **内容**: 包含攻击样本的安全日志
- **用途**: 安全分析、入侵检测测试
- **包含**:
  - Apache 攻击日志
  - SSH 暴力破解记录
  - Web 应用防火墙日志

**下载**:
```bash
wget http://www.secrepo.com/maccdc2012/http.log.gz
```

---

### 4. Kaggle 数据集（需注册）
访问 https://www.kaggle.com/datasets 搜索 "web logs"

推荐数据集：
- **Web Server Log Samples**: 包含正常和攻击请求
- **E-commerce Logs**: 电商网站用户行为数据
- **System Logs**: Linux 系统日志

下载方式：
```bash
# 先安装 kaggle API: pip install kaggle
# 在 https://www.kaggle.com/account 获取 API Token

python example/download_kaggle_logs.py
```

---

## 格式转换

如果下载的数据集格式不匹配，使用转换脚本：

```bash
# 转换为 Nginx 格式（系统默认）
python example/convert_logs.py \
    input.csv \
    output.txt \
    --input-format csv \
    --output-format nginx

# 转换为 JSON 格式
python example/convert_logs.py \
    nasa_logs.txt \
    output.json \
    --input-format nasa \
    --output-format json
```

---

## 数据说明

### 数据隐私
- **NASA 日志**: 1995年的公开数据，无隐私问题
- **开源数据集**: 均已脱敏处理
- **建议使用**: 学术研究、系统测试

### 数据规模参考
| 数据集 | 大小 | 记录数 | 适用场景 |
|--------|------|--------|----------|
| test_logs.txt | 2KB | 10条 | 功能测试 |
| NASA Jul 95 | 200MB | 1.3M条 | 性能测试 |
| SecRepo | 500MB+ | 数百万条 | 安全分析 |

---

## 使用建议

### 毕业设计/论文使用
1. **功能演示**: 使用 `test_logs.txt` 或 NASA 日志
2. **性能测试**: 使用完整的 NASA 数据集
3. **安全分析**: 使用 SecRepo 的攻击日志

### 数据来源声明
在论文中引用时，请注明数据来源：
```
本研究使用的测试数据来源于：
1. NASA HTTP Logs (1995) - NASA Kennedy Space Center
2. SecRepo Security Datasets - http://www.secrepo.com/
```

---

## 自定义数据生成

如果需要特定格式的测试数据，使用以下工具：

### 1. Flog (Go 日志生成器)
```bash
go install github.com/mingrammer/flog@latest

# 生成 Nginx 格式日志
flog -n 10000 -f nginx > my_logs.txt
```

### 2. Python 脚本生成
```python
# example/generate_logs.py 已提供
python example/generate_logs.py --count 1000 --format nginx
```
