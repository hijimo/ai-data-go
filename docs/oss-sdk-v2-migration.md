# 阿里云OSS Go SDK V2迁移指南

## 概述

本文档描述了从阿里云OSS Go SDK V1迁移到V2的过程和主要变化。

## 主要变化

### 1. 依赖包更新

**V1 (旧版本):**

```go
import "github.com/aliyun/aliyun-oss-go-sdk/oss"
```

**V2 (新版本):**

```go
import (
    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)
```

### 2. 客户端初始化

**V1 (旧版本):**

```go
client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
bucket, err := client.Bucket(bucketName)
```

**V2 (新版本):**

```go
cfg := oss.LoadDefaultConfig().
    WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret)).
    WithRegion(region)

client := oss.NewClient(cfg)
```

### 3. API调用方式

**V1 上传文件:**

```go
err = bucket.PutObject(objectKey, reader)
```

**V2 上传文件:**

```go
request := &oss.PutObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectKey),
    Body:   reader,
}
_, err = client.PutObject(ctx, request)
```

**V1 删除文件:**

```go
err = bucket.DeleteObject(objectKey)
```

**V2 删除文件:**

```go
request := &oss.DeleteObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectKey),
}
_, err = client.DeleteObject(ctx, request)
```

**V1 生成预签名URL:**

```go
url, err := bucket.SignURL(objectKey, oss.HTTPGet, expiry)
```

**V2 生成预签名URL:**

```go
request := &oss.GetObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectKey),
}
presigner := oss.NewPresigner(client)
result, err := presigner.PresignGetObject(ctx, request, func(options *oss.PresignOptions) {
    options.Expiry = expiry
})
url = result.URL
```

**V1 检查文件存在:**

```go
exists, err := bucket.IsObjectExist(objectKey)
```

**V2 检查文件存在:**

```go
request := &oss.HeadObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectKey),
}
_, err := client.HeadObject(ctx, request)
// 通过错误判断文件是否存在
```

### 4. 环境变量配置

V2 SDK支持标准的环境变量名称：

```bash
# 推荐使用环境变量方式配置访问凭证
OSS_ACCESS_KEY_ID=your_access_key_id
OSS_ACCESS_KEY_SECRET=your_access_key_secret
```

**使用环境变量创建客户端:**

```go
cfg := oss.LoadDefaultConfig().
    WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
    WithRegion(region)

client := oss.NewClient(cfg)
```

### 5. 错误处理

V2 SDK提供了更详细的错误信息，包括：

- HTTP状态码
- 错误消息
- 请求ID
- EC错误码（用于问题诊断）

### 6. 上下文支持

V2 SDK所有API都支持context.Context，提供更好的超时控制和取消机制。

## 迁移步骤

### 1. 更新依赖

在`go.mod`中更新依赖：

```go
// 移除旧版本
// github.com/aliyun/aliyun-oss-go-sdk v2.2.9+incompatible

// 添加新版本
github.com/aliyun/alibabacloud-oss-go-sdk-v2 v1.0.2
```

### 2. 更新导入

```go
// 旧版本
import "github.com/aliyun/aliyun-oss-go-sdk/oss"

// 新版本
import (
    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)
```

### 3. 更新客户端初始化代码

参考上面的API调用方式示例。

### 4. 配置环境变量

```bash
# 设置访问凭证
export OSS_ACCESS_KEY_ID="your_access_key_id"
export OSS_ACCESS_KEY_SECRET="your_access_key_secret"

# 或者在.env文件中配置
OSS_ACCESS_KEY_ID=your_access_key_id
OSS_ACCESS_KEY_SECRET=your_access_key_secret
OSS_BUCKET_NAME=your_bucket_name
OSS_REGION=cn-hangzhou
```

### 5. 测试验证

运行测试确保所有功能正常工作：

```bash
go test ./internal/storage -v
go test ./internal/service -v -run TestFileService
```

## V2 SDK的优势

1. **更好的性能**: 优化的HTTP客户端和连接池管理
2. **更强的安全性**: 默认使用V4签名，提供更高的安全性
3. **更好的错误处理**: 详细的错误信息和EC错误码
4. **标准化**: 遵循AWS SDK的设计模式
5. **上下文支持**: 所有API都支持context.Context
6. **更好的测试支持**: 内置Mock和测试工具

## 注意事项

1. **向后兼容性**: V2 SDK与V1不兼容，需要完全迁移
2. **配置方式**: 推荐使用环境变量配置访问凭证
3. **错误处理**: 需要适配新的错误处理方式
4. **API变化**: 所有API都需要传入context.Context参数

## 参考资源

- [阿里云OSS Go SDK V2官方文档](https://help.aliyun.com/zh/oss/developer-reference/manual-for-go-sdk-v2/)
- [SDK V2 GitHub仓库](https://github.com/aliyun/alibabacloud-oss-go-sdk-v2)
- [迁移指南](https://help.aliyun.com/zh/oss/developer-reference/manual-for-go-sdk-v2/#51393d41f4awh)
