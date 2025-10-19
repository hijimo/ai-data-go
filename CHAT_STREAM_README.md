# 流式对话功能使用说明

## 快速开始

### 1. 启动服务

```bash
make run
# 或
go run cmd/server/main.go
```

### 2. 测试流式接口

#### 方法一：使用测试脚本

```bash
./test_chat_stream.sh
```

#### 方法二：使用 curl

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "你好，请介绍一下你自己"}'
```

#### 方法三：使用浏览器演示页面

在浏览器中打开 `docs/chat_stream_demo.html` 文件

## 接口说明

### 流式接口

- **路径**: `POST /api/v1/chat/stream`
- **响应格式**: Server-Sent Events (SSE)
- **特点**: 实时返回 AI 生成的内容

### 普通接口（对比）

- **路径**: `POST /api/v1/chat`
- **响应格式**: JSON
- **特点**: 一次性返回完整响应

## 请求示例

```json
{
  "message": "你好，请介绍一下你自己",
  "messageId": "可选的会话ID",
  "options": {
    "temperature": 0.7,
    "maxTokens": 2048
  }
}
```

## 响应示例

```
data: {"sessionId":"xxx","content":"你","done":false}

data: {"sessionId":"xxx","content":"好","done":false}

data: {"sessionId":"xxx","content":"！","done":false}

data: {"sessionId":"xxx","content":"","done":true,"model":"gemini-2.5-flash","usage":{"promptTokens":10,"completionTokens":20,"totalTokens":30}}
```

## 客户端集成

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
      if (data.content) {
        console.log(data.content);
      }
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

## 文档

- **完整指南**: [docs/CHAT_STREAM_GUIDE.md](docs/CHAT_STREAM_GUIDE.md)
- **快速参考**: [docs/CHAT_STREAM_QUICK_REF.md](docs/CHAT_STREAM_QUICK_REF.md)
- **实现摘要**: [CHAT_STREAM_IMPLEMENTATION_SUMMARY.md](CHAT_STREAM_IMPLEMENTATION_SUMMARY.md)

## 功能特点

✅ 实时流式输出  
✅ 支持会话管理  
✅ 完整的错误处理  
✅ Token 使用统计  
✅ 支持参数配置  
✅ 兼容普通接口  

## 技术栈

- Server-Sent Events (SSE)
- Genkit Streaming API
- Go Channels
- HTTP Flusher

## 注意事项

1. 使用 `-N` 参数（curl）禁用缓冲
2. 确保客户端支持 SSE
3. 设置合理的超时时间
4. 处理连接中断情况

## 相关接口

- `POST /api/v1/chat` - 普通对话接口
- `POST /api/v1/chat/stream` - 流式对话接口
- `POST /api/v1/chat/abort` - 中止对话接口

## 问题反馈

如有问题，请查看详细文档或提交 Issue。
