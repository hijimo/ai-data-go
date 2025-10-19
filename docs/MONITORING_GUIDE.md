# 性能监控配置指南

本文档介绍如何配置和使用认证系统的性能监控功能。

## 概述

性能监控模块提供以下功能：

1. **性能指标收集**：自动收集认证相关的性能指标
2. **慢查询日志**：记录超过阈值的数据库查询
3. **告警规则**：自动检测异常情况并触发告警

## 快速开始

### 1. 环境变量配置

在 `.env` 文件中添加以下配置：

```bash
# 慢查询阈值（毫秒）
SLOW_QUERY_THRESHOLD=200

# 告警检查间隔（分钟）
ALERT_CHECK_INTERVAL=1

# 是否启用慢查询日志
ENABLE_SLOW_QUERY_LOG=true

# 是否启用性能监控
ENABLE_MONITORING=true
```

### 2. 初始化监控模块

在 `cmd/server/main.go` 中初始化监控：

```go
import (
    "time"
    "github.com/your-org/genkit-ai-service/internal/monitoring"
    "github.com/your-org/genkit-ai-service/internal/api/handler"
    "github.com/your-org/genkit-ai-service/internal/api/routes"
)

func main() {
    // ... 其他初始化代码
    
    // 创建告警管理器
    alertManager := monitoring.NewAlertManager(1 * time.Minute)
    
    // 添加默认告警规则
    for _, rule := range monitoring.GetDefaultAlertRules() {
        alertManager.AddRule(rule)
    }
    
    // 添加日志告警处理器
    logHandler := &monitoring.LogAlertHandler{
        logger: logger,
    }
    alertManager.AddHandler(logHandler)
    
    // 启动告警检查
    alertManager.Start()
    defer alertManager.Stop()
    
    // 配置慢查询日志
    slowQueryLogger := monitoring.NewSlowQueryLogger(monitoring.SlowQueryConfig{
        SlowThreshold: 200 * time.Millisecond,
        Enabled:       true,
        Logger:        gormLogger,
    })
    
    // 初始化数据库时使用慢查询 logger
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: slowQueryLogger,
    })
    
    // 创建监控处理器
    monitoringHandler := handler.NewMonitoringHandler(alertManager)
    
    // 注册监控路由
    routes.RegisterMonitoringRoutes(mux, monitoringHandler, jwtAuthMiddleware, rbacMiddleware)
    
    // ... 启动服务器
}
```

### 3. 在认证服务中集成监控

修改 `internal/service/auth/auth_service.go`：

```go
import (
    "time"
    "github.com/your-org/genkit-ai-service/internal/monitoring"
)

func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
    start := time.Now()
    
    // 执行登录逻辑
    user, err := s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
    if err != nil {
        // 记录失败
        monitoring.GetMetrics().RecordLoginAttempt(false, time.Since(start))
        return nil, err
    }
    
    // 验证密码
    if err := crypto.VerifyPassword(user.PasswordHash, req.Password); err != nil {
        // 记录失败
        monitoring.GetMetrics().RecordLoginAttempt(false, time.Since(start))
        // 记录暴力破解尝试
        monitoring.GetMetrics().RecordBruteForceAttempt()
        return nil, ErrInvalidCredentials
    }
    
    // 生成 Token
    accessToken, err := s.tokenService.GenerateAccessToken(user)
    if err != nil {
        monitoring.GetMetrics().RecordLoginAttempt(false, time.Since(start))
        return nil, err
    }
    
    refreshToken, _, err := s.tokenService.GenerateRefreshToken(user)
    if err != nil {
        monitoring.GetMetrics().RecordLoginAttempt(false, time.Since(start))
        return nil, err
    }
    
    // 记录成功
    monitoring.GetMetrics().RecordLoginAttempt(true, time.Since(start))
    
    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    3600,
        TokenType:    "Bearer",
        User:         user,
    }, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
    start := time.Now()
    
    // 执行刷新逻辑
    // ...
    
    // 记录结果
    success := err == nil
    monitoring.GetMetrics().RecordTokenRefresh(success, time.Since(start))
    
    return resp, err
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
    err := s.tokenService.RevokeRefreshToken(ctx, refreshToken)
    
    if err == nil {
        monitoring.GetMetrics().RecordLogout()
    }
    
    return err
}
```

### 4. 在 JWT 中间件中集成监控

修改 `internal/api/middleware/jwt_auth.go`：

```go
import "github.com/your-org/genkit-ai-service/internal/monitoring"

func JWTAuth(tokenService TokenService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // ... 提取 token
            
            // 验证 token
            claims, err := tokenService.ValidateAccessToken(tokenString)
            if err != nil {
                // 记录错误类型
                if errors.Is(err, jwt.ErrTokenExpired) {
                    monitoring.GetMetrics().RecordExpiredToken()
                } else {
                    monitoring.GetMetrics().RecordInvalidToken()
                }
                
                http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
                return
            }
            
            // ... 继续处理
        })
    }
}
```

## API 使用

### 获取性能指标

```bash
curl -X GET http://localhost:8080/api/v1/monitoring/metrics \
  -H "Authorization: Bearer <admin_token>"
```

### 获取活跃告警

```bash
curl -X GET http://localhost:8080/api/v1/monitoring/alerts \
  -H "Authorization: Bearer <admin_token>"
```

### 清空告警

```bash
curl -X DELETE http://localhost:8080/api/v1/monitoring/alerts \
  -H "Authorization: Bearer <admin_token>"
```

### 重置指标

```bash
curl -X POST http://localhost:8080/api/v1/monitoring/metrics/reset \
  -H "Authorization: Bearer <admin_token>"
```

### 健康检查

```bash
curl -X GET http://localhost:8080/api/v1/monitoring/health
```

## 告警规则配置

### 默认告警规则

系统内置以下告警规则：

| 规则名称 | 指标 | 阈值 | 级别 | 说明 |
|---------|------|------|------|------|
| 高登录失败率 | login_success_rate | < 70% | Warning | 登录失败率过高 |
| 大量登录失败 | login_failures | > 50 | Critical | 可能存在暴力破解 |
| Token 刷新失败率高 | token_refresh_failures | > 10% | Warning | Token 刷新异常 |
| 慢查询过多 | slow_queries | > 20 | Warning | 数据库性能问题 |
| 数据库错误频繁 | db_errors | > 10 | Critical | 数据库连接问题 |
| 无效 Token 过多 | invalid_tokens | > 30 | Warning | 可能存在攻击 |
| 暴力破解攻击 | brute_force_attempts | > 20 | Critical | 检测到攻击 |
| 登录响应时间过长 | p95_login_duration | > 1000ms | Warning | 性能下降 |
| Token 刷新响应时间过长 | p95_refresh_duration | > 500ms | Warning | 性能下降 |

### 自定义告警规则

```go
// 添加自定义告警规则
customRule := monitoring.AlertRule{
    Name:        "活跃用户数过低",
    Description: "活跃用户数低于预期",
    Metric:      "active_users",
    Threshold:   100.0,
    Level:       monitoring.AlertLevelWarning,
    Condition: func(s monitoring.MetricsSnapshot) (bool, float64) {
        value := float64(s.ActiveUsers)
        if value < 100.0 {
            return true, value
        }
        return false, value
    },
}

alertManager.AddRule(customRule)
```

### 自定义告警处理器

#### 邮件告警

```go
type EmailAlertHandler struct {
    smtpHost     string
    smtpPort     int
    smtpUser     string
    smtpPassword string
    recipients   []string
}

func (h *EmailAlertHandler) HandleAlert(alert monitoring.Alert) error {
    subject := fmt.Sprintf("[%s] %s", alert.Level, alert.Title)
    body := fmt.Sprintf(
        "告警: %s\n描述: %s\n指标: %s\n当前值: %.2f\n阈值: %.2f\n时间: %s",
        alert.Title,
        alert.Message,
        alert.Metric,
        alert.Value,
        alert.Threshold,
        alert.Timestamp.Format(time.RFC3339),
    )
    
    return h.sendEmail(subject, body)
}

// 注册处理器
alertManager.AddHandler(&EmailAlertHandler{
    smtpHost:     "smtp.example.com",
    smtpPort:     587,
    smtpUser:     "alerts@example.com",
    smtpPassword: "password",
    recipients:   []string{"admin@example.com"},
})
```

#### Webhook 告警

```go
type WebhookAlertHandler struct {
    webhookURL string
    client     *http.Client
}

func (h *WebhookAlertHandler) HandleAlert(alert monitoring.Alert) error {
    payload, err := json.Marshal(alert)
    if err != nil {
        return err
    }
    
    resp, err := h.client.Post(h.webhookURL, "application/json", bytes.NewBuffer(payload))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("webhook returned status %d", resp.StatusCode)
    }
    
    return nil
}

// 注册处理器
alertManager.AddHandler(&WebhookAlertHandler{
    webhookURL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    client:     &http.Client{Timeout: 10 * time.Second},
})
```

## 集成 Prometheus

### 1. 安装 Prometheus 客户端

```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
```

### 2. 创建 Prometheus 导出器

```go
package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    loginAttempts = promauto.NewCounter(prometheus.CounterOpts{
        Name: "auth_login_attempts_total",
        Help: "Total number of login attempts",
    })
    
    loginSuccesses = promauto.NewCounter(prometheus.CounterOpts{
        Name: "auth_login_successes_total",
        Help: "Total number of successful logins",
    })
    
    loginDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "auth_login_duration_seconds",
        Help:    "Login request duration in seconds",
        Buckets: prometheus.DefBuckets,
    })
    
    activeTenants = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "auth_active_tenants",
        Help: "Number of active tenants",
    })
)

// ExportToPrometheus 导出指标到 Prometheus
func ExportToPrometheus() {
    snapshot := GetMetrics().GetSnapshot()
    
    loginAttempts.Add(float64(snapshot.LoginAttempts))
    loginSuccesses.Add(float64(snapshot.LoginSuccesses))
    activeTenants.Set(float64(snapshot.ActiveTenants))
}
```

### 3. 注册 Prometheus 端点

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    // ... 其他初始化
    
    // 注册 Prometheus metrics 端点
    http.Handle("/metrics", promhttp.Handler())
    
    // 定期导出指标
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()
        
        for range ticker.C {
            monitoring.ExportToPrometheus()
        }
    }()
}
```

### 4. Prometheus 配置

在 `prometheus.yml` 中添加：

```yaml
scrape_configs:
  - job_name: 'genkit-auth'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## 集成 Grafana

### 1. 添加 Prometheus 数据源

在 Grafana 中添加 Prometheus 数据源，指向 Prometheus 服务器。

### 2. 创建 Dashboard

创建一个新的 Dashboard，添加以下面板：

#### 登录成功率面板

```promql
rate(auth_login_successes_total[5m]) / rate(auth_login_attempts_total[5m]) * 100
```

#### 登录响应时间面板

```promql
histogram_quantile(0.95, rate(auth_login_duration_seconds_bucket[5m]))
```

#### 活跃租户数面板

```promql
auth_active_tenants
```

### 3. 配置告警

在 Grafana 中配置告警规则，例如：

- 登录成功率低于 70%
- P95 响应时间超过 1 秒
- 慢查询数量超过阈值

## 性能优化建议

### 1. 定期重置指标

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

### 2. 调整慢查询阈值

根据实际情况调整慢查询阈值：

```bash
# 开发环境：较宽松
SLOW_QUERY_THRESHOLD=500

# 生产环境：较严格
SLOW_QUERY_THRESHOLD=100
```

### 3. 优化告警检查频率

```bash
# 高负载环境：降低检查频率
ALERT_CHECK_INTERVAL=5

# 低负载环境：提高检查频率
ALERT_CHECK_INTERVAL=1
```

## 故障排查

### 指标不更新

1. 检查监控是否已启用：`ENABLE_MONITORING=true`
2. 检查是否正确调用了记录方法
3. 查看日志是否有错误信息

### 告警不触发

1. 检查告警管理器是否已启动
2. 检查告警规则条件是否正确
3. 检查告警处理器是否正常工作
4. 查看日志中的告警信息

### 慢查询日志不记录

1. 检查慢查询日志是否已启用：`ENABLE_SLOW_QUERY_LOG=true`
2. 检查阈值设置是否合理
3. 检查 GORM Logger 配置是否正确

## 最佳实践

1. **监控关键路径**：在所有认证相关的关键路径上添加监控
2. **设置合理阈值**：根据实际业务情况调整告警阈值
3. **定期审查指标**：定期查看监控指标，发现潜在问题
4. **集成外部系统**：将指标导出到 Prometheus、Grafana 等专业监控系统
5. **告警分级处理**：根据告警级别采取不同的处理措施
6. **保护敏感信息**：不要在指标和日志中记录密码、Token 等敏感信息

## 参考资料

- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [GORM 日志](https://gorm.io/docs/logger.html)
- [Go 性能优化](https://golang.org/doc/diagnostics.html)
