.PHONY: help build run test clean swagger swagger-install auth-migrate auth-init auth-setup \
        test-env-setup test-env-verify test-env-init run-test test-e2e test-performance

# 默认目标
help:
	@echo "可用的命令："
	@echo "  make build              - 编译项目"
	@echo "  make run                - 运行服务器"
	@echo "  make test               - 运行测试"
	@echo "  make clean              - 清理编译文件"
	@echo "  make swagger            - 生成 Swagger 文档"
	@echo "  make swagger-install    - 安装 Swagger 工具"
	@echo "  make dev                - 开发模式（生成文档并运行）"
	@echo "  make auth-migrate       - 执行认证系统数据库迁移"
	@echo "  make auth-init          - 初始化默认租户和管理员"
	@echo "  make auth-setup         - 完整认证系统初始化（迁移+初始化）"
	@echo ""
	@echo "测试环境命令："
	@echo "  make test-env-setup     - 配置测试环境"
	@echo "  make test-env-verify    - 验证测试环境配置"
	@echo "  make test-env-init      - 初始化测试数据"
	@echo "  make run-test           - 使用测试环境配置运行服务"
	@echo "  make test-e2e           - 运行端到端测试"
	@echo "  make test-performance   - 运行性能测试"

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
	@./scripts/fix_swagger_names.sh
	@echo "✅ Swagger 文档生成完成"

# 开发模式：生成文档并运行
dev: swagger build
	@echo "启动开发服务器..."
	@./bin/server

# 执行认证系统数据库迁移
auth-migrate:
	@echo "执行认证系统数据库迁移..."
	@go run scripts/auth_migrate.go

# 初始化默认租户和管理员
auth-init:
	@echo "初始化默认租户和管理员..."
	@go run scripts/init_auth.go

# 完整认证系统初始化（迁移+初始化）
auth-setup: auth-migrate auth-init
	@echo ""
	@echo "✅ 认证系统初始化完成！"
	@echo "📖 查看使用说明: docs/AUTH_SETUP.md"

# ========================================
# 测试环境命令
# ========================================

# 配置测试环境
test-env-setup:
	@echo "配置测试环境..."
	@./scripts/setup_test_env.sh

# 验证测试环境配置
test-env-verify:
	@echo "验证测试环境配置..."
	@./scripts/verify_test_env.sh

# 初始化测试数据
test-env-init:
	@echo "初始化测试数据..."
	@./scripts/init_test_data.sh

# 使用测试环境配置运行服务
run-test: build
	@echo "使用测试环境配置启动服务器..."
	@if [ -f .env.test ]; then \
		export $$(cat .env.test | grep -v '^#' | xargs) && ./bin/server; \
	else \
		echo "❌ 错误: .env.test 文件不存在"; \
		echo "请先运行: make test-env-setup"; \
		exit 1; \
	fi

# 运行端到端测试
test-e2e:
	@echo "运行端到端测试..."
	@if [ -f .env.test ]; then \
		export $$(cat .env.test | grep -v '^#' | xargs) && \
		go test -v -timeout 10m ./test/e2e/...; \
	else \
		echo "❌ 错误: .env.test 文件不存在"; \
		echo "请先运行: make test-env-setup"; \
		exit 1; \
	fi

# 运行性能测试
test-performance:
	@echo "运行性能测试..."
	@if [ -f .env.test ]; then \
		export $$(cat .env.test | grep -v '^#' | xargs) && \
		./test/test_performance.sh; \
	else \
		echo "❌ 错误: .env.test 文件不存在"; \
		echo "请先运行: make test-env-setup"; \
		exit 1; \
	fi
