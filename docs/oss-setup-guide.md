# 阿里云OSS配置指南

## 概述

本指南介绍如何配置和使用阿里云对象存储服务(OSS)与AI知识管理平台。我们使用阿里云OSS Go SDK V2来实现文件存储功能。

## 前置条件

1. 阿里云账号
2. 已创建的OSS存储桶(Bucket)
3. RAM用户访问密钥(AccessKey)

## 配置步骤

### 1. 创建RAM用户和访问密钥

1. 登录[阿里云RAM控制台](https://ram.console.aliyun.com/users/create)
2. 创建RAM用户，选择"使用永久AccessKey访问"
3. 为RAM用户授予`AliyunOSSFullAccess`权限
4. 保存AccessKey ID和AccessKey Secret

### 2. 创建OSS存储桶

1. 登录[阿里云OSS控制台](https://oss.console.aliyun.com/)
2. 创建存储桶，选择合适的地域
3. 记录存储桶名称和地域信息

### 3. 配置环境变量

#### 方式一：使用.env文件（推荐）

复制`.env.example`文件为`.env`：

```bash
cp .env.example .env
```

编辑`.env`文件，填入你的OSS配置：

```bash
# OSS配置
OSS_ACCESS_KEY_ID=your_access_key_id
OSS_ACCESS_KEY_SECRET=your_access_key_secret
OSS_BUCKET_NAME=your_bucket_name
OSS_REGION=cn-hangzhou
OSS_ENDPOINT=
```

#### 方式二：设置系统环境变量

**Linux/macOS (Bash):**

```bash
export OSS_ACCESS_KEY_ID="your_access_key_id"
export OSS_ACCESS_KEY_SECRET="your_access_key_secret"
export OSS_BUCKET_NAME="your_bucket_name"
export OSS_REGION="cn-hangzhou"
```

**Linux/macOS (Zsh):**

```bash
echo 'export OSS_ACCESS_KEY_ID="your_access_key_id"' >> ~/.zshrc
echo 'export OSS_ACCESS_KEY_SECRET="your_access_key_secret"' >> ~/.zshrc
echo 'export OSS_BUCKET_NAME="your_bucket_name"' >> ~/.zshrc
echo 'export OSS_REGION="cn-hangzhou"' >> ~/.zshrc
source ~/.zshrc
```

**Windows (PowerShell):**

```powershell
[Environment]::SetEnvironmentVariable("OSS_ACCESS_KEY_ID", "your_access_key_id", [EnvironmentVariableTarget]::User)
[Environment]::SetEnvironmentVariable("OSS_ACCESS_KEY_SECRET", "your_access_key_secret", [EnvironmentVariableTarget]::User)
[Environment]::SetEnvironmentVariable("OSS_BUCKET_NAME", "your_bucket_name", [EnvironmentVariableTarget]::User)
[Environment]::SetEnvironmentVariable("OSS_REGION", "cn-hangzhou", [EnvironmentVariableTarget]::User)
```

### 4. 验证配置

运行以下命令验证环境变量是否设置正确：

```bash
echo $OSS_ACCESS_KEY_ID
echo $OSS_ACCESS_KEY_SECRET
echo $OSS_BUCKET_NAME
echo $OSS_REGION
```

## 地域和访问域名

### 常用地域

| 地域名称 | Region ID | 公网Endpoint |
|---------|-----------|-------------|
| 华东1（杭州） | cn-hangzhou | oss-cn-hangzhou.aliyuncs.com |
| 华东2（上海） | cn-shanghai | oss-cn-shanghai.aliyuncs.com |
| 华北1（青岛） | cn-qingdao | oss-cn-qingdao.aliyuncs.com |
| 华北2（北京） | cn-beijing | oss-cn-beijing.aliyuncs.com |
| 华南1（深圳） | cn-shenzhen | oss-cn-shenzhen.aliyuncs.com |

完整的地域列表请参考：[OSS地域和访问域名](https://help.aliyun.com/zh/oss/regions-and-endpoints)

### 内网访问

如果你的应用部署在阿里云ECS上，可以使用内网域名来降低流量成本：

```bash
OSS_ENDPOINT=https://oss-cn-hangzhou-internal.aliyuncs.com
```

## 使用示例

### 基本用法

```go
package main

import (
    "context"
    "log"
    
    "ai-knowledge-platform/internal/storage"
)

func main() {
    // 从环境变量创建OSS客户端
    client, err := storage.NewOSSClientFromEnv("your-bucket-name", "cn-hangzhou")
    if err != nil {
        log.Fatal(err)
    }
    
    // 检查文件是否存在
    exists, err := client.CheckFileExists(context.Background(), "test-file.txt")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("文件存在: %v", exists)
}
```

### 自定义配置

```go
config := &storage.OSSConfig{
    AccessKeyID:     "your_access_key_id",
    AccessKeySecret: "your_access_key_secret",
    BucketName:      "your_bucket_name",
    Region:          "cn-hangzhou",
    Endpoint:        "", // 可选，留空使用默认域名
}

client, err := storage.NewOSSClient(config)
```

## 测试

### 单元测试

运行Mock测试：

```bash
go test ./internal/storage -v
```

### 集成测试

设置环境变量后运行集成测试：

```bash
export RUN_INTEGRATION_TESTS=true
go test ./internal/storage -v -run Integration
```

## 安全最佳实践

1. **使用RAM用户**: 不要使用主账号的AccessKey
2. **最小权限原则**: 只授予必要的OSS权限
3. **定期轮换**: 定期更换AccessKey
4. **环境变量**: 使用环境变量而不是硬编码凭证
5. **网络安全**: 在生产环境中使用HTTPS
6. **访问控制**: 配置适当的存储桶策略和ACL

## 故障排除

### 常见错误

1. **InvalidAccessKeyId**: 检查AccessKey ID是否正确
2. **SignatureDoesNotMatch**: 检查AccessKey Secret是否正确
3. **NoSuchBucket**: 检查存储桶名称和地域是否正确
4. **AccessDenied**: 检查RAM用户权限是否足够

### 调试方法

1. 检查环境变量是否设置正确
2. 验证RAM用户权限
3. 确认存储桶和地域配置
4. 查看详细的错误信息和EC错误码

### 获取帮助

- [阿里云OSS官方文档](https://help.aliyun.com/zh/oss/)
- [Go SDK V2文档](https://help.aliyun.com/zh/oss/developer-reference/manual-for-go-sdk-v2/)
- [问题自助诊断](https://api.aliyun.com/troubleshoot)

## 性能优化

1. **选择合适的地域**: 选择离用户最近的地域
2. **使用内网域名**: ECS实例使用内网域名访问
3. **启用传输加速**: 对于跨地域访问，可以启用传输加速
4. **合理设置超时**: 根据网络情况调整连接和读写超时时间
5. **使用CDN**: 对于频繁访问的文件，可以配置CDN加速

## 监控和日志

1. **启用访问日志**: 在OSS控制台启用访问日志功能
2. **监控指标**: 关注请求量、错误率、延迟等指标
3. **设置报警**: 为异常情况设置报警规则
4. **日志分析**: 定期分析访问日志，优化使用模式
