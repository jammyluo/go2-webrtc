.PHONY: build run clean test

# 构建可执行文件
build:
	go build -o go2-webrtc go2_webrtc.go

# 运行程序
run: build
	./go2-webrtc

# 使用环境变量运行
run-with-env:
	GO2_IP=192.168.123.161 GO2_TOKEN=your_token_here ./go2-webrtc

# 清理构建文件
clean:
	rm -f go2-webrtc

# 安装依赖
deps:
	go mod tidy

# 测试编译
test:
	go build -o go2-webrtc go2_webrtc.go
	@echo "✅ 编译成功"

# 显示帮助
help:
	@echo "可用的命令:"
	@echo "  make build      - 构建可执行文件"
	@echo "  make run        - 构建并运行程序"
	@echo "  make run-with-env - 使用默认环境变量运行"
	@echo "  make clean      - 清理构建文件"
	@echo "  make deps       - 安装依赖"
	@echo "  make test       - 测试编译"
	@echo "  make help       - 显示此帮助" 