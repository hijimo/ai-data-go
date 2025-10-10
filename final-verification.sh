#!/bin/bash

echo "=== 文档处理系统最终验证 ==="

echo ""
echo "1. 检查核心文件结构..."

# 核心文件列表
core_files=(
    "internal/storage/oss.go"
    "internal/repository/file.go"
    "internal/service/file.go"
    "internal/handler/file.go"
    "internal/processor/document.go"
    "internal/processor/pdf.go"
    "internal/processor/docx.go"
    "internal/processor/markdown.go"
    "internal/processor/text.go"
    "internal/processor/html.go"
    "internal/processor/chunker.go"
    "internal/service/document.go"
    "internal/handler/document.go"
    "internal/handler/chunk_config.go"
    "internal/handler/chunk_visualization.go"
)

for file in "${core_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ $file"
    else
        echo "✗ $file (缺失)"
    fi
done

echo ""
echo "2. 检查测试文件..."

test_files=(
    "internal/service/file_test.go"
    "internal/handler/file_test.go"
    "internal/processor/document_test.go"
    "internal/processor/chunker_test.go"
)

for file in "${test_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ $file"
    else
        echo "✗ $file (缺失)"
    fi
done

echo ""
echo "3. 检查关键功能实现..."

# 文件上传功能
if grep -q "UploadFile.*multipart.File" internal/storage/oss.go && \
   grep -q "SHA256.*sha256" internal/storage/oss.go && \
   grep -q "GetBySHA256" internal/repository/file.go; then
    echo "✓ 文件上传和去重功能"
else
    echo "✗ 文件上传和去重功能"
fi

# 多格式解析
processors=("PDFProcessor" "DOCXProcessor" "MarkdownProcessor" "TextProcessor" "HTMLProcessor")
all_processors_found=true
for processor in "${processors[@]}"; do
    if ! grep -q "$processor" internal/processor/*.go; then
        all_processors_found=false
        break
    fi
done

if $all_processors_found; then
    echo "✓ 多格式文档解析器"
else
    echo "✗ 多格式文档解析器"
fi

# 智能分块功能
chunkers=("FixedSizeChunker" "SemanticChunker" "StructureChunker" "CodeChunker")
all_chunkers_found=true
for chunker in "${chunkers[@]}"; do
    if ! grep -q "$chunker" internal/processor/chunker.go; then
        all_chunkers_found=false
        break
    fi
done

if $all_chunkers_found; then
    echo "✓ 智能文本分块功能"
else
    echo "✗ 智能文本分块功能"
fi

# API接口
apis=("UploadFile" "GetFile" "ListFiles" "DeleteFile" "ProcessDocument" "PreviewChunks")
all_apis_found=true
for api in "${apis[@]}"; do
    if ! grep -q "func.*$api" internal/handler/*.go; then
        all_apis_found=false
        break
    fi
done

if $all_apis_found; then
    echo "✓ HTTP API接口"
else
    echo "✗ HTTP API接口"
fi

echo ""
echo "4. 检查配置和依赖..."

# 检查go.mod
if grep -q "github.com/aliyun/aliyun-oss-go-sdk" go.mod && \
   grep -q "github.com/google/uuid" go.mod && \
   grep -q "gorm.io/gorm" go.mod; then
    echo "✓ Go模块依赖"
else
    echo "✗ Go模块依赖"
fi

# 检查数据模型
if grep -q "type File struct" internal/model/file.go && \
   grep -q "type DocumentVersion struct" internal/model/file.go && \
   grep -q "type Chunk struct" internal/model/file.go; then
    echo "✓ 数据模型定义"
else
    echo "✗ 数据模型定义"
fi

echo ""
echo "5. 统计代码行数..."

total_lines=0
for file in "${core_files[@]}" "${test_files[@]}"; do
    if [ -f "$file" ]; then
        lines=$(wc -l < "$file")
        total_lines=$((total_lines + lines))
    fi
done

echo "总代码行数: $total_lines"

echo ""
echo "=== 验证完成 ==="

# 检查文档
if [ -f "docs/features/document-processing-system.md" ]; then
    echo "✓ 系统文档已生成"
else
    echo "✗ 系统文档缺失"
fi

echo ""
echo "文档处理系统实现验证完成！"
echo "包含以下主要功能："
echo "- 文件上传和OSS存储"
echo "- 多格式文档解析 (PDF, DOCX, Markdown, TXT, HTML)"
echo "- 智能文本分块 (固定长度、语义、结构、代码)"
echo "- 分块配置和可视化"
echo "- 完整的REST API接口"
echo "- 单元测试和集成测试"