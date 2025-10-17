.PHONY: help build run test clean swagger swagger-install

# 默认目标
help:
	@echo "可用的命令："
	@echo "  make build           - 编译项目"
	@echo "  make run             - 运行服务器"
	@echo "  make test            - 运行测试"
	@echo "  make clean           - 清理编译文件"
	@echo "  make swagger         - 生成 Swagger 文档"
	@echo "  make swagger-install - 安装 Swagger 工具"
	@echo "  make dev             - 开发模式（生成文档并运行）"

# 编译项目
build:
	@echo "编译项目..."
	@go build -o bin/server cmd/server/main.go
	@echo "✅ 编译完成: bin/server"

# 运行服务器
run: build
	@echo "启动服务器..."
	@./bin/server

# 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 清理编译文件
clean:
	@echo "清理编译文件..."
	@rm -rf bin/
	@rm -rf docs/docs.go docs/swagger.json docs/swagger.yaml
	@echo "✅ 清理完成"

# 安装 Swagger 工具
swagger-install:
	@echo "安装 Swagger 工具..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ Swagger 工具安装完成"

# 生成 Swagger 文档
swagger:
	@echo "生成 Swagger 文档..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal; \
	elif [ -f ~/go/bin/swag ]; then \
		~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal; \
	else \
		echo "❌ 错误: swag 命令未找到"; \
		echo "请运行: make swagger-install"; \
		exit 1; \
	fi
	@echo "✅ Swagger 文档生成完成"

# 开发模式：生成文档并运行
dev: swagger build
	@echo "启动开发服务器..."
	@./bin/server
