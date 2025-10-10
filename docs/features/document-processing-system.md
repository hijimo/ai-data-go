# 文档处理系统实现文档

## 概述

本文档描述了AI知识管理平台的文档处理系统的完整实现，包括文件上传、多格式文档解析和智能文本分块功能。

## 系统架构

### 核心组件

1. **文件上传模块** (`internal/storage/oss.go`)
   - OSS/S3客户端集成
   - 文件SHA256校验和去重
   - 支持多种存储后端

2. **文档解析模块** (`internal/processor/`)
   - 支持PDF、DOCX、Markdown、TXT、HTML格式
   - 统一的文档处理器接口
   - 内容提取和清理功能

3. **智能分块模块** (`internal/processor/chunker.go`)
   - 多种分块策略：固定长度、语义、结构、代码
   - 分块配置和预览API
   - 分块结果可视化

### 数据流程

```
文件上传 → 格式检测 → 文档解析 → 内容提取 → 智能分块 → 存储到数据库
```

## 功能特性

### 1. 文件上传功能

#### 支持的文件格式

- PDF文档 (.pdf)
- Word文档 (.doc, .docx)
- Markdown文档 (.md, .markdown)
- 纯文本文档 (.txt)
- HTML文档 (.html, .htm)
- JSON文档 (.json)
- XML文档 (.xml)
- CSV文档 (.csv)

#### 核心特性

- **文件去重**：基于SHA256哈希值自动检测重复文件
- **元数据提取**：自动提取文件大小、MIME类型、上传时间等信息
- **安全存储**：文件存储到OSS，数据库仅保存元数据
- **权限控制**：基于项目的文件隔离和访问控制

#### API接口

```http
POST /api/v1/files/upload          # 上传文件
GET  /api/v1/files                 # 获取文件列表
GET  /api/v1/files/{id}            # 获取文件详情
DELETE /api/v1/files/{id}          # 删除文件
GET  /api/v1/files/{id}/url        # 获取文件访问URL
```

### 2. 多格式文档解析

#### 解析器架构

```go
type DocumentProcessor interface {
    SupportedTypes() []string
    Parse(ctx context.Context, reader io.Reader, metadata *FileMetadata) (*Document, error)
}
```

#### 各格式解析器特性

**PDF解析器**

- 文本内容提取
- 页数估算
- 简单的结构识别
- 支持中英文混合文档

**DOCX解析器**

- XML结构解析
- 表格提取
- 图片信息提取
- 链接识别
- 文档属性提取

**Markdown解析器**

- 标题层级解析
- 表格解析
- 图片和链接提取
- 代码块识别
- Front Matter支持

**文本解析器**

- 编码检测和转换
- 简单表格识别
- URL和邮箱提取
- 标题模式识别

**HTML解析器**

- 标签清理和内容提取
- 表格结构解析
- 图片和链接提取
- 脚本和样式过滤
- 语言信息提取

#### 提取的信息结构

```go
type Document struct {
    Title       string                 // 文档标题
    Content     string                 // 文档内容
    Metadata    map[string]interface{} // 文档元数据
    Structure   *DocumentStructure     // 文档结构
    Images      []ImageInfo            // 图片信息
    Tables      []TableInfo            // 表格信息
    Links       []LinkInfo             // 链接信息
    Language    string                 // 文档语言
    WordCount   int                    // 字数统计
    PageCount   int                    // 页数
}
```

### 3. 智能文本分块

#### 分块策略

**1. 固定长度分块 (fixed_size)**

- 按固定字符数分割
- 支持重叠设置
- 保持句子完整性
- 适用于通用文档

**2. 语义分块 (semantic)**

- 基于段落边界分割
- 保持语义完整性
- 适用于文章和报告

**3. 结构化分块 (structure)**

- 基于文档结构分割
- 按章节和标题分块
- 适用于技术文档

**4. 代码分块 (code)**

- 基于代码结构分割
- 按函数和类分块
- 适用于代码文档

#### 分块配置

```go
type ChunkConfig struct {
    Strategy        ChunkStrategy // 分块策略
    MaxSize         int          // 最大字符数
    Overlap         int          // 重叠字符数
    Separators      []string     // 分隔符
    PreserveContext bool         // 保持上下文完整性
}
```

#### 分块管理API

```http
GET  /api/v1/chunk-configs/defaults     # 获取默认配置
POST /api/v1/chunk-configs/validate     # 验证配置
POST /api/v1/chunk-configs/recommend    # 获取推荐配置
POST /api/v1/documents/preview-chunks   # 预览分块结果
```

#### 分块可视化API

```http
GET  /api/v1/chunks/{version_id}/visualization  # 分块可视化
GET  /api/v1/chunks/{version_id}/statistics     # 分块统计
POST /api/v1/chunks/compare-strategies          # 策略比较
```

## 技术实现

### 存储架构

**文件存储**

- 使用阿里云OSS SDK V2存储文件内容
- 支持文件去重和版本管理
- 提供临时访问URL

**元数据存储**

- PostgreSQL存储文件元数据
- 支持逻辑删除和软删除
- 完整的关联关系管理

### 数据模型

**核心表结构**

```sql
-- 文件表
CREATE TABLE files (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    oss_path VARCHAR(500) NOT NULL,
    uploader_id UUID NOT NULL,
    status INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 文档版本表
CREATE TABLE document_versions (
    id UUID PRIMARY KEY,
    file_id UUID NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    chunk_config JSONB NOT NULL,
    chunk_count INTEGER DEFAULT 0,
    status INTEGER DEFAULT 0,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 文档块表
CREATE TABLE chunks (
    id UUID PRIMARY KEY,
    document_version_id UUID NOT NULL,
    sequence INTEGER NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    embedding_status INTEGER DEFAULT 0,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### 异步处理

**任务队列**

- 使用Redis + asynq实现异步任务处理
- 支持文档处理、分块生成等长时间任务
- 提供任务状态跟踪和进度更新

**处理流程**

1. 用户上传文件，立即返回文件ID
2. 异步处理文档解析和分块
3. 更新处理状态和结果
4. 通知用户处理完成

## 配置和部署

### 环境变量配置

```bash
# OSS配置 (使用阿里云OSS Go SDK V2)
# 注意：OSS_ACCESS_KEY_ID 和 OSS_ACCESS_KEY_SECRET 是SDK V2的标准环境变量名
OSS_ACCESS_KEY_ID=your_access_key_id
OSS_ACCESS_KEY_SECRET=your_access_key_secret
OSS_BUCKET_NAME=your_bucket_name
OSS_REGION=cn-hangzhou
OSS_ENDPOINT=

# 数据库配置
DATABASE_URL=postgres://user:pass@localhost:5432/aiplatform

# Redis配置
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Docker部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

## 测试

### 单元测试

- 文档处理器测试：`internal/processor/*_test.go`
- 分块器测试：`internal/processor/chunker_test.go`
- 服务层测试：`internal/service/*_test.go`
- 处理器测试：`internal/handler/*_test.go`

### 集成测试

- API接口测试
- 文件上传流程测试
- 文档处理流程测试

### 性能测试

- 大文件处理性能
- 并发上传测试
- 分块处理效率测试

## 监控和日志

### 监控指标

- 文件上传成功率
- 文档处理耗时
- 分块生成速度
- 存储使用量

### 日志记录

- 结构化日志格式
- 错误和异常记录
- 性能指标记录
- 用户操作审计

## 扩展性

### 支持新格式

1. 实现`DocumentProcessor`接口
2. 注册到`ProcessorManager`
3. 添加相应的测试

### 支持新分块策略

1. 实现`Chunker`接口
2. 注册到`ChunkerManager`
3. 更新配置管理

### 支持新存储后端

1. 实现`OSSClient`接口
2. 添加配置选项
3. 更新部署配置

## 总结

文档处理系统提供了完整的文档管理解决方案，包括：

1. **文件上传功能**：支持多种格式，自动去重，安全存储
2. **多格式解析**：统一接口，丰富的内容提取，结构化数据
3. **智能分块**：多种策略，配置灵活，可视化管理

系统设计遵循了模块化、可扩展的原则，为后续的向量化、检索和问答功能提供了坚实的基础。

## OSS SDK V2 升级说明

### 主要变化

本系统已升级到阿里云OSS Go SDK V2，主要变化包括：

1. **更好的性能**: 优化的HTTP客户端和连接池管理
2. **更强的安全性**: 默认使用V4签名，提供更高的安全性
3. **标准化**: 遵循AWS SDK的设计模式，API更加一致
4. **环境变量支持**: 原生支持标准环境变量名称
5. **上下文支持**: 所有API都支持context.Context
6. **错误处理**: 提供详细的错误信息和EC错误码用于问题诊断

### 配置方式

**推荐使用环境变量方式：**

```bash
export OSS_ACCESS_KEY_ID="your_access_key_id"
export OSS_ACCESS_KEY_SECRET="your_access_key_secret"
```

**代码中使用：**

```go
// 使用环境变量创建客户端（推荐）
client, err := storage.NewOSSClientFromEnv(bucketName, region)

// 或使用配置结构
config := &storage.OSSConfig{
    AccessKeyID:     "your_access_key_id",
    AccessKeySecret: "your_access_key_secret",
    BucketName:      "your_bucket_name",
    Region:          "cn-hangzhou",
}
client, err := storage.NewOSSClient(config)
```

### 相关文档

- [OSS SDK V2迁移指南](../oss-sdk-v2-migration.md)
- [OSS配置指南](../oss-setup-guide.md)
- [阿里云OSS Go SDK V2官方文档](https://help.aliyun.com/zh/oss/developer-reference/manual-for-go-sdk-v2/)
