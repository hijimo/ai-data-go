# Token 黑名单机制

## 概述

Token 黑名单机制用于在用户注销时立即撤销 Access Token，防止已注销的 token 继续被使用。该机制使用 Redis 存储已撤销的 token，并在 JWT 验证中检查黑名单。

## 功能特性

- **即时撤销**：用户注销时，Access Token 立即加入黑名单
- **自动过期**：黑名单条目会在 token 过期时自动清理（由 Redis TTL 机制处理）
- **高性能**：使用 Redis 内存存储，检查速度快
- **容错设计**：Redis 不可用时不会阻止正常请求，但黑名单功能会被禁用
- **可选启用**：可以通过配置开关控制是否启用黑名单功能

## 工作原理

### 1. Token 加入黑名单

当用户注销时：

1. 从 Authorization 头提取 Access Token
2. 验证 token 并获取过期时间
3. 计算 TTL（token 过期时间 - 当前时间）
4. 将 token 的 SHA256 哈希值存储到 Redis，设置 TTL
5. 撤销 Refresh Token

### 2. Token 黑名单检查

当请求需要认证时：

1. JWT 中间件从 Authorization 头提取 token
2. 计算 token 的 SHA256 哈希值
3. 检查 Redis 中是否存在该哈希值
4. 如果存在，拒绝请求并返回 401 错误
5. 如果不存在，继续验证 JWT 签名和过期时间

### 3. 自动清理

- Redis 会自动清理过期的键（基于 TTL）
- 无需手动清理黑名单条目
- 不会占用过多内存

## 配置说明

### 环境变量

在 `.env` 文件中配置以下参数：

```bash
# Redis 配置
REDIS_ENABLED=true              # 是否启用 Redis
REDIS_HOST=localhost            # Redis 主机
REDIS_PORT=6379                 # Redis 端口
REDIS_PASSWORD=                 # Redis 密码（如果没有则留空）
REDIS_DB=0                      # Redis 数据库编号（0-15）

# Token 黑名单配置
ENABLE_TOKEN_BLACKLIST=true     # 是否启用 Token 黑名单
```

### 配置说明

- **REDIS_ENABLED**: 控制是否启用 Redis 连接。如果设置为 `false`，黑名单功能将不可用
- **ENABLE_TOKEN_BLACKLIST**: 控制是否启用 Token 黑名单功能。即使 Redis 可用，也可以通过此开关禁用黑名单

## 使用示例

### 用户注销

```bash
# 注销请求需要同时提供 Access Token 和 Refresh Token
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "<refresh_token>"
  }'
```

注销成功后：

- Access Token 被加入黑名单
- Refresh Token 被撤销
- 记录注销审计日志

### 使用已撤销的 Token

```bash
# 尝试使用已注销的 token 访问受保护的资源
curl -X GET http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <revoked_access_token>"
```

响应：

```json
{
  "code": 401,
  "message": "身份认证已被撤销，请重新登录",
  "data": null
}
```

## Redis 键格式

黑名单条目在 Redis 中的键格式：

```
token:blacklist:<token_sha256_hash>
```

- 键名前缀：`token:blacklist:`
- 键值：token 的 SHA256 哈希值（64 个十六进制字符）
- 存储值：撤销时间戳（Unix 时间戳）
- TTL：token 的剩余有效期

示例：

```
token:blacklist:a1b2c3d4e5f6...  -> 1697712000
TTL: 3600 秒
```

## 安全考虑

### 1. 哈希存储

- Token 不以明文形式存储在 Redis 中
- 使用 SHA256 哈希算法计算 token 的哈希值
- 即使 Redis 数据泄露，也无法还原原始 token

### 2. 容错设计

- Redis 连接失败时，黑名单检查会被跳过
- 不会因为 Redis 故障导致所有请求被拒绝
- 记录错误日志但不阻止正常流程

### 3. 性能优化

- 使用 Redis 内存存储，检查速度快（< 1ms）
- 自动过期机制，无需手动清理
- 不会影响正常请求的性能

## 故障排查

### Redis 连接失败

**症状**：日志中出现 "Redis 连接失败" 警告

**原因**：

- Redis 服务未启动
- Redis 连接配置错误
- 网络问题

**解决方案**：

1. 检查 Redis 服务是否运行：`redis-cli ping`
2. 验证 Redis 配置（主机、端口、密码）
3. 检查防火墙和网络连接
4. 如果不需要黑名单功能，可以设置 `REDIS_ENABLED=false`

### 黑名单检查失败

**症状**：日志中出现 "检查 token 黑名单状态失败" 错误

**原因**：

- Redis 临时不可用
- Redis 连接超时

**影响**：

- 黑名单检查会被跳过
- 已撤销的 token 可能仍然有效（直到过期）

**解决方案**：

1. 检查 Redis 服务状态
2. 增加 Redis 连接超时时间
3. 考虑使用 Redis 集群提高可用性

### Token 未被撤销

**症状**：注销后 token 仍然可以使用

**可能原因**：

1. Redis 未启用：检查 `REDIS_ENABLED` 配置
2. 黑名单功能未启用：检查 `ENABLE_TOKEN_BLACKLIST` 配置
3. Redis 连接失败：查看日志中的错误信息
4. 注销请求未提供 Access Token：确保在 Authorization 头中包含 token

**解决方案**：

1. 确认配置正确
2. 检查 Redis 服务状态
3. 查看应用日志获取详细错误信息

## 性能指标

### Redis 操作性能

- **写入操作**（加入黑名单）：< 1ms
- **读取操作**（检查黑名单）：< 1ms
- **内存占用**：每个黑名单条目约 100 字节

### 估算

假设：

- 每天 10,000 次注销
- Access Token 有效期 60 分钟
- 平均每分钟有 ~7 个活跃的黑名单条目

内存占用：

```
7 条目 × 100 字节 = 700 字节
```

即使在高负载场景下，黑名单功能的内存占用也非常小。

## 最佳实践

### 1. 生产环境配置

- 使用 Redis 集群或哨兵模式提高可用性
- 配置 Redis 持久化（AOF 或 RDB）
- 设置合理的 Redis 内存限制和淘汰策略
- 监控 Redis 性能指标

### 2. 安全建议

- 使用强密码保护 Redis
- 限制 Redis 网络访问（仅允许应用服务器连接）
- 定期更新 Redis 版本
- 启用 Redis TLS 加密（如果需要）

### 3. 监控和告警

建议监控以下指标：

- Redis 连接状态
- 黑名单检查失败率
- Redis 内存使用率
- Redis 操作延迟

## 与其他功能的集成

### 1. Refresh Token 轮换

- 注销时同时撤销 Refresh Token
- 防止使用旧的 Refresh Token 获取新的 Access Token

### 2. 审计日志

- 注销操作会记录审计日志
- 包含用户 ID、租户 ID、IP 地址等信息

### 3. 多设备登录

- 可以扩展为支持单设备注销
- 或者注销所有设备（撤销用户的所有 token）

## 未来扩展

### 1. 批量撤销

支持批量撤销用户的所有 token：

```go
// 撤销用户的所有 Access Token
func (s *authService) LogoutAllDevices(ctx context.Context, userID string) error {
    // 1. 获取用户的所有活跃 Refresh Token
    // 2. 遍历并撤销每个 Refresh Token
    // 3. 将对应的 Access Token 加入黑名单
}
```

### 2. Token 使用统计

记录 token 的使用情况：

- 最后使用时间
- 使用次数
- 使用的 IP 地址

### 3. 异常检测

检测异常的 token 使用模式：

- 短时间内多次使用已撤销的 token
- 从不同 IP 地址使用同一 token
- 触发安全告警

## 参考资料

- [Redis 官方文档](https://redis.io/documentation)
- [JWT 最佳实践](https://tools.ietf.org/html/rfc8725)
- [OWASP Token 管理指南](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
