# Go2 WebRTC 代理服务器 Makefile

.PHONY: build run clean test deps

# 默认目标
all: build

# 安装依赖
deps:
	go mod tidy
	go mod download

# 构建代理服务器
build: deps
	go build -o webrtc-proxy .

arm:
	GOOS=linux GOARCH=arm64 go build -o webrtc-proxy-arm64 .
# 运行代理服务器
run: build
	./webrtc-proxy

# 开发模式运行（自动重新构建）
dev: build
	@echo "启动WebRTC代理服务器..."
	@echo "访问 http://localhost:8080 查看控制面板"
	./webrtc-proxy

# 清理构建文件
clean:
	rm -f webrtc-proxy
	go clean

# 运行测试
test:
	go test ./...

# 格式化代码
fmt:
	go fmt .

# 代码检查
lint:
	golangci-lint run

# 安装开发工具
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 帮助信息
help:
	@echo "可用的命令:"
	@echo "  build        - 构建代理服务器"
	@echo "  run          - 构建并运行代理服务器"
	@echo "  dev          - 开发模式运行"
	@echo "  clean        - 清理构建文件"
	@echo "  test         - 运行测试"
	@echo "  fmt          - 格式化代码"
	@echo "  lint         - 代码检查"
	@echo "  deps         - 安装依赖"
	@echo "  install-tools - 安装开发工具"
	@echo "  help         - 显示此帮助信息" 