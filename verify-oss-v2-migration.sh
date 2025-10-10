#!/bin/bash

echo "验证阿里云OSS Go SDK V2迁移..."

echo ""
echo "检查依赖更新:"

# 检查go.mod中的依赖
if grep -q "github.com/aliyun/alibabacloud-oss-go-sdk-v2" go.mod; then
    echo "✓ OSS Go SDK V2依赖已添加"
else
    echo "✗ OSS Go SDK V2依赖未添加"
fi

if grep -q "github.com/aliyun/aliyun-oss-go-sdk" go.mod; then
    echo "⚠ 发现旧版本OSS SDK依赖，建议移除"
else
    echo "✓ 旧版本OSS SDK依赖已移除"
fi

echo ""
echo "检查代码更新:"

# 检查导入更新
if grep -q "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss" internal/storage/oss.go; then
    echo "✓ OSS V2 SDK导入已更新"
else
    echo "✗ OSS V2 SDK导入未更新"
fi

if grep -q "credentials" internal/storage/oss.go; then
    echo "✓ 凭证提供者导入已添加"
else
    echo "✗ 凭证提供者导入未添加"
fi

# 检查API调用更新
if grep -q "PutObjectRequest" internal/storage/oss.go; then
    echo "✓ PutObject API已更新为V2格式"
else
    echo "✗ PutObject API未更新"
fi

if grep -q "DeleteObjectRequest" internal/storage/oss.go; then
    echo "✓ DeleteObject API已更新为V2格式"
else
    echo "✗ DeleteObject API未更新"
fi

if grep -q "HeadObjectRequest" internal/storage/oss.go; then
    echo "✓ HeadObject API已更新为V2格式"
else
    echo "✗ HeadObject API未更新"
fi

if grep -q "PresignGetObject" internal/storage/oss.go; then
    echo "✓ 预签名URL API已更新为V2格式"
else
    echo "✗ 预签名URL API未更新"
fi

# 检查环境变量支持
if grep -q "NewEnvironmentVariableCredentialsProvider" internal/storage/oss.go; then
    echo "✓ 环境变量凭证提供者已添加"
else
    echo "✗ 环境变量凭证提供者未添加"
fi

echo ""
echo "检查配置文件:"

# 检查环境变量示例文件
if [ -f ".env.example" ]; then
    echo "✓ 环境变量示例文件已创建"
    if grep -q "OSS_ACCESS_KEY_ID" .env.example; then
        echo "✓ OSS环境变量配置已添加"
    else
        echo "✗ OSS环境变量配置未添加"
    fi
else
    echo "✗ 环境变量示例文件未创建"
fi

echo ""
echo "检查文档更新:"

# 检查迁移文档
if [ -f "docs/oss-sdk-v2-migration.md" ]; then
    echo "✓ OSS SDK V2迁移文档已创建"
else
    echo "✗ OSS SDK V2迁移文档未创建"
fi

# 检查主文档更新
if grep -q "SDK V2" docs/features/document-processing-system.md; then
    echo "✓ 主文档已更新SDK V2信息"
else
    echo "✗ 主文档未更新SDK V2信息"
fi

echo ""
echo "OSS Go SDK V2迁移验证完成!"

echo ""
echo "下一步操作建议:"
echo "1. 运行 'go mod tidy' 更新依赖"
echo "2. 设置环境变量 OSS_ACCESS_KEY_ID 和 OSS_ACCESS_KEY_SECRET"
echo "3. 运行测试验证功能: 'go test ./internal/storage -v'"
echo "4. 查看迁移文档: docs/oss-sdk-v2-migration.md"