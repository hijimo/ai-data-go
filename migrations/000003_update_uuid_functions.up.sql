-- 更新UUID生成函数为PostgreSQL内置的gen_random_uuid()
-- 这个函数从PostgreSQL 13开始内置，不需要额外扩展

-- 为所有表添加更新触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要updated_at字段的表添加触发器
CREATE TRIGGER update_projects_updated_at 
    BEFORE UPDATE ON projects 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_files_updated_at 
    BEFORE UPDATE ON files 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_vector_indexes_updated_at 
    BEFORE UPDATE ON vector_indexes 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_llm_providers_updated_at 
    BEFORE UPDATE ON llm_providers 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agents_updated_at 
    BEFORE UPDATE ON agents 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chat_sessions_updated_at 
    BEFORE UPDATE ON chat_sessions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_training_jobs_updated_at 
    BEFORE UPDATE ON training_jobs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 添加表级别的约束和检查
-- 确保状态字段的有效性
ALTER TABLE files ADD CONSTRAINT check_files_status CHECK (status IN (0, 1, 2));
ALTER TABLE document_versions ADD CONSTRAINT check_document_versions_status CHECK (status IN (0, 1, 2));
ALTER TABLE chunks ADD CONSTRAINT check_chunks_embedding_status CHECK (embedding_status IN (0, 1, 2));
ALTER TABLE vector_indexes ADD CONSTRAINT check_vector_indexes_status CHECK (status IN (0, 1, 2));
ALTER TABLE questions ADD CONSTRAINT check_questions_status CHECK (status IN (0, 1, 2));
ALTER TABLE questions ADD CONSTRAINT check_questions_difficulty CHECK (difficulty BETWEEN 1 AND 5);
ALTER TABLE tasks ADD CONSTRAINT check_tasks_status CHECK (status IN (0, 1, 2, 3));
ALTER TABLE tasks ADD CONSTRAINT check_tasks_progress CHECK (progress BETWEEN 0 AND 100);
ALTER TABLE training_jobs ADD CONSTRAINT check_training_jobs_status CHECK (status IN (0, 1, 2, 3));
ALTER TABLE training_jobs ADD CONSTRAINT check_training_jobs_progress CHECK (progress BETWEEN 0 AND 100);

-- 添加角色检查约束
ALTER TABLE project_members ADD CONSTRAINT check_project_members_role 
    CHECK (role IN ('owner', 'admin', 'member', 'viewer'));

-- 添加对话消息角色检查约束
ALTER TABLE chat_messages ADD CONSTRAINT check_chat_messages_role 
    CHECK (role IN ('user', 'assistant', 'system'));

-- 添加问题类型检查约束
ALTER TABLE questions ADD CONSTRAINT check_questions_type 
    CHECK (question_type IN ('factual', 'reasoning', 'application', 'analytical', 'creative'));

-- 添加LLM模型类型检查约束
ALTER TABLE llm_models ADD CONSTRAINT check_llm_models_type 
    CHECK (model_type IN ('chat', 'completion', 'embedding'));

-- 添加LLM提供商类型检查约束
ALTER TABLE llm_providers ADD CONSTRAINT check_llm_providers_type 
    CHECK (provider_type IN ('openai', 'azure', 'qianwen', 'claude', 'baichuan', 'chatglm'));

-- 添加向量提供商类型检查约束
ALTER TABLE vector_indexes ADD CONSTRAINT check_vector_indexes_provider 
    CHECK (provider IN ('adbpg', 'elasticsearch', 'milvus', 'pinecone', 'weaviate'));

-- 添加任务类型检查约束
ALTER TABLE tasks ADD CONSTRAINT check_tasks_type 
    CHECK (task_type IN ('document_process', 'question_generate', 'answer_generate', 
                        'vector_index', 'dataset_export', 'model_train', 'data_distill'));

-- 添加文件大小检查约束（最大10GB）
ALTER TABLE files ADD CONSTRAINT check_files_size CHECK (size > 0 AND size <= 10737418240);

-- 添加SHA256格式检查约束
ALTER TABLE files ADD CONSTRAINT check_files_sha256 CHECK (sha256 ~ '^[a-f0-9]{64}$');

-- 添加质量评分检查约束
ALTER TABLE answers ADD CONSTRAINT check_answers_quality_score 
    CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1));

-- 添加版本号检查约束
ALTER TABLE document_versions ADD CONSTRAINT check_document_versions_version CHECK (version > 0);

-- 添加序列号检查约束
ALTER TABLE chunks ADD CONSTRAINT check_chunks_sequence CHECK (sequence >= 0);

-- 添加分块数量检查约束
ALTER TABLE document_versions ADD CONSTRAINT check_document_versions_chunk_count CHECK (chunk_count >= 0);

-- 添加注释说明
COMMENT ON TABLE projects IS '项目表 - 存储项目基本信息和配置';
COMMENT ON TABLE project_members IS '项目成员表 - 存储项目成员关系和角色';
COMMENT ON TABLE files IS '文件表 - 存储上传文件的元数据信息';
COMMENT ON TABLE document_versions IS '文档版本表 - 存储文档的不同处理版本';
COMMENT ON TABLE chunks IS '文档块表 - 存储文档分块后的内容';
COMMENT ON TABLE vector_indexes IS '向量索引表 - 存储向量索引的配置和状态';
COMMENT ON TABLE vector_records IS '向量记录表 - 存储向量化记录的映射关系';
COMMENT ON TABLE llm_providers IS 'LLM提供商表 - 存储大语言模型提供商配置';
COMMENT ON TABLE llm_models IS 'LLM模型表 - 存储可用的大语言模型信息';
COMMENT ON TABLE agents IS 'Agent表 - 存储智能体配置和参数';
COMMENT ON TABLE chat_sessions IS '对话会话表 - 存储对话会话信息';
COMMENT ON TABLE chat_messages IS '对话消息表 - 存储对话消息内容';
COMMENT ON TABLE questions IS '问题表 - 存储生成的问题内容';
COMMENT ON TABLE answers IS '答案表 - 存储问题对应的答案';
COMMENT ON TABLE tasks IS '任务表 - 存储异步任务的执行状态';
COMMENT ON TABLE training_jobs IS '训练任务表 - 存储模型训练任务信息';

-- 添加列注释
COMMENT ON COLUMN files.status IS '文件状态: 0-上传中, 1-已完成, 2-处理失败';
COMMENT ON COLUMN document_versions.status IS '处理状态: 0-处理中, 1-已完成, 2-失败';
COMMENT ON COLUMN chunks.embedding_status IS '向量化状态: 0-未处理, 1-已完成, 2-失败';
COMMENT ON COLUMN vector_indexes.status IS '索引状态: 0-创建中, 1-可用, 2-错误';
COMMENT ON COLUMN questions.status IS '审核状态: 0-待审核, 1-已审核, 2-已拒绝';
COMMENT ON COLUMN questions.difficulty IS '难度等级: 1-5级';
COMMENT ON COLUMN tasks.status IS '任务状态: 0-处理中, 1-已完成, 2-失败, 3-已中断';
COMMENT ON COLUMN tasks.progress IS '任务进度: 0-100百分比';
COMMENT ON COLUMN training_jobs.status IS '训练状态: 0-提交中, 1-训练中, 2-已完成, 3-失败';
COMMENT ON COLUMN training_jobs.progress IS '训练进度: 0-100百分比';
COMMENT ON COLUMN answers.quality_score IS '答案质量评分: 0-1之间的小数';