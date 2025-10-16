#!/bin/bash

# 启动 Genkit AI 服务
# 使用方法: ./start.sh

set -e

# 检查 .env 文件是否存在
if [ ! -f ".env" ]; then
    echo "错误: .env 文件不存在"
    echo "请复制 .env.example 到 .env 并配置必要的环境变量"
    exit 1
fi

# 加载环境变量
export $(cat .env | grep -v '^#' | xargs)

# 检查必需的环境变量
if [ -z "$GENKIT_API_KEY" ]; then
    echo "错误: GENKIT_API_KEY 环境变量未设置"
    exit 1
fi

if [ -z "$DB_HOST" ]; then
    echo "错误: DB_HOST 环境变量未设置"
    exit 1
fi

# 构建服务
echo "构建服务..."
go build -o bin/server ./cmd/server

# 启动服务
echo "启动 Genkit AI 服务..."
./bin/server
