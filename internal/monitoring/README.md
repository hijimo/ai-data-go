# 性能监控模块

本模块提供了认证系统的性能监控、慢查询日志和告警功能。

## 功能特性

### 1. 性能指标收集

自动收集以下指标：

#### 认证指标

- 登录尝试总数
- 登录成功/失败次数
- 登录成功率
- Token 刷新次数
- Token 刷新失败次数
- 注销次数

#### 性能指标

- 平均登录响应时间
- P95 登录响应时间
- 平均 Token 刷新响应时间
- P95 Token 刷新响应时间

#### 数据库指标

- 慢查询总数
- 数据库错误总数

#### 安全指标

- 无效 Token 总数
- 过期 Token 总数
- 已撤销 Token 总数
- 暴力破解尝试总数

#### 租户指标

- 活跃租户数
- 活跃用户数

### 2. 慢查询日志

自动记录超过阈值的数据库查询：

- 默认阈值：200ms
- 自动记录慢查询到日志
- 自动更新慢查询计数器
- 支持自定义阈值

### 3. 告警规则

内置多种告警规则：

- **高登录失败率**：登录失败率超过 30%
- **大量登录失败**：登录失败次数超过 50 次
- **Token 刷新失败率高**：刷新失败率超过 10%
- **慢查询过多**：慢查询数量超过 20 次
- **数据库错误频繁**：数据库错误超过 10 次
- **无效 Token 过多**：无效 Token 超过 30 次
- **暴力破解攻击**：暴力破解尝试超过 20 次
- **登录响应时间过长**：P95 响应时间超过 1 秒
- **Token 刷新响应时间过长**：P95 响应时间超过 500ms

## 使用方法

### 1. 初始化监控

在应用启动时初始化监控模块：

```go
import (
    "github.com/your-org/genkit-ai-service/internal/monitoring"
    "github.com/your-org/genkit-ai-service/internal/logger"
)

// 创建告警管理器
alertManager := monitoring.NewAlertManager(1 * time.Minute)

// 添加默认告警规则
for _, rule := range monitoring.GetDefaultAlertRules() {
    alertManager.AddRule(rule)
}

// 添加日志告警处理器
logHandler := &monitoring.LogAlertHandler{
    logger: logger.GetLogger(),
}
alertManager.AddHandler(logHandler)

// 启动告警检查
alertManager.Start()
defer alertManager.Stop()
```

### 2. 配置慢查询日志

在数据库初始化时配置慢查询日志：

```go
import (
    "gorm.io/gorm"
    "github.com/your-org/genkit-ai-service/internal/monitoring"
)

// 方式1：使用自定义 Logger
slowQueryLogger := monitoring.NewSlowQueryLogger(monitoring.SlowQueryConfig{
    SlowThreshold: 200 * time.Millisecond,
    Enabled:       true,
    Logger:        gormLogger,
})

db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: slowQueryLogger,
})

// 方式2：使用中间件插件
slowQueryMiddleware := monitoring.NewSlowQueryMiddleware(monitoring.SlowQueryConfig{
    SlowThreshold: 200 * time.Millisecond,
    Enabled:       true,
    Logger:        gormLogger,
})

db.Use(slowQueryMiddleware)
```

### 3. 在代码中记录指标

#### 记录登录尝试

```go
import (
    "time"
    "github.com/your-org/genkit-ai-service/internal/monitoring"
)

func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
    start := time.Now()
    
    // 执行登录逻辑
    resp, err := s.doLogin(ctx, req)
    
    // 记录指标
    duration := time.Since(start)
    success := err == nil
    monitoring.GetMetrics().RecordLoginAttempt(success, duration)
    
    return resp, err
}
```

#### 记录 Token 刷新

```go
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
    start := time.Now()
    
    // 执行刷新逻辑
    resp, err := s.doRefresh(ctx, refreshToken)
    
    // 记录指标
    duration := time.Since(start)
    success := err == nil
    monitoring.GetMetrics().RecordTokenRefresh(success, duration)
    
    return resp, err
}
```

#### 记录注销

```go
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
    err := s.doLogout(ctx, refreshToken)
    
    if err == nil {
        monitoring.GetMetrics().RecordLogout()
    }
    
    return err
}
```

#### 记录安全事件

```go
// 记录无效 Token
monitoring.GetMetrics().RecordInvalidToken()

// 记录过期 Token
monitoring.GetMetrics().RecordExpiredToken()

// 记录已撤销 Token
monitoring.GetMetrics().RecordRevokedToken()

// 记录暴力破解尝试
monitoring.GetMetrics().RecordBruteForceAttempt()
```

#### 更新租户和用户统计

```go
// 定期更新活跃租户数
activeTenants := countActiveTenants()
monitoring.GetMetrics().UpdateActiveTenants(activeTenants)

// 定期更新活跃用户数
activeUsers := countActiveUsers()
monitoring.GetMetrics().UpdateActiveUsers(activeUsers)
```

### 4. 注册监控路由

在路由配置中注册监控端点：

```go
import (
    "github.com/your-org/genkit-ai-service/internal/api/handler"
    "github.com/your-org/genkit-ai-service/internal/api/routes"
)

// 创建监控处理器
monitoringHandler := handler.NewMonitoringHandler(alertManager)

// 注册监控路由
routes.RegisterMonitoringRoutes(mux, monitoringHandler, jwtAuthMiddleware, rbacMiddleware)
```

## API 端点

### 获取性能指标

```bash
GET /api/v1/monitoring/metrics
Authorization: Bearer <admin_token>
```

响应示例：

```json
{
  "code": 200,
  "message": "获取指标成功",
  "data": {
    "login_attempts": 150,
    "login_successes": 145,
    "login_failures": 5,
    "login_success_rate": 96.67,
    "token_refreshes": 80,
    "token_refresh_failures": 2,
    "logouts": 30,
    "avg_login_duration_ms": 125.5,
    "p95_login_duration_ms": 250.0,
    "avg_refresh_duration_ms": 45.2,
    "p95_refresh_duration_ms": 80.0,
    "slow_queries": 5,
    "db_errors": 0,
    "invalid_tokens": 8,
    "expired_tokens": 12,
    "revoked_tokens": 30,
    "brute_force_attempts": 0,
    "active_tenants": 10,
    "active_users": 150,
    "timestamp": "2025-10-19T10:30:00Z"
  }
}
```

### 获取活跃告警

```bash
GET /api/v1/monitoring/alerts
Authorization: Bearer <admin_token>
```

响应示例：

```json
{
  "code": 200,
  "message": "获取告警成功",
  "data": [
    {
      "level": "warning",
      "title": "慢查询过多",
      "message": "慢查询数量超过阈值",
      "metric": "slow_queries",
      "value": 25,
      "threshold": 20,
      "timestamp": "2025-10-19T10:25:00Z"
    }
  ]
}
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

### 健康检查（含监控信息）

```bash
GET /api/v1/monitoring/health
```

响应示例：

```json
{
  "code": 200,
  "message": "健康检查成功",
  "data": {
    "status": "healthy",
    "timestamp": "2025-10-19T10:30:00Z",
    "active_alerts": 0,
    "critical_alerts": 0,
    "metrics": {
      "login_success_rate": 96.67,
      "avg_login_duration_ms": 125.5,
      "slow_queries": 5,
      "db_errors": 0,
      "active_tenants": 10,
      "active_users": 150
    }
  }
}
```

## 配置选项

### 环境变量

```bash
# 慢查询阈值（毫秒）
SLOW_QUERY_THRESHOLD=200

# 告警检查间隔（分钟）
ALERT_CHECK_INTERVAL=1

# 是否启用慢查询日志
ENABLE_SLOW_QUERY_LOG=true
```

### 自定义告警规则

```go
// 添加自定义告警规则
customRule := monitoring.AlertRule{
    Name:        "自定义告警",
    Description: "自定义告警描述",
    Metric:      "custom_metric",
    Threshold:   100.0,
    Level:       monitoring.AlertLevelWarning,
    Condition: func(s monitoring.MetricsSnapshot) (bool, float64) {
        // 自定义条件逻辑
        value := float64(s.LoginAttempts)
        if value > 100.0 {
            return true, value
        }
        return false, value
    },
}

alertManager.AddRule(customRule)
```

### 自定义告警处理器

```go
// 实现自定义告警处理器
type EmailAlertHandler struct {
    emailService EmailService
}

func (h *EmailAlertHandler) HandleAlert(alert monitoring.Alert) error {
    // 发送告警邮件
    return h.emailService.SendAlert(alert)
}

// 添加到告警管理器
alertManager.AddHandler(&EmailAlertHandler{
    emailService: emailService,
})
```

## 最佳实践

### 1. 定期重置指标

建议定期重置指标以避免数据累积：

```go
// 每天凌晨重置指标
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        monitoring.GetMetrics().Reset()
    }
}()
```

### 2. 监控关键路径

在所有认证相关的关键路径上添加监控：

- 登录
- Token 刷新
- 注销
- 密码修改
- Token 验证

### 3. 设置合理的阈值

根据实际业务情况调整告警阈值：

- 慢查询阈值：根据数据库性能设置
- 登录失败率：根据正常业务波动设置
- 响应时间：根据 SLA 要求设置

### 4. 集成外部监控系统

可以将指标导出到 Prometheus、Grafana 等监控系统：

```go
// 定期导出指标到 Prometheus
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        snapshot := monitoring.GetMetrics().GetSnapshot()
        // 导出到 Prometheus
        exportToPrometheus(snapshot)
    }
}()
```

## 故障排查

### 指标不更新

1. 检查是否正确调用了记录方法
2. 检查是否有并发访问问题
3. 检查日志是否有错误信息

### 告警不触发

1. 检查告警管理器是否已启动
2. 检查告警规则条件是否正确
3. 检查告警处理器是否正常工作

### 慢查询日志不记录

1. 检查慢查询日志是否已启用
2. 检查阈值设置是否合理
3. 检查 GORM Logger 配置是否正确

## 性能影响

监控模块对系统性能的影响：

- **CPU 开销**：< 1%
- **内存开销**：约 10-20MB（取决于数据量）
- **响应时间影响**：< 1ms

建议在生产环境中启用监控功能。

## 安全注意事项

1. **访问控制**：监控端点应该只允许管理员访问
2. **敏感信息**：不要在指标中记录密码、Token 等敏感信息
3. **日志安全**：慢查询日志中可能包含敏感数据，注意日志权限控制
4. **告警通知**：告警通知中不要包含敏感信息

## 扩展功能

### 集成 Prometheus

```go
import "github.com/prometheus/client_golang/prometheus"

// 创建 Prometheus 指标
var (
    loginAttempts = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "auth_login_attempts_total",
        Help: "Total number of login attempts",
    })
)

// 注册指标
prometheus.MustRegister(loginAttempts)

// 记录指标
loginAttempts.Inc()
```

### 集成 Grafana

创建 Grafana Dashboard 来可视化监控指标：

1. 配置 Prometheus 数据源
2. 创建 Dashboard
3. 添加面板展示各项指标
4. 配置告警规则

### 集成 ELK Stack

将慢查询日志发送到 Elasticsearch：

```go
// 配置 Logstash 输出
logstashOutput := &LogstashOutput{
    Host: "logstash:5000",
}

// 发送慢查询日志
logstashOutput.Send(slowQueryLog)
```

## 参考资料

- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [GORM 日志](https://gorm.io/docs/logger.html)
- [Go 性能优化](https://golang.org/doc/diagnostics.html)
