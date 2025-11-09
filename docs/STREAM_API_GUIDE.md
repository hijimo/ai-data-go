# 流式消息接口使用指南

## 概述

流式消息接口使用 **Server-Sent Events (SSE)** 技术，允许客户端实时接收 AI 生成的消息内容，提供更好的用户体验。

## 接口信息

### 端点

```
POST /api/v1/chat/sessions/{id}/messages/stream
```

### 认证

需要在请求头中提供 JWT Token：

```
Authorization: Bearer <your-jwt-token>
```

### 请求参数

**路径参数：**

- `id` (string, required): 会话 ID

**请求体 (JSON)：**

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "message": "你好，请介绍一下你自己",
  "options": {
    "temperature": 0.7,
    "maxTokens": 2048,
    "topP": 0.9,
    "topK": 40
  }
}
```

**字段说明：**

- `sessionId` (string, required): 会话 ID（必须与路径参数一致）
- `message` (string, required): 用户消息内容
- `options` (object, optional): AI 高级参数
  - `temperature` (float, optional): 温度值，控制输出的随机性（0-2）
  - `maxTokens` (int, optional): 最大 token 数
  - `topP` (float, optional): Top-P 采样参数（0-1）
  - `topK` (int, optional): Top-K 采样参数

### 响应格式

响应使用 **SSE (Server-Sent Events)** 格式：

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

每个事件的格式为：

```
data: <json-data>

```

## 事件类型

### 1. user_message 事件

用户消息已保存到数据库。

```json
{
  "event": "user_message",
  "userMessage": {
    "id": "msg-user-123",
    "role": "user",
    "content": "你好，请介绍一下你自己",
    "sequence": 1,
    "createdAt": "2024-01-01T12:00:00Z"
  }
}
```

### 2. content 事件

AI 生成的内容片段。客户端应该累积这些片段以显示完整内容。

**第一个 content 事件（包含 AI 消息 ID）：**

```json
{
  "event": "content",
  "aiMessageId": "msg-ai-456",
  "content": "你好"
}
```

**后续 content 事件：**

```json
{
  "event": "content",
  "content": "！我是"
}
```

```json
{
  "event": "content",
  "content": "一个 AI 助手"
}
```

### 3. done 事件

流式传输完成。

```json
{
  "event": "done",
  "done": true,
  "model": "gpt-4",
  "usage": {
    "promptTokens": 10,
    "completionTokens": 50,
    "totalTokens": 60
  }
}
```

### 4. error 事件

发生错误。

```json
{
  "event": "error",
  "error": "AI 服务调用失败: connection timeout"
}
```

## 使用示例

### JavaScript (Fetch API)

```javascript
async function sendStreamMessage(sessionId, message, accessToken) {
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

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  // 读取 SSE 流
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let fullContent = '';

  while (true) {
    const { done, value } = await reader.read();

    if (done) {
      break;
    }

    // 解码数据
    buffer += decoder.decode(value, { stream: true });

    // 处理 SSE 消息
    const lines = buffer.split('\n');
    buffer = lines.pop(); // 保留不完整的行

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = line.slice(6);
        const chunk = JSON.parse(data);

        if (chunk.event === 'user_message') {
          console.log('用户消息已保存:', chunk.userMessage);
        } else if (chunk.event === 'content') {
          fullContent += chunk.content;
          console.log('当前内容:', fullContent);
          // 更新 UI 显示内容
        } else if (chunk.event === 'done') {
          console.log('完成:', chunk);
        } else if (chunk.event === 'error') {
          console.error('错误:', chunk.error);
        }
      }
    }
  }
}
```

### JavaScript (EventSource - 仅支持 GET)

注意：EventSource API 仅支持 GET 请求，不适用于我们的 POST 接口。建议使用 Fetch API。

### cURL

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/sessions/{session-id}/messages/stream \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-jwt-token" \
  -d '{
    "sessionId": "550e8400-e29b-41d4-a716-446655440000",
    "message": "你好，请介绍一下你自己"
  }'
```

参数说明：

- `-N`: 禁用缓冲，立即显示输出

### Python

```python
import requests
import json

def send_stream_message(session_id, message, access_token):
    url = f"http://localhost:8080/api/v1/chat/sessions/{session_id}/messages/stream"
    
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {access_token}"
    }
    
    data = {
        "sessionId": session_id,
        "message": message
    }
    
    response = requests.post(url, headers=headers, json=data, stream=True)
    response.raise_for_status()
    
    full_content = ""
    
    for line in response.iter_lines():
        if line:
            line = line.decode('utf-8')
            if line.startswith('data: '):
                data = line[6:]
                chunk = json.loads(data)
                
                if chunk['event'] == 'user_message':
                    print(f"用户消息已保存: {chunk['userMessage']}")
                elif chunk['event'] == 'content':
                    full_content += chunk['content']
                    print(f"当前内容: {full_content}")
                elif chunk['event'] == 'done':
                    print(f"完成: {chunk}")
                elif chunk['event'] == 'error':
                    print(f"错误: {chunk['error']}")

# 使用示例
send_stream_message(
    session_id="550e8400-e29b-41d4-a716-446655440000",
    message="你好，请介绍一下你自己",
    access_token="your-jwt-token"
)
```

### Go

```go
package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

type StreamChunk struct {
    Event       string      `json:"event"`
    UserMessage interface{} `json:"userMessage,omitempty"`
    AIMessageID string      `json:"aiMessageId,omitempty"`
    Content     string      `json:"content,omitempty"`
    Done        bool        `json:"done,omitempty"`
    Model       string      `json:"model,omitempty"`
    Usage       interface{} `json:"usage,omitempty"`
    Error       string      `json:"error,omitempty"`
}

func sendStreamMessage(sessionID, message, accessToken string) error {
    url := fmt.Sprintf("http://localhost:8080/api/v1/chat/sessions/%s/messages/stream", sessionID)
    
    reqBody := map[string]interface{}{
        "sessionId": sessionID,
        "message":   message,
    }
    
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+accessToken)
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }
    
    reader := bufio.NewReader(resp.Body)
    fullContent := ""
    
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                break
            }
            return err
        }
        
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "data: ") {
            data := line[6:]
            
            var chunk StreamChunk
            if err := json.Unmarshal([]byte(data), &chunk); err != nil {
                fmt.Printf("解析错误: %v\n", err)
                continue
            }
            
            switch chunk.Event {
            case "user_message":
                fmt.Printf("用户消息已保存: %+v\n", chunk.UserMessage)
            case "content":
                fullContent += chunk.Content
                fmt.Printf("当前内容: %s\n", fullContent)
            case "done":
                fmt.Printf("完成: 模型=%s, 使用=%+v\n", chunk.Model, chunk.Usage)
            case "error":
                fmt.Printf("错误: %s\n", chunk.Error)
            }
        }
    }
    
    return nil
}

func main() {
    err := sendStreamMessage(
        "550e8400-e29b-41d4-a716-446655440000",
        "你好，请介绍一下你自己",
        "your-jwt-token",
    )
    if err != nil {
        fmt.Printf("发送失败: %v\n", err)
    }
}
```

## 测试页面

我们提供了一个交互式的测试页面，可以直接在浏览器中测试流式接口：

```
docs/chat_stream_sse_demo.html
```

使用方法：

1. 在浏览器中打开 `chat_stream_sse_demo.html`
2. 填写 API 地址（如 `http://localhost:8080`）
3. 填写会话 ID
4. 填写访问令牌（JWT Token）
5. 输入消息并点击发送

## 错误处理

### 常见错误

1. **401 Unauthorized**
   - 原因：JWT Token 无效或过期
   - 解决：重新登录获取新的 Token

2. **403 Forbidden**
   - 原因：无权访问该会话
   - 解决：确认会话属于当前用户

3. **404 Not Found**
   - 原因：会话不存在
   - 解决：使用有效的会话 ID

4. **422 Unprocessable Entity**
   - 原因：请求参数验证失败
   - 解决：检查请求参数格式

5. **500 Internal Server Error**
   - 原因：服务器内部错误
   - 解决：查看服务器日志

### 错误事件处理

当收到 `error` 事件时，应该：

1. 停止处理后续内容
2. 向用户显示错误信息
3. 记录错误日志
4. 允许用户重试

## 性能优化建议

### 客户端

1. **使用缓冲区**：累积小的内容片段，批量更新 UI
2. **节流更新**：限制 UI 更新频率（如每 50ms 更新一次）
3. **虚拟滚动**：对于长消息，使用虚拟滚动技术
4. **取消请求**：用户离开页面时，取消正在进行的请求

### 服务端

1. **连接池管理**：合理配置 AI 服务的连接池
2. **超时控制**：设置合理的超时时间
3. **资源限制**：限制并发流式请求数量
4. **监控告警**：监控流式请求的性能指标

## 与非流式接口的对比

| 特性 | 流式接口 | 非流式接口 |
|------|---------|-----------|
| 端点 | `/messages/stream` | `/messages` |
| 响应类型 | SSE (text/event-stream) | JSON (application/json) |
| 用户体验 | 实时显示，体验更好 | 等待完成后显示 |
| 实现复杂度 | 较高 | 较低 |
| 适用场景 | 长文本生成、实时交互 | 短文本、批量处理 |
| 网络开销 | 较低（增量传输） | 较高（一次性传输） |

## 最佳实践

1. **优先使用流式接口**：对于用户交互场景，优先使用流式接口
2. **提供降级方案**：如果流式接口失败，自动降级到非流式接口
3. **显示进度指示**：在等待第一个内容块时，显示加载动画
4. **保存完整内容**：在客户端累积完整内容，便于后续操作
5. **错误重试**：实现智能重试机制，提高可靠性

## 安全注意事项

1. **JWT Token 保护**：不要在 URL 中传递 Token，使用 Header
2. **HTTPS**：生产环境必须使用 HTTPS
3. **速率限制**：实施合理的速率限制，防止滥用
4. **内容过滤**：对生成的内容进行安全过滤
5. **会话验证**：严格验证会话所有权

## 故障排查

### 问题：无法建立 SSE 连接

**可能原因：**

- 代理服务器（如 Nginx）缓冲了响应
- 防火墙阻止了长连接

**解决方案：**

- 配置 Nginx：`proxy_buffering off;`
- 添加响应头：`X-Accel-Buffering: no`

### 问题：内容显示不完整

**可能原因：**

- 客户端解析 SSE 格式错误
- 网络中断

**解决方案：**

- 检查 SSE 解析逻辑
- 实现断线重连机制

### 问题：性能问题

**可能原因：**

- UI 更新过于频繁
- 内存泄漏

**解决方案：**

- 使用节流/防抖
- 及时清理事件监听器

## 相关文档

- [会话管理 API 文档](./SESSION_API.md)
- [非流式消息接口文档](./MESSAGE_API.md)
- [认证授权文档](./AUTH_API.md)
- [错误码参考](./ERROR_CODES.md)
