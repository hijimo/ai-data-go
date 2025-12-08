#!/bin/bash

# 测试环境配置脚本
# 用于准备多提供商支持的测试环境
# 用法: ./scripts/setup_test_env.sh

set -e

echo "========================================="
echo "测试环境配置向导"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查 .env.test 文件是否存在
if [ -f ".env.test" ]; then
    echo -e "${YELLOW}⚠️  .env.test 文件已存在${NC}"
    read -p "是否覆盖现有配置？(y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "取消配置"
        exit 0
    fi
fi

# 复制示例配置文件
echo -e "${BLUE}📋 复制配置文件模板...${NC}"
cp .env.test.example .env.test
echo -e "${GREEN}✓ 配置文件已创建: .env.test${NC}"
echo ""

# 配置向导
echo "========================================="
echo "配置向导"
echo "========================================="
echo ""
echo "请按照提示输入配置信息（按 Enter 使用默认值）"
echo ""

# 数据库配置
echo -e "${BLUE}--- 数据库配置 ---${NC}"
read -p "数据库主机 [localhost]: " db_host
db_host=${db_host:-localhost}
sed -i.bak "s/DB_HOST=localhost/DB_HOST=$db_host/" .env.test

read -p "数据库端口 [5432]: " db_port
db_port=${db_port:-5432}
sed -i.bak "s/DB_PORT=5432/DB_PORT=$db_port/" .env.test

read -p "数据库名称 [ai_service_test]: " db_name
db_name=${db_name:-ai_service_test}
sed -i.bak "s/DB_NAME=ai_service_test/DB_NAME=$db_name/" .env.test

read -p "数据库用户 [postgres]: " db_user
db_user=${db_user:-postgres}
sed -i.bak "s/DB_USER=postgres/DB_USER=$db_user/" .env.test

read -sp "数据库密码: " db_password
echo
if [ -n "$db_password" ]; then
    sed -i.bak "s/DB_PASSWORD=your_test_db_password/DB_PASSWORD=$db_password/" .env.test
fi
echo ""

# Redis 配置
echo -e "${BLUE}--- Redis 配置 ---${NC}"
read -p "Redis 主机 [localhost]: " redis_host
redis_host=${redis_host:-localhost}
sed -i.bak "s/REDIS_HOST=localhost/REDIS_HOST=$redis_host/" .env.test

read -p "Redis 端口 [6379]: " redis_port
redis_port=${redis_port:-6379}
sed -i.bak "s/REDIS_PORT=6379/REDIS_PORT=$redis_port/" .env.test

read -sp "Redis 密码（可选）: " redis_password
echo
if [ -n "$redis_password" ]; then
    sed -i.bak "s/REDIS_PASSWORD=your_test_redis_password/REDIS_PASSWORD=$redis_password/" .env.test
fi
echo ""

# Google AI 配置
echo -e "${BLUE}--- Google AI (Gemini) 配置 ---${NC}"
read -sp "Google API Key: " google_api_key
echo
if [ -n "$google_api_key" ]; then
    sed -i.bak "s/GOOGLE_API_KEY=your_google_api_key_here/GOOGLE_API_KEY=$google_api_key/" .env.test
    sed -i.bak "s/GENKIT_API_KEY=your_google_api_key_here/GENKIT_API_KEY=$google_api_key/" .env.test
    echo -e "${GREEN}✓ Google AI 配置已设置${NC}"
else
    echo -e "${YELLOW}⚠️  跳过 Google AI 配置${NC}"
fi
echo ""

# Azure OpenAI 配置
echo -e "${BLUE}--- Azure OpenAI 配置 ---${NC}"
read -p "是否配置 Azure OpenAI？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -sp "Azure OpenAI API Key: " azure_api_key
    echo
    if [ -n "$azure_api_key" ]; then
        sed -i.bak "s/AZURE_OPENAI_API_KEY=your_azure_api_key_here/AZURE_OPENAI_API_KEY=$azure_api_key/" .env.test
    fi
    
    read -p "Azure OpenAI Endpoint: " azure_endpoint
    if [ -n "$azure_endpoint" ]; then
        sed -i.bak "s|AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com|AZURE_OPENAI_ENDPOINT=$azure_endpoint|" .env.test
    fi
    
    read -p "Azure OpenAI Deployment: " azure_deployment
    if [ -n "$azure_deployment" ]; then
        sed -i.bak "s/AZURE_OPENAI_DEPLOYMENT=gpt-4/AZURE_OPENAI_DEPLOYMENT=$azure_deployment/" .env.test
    fi
    
    echo -e "${GREEN}✓ Azure OpenAI 配置已设置${NC}"
else
    echo -e "${YELLOW}⚠️  跳过 Azure OpenAI 配置${NC}"
fi
echo ""

# 百炼配置
echo -e "${BLUE}--- 阿里云百炼配置 ---${NC}"
read -p "是否配置阿里云百炼？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -sp "百炼 API Key: " bailian_api_key
    echo
    if [ -n "$bailian_api_key" ]; then
        sed -i.bak "s/BAILIAN_API_KEY=your_bailian_api_key_here/BAILIAN_API_KEY=$bailian_api_key/" .env.test
    fi
    
    read -p "百炼 Endpoint [https://dashscope.aliyuncs.com]: " bailian_endpoint
    bailian_endpoint=${bailian_endpoint:-https://dashscope.aliyuncs.com}
    sed -i.bak "s|BAILIAN_ENDPOINT=https://dashscope.aliyuncs.com|BAILIAN_ENDPOINT=$bailian_endpoint|" .env.test
    
    echo -e "${GREEN}✓ 百炼配置已设置${NC}"
else
    echo -e "${YELLOW}⚠️  跳过百炼配置${NC}"
fi
echo ""

# 清理备份文件
rm -f .env.test.bak

echo "========================================="
echo -e "${GREEN}✓ 配置完成！${NC}"
echo "========================================="
echo ""
echo "配置文件已保存到: .env.test"
echo ""
echo "下一步："
echo "1. 检查并编辑 .env.test 文件，确认所有配置正确"
echo "2. 运行数据库迁移: make migrate"
echo "3. 初始化测试数据: ./scripts/init_test_data.sh"
echo "4. 启动服务: make run-test"
echo ""
