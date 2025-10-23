#!/bin/bash

# 修复时间戳字段的脚本

echo "开始修复时间戳字段..."

# 运行 Go 脚本
go run scripts/fix_timestamps.go

if [ $? -eq 0 ]; then
    echo "✅ 时间戳字段修复成功！"
else
    echo "❌ 时间戳字段修复失败，请检查错误信息"
    exit 1
fi
