#!/bin/bash

# 修复 Swagger 文档中的类型名称，去掉包路径前缀

echo "修复 Swagger 文档中的类型名称..."

# 定义要替换的前缀
PREFIX1="genkit-ai-service_internal_model\\."
PREFIX2="internal_api_handler\\."
PREFIX3="genkit-ai-service_internal_service_health\\."
PREFIX4="model\\."
PREFIX5="handler\\."

# 替换 swagger.json 中的类型名称
if [ -f "docs/swagger.json" ]; then
    sed -i.bak "s/${PREFIX1}//g" docs/swagger.json
    sed -i.bak "s/${PREFIX2}//g" docs/swagger.json
    sed -i.bak "s/${PREFIX3}//g" docs/swagger.json
    sed -i.bak "s/${PREFIX4}//g" docs/swagger.json
    sed -i.bak "s/${PREFIX5}//g" docs/swagger.json
    rm -f docs/swagger.json.bak
    echo "✅ 已修复 swagger.json"
fi

# 替换 swagger.yaml 中的类型名称
if [ -f "docs/swagger.yaml" ]; then
    sed -i.bak "s/${PREFIX1}//g" docs/swagger.yaml
    sed -i.bak "s/${PREFIX2}//g" docs/swagger.yaml
    sed -i.bak "s/${PREFIX3}//g" docs/swagger.yaml
    sed -i.bak "s/${PREFIX4}//g" docs/swagger.yaml
    sed -i.bak "s/${PREFIX5}//g" docs/swagger.yaml
    rm -f docs/swagger.yaml.bak
    echo "✅ 已修复 swagger.yaml"
fi

# 替换 docs.go 中的类型名称
if [ -f "docs/docs.go" ]; then
    sed -i.bak "s/${PREFIX1}//g" docs/docs.go
    sed -i.bak "s/${PREFIX2}//g" docs/docs.go
    sed -i.bak "s/${PREFIX3}//g" docs/docs.go
    sed -i.bak "s/${PREFIX4}//g" docs/docs.go
    sed -i.bak "s/${PREFIX5}//g" docs/docs.go
    rm -f docs/docs.go.bak
    echo "✅ 已修复 docs.go"
fi

echo "✅ Swagger 文档类型名称修复完成"
