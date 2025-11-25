# 配置解析逻辑使用指南

## 概述

配置解析逻辑允许 Genkit Client 从数据库动态获取模型配置，并根据租户 ID 和模型名称初始化相应的 Genkit 实例。

## 基本用法

### 1. 创建带仓储的客户端

```go
import (
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/repository"
)

// 创建模型配置仓储
configRepo := repository.NewModelConfigurationRepository(db)

// 创建 Genkit 客户端并注入仓储
client := genkit.NewClientWithRepo(configRepo)
```

### 2. 使用动态配置

```go
// 调用 getOrInitGenkit 方法
// 该方法会自动从数据库查询配置、解析配置、初始化 Genkit 实例
tenantID := "550e8400-e29b-41d4-a716-446655440000"
modelName := "gemini-pro"

g, genkitConfig, err := client.getOrInitGenkit(ctx, tenantID, modelName)
if err != nil {
    // 处理错误
    log.Fatalf("初始化 Genkit 失败: %v", err)
}

// 使用 Genkit 实例和配置
fmt.Printf("模型: %s\n", genkitConfig.Model)
fmt.Printf("温度: %.2f\n", genkitConfig.DefaultTemperature)
fmt.Printf("最大 Token: %d\n", genkitConfig.DefaultMaxTokens)
```

## 配置格式

### 数据库表结构

```sql
CREATE TABLE model_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    model VARCHAR(255) NOT NULL,
    model_provider VARCHAR(50) NOT NULL,
    api_key TEXT NOT NULL,
    query_params JSONB,  -- 配置 JSON
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### QueryParams JSON 格式

#### Google AI 配置

```json
{
  "model": "gemini-1.5-pro",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

#### Azure OpenAI 配置

```json
{
  "model": "gpt-4",
  "azureEndpoint": "https://your-resource.openai.azure.com",
  "azureDeployment": "gpt-4",
  "azureApiVersion": "2024-02-15-preview",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

#### 百炼配置

```json
{
  "model": "qwen-turbo",
  "bailianEndpoint": "https://dashscope.aliyuncs.com",
  "bailianWorkspace": "default",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

## 配置解析流程

```
1. 从数据库查询配置
   ↓
2. 检查模型是否启用
   ↓
3. 解析 QueryParams JSON
   ↓
4. 验证配置完整性
   ↓
5. 根据提供商类型初始化插件
   ↓
6. 缓存 Genkit 实例
   ↓
7. 返回实例和配置
```

## 缓存机制

### 缓存键格式

```
{tenantID}_{modelName}
```

例如：

```
550e8400-e29b-41d4-a716-446655440000_gemini-pro
```

### 缓存操作

```go
// 获取缓存大小
size := client.GetCacheSize()

// 清理指定缓存
client.ClearCache(tenantID, modelName)

// 清理所有缓存
client.ClearAllCache()

// 关闭客户端（会清理所有缓存）
client.Close()
```

## 错误处理

### 常见错误

1. **配置不存在**

```go
err := "获取模型配置失败: record not found"
```

2. **模型已禁用**

```go
err := "模型已禁用: gemini-pro"
```

3. **无效的 JSON 配置**

```go
err := "解析模型配置失败: invalid character '}' looking for beginning of object key string"
```

4. **配置验证失败**

```go
err := "配置验证失败: Azure OpenAI 配置缺少必需字段: azureEndpoint"
```

5. **不支持的提供商**

```go
err := "不支持的提供商类型: unknown-provider"
```

### 错误处理示例

```go
g, genkitConfig, err := client.getOrInitGenkit(ctx, tenantID, modelName)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "模型已禁用"):
        // 处理模型禁用的情况
        return fmt.Errorf("模型不可用: %w", err)
    
    case strings.Contains(err.Error(), "配置不存在"):
        // 处理配置不存在的情况
        return fmt.Errorf("请先配置模型: %w", err)
    
    case strings.Contains(err.Error(), "配置验证失败"):
        // 处理配置验证失败的情况
        return fmt.Errorf("配置错误: %w", err)
    
    default:
        // 处理其他错误
        return fmt.Errorf("初始化失败: %w", err)
    }
}
```

## 并发安全

配置解析逻辑使用双重检查锁定模式确保并发安全：

```go
// 第一次检查（读锁）
c.mu.RLock()
g, exists := c.instances[cacheKey]
c.mu.RUnlock()

if exists {
    return g, config, nil
}

// 获取写锁
c.mu.Lock()
defer c.mu.Unlock()

// 第二次检查
if g, exists := c.instances[cacheKey]; exists {
    return g, config, nil
}

// 初始化新实例
// ...
```

## 性能优化建议

1. **预热缓存**: 在应用启动时预先初始化常用模型

```go
// 预热常用模型
commonModels := []struct {
    tenantID  string
    modelName string
}{
    {"tenant-1", "gemini-pro"},
    {"tenant-1", "gpt-4"},
}

for _, m := range commonModels {
    _, _, err := client.getOrInitGenkit(ctx, m.tenantID, m.modelName)
    if err != nil {
        log.Printf("预热失败: %v", err)
    }
}
```

2. **定期清理缓存**: 避免内存泄漏

```go
// 每小时清理一次缓存
ticker := time.NewTicker(1 * time.Hour)
go func() {
    for range ticker.C {
        client.ClearAllCache()
    }
}()
```

3. **监控缓存大小**: 及时发现问题

```go
// 定期记录缓存大小
go func() {
    for {
        size := client.GetCacheSize()
        log.Printf("当前缓存大小: %d", size)
        time.Sleep(5 * time.Minute)
    }
}()
```

## 测试

### 单元测试示例

```go
func TestParseConfiguration(t *testing.T) {
    // 创建模拟仓储
    mockRepo := new(MockModelConfigurationRepository)
    
    // 准备测试数据
    queryParams := `{"model":"gemini-1.5-pro","defaultTemperature":0.7}`
    modelConfig := &model.ModelConfiguration{
        Model:         "gemini-1.5-pro",
        ModelProvider: "googlegenai",
        APIKey:        "test-key",
        QueryParams:   &queryParams,
        IsEnabled:     true,
    }
    
    // 设置模拟期望
    mockRepo.On("GetByTenantAndModel", ctx, tenantID, modelName).
        Return(modelConfig, nil)
    
    // 创建客户端
    client := genkit.NewClientWithRepo(mockRepo)
    
    // 测试
    g, config, err := client.getOrInitGenkit(ctx, tenantID.String(), modelName)
    
    // 验证
    assert.NoError(t, err)
    assert.NotNil(t, g)
    assert.Equal(t, "gemini-1.5-pro", config.Model)
    assert.Equal(t, 0.7, config.DefaultTemperature)
}
```

## 注意事项

1. **API Key 安全**: 确保 API key 在数据库中加密存储
2. **配置验证**: 在保存配置到数据库前进行验证
3. **错误日志**: 记录详细的错误信息用于排查问题
4. **缓存更新**: 配置更新后需要清理对应的缓存
5. **并发限制**: 考虑限制并发初始化的数量，避免资源耗尽

## 下一步

- 实现 Azure OpenAI 插件创建逻辑
- 实现百炼插件创建逻辑
- 添加配置热更新功能
- 实现配置版本管理
