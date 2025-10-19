# 性能监控快速参考

## 快速开始

### 1. 环境变量配置

```bash
# .env
SLOW_QUERY_THRESHOLD=200
ALERT_CHECK_INTERVAL=1
ENABLE_SLOW_QUERY_LOG=true
ENABLE_MONITORING=true
```

### 2. 初始化代码

```go
// 创建告警管理器
alertManager := monitoring.NewAlertManager(1 * time.Minute)

// 添加默认规则
for _, rule := range monitoring.GetDefaultAlertRules() {
    alertManager.AddRule(rule)
}

// 添加处理器
alertManager.AddHandler(&monitoring.LogAlertHandler{logger: logger})

// 启动
alertManager.Start()
defer alertManager.Stop()
```

### 3. 记录指标

```go
// 登录
start := time.Now()
// ... 执行登录
monitoring.GetMetrics().RecordLoginAttempt(success, time.Since(start))

// Token 刷新
monitoring.GetMetrics().RecordTokenRefresh(success, duration)

// 注销
monitoring.GetMetrics().RecordLogout()

// 安全事件
monitoring.GetMetrics().RecordInvalidToken()
monitoring.GetMetrics().RecordExpiredToken()
monitoring.GetMetrics().RecordBruteForceAttempt()
```

## API 端点

### 获取指标

```bash
GET /api/v1/monitoring/metrics
Authorization: Bearer <admin_token>
```

### 获取告警

```bash
GET /api/v1/monitoring/alerts
Authorization: Bearer <admin_token>
```

### 清空告警

```bash
DELETE /api/v1/monitoring/alerts
Authorization: Bearer <admin_token>
```

### 重置指标

```bash
POST /api/v1/monitoring/metrics/reset
Authorization: Bearer <admin_token>
```

### 健康检查

```bash
GET /api/v1/monitoring/health
```

## 指标说明

| 指标 | 说明 |
|------|------|
| login_attempts | 登录尝试总数 |
| login_successes | 登录成功总数 |
| login_failures | 登录失败总数 |
| login_success_rate | 登录成功率 (%) |
| token_refreshes | Token 刷新总数 |
| token_refresh_failures | Token 刷新失败总数 |
| logouts | 注销总数 |
| avg_login_duration_ms | 平均登录时间 (ms) |
| p95_login_duration_ms | P95 登录时间 (ms) |
| avg_refresh_duration_ms | 平均刷新时间 (ms) |
| p95_refresh_duration_ms | P95 刷新时间 (ms) |
| slow_queries | 慢查询总数 |
| db_errors | 数据库错误总数 |
| invalid_tokens | 无效 Token 总数 |
| expired_tokens | 过期 Token 总数 |
| revoked_tokens | 已撤销 Token 总数 |
| brute_force_attempts | 暴力破解尝试总数 |
| active_tenants | 活跃租户数 |
| active_users | 活跃用户数 |

## 默认告警规则

| 规则 | 阈值 | 级别 |
|------|------|------|
| 高登录失败率 | < 70% | Warning |
| 大量登录失败 | > 50 次 | Critical |
| Token 刷新失败率高 | > 10% | Warning |
| 慢查询过多 | > 20 次 | Warning |
| 数据库错误频繁 | > 10 次 | Critical |
| 无效 Token 过多 | > 30 次 | Warning |
| 暴力破解攻击 | > 20 次 | Critical |
| 登录响应时间过长 | > 1000ms | Warning |
| Token 刷新响应时间过长 | > 500ms | Warning |

## 测试脚本

```bash
# 运行监控测试
./scripts/test_monitoring.sh

# 使用自定义 URL 和 Token
BASE_URL=http://localhost:8080 ADMIN_TOKEN=your_token ./scripts/test_monitoring.sh
```

## 常用命令

```bash
# 查看指标
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/monitoring/metrics | jq

# 查看告警
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/monitoring/alerts | jq

# 健康检查
curl http://localhost:8080/api/v1/monitoring/health | jq

# 清空告警
curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/monitoring/alerts

# 重置指标
curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/monitoring/metrics/reset
```

## 集成示例

### 在 AuthService 中

```go
func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
    start := time.Now()
    defer func() {
        success := err == nil
        monitoring.GetMetrics().RecordLoginAttempt(success, time.Since(start))
    }()
    
    // 登录逻辑...
}
```

### 在 JWT 中间件中

```go
func JWTAuth(tokenService TokenService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims, err := tokenService.ValidateAccessToken(token)
            if err != nil {
                if errors.Is(err, jwt.ErrTokenExpired) {
                    monitoring.GetMetrics().RecordExpiredToken()
                } else {
                    monitoring.GetMetrics().RecordInvalidToken()
                }
                // 返回错误...
            }
            // 继续处理...
        })
    }
}
```

### 配置慢查询日志

```go
// 方式1：使用 Logger
slowQueryLogger := monitoring.NewSlowQueryLogger(monitoring.SlowQueryConfig{
    SlowThreshold: 200 * time.Millisecond,
    Enabled:       true,
    Logger:        gormLogger,
})

db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: slowQueryLogger,
})

// 方式2：使用中间件
slowQueryMiddleware := monitoring.NewSlowQueryMiddleware(monitoring.SlowQueryConfig{
    SlowThreshold: 200 * time.Millisecond,
    Enabled:       true,
    Logger:        gormLogger,
})

db.Use(slowQueryMiddleware)
```

## 自定义告警

```go
// 添加自定义规则
customRule := monitoring.AlertRule{
    Name:        "自定义告警",
    Description: "描述",
    Metric:      "metric_name",
    Threshold:   100.0,
    Level:       monitoring.AlertLevelWarning,
    Condition: func(s monitoring.MetricsSnapshot) (bool, float64) {
        value := float64(s.LoginAttempts)
        return value > 100.0, value
    },
}
alertManager.AddRule(customRule)

// 添加自定义处理器
type CustomHandler struct{}

func (h *CustomHandler) HandleAlert(alert monitoring.Alert) error {
    // 自定义处理逻辑
    return nil
}

alertManager.AddHandler(&CustomHandler{})
```

## 故障排查

### 指标不更新

- 检查 `ENABLE_MONITORING=true`
- 检查是否调用了记录方法
- 查看日志错误

### 告警不触发

- 检查告警管理器是否启动
- 检查规则条件
- 查看日志

### 慢查询不记录

- 检查 `ENABLE_SLOW_QUERY_LOG=true`
- 检查阈值设置
- 检查 GORM Logger 配置

## 性能影响

- CPU 开销: < 1%
- 内存开销: 10-20MB
- 响应时间影响: < 1ms

## 安全注意事项

1. 监控端点需要管理员权限
2. 不要记录敏感信息（密码、Token）
3. 注意日志权限控制
4. 告警通知不要包含敏感信息
