# 邮箱验证功能使用示例

本文档提供邮箱验证功能的实际使用示例。

## 场景1：用户注册并验证邮箱

### 步骤1：用户注册

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "password": "SecurePass123",
    "displayName": "张三"
  }'
```

**响应示例：**

```json
{
  "code": 201,
  "message": "注册成功",
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "emailVerified": false,
    "displayName": "张三",
    "isActive": true,
    "createdAt": "2025-01-15T10:30:00Z"
  }
}
```

**注意：** 注册成功后，系统会自动发送验证邮件到用户邮箱。在开发环境中，邮件内容会输出到控制台。

### 步骤2：查看控制台输出的验证邮件

在服务器控制台中，你会看到类似以下的输出：

```
=== 发送邮件 ===
收件人: user@example.com
主题: 验证您的邮箱地址
内容:
<html>
<body>
    <h2>邮箱验证</h2>
    <p>请点击下面的链接验证您的邮箱地址：</p>
    <p><a href="https://your-domain.com/verify-email?token=770e8400-e29b-41d4-a716-446655440002">验证邮箱</a></p>
    <p>或者复制以下链接到浏览器：</p>
    <p>https://your-domain.com/verify-email?token=770e8400-e29b-41d4-a716-446655440002</p>
    <p>此链接将在 24 小时后过期。</p>
    <p>如果您没有注册账户，请忽略此邮件。</p>
</body>
</html>
===============
```

### 步骤3：验证邮箱

从邮件中复制验证令牌（token），然后调用验证接口：

```bash
curl -X POST http://localhost:8080/api/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{
    "token": "770e8400-e29b-41d4-a716-446655440002"
  }'
```

**成功响应：**

```json
{
  "code": 200,
  "message": "邮箱验证成功",
  "data": null
}
```

### 步骤4：验证用户邮箱状态

登录后查看用户信息，确认 `emailVerified` 字段已更新为 `true`：

```bash
curl -X GET http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**响应示例：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "emailVerified": true,
    "displayName": "张三",
    "isActive": true
  }
}
```

## 场景2：重新发送验证邮件

如果用户没有收到验证邮件或验证链接已过期，可以重新发送验证邮件。

```bash
curl -X POST http://localhost:8080/api/v1/auth/resend-verification \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "userId": "660e8400-e29b-41d4-a716-446655440001"
  }'
```

**成功响应：**

```json
{
  "code": 200,
  "message": "验证邮件已发送",
  "data": null
}
```

## 场景3：验证失败的情况

### 使用过期的令牌

```bash
curl -X POST http://localhost:8080/api/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{
    "token": "expired-token-12345"
  }'
```

**错误响应：**

```json
{
  "code": 400,
  "message": "验证令牌无效或已过期",
  "data": null
}
```

### 使用已使用的令牌

```bash
curl -X POST http://localhost:8080/api/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{
    "token": "770e8400-e29b-41d4-a716-446655440002"
  }'
```

**错误响应：**

```json
{
  "code": 400,
  "message": "验证令牌已使用",
  "data": null
}
```

### 重新发送验证邮件给已验证的用户

```bash
curl -X POST http://localhost:8080/api/v1/auth/resend-verification \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "userId": "660e8400-e29b-41d4-a716-446655440001"
  }'
```

**错误响应：**

```json
{
  "code": 400,
  "message": "邮箱已验证",
  "data": null
}
```

## 前端集成示例

### React 示例

```typescript
// 验证邮箱
async function verifyEmail(token: string) {
  try {
    const response = await fetch('http://localhost:8080/api/v1/auth/verify-email', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ token }),
    });

    const data = await response.json();

    if (response.ok) {
      console.log('邮箱验证成功:', data.message);
      // 跳转到登录页面或显示成功消息
      window.location.href = '/login?verified=true';
    } else {
      console.error('邮箱验证失败:', data.message);
      // 显示错误消息
      alert(data.message);
    }
  } catch (error) {
    console.error('请求失败:', error);
    alert('网络错误，请稍后重试');
  }
}

// 重新发送验证邮件
async function resendVerification(tenantId: string, userId: string) {
  try {
    const response = await fetch('http://localhost:8080/api/v1/auth/resend-verification', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ tenantId, userId }),
    });

    const data = await response.json();

    if (response.ok) {
      console.log('验证邮件已发送:', data.message);
      alert('验证邮件已发送，请查收邮箱');
    } else {
      console.error('发送失败:', data.message);
      alert(data.message);
    }
  } catch (error) {
    console.error('请求失败:', error);
    alert('网络错误，请稍后重试');
  }
}

// 从URL参数中获取token并验证
function handleEmailVerification() {
  const urlParams = new URLSearchParams(window.location.search);
  const token = urlParams.get('token');

  if (token) {
    verifyEmail(token);
  }
}

// 页面加载时执行
window.addEventListener('load', handleEmailVerification);
```

### Vue 3 示例

```vue
<template>
  <div class="email-verification">
    <div v-if="loading">
      <p>正在验证邮箱...</p>
    </div>
    <div v-else-if="success">
      <h2>邮箱验证成功！</h2>
      <p>您的邮箱已成功验证，现在可以登录了。</p>
      <button @click="goToLogin">前往登录</button>
    </div>
    <div v-else-if="error">
      <h2>验证失败</h2>
      <p>{{ errorMessage }}</p>
      <button @click="resendEmail">重新发送验证邮件</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';

const route = useRoute();
const router = useRouter();

const loading = ref(true);
const success = ref(false);
const error = ref(false);
const errorMessage = ref('');

const verifyEmail = async (token: string) => {
  try {
    const response = await fetch('http://localhost:8080/api/v1/auth/verify-email', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ token }),
    });

    const data = await response.json();

    if (response.ok) {
      success.value = true;
    } else {
      error.value = true;
      errorMessage.value = data.message;
    }
  } catch (err) {
    error.value = true;
    errorMessage.value = '网络错误，请稍后重试';
  } finally {
    loading.value = false;
  }
};

const resendEmail = async () => {
  // 实现重新发送逻辑
  alert('请联系管理员重新发送验证邮件');
};

const goToLogin = () => {
  router.push('/login');
};

onMounted(() => {
  const token = route.query.token as string;
  if (token) {
    verifyEmail(token);
  } else {
    error.value = true;
    errorMessage.value = '缺少验证令牌';
    loading.value = false;
  }
});
</script>
```

## 测试建议

### 1. 手动测试流程

1. 注册新用户
2. 查看控制台输出的验证邮件
3. 复制验证令牌
4. 调用验证接口
5. 登录并确认邮箱已验证

### 2. 自动化测试

使用Postman或类似工具创建测试集合：

1. **注册用户** - 保存返回的用户ID
2. **验证邮箱** - 使用从数据库或日志中获取的token
3. **重新发送验证邮件** - 使用保存的用户ID
4. **验证已验证的邮箱** - 应该返回错误

### 3. 集成测试

参考 `internal/service/auth/email_service_test.go` 中的测试用例。

## 常见问题排查

### 问题1：没有收到验证邮件

**可能原因：**

- 开发环境使用控制台输出，检查服务器日志
- 生产环境检查SMTP配置是否正确
- 检查邮件是否被标记为垃圾邮件

**解决方案：**

- 查看服务器日志确认邮件是否发送
- 使用重新发送功能
- 检查邮箱的垃圾邮件文件夹

### 问题2：验证链接点击后显示令牌无效

**可能原因：**

- 令牌已过期（超过24小时）
- 令牌已被使用
- 令牌格式错误

**解决方案：**

- 使用重新发送功能获取新的验证链接
- 确保URL中的token参数完整且未被截断

### 问题3：重新发送验证邮件失败

**可能原因：**

- 邮箱已经验证
- 用户ID或租户ID错误
- 用户不存在

**解决方案：**

- 检查用户的邮箱验证状态
- 确认用户ID和租户ID正确
- 查看服务器日志获取详细错误信息

## 相关文档

- [邮箱验证功能文档](./EMAIL_VERIFICATION.md)
- [认证系统快速参考](./AUTH_QUICK_REFERENCE.md)
- [API参考文档](./API_REFERENCE_CARD.md)
