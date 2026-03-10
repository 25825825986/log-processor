# 日志处理器 - 项目结构说明

```
Log_processor/
│
├── 📄 config.example.json      # 配置文件示例（用户可复制为 config.json 使用）
├── 📄 go.mod, go.sum          # Go 模块依赖定义
├── 📄 Makefile                # 构建和测试命令
├── 📄 README.md               # 项目主文档
├── 📄 PROJECT_STRUCTURE.md    # 本文件
│
├── 📁 cmd/                    # 应用程序入口
│   └── 📁 server/
│       └── 📄 main.go         # 主程序入口
│
├── 📁 internal/               # 内部包（不可被外部导入）
│   ├── 📁 config/             # 配置管理
│   │   └── 📄 config.go
│   ├── 📁 exporter/           # 数据导出功能
│   │   └── 📄 exporter.go
│   ├── 📁 models/             # 数据模型
│   │   ├── 📄 log.go          # 日志条目模型
│   │   └── 📄 uuid.go         # UUID 生成
│   ├── 📁 parser/             # 日志解析器
│   │   └── 📄 parser.go
│   ├── 📁 processor/          # 日志处理器
│   │   └── 📄 processor.go
│   ├── 📁 receiver/           # 网络接收器 (TCP/UDP/HTTP)
│   │   └── 📄 receiver.go
│   ├── 📁 server/             # Web 服务器
│   │   └── 📄 server.go
│   └── 📁 storage/            # 数据存储 (SQLite)
│       └── 📄 storage.go
│
├── 📁 web/                    # 前端静态资源
│   ├── 📄 index.html          # 主页面
│   ├── 📁 css/
│   │   └── 📄 style.css       # 样式表
│   └── 📁 js/
│       └── 📄 app.js          # 前端逻辑
│
├── 📁 example/                # 测试工具和数据
│   ├── 📄 README.md           # 测试工具使用说明
│   ├── 📄 DATASETS.md         # 开源数据集说明
│   ├── 📁 benchmark/          # 性能测试脚本
│   │   ├── 📄 stress_test.py       # Python 压力测试
│   │   ├── 📄 stress_test.go       # Go 压力测试
│   │   └── 📄 find_max_capacity.py # 查找系统极限
│   ├── 📁 data/               # 测试数据文件
│   │   ├── 📄 test_log.txt
│   │   ├── 📄 test_logs.txt
│   │   └── 📄 test_logs_json.txt
│   └── 📁 tools/              # 辅助工具
│       ├── 📄 generate_test_logs.py    # 生成测试日志
│       ├── 📄 convert_log_format.py    # 格式转换
│       ├── 📄 download_nasa_unix.sh    # NASA 数据集 (Unix)
│       ├── 📄 download_nasa_windows.ps1# NASA 数据集 (Windows)
│       ├── 📄 send_logs_unix.sh        # 发送日志 (Unix)
│       └── 📄 send_logs_windows.ps1    # 发送日志 (Windows)
│
├── 📁 data/                   # 运行时数据存储
│   ├── 📄 logs.db             # SQLite 数据库
│   ├── 📄 logs.db-shm         # SQLite 共享内存
│   └── 📄 logs.db-wal         # SQLite WAL 文件
│
├── 📁 logs/                   # 应用程序日志
│   └── 📄 YYYY-MM-DD_HH-MM-SS.log
│
├── 📁 exports/                # 数据导出目录（运行时生成）
│
└── 📁 temp/                   # 临时文件目录（运行时生成）
```

---

## 🚀 快速开始

```bash
# 1. 安装依赖
go mod download

# 2. 复制配置文件
cp config.example.json config.json

# 3. 启动服务
go run cmd/server/main.go
# 或使用配置
go run cmd/server/main.go -config config.json

# 4. 访问 Web 界面
open http://localhost:8080
```

---

## 🧪 性能测试

```bash
# 压力测试
cd example/benchmark
python stress_test.py -protocol tcp -total 10000 -c 10

# 查找系统极限
python find_max_capacity.py
```

---

## 📊 存储占用说明

| 目录 | 用途 | 可删除 |
|------|------|--------|
| `data/` | SQLite 数据库 | ❌ 生产数据 |
| `logs/` | 应用日志 | ⚠️ 保留最近2个 |
| `exports/` | 导出的报表 | ✅ 可清理 |
| `temp/` | 临时上传文件 | ✅ 可清理 |

---

## 📝 开发规范

- **cmd/**: 只包含 main 包和程序入口
- **internal/**: 核心业务逻辑，不对外暴露
- **web/**: 纯静态文件，无后端渲染
- **example/**: 测试和工具脚本，不参与主构建
