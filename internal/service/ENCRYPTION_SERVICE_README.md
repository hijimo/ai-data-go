# API密钥加密服务

## 概述

`EncryptionService` 提供了安全的API密钥加密、解密和脱敏功能，用于保护模型配置中的敏感信息。

## 特性

- **AES-256-GCM加密**: 使用业界标准的AES-256-GCM算法进行加密
- **随机Nonce**: 每次加密使用随机生成的nonce，确保相同明文产生不同密文
- **Base64编码**: 加密结果使用Base64编码，便于存储和传输
- **密钥脱敏**: 提供密钥脱敏功能，仅显示前4位和后4位

## 配置

### 环境变量

在 `.env` 文件中配置加密密钥：

```bash
# API密钥加密密钥（必须是32字节）
ENCRYPTION_SECRET_KEY=your-32-byte-secret-key-here!!

# 模型配置验证超时时间（秒）
PROVIDER_VALIDATION_TIMEOUT=30
```

**重要提示**：

- 加密密钥必须是32字节（256位）
- 生产环境必须使用强随机密钥
- 密钥一旦设置，不应随意更改，否则已加密的数据将无法解密

### 配置结构

在 `internal/config/config.go` 中定义：

```go
type EncryptionConfig struct {
    SecretKey              string        // API密钥加密密钥（32字节）
    ProviderValidationTimeout time.Duration // 模型配置验证超时时间
}
```

## 使用方法

### 创建服务实例

#### 方式1：从环境变量创建（推荐）

```go
import "genkit-ai-service/internal/service"

// 从环境变量 ENCRYPTION_SECRET_KEY 创建服务
encryptionService, err := service.NewEncryptionServiceFromEnv()
if err != nil {
    log.Fatalf("创建加密服务失败: %v", err)
}
```

#### 方式2：使用自定义密钥创建

```go
import "genkit-ai-service/internal/service"

// 使用32字节密钥创建服务
secretKey := []byte("12345678901234567890123456789012")
encryptionService, err := service.NewEncryptionService(secretKey)
if err != nil {
    log.Fatalf("创建加密服务失败: %v", err)
}
```

### 加密API密钥

```go
// 加密API密钥
plaintext := "sk-test-api-key-12345"
encrypted, err := encryptionService.EncryptAPIKey(plaintext)
if err != nil {
    log.Printf("加密失败: %v", err)
    return
}

// encrypted 是Base64编码的密文，可以安全存储到数据库
fmt.Printf("加密后: %s\n", encrypted)
```

### 解密API密钥

```go
// 解密API密钥
encrypted := "base64-encoded-ciphertext"
plaintext, err := encryptionService.DecryptAPIKey(encrypted)
if err != nil {
    log.Printf("解密失败: %v", err)
    return
}

// plaintext 是原始的API密钥
fmt.Printf("解密后: %s\n", plaintext)
```

### 脱敏API密钥

```go
// 脱敏API密钥（用于显示）
apiKey := "sk-test-api-key-12345"
masked := encryptionService.MaskAPIKey(apiKey)

// 输出: sk-t****2345
fmt.Printf("脱敏后: %s\n", masked)
```

## 在模型配置服务中使用

### 创建模型配置时加密

```go
func (s *ProviderService) Create(ctx context.Context, req CreateProviderRequest) (*ModelConfiguration, error) {
    // 加密API密钥
    encryptedKey, err := s.encryptionService.EncryptAPIKey(req.APIKey)
    if err != nil {
        return nil, errors.NewInternalError("加密API密钥失败")
    }
    
    config := &ModelConfiguration{
        // ... 其他字段
        APIKey: encryptedKey, // 存储加密后的密钥
    }
    
    return s.repo.Create(ctx, config)
}
```

### 查询时脱敏

```go
func (s *ProviderService) Get(ctx context.Context, id uuid.UUID) (*ModelConfiguration, error) {
    config, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 脱敏API密钥
    config.APIKey = s.encryptionService.MaskAPIKey(config.APIKey)
    
    return config, nil
}
```

### 验证时解密

```go
func (s *ProviderService) Validate(ctx context.Context, id uuid.UUID) (*ValidationResult, error) {
    config, err := s.Get(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 解密API密钥用于验证
    apiKey, err := s.encryptionService.DecryptAPIKey(config.APIKey)
    if err != nil {
        return &ValidationResult{
            Valid:   false,
            Message: "解密API密钥失败",
        }, nil
    }
    
    // 使用解密后的密钥进行验证
    return s.validateProvider(ctx, config, apiKey)
}
```

## 安全考虑

### 密钥管理

1. **生成强密钥**: 使用加密安全的随机数生成器生成32字节密钥
2. **安全存储**: 密钥应存储在环境变量或密钥管理服务中，不应硬编码
3. **密钥轮换**: 定期更换加密密钥（需要重新加密所有数据）
4. **访问控制**: 限制对加密密钥的访问权限

### 数据保护

1. **传输安全**: 仅通过HTTPS传输加密数据
2. **存储安全**: 加密数据存储在数据库中
3. **日志安全**: 不在日志中记录明文密钥或加密密钥
4. **错误处理**: 解密失败时不泄露敏感信息

### 最佳实践

1. **最小权限原则**: 仅在需要时解密密钥
2. **短生命周期**: 解密后的密钥应尽快使用并清除
3. **审计日志**: 记录所有加密/解密操作（不包含密钥内容）
4. **定期审查**: 定期审查加密实现和密钥管理流程

## 错误处理

### 常见错误

1. **密钥长度错误**

   ```
   错误: 加密密钥必须是32字节（AES-256）
   解决: 确保 ENCRYPTION_SECRET_KEY 至少32个字符
   ```

2. **环境变量未设置**

   ```
   错误: 环境变量 ENCRYPTION_SECRET_KEY 未设置
   解决: 在 .env 文件中设置 ENCRYPTION_SECRET_KEY
   ```

3. **解密失败**

   ```
   错误: 解密失败: cipher: message authentication failed
   解决: 检查是否使用了正确的加密密钥，或数据是否被篡改
   ```

4. **Base64解码失败**

   ```
   错误: Base64解码失败
   解决: 确保存储的是有效的Base64编码字符串
   ```

## 性能考虑

- **加密性能**: AES-256-GCM加密速度快，适合实时加密
- **内存使用**: 加密过程内存占用小
- **并发安全**: 服务实例是并发安全的，可以在多个goroutine中使用

## 测试

运行加密服务测试：

```bash
go test -v ./internal/service/encryption_service_test.go ./internal/service/encryption_service.go
```

测试覆盖：

- ✅ 创建服务实例
- ✅ 从环境变量创建服务
- ✅ 加密API密钥
- ✅ 解密API密钥
- ✅ 加密解密循环测试
- ✅ 脱敏API密钥
- ✅ 不同密钥的隔离性
- ✅ 错误处理

## 示例代码

完整的使用示例：

```go
package main

import (
    "fmt"
    "log"
    "genkit-ai-service/internal/service"
)

func main() {
    // 创建加密服务
    encryptionService, err := service.NewEncryptionServiceFromEnv()
    if err != nil {
        log.Fatalf("创建加密服务失败: %v", err)
    }
    
    // 原始API密钥
    originalKey := "sk-test-api-key-12345"
    fmt.Printf("原始密钥: %s\n", originalKey)
    
    // 加密
    encrypted, err := encryptionService.EncryptAPIKey(originalKey)
    if err != nil {
        log.Fatalf("加密失败: %v", err)
    }
    fmt.Printf("加密后: %s\n", encrypted)
    
    // 解密
    decrypted, err := encryptionService.DecryptAPIKey(encrypted)
    if err != nil {
        log.Fatalf("解密失败: %v", err)
    }
    fmt.Printf("解密后: %s\n", decrypted)
    
    // 脱敏
    masked := encryptionService.MaskAPIKey(originalKey)
    fmt.Printf("脱敏后: %s\n", masked)
    
    // 验证
    if decrypted == originalKey {
        fmt.Println("✅ 加密解密成功！")
    }
}
```

## 相关文档

- [模型配置模块设计文档](../../.kiro/specs/model-configuration/design.md)
- [模型配置模块需求文档](../../.kiro/specs/model-configuration/requirements.md)
- [模型配置模块实现计划](../../.kiro/specs/model-configuration/tasks.md)
