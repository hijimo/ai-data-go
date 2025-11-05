# Genkit 会话管理模块部署指南

## 概述

本文档描述如何在不同环境中部署 Genkit 会话管理模块，包括开发环境、测试环境和生产环境。

## 目录

- [系统要求](#系统要求)
- [环境准备](#环境准备)
- [本地开发部署](#本地开发部署)
- [Docker 部署](#docker-部署)
- [Kubernetes 部署](#kubernetes-部署)
- [配置管理](#配置管理)
- [数据库迁移](#数据库迁移)
- [监控和日志](#监控和日志)
- [故障排查](#故障排查)

## 系统要求

### 硬件要求

| 环境 | CPU | 内存 | 存储 |
|-----|-----|------|------|
| 开发 | 2 核 | 4 GB | 20 GB |
| 测试 | 4 核 | 8 GB | 50 GB |
| 生产 | 8 核 | 16 GB | 100 GB |

### 软件要求

- **Go**: 1.21 或更高版本
- **PostgreSQL**: 14.0 或更高版本（需支持 pgvector 扩展）
- **Redis**: 6.0 或更高版本
- **Docker**: 20.10 或更高版本（可选）
- **Kubernetes**: 1.24 或更高版本（可选）

### 依赖服务

- **Google AI API**: 用于 AI 生成和向量嵌入
- **Prometheus**: 用于监控指标收集（可选）
- **Jaeger**: 用于分布式追踪（可选）

## 环境准备

### 1. 安装 Go

```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 验证安装
go version
```

### 2. 安装 PostgreSQL

```bash
# macOS
brew install postgresql@14
brew services start postgresql@14

# Linux (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install postgresql-14 postgresql-contrib-14

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

### 3. 安装 pgvector 扩展

```bash
# macOS
brew install pgvector

# Linux
git clone https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install

# 在数据库中启用扩展
psql -U postgres -d your_database -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

### 4. 安装 Redis

```bash
# macOS
brew install redis
brew services start redis

# Linux (Ubuntu/Debian)
sudo apt-get install redis-server

# 启动服务
sudo systemctl start redis
sudo systemctl enable redis
```

### 5. 创建数据库

```bash
# 连接到 PostgreSQL
psql -U postgres

# 创建数据库
CREATE DATABASE genkit_dev;
CREATE DATABASE genkit_test;
CREATE DATABASE genkit_prod;

# 创建用户
CREATE USER genkit_user WITH PASSWORD 'your_password';

# 授权
GRANT ALL PRIVILEGES ON DATABASE genkit_dev TO genkit_user;
GRANT ALL PRIVILEGES ON DATABASE genkit_test TO genkit_user;
GRANT ALL PRIVILEGES ON DATABASE genkit_prod TO genkit_user;

# 退出
\q
```

## 本地开发部署

### 1. 克隆代码

```bash
git clone https://github.com/your-org/genkit-service.git
cd genkit-service
```

### 2. 安装依赖

```bash
go mod download
go mod verify
```

### 3. 配置环境变量

创建 `.env` 文件：

```bash
# 服务配置
SERVER_PORT=8080
SERVER_MODE=debug

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=genkit_dev
DB_USER=genkit_user
DB_PASSWORD=your_password
DB_MAX_CONNECTIONS=10
DB_SSL_MODE=disable

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Google AI 配置
GENAI_API_KEY=your_google_ai_api_key
GENAI_MODEL=gemini-1.5-flash

# JWT 配置
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRATION=24h

# 日志配置
LOG_LEVEL=debug
LOG_FORMAT=json
```

### 4. 运行数据库迁移

```bash
# 执行迁移
go run cmd/migrate/main.go up

# 验证迁移
psql -U genkit_user -d genkit_dev -c "\dt"
```

### 5. 启动服务

```bash
# 开发模式（带热重载）
go run cmd/server/main.go

# 或使用 air 进行热重载
air

# 验证服务
curl http://localhost:8080/health
```

### 6. 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/service/...

# 运行集成测试
go test -tags=integration ./test/integration/...

# 查看测试覆盖率
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Docker 部署

### 1. 构建 Docker 镜像

创建 `Dockerfile`：

```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git make

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o genkit-service ./cmd/server

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 复制二进制文件
COPY --from=builder /app/genkit-service .

# 复制配置文件
COPY config ./config

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动服务
CMD ["./genkit-service"]
```

构建镜像：

```bash
# 构建镜像
docker build -t genkit-service:latest .

# 查看镜像
docker images | grep genkit-service
```

### 2. 使用 Docker Compose

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  postgres:
    image: pgvector/pgvector:pg14
    container_name: genkit-postgres
    environment:
      POSTGRES_DB: genkit_dev
      POSTGRES_USER: genkit_user
      POSTGRES_PASSWORD: your_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U genkit_user"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: genkit-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5

  genkit-service:
    build: .
    container_name: genkit-service
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - SERVER_MODE=release
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=genkit_dev
      - DB_USER=genkit_user
      - DB_PASSWORD=your_password
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - GENAI_API_KEY=${GENAI_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

启动服务：

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f genkit-service

# 停止服务
docker-compose down

# 停止并删除数据
docker-compose down -v
```

### 3. Docker 镜像优化

多阶段构建优化：

```dockerfile
# 使用特定版本
FROM golang:1.21.5-alpine3.18 AS builder

# 使用构建缓存
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# 优化镜像大小
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/genkit-service /genkit-service
ENTRYPOINT ["/genkit-service"]
```

## Kubernetes 部署

### 1. 创建命名空间

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: genkit
```

```bash
kubectl apply -f namespace.yaml
```

### 2. 创建 ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: genkit-config
  namespace: genkit
data:
  config.yaml: |
    server:
      port: 8080
      mode: release
    
    database:
      host: postgres-service
      port: 5432
      database: genkit_prod
      max_connections: 50
      ssl_mode: require
    
    redis:
      host: redis-service
      port: 6379
      database: 0
    
    genkit:
      provider: google
      model: gemini-1.5-flash
      log_level: info
```

```bash
kubectl apply -f configmap.yaml
```

### 3. 创建 Secret

```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: genkit-secret
  namespace: genkit
type: Opaque
stringData:
  db-password: your_db_password
  redis-password: your_redis_password
  genai-api-key: your_google_ai_api_key
  jwt-secret: your_jwt_secret
```

```bash
kubectl apply -f secret.yaml
```

### 4. 创建 Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: genkit-service
  namespace: genkit
  labels:
    app: genkit-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: genkit-service
  template:
    metadata:
      labels:
        app: genkit-service
    spec:
      containers:
      - name: genkit-service
        image: genkit-service:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: SERVER_PORT
          value: "8080"
        - name: SERVER_MODE
          value: "release"
        - name: DB_HOST
          value: "postgres-service"
        - name: DB_PORT
          value: "5432"
        - name: DB_NAME
          value: "genkit_prod"
        - name: DB_USER
          value: "genkit_user"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: genkit-secret
              key: db-password
        - name: REDIS_HOST
          value: "redis-service"
        - name: REDIS_PORT
          value: "6379"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: genkit-secret
              key: redis-password
        - name: GENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: genkit-secret
              key: genai-api-key
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: genkit-secret
              key: jwt-secret
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        volumeMounts:
        - name: config
          mountPath: /root/config
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: genkit-config
```

```bash
kubectl apply -f deployment.yaml
```

### 5. 创建 Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: genkit-service
  namespace: genkit
  labels:
    app: genkit-service
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: genkit-service
```

```bash
kubectl apply -f service.yaml
```

### 6. 创建 Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: genkit-ingress
  namespace: genkit
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - api.example.com
    secretName: genkit-tls
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: genkit-service
            port:
              number: 80
```

```bash
kubectl apply -f ingress.yaml
```

### 7. 创建 HorizontalPodAutoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: genkit-hpa
  namespace: genkit
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: genkit-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

```bash
kubectl apply -f hpa.yaml
```

### 8. 部署 PostgreSQL

```yaml
# postgres-statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: genkit
spec:
  serviceName: postgres-service
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: pgvector/pgvector:pg14
        ports:
        - containerPort: 5432
          name: postgres
        env:
        - name: POSTGRES_DB
          value: genkit_prod
        - name: POSTGRES_USER
          value: genkit_user
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: genkit-secret
              key: db-password
        - name: PGDATA
          value: /var/lib/postgresql/data/pgdata
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 50Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: genkit
spec:
  ports:
  - port: 5432
    targetPort: 5432
  selector:
    app: postgres
  clusterIP: None
```

### 9. 部署 Redis

```yaml
# redis-statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
  namespace: genkit
spec:
  serviceName: redis-service
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
          name: redis
        command:
        - redis-server
        - --requirepass
        - $(REDIS_PASSWORD)
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: genkit-secret
              key: redis-password
        volumeMounts:
        - name: redis-storage
          mountPath: /data
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
  volumeClaimTemplates:
  - metadata:
      name: redis-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
---
apiVersion: v1
kind: Service
metadata:
  name: redis-service
  namespace: genkit
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
  clusterIP: None
```

### 10. 验证部署

```bash
# 查看所有资源
kubectl get all -n genkit

# 查看 Pod 状态
kubectl get pods -n genkit

# 查看 Pod 日志
kubectl logs -f deployment/genkit-service -n genkit

# 查看服务
kubectl get svc -n genkit

# 测试服务
kubectl port-forward svc/genkit-service 8080:80 -n genkit
curl http://localhost:8080/health
```

## 配置管理

### 1. 环境配置文件

创建不同环境的配置文件：

**开发环境** (`config/dev.yaml`):

```yaml
server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 5432
  database: genkit_dev
  user: genkit_user
  password: ${DB_PASSWORD}
  max_connections: 10
  ssl_mode: disable

redis:
  host: localhost
  port: 6379
  password: ""
  database: 0

genkit:
  provider: google
  api_key: ${GENAI_API_KEY}
  model: gemini-1.5-flash
  log_level: debug

jwt:
  secret: ${JWT_SECRET}
  expiration: 24h

logging:
  level: debug
  format: json
  output: stdout
```

**生产环境** (`config/prod.yaml`):

```yaml
server:
  port: 8080
  mode: release

database:
  host: ${DB_HOST}
  port: 5432
  database: ${DB_NAME}
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  max_connections: 50
  ssl_mode: require

redis:
  host: ${REDIS_HOST}
  port: 6379
  password: ${REDIS_PASSWORD}
  database: 0

genkit:
  provider: google
  api_key: ${GENAI_API_KEY}
  model: gemini-1.5-flash
  log_level: info

jwt:
  secret: ${JWT_SECRET}
  expiration: 24h

logging:
  level: info
  format: json
  output: stdout

monitoring:
  prometheus_port: 9090
  jaeger_endpoint: ${JAEGER_ENDPOINT}

rate_limiting:
  enabled: true
  requests_per_minute: 60
```

### 2. 配置加载

```go
// internal/config/config.go
package config

import (
    "os"
    "strings"
    
    "gopkg.in/yaml.v3"
)

type Config struct {
    Server     ServerConfig     `yaml:"server"`
    Database   DatabaseConfig   `yaml:"database"`
    Redis      RedisConfig      `yaml:"redis"`
    Genkit     GenkitConfig     `yaml:"genkit"`
    JWT        JWTConfig        `yaml:"jwt"`
    Logging    LoggingConfig    `yaml:"logging"`
    Monitoring MonitoringConfig `yaml:"monitoring"`
}

func Load(env string) (*Config, error) {
    // 加载配置文件
    configFile := fmt.Sprintf("config/%s.yaml", env)
    data, err := os.ReadFile(configFile)
    if err != nil {
        return nil, err
    }
    
    // 替换环境变量
    content := os.ExpandEnv(string(data))
    
    // 解析配置
    var config Config
    if err := yaml.Unmarshal([]byte(content), &config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

### 3. 环境变量管理

使用 `.env` 文件管理敏感信息：

```bash
# .env.example
DB_PASSWORD=your_db_password
REDIS_PASSWORD=your_redis_password
GENAI_API_KEY=your_google_ai_api_key
JWT_SECRET=your_jwt_secret
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
```

加载环境变量：

```go
// 使用 godotenv
import "github.com/joho/godotenv"

func init() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
}
```

## 数据库迁移

### 1. 迁移工具

使用 `golang-migrate` 进行数据库迁移：

```bash
# 安装 migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 创建迁移文件
migrate create -ext sql -dir migrations -seq create_tables

# 执行迁移
migrate -path migrations -database "postgresql://user:pass@localhost:5432/db?sslmode=disable" up

# 回滚迁移
migrate -path migrations -database "postgresql://user:pass@localhost:5432/db?sslmode=disable" down 1
```

### 2. 迁移脚本

创建迁移脚本 (`scripts/migrate.sh`):

```bash
#!/bin/bash

set -e

# 配置
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-genkit_dev}
DB_USER=${DB_USER:-genkit_user}
DB_PASSWORD=${DB_PASSWORD}

# 构建连接字符串
DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# 执行迁移
case "$1" in
    up)
        echo "Running migrations..."
        migrate -path migrations -database "${DB_URL}" up
        ;;
    down)
        echo "Rolling back migrations..."
        migrate -path migrations -database "${DB_URL}" down ${2:-1}
        ;;
    version)
        migrate -path migrations -database "${DB_URL}" version
        ;;
    force)
        migrate -path migrations -database "${DB_URL}" force $2
        ;;
    *)
        echo "Usage: $0 {up|down|version|force}"
        exit 1
        ;;
esac

echo "Migration completed successfully"
```

### 3. 自动迁移

在应用启动时自动执行迁移：

```go
// cmd/server/main.go
func main() {
    // 加载配置
    config := loadConfig()
    
    // 执行数据库迁移
    if err := runMigrations(config.Database); err != nil {
        log.Fatalf("Migration failed: %v", err)
    }
    
    // 启动服务
    startServer(config)
}

func runMigrations(dbConfig DatabaseConfig) error {
    dbURL := fmt.Sprintf(
        "postgresql://%s:%s@%s:%d/%s?sslmode=%s",
        dbConfig.User,
        dbConfig.Password,
        dbConfig.Host,
        dbConfig.Port,
        dbConfig.Database,
        dbConfig.SSLMode,
    )
    
    m, err := migrate.New(
        "file://migrations",
        dbURL,
    )
    if err != nil {
        return err
    }
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    
    return nil
}
```

## 监控和日志

### 1. Prometheus 监控

部署 Prometheus：

```yaml
# prometheus-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: genkit
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    
    scrape_configs:
    - job_name: 'genkit-service'
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
          - genkit
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: genkit-service
      - source_labels: [__meta_kubernetes_pod_ip]
        target_label: __address__
        replacement: $1:9090
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: genkit
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
        - name: storage
          mountPath: /prometheus
      volumes:
      - name: config
        configMap:
          name: prometheus-config
      - name: storage
        emptyDir: {}
```

### 2. Grafana 仪表板

部署 Grafana：

```yaml
# grafana-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: genkit
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:latest
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_PASSWORD
          valueFrom:
            secretKeyRef:
              name: genkit-secret
              key: grafana-password
        volumeMounts:
        - name: storage
          mountPath: /var/lib/grafana
      volumes:
      - name: storage
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: genkit
spec:
  ports:
  - port: 3000
    targetPort: 3000
  selector:
    app: grafana
```

### 3. 日志收集

使用 Fluentd 收集日志：

```yaml
# fluentd-daemonset.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd
  namespace: genkit
spec:
  selector:
    matchLabels:
      app: fluentd
  template:
    metadata:
      labels:
        app: fluentd
    spec:
      containers:
      - name: fluentd
        image: fluent/fluentd-kubernetes-daemonset:v1-debian-elasticsearch
        env:
        - name: FLUENT_ELASTICSEARCH_HOST
          value: "elasticsearch.logging.svc.cluster.local"
        - name: FLUENT_ELASTICSEARCH_PORT
          value: "9200"
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
```

## 故障排查

### 1. 常见问题

#### 服务无法启动

```bash
# 检查 Pod 状态
kubectl describe pod <pod-name> -n genkit

# 查看日志
kubectl logs <pod-name> -n genkit

# 常见原因：
# - 配置错误
# - 数据库连接失败
# - 缺少必要的环境变量
```

#### 数据库连接失败

```bash
# 测试数据库连接
kubectl run -it --rm debug --image=postgres:14 --restart=Never -n genkit -- \
  psql -h postgres-service -U genkit_user -d genkit_prod

# 检查数据库服务
kubectl get svc postgres-service -n genkit

# 检查数据库 Pod
kubectl get pods -l app=postgres -n genkit
```

#### Redis 连接失败

```bash
# 测试 Redis 连接
kubectl run -it --rm debug --image=redis:7-alpine --restart=Never -n genkit -- \
  redis-cli -h redis-service -a <password> ping

# 检查 Redis 服务
kubectl get svc redis-service -n genkit
```

### 2. 性能问题

#### CPU 使用率高

```bash
# 查看资源使用
kubectl top pods -n genkit

# 增加副本数
kubectl scale deployment genkit-service --replicas=5 -n genkit

# 调整资源限制
kubectl edit deployment genkit-service -n genkit
```

#### 内存泄漏

```bash
# 查看内存使用趋势
kubectl top pods -n genkit --watch

# 重启 Pod
kubectl rollout restart deployment genkit-service -n genkit

# 启用内存分析
# 在应用中添加 pprof 端点
```

### 3. 日志分析

```bash
# 查看最近的错误日志
kubectl logs deployment/genkit-service -n genkit | grep ERROR

# 查看特定时间段的日志
kubectl logs deployment/genkit-service -n genkit --since=1h

# 跟踪实时日志
kubectl logs -f deployment/genkit-service -n genkit

# 查看所有副本的日志
kubectl logs -l app=genkit-service -n genkit --all-containers=true
```

## 安全最佳实践

### 1. 使用 Secret 管理敏感信息

```bash
# 创建 Secret
kubectl create secret generic genkit-secret \
  --from-literal=db-password=<password> \
  --from-literal=jwt-secret=<secret> \
  -n genkit

# 使用外部 Secret 管理工具
# - HashiCorp Vault
# - AWS Secrets Manager
# - Azure Key Vault
```

### 2. 网络策略

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: genkit-network-policy
  namespace: genkit
spec:
  podSelector:
    matchLabels:
      app: genkit-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

### 3. Pod Security Policy

```yaml
# pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: genkit-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
  - ALL
  volumes:
  - 'configMap'
  - 'emptyDir'
  - 'projected'
  - 'secret'
  - 'downwardAPI'
  - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: false
```

## 备份和恢复

### 1. 数据库备份

```bash
# 创建备份脚本
#!/bin/bash
BACKUP_DIR="/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/genkit_backup_${TIMESTAMP}.sql"

# 执行备份
kubectl exec -n genkit postgres-0 -- \
  pg_dump -U genkit_user genkit_prod > ${BACKUP_FILE}

# 压缩备份
gzip ${BACKUP_FILE}

# 上传到云存储
aws s3 cp ${BACKUP_FILE}.gz s3://your-bucket/backups/
```

### 2. 数据库恢复

```bash
# 下载备份
aws s3 cp s3://your-bucket/backups/genkit_backup_20240101_120000.sql.gz .

# 解压
gunzip genkit_backup_20240101_120000.sql.gz

# 恢复数据库
kubectl exec -i -n genkit postgres-0 -- \
  psql -U genkit_user genkit_prod < genkit_backup_20240101_120000.sql
```

## 总结

本部署指南涵盖了从本地开发到生产环境的完整部署流程。关键要点：

1. **环境准备**：确保所有依赖服务正确安装和配置
2. **配置管理**：使用环境变量和配置文件管理不同环境
3. **容器化**：使用 Docker 实现一致的部署环境
4. **编排**：使用 Kubernetes 实现自动化部署和扩展
5. **监控**：部署 Prometheus 和 Grafana 进行监控
6. **日志**：使用 Fluentd 收集和分析日志
7. **安全**：实施网络策略和 Secret 管理
8. **备份**：定期备份数据库和关键数据

---

**文档版本**: v1.0.0  
**最后更新**: 2024-01-01  
**维护者**: 运维团队
