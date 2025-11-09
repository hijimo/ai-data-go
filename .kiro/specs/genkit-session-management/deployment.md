# Genkit 会话管理系统部署指南

## 目录

1. [系统要求](#系统要求)
2. [环境变量配置](#环境变量配置)
3. [数据库配置](#数据库配置)
4. [Redis 配置](#redis-配置)
5. [Qdrant 向量数据库配置](#qdrant-向量数据库配置)
6. [Genkit AI 配置](#genkit-ai-配置)
7. [监控配置](#监控配置)
8. [部署步骤](#部署步骤)
9. [健康检查](#健康检查)
10. [故障排查](#故障排查)

---

## 系统要求

### 硬件要求

- **CPU**: 最低 2 核，推荐 4 核或更多
- **内存**: 最低 4GB，推荐 8GB 或更多
- **磁盘**: 最低 20GB 可用空间，推荐 50GB 或更多（用于日志和数据存储）

### 软件要求

- **操作系统**: Linux (Ubuntu 20.04+, CentOS 8+) 或 macOS
- **Go**: 1.21 或更高版本
- **PostgreSQL**: 14 或更高版本（需要 pgvector 扩展）
- **Redis**: 6.0 或更高版本
- **Qdrant**: 最新稳定版本（云服务或自托管）
- **Docker**: 20.10+ (可选，用于容器化部署)
- **Kubernetes**: 1.24+ (可选，用于 K8s 部署)

---

## 环境变量配置

本系统使用 `.env` 文件管理所有配置项。请复制 `.env.example` 文件为 `.env` 并根据实际环境填写配置值。

### 配置文件位置

```bash
# 项目根目录下
.env
```

### 核心配置项

#### 1. 服务器配置

```bash
# 服务器监听端口
SERVER_PORT=8080

# 服务器监听地址（0.0.0.0 表示监听所有网络接口）
SERVER_HOST=0.0.0.0
```

**说明**：

- `SERVER_PORT`: 应用程序监听的端口号，默认 8080
- `SERVER_HOST`: 监听地址，生产环境建议使用 `0.0.0.0`

#### 2. 数据库配置

```bash
# PostgreSQL 主机地址
DB_HOST=localhost

# PostgreSQL 端口
DB_PORT=5432

# 数据库名称
DB_NAME=ai_service

# 数据库用户名
DB_USER=postgres

# 数据库密码
DB_PASSWORD=your_secure_password

# SSL 模式 (disable, require, verify-ca, verify-full)
DB_SSL_MODE=disable

# 最大打开连接数
DB_MAX_OPEN_CONNS=25

# 最大空闲连接数
DB_MAX_IDLE_CONNS=5

# 连接最大生命周期
DB_CONN_MAX_LIFETIME=5m

# 数据库日志级别 (silent, error, warn, info)
DB_LOG_LEVEL=warn
```

**说明**：

- `DB_HOST`: PostgreSQL 服务器地址
- `DB_PORT`: PostgreSQL 端口，默认 5432
- `DB_NAME`: 数据库名称，需要提前创建
- `DB_USER`: 数据库用户，需要有创建表和索引的权限
- `DB_PASSWORD`: 数据库密码，生产环境必须使用强密码
- `DB_SSL_MODE`: 生产环境建议使用 `require` 或更高级别
- `DB_MAX_OPEN_CONNS`: 根据服务器负载调整，建议 25-100
- `DB_MAX_IDLE_CONNS`: 建议设置为 `DB_MAX_OPEN_CONNS` 的 20%
- `DB_CONN_MAX_LIFETIME`: 连接最大生命周期，建议 5-15 分钟

#### 3. Redis 配置

```bash
# 是否启用 Redis
REDIS_ENABLED=true

# Redis 主机地址
REDIS_HOST=localhost

# Redis 端口
REDIS_PORT=6379

# Redis 密码（如果没有密码则留空）
REDIS_PASSWORD=your_redis_password

# Redis 数据库编号（0-15）
REDIS_DB=0
```

**说明**：

- `REDIS_ENABLED`: 是否启用 Redis 缓存，生产环境强烈建议启用
- `REDIS_HOST`: Redis 服务器地址
- `REDIS_PORT`: Redis 端口，默认 6379
- `REDIS_PASSWORD`: Redis 密码，生产环境必须设置
- `REDIS_DB`: Redis 数据库编号，建议使用独立的数据库编号

#### 4. Qdrant 向量数据库配置

```bash
# Qdrant 访问密钥
QDRANT_ACCESS_KEY=your_qdrant_access_key

# Qdrant 端点地址
QDRANT_ENDPOINT=https://your-cluster.cloud.qdrant.io

# Qdrant 集群 ID
QDRANT_CLUSTER_ID=your_cluster_id
```

**说明**：

- `QDRANT_ACCESS_KEY`: Qdrant 云服务的访问密钥，从 Qdrant 控制台获取
- `QDRANT_ENDPOINT`: Qdrant 服务端点，云服务格式为 `https://<cluster>.cloud.qdrant.io`
- `QDRANT_CLUSTER_ID`: Qdrant 集群 ID，从 Qdrant 控制台获取

**重要提示**：

- 本系统使用**单 Collection 多租户架构**
- Collection 名称固定为 `conversation_memories`
- 通过 `tenant_id` 字段实现租户隔离
- 系统启动时会自动创建 Collection（如果不存在）

#### 5. Genkit AI 配置

```bash
# Genkit API 密钥
GENKIT_API_KEY=your_genkit_api_key

# 默认使用的模型
GENKIT_MODEL=gemini-2.5-flash

# 默认温度参数（0.0-1.0）
GENKIT_DEFAULT_TEMPERATURE=0.7

# 默认最大 Token 数
GENKIT_DEFAULT_MAX_TOKENS=2000
```

**说明**：

- `GENKIT_API_KEY`: Genkit AI 服务的 API 密钥，必填
- `GENKIT_MODEL`: 默认使用的 AI 模型，推荐 `gemini-2.5-flash`
- `GENKIT_DEFAULT_TEMPERATURE`: 生成文本的随机性，0.0 最确定，1.0 最随机
- `GENKIT_DEFAULT_MAX_TOKENS`: 单次生成的最大 Token 数

**支持的模型**：

- `gemini-2.5-flash`: 快速响应，适合大多数场景
- `gemini-2.5-pro`: 更高质量，适合复杂任务
- `gemini-1.5-flash`: 旧版快速模型
- `gemini-1.5-pro`: 旧版高质量模型

#### 6. 缓存 TTL 配置

```bash
# 会话上下文缓存时间（秒）
CACHE_CONTEXT_TTL=300

# 向量查询结果缓存时间（秒）
CACHE_VECTOR_RESULT_TTL=1800

# 摘要缓存时间（秒）
CACHE_SUMMARY_TTL=3600

# 会话列表缓存时间（秒）
CACHE_SESSION_LIST_TTL=600

# Token 使用统计缓存时间（秒）
CACHE_TOKEN_USAGE_TTL=300
```

**说明**：

- `CACHE_CONTEXT_TTL`: 上下文缓存 5 分钟，频繁变化的数据
- `CACHE_VECTOR_RESULT_TTL`: 向量查询结果缓存 30 分钟，相对稳定
- `CACHE_SUMMARY_TTL`: 摘要缓存 1 小时，变化较少
- `CACHE_SESSION_LIST_TTL`: 会话列表缓存 10 分钟
- `CACHE_TOKEN_USAGE_TTL`: Token 统计缓存 5 分钟

**调优建议**：

- 高并发场景：适当增加 TTL 以减少数据库压力
- 实时性要求高：减少 TTL 以获取最新数据
- 根据实际业务场景调整各项 TTL

#### 7. JWT 认证配置

```bash
# JWT 签名密钥（生产环境必须使用强随机密钥，至少 32 个字符）
JWT_SECRET=your_jwt_secret_key_min_32_characters

# JWT 签发者
JWT_ISSUER=genkit-ai-service

# JWT 受众
JWT_AUDIENCE=genkit-api

# Access Token 生命周期
ACCESS_TOKEN_TTL=60m

# Refresh Token 生命周期（30 天）
REFRESH_TOKEN_TTL=720h
```

**说明**：

- `JWT_SECRET`: JWT 签名密钥，**必须至少 32 个字符**，生产环境使用强随机字符串
- `JWT_ISSUER`: JWT 签发者标识
- `JWT_AUDIENCE`: JWT 受众标识
- `ACCESS_TOKEN_TTL`: 访问令牌有效期，建议 15-60 分钟
- `REFRESH_TOKEN_TTL`: 刷新令牌有效期，建议 7-30 天

**安全建议**：

- 生产环境必须使用强随机密钥
- 定期轮换 JWT 密钥
- 不要在代码中硬编码密钥
- 使用环境变量或密钥管理服务

#### 8. 日志配置

```bash
# 日志级别 (debug, info, warn, error)
LOG_LEVEL=info

# 日志格式 (json, text)
LOG_FORMAT=json

# 是否启用文件日志（按天存储）
LOG_ENABLE_FILE=true

# 日志文件目录
LOG_DIR=logs

# 是否同时输出到控制台
LOG_ENABLE_CONSOLE=true
```

**说明**：

- `LOG_LEVEL`: 日志级别，生产环境建议 `info` 或 `warn`
- `LOG_FORMAT`: 日志格式，生产环境建议 `json` 便于日志分析
- `LOG_ENABLE_FILE`: 是否写入文件，生产环境建议启用
- `LOG_DIR`: 日志文件存储目录
- `LOG_ENABLE_CONSOLE`: 是否输出到控制台，容器环境建议启用

#### 9. 追踪配置

```bash
# 是否启用追踪
TRACING_ENABLED=true

# 服务名称
TRACING_SERVICE_NAME=genkit-ai-service

# 服务版本
TRACING_SERVICE_VERSION=1.0.0

# 环境（development/staging/production）
TRACING_ENVIRONMENT=production

# OTLP 端点地址（支持 Jaeger v1.35+、Tempo 等）
OTLP_ENDPOINT=localhost:4318

# 采样率（0.0-1.0，1.0表示100%采样）
TRACING_SAMPLING_RATE=0.1
```

**说明**：

- `TRACING_ENABLED`: 是否启用分布式追踪
- `TRACING_SERVICE_NAME`: 服务名称，用于追踪系统识别
- `TRACING_SERVICE_VERSION`: 服务版本号
- `TRACING_ENVIRONMENT`: 部署环境标识
- `OTLP_ENDPOINT`: OpenTelemetry 协议端点
- `TRACING_SAMPLING_RATE`: 采样率，生产环境建议 0.01-0.1（1%-10%）

#### 10. 系统初始化配置

```bash
# 平台管理员邮箱
PLATFORM_ADMIN_EMAIL=admin@example.com

# 平台管理员初始密码（留空则自动生成）
PLATFORM_ADMIN_PASSWORD=

# 平台管理员显示名称
PLATFORM_ADMIN_NAME=Platform Admin

# 平台租户名称
PLATFORM_TENANT_NAME=Platform

# 平台租户域名
PLATFORM_TENANT_DOMAIN=system.local
```

**说明**：

- `PLATFORM_ADMIN_EMAIL`: 平台管理员邮箱，首次启动时创建
- `PLATFORM_ADMIN_PASSWORD`: 初始密码，留空则自动生成并输出到日志
- `PLATFORM_ADMIN_NAME`: 管理员显示名称
- `PLATFORM_TENANT_NAME`: 平台租户名称
- `PLATFORM_TENANT_DOMAIN`: 平台租户域名

---

## 数据库配置

### PostgreSQL 安装

#### Ubuntu/Debian

```bash
# 添加 PostgreSQL 官方仓库
sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -

# 更新包列表并安装
sudo apt-get update
sudo apt-get install -y postgresql-14 postgresql-contrib-14
```

#### CentOS/RHEL

```bash
# 安装 PostgreSQL 仓库
sudo yum install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-8-x86_64/pgdg-redhat-repo-latest.noarch.rpm

# 安装 PostgreSQL
sudo yum install -y postgresql14-server postgresql14-contrib

# 初始化数据库
sudo /usr/pgsql-14/bin/postgresql-14-setup initdb

# 启动服务
sudo systemctl enable postgresql-14
sudo systemctl start postgresql-14
```

#### macOS

```bash
# 使用 Homebrew 安装
brew install postgresql@14

# 启动服务
brew services start postgresql@14
```

### 安装 pgvector 扩展

pgvector 是 PostgreSQL 的向量相似度搜索扩展，本系统用于存储和检索会话记忆的向量嵌入。

#### 从源码编译安装

```bash
# 安装依赖
sudo apt-get install -y postgresql-server-dev-14 build-essential

# 克隆 pgvector 仓库
git clone --branch v0.5.1 https://github.com/pgvector/pgvector.git
cd pgvector

# 编译并安装
make
sudo make install
```

#### 使用包管理器安装（Ubuntu）

```bash
# 添加 pgvector 仓库
sudo apt-get install -y postgresql-14-pgvector
```

### 创建数据库和启用扩展

```bash
# 连接到 PostgreSQL
sudo -u postgres psql

# 创建数据库
CREATE DATABASE ai_service;

# 连接到新数据库
\c ai_service

# 启用 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;

# 启用 UUID 生成函数（PostgreSQL 13+）
-- PostgreSQL 13+ 内置，无需额外操作

# 创建数据库用户（如果需要）
CREATE USER ai_service_user WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE ai_service TO ai_service_user;

# 退出
\q
```

### 验证 pgvector 安装

```sql
-- 连接到数据库
\c ai_service

-- 检查扩展
SELECT * FROM pg_extension WHERE extname = 'vector';

-- 测试向量操作
SELECT '[1,2,3]'::vector;

-- 测试余弦距离
SELECT '[1,2,3]'::vector <=> '[4,5,6]'::vector;
```

### 数据库性能优化

#### postgresql.conf 配置建议

```ini
# 内存配置（根据服务器内存调整）
shared_buffers = 2GB                    # 25% 的系统内存
effective_cache_size = 6GB              # 50-75% 的系统内存
maintenance_work_mem = 512MB
work_mem = 16MB

# 连接配置
max_connections = 100

# WAL 配置
wal_buffers = 16MB
checkpoint_completion_target = 0.9

# 查询规划器
random_page_cost = 1.1                  # SSD 存储
effective_io_concurrency = 200          # SSD 存储

# 日志配置
logging_collector = on
log_directory = 'log'
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_rotation_age = 1d
log_rotation_size = 100MB
log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h '
log_min_duration_statement = 1000       # 记录慢查询（>1秒）
```

#### 重启 PostgreSQL 使配置生效

```bash
sudo systemctl restart postgresql-14
```

---

## Redis 配置

### Redis 安装

#### Ubuntu/Debian

```bash
# 安装 Redis
sudo apt-get update
sudo apt-get install -y redis-server

# 启动服务
sudo systemctl enable redis-server
sudo systemctl start redis-server
```

#### CentOS/RHEL

```bash
# 安装 EPEL 仓库
sudo yum install -y epel-release

# 安装 Redis
sudo yum install -y redis

# 启动服务
sudo systemctl enable redis
sudo systemctl start redis
```

#### macOS

```bash
# 使用 Homebrew 安装
brew install redis

# 启动服务
brew services start redis
```

### Redis 配置优化

编辑 Redis 配置文件（通常位于 `/etc/redis/redis.conf`）：

```conf
# 绑定地址（生产环境建议只绑定内网地址）
bind 127.0.0.1

# 端口
port 6379

# 设置密码（生产环境必须设置）
requirepass your_strong_redis_password

# 最大内存限制（根据服务器内存调整）
maxmemory 2gb

# 内存淘汰策略（推荐 allkeys-lru）
maxmemory-policy allkeys-lru

# 持久化配置
# RDB 快照
save 900 1
save 300 10
save 60 10000

# AOF 持久化（可选，更安全但性能略低）
appendonly yes
appendfsync everysec

# 日志级别
loglevel notice

# 日志文件
logfile /var/log/redis/redis-server.log

# 数据库数量
databases 16

# 慢查询日志
slowlog-log-slower-than 10000
slowlog-max-len 128
```

### 重启 Redis 使配置生效

```bash
sudo systemctl restart redis-server
```

### 验证 Redis 安装

```bash
# 连接到 Redis
redis-cli

# 如果设置了密码
redis-cli -a your_strong_redis_password

# 测试连接
127.0.0.1:6379> PING
PONG

# 设置和获取值
127.0.0.1:6379> SET test "Hello Redis"
OK
127.0.0.1:6379> GET test
"Hello Redis"

# 退出
127.0.0.1:6379> EXIT
```

### Redis 性能监控

```bash
# 查看 Redis 信息
redis-cli INFO

# 查看内存使用
redis-cli INFO memory

# 查看慢查询日志
redis-cli SLOWLOG GET 10

# 实时监控命令
redis-cli MONITOR
```

### Redis 安全建议

1. **设置强密码**：使用 `requirepass` 设置复杂密码
2. **绑定内网地址**：不要暴露到公网
3. **禁用危险命令**：

   ```conf
   rename-command FLUSHDB ""
   rename-command FLUSHALL ""
   rename-command CONFIG ""
   ```

4. **启用防火墙**：只允许应用服务器访问 Redis 端口
5. **定期备份**：使用 RDB 或 AOF 持久化

---

## Qdrant 向量数据库配置

### Qdrant 部署选项

本系统支持两种 Qdrant 部署方式：

1. **Qdrant Cloud**（推荐）：托管服务，无需维护
2. **自托管 Qdrant**：完全控制，需要自行维护

### 选项 1：使用 Qdrant Cloud（推荐）

#### 1. 注册 Qdrant Cloud 账号

访问 [https://cloud.qdrant.io](https://cloud.qdrant.io) 注册账号。

#### 2. 创建集群

1. 登录 Qdrant Cloud 控制台
2. 点击 "Create Cluster"
3. 选择区域（建议选择离应用服务器最近的区域）
4. 选择集群规格：
   - **Free Tier**: 适合开发和测试
   - **Starter**: 适合小型生产环境
   - **Professional**: 适合中大型生产环境
5. 创建集群

#### 3. 获取连接信息

创建完成后，在集群详情页面获取：

- **Cluster URL**: `https://your-cluster.cloud.qdrant.io`
- **API Key**: 点击 "Generate API Key" 生成
- **Cluster ID**: 在集群详情中查看

#### 4. 配置环境变量

```bash
QDRANT_ACCESS_KEY=your_generated_api_key
QDRANT_ENDPOINT=https://your-cluster.cloud.qdrant.io
QDRANT_CLUSTER_ID=your_cluster_id
```

### 选项 2：自托管 Qdrant

#### 使用 Docker 部署

```bash
# 拉取 Qdrant 镜像
docker pull qdrant/qdrant:latest

# 创建数据目录
mkdir -p /var/lib/qdrant/storage

# 运行 Qdrant 容器
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  -v /var/lib/qdrant/storage:/qdrant/storage \
  -e QDRANT__SERVICE__API_KEY=your_api_key \
  qdrant/qdrant:latest
```

#### 使用 Docker Compose 部署

创建 `docker-compose.yml` 文件：

```yaml
version: '3.8'

services:
  qdrant:
    image: qdrant/qdrant:latest
    container_name: qdrant
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - ./qdrant_storage:/qdrant/storage
    environment:
      - QDRANT__SERVICE__API_KEY=your_api_key
    restart: unless-stopped
```

启动服务：

```bash
docker-compose up -d
```

#### 配置环境变量（自托管）

```bash
QDRANT_ACCESS_KEY=your_api_key
QDRANT_ENDPOINT=http://localhost:6333
QDRANT_CLUSTER_ID=local
```

### Qdrant Collection 架构

本系统使用**单 Collection 多租户架构**，具有以下特点：

#### Collection 配置

- **Collection 名称**: `conversation_memories`
- **向量维度**: 1536（使用 text-embedding-ada-002 或兼容模型）
- **距离度量**: Cosine（余弦相似度）
- **分片策略**: 按租户自动分片

#### Payload 结构

每个向量点包含以下 Payload 字段：

```json
{
  "tenant_id": "uuid",           // 租户 ID（索引字段）
  "session_id": "uuid",          // 会话 ID
  "memory_id": "uuid",           // 记忆 ID
  "memory_type": "string",       // 记忆类型
  "content": "string",           // 记忆内容
  "importance": 0.85,            // 重要性评分
  "created_at": "timestamp",     // 创建时间
  "metadata": {}                 // 其他元数据
}
```

#### 租户隔离索引

系统会为 `tenant_id` 字段创建索引，确保高效的租户级别过滤：

```json
{
  "field_name": "tenant_id",
  "field_schema": "keyword",
  "is_tenant": true
}
```

#### 自动初始化

系统启动时会自动执行以下操作：

1. 检查 Collection 是否存在
2. 如果不存在，创建 Collection 并配置：
   - 向量维度：1536
   - 距离度量：Cosine
   - 分片数量：根据预期租户数量自动计算
3. 创建 `tenant_id` 索引
4. 创建 `session_id` 索引
5. 创建 `memory_type` 索引

### Qdrant 性能优化

#### 1. 分片策略配置

根据租户数量配置分片：

- **< 10 租户**: 1 个分片
- **10-100 租户**: 2-4 个分片
- **100-1000 租户**: 4-8 个分片
- **> 1000 租户**: 8-16 个分片

#### 2. HNSW 索引参数优化

```json
{
  "hnsw_config": {
    "m": 16,                    // 每个节点的连接数（推荐 16-32）
    "ef_construct": 100,        // 构建时的搜索深度（推荐 100-200）
    "full_scan_threshold": 10000
  }
}
```

**参数说明**：

- `m`: 更大的值提高召回率但增加内存使用
- `ef_construct`: 更大的值提高索引质量但增加构建时间
- `full_scan_threshold`: 小于此数量的点使用全扫描

#### 3. 查询优化参数

```json
{
  "params": {
    "hnsw_ef": 128,             // 查询时的搜索深度（推荐 64-256）
    "exact": false              // 是否使用精确搜索
  }
}
```

#### 4. 定期维护

```bash
# 优化 Collection（压缩和重建索引）
curl -X POST 'http://localhost:6333/collections/conversation_memories/optimize' \
  -H 'api-key: your_api_key'

# 查看 Collection 信息
curl 'http://localhost:6333/collections/conversation_memories' \
  -H 'api-key: your_api_key'
```

### Qdrant 监控

#### 查看集群状态

```bash
# 健康检查
curl http://localhost:6333/healthz

# 查看所有 Collections
curl http://localhost:6333/collections \
  -H 'api-key: your_api_key'

# 查看 Collection 详情
curl http://localhost:6333/collections/conversation_memories \
  -H 'api-key: your_api_key'
```

#### 监控指标

Qdrant 提供 Prometheus 格式的监控指标：

```bash
# 访问指标端点
curl http://localhost:6333/metrics
```

主要监控指标：

- `qdrant_collections_total`: Collection 总数
- `qdrant_points_total`: 向量点总数
- `qdrant_search_duration_seconds`: 搜索延迟
- `qdrant_memory_usage_bytes`: 内存使用量

### Qdrant 备份和恢复

#### 创建快照

```bash
# 创建 Collection 快照
curl -X POST 'http://localhost:6333/collections/conversation_memories/snapshots' \
  -H 'api-key: your_api_key'
```

#### 下载快照

```bash
# 列出快照
curl 'http://localhost:6333/collections/conversation_memories/snapshots' \
  -H 'api-key: your_api_key'

# 下载快照
curl 'http://localhost:6333/collections/conversation_memories/snapshots/{snapshot_name}' \
  -H 'api-key: your_api_key' \
  -o snapshot.tar
```

#### 恢复快照

```bash
# 上传并恢复快照
curl -X PUT 'http://localhost:6333/collections/conversation_memories/snapshots/upload' \
  -H 'api-key: your_api_key' \
  -F 'snapshot=@snapshot.tar'
```

---

## Genkit AI 配置

### 获取 Genkit API Key

#### 1. 注册 Google AI Studio 账号

访问 [https://makersuite.google.com/app/apikey](https://makersuite.google.com/app/apikey) 注册账号。

#### 2. 创建 API Key

1. 登录 Google AI Studio
2. 点击 "Get API Key"
3. 选择或创建一个 Google Cloud 项目
4. 点击 "Create API Key"
5. 复制生成的 API Key

#### 3. 配置环境变量

```bash
GENKIT_API_KEY=your_generated_api_key
```

### Genkit 模型选择

本系统支持多种 Gemini 模型，根据业务需求选择：

#### 模型对比

| 模型 | 速度 | 质量 | 成本 | 适用场景 |
|------|------|------|------|----------|
| gemini-2.5-flash | ⚡⚡⚡ | ⭐⭐⭐ | 💰 | 快速响应、高并发 |
| gemini-2.5-pro | ⚡⚡ | ⭐⭐⭐⭐⭐ | 💰💰💰 | 复杂任务、高质量要求 |
| gemini-1.5-flash | ⚡⚡⚡ | ⭐⭐ | 💰 | 旧版快速模型 |
| gemini-1.5-pro | ⚡⚡ | ⭐⭐⭐⭐ | 💰💰 | 旧版高质量模型 |

#### 推荐配置

**开发环境**：

```bash
GENKIT_MODEL=gemini-2.5-flash
GENKIT_DEFAULT_TEMPERATURE=0.7
GENKIT_DEFAULT_MAX_TOKENS=2000
```

**生产环境（高并发）**：

```bash
GENKIT_MODEL=gemini-2.5-flash
GENKIT_DEFAULT_TEMPERATURE=0.5
GENKIT_DEFAULT_MAX_TOKENS=1500
```

**生产环境（高质量）**：

```bash
GENKIT_MODEL=gemini-2.5-pro
GENKIT_DEFAULT_TEMPERATURE=0.3
GENKIT_DEFAULT_MAX_TOKENS=3000
```

### Genkit 参数说明

#### Temperature（温度）

控制生成文本的随机性：

- **0.0**: 完全确定性，每次生成相同结果
- **0.3-0.5**: 较低随机性，适合事实性任务
- **0.7**: 平衡随机性和确定性（默认）
- **0.9-1.0**: 高随机性，适合创意性任务

#### Max Tokens（最大 Token 数）

控制单次生成的最大长度：

- **500-1000**: 简短回复
- **1500-2000**: 中等长度回复（默认）
- **3000-4000**: 长篇回复
- **8000+**: 超长文本生成

### Genkit 配额管理

#### 查看配额使用

访问 [Google Cloud Console](https://console.cloud.google.com/apis/api/generativelanguage.googleapis.com/quotas) 查看 API 配额使用情况。

#### 配额限制

免费层限制（可能变化，请查看官方文档）：

- **每分钟请求数**: 60 RPM
- **每天请求数**: 1500 RPD
- **每分钟 Token 数**: 32,000 TPM

付费层限制：

- **每分钟请求数**: 1000+ RPM
- **每天请求数**: 无限制
- **每分钟 Token 数**: 4,000,000+ TPM

#### 配额优化建议

1. **启用缓存**: 使用 Redis 缓存常见查询结果
2. **批量处理**: 合并多个请求减少 API 调用
3. **降级策略**: 配额耗尽时使用缓存响应
4. **监控告警**: 设置配额使用告警

### Genkit 错误处理

#### 常见错误码

| 错误码 | 说明 | 处理方式 |
|--------|------|----------|
| 400 | 请求参数错误 | 检查输入参数 |
| 401 | API Key 无效 | 检查 API Key 配置 |
| 403 | 权限不足 | 检查 API Key 权限 |
| 429 | 配额超限 | 启用降级策略 |
| 500 | 服务器错误 | 重试或降级 |
| 503 | 服务不可用 | 重试或降级 |

#### 重试策略

系统内置指数退避重试机制：

```go
// 重试配置
maxRetries := 3
baseDelay := 1 * time.Second
maxDelay := 10 * time.Second

// 重试间隔：1s, 2s, 4s
```

### Genkit 性能优化

#### 1. 并发控制

```bash
# 限制并发请求数
GENKIT_MAX_CONCURRENT_REQUESTS=10
```

#### 2. 超时配置

```bash
# 请求超时时间
GENKIT_REQUEST_TIMEOUT=30s
```

#### 3. 连接池配置

```bash
# HTTP 连接池大小
GENKIT_MAX_IDLE_CONNS=100
GENKIT_MAX_CONNS_PER_HOST=10
```

---

## 监控配置

### Prometheus 监控

#### 1. 安装 Prometheus

```bash
# 下载 Prometheus
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz

# 解压
tar xvfz prometheus-2.45.0.linux-amd64.tar.gz
cd prometheus-2.45.0.linux-amd64
```

#### 2. 配置 Prometheus

创建 `prometheus.yml` 配置文件：

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  # Genkit AI 服务监控
  - job_name: 'genkit-ai-service'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    
  # PostgreSQL 监控（需要 postgres_exporter）
  - job_name: 'postgresql'
    static_configs:
      - targets: ['localhost:9187']
      
  # Redis 监控（需要 redis_exporter）
  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']
      
  # Qdrant 监控
  - job_name: 'qdrant'
    static_configs:
      - targets: ['localhost:6333']
    metrics_path: '/metrics'
```

#### 3. 启动 Prometheus

```bash
./prometheus --config.file=prometheus.yml
```

访问 Prometheus UI: `http://localhost:9090`

### 关键监控指标

#### 应用层指标

```promql
# Flow 执行次数
genkit_flow_executions_total

# Flow 执行时间
genkit_flow_duration_seconds

# Token 使用量
genkit_token_usage_total

# 缓存命中率
rate(genkit_cache_hits_total[5m]) / (rate(genkit_cache_hits_total[5m]) + rate(genkit_cache_misses_total[5m]))

# API 请求延迟（P95）
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# 错误率
rate(genkit_flow_executions_total{status="error"}[5m]) / rate(genkit_flow_executions_total[5m])
```

#### 数据库指标

```promql
# 数据库连接数
pg_stat_database_numbackends

# 查询延迟
pg_stat_statements_mean_exec_time_seconds

# 缓存命中率
pg_stat_database_blks_hit / (pg_stat_database_blks_hit + pg_stat_database_blks_read)

# 死锁数量
rate(pg_stat_database_deadlocks[5m])
```

#### Redis 指标

```promql
# 内存使用
redis_memory_used_bytes

# 命令执行速率
rate(redis_commands_processed_total[5m])

# 缓存命中率
redis_keyspace_hits_total / (redis_keyspace_hits_total + redis_keyspace_misses_total)

# 连接数
redis_connected_clients
```

#### Qdrant 指标

```promql
# 向量点总数
qdrant_points_total

# 搜索延迟
qdrant_search_duration_seconds

# 内存使用
qdrant_memory_usage_bytes

# Collection 数量
qdrant_collections_total
```

### Grafana 可视化

#### 1. 安装 Grafana

```bash
# Ubuntu/Debian
sudo apt-get install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y grafana

# 启动服务
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
```

#### 2. 配置数据源

1. 访问 Grafana: `http://localhost:3000`（默认用户名/密码: admin/admin）
2. 添加 Prometheus 数据源：
   - Configuration → Data Sources → Add data source
   - 选择 Prometheus
   - URL: `http://localhost:9090`
   - 点击 "Save & Test"

#### 3. 导入仪表板

推荐的 Grafana 仪表板：

- **PostgreSQL**: Dashboard ID 9628
- **Redis**: Dashboard ID 11835
- **Go 应用**: Dashboard ID 10826

#### 4. 自定义仪表板

创建包含以下面板的自定义仪表板：

1. **Flow 执行统计**
   - Flow 执行次数（按 Flow 名称分组）
   - Flow 执行时间分布
   - Flow 错误率

2. **Token 使用统计**
   - Token 使用量趋势
   - 按租户的 Token 使用量
   - Token 配额使用率

3. **缓存性能**
   - 缓存命中率
   - 缓存大小
   - 缓存操作延迟

4. **API 性能**
   - 请求速率
   - 响应时间（P50, P95, P99）
   - 错误率

5. **资源使用**
   - CPU 使用率
   - 内存使用率
   - 磁盘 I/O
   - 网络流量

### 日志聚合

#### 使用 ELK Stack

##### 1. 安装 Elasticsearch

```bash
# 添加 Elasticsearch 仓库
wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
echo "deb https://artifacts.elastic.co/packages/8.x/apt stable main" | sudo tee /etc/apt/sources.list.d/elastic-8.x.list

# 安装
sudo apt-get update
sudo apt-get install -y elasticsearch

# 启动服务
sudo systemctl enable elasticsearch
sudo systemctl start elasticsearch
```

##### 2. 安装 Logstash

```bash
sudo apt-get install -y logstash
```

配置 Logstash (`/etc/logstash/conf.d/genkit.conf`):

```conf
input {
  file {
    path => "/path/to/genkit-ai-service/logs/*.log"
    start_position => "beginning"
    codec => json
  }
}

filter {
  json {
    source => "message"
  }
  
  date {
    match => ["timestamp", "ISO8601"]
    target => "@timestamp"
  }
}

output {
  elasticsearch {
    hosts => ["localhost:9200"]
    index => "genkit-logs-%{+YYYY.MM.dd}"
  }
}
```

启动 Logstash:

```bash
sudo systemctl enable logstash
sudo systemctl start logstash
```

##### 3. 安装 Kibana

```bash
sudo apt-get install -y kibana

# 启动服务
sudo systemctl enable kibana
sudo systemctl start kibana
```

访问 Kibana: `http://localhost:5601`

### 告警配置

#### Prometheus Alertmanager

##### 1. 安装 Alertmanager

```bash
wget https://github.com/prometheus/alertmanager/releases/download/v0.26.0/alertmanager-0.26.0.linux-amd64.tar.gz
tar xvfz alertmanager-0.26.0.linux-amd64.tar.gz
cd alertmanager-0.26.0.linux-amd64
```

##### 2. 配置告警规则

创建 `alerts.yml`:

```yaml
groups:
  - name: genkit_alerts
    interval: 30s
    rules:
      # Flow 错误率告警
      - alert: HighFlowErrorRate
        expr: rate(genkit_flow_executions_total{status="error"}[5m]) / rate(genkit_flow_executions_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Flow 错误率过高"
          description: "Flow {{ $labels.flow_name }} 错误率超过 5%"
      
      # API 响应时间告警
      - alert: HighAPILatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API 响应时间过长"
          description: "P95 响应时间超过 2 秒"
      
      # 数据库连接数告警
      - alert: HighDatabaseConnections
        expr: pg_stat_database_numbackends > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "数据库连接数过高"
          description: "数据库连接数超过 80"
      
      # Redis 内存使用告警
      - alert: HighRedisMemory
        expr: redis_memory_used_bytes / redis_memory_max_bytes > 0.9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Redis 内存使用过高"
          description: "Redis 内存使用超过 90%"
      
      # Token 配额告警
      - alert: TokenQuotaExhausted
        expr: genkit_token_quota_remaining < 1000
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Token 配额即将耗尽"
          description: "租户 {{ $labels.tenant_id }} 的 Token 配额不足 1000"
```

##### 3. 配置通知渠道

编辑 `alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'cluster']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'default'

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops@example.com'
        from: 'alertmanager@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alertmanager@example.com'
        auth_password: 'password'
    
    webhook_configs:
      - url: 'http://your-webhook-url'
        send_resolved: true
    
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

##### 4. 启动 Alertmanager

```bash
./alertmanager --config.file=alertmanager.yml
```

### 分布式追踪

#### 使用 Jaeger

##### 1. 安装 Jaeger

```bash
# 使用 Docker 快速启动
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest
```

##### 2. 配置环境变量

```bash
TRACING_ENABLED=true
TRACING_SERVICE_NAME=genkit-ai-service
TRACING_SERVICE_VERSION=1.0.0
TRACING_ENVIRONMENT=production
OTLP_ENDPOINT=localhost:4318
TRACING_SAMPLING_RATE=0.1
```

##### 3. 访问 Jaeger UI

访问 `http://localhost:16686` 查看追踪数据。

---

## 部署步骤

### 前置准备

#### 1. 克隆代码仓库

```bash
git clone https://github.com/your-org/genkit-ai-service.git
cd genkit-ai-service
```

#### 2. 安装依赖

```bash
# 下载 Go 依赖
go mod download

# 验证依赖
go mod verify
```

#### 3. 配置环境变量

```bash
# 复制示例配置文件
cp .env.example .env

# 编辑配置文件
vim .env
```

确保填写所有必需的配置项（参见[环境变量配置](#环境变量配置)章节）。

### 开发环境部署

#### 1. 启动依赖服务

```bash
# 使用 Docker Compose 启动 PostgreSQL、Redis、Qdrant
docker-compose up -d postgres redis qdrant
```

#### 2. 运行数据库迁移

```bash
# 执行数据库迁移
go run cmd/migrate/main.go up

# 验证迁移
go run cmd/migrate/main.go status
```

#### 3. 初始化系统

```bash
# 创建平台管理员和租户
go run cmd/init/main.go
```

系统会输出平台管理员的初始密码（如果未在 `.env` 中设置）。

#### 4. 启动应用

```bash
# 开发模式启动
go run cmd/server/main.go

# 或使用 air 实现热重载
air
```

应用将在 `http://localhost:8080` 启动。

#### 5. 验证部署

```bash
# 健康检查
curl http://localhost:8080/health

# 预期响应
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "services": {
    "database": "healthy",
    "redis": "healthy",
    "qdrant": "healthy"
  }
}
```

### 生产环境部署

#### 方式 1：直接部署

##### 1. 编译应用

```bash
# 编译生产版本
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o genkit-service -ldflags="-s -w" cmd/server/main.go

# 验证编译结果
./genkit-service --version
```

##### 2. 创建系统服务

创建 systemd 服务文件 `/etc/systemd/system/genkit-service.service`:

```ini
[Unit]
Description=Genkit AI Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=genkit
Group=genkit
WorkingDirectory=/opt/genkit-ai-service
EnvironmentFile=/opt/genkit-ai-service/.env
ExecStart=/opt/genkit-ai-service/genkit-service
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=genkit-service

# 安全加固
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/genkit-ai-service/logs

[Install]
WantedBy=multi-user.target
```

##### 3. 启动服务

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start genkit-service

# 设置开机自启
sudo systemctl enable genkit-service

# 查看服务状态
sudo systemctl status genkit-service

# 查看日志
sudo journalctl -u genkit-service -f
```

#### 方式 2：Docker 部署

##### 1. 构建 Docker 镜像

```bash
# 构建镜像
docker build -t genkit-ai-service:latest .

# 验证镜像
docker images | grep genkit-ai-service
```

##### 2. 运行容器

```bash
# 运行容器
docker run -d \
  --name genkit-service \
  --env-file .env \
  -p 8080:8080 \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  genkit-ai-service:latest

# 查看容器日志
docker logs -f genkit-service

# 查看容器状态
docker ps | grep genkit-service
```

##### 3. 使用 Docker Compose

创建 `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  app:
    image: genkit-ai-service:latest
    container_name: genkit-service
    env_file:
      - .env
    ports:
      - "8080:8080"
    volumes:
      - ./logs:/app/logs
    depends_on:
      - postgres
      - redis
    restart: unless-stopped
    networks:
      - genkit-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  postgres:
    image: pgvector/pgvector:pg14
    container_name: genkit-postgres
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    restart: unless-stopped
    networks:
      - genkit-network

  redis:
    image: redis:7-alpine
    container_name: genkit-redis
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    restart: unless-stopped
    networks:
      - genkit-network

volumes:
  postgres_data:
  redis_data:

networks:
  genkit-network:
    driver: bridge
```

启动服务：

```bash
docker-compose -f docker-compose.prod.yml up -d
```

#### 方式 3：Kubernetes 部署

##### 1. 创建 Namespace

```bash
kubectl create namespace genkit-ai
```

##### 2. 创建 ConfigMap

```bash
kubectl create configmap genkit-config \
  --from-env-file=.env \
  -n genkit-ai
```

##### 3. 创建 Secret

```bash
# 创建数据库密码 Secret
kubectl create secret generic db-secret \
  --from-literal=password=${DB_PASSWORD} \
  -n genkit-ai

# 创建 Redis 密码 Secret
kubectl create secret generic redis-secret \
  --from-literal=password=${REDIS_PASSWORD} \
  -n genkit-ai

# 创建 Genkit API Key Secret
kubectl create secret generic genkit-secret \
  --from-literal=api-key=${GENKIT_API_KEY} \
  -n genkit-ai

# 创建 Qdrant Secret
kubectl create secret generic qdrant-secret \
  --from-literal=access-key=${QDRANT_ACCESS_KEY} \
  -n genkit-ai
```

##### 4. 创建 Deployment

创建 `k8s/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: genkit-service
  namespace: genkit-ai
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
        image: genkit-ai-service:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: SERVER_PORT
          value: "8080"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: password
        - name: GENKIT_API_KEY
          valueFrom:
            secretKeyRef:
              name: genkit-secret
              key: api-key
        - name: QDRANT_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: qdrant-secret
              key: access-key
        envFrom:
        - configMapRef:
            name: genkit-config
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
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
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        volumeMounts:
        - name: logs
          mountPath: /app/logs
      volumes:
      - name: logs
        emptyDir: {}
```

##### 5. 创建 Service

创建 `k8s/service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: genkit-service
  namespace: genkit-ai
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

##### 6. 创建 Ingress

创建 `k8s/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: genkit-ingress
  namespace: genkit-ai
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

##### 7. 应用配置

```bash
# 应用所有配置
kubectl apply -f k8s/

# 查看部署状态
kubectl get pods -n genkit-ai
kubectl get svc -n genkit-ai
kubectl get ingress -n genkit-ai

# 查看日志
kubectl logs -f deployment/genkit-service -n genkit-ai
```

##### 8. 配置水平自动扩缩容（HPA）

创建 `k8s/hpa.yaml`:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: genkit-service-hpa
  namespace: genkit-ai
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
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
      - type: Pods
        value: 2
        periodSeconds: 30
      selectPolicy: Max
```

应用 HPA:

```bash
kubectl apply -f k8s/hpa.yaml
kubectl get hpa -n genkit-ai
```

### 数据库迁移

#### 执行迁移

```bash
# 查看迁移状态
go run cmd/migrate/main.go status

# 执行所有待执行的迁移
go run cmd/migrate/main.go up

# 回滚最后一次迁移
go run cmd/migrate/main.go down

# 回滚到指定版本
go run cmd/migrate/main.go goto <version>
```

#### 生产环境迁移最佳实践

1. **备份数据库**：迁移前务必备份

   ```bash
   pg_dump -U postgres -d ai_service > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **在测试环境验证**：先在测试环境执行迁移

3. **维护窗口**：在低峰期执行迁移

4. **监控迁移过程**：实时监控数据库性能

5. **准备回滚方案**：确保可以快速回滚

### 零停机部署

#### 使用蓝绿部署

1. **部署新版本**（绿环境）

   ```bash
   kubectl apply -f k8s/deployment-green.yaml
   ```

2. **验证新版本**

   ```bash
   kubectl port-forward deployment/genkit-service-green 8081:8080
   curl http://localhost:8081/health
   ```

3. **切换流量**

   ```bash
   kubectl patch service genkit-service -p '{"spec":{"selector":{"version":"green"}}}'
   ```

4. **监控新版本**
   - 观察错误率
   - 观察响应时间
   - 观察资源使用

5. **回滚（如果需要）**

   ```bash
   kubectl patch service genkit-service -p '{"spec":{"selector":{"version":"blue"}}}'
   ```

#### 使用滚动更新

```bash
# 更新镜像
kubectl set image deployment/genkit-service \
  genkit-service=genkit-ai-service:v2.0.0 \
  -n genkit-ai

# 查看滚动更新状态
kubectl rollout status deployment/genkit-service -n genkit-ai

# 暂停滚动更新
kubectl rollout pause deployment/genkit-service -n genkit-ai

# 恢复滚动更新
kubectl rollout resume deployment/genkit-service -n genkit-ai

# 回滚到上一个版本
kubectl rollout undo deployment/genkit-service -n genkit-ai

# 回滚到指定版本
kubectl rollout undo deployment/genkit-service --to-revision=2 -n genkit-ai
```

---

## 健康检查

### 健康检查端点

#### 基础健康检查

```bash
GET /health
```

响应示例：

```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "1.0.0",
  "services": {
    "database": "healthy",
    "redis": "healthy",
    "qdrant": "healthy",
    "genkit": "healthy"
  }
}
```

#### 详细健康检查

```bash
GET /health/detailed
```

响应示例：

```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "1.0.0",
  "uptime": "72h30m15s",
  "services": {
    "database": {
      "status": "healthy",
      "latency_ms": 2.5,
      "connections": {
        "active": 15,
        "idle": 5,
        "max": 25
      }
    },
    "redis": {
      "status": "healthy",
      "latency_ms": 1.2,
      "memory_used_mb": 128,
      "connected_clients": 10
    },
    "qdrant": {
      "status": "healthy",
      "latency_ms": 5.8,
      "collections": 1,
      "points_total": 150000
    },
    "genkit": {
      "status": "healthy",
      "model": "gemini-2.5-flash",
      "quota_remaining": 50000
    }
  },
  "metrics": {
    "requests_total": 1000000,
    "requests_per_second": 150,
    "average_response_time_ms": 45,
    "error_rate": 0.001
  }
}
```

### 就绪检查

```bash
GET /ready
```

用于 Kubernetes readinessProbe，检查服务是否准备好接收流量。

### 存活检查

```bash
GET /live
```

用于 Kubernetes livenessProbe，检查服务是否存活。

### 监控健康状态

#### 使用脚本监控

创建 `scripts/health_check.sh`:

```bash
#!/bin/bash

ENDPOINT="http://localhost:8080/health"
MAX_RETRIES=3
RETRY_DELAY=5

for i in $(seq 1 $MAX_RETRIES); do
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" $ENDPOINT)
  
  if [ $HTTP_CODE -eq 200 ]; then
    echo "✓ 健康检查通过"
    exit 0
  else
    echo "✗ 健康检查失败 (HTTP $HTTP_CODE), 重试 $i/$MAX_RETRIES"
    sleep $RETRY_DELAY
  fi
done

echo "✗ 健康检查失败，已达最大重试次数"
exit 1
```

#### 定时健康检查

```bash
# 添加到 crontab
*/5 * * * * /path/to/scripts/health_check.sh || /path/to/scripts/alert.sh
```

---

## 故障排查

### 常见问题

#### 1. 应用无法启动

**症状**：应用启动失败或立即退出

**可能原因和解决方案**：

##### 数据库连接失败

```bash
# 检查错误日志
tail -f logs/app.log | grep -i "database"

# 常见错误信息
Error: failed to connect to database: connection refused
```

**解决方案**：
- 检查 PostgreSQL 是否运行：`sudo systemctl status postgresql`
- 检查数据库配置：验证 `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`
- 检查网络连接：`telnet localhost 5432`
- 检查防火墙规则：确保端口 5432 开放

##### Redis 连接失败

```bash
# 检查 Redis 状态
redis-cli ping

# 如果设置了密码
redis-cli -a your_password ping
```

**解决方案**：
- 检查 Redis 是否运行：`sudo systemctl status redis`
- 检查 Redis 配置：验证 `REDIS_HOST`、`REDIS_PORT`、`REDIS_PASSWORD`
- 检查 Redis 密码：确保密码正确
- 检查 Redis 绑定地址：编辑 `/etc/redis/redis.conf`

##### Qdrant 连接失败

```bash
# 检查 Qdrant 健康状态
curl http://localhost:6333/healthz

# 或使用 API Key
curl -H "api-key: your_key" https://your-cluster.cloud.qdrant.io/healthz
```

**解决方案**：
- 检查 Qdrant 配置：验证 `QDRANT_ENDPOINT`、`QDRANT_ACCESS_KEY`
- 检查网络连接：确保可以访问 Qdrant 端点
- 检查 API Key：确保 Key 有效且未过期
- 检查防火墙：确保可以访问 Qdrant 端口

##### 环境变量缺失

```bash
# 检查必需的环境变量
env | grep -E "DB_|REDIS_|QDRANT_|GENKIT_|JWT_"
```

**解决方案**：
- 确保 `.env` 文件存在且位于正确位置
- 检查所有必需的环境变量是否已设置
- 验证环境变量值的格式是否正确

#### 2. API 请求失败

**症状**：API 返回 500 错误或超时

##### 数据库查询慢

```bash
# 查看慢查询日志
sudo tail -f /var/log/postgresql/postgresql-*.log | grep "duration"

# 查看当前活动查询
psql -U postgres -d ai_service -c "SELECT pid, now() - query_start as duration, query FROM pg_stat_activity WHERE state = 'active' ORDER BY duration DESC;"
```

**解决方案**：
- 添加缺失的索引
- 优化查询语句
- 增加数据库连接池大小
- 升级数据库硬件

##### Redis 内存不足

```bash
# 检查 Redis 内存使用
redis-cli INFO memory

# 查看内存使用率
redis-cli INFO stats | grep used_memory
```

**解决方案**：
- 增加 Redis 最大内存：编辑 `redis.conf` 中的 `maxmemory`
- 调整淘汰策略：设置 `maxmemory-policy`
- 清理过期键：`redis-cli --scan --pattern "*" | xargs redis-cli DEL`
- 升级 Redis 服务器

##### Genkit API 配额耗尽

```bash
# 检查日志中的配额错误
tail -f logs/app.log | grep -i "quota"

# 常见错误
Error: API quota exceeded (429)
```

**解决方案**：
- 检查 Google Cloud Console 中的配额使用情况
- 启用缓存减少 API 调用
- 升级到付费计划
- 实施请求限流

#### 3. 向量检索性能差

**症状**：向量搜索响应时间过长

```bash
# 检查 Qdrant 性能指标
curl http://localhost:6333/metrics | grep search_duration
```

**解决方案**：

##### 优化 HNSW 参数

```bash
# 更新 Collection 配置
curl -X PATCH 'http://localhost:6333/collections/conversation_memories' \
  -H 'Content-Type: application/json' \
  -H 'api-key: your_key' \
  -d '{
    "hnsw_config": {
      "m": 32,
      "ef_construct": 200
    }
  }'
```

##### 增加查询时的搜索深度

在代码中调整 `hnsw_ef` 参数：

```go
searchParams := &qdrant.SearchParams{
    HnswEf: 128,  // 增加到 128 或更高
}
```

##### 优化 Payload 索引

```bash
# 为常用字段创建索引
curl -X PUT 'http://localhost:6333/collections/conversation_memories/index' \
  -H 'Content-Type: application/json' \
  -H 'api-key: your_key' \
  -d '{
    "field_name": "tenant_id",
    "field_schema": "keyword"
  }'
```

#### 4. 内存泄漏

**症状**：应用内存使用持续增长

```bash
# 监控内存使用
top -p $(pgrep genkit-service)

# 或使用 htop
htop -p $(pgrep genkit-service)
```

**解决方案**：

##### 分析内存使用

```bash
# 生成内存 profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# 分析 profile
go tool pprof heap.prof
```

##### 检查 Goroutine 泄漏

```bash
# 查看 Goroutine 数量
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

##### 常见原因

- 未关闭的数据库连接
- 未关闭的 HTTP 连接
- 缓存无限增长
- Goroutine 泄漏

**修复方法**：
- 确保所有资源正确关闭
- 设置缓存大小限制
- 使用 context 超时控制 Goroutine 生命周期
- 定期重启应用（临时方案）

#### 5. 认证失败

**症状**：API 返回 401 Unauthorized

##### JWT Token 无效

```bash
# 检查 Token 格式
echo "your_token" | cut -d. -f2 | base64 -d | jq .

# 验证 Token 签名
# 使用 jwt.io 或类似工具验证
```

**解决方案**：
- 检查 `JWT_SECRET` 配置是否正确
- 确保 Token 未过期
- 验证 Token 签名算法
- 检查 Token 的 issuer 和 audience

##### Token 已过期

**解决方案**：
- 使用 Refresh Token 获取新的 Access Token
- 调整 Token 有效期：修改 `ACCESS_TOKEN_TTL`
- 实现 Token 自动刷新机制

#### 6. 租户隔离问题

**症状**：用户可以访问其他租户的数据

**检查步骤**：

```bash
# 查看审计日志
tail -f logs/app.log | grep "permission_denied"

# 检查数据库查询
tail -f logs/app.log | grep "tenant_id"
```

**解决方案**：
- 检查服务层是否正确验证租户 ID
- 确保所有查询都包含 `tenant_id` 过滤
- 检查 JWT Claims 中的租户 ID
- 审查权限验证逻辑

### 日志分析

#### 查看应用日志

```bash
# 实时查看日志
tail -f logs/app.log

# 查看错误日志
tail -f logs/app.log | grep -i "error"

# 查看特定租户的日志
tail -f logs/app.log | grep "tenant_id.*your-tenant-id"

# 查看特定时间段的日志
grep "2024-01-01T10:" logs/app.log
```

#### 日志级别

根据环境调整日志级别：

- **开发环境**：`LOG_LEVEL=debug`
- **测试环境**：`LOG_LEVEL=info`
- **生产环境**：`LOG_LEVEL=warn` 或 `error`

#### 结构化日志查询

如果使用 JSON 格式日志：

```bash
# 查询特定字段
cat logs/app.log | jq 'select(.level == "error")'

# 统计错误类型
cat logs/app.log | jq -r 'select(.level == "error") | .error' | sort | uniq -c

# 查询慢请求
cat logs/app.log | jq 'select(.duration_ms > 1000)'
```

### 性能调优

#### 数据库优化

```sql
-- 查看表大小
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 查看索引使用情况
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC;

-- 查看未使用的索引
SELECT 
    schemaname,
    tablename,
    indexname
FROM pg_stat_user_indexes
WHERE idx_scan = 0
AND indexname NOT LIKE '%_pkey';

-- 分析表统计信息
ANALYZE VERBOSE;

-- 重建索引
REINDEX TABLE conversation_memories;
```

#### Redis 优化

```bash
# 查看慢查询
redis-cli SLOWLOG GET 10

# 查看内存碎片率
redis-cli INFO memory | grep mem_fragmentation_ratio

# 如果碎片率过高，重启 Redis
sudo systemctl restart redis
```

#### Qdrant 优化

```bash
# 优化 Collection
curl -X POST 'http://localhost:6333/collections/conversation_memories/optimize' \
  -H 'api-key: your_key'

# 查看 Collection 统计
curl 'http://localhost:6333/collections/conversation_memories' \
  -H 'api-key: your_key' | jq .
```

### 备份和恢复

#### 数据库备份

```bash
# 完整备份
pg_dump -U postgres -d ai_service -F c -f backup_$(date +%Y%m%d_%H%M%S).dump

# 仅备份数据（不包括结构）
pg_dump -U postgres -d ai_service -a -F c -f data_backup_$(date +%Y%m%d_%H%M%S).dump

# 仅备份结构（不包括数据）
pg_dump -U postgres -d ai_service -s -F c -f schema_backup_$(date +%Y%m%d_%H%M%S).dump
```

#### 数据库恢复

```bash
# 恢复完整备份
pg_restore -U postgres -d ai_service -c backup_20240101_120000.dump

# 恢复到新数据库
createdb -U postgres ai_service_restore
pg_restore -U postgres -d ai_service_restore backup_20240101_120000.dump
```

#### Redis 备份

```bash
# 触发 RDB 快照
redis-cli BGSAVE

# 复制 RDB 文件
cp /var/lib/redis/dump.rdb /backup/redis_backup_$(date +%Y%m%d_%H%M%S).rdb

# 使用 AOF 备份
cp /var/lib/redis/appendonly.aof /backup/redis_aof_$(date +%Y%m%d_%H%M%S).aof
```

#### Qdrant 备份

```bash
# 创建快照
curl -X POST 'http://localhost:6333/collections/conversation_memories/snapshots' \
  -H 'api-key: your_key'

# 下载快照
curl 'http://localhost:6333/collections/conversation_memories/snapshots/snapshot-2024-01-01-12-00-00' \
  -H 'api-key: your_key' \
  -o qdrant_backup_$(date +%Y%m%d_%H%M%S).tar
```

### 监控告警

#### 关键指标阈值

| 指标 | 警告阈值 | 严重阈值 | 说明 |
|------|---------|---------|------|
| CPU 使用率 | 70% | 90% | 持续 5 分钟 |
| 内存使用率 | 80% | 95% | 持续 5 分钟 |
| 磁盘使用率 | 80% | 90% | 任意时刻 |
| API 错误率 | 1% | 5% | 5 分钟内 |
| API P95 延迟 | 1s | 3s | 5 分钟内 |
| 数据库连接数 | 80% | 95% | 任意时刻 |
| Redis 内存使用 | 80% | 95% | 任意时刻 |
| Token 配额剩余 | 10% | 5% | 任意时刻 |

#### 告警响应流程

1. **收到告警**：查看告警详情和严重程度
2. **初步诊断**：检查相关日志和监控指标
3. **确定影响范围**：评估受影响的用户和功能
4. **采取行动**：根据问题类型执行相应的修复步骤
5. **验证修复**：确认问题已解决
6. **记录事件**：更新事件日志和知识库
7. **事后分析**：分析根本原因并制定预防措施

### 联系支持

如果遇到无法解决的问题，请联系技术支持：

- **邮箱**：support@example.com
- **Slack**：#genkit-support
- **工单系统**：https://support.example.com

提供以下信息以加快问题解决：

1. 问题描述和复现步骤
2. 错误日志（最近 100 行）
3. 系统配置信息
4. 监控指标截图
5. 已尝试的解决方案

---

## 附录

### 环境变量完整列表

请参考 `.env.example` 文件获取所有可用的环境变量及其说明。

### 端口使用

| 服务 | 端口 | 说明 |
|------|------|------|
| 应用服务 | 8080 | HTTP API |
| PostgreSQL | 5432 | 数据库 |
| Redis | 6379 | 缓存 |
| Qdrant | 6333 | 向量数据库 API |
| Qdrant | 6334 | 向量数据库 gRPC |
| Prometheus | 9090 | 监控 |
| Grafana | 3000 | 可视化 |
| Jaeger | 16686 | 追踪 UI |
| Elasticsearch | 9200 | 日志存储 |
| Kibana | 5601 | 日志可视化 |

### 有用的命令

```bash
# 查看应用版本
./genkit-service --version

# 查看配置
./genkit-service config show

# 验证配置
./genkit-service config validate

# 数据库迁移状态
go run cmd/migrate/main.go status

# 清理过期数据
go run cmd/cleanup/main.go --dry-run

# 生成 API 文档
go run cmd/docs/main.go

# 运行健康检查
./scripts/health_check.sh

# 性能测试
./scripts/load_test.sh
```

### 相关文档

- [API 文档](./api-documentation.md)
- [架构设计](./design.md)
- [需求文档](./requirements.md)
- [开发指南](./development-guide.md)

---

**部署文档版本**: 1.0.0  
**最后更新**: 2024-01-01  
**维护者**: DevOps Team
