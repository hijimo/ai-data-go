# Genkit 多模型 API 使用示例

## 概述

本文档提供了 Genkit 多模型支持系统的完整 API 使用示例，包括配置管理、消息发送、流式响应等各种场景的实际代码示例。

## 目录

- [认证](#认证)
- [模型配置管理](#模型配置管理)
- [发送消息](#发送消息)
- [流式响应](#流式响应)
- [错误处理](#错误处理)
- [完整示例](#完整示例)

## 认证

所有 API 请求都需要在 HTTP 头中包含 JWT 令牌：

```bash
Authorization: Bearer <your-jwt-token>
```

### 获取令牌

```bash
# 登录获取令牌
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "your-password"
  }'
```

**响应示例**:

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": 3600
  }
}
```

## 模型配置管理

### 1. 创建模型配置

#### 创建 Google AI (Gemini) 配置

```bash
curl -X POST https://api.example.com/api/v1/model-configurations \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gemini Pro",
    "model": "gemini-1.5-pro",
    "modelProvider": "googlegenai",
    "apiKey": "AIzaSy...",
    "queryParams": {
      "defaultTemperature": 0.7,
      "defaultMaxTokens": 2048
    }
  }'
```

**响应示例**:

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "name": "Gemini Pro",
    "model": "gemini-1.5-pro",
    "modelProvider": "googlegenai",
    "apiKey": "AIza****...",
    "queryParams": {
      "defaultTemperature": 0.7,
      "defaultMaxTokens": 2048
    },
    "isEnabled": true,
    "createdAt": "2025-12-01T10:00:00Z"
  }
}
```

#### 创建 OpenAI 配置

```bash
curl -X POST https://api.example.com/api/v1/model-configurations \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GPT-4",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-...",
    "queryParams": {
      "defaultTemperature": 0.7,
      "defaultMaxTokens": 4096
    }
  }'
```

#### 创建 Azure OpenAI 配置

```bash
curl -X POST https://api.example.com/api/v1/model-configurations \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Azure GPT-4",
    "model": "gpt-4",
    "modelProvider": "azureopenai",
    "apiKey": "your-azure-api-key",
    "queryParams": {
      "azureEndpoint": "https://your-resource.openai.azure.com",
      "azureDeployment": "gpt-4",
      "azureApiVersion": "2024-02-15-preview",
      "defaultTemperature": 0.7,
      "defaultMaxTokens": 8192
    }
  }'
```

#### 创建阿里云百炼配置

```bash
curl -X POST https://api.example.com/api/v1/model-configurations \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "通义千问 Turbo",
    "model": "qwen-turbo",
    "modelProvider": "bianlian",
    "apiKey": "sk-...",
    "queryParams": {
      "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
      "bailianWorkspace": "default",
      "defaultTemperature": 0.7,
      "defaultMaxTokens": 2048
    }
  }'
```

### 2. 查询模型配置列表

```bash
# 查询所有配置
curl -X GET "https://api.example.com/api/v1/model-configurations?pageNo=1&pageSize=10" \
  -H "Authorization: Bearer <your-token>"

# 按提供商过滤
curl -X GET "https://api.example.com/api/v1/model-configurations?modelProvider=openai" \
  -H "Authorization: Bearer <your-token>"

# 只查询启用的配置
curl -X GET "https://api.example.com/api/v1/model-configurations?isEnabled=true" \
  -H "Authorization: Bearer <your-token>"
```

**响应示例**:

```json
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "data": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
        "name": "GPT-4",
        "model": "gpt-4",
        "modelProvider": "openai",
        "isEnabled": true,
        "createdAt": "2025-12-01T10:00:00Z"
      },
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
        "name": "Gemini Pro",
        "model": "gemini-1.5-pro",
        "modelProvider": "googlegenai",
        "isEnabled": true,
        "createdAt": "2025-12-01T10:05:00Z"
      }
    ],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 2,
    "totalPage": 1
  }
}
```

### 3. 查询单个配置

```bash
curl -X GET https://api.example.com/api/v1/model-configurations/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <your-token>"
```

**响应示例**:

```json
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "name": "GPT-4",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-yo****here",
    "queryParams": {
      "defaultTemperature": 0.7,
      "defaultMaxTokens": 4096
    },
    "isEnabled": true,
    "createdAt": "2025-12-01T10:00:00Z"
  }
}
```

### 4. 更新模型配置

```bash
curl -X PUT https://api.example.com/api/v1/model-configurations/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GPT-4 Turbo",
    "model": "gpt-4-turbo",
    "queryParams": {
      "defaultTemperature": 0.8,
      "defaultMaxTokens": 8192
    }
  }'
```

### 5. 启用/禁用模型

```bash
# 禁用模型
curl -X PATCH https://api.example.com/api/v1/model-configurations/550e8400-e29b-41d4-a716-446655440000/status \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "disabled"
  }'

# 启用模型
curl -X PATCH https://api.example.com/api/v1/model-configurations/550e8400-e29b-41d4-a716-446655440000/status \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "enabled"
  }'
```

### 6. 删除模型配置

```bash
curl -X DELETE https://api.example.com/api/v1/model-configurations/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <your-token>"
```

## 发送消息

### 1. 创建会话

```bash
curl -X POST https://api.example.com/api/v1/chat/sessions \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "我的对话",
    "modelName": "GPT-4"
  }'
```

**响应示例**:

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "userId": "880e8400-e29b-41d4-a716-446655440003",
    "title": "我的对话",
    "modelName": "GPT-4",
    "createdAt": "2025-12-01T11:00:00Z"
  }
}
```

### 2. 发送消息（非流式）

#### 使用默认模型

```bash
curl -X POST https://api.example.com/api/v1/chat/sessions/770e8400-e29b-41d4-a716-446655440002/messages \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请介绍一下自己"
  }'
```

#### 指定模型

```bash
curl -X POST https://api.example.com/api/v1/chat/sessions/770e8400-e29b-41d4-a716-446655440002/messages \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "用中文写一首关于春天的诗",
    "options": {
      "modelName": "通义千问 Turbo",
      "temperature": 0.8,
      "maxTokens": 2000
    }
  }'
```

#### 使用 Azure OpenAI

```bash
curl -X POST https://api.example.com/api/v1/chat/sessions/770e8400-e29b-41d4-a716-446655440002/messages \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "解释一下量子计算的基本原理",
    "options": {
      "modelName": "Azure GPT-4",
      "temperature": 0.3,
      "maxTokens": 3000
    }
  }'
```

**响应示例**:

```json
{
  "code": 200,
  "message": "发送成功",
  "data": {
    "userMessage": {
      "id": "990e8400-e29b-41d4-a716-446655440004",
      "sessionId": "770e8400-e29b-41d4-a716-446655440002",
      "role": "user",
      "content": "你好，请介绍一下自己",
      "createdAt": "2025-12-01T11:05:00Z"
    },
    "assistantMessage": {
      "id": "aa0e8400-e29b-41d4-a716-446655440005",
      "sessionId": "770e8400-e29b-41d4-a716-446655440002",
      "role": "assistant",
      "content": "你好！我是一个AI助手，基于大型语言模型开发。我可以帮助你回答问题、提供信息、进行对话交流等。我的目标是为你提供有用、准确的帮助。有什么我可以帮你的吗？",
      "modelName": "GPT-4",
      "usage": {
        "promptTokens": 15,
        "completionTokens": 45,
        "totalTokens": 60
      },
      "createdAt": "2025-12-01T11:05:02Z"
    }
  }
}
```

## 流式响应

### 1. 发送流式消息

```bash
curl -X POST https://api.example.com/api/v1/chat/sessions/770e8400-e29b-41d4-a716-446655440002/messages/stream \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "写一首关于春天的诗",
    "options": {
      "modelName": "GPT-4",
      "temperature": 0.8
    }
  }'
```

**响应格式** (Server-Sent Events):

```
data: {"event":"message","data":{"delta":"春"}}

data: {"event":"message","data":{"delta":"风"}}

data: {"event":"message","data":{"delta":"拂"}}

data: {"event":"message","data":{"delta":"面"}}

data: {"event":"message","data":{"delta":"，"}}

data: {"event":"message","data":{"delta":"万"}}

data: {"event":"message","data":{"delta":"物"}}

data: {"event":"message","data":{"delta":"复"}}

data: {"event":"message","data":{"delta":"苏"}}

data: {"event":"done","data":{"usage":{"promptTokens":12,"completionTokens":48,"totalTokens":60}}}
```

### 2. JavaScript 客户端示例

```javascript
// 使用 Fetch API 处理 SSE
async function sendStreamMessage(sessionId, message, modelName) {
  const response = await fetch(`/api/v1/chat/sessions/${sessionId}/messages/stream`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      message: message,
      options: {
        modelName: modelName,
        temperature: 0.7
      }
    })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let fullContent = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    const chunk = decoder.decode(value);
    const lines = chunk.split('\n');

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = JSON.parse(line.slice(6));
        
        if (data.event === 'message') {
          fullContent += data.data.delta;
          console.log('收到内容:', data.data.delta);
          // 更新 UI
          updateMessageDisplay(fullContent);
        } else if (data.event === 'done') {
          console.log('Token 使用:', data.data.usage);
          // 显示完成状态
          showCompletionStatus(data.data.usage);
        }
      }
    }
  }

  return fullContent;
}
```

### 3. Python 客户端示例

```python
import requests
import json

def send_stream_message(session_id, message, model_name, token):
    url = f"https://api.example.com/api/v1/chat/sessions/{session_id}/messages/stream"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    data = {
        "message": message,
        "options": {
            "modelName": model_name,
            "temperature": 0.7
        }
    }
    
    response = requests.post(url, headers=headers, json=data, stream=True)
    full_content = ""
    
    for line in response.iter_lines():
        if line:
            line = line.decode('utf-8')
            if line.startswith('data: '):
                event_data = json.loads(line[6:])
                
                if event_data['event'] == 'message':
                    delta = event_data['data']['delta']
                    full_content += delta
                    print(delta, end='', flush=True)
                elif event_data['event'] == 'done':
                    usage = event_data['data']['usage']
                    print(f"\n\nToken 使用: {usage}")
    
    return full_content

# 使用示例
token = "your-jwt-token"
session_id = "770e8400-e29b-41d4-a716-446655440002"
result = send_stream_message(session_id, "写一首诗", "GPT-4", token)
```

## 错误处理

### 常见错误响应

#### 1. 配置不存在

```json
{
  "code": 404,
  "message": "模型配置不存在: GPT-5"
}
```

#### 2. 模型已禁用

```json
{
  "code": 400,
  "message": "模型已禁用: GPT-4"
}
```

#### 3. API 密钥无效

```json
{
  "code": 500,
  "message": "AI 调用失败: invalid API key"
}
```

#### 4. 权限不足

```json
{
  "code": 403,
  "message": "权限不足：无法访问其他租户的模型配置"
}
```

#### 5. 参数验证失败

```json
{
  "code": 400,
  "message": "参数验证失败: temperature 必须在 0-2 之间"
}
```

### 错误处理最佳实践

#### JavaScript 示例

```javascript
async function sendMessage(sessionId, message, modelName) {
  try {
    const response = await fetch(`/api/v1/chat/sessions/${sessionId}/messages`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        message: message,
        options: { modelName: modelName }
      })
    });

    const result = await response.json();

    if (result.code !== 200) {
      // 处理业务错误
      console.error('业务错误:', result.message);
      showError(result.message);
      return null;
    }

    return result.data;
  } catch (error) {
    // 处理网络错误
    console.error('网络错误:', error);
    showError('网络连接失败，请稍后重试');
    return null;
  }
}
```

#### Python 示例

```python
import requests
from typing import Optional, Dict

class APIError(Exception):
    """API 错误"""
    def __init__(self, code: int, message: str):
        self.code = code
        self.message = message
        super().__init__(f"[{code}] {message}")

def send_message(session_id: str, message: str, model_name: str, token: str) -> Optional[Dict]:
    """发送消息"""
    url = f"https://api.example.com/api/v1/chat/sessions/{session_id}/messages"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    data = {
        "message": message,
        "options": {
            "modelName": model_name
        }
    }
    
    try:
        response = requests.post(url, headers=headers, json=data, timeout=30)
        result = response.json()
        
        if result['code'] != 200:
            raise APIError(result['code'], result['message'])
        
        return result['data']
    
    except requests.exceptions.Timeout:
        print("请求超时，请稍后重试")
        return None
    except requests.exceptions.ConnectionError:
        print("网络连接失败")
        return None
    except APIError as e:
        print(f"API 错误: {e.message}")
        return None
    except Exception as e:
        print(f"未知错误: {str(e)}")
        return None

# 使用示例
result = send_message(
    session_id="770e8400-e29b-41d4-a716-446655440002",
    message="你好",
    model_name="GPT-4",
    token="your-jwt-token"
)

if result:
    print("用户消息:", result['userMessage']['content'])
    print("AI 回复:", result['assistantMessage']['content'])
```

## 完整示例

### 场景：创建配置并发送消息

#### Bash 脚本

```bash
#!/bin/bash

# 配置
API_BASE="https://api.example.com/api/v1"
EMAIL="user@example.com"
PASSWORD="your-password"

# 1. 登录获取令牌
echo "1. 登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.accessToken')
echo "令牌: ${TOKEN:0:20}..."

# 2. 创建模型配置
echo -e "\n2. 创建模型配置..."
CONFIG_RESPONSE=$(curl -s -X POST "$API_BASE/model-configurations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GPT-4",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-your-api-key"
  }')

CONFIG_ID=$(echo $CONFIG_RESPONSE | jq -r '.data.id')
echo "配置ID: $CONFIG_ID"


# 3. 创建会话
echo -e "\n3. 创建会话..."
SESSION_RESPONSE=$(curl -s -X POST "$API_BASE/chat/sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试对话",
    "modelName": "GPT-4"
  }')

SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.data.id')
echo "会话ID: $SESSION_ID"

# 4. 发送消息
echo -e "\n4. 发送消息..."
MESSAGE_RESPONSE=$(curl -s -X POST "$API_BASE/chat/sessions/$SESSION_ID/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请介绍一下自己",
    "options": {
      "modelName": "GPT-4",
      "temperature": 0.7
    }
  }')

echo "AI 回复:"
echo $MESSAGE_RESPONSE | jq -r '.data.assistantMessage.content'

# 5. 查询配置列表
echo -e "\n5. 查询配置列表..."
curl -s -X GET "$API_BASE/model-configurations" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.data[] | {name, model, modelProvider}'
```

#### Python 完整示例

```python
import requests
import json
from typing import Optional

class GenkitAPIClient:
    """Genkit API 客户端"""
    
    def __init__(self, base_url: str):
        self.base_url = base_url
        self.token: Optional[str] = None
    
    def login(self, email: str, password: str) -> bool:
        """登录"""
        url = f"{self.base_url}/auth/login"
        data = {"email": email, "password": password}
        
        response = requests.post(url, json=data)
        result = response.json()
        
        if result['code'] == 200:
            self.token = result['data']['accessToken']
            return True
        return False
    
    def _headers(self):
        """获取请求头"""
        return {
            "Authorization": f"Bearer {self.token}",
            "Content-Type": "application/json"
        }
    
    def create_model_config(self, name: str, model: str, provider: str, api_key: str, query_params: dict = None):
        """创建模型配置"""
        url = f"{self.base_url}/model-configurations"
        data = {
            "name": name,
            "model": model,
            "modelProvider": provider,
            "apiKey": api_key
        }
        if query_params:
            data["queryParams"] = query_params
        
        response = requests.post(url, headers=self._headers(), json=data)
        return response.json()
    
    def list_model_configs(self, page_no: int = 1, page_size: int = 10):
        """查询模型配置列表"""
        url = f"{self.base_url}/model-configurations"
        params = {"pageNo": page_no, "pageSize": page_size}
        
        response = requests.get(url, headers=self._headers(), params=params)
        return response.json()
    
    def create_session(self, title: str, model_name: str):
        """创建会话"""
        url = f"{self.base_url}/chat/sessions"
        data = {"title": title, "modelName": model_name}
        
        response = requests.post(url, headers=self._headers(), json=data)
        return response.json()
    
    def send_message(self, session_id: str, message: str, model_name: str = None, temperature: float = 0.7):
        """发送消息"""
        url = f"{self.base_url}/chat/sessions/{session_id}/messages"
        data = {"message": message}
        
        if model_name:
            data["options"] = {
                "modelName": model_name,
                "temperature": temperature
            }
        
        response = requests.post(url, headers=self._headers(), json=data)
        return response.json()


    def send_stream_message(self, session_id: str, message: str, model_name: str = None, temperature: float = 0.7):
        """发送流式消息"""
        url = f"{self.base_url}/chat/sessions/{session_id}/messages/stream"
        data = {"message": message}
        
        if model_name:
            data["options"] = {
                "modelName": model_name,
                "temperature": temperature
            }
        
        response = requests.post(url, headers=self._headers(), json=data, stream=True)
        
        full_content = ""
        for line in response.iter_lines():
            if line:
                line = line.decode('utf-8')
                if line.startswith('data: '):
                    event_data = json.loads(line[6:])
                    
                    if event_data['event'] == 'message':
                        delta = event_data['data']['delta']
                        full_content += delta
                        yield delta
                    elif event_data['event'] == 'done':
                        yield {'usage': event_data['data']['usage']}
        
        return full_content

# 使用示例
def main():
    # 初始化客户端
    client = GenkitAPIClient("https://api.example.com/api/v1")
    
    # 1. 登录
    print("1. 登录...")
    if not client.login("user@example.com", "your-password"):
        print("登录失败")
        return
    print("登录成功")
    
    # 2. 创建模型配置
    print("\n2. 创建模型配置...")
    config_result = client.create_model_config(
        name="GPT-4",
        model="gpt-4",
        provider="openai",
        api_key="sk-your-api-key",
        query_params={
            "defaultTemperature": 0.7,
            "defaultMaxTokens": 4096
        }
    )
    print(f"配置ID: {config_result['data']['id']}")
    
    # 3. 查询配置列表
    print("\n3. 查询配置列表...")
    configs = client.list_model_configs()
    for config in configs['data']['data']:
        print(f"  - {config['name']} ({config['modelProvider']})")
    
    # 4. 创建会话
    print("\n4. 创建会话...")
    session_result = client.create_session("测试对话", "GPT-4")
    session_id = session_result['data']['id']
    print(f"会话ID: {session_id}")
    
    # 5. 发送普通消息
    print("\n5. 发送普通消息...")
    message_result = client.send_message(
        session_id=session_id,
        message="你好，请介绍一下自己",
        model_name="GPT-4"
    )
    print(f"AI 回复: {message_result['data']['assistantMessage']['content']}")
    
    # 6. 发送流式消息
    print("\n6. 发送流式消息...")
    print("AI 回复: ", end='', flush=True)
    for chunk in client.send_stream_message(
        session_id=session_id,
        message="写一首关于春天的诗",
        model_name="GPT-4"
    ):
        if isinstance(chunk, dict) and 'usage' in chunk:
            print(f"\n\nToken 使用: {chunk['usage']}")
        else:
            print(chunk, end='', flush=True)

if __name__ == "__main__":
    main()
```

#### Node.js 完整示例

```javascript
const axios = require('axios');

class GenkitAPIClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
    this.token = null;
  }

  async login(email, password) {
    const response = await axios.post(`${this.baseUrl}/auth/login`, {
      email,
      password
    });
    
    if (response.data.code === 200) {
      this.token = response.data.data.accessToken;
      return true;
    }
    return false;
  }

  getHeaders() {
    return {
      'Authorization': `Bearer ${this.token}`,
      'Content-Type': 'application/json'
    };
  }

  async createModelConfig(name, model, provider, apiKey, queryParams = null) {
    const data = {
      name,
      model,
      modelProvider: provider,
      apiKey
    };
    if (queryParams) {
      data.queryParams = queryParams;
    }

    const response = await axios.post(
      `${this.baseUrl}/model-configurations`,
      data,
      { headers: this.getHeaders() }
    );
    return response.data;
  }

  async listModelConfigs(pageNo = 1, pageSize = 10) {
    const response = await axios.get(
      `${this.baseUrl}/model-configurations`,
      {
        headers: this.getHeaders(),
        params: { pageNo, pageSize }
      }
    );
    return response.data;
  }

  async createSession(title, modelName) {
    const response = await axios.post(
      `${this.baseUrl}/chat/sessions`,
      { title, modelName },
      { headers: this.getHeaders() }
    );
    return response.data;
  }

  async sendMessage(sessionId, message, modelName = null, temperature = 0.7) {
    const data = { message };
    if (modelName) {
      data.options = { modelName, temperature };
    }

    const response = await axios.post(
      `${this.baseUrl}/chat/sessions/${sessionId}/messages`,
      data,
      { headers: this.getHeaders() }
    );
    return response.data;
  }

  async sendStreamMessage(sessionId, message, modelName = null, temperature = 0.7, onChunk) {
    const data = { message };
    if (modelName) {
      data.options = { modelName, temperature };
    }

    const response = await axios.post(
      `${this.baseUrl}/chat/sessions/${sessionId}/messages/stream`,
      data,
      {
        headers: this.getHeaders(),
        responseType: 'stream'
      }
    );

    let fullContent = '';
    
    return new Promise((resolve, reject) => {
      response.data.on('data', (chunk) => {
        const lines = chunk.toString().split('\n');
        
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const eventData = JSON.parse(line.slice(6));
              
              if (eventData.event === 'message') {
                const delta = eventData.data.delta;
                fullContent += delta;
                if (onChunk) onChunk(delta);
              } else if (eventData.event === 'done') {
                if (onChunk) onChunk({ usage: eventData.data.usage });
                resolve(fullContent);
              }
            } catch (e) {
              // 忽略解析错误
            }
          }
        }
      });

      response.data.on('error', reject);
    });
  }
}

// 使用示例
async function main() {
  const client = new GenkitAPIClient('https://api.example.com/api/v1');

  try {
    // 1. 登录
    console.log('1. 登录...');
    await client.login('user@example.com', 'your-password');
    console.log('登录成功');

    // 2. 创建模型配置
    console.log('\n2. 创建模型配置...');
    const configResult = await client.createModelConfig(
      'GPT-4',
      'gpt-4',
      'openai',
      'sk-your-api-key',
      {
        defaultTemperature: 0.7,
        defaultMaxTokens: 4096
      }
    );
    console.log(`配置ID: ${configResult.data.id}`);

    // 3. 查询配置列表
    console.log('\n3. 查询配置列表...');
    const configs = await client.listModelConfigs();
    configs.data.data.forEach(config => {
      console.log(`  - ${config.name} (${config.modelProvider})`);
    });

    // 4. 创建会话
    console.log('\n4. 创建会话...');
    const sessionResult = await client.createSession('测试对话', 'GPT-4');
    const sessionId = sessionResult.data.id;
    console.log(`会话ID: ${sessionId}`);

    // 5. 发送普通消息
    console.log('\n5. 发送普通消息...');
    const messageResult = await client.sendMessage(
      sessionId,
      '你好，请介绍一下自己',
      'GPT-4'
    );
    console.log(`AI 回复: ${messageResult.data.assistantMessage.content}`);

    // 6. 发送流式消息
    console.log('\n6. 发送流式消息...');
    process.stdout.write('AI 回复: ');
    await client.sendStreamMessage(
      sessionId,
      '写一首关于春天的诗',
      'GPT-4',
      0.8,
      (chunk) => {
        if (typeof chunk === 'string') {
          process.stdout.write(chunk);
        } else if (chunk.usage) {
          console.log(`\n\nToken 使用: ${JSON.stringify(chunk.usage)}`);
        }
      }
    );

  } catch (error) {
    console.error('错误:', error.message);
  }
}

main();
```

## 高级用法

### 1. 批量创建配置

```bash
#!/bin/bash

TOKEN="your-jwt-token"
API_BASE="https://api.example.com/api/v1"

# 配置数组
declare -a CONFIGS=(
  "GPT-4:gpt-4:openai:sk-openai-key"
  "Gemini Pro:gemini-1.5-pro:googlegenai:AIza-google-key"
  "通义千问:qwen-turbo:bianlian:sk-bailian-key"
)

# 批量创建
for config in "${CONFIGS[@]}"; do
  IFS=':' read -r name model provider apikey <<< "$config"
  
  echo "创建配置: $name"
  curl -s -X POST "$API_BASE/model-configurations" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"$name\",
      \"model\": \"$model\",
      \"modelProvider\": \"$provider\",
      \"apiKey\": \"$apikey\"
    }" | jq -r '.data.id'
done
```

### 2. 模型切换测试

```python
def test_model_switching(client, session_id):
    """测试不同模型的响应"""
    models = ["GPT-4", "Gemini Pro", "通义千问 Turbo"]
    message = "用一句话介绍人工智能"
    
    results = {}
    for model in models:
        print(f"\n测试模型: {model}")
        result = client.send_message(
            session_id=session_id,
            message=message,
            model_name=model
        )
        
        if result['code'] == 200:
            content = result['data']['assistantMessage']['content']
            usage = result['data']['assistantMessage']['usage']
            
            results[model] = {
                'content': content,
                'tokens': usage['totalTokens']
            }
            
            print(f"回复: {content}")
            print(f"Token: {usage['totalTokens']}")
    
    return results
```

### 3. 性能监控

```javascript
class PerformanceMonitor {
  constructor(client) {
    this.client = client;
    this.metrics = [];
  }

  async sendMessageWithMetrics(sessionId, message, modelName) {
    const startTime = Date.now();
    
    try {
      const result = await this.client.sendMessage(sessionId, message, modelName);
      const endTime = Date.now();
      
      const metric = {
        modelName,
        duration: endTime - startTime,
        tokens: result.data.assistantMessage.usage.totalTokens,
        success: true,
        timestamp: new Date().toISOString()
      };
      
      this.metrics.push(metric);
      return result;
    } catch (error) {
      const endTime = Date.now();
      
      this.metrics.push({
        modelName,
        duration: endTime - startTime,
        success: false,
        error: error.message,
        timestamp: new Date().toISOString()
      });
      
      throw error;
    }
  }

  getAverageMetrics(modelName) {
    const modelMetrics = this.metrics.filter(m => m.modelName === modelName && m.success);
    
    if (modelMetrics.length === 0) return null;
    
    const avgDuration = modelMetrics.reduce((sum, m) => sum + m.duration, 0) / modelMetrics.length;
    const avgTokens = modelMetrics.reduce((sum, m) => sum + m.tokens, 0) / modelMetrics.length;
    
    return {
      modelName,
      count: modelMetrics.length,
      avgDuration: Math.round(avgDuration),
      avgTokens: Math.round(avgTokens)
    };
  }

  printReport() {
    console.log('\n=== 性能报告 ===');
    const models = [...new Set(this.metrics.map(m => m.modelName))];
    
    models.forEach(model => {
      const metrics = this.getAverageMetrics(model);
      if (metrics) {
        console.log(`\n${metrics.modelName}:`);
        console.log(`  请求次数: ${metrics.count}`);
        console.log(`  平均延迟: ${metrics.avgDuration}ms`);
        console.log(`  平均Token: ${metrics.avgTokens}`);
      }
    });
  }
}

// 使用示例
const monitor = new PerformanceMonitor(client);

// 发送多个请求
for (let i = 0; i < 10; i++) {
  await monitor.sendMessageWithMetrics(sessionId, `测试消息 ${i}`, 'GPT-4');
}

// 打印报告
monitor.printReport();
```

### 4. 重试机制

```python
import time
from typing import Optional, Callable

def retry_with_backoff(
    func: Callable,
    max_retries: int = 3,
    initial_delay: float = 1.0,
    backoff_factor: float = 2.0
) -> Optional[any]:
    """带指数退避的重试机制"""
    delay = initial_delay
    
    for attempt in range(max_retries):
        try:
            return func()
        except Exception as e:
            if attempt == max_retries - 1:
                print(f"重试 {max_retries} 次后仍然失败: {str(e)}")
                raise
            
            print(f"尝试 {attempt + 1} 失败，{delay}秒后重试...")
            time.sleep(delay)
            delay *= backoff_factor
    
    return None

# 使用示例
def send_with_retry(client, session_id, message, model_name):
    """带重试的消息发送"""
    return retry_with_backoff(
        lambda: client.send_message(session_id, message, model_name),
        max_retries=3,
        initial_delay=1.0,
        backoff_factor=2.0
    )

# 调用
result = send_with_retry(client, session_id, "你好", "GPT-4")
```

### 5. 并发请求

```python
import asyncio
import aiohttp
from typing import List, Dict

class AsyncGenkitClient:
    """异步 Genkit 客户端"""
    
    def __init__(self, base_url: str, token: str):
        self.base_url = base_url
        self.token = token
    
    async def send_message(self, session: aiohttp.ClientSession, session_id: str, message: str, model_name: str) -> Dict:
        """异步发送消息"""
        url = f"{self.base_url}/chat/sessions/{session_id}/messages"
        headers = {
            "Authorization": f"Bearer {self.token}",
            "Content-Type": "application/json"
        }
        data = {
            "message": message,
            "options": {"modelName": model_name}
        }
        
        async with session.post(url, headers=headers, json=data) as response:
            return await response.json()
    
    async def send_multiple_messages(self, session_id: str, messages: List[tuple]) -> List[Dict]:
        """并发发送多条消息"""
        async with aiohttp.ClientSession() as session:
            tasks = [
                self.send_message(session, session_id, msg, model)
                for msg, model in messages
            ]
            return await asyncio.gather(*tasks)

# 使用示例
async def main():
    client = AsyncGenkitClient(
        "https://api.example.com/api/v1",
        "your-jwt-token"
    )
    
    # 准备多条消息
    messages = [
        ("介绍一下人工智能", "GPT-4"),
        ("什么是机器学习", "Gemini Pro"),
        ("深度学习的应用", "通义千问 Turbo")
    ]
    
    # 并发发送
    results = await client.send_multiple_messages(
        "770e8400-e29b-41d4-a716-446655440002",
        messages
    )
    
    # 打印结果
    for i, result in enumerate(results):
        if result['code'] == 200:
            content = result['data']['assistantMessage']['content']
            print(f"\n消息 {i+1} 回复: {content[:100]}...")

# 运行
asyncio.run(main())
```

## 最佳实践

### 1. Token 管理

```javascript
class TokenManager {
  constructor() {
    this.accessToken = null;
    this.refreshToken = null;
    this.expiresAt = null;
  }

  setTokens(accessToken, refreshToken, expiresIn) {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
    this.expiresAt = Date.now() + (expiresIn * 1000);
  }

  isTokenExpired() {
    return Date.now() >= this.expiresAt - 60000; // 提前1分钟刷新
  }

  async refreshIfNeeded(client) {
    if (this.isTokenExpired()) {
      console.log('Token 即将过期，刷新中...');
      const result = await client.refreshToken(this.refreshToken);
      if (result.code === 200) {
        this.setTokens(
          result.data.accessToken,
          result.data.refreshToken,
          result.data.expiresIn
        );
      }
    }
  }

  getAccessToken() {
    return this.accessToken;
  }
}
```

### 2. 请求限流

```python
import time
from collections import deque

class RateLimiter:
    """请求限流器"""
    
    def __init__(self, max_requests: int, time_window: int):
        """
        Args:
            max_requests: 时间窗口内最大请求数
            time_window: 时间窗口（秒）
        """
        self.max_requests = max_requests
        self.time_window = time_window
        self.requests = deque()
    
    def acquire(self):
        """获取请求许可"""
        now = time.time()
        
        # 移除过期的请求记录
        while self.requests and self.requests[0] < now - self.time_window:
            self.requests.popleft()
        
        # 检查是否超过限制
        if len(self.requests) >= self.max_requests:
            # 计算需要等待的时间
            wait_time = self.requests[0] + self.time_window - now
            print(f"达到速率限制，等待 {wait_time:.2f} 秒...")
            time.sleep(wait_time)
            return self.acquire()
        
        # 记录本次请求
        self.requests.append(now)
        return True

# 使用示例
limiter = RateLimiter(max_requests=10, time_window=60)  # 每分钟最多10个请求

for i in range(20):
    limiter.acquire()
    result = client.send_message(session_id, f"消息 {i}", "GPT-4")
    print(f"发送消息 {i}")
```

### 3. 响应缓存

```python
import hashlib
import json
from typing import Optional, Dict
from datetime import datetime, timedelta

class ResponseCache:
    """响应缓存"""
    
    def __init__(self, ttl_seconds: int = 3600):
        """
        Args:
            ttl_seconds: 缓存过期时间（秒）
        """
        self.cache: Dict[str, tuple] = {}
        self.ttl = timedelta(seconds=ttl_seconds)
    
    def _generate_key(self, session_id: str, message: str, model_name: str) -> str:
        """生成缓存键"""
        data = f"{session_id}:{message}:{model_name}"
        return hashlib.md5(data.encode()).hexdigest()
    
    def get(self, session_id: str, message: str, model_name: str) -> Optional[Dict]:
        """获取缓存"""
        key = self._generate_key(session_id, message, model_name)
        
        if key in self.cache:
            value, timestamp = self.cache[key]
            if datetime.now() - timestamp < self.ttl:
                print(f"缓存命中: {key[:8]}...")
                return value
            else:
                # 缓存过期，删除
                del self.cache[key]
        
        return None
    
    def set(self, session_id: str, message: str, model_name: str, response: Dict):
        """设置缓存"""
        key = self._generate_key(session_id, message, model_name)
        self.cache[key] = (response, datetime.now())
    
    def clear(self):
        """清空缓存"""
        self.cache.clear()

# 使用示例
cache = ResponseCache(ttl_seconds=3600)

def send_message_with_cache(client, session_id, message, model_name):
    """带缓存的消息发送"""
    # 尝试从缓存获取
    cached_response = cache.get(session_id, message, model_name)
    if cached_response:
        return cached_response
    
    # 发送请求
    response = client.send_message(session_id, message, model_name)
    
    # 缓存响应
    if response['code'] == 200:
        cache.set(session_id, message, model_name, response)
    
    return response

# 第一次调用 - 发送请求
result1 = send_message_with_cache(client, session_id, "你好", "GPT-4")

# 第二次调用 - 使用缓存
result2 = send_message_with_cache(client, session_id, "你好", "GPT-4")
```

### 4. 日志记录

```python
import logging
from datetime import datetime

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('genkit_api.log'),
        logging.StreamHandler()
    ]
)

logger = logging.getLogger('GenkitAPI')

class LoggingClient:
    """带日志的客户端包装器"""
    
    def __init__(self, client):
        self.client = client
    
    def send_message(self, session_id, message, model_name):
        """发送消息（带日志）"""
        logger.info(f"发送消息 - 会话: {session_id}, 模型: {model_name}")
        logger.debug(f"消息内容: {message[:100]}...")
        
        start_time = datetime.now()
        
        try:
            result = self.client.send_message(session_id, message, model_name)
            
            duration = (datetime.now() - start_time).total_seconds()
            
            if result['code'] == 200:
                usage = result['data']['assistantMessage']['usage']
                logger.info(
                    f"消息发送成功 - 耗时: {duration:.2f}s, "
                    f"Token: {usage['totalTokens']}"
                )
            else:
                logger.error(f"消息发送失败 - 错误: {result['message']}")
            
            return result
        
        except Exception as e:
            duration = (datetime.now() - start_time).total_seconds()
            logger.error(
                f"消息发送异常 - 耗时: {duration:.2f}s, "
                f"错误: {str(e)}"
            )
            raise

# 使用示例
logging_client = LoggingClient(client)
result = logging_client.send_message(session_id, "你好", "GPT-4")
```

## 故障排查

### 常见问题诊断脚本

```bash
#!/bin/bash

# Genkit API 诊断脚本

API_BASE="https://api.example.com/api/v1"
TOKEN="your-jwt-token"

echo "=== Genkit API 诊断 ==="

# 1. 检查 API 连通性
echo -e "\n1. 检查 API 连通性..."
if curl -s -f "$API_BASE/health" > /dev/null; then
    echo "✓ API 可访问"
else
    echo "✗ API 不可访问"
    exit 1
fi

# 2. 检查认证
echo -e "\n2. 检查认证..."
AUTH_RESPONSE=$(curl -s -X GET "$API_BASE/model-configurations" \
    -H "Authorization: Bearer $TOKEN" \
    -w "\n%{http_code}")

HTTP_CODE=$(echo "$AUTH_RESPONSE" | tail -n 1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ 认证成功"
else
    echo "✗ 认证失败 (HTTP $HTTP_CODE)"
    exit 1
fi

# 3. 检查模型配置
echo -e "\n3. 检查模型配置..."
CONFIGS=$(curl -s -X GET "$API_BASE/model-configurations" \
    -H "Authorization: Bearer $TOKEN" | jq -r '.data.data | length')

if [ "$CONFIGS" -gt 0 ]; then
    echo "✓ 找到 $CONFIGS 个模型配置"
    curl -s -X GET "$API_BASE/model-configurations" \
        -H "Authorization: Bearer $TOKEN" | \
        jq -r '.data.data[] | "  - \(.name) (\(.modelProvider))"'
else
    echo "✗ 未找到模型配置"
fi

# 4. 测试消息发送
echo -e "\n4. 测试消息发送..."
# 这里需要一个有效的会话ID
# SESSION_ID="your-session-id"
# TEST_RESULT=$(curl -s -X POST "$API_BASE/chat/sessions/$SESSION_ID/messages" \
#     -H "Authorization: Bearer $TOKEN" \
#     -H "Content-Type: application/json" \
#     -d '{"message":"测试"}')
# 
# if echo "$TEST_RESULT" | jq -e '.code == 200' > /dev/null; then
#     echo "✓ 消息发送成功"
# else
#     echo "✗ 消息发送失败"
#     echo "$TEST_RESULT" | jq -r '.message'
# fi

echo -e "\n=== 诊断完成 ==="
```

## 相关资源

- [多提供商使用指南](./MULTI_PROVIDER_GUIDE.md)
- [配置指南](./CONFIGURATION_GUIDE.md)
- [故障排查指南](./TROUBLESHOOTING.md)
- [API 参考文档](./API_REFERENCE.md)

## 获取帮助

如果遇到问题，请：

1. 查看 [故障排查指南](./TROUBLESHOOTING.md)
2. 检查 [常见问题](./MULTI_PROVIDER_GUIDE.md#常见问题)
3. 查看系统日志获取详细错误信息
4. 联系技术支持团队

---

**最后更新**: 2025-12-01  
**版本**: 1.0.0
