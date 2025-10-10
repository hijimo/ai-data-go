#!/bin/bash

echo "验证多格式文档解析器实现..."

# 检查必要的文件是否存在
files=(
    "internal/processor/document.go"
    "internal/processor/pdf.go"
    "internal/processor/docx.go"
    "internal/processor/markdown.go"
    "internal/processor/text.go"
    "internal/processor/html.go"
    "internal/processor/chunker.go"
    "internal/service/document.go"
    "internal/handler/document.go"
    "internal/processor/document_test.go"
    "internal/processor/chunker_test.go"
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

# 检查文档处理器接口
if grep -q "type DocumentProcessor interface" internal/processor/document.go; then
    echo "✓ 文档处理器接口已定义"
else
    echo "✗ 文档处理器接口未定义"
fi

# 检查处理器管理器
if grep -q "type ProcessorManager struct" internal/processor/document.go; then
    echo "✓ 处理器管理器已实现"
else
    echo "✗ 处理器管理器未实现"
fi

# 检查各种格式处理器
formats=("PDF" "DOCX" "Markdown" "Text" "HTML")
for format in "${formats[@]}"; do
    if grep -q "${format}Processor" internal/processor/*.go; then
        echo "✓ ${format}处理器已实现"
    else
        echo "✗ ${format}处理器未实现"
    fi
done

echo ""
echo "多格式文档解析器实现验证完成!"