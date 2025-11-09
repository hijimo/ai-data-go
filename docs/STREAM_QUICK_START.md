# 流式消息接口快速开始

## 5 分钟快速上手

### 1. 准备工作

确保你已经：

- ✅ 启动了服务（默认端口 8080）
- ✅ 获取了 JWT Token（通过登录接口）
- ✅ 创建了一个会话（通过创建会话接口）

### 2. 使用测试页面（最简单）

1. 在浏览器中打开测试页面：

   ```
   docs/chat_stream_sse_demo.html
   ```

2. 填写配置信息：
   - **API 地址**：`http://localhost:8080`
   - **会话 ID**：你的会话 ID
   - **访问令牌**：你的 JWT Token

3. 输入消息并点击"发送"，即可看到流式返回的效果！

### 3. 使用 cURL 测试

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/sessions/YOUR_SESSION_ID/messages/stream \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "sessionId": "YOUR_SESSION_ID",
    "message": "你好，请介绍一下你自己"
  }'
```

**替换以下内容**：

- `YOUR_SESSION_ID`：替换为你的会话 ID
- `YOUR_JWT_TOKEN`：替换为你的 JWT Token

### 4. 使用 JavaScript

```javascript
async function testStreamAPI() {
  const sessionId = 'YOUR_SESSION_ID';
  const accessToken = 'YOUR_JWT_TOKEN';
  const message = '你好，请介绍一下你自己';

  const url = `http://localhost:8080/api/v1/chat/sessions/${sessionId}/messages/stream`;

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${accessToken}`,
    },
    body: JSON.stringify({
      sessionId: sessionId,
      message: message,
    }),
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let fullContent = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const chunk = JSON.parse(line.slice(6));
        
        if (chunk.event === 'content') {
          fullContent += chunk.content;
          console.log('当前内容:', fullContent);
        } else if (chunk.event === 'done') {
          console.log('完成!', chunk);
        } else if (chunk.event === 'error') {
          console.error('错误:', chunk.error);
        }
      }
    }
  }
}

testStreamAPI();
```

## 响应示例

### 事件流示例

```
data: {"event":"user_message","userMessage":{"id":"msg-123","role":"user","content":"你好","sequence":1,"createdAt":"2024-01-01T12:00:00Z"}}

data: {"event":"content","aiMessageId":"msg-456","content":"你好"}

data: {"event":"content","content":"！我是"}

data: {"event":"content","content":"一个 AI 助手"}

data: {"event":"done","done":true,"model":"gpt-4","usage":{"promptTokens":10,"completionTokens":50,"totalTokens":60}}

```

### 累积后的完整内容

```
你好！我是一个 AI 助手
```

## 常见问题

### Q: 为什么看不到流式效果？

**A**: 检查以下几点：

1. 确保使用了 `-N` 参数（cURL）或正确处理了流式响应（代码）
2. 检查代理服务器（如 Nginx）是否禁用了缓冲
3. 确认 AI 服务支持流式返回

### Q: 如何获取 JWT Token？

**A**: 通过登录接口获取：

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your-email@example.com",
    "password": "your-password"
  }'
```

### Q: 如何创建会话？

**A**: 通过创建会话接口：

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "测试会话",
    "modelName": "gpt-4"
  }'
```

### Q: 流式接口和非流式接口有什么区别？

**A**:

- **流式接口** (`/messages/stream`)：实时返回内容，用户体验更好
- **非流式接口** (`/messages`)：等待完成后一次性返回，实现更简单

推荐在用户交互场景使用流式接口。

## 下一步

- 📖 阅读 [完整的流式 API 指南](./STREAM_API_GUIDE.md)
- 🔧 查看 [API 路由文档](../internal/api/routes/README.md)
- 🎨 自定义测试页面的样式和功能
- 🚀 集成到你的应用中

## 技术支持

如有问题，请查看：

- [完整文档](./STREAM_API_GUIDE.md)
- [错误码参考](./ERROR_CODES.md)
- 服务器日志
