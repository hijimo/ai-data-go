#!/bin/bash

echo "验证智能文本分块功能实现..."

# 检查必要的文件是否存在
files=(
    "internal/processor/chunker.go"
    "internal/processor/chunker_test.go"
    "internal/handler/chunk_config.go"
    "internal/handler/chunk_visualization.go"
)

echo "检查文件是否存在:"
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ $file"
    else
        echo "✗ $file (缺失)"
    fi
done

echo ""
echo "检查关键功能实现:"

# 检查分块器接口
if grep -q "type Chunker interface" internal/processor/chunker.go; then
    echo "✓ 分块器接口已定义"
else
    echo "✗ 分块器接口未定义"
fi

# 检查分块策略
strategies=("FixedSizeChunker" "SemanticChunker" "StructureChunker" "CodeChunker")
for strategy in "${strategies[@]}"; do
    if grep -q "$strategy" internal/processor/chunker.go; then
        echo "✓ ${strategy}已实现"
    else
        echo "✗ ${strategy}未实现"
    fi
done

# 检查分块配置管理
if grep -q "GetDefaultConfigs" internal/handler/chunk_config.go; then
    echo "✓ 默认配置管理已实现"
else
    echo "✗ 默认配置管理未实现"
fi

# 检查配置验证
if grep -q "ValidateConfig" internal/handler/chunk_config.go; then
    echo "✓ 配置验证已实现"
else
    echo "✗ 配置验证未实现"
fi

# 检查配置推荐
if grep -q "GetRecommendedConfig" internal/handler/chunk_config.go; then
    echo "✓ 配置推荐已实现"
else
    echo "✗ 配置推荐未实现"
fi

# 检查分块可视化
if grep -q "VisualizeChunks" internal/handler/chunk_visualization.go; then
    echo "✓ 分块可视化已实现"
else
    echo "✗ 分块可视化未实现"
fi

# 检查策略比较
if grep -q "CompareChunkStrategies" internal/handler/chunk_visualization.go; then
    echo "✓ 策略比较已实现"
else
    echo "✗ 策略比较未实现"
fi

# 检查统计信息
if grep -q "GetChunkStatistics" internal/handler/chunk_visualization.go; then
    echo "✓ 统计信息已实现"
else
    echo "✗ 统计信息未实现"
fi

echo ""
echo "智能文本分块功能实现验证完成!"