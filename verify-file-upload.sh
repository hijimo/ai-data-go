#!/bin/bash

echo "验证文件上传功能实现..."

# 检查必要的文件是否存在
files=(
    "internal/storage/oss.go"
    "internal/repository/file.go"
    "internal/service/file.go"
    "internal/handler/file.go"
    "internal/service/file_test.go"
    "internal/handler/file_test.go"
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

# 检查OSS客户端接口
if grep -q "type OSSClient interface" internal/storage/oss.go; then
    echo "✓ OSS客户端接口已定义"
else
    echo "✗ OSS客户端接口未定义"
fi

# 检查文件上传方法
if grep -q "UploadFile.*multipart.File" internal/storage/oss.go; then
    echo "✓ 文件上传方法已实现"
else
    echo "✗ 文件上传方法未实现"
fi

# 检查SHA256计算
if grep -q "sha256.Sum256" internal/storage/oss.go; then
    echo "✓ SHA256哈希计算已实现"
else
    echo "✗ SHA256哈希计算未实现"
fi

# 检查文件去重
if grep -q "GetBySHA256" internal/repository/file.go; then
    echo "✓ 文件去重功能已实现"
else
    echo "✗ 文件去重功能未实现"
fi

# 检查文件格式验证
if grep -q "validateFileFormat" internal/service/file.go; then
    echo "✓ 文件格式验证已实现"
else
    echo "✗ 文件格式验证未实现"
fi

# 检查HTTP处理器
if grep -q "UploadFile.*gin.Context" internal/handler/file.go; then
    echo "✓ HTTP上传处理器已实现"
else
    echo "✗ HTTP上传处理器未实现"
fi

# 检查测试文件
if grep -q "TestFileService_UploadFile" internal/service/file_test.go; then
    echo "✓ 服务层测试已编写"
else
    echo "✗ 服务层测试未编写"
fi

if grep -q "TestFileHandler_UploadFile" internal/handler/file_test.go; then
    echo "✓ 处理器测试已编写"
else
    echo "✗ 处理器测试未编写"
fi

echo ""
echo "文件上传功能实现验证完成!"