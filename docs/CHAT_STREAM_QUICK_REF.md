# 流式对话接口快速参考

## 接口地址

```
POST /api/v1/chat/stream
```

## 快速开始

### 最简单的请求

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "你好"}'
```

### 带参数的请求

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "message": "写一首诗",
    "options": {
      "temperature": 0.9,
      "maxTokens": 500
    }
  }'
```

## 请求参数

```json
{
  "message": "必填，用户消息",
  "messageId": "可选，会话ID",
  "options": {
    "temperature": 0.7,    // 可选，0-2
    "maxTokens": 2048,     // 可选
    "topP": 0.9,           // 可选，0-1
    "topK": 40             // 可选
  }
}
```

## 响应格式

### 流式数据块

```json
// 内容块
{"sessionId":"xxx","content":"你","done":false}
{"sessionId":"xxx","content":"好","done":false}

// 完成块
{"sessionId":"xxx","content":"","done":true,"model":"gemini-2.5-flash","usage":{"promptTokens":10,"completionTokens":20,"totalTokens":30}}
```

### 错误块

```json
{"done":true,"error":{"code":500,"message":"错误信息"}}
```

## 客户端示例

### JavaScript

```javascript
const response = await fetch('/api/v1/chat/stream', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({message: '你好'})
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const {done, value} = await reader.read();
  if (done) break;
  
  const chunk = decoder.decode(value);
  const lines = chunk.split('\n');
  
  for (const line of lines) {
    if (line.startsWith('data: ')) {
      const data = JSON.parse(line.substring(6));
      console.log(data.content);
    }
  }
}
```

### Python

```python
import requests
import json

response = requests.post(
    'http://localhost:8080/api/v1/chat/stream',
    json={'message': '你好'},
    stream=True
)

for line in response.iter_lines():
    if line and line.startswith(b'data: '):
        chunk = json.loads(line[6:])
        if chunk.get('content'):
            print(chunk['content'], end='', flush=True)
```

### Go

```go
resp, _ := http.Post(url, "application/json", body)
defer resp.Body.Close()

scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "data: ") {
        var chunk StreamChunk
        json.Unmarshal([]byte(line[6:]), &chunk)
        fmt.Print(chunk.Content)
    }
}
```

## 测试工具

### 1. 命令行测试

```bash
./test_chat_stream.sh
```

### 2. 浏览器测试

打开 `docs/chat_stream_demo.html` 文件

### 3. 手动测试

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "测试消息"}'
```

## 常见问题

### Q: 为什么没有实时输出？

A: 确保使用 `-N` 参数（curl）或禁用缓冲

### Q: 如何取消正在进行的请求？

A: 使用 `/api/v1/chat/abort` 接口

### Q: 支持哪些模型？

A: 支持所有 Genkit 兼容的模型，默认使用配置中的模型

### Q: 如何处理错误？

A: 检查响应中的 `error` 字段

## 性能建议

1. 使用连接池
2. 设置合理的超时时间
3. 实现重试机制
4. 客户端增量渲染

## 相关接口

- `POST /api/v1/chat` - 普通对话（非流式）
- `POST /api/v1/chat/abort` - 中止对话

## 技术栈

- Server-Sent Events (SSE)
- Genkit Streaming API
- Go Channels
- HTTP Flusher

## 更多信息

详细文档: `docs/CHAT_STREAM_GUIDE.md`
