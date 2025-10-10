-- 回滚UUID函数更新和约束添加

-- 删除表注释
COMMENT ON TABLE projects IS NULL;
COMMENT ON TABLE project_members IS NULL;
COMMENT ON TABLE files IS NULL;
COMMENT ON TABLE document_versions IS NULL;
COMMENT ON TABLE chunks IS NULL;
COMMENT ON TABLE vector_indexes IS NULL;
COMMENT ON TABLE vector_records IS NULL;
COMMENT ON TABLE llm_providers IS NULL;
COMMENT ON TABLE llm_models IS NULL;
COMMENT ON TABLE agents IS NULL;
COMMENT ON TABLE chat_sessions IS NULL;
COMMENT ON TABLE chat_messages IS NULL;
COMMENT ON TABLE questions IS NULL;
COMMENT ON TABLE answers IS NULL;
COMMENT ON TABLE tasks IS NULL;
COMMENT ON TABLE training_jobs IS NULL;

-- 删除列注释
COMMENT ON COLUMN files.status IS NULL;
COMMENT ON COLUMN document_versions.status IS NULL;
COMMENT ON COLUMN chunks.embedding_status IS NULL;
COMMENT ON COLUMN vector_indexes.status IS NULL;
COMMENT ON COLUMN questions.status IS NULL;
COMMENT ON COLUMN questions.difficulty IS NULL;
COMMENT ON COLUMN tasks.status IS NULL;
COMMENT ON COLUMN tasks.progress IS NULL;
COMMENT ON COLUMN training_jobs.status IS NULL;
COMMENT ON COLUMN training_jobs.progress IS NULL;
COMMENT ON COLUMN answers.quality_score IS NULL;

-- 删除检查约束
ALTER TABLE files DROP CONSTRAINT IF EXISTS check_files_status;
ALTER TABLE files DROP CONSTRAINT IF EXISTS check_files_size;
ALTER TABLE files DROP CONSTRAINT IF EXISTS check_files_sha256;

ALTER TABLE document_versions DROP CONSTRAINT IF EXISTS check_document_versions_status;
ALTER TABLE document_versions DROP CONSTRAINT IF EXISTS check_document_versions_version;
ALTER TABLE document_versions DROP CONSTRAINT IF EXISTS check_document_versions_chunk_count;

ALTER TABLE chunks DROP CONSTRAINT IF EXISTS check_chunks_embedding_status;
ALTER TABLE chunks DROP CONSTRAINT IF EXISTS check_chunks_sequence;

ALTER TABLE vector_indexes DROP CONSTRAINT IF EXISTS check_vector_indexes_status;
ALTER TABLE vector_indexes DROP CONSTRAINT IF EXISTS check_vector_indexes_provider;

ALTER TABLE questions DROP CONSTRAINT IF EXISTS check_questions_status;
ALTER TABLE questions DROP CONSTRAINT IF EXISTS check_questions_difficulty;
ALTER TABLE questions DROP CONSTRAINT IF EXISTS check_questions_type;

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS check_tasks_status;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS check_tasks_progress;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS check_tasks_type;

ALTER TABLE training_jobs DROP CONSTRAINT IF EXISTS check_training_jobs_status;
ALTER TABLE training_jobs DROP CONSTRAINT IF EXISTS check_training_jobs_progress;

ALTER TABLE project_members DROP CONSTRAINT IF EXISTS check_project_members_role;

ALTER TABLE chat_messages DROP CONSTRAINT IF EXISTS check_chat_messages_role;

ALTER TABLE llm_models DROP CONSTRAINT IF EXISTS check_llm_models_type;

ALTER TABLE llm_providers DROP CONSTRAINT IF EXISTS check_llm_providers_type;

ALTER TABLE answers DROP CONSTRAINT IF EXISTS check_answers_quality_score;

-- 删除更新触发器
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_files_updated_at ON files;
DROP TRIGGER IF EXISTS update_vector_indexes_updated_at ON vector_indexes;
DROP TRIGGER IF EXISTS update_llm_providers_updated_at ON llm_providers;
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;
DROP TRIGGER IF EXISTS update_chat_sessions_updated_at ON chat_sessions;
DROP TRIGGER IF EXISTS update_training_jobs_updated_at ON training_jobs;

-- 删除触发器函数
DROP FUNCTION IF EXISTS update_updated_at_column();