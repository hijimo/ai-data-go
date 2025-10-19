# 流式对话接口使用指南

## 概述

流式对话接口允许客户端以流式方式接收 AI 生成的内容，而不是等待完整响应。这提供了更好的用户体验，特别是对于长文本生成场景。

## 接口信息

- **路径**: `/api/v1/chat/stream`
- **方法**: `POST`
- **Content-Type**: `application/json`
- **响应格式**: `text/event-stream` (Server-Sent Events)

## 请求格式

### 请求体

```json
{
  "message": "你好，请介绍一下你自己",
  "messageId": "550e8400-e29b-41d4-a716-446655440000",  // 可选，用于继续对话
  "options": {  // 可选，AI 高级参数
    "temperature": 0.7,
    "maxTokens": 2048,
    "topP": 0.9,
    "topK": 40
  }
}
```

### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| message | string | 是 | 用户消息内容 |
| messageId | string | 否 | 消息ID，用于继续对话 |
| options | object | 否 | AI 高级参数 |
| options.temperature | float | 否 | 温度值，控制输出随机性（0-2） |
| options.maxTokens | int | 否 | 最大 token 数 |
| options.topP | float | 否 | Top-P 采样参数（0-1） |
| options.topK | int | 否 | Top-K 采样参数 |

## 响应格式

### Server-Sent Events (SSE) 格式

响应采用 SSE 格式，每个数据块格式如下：

```
data: {"sessionId":"xxx","content":"你","done":false}

data: {"sessionId":"xxx","content":"好","done":false}

data: {"sessionId":"xxx","content":"！","done":false}

data: {"sessionId":"xxx","content":"","done":true,"model":"gemini-2.5-flash","usage":{"promptTokens":10,"completionTokens":20,"totalTokens":30}}
```

### 数据块结构

```json
{
  "sessionId": "session-123456",      // 会话ID
  "content": "文本片段",               // 内容片段
  "done": false,                      // 是否完成
  "model": "gemini-2.5-flash",       // 模型名称（仅在 done=true 时提供）
  "usage": {                          // Token 使用情况（仅在 done=true 时提供）
    "promptTokens": 10,
    "completionTokens": 20,
    "totalTokens": 30
  },
  "error": null                       // 错误信息（如果有）
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| sessionId | string | 会话ID，用于标识对话会话 |
| content | string | 生成的文本片段 |
| done | boolean | 是否完成生成 |
| model | string | 使用的模型名称（仅在完成时提供） |
| usage | object | Token 使用统计（仅在完成时提供） |
| error | object | 错误信息（如果发生错误） |

## 使用示例

### 1. 使用 curl

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请介绍一下你自己"
  }'
```

**注意**: `-N` 参数禁用缓冲，确保实时接收流式数据。

### 2. 使用 JavaScript (Fetch API)

```javascript
async function chatStream(message) {
  const response = await fetch('http://localhost:8080/api/v1/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message: message
    })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    
    if (done) break;
    
    const chunk = decoder.decode(value);
    const lines = chunk.split('\n');
    
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = JSON.parse(line.substring(6));
        
        if (data.content) {
          console.log(data.content);
        }
        
        if (data.done) {
          console.log('完成！模型:', data.model);
          console.log('Token 使用:', data.usage);
        }
        
        if (data.error) {
          console.error('错误:', data.error);
        }
      }
    }
  }
}

// 使用示例
chatStream('你好，请介绍一下你自己');
```

### 3. 使用 EventSource (推荐)

**注意**: EventSource 只支持 GET 请求，对于 POST 请求需要使用 Fetch API 或其他库。

对于支持 GET 的场景，可以使用 EventSource：

```javascript
const eventSource = new EventSource('http://localhost:8080/api/v1/chat/stream?message=你好');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  if (data.content) {
    console.log(data.content);
  }
  
  if (data.done) {
    console.log('完成！');
    eventSource.close();
  }
};

eventSource.onerror = (error) => {
  console.error('错误:', error);
  eventSource.close();
};
```

### 4. 使用 Python

```python
import requests
import json

def chat_stream(message):
    url = 'http://localhost:8080/api/v1/chat/stream'
    headers = {'Content-Type': 'application/json'}
    data = {'message': message}
    
    response = requests.post(url, headers=headers, json=data, stream=True)
    
    for line in response.iter_lines():
        if line:
            line = line.decode('utf-8')
            if line.startswith('data: '):
                chunk = json.loads(line[6:])
                
                if chunk.get('content'):
                    print(chunk['content'], end='', flush=True)
                
                if chunk.get('done'):
                    print('\n完成！')
                    print(f"模型: {chunk.get('model')}")
                    print(f"Token 使用: {chunk.get('usage')}")
                
                if chunk.get('error'):
                    print(f"\n错误: {chunk['error']}")

# 使用示例
chat_stream('你好，请介绍一下你自己')
```

### 5. 使用 Go

```go
package main

import (
 "bufio"
 "bytes"
 "encoding/json"
 "fmt"
 "net/http"
 "strings"
)

type ChatRequest struct {
 Message string `json:"message"`
}

type StreamChunk struct {
 SessionID string  `json:"sessionId,omitempty"`
 Content   string  `json:"content"`
 Done      bool    `json:"done"`
 Model     string  `json:"model,omitempty"`
 Usage     *Usage  `json:"usage,omitempty"`
 Error     *string `json:"error,omitempty"`
}

type Usage struct {
 PromptTokens     int `json:"promptTokens"`
 CompletionTokens int `json:"completionTokens"`
 TotalTokens      int `json:"totalTokens"`
}

func chatStream(message string) error {
 url := "http://localhost:8080/api/v1/chat/stream"
 
 reqBody, _ := json.Marshal(ChatRequest{Message: message})
 req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
 if err != nil {
  return err
 }
 
 req.Header.Set("Content-Type", "application/json")
 
 client := &http.Client{}
 resp, err := client.Do(req)
 if err != nil {
  return err
 }
 defer resp.Body.Close()
 
 scanner := bufio.NewScanner(resp.Body)
 for scanner.Scan() {
  line := scanner.Text()
  
  if strings.HasPrefix(line, "data: ") {
   var chunk StreamChunk
   if err := json.Unmarshal([]byte(line[6:]), &chunk); err != nil {
    continue
   }
   
   if chunk.Content != "" {
    fmt.Print(chunk.Content)
   }
   
   if chunk.Done {
    fmt.Println("\n完成！")
    fmt.Printf("模型: %s\n", chunk.Model)
    if chunk.Usage != nil {
     fmt.Printf("Token 使用: %+v\n", chunk.Usage)
    }
   }
   
   if chunk.Error != nil {
    fmt.Printf("\n错误: %s\n", *chunk.Error)
   }
  }
 }
 
 return scanner.Err()
}

func main() {
 chatStream("你好，请介绍一下你自己")
}
```

## 错误处理

### 错误响应格式

当发生错误时，会在流中发送包含错误信息的数据块：

```json
{
  "done": true,
  "error": {
    "code": 500,
    "message": "AI 服务调用失败"
  }
}
```

### 常见错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|-------------|------|
| 400 | 400 | 请求参数错误 |
| 422 | 422 | 参数验证失败 |
| 500 | 500 | 服务器内部错误 |
| 503 | 503 | AI 服务不可用 |

## 最佳实践

### 1. 连接管理

- 使用连接池管理 HTTP 连接
- 设置合理的超时时间
- 在完成后及时关闭连接

### 2. 错误处理

- 始终检查 `error` 字段
- 实现重试机制（指数退避）
- 记录错误日志便于排查

### 3. 性能优化

- 使用流式处理，避免缓冲整个响应
- 在客户端实现增量渲染
- 考虑使用 WebSocket 作为替代方案（如果需要双向通信）

### 4. 用户体验

- 显示加载指示器
- 实时显示生成的内容
- 提供取消功能（使用 `/api/v1/chat/abort` 接口）

## 与普通接口的对比

| 特性 | 普通接口 (`/api/v1/chat`) | 流式接口 (`/api/v1/chat/stream`) |
|------|---------------------------|----------------------------------|
| 响应方式 | 一次性返回完整响应 | 流式返回内容片段 |
| 用户体验 | 需要等待完整响应 | 实时看到生成过程 |
| 适用场景 | 短文本、快速响应 | 长文本、需要实时反馈 |
| 实现复杂度 | 简单 | 稍复杂 |
| 网络效率 | 较低（需等待完整响应） | 较高（边生成边传输） |

## 注意事项

1. **浏览器兼容性**: 确保客户端支持 Server-Sent Events
2. **代理配置**: 某些代理服务器（如 nginx）可能需要特殊配置以支持流式传输
3. **超时设置**: 流式请求可能持续较长时间，需要设置合适的超时时间
4. **并发限制**: 考虑限制同时进行的流式请求数量

## 测试

使用提供的测试脚本进行测试：

```bash
./test_chat_stream.sh
```

或者手动测试：

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "你好"}'
```

## 相关接口

- **普通对话接口**: `/api/v1/chat` - 一次性返回完整响应
- **中止对话接口**: `/api/v1/chat/abort` - 取消正在进行的对话

## 技术实现

流式接口基于以下技术实现：

1. **Server-Sent Events (SSE)**: 标准的服务器推送技术
2. **Genkit Streaming API**: 使用 `ai.WithStreaming` 选项
3. **Go Channels**: 用于在 goroutine 之间传递流式数据
4. **HTTP Flusher**: 确保数据实时发送到客户端

## 参考资料

- [Server-Sent Events 规范](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [Genkit 文档](https://firebase.google.com/docs/genkit)
- [MDN - Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
