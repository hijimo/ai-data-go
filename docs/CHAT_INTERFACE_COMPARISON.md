# Chat 接口对比

## 接口概览

本项目提供两种 Chat 接口，分别适用于不同的使用场景。

## 对比表格

| 特性 | 普通接口 | 流式接口 |
|------|---------|---------|
| **路径** | `/api/v1/chat` | `/api/v1/chat/stream` |
| **HTTP 方法** | POST | POST |
| **响应格式** | `application/json` | `text/event-stream` |
| **响应方式** | 一次性返回完整响应 | 流式返回内容片段 |
| **用户体验** | 需要等待完整响应 | 实时看到生成过程 |
| **感知延迟** | 较高 | 较低 |
| **适用场景** | 短文本、快速响应 | 长文本、需要实时反馈 |
| **实现复杂度** | 简单 | 稍复杂 |
| **客户端要求** | 标准 HTTP 客户端 | 支持 SSE 或流式处理 |
| **网络效率** | 较低（需等待完整响应） | 较高（边生成边传输） |
| **取消支持** | 通过 `/api/v1/chat/abort` | 通过 `/api/v1/chat/abort` |
| **会话管理** | 支持 | 支持 |
| **参数配置** | 支持 | 支持 |

## 请求格式对比

### 普通接口

```http
POST /api/v1/chat
Content-Type: application/json

{
  "message": "你好",
  "messageId": "可选",
  "options": {
    "temperature": 0.7,
    "maxTokens": 2048
  }
}
```

### 流式接口

```http
POST /api/v1/chat/stream
Content-Type: application/json

{
  "message": "你好",
  "messageId": "可选",
  "options": {
    "temperature": 0.7,
    "maxTokens": 2048
  }
}
```

**注意**: 请求格式完全相同！

## 响应格式对比

### 普通接口响应

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    "sessionId": "session-123456",
    "message": "你好！我是一个 AI 助手，很高兴为你服务。",
    "model": "gemini-2.5-flash",
    "usage": {
      "promptTokens": 10,
      "completionTokens": 20,
      "totalTokens": 30
    }
  }
}
```

### 流式接口响应

```
data: {"sessionId":"xxx","content":"你","done":false}

data: {"sessionId":"xxx","content":"好","done":false}

data: {"sessionId":"xxx","content":"！","done":false}

data: {"sessionId":"xxx","content":"我","done":false}

data: {"sessionId":"xxx","content":"是","done":false}

data: {"sessionId":"xxx","content":"一个","done":false}

data: {"sessionId":"xxx","content":" AI ","done":false}

data: {"sessionId":"xxx","content":"助手","done":false}

data: {"sessionId":"xxx","content":"","done":true,"model":"gemini-2.5-flash","usage":{"promptTokens":10,"completionTokens":20,"totalTokens":30}}
```

## 使用场景建议

### 使用普通接口的场景

✅ 短文本生成（< 100 字）  
✅ 快速问答  
✅ 简单的客户端实现  
✅ 不需要实时反馈  
✅ 批量处理  

### 使用流式接口的场景

✅ 长文本生成（> 100 字）  
✅ 文章、代码生成  
✅ 需要实时反馈的场景  
✅ 提升用户体验  
✅ 降低感知延迟  

## 性能对比

### 时间线对比

#### 普通接口

```
客户端发送请求 ──────────────────────────────────────────────────────────> 收到完整响应
                    [等待 AI 生成完整内容]                              [显示内容]
                    ←────────── 5秒 ──────────→                         
```

#### 流式接口

```
客户端发送请求 ──> 收到第一个字 ──> 收到更多内容 ──> ... ──> 收到完成标记
                    [0.5秒]        [持续接收]              [5秒总时长]
                    ↓              ↓                       ↓
                  [显示]         [更新显示]              [完成]
```

**关键差异**: 虽然总时长相同，但流式接口的首字节时间（TTFB）更短，用户感知延迟更低。

## 客户端实现复杂度对比

### 普通接口（简单）

```javascript
// 简单的 fetch 请求
const response = await fetch('/api/v1/chat', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({message: '你好'})
});

const data = await response.json();
console.log(data.data.message);
```

### 流式接口（稍复杂）

```javascript
// 需要处理流式数据
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
  // 处理 SSE 格式...
}
```

## 错误处理对比

### 普通接口

```json
{
  "code": 500,
  "message": "AI 服务调用失败"
}
```

### 流式接口

```
data: {"done":true,"error":{"code":500,"message":"AI 服务调用失败"}}
```

## 选择建议

### 选择普通接口，如果

- 你的应用主要处理短文本
- 你需要简单的客户端实现
- 你不关心实时反馈
- 你的用户可以接受等待

### 选择流式接口，如果

- 你的应用需要生成长文本
- 你想提供更好的用户体验
- 你需要实时显示生成过程
- 你的客户端支持 SSE

### 同时支持两种接口

本项目同时提供两种接口，你可以根据具体场景选择使用：

```javascript
// 根据消息长度选择接口
async function chat(message) {
  const endpoint = message.length > 100 
    ? '/api/v1/chat/stream'  // 长文本使用流式
    : '/api/v1/chat';         // 短文本使用普通
  
  // 调用相应接口...
}
```

## 测试对比

### 测试普通接口

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "你好"}'
```

### 测试流式接口

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "你好"}'
```

**注意**: 流式接口需要 `-N` 参数禁用缓冲。

## 总结

- **普通接口**: 简单、直接，适合短文本和简单场景
- **流式接口**: 更好的用户体验，适合长文本和需要实时反馈的场景

两种接口可以共存，根据实际需求选择使用。建议在用户界面中优先使用流式接口以提供更好的体验。

## 相关文档

- [流式接口完整指南](CHAT_STREAM_GUIDE.md)
- [流式接口快速参考](CHAT_STREAM_QUICK_REF.md)
- [实现摘要](../CHAT_STREAM_IMPLEMENTATION_SUMMARY.md)
