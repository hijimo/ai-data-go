# AI知识管理平台 Makefile

.PHONY: help build run test clean deps swagger build-migrate migrate-up migrate-down migrate-version migrate-force migrate-drop migrate-create db-init db-seed db-clean docker-build docker-run

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  deps         - 安装依赖"
	@echo "  build        - 构建应用"
	@echo "  run          - 运行应用"
	@echo "  test         - 运行测试"
	@echo "  clean        - 清理构建文件"
	@echo "  swagger      - 生成Swagger文档"
	@echo "  migrate-up   - 运行数据库迁移"
	@echo "  migrate-down - 回滚数据库迁移"
	@echo "  migrate-version - 查看数据库版本"
	@echo "  migrate-create - 创建新的迁移文件"
	@echo "  db-init      - 初始化数据库（迁移+种子数据）"
	@echo "  db-seed      - 初始化种子数据"
	@echo "  docker-build - 构建Docker镜像"
	@echo "  docker-run   - 运行Docker容器"

# 安装依赖
deps:
	go mod download
	go mod tidy

# 构建应用
build:
	go build -o bin/server cmd/server/main.go

# 运行应用
run:
	go run cmd/server/main.go

# 运行测试
test:
	go test -v ./...

# 清理构建文件
clean:
	rm -rf bin/
	go clean

# 生成Swagger文档
swagger:
	swag init -g cmd/server/main.go -o docs

# 构建迁移工具
build-migrate:
	go build -o bin/migrate cmd/migrate/main.go

# 运行数据库迁移
migrate-up: build-migrate
	./bin/migrate -action=up

# 回滚数据库迁移
migrate-down: build-migrate
	./bin/migrate -action=down

# 查看数据库版本
migrate-version: build-migrate
	./bin/migrate -action=version

# 强制设置数据库版本
migrate-force: build-migrate
	@read -p "输入目标版本号: " version; \
	./bin/migrate -action=force -version=$$version

# 删除所有数据库表
migrate-drop: build-migrate
	./bin/migrate -action=drop

# 初始化数据库（迁移+种子数据）
db-init: build-migrate
	./bin/migrate -action=init

# 初始化种子数据
db-seed: build-migrate
	./bin/migrate -action=seed

# 清理种子数据
db-clean: build-migrate
	./bin/migrate -action=clean

# 创建新的迁移文件
migrate-create: build-migrate
	@read -p "输入迁移文件名: " name; \
	./bin/migrate -action=create -name=$$name

# 构建Docker镜像
docker-build:
	docker build -t ai-knowledge-platform .

# 运行Docker容器
docker-run:
	docker-compose up -d

# 停止Docker容器
docker-stop:
	docker-compose down

# 查看日志
docker-logs:
	docker-compose logs -f

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 安装开发工具
install-tools:
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 开发环境设置
dev-setup: install-tools deps
	cp .env.example .env
	@echo "请编辑 .env 文件配置您的环境变量"

# 完整的开发流程
dev: dev-setup swagger build run