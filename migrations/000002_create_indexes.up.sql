-- 项目相关索引（包含逻辑删除）
CREATE INDEX idx_projects_not_deleted ON projects(id) WHERE is_deleted = FALSE;
CREATE INDEX idx_projects_owner_id ON projects(owner_id) WHERE is_deleted = FALSE;

-- 项目成员索引
CREATE INDEX idx_project_members_project_id ON project_members(project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_project_members_user_id ON project_members(user_id) WHERE is_deleted = FALSE;
CREATE UNIQUE INDEX idx_project_members_unique ON project_members(project_id, user_id) WHERE is_deleted = FALSE;

-- 文件相关索引
CREATE INDEX idx_files_project_id_not_deleted ON files(project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_files_status ON files(status) WHERE is_deleted = FALSE;
CREATE INDEX idx_files_sha256 ON files(sha256);
CREATE INDEX idx_files_uploader_id ON files(uploader_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_files_created_at ON files(created_at) WHERE is_deleted = FALSE;

-- 文档版本索引
CREATE INDEX idx_document_versions_file_id ON document_versions(file_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_document_versions_status ON document_versions(status) WHERE is_deleted = FALSE;

-- 文档块索引
CREATE INDEX idx_chunks_document_version_id ON chunks(document_version_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_chunks_embedding_status ON chunks(embedding_status) WHERE is_deleted = FALSE;
CREATE INDEX idx_chunks_sequence ON chunks(document_version_id, sequence) WHERE is_deleted = FALSE;

-- 向量索引相关
CREATE INDEX idx_vector_indexes_project_id ON vector_indexes(project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_vector_indexes_status ON vector_indexes(status) WHERE is_deleted = FALSE;

-- 向量记录索引
CREATE INDEX idx_vector_records_chunk_id ON vector_records(chunk_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_vector_records_index_id ON vector_records(vector_index_id) WHERE is_deleted = FALSE;
CREATE UNIQUE INDEX idx_vector_records_unique ON vector_records(chunk_id, vector_index_id) WHERE is_deleted = FALSE;

-- LLM提供商和模型索引
CREATE INDEX idx_llm_providers_active ON llm_providers(is_active) WHERE is_deleted = FALSE;
CREATE INDEX idx_llm_models_provider_id ON llm_models(provider_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_llm_models_active ON llm_models(is_active) WHERE is_deleted = FALSE;

-- Agent相关索引
CREATE INDEX idx_agents_project_id ON agents(project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_agents_created_by ON agents(created_by) WHERE is_deleted = FALSE;
CREATE INDEX idx_agents_llm_model_id ON agents(llm_model_id) WHERE is_deleted = FALSE;

-- 对话相关索引
CREATE INDEX idx_chat_sessions_project_user ON chat_sessions(project_id, user_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_chat_sessions_agent_id ON chat_sessions(agent_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_chat_sessions_created_at ON chat_sessions(created_at) WHERE is_deleted = FALSE;

CREATE INDEX idx_chat_messages_session_id ON chat_messages(session_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_chat_messages_created_at ON chat_messages(session_id, created_at) WHERE is_deleted = FALSE;

-- 问题答案相关索引
CREATE INDEX idx_questions_project_id ON questions(project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_questions_chunk_id ON questions(chunk_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_questions_status ON questions(status) WHERE is_deleted = FALSE;
CREATE INDEX idx_questions_type ON questions(question_type) WHERE is_deleted = FALSE;

CREATE INDEX idx_answers_question_id ON answers(question_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_answers_llm_model_id ON answers(llm_model_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_answers_reviewed ON answers(is_reviewed) WHERE is_deleted = FALSE;

-- 任务相关索引
CREATE INDEX idx_tasks_project_type_status ON tasks(project_id, task_type, status) WHERE is_deleted = FALSE;
CREATE INDEX idx_tasks_status ON tasks(status) WHERE is_deleted = FALSE;
CREATE INDEX idx_tasks_created_at ON tasks(created_at) WHERE is_deleted = FALSE;

-- 训练任务索引
CREATE INDEX idx_training_jobs_project_id ON training_jobs(project_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_training_jobs_status ON training_jobs(status) WHERE is_deleted = FALSE;
CREATE INDEX idx_training_jobs_external_id ON training_jobs(external_job_id) WHERE is_deleted = FALSE;