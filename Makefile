.PHONY: build run test clean install docker

# 变量
BINARY_NAME=log-processor
BUILD_DIR=build
GO=go
GOFLAGS=-v

# 构建
build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server/main.go

# 运行
run:
	$(GO) run ./cmd/server/main.go

# 运行（带配置）
run-config:
	$(GO) run ./cmd/server/main.go -config ./config.json

# 测试
test:
	$(GO) test -v ./...

# 清理
clean:
	@rm -rf $(BUILD_DIR)
	@rm -rf data/
	@rm -rf temp/
	@rm -rf exports/
	@echo "Cleaned"

# 安装依赖
deps:
	$(GO) mod download
	$(GO) mod tidy

# 格式化代码
fmt:
	$(GO) fmt ./...

# 代码检查
lint:
	golangci-lint run

# 交叉编译
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server/main.go
	GOOS=linux GOARCH=arm64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/server/main.go
	# macOS
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server/main.go
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/server/main.go
	# Windows
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/server/main.go

# Docker 构建
docker-build:
	docker build -t log-processor:latest .

# Docker 运行
docker-run:
	docker run -p 8080:8080 -p 9000:9000 -p 9001:9001 -p 9002:9002 -v $(PWD)/data:/app/data log-processor:latest

# 帮助
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  run          - Run the server"
	@echo "  run-config   - Run with config file"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  fmt          - Format code"
	@echo "  build-all    - Build for all platforms"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
