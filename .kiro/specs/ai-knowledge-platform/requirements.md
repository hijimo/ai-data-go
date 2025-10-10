# 需求文档

## 介绍

本项目旨在构建一个前后端分离的大模型知识管理平台，支持RAG（检索增强生成）、模型蒸馏、SFT（监督微调）等功能。平台采用云厂商提供的在线API与托管服务，以轻量化MVP方式实现，同时保留良好的扩展性。

## 需求

### 需求 1：文档管理与存储

**用户故事：** 作为平台用户，我希望能够上传和管理各种格式的文档，以便构建知识库。

#### 验收标准

1. WHEN 用户上传文件 THEN 系统 SHALL 将文件存储到OSS并记录元数据到PostgreSQL
2. WHEN 文件上传完成 THEN 系统 SHALL 生成文件的SHA256哈希值和基本信息（大小、MIME类型、上传时间）
3. WHEN 用户查看文件列表 THEN 系统 SHALL 显示所有已上传文件的状态和基本信息
4. IF 文件格式不支持 THEN 系统 SHALL 返回明确的错误信息

### 需求 2：多格式文档处理

**用户故事：** 作为平台用户，我希望系统能够处理多种文档格式并智能分块。

#### 验收标准

1. WHEN 用户上传PDF文档 THEN 系统 SHALL 提取文本内容并保持格式结构
2. WHEN 用户上传Markdown文档 THEN 系统 SHALL 解析标题层级和代码块结构
3. WHEN 用户上传DOCX文档 THEN 系统 SHALL 提取文本、表格和图片信息
4. WHEN 用户上传TXT文档 THEN 系统 SHALL 进行编码检测和文本清理
5. WHEN 用户上传HTML文档 THEN 系统 SHALL 提取正文内容并过滤标签
6. WHEN 执行智能分块 THEN 系统 SHALL 基于语义进行智能分割并保持上下文完整性
7. WHEN 配置分块参数 THEN 系统 SHALL 支持1500-2000字符的可配置分块大小
8. WHEN 分块结果不满意 THEN 系统 SHALL 支持手动调整分块边界

### 需求 3：向量化与索引

**用户故事：** 作为平台用户，我希望系统能够将文本块转换为向量并建立索引，以支持语义检索。

#### 验收标准

1. WHEN 文本分块完成 THEN 系统 SHALL 调用云端Embedding API生成向量
2. WHEN 向量生成完成 THEN 系统 SHALL 将向量存储到ADB-PG向量数据库
3. WHEN 向量存储完成 THEN 系统 SHALL 建立向量索引以支持高效检索
4. IF Embedding API调用失败 THEN 系统 SHALL 重试并记录错误日志

### 需求 4：RAG检索与查询

**用户故事：** 作为平台用户，我希望能够通过自然语言查询知识库，获得相关的文档片段。

#### 验收标准

1. WHEN 用户输入查询文本 THEN 系统 SHALL 将查询转换为向量并执行相似度检索
2. WHEN 检索完成 THEN 系统 SHALL 返回相关度排序的文档片段
3. WHEN 用户选择重排序 THEN 系统 SHALL 支持rerank和CoT（思维链）增强
4. WHEN 用户需要混合检索 THEN 系统 SHALL 支持向量检索与关键词检索的结合

### 需求 5：数据集导出与格式化

**用户故事：** 作为平台用户，我希望能够将知识库数据导出为标准格式，用于模型训练。

#### 验收标准

1. WHEN 用户选择导出数据集 THEN 系统 SHALL 支持Alpaca和ShareGPT格式导出
2. WHEN 导出任务启动 THEN 系统 SHALL 异步生成JSONL格式文件并上传到OSS
3. WHEN 导出完成 THEN 系统 SHALL 通知用户并提供下载链接
4. IF 导出数据为空 THEN 系统 SHALL 返回明确的提示信息

### 需求 6：模型训练管理

**用户故事：** 作为平台用户，我希望能够基于导出的数据集启动SFT训练任务。

#### 验收标准

1. WHEN 用户发起训练任务 THEN 系统 SHALL 生成训练配置YAML并调用阿里百炼API
2. WHEN 训练任务提交 THEN 系统 SHALL 在training_jobs表中记录任务状态
3. WHEN 训练状态更新 THEN 系统 SHALL 通过回调或轮询方式同步状态
4. WHEN 训练完成 THEN 系统 SHALL 通知用户并记录模型信息

### 需求 7：用户权限与安全

**用户故事：** 作为系统管理员，我希望平台具备完善的权限控制和安全机制。

#### 验收标准

1. WHEN 用户访问系统 THEN 系统 SHALL 通过JWT或OIDC进行身份验证
2. WHEN 用户执行操作 THEN 系统 SHALL 基于RBAC进行权限检查
3. WHEN 敏感操作执行 THEN 系统 SHALL 记录审计日志
4. WHEN 存储API密钥 THEN 系统 SHALL 使用KMS进行加密存储

### 需求 8：系统监控与可观测性

**用户故事：** 作为运维人员，我希望能够监控系统运行状态和性能指标。

#### 验收标准

1. WHEN 系统运行 THEN 系统 SHALL 暴露Prometheus格式的监控指标
2. WHEN 异常发生 THEN 系统 SHALL 记录结构化日志并支持链路追踪
3. WHEN 关键指标异常 THEN 系统 SHALL 触发告警通知
4. WHEN 查看监控面板 THEN 系统 SHALL 提供Grafana仪表板展示关键指标

### 需求 9：向量存储适配器

**用户故事：** 作为系统架构师，我希望系统支持多种向量存储后端，便于未来迁移和扩展。

#### 验收标准

1. WHEN 系统初始化 THEN 系统 SHALL 通过VectorProvider接口抽象向量存储操作
2. WHEN 配置向量存储 THEN 系统 SHALL 支持运行时切换不同的provider实现
3. WHEN 使用ADB-PG THEN 系统 SHALL 提供完整的ADB-PG适配器实现
4. WHEN 需要迁移 THEN 系统 SHALL 支持向量数据的导出和导入

### 需求 10：大模型对话功能

**用户故事：** 作为平台用户，我希望能够与大模型进行对话，并基于知识库内容获得增强的回答。

#### 验收标准

1. WHEN 用户发起对话 THEN 系统 SHALL 支持多轮对话并维护会话上下文
2. WHEN 对话涉及知识查询 THEN 系统 SHALL 自动检索相关知识库内容并融入回答
3. WHEN 用户选择不同模型 THEN 系统 SHALL 支持切换不同的LLM提供商（GPT-4o、千问等）
4. WHEN 对话历史需要保存 THEN 系统 SHALL 将对话记录存储到数据库供后续分析

### 需求 11：Agent智能体管理

**用户故事：** 作为平台用户，我希望能够创建和管理专门的AI Agent，用于特定领域的知识问答和任务处理。

#### 验收标准

1. WHEN 用户创建Agent THEN 系统 SHALL 支持配置Agent的角色、知识库范围和行为参数
2. WHEN Agent执行任务 THEN 系统 SHALL 支持工具调用、知识检索和推理链组合
3. WHEN 多个Agent协作 THEN 系统 SHALL 支持Agent间的消息传递和任务分发
4. WHEN Agent性能评估 THEN 系统 SHALL 记录Agent的响应质量和用户满意度指标

### 需求 12：DeepSearch深度搜索

**用户故事：** 作为平台用户，我希望系统提供深度搜索能力，能够理解复杂查询意图并提供精准结果。

#### 验收标准

1. WHEN 用户输入复杂查询 THEN 系统 SHALL 分析查询意图并分解为多个子查询
2. WHEN 执行深度搜索 THEN 系统 SHALL 结合向量检索、关键词匹配、语义推理等多种策略
3. WHEN 搜索结果不满意 THEN 系统 SHALL 支持查询重写和结果重排序
4. WHEN 需要跨文档推理 THEN 系统 SHALL 支持多跳推理和知识图谱增强检索

### 需求 13：LLM模型管理

**用户故事：** 作为系统管理员，我希望能够统一管理各种LLM提供商的连接配置和模型列表。

#### 验收标准

1. WHEN 管理员添加LLM提供商 THEN 系统 SHALL 支持配置API密钥、端点地址和认证信息
2. WHEN 配置模型列表 THEN 系统 SHALL 按厂商分类管理可用模型（GPT系列、千问系列、Claude等）
3. WHEN 更新模型配置 THEN 系统 SHALL 支持模型参数设置（温度、最大token、系统提示词等）
4. WHEN 测试连接 THEN 系统 SHALL 提供连接测试功能验证API密钥和模型可用性
5. WHEN 存储敏感信息 THEN 系统 SHALL 使用KMS加密存储所有API密钥

### 需求 14：项目管理与数据隔离

**用户故事：** 作为平台用户，我希望能够创建不同的项目来组织和隔离文档、知识库和相关资源。

#### 验收标准

1. WHEN 用户创建项目 THEN 系统 SHALL 为项目分配独立的命名空间和资源隔离
2. WHEN 编辑项目 THEN 系统 SHALL 支持项目信息修改和配置更新
3. WHEN 删除项目 THEN 系统 SHALL 安全删除项目及其所有关联数据
4. WHEN 项目数据迁移 THEN 系统 SHALL 支持项目间的数据迁移和合并
5. WHEN 用户切换项目 THEN 系统 SHALL 只显示当前项目的文档、Agent和对话历史
6. WHEN 项目协作 THEN 系统 SHALL 支持项目成员管理和权限分配

### 需求 15：智能问题生成

**用户故事：** 作为平台用户，我希望系统能够基于文档内容自动生成高质量的问题。

#### 验收标准

1. WHEN 文档分块完成 THEN 系统 SHALL 从每个文本分块提取相关问题
2. WHEN 生成问题 THEN 系统 SHALL 支持事实性、推理性、应用性等多样化问题类型
3. WHEN 问题生成完成 THEN 系统 SHALL 自动生成问题标签树进行分类
4. WHEN 批量处理 THEN 系统 SHALL 支持批量问题生成和处理
5. WHEN 配置提示词 THEN 系统 SHALL 支持全局提示、问题生成提示、标签提示和领域树提示的配置

### 需求 16：训练数据集生成

**用户故事：** 作为平台用户，我希望能够基于生成的问题创建高质量的训练数据集。

#### 验收标准

1. WHEN 选择问题 THEN 系统 SHALL 支持从问题库中选择需要生成答案的问题
2. WHEN 生成答案 THEN 系统 SHALL 使用配置的LLM生成详细答案
3. WHEN 需要推理过程 THEN 系统 SHALL 支持思维链(Chain of Thought)推理模式
4. WHEN 质量控制 THEN 系统 SHALL 提供人工审核和编辑功能
5. WHEN 导出数据集 THEN 系统 SHALL 支持Alpaca格式和ShareGPT格式的JSON/JSONL导出

### 需求 17：任务管理与状态跟踪

**用户故事：** 作为平台用户，我希望系统能够高效管理各种异步任务并提供详细的状态跟踪。

#### 验收标准

1. WHEN 提交文档处理任务 THEN 系统 SHALL 支持大文件的异步处理
2. WHEN 执行问题生成任务 THEN 系统 SHALL 支持批量问题生成的异步处理
3. WHEN 执行答案生成任务 THEN 系统 SHALL 支持批量答案生成的异步处理
4. WHEN 执行数据蒸馏任务 THEN 系统 SHALL 支持数据质量优化处理
5. WHEN 查询任务状态 THEN 系统 SHALL 提供处理中(0)、已完成(1)、失败(2)、已中断(3)的状态跟踪
6. WHEN 任务失败 THEN 系统 SHALL 支持重试机制和错误处理
7. WHEN 复杂工作流需求 THEN 系统 SHALL 预留Temporal集成扩展点
