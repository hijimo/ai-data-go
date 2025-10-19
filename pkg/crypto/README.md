# Crypto 包

密码安全工具包，提供密码哈希、验证和强度检查功能。

## 功能特性

- **密码哈希**: 使用 bcrypt 算法（cost=12）对密码进行安全哈希
- **密码验证**: 验证明文密码与哈希值是否匹配
- **强度验证**: 检查密码是否满足安全要求
- **强度评估**: 评估密码强度等级（弱/中/强）
- **常见密码检测**: 检测是否使用了常见的弱密码

## 安装

```bash
go get golang.org/x/crypto/bcrypt
```

## 使用示例

### 密码哈希

```go
import "genkit-ai-service/pkg/crypto"

// 对密码进行哈希
password := "MySecurePassword123!"
hash, err := crypto.HashPassword(password)
if err != nil {
    // 处理错误（密码太短、太长等）
    log.Fatal(err)
}

// 存储 hash 到数据库
```

### 密码验证

```go
// 从数据库获取存储的哈希值
storedHash := "..." 

// 验证用户输入的密码
err := crypto.VerifyPassword(storedHash, userInputPassword)
if err != nil {
    if err == crypto.ErrPasswordMismatch {
        // 密码不匹配
        return errors.New("用户名或密码错误")
    }
    // 其他错误
    return err
}

// 密码验证成功
```

### 密码强度验证

```go
// 在用户注册或修改密码时验证密码强度
password := "UserNewPassword123!"

// 方式1: 基本强度验证（要求至少3种字符类型）
err := crypto.ValidatePasswordStrength(password)
if err != nil {
    // 密码强度不足
    return err
}

// 方式2: 综合验证（包含强度验证和常见密码检查）
err = crypto.ValidatePassword(password)
if err != nil {
    // 密码不符合要求
    return err
}
```

### 密码强度评估

```go
// 获取密码强度等级
strength := crypto.GetPasswordStrength(password)

switch strength {
case crypto.PasswordStrengthWeak:
    fmt.Println("密码强度：弱")
case crypto.PasswordStrengthMedium:
    fmt.Println("密码强度：中等")
case crypto.PasswordStrengthStrong:
    fmt.Println("密码强度：强")
}
```

### 常见密码检测

```go
// 检查是否为常见弱密码
if crypto.IsCommonPassword(password) {
    return errors.New("不能使用常见的弱密码")
}
```

## 密码要求

### 长度要求

- 最小长度: 8 个字符
- 最大长度: 128 个字符

### 强度要求

密码必须包含至少 **3 种** 以下字符类型：

1. 大写字母 (A-Z)
2. 小写字母 (a-z)
3. 数字 (0-9)
4. 特殊字符 (如 !@#$%^&*)

### 强度等级判定

- **强密码**:
  - 长度 ≥ 12 且包含 4 种字符类型，或
  - 长度 ≥ 10 且包含 3 种字符类型

- **中等密码**:
  - 长度 ≥ 8 且包含 3 种字符类型

- **弱密码**:
  - 不满足以上条件

### 禁止使用的常见密码

系统会拒绝以下常见弱密码：

- password
- 12345678
- 123456789
- qwerty
- abc123
- admin
- 等等...

## 错误处理

包提供了以下预定义错误：

```go
var (
    ErrPasswordTooShort   = errors.New("密码长度不能少于8个字符")
    ErrPasswordTooLong    = errors.New("密码长度不能超过128个字符")
    ErrPasswordTooWeak    = errors.New("密码强度不足，必须包含大写字母、小写字母、数字和特殊字符")
    ErrPasswordMismatch   = errors.New("密码不匹配")
)
```

## 安全建议

1. **永远不要存储明文密码**: 始终使用 `HashPassword` 对密码进行哈希后再存储
2. **使用 HTTPS**: 确保密码在传输过程中加密
3. **实施速率限制**: 防止暴力破解攻击
4. **启用账户锁定**: 多次登录失败后临时锁定账户
5. **定期更新密码**: 建议用户定期更换密码
6. **使用强密码策略**: 在用户注册和修改密码时强制执行密码强度要求

## 性能考虑

bcrypt 算法设计为计算密集型，这是有意为之的安全特性：

- **哈希操作**: 约 200-400ms（cost=12）
- **验证操作**: 约 200-400ms（cost=12）

这种延迟可以有效防止暴力破解攻击。在高并发场景下，建议：

1. 使用异步处理
2. 实施请求队列
3. 考虑使用缓存（谨慎使用）

## 测试

运行测试：

```bash
go test -v ./pkg/crypto
```

运行基准测试：

```bash
go test -bench=. ./pkg/crypto
```

## 常量配置

```go
const (
    BcryptCost        = 12  // bcrypt cost factor
    MinPasswordLength = 8   // 最小密码长度
    MaxPasswordLength = 128 // 最大密码长度
)
```

## 相关需求

本包实现了以下需求：

- 需求 9.1: 密码哈希存储
- 需求 9.2: 密码验证
- 需求 9.3: 密码强度验证
- 需求 9.4: 不存储明文密码
- 需求 9.5: 密码修改验证

## 许可证

本项目的一部分，遵循项目整体许可证。
