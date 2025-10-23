#!/bin/bash

# 测试时间戳修复的脚本

echo "=========================================="
echo "测试时间戳字段修复"
echo "=========================================="
echo ""

# 1. 检查数据库连接
echo "1. 检查数据库连接..."
if ! psql -h localhost -U postgres -d genkit_ai_service -c "SELECT 1" > /dev/null 2>&1; then
    echo "❌ 无法连接到数据库，请检查数据库配置"
    exit 1
fi
echo "✅ 数据库连接正常"
echo ""

# 2. 检查表结构
echo "2. 检查 tenants 表的时间戳字段..."
CREATED_AT_DEFAULT=$(psql -h localhost -U postgres -d genkit_ai_service -t -c "
    SELECT column_default 
    FROM information_schema.columns 
    WHERE table_name = 'tenants' 
    AND column_name = 'created_at'
")

if [[ $CREATED_AT_DEFAULT == *"CURRENT_TIMESTAMP"* ]]; then
    echo "✅ created_at 字段已设置默认值"
else
    echo "❌ created_at 字段缺少默认值"
    echo "   请运行: ./fix_timestamps.sh"
    exit 1
fi

UPDATED_AT_DEFAULT=$(psql -h localhost -U postgres -d genkit_ai_service -t -c "
    SELECT column_default 
    FROM information_schema.columns 
    WHERE table_name = 'tenants' 
    AND column_name = 'updated_at'
")

if [[ $UPDATED_AT_DEFAULT == *"CURRENT_TIMESTAMP"* ]]; then
    echo "✅ updated_at 字段已设置默认值"
else
    echo "❌ updated_at 字段缺少默认值"
    echo "   请运行: ./fix_timestamps.sh"
    exit 1
fi
echo ""

# 3. 检查已存在记录的时间戳
echo "3. 检查已存在记录的时间戳..."
NULL_TIMESTAMPS=$(psql -h localhost -U postgres -d genkit_ai_service -t -c "
    SELECT COUNT(*) 
    FROM tenants 
    WHERE created_at IS NULL OR updated_at IS NULL
")

if [ "$NULL_TIMESTAMPS" -eq 0 ]; then
    echo "✅ 所有记录都有时间戳"
else
    echo "⚠️  发现 $NULL_TIMESTAMPS 条记录缺少时间戳"
    echo "   请运行: ./fix_timestamps.sh"
fi
echo ""

echo "=========================================="
echo "测试完成！"
echo "=========================================="
