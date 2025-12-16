# 需求文档

## 简介

本文档定义了腾讯乐享知识库 API 集成模块的需求。该模块提供与乐享 API 的无缝集成，实现 access_token 的自动管理，让业务代码对 token 完全无感。模块支持知识库管理、知识节点管理、文件上传下载、知识反馈等核心功能。

## 术语表

- **LexiangClient**: 乐享 API 客户端，封装所有与乐享 API 的交互
- **TokenManager**: Token 管理器，负责 access_token 的获取、缓存和自动刷新
- **Space**: 知识库，用于组织和存储知识内容
- **Entry**: 知识节点，包括文件夹、文件、视频、音频、链接、线上文档等类型
- **COS**: 腾讯云对象存储服务，用于文件上传
- **state**: 文件上传临时标识，用于关联上传的文件与知识节点

## 需求

### 需求 1

**用户故事：** 作为开发者，我希望系统能自动管理乐享 API 的 access_token，以便我在调用业务接口时无需关心 token 的获取和刷新。

#### 验收标准

1. WHEN TokenManager 首次被请求获取 token THEN TokenManager SHALL 调用乐享 token API 获取新的 access_token 并缓存
2. WHEN 缓存的 token 距离过期时间小于 5 分钟 THEN TokenManager SHALL 自动刷新 token
3. WHEN 多个 goroutine 同时请求 token 且 token 需要刷新 THEN TokenManager SHALL 仅执行一次刷新操作
4. WHEN API 请求返回 401 状态码 THEN LexiangClient SHALL 自动使缓存的 token 失效并重新获取后重试请求一次
5. WHEN token 刷新成功 THEN TokenManager SHALL 根据响应中的 expires_in 计算并更新过期时间

### 需求 2

**用户故事：** 作为开发者，我希望能够通过统一的客户端接口调用乐享 API，以便简化 HTTP 请求的处理。

#### 验收标准

1. WHEN 创建 LexiangClient THEN LexiangClient SHALL 支持从环境变量读取 APP_KEY 和 APP_SECRET
2. WHEN 发送 HTTP 请求 THEN LexiangClient SHALL 自动在请求头中添加 Authorization Bearer token
3. WHEN 发送 HTTP 请求 THEN LexiangClient SHALL 自动设置 Content-Type 为 application/json; charset=utf-8
4. WHEN 需要设置自定义请求头（如 x-staff-id）THEN LexiangClient SHALL 提供 DoWithHeader 方法支持自定义请求头
5. WHEN HTTP 请求超时 THEN LexiangClient SHALL 在 30 秒后返回超时错误

### 需求 3

**用户故事：** 作为开发者，我希望能够管理知识库（创建、更新、删除、查询），以便组织和存储知识内容。

#### 验收标准

1. WHEN 调用 CreateSpace 方法 THEN LexiangClient SHALL 向乐享 API 发送创建知识库请求并返回包含知识库 ID 和根目录 ID 的响应
2. WHEN 调用 UpdateSpace 方法 THEN LexiangClient SHALL 向乐享 API 发送更新知识库请求
3. WHEN 调用 DeleteSpace 方法 THEN LexiangClient SHALL 向乐享 API 发送删除知识库请求
4. WHEN 调用 ListSpaces 方法 THEN LexiangClient SHALL 返回指定团队下的知识库列表并支持分页
5. WHEN 调用 GetSpace 方法 THEN LexiangClient SHALL 返回指定知识库的详细信息

### 需求 4

**用户故事：** 作为开发者，我希望能够管理知识节点（创建文件夹、上传文件、删除节点、查询列表），以便在知识库中组织内容。

#### 验收标准

1. WHEN 调用 CreateFolder 方法 THEN LexiangClient SHALL 在指定父节点下创建文件夹并返回节点信息
2. WHEN 调用 CreateFileEntry 方法 THEN LexiangClient SHALL 使用上传的 state 创建文件类型的知识节点
3. WHEN 调用 ReuploadFile 方法 THEN LexiangClient SHALL 更新指定节点的文件内容
4. WHEN 调用 DeleteEntry 方法 THEN LexiangClient SHALL 删除指定的知识节点
5. WHEN 调用 ListEntries 方法 THEN LexiangClient SHALL 返回指定知识库或父节点下的知识节点列表并支持分页
6. WHEN 调用 GetEntry 方法 THEN LexiangClient SHALL 返回指定知识节点的详细信息
7. WHEN 调用 GetEntryContent 方法且节点类型为 page THEN LexiangClient SHALL 返回线上文档的 HTML 内容

### 需求 5

**用户故事：** 作为开发者，我希望能够上传文件到乐享知识库，以便将本地文件存储到云端。

#### 验收标准

1. WHEN 调用 GetUploadSign 方法 THEN LexiangClient SHALL 返回包含 COS 上传签名和 state 的响应
2. WHEN 调用 UploadFileToCOS 方法 THEN LexiangClient SHALL 使用签名信息将文件上传到腾讯云 COS
3. WHEN 文件上传成功 THEN COS 响应 SHALL 包含 ETag 头
4. WHEN 调用 UploadFile 方法 THEN LexiangClient SHALL 完成获取签名、上传到 COS 的完整流程并返回 state
5. WHEN 序列化上传签名响应 THEN 系统 SHALL 正确解析 JSON 响应并映射到 UploadSignResponse 结构体
6. WHEN 反序列化上传签名响应后再序列化 THEN 系统 SHALL 产生等价的 JSON 结构

### 需求 6

**用户故事：** 作为开发者，我希望能够下载知识库中的附件，以便获取存储在云端的文件内容。

#### 验收标准

1. WHEN 调用 GetDocFile 方法 THEN LexiangClient SHALL 返回附件的详情信息包括下载链接
2. WHEN 调用 DownloadDocFile 方法 THEN LexiangClient SHALL 下载附件内容并返回文件数据和文件名

### 需求 7

**用户故事：** 作为开发者，我希望能够获取知识库的用户反馈列表，以便了解用户对知识内容的意见。

#### 验收标准

1. WHEN 调用 ListFeedbacks 方法 THEN LexiangClient SHALL 返回指定知识库的反馈列表并支持分页
2. WHEN 反馈列表响应包含 included 字段 THEN LexiangClient SHALL 正确解析关联的用户和节点信息

### 需求 8

**用户故事：** 作为开发者，我希望系统能够正确处理 API 错误，以便我能够根据错误类型采取相应的处理措施。

#### 验收标准

1. WHEN API 返回 400 状态码 THEN LexiangClient SHALL 返回包含请求参数错误信息的错误
2. WHEN API 返回 403 状态码 THEN LexiangClient SHALL 返回包含权限不足信息的错误
3. WHEN API 返回 404 状态码 THEN LexiangClient SHALL 返回包含资源不存在信息的错误
4. WHEN API 返回 429 状态码 THEN LexiangClient SHALL 返回包含请求频率超限信息的错误
5. WHEN API 返回 500 状态码 THEN LexiangClient SHALL 返回包含服务器内部错误信息的错误
