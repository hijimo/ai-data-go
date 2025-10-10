-- 删除所有索引
DROP INDEX IF EXISTS idx_training_jobs_external_id;
DROP INDEX IF EXISTS idx_training_jobs_status;
DROP INDEX IF EXISTS idx_training_jobs_project_id;

DROP INDEX IF EXISTS idx_tasks_created_at;
DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_project_type_status;

DROP INDEX IF EXISTS idx_answers_reviewed;
DROP INDEX IF EXISTS idx_answers_llm_model_id;
DROP INDEX IF EXISTS idx_answers_question_id;

DROP INDEX IF EXISTS idx_questions_type;
DROP INDEX IF EXISTS idx_questions_status;
DROP INDEX IF EXISTS idx_questions_chunk_id;
DROP INDEX IF EXISTS idx_questions_project_id;

DROP INDEX IF EXISTS idx_chat_messages_created_at;
DROP INDEX IF EXISTS idx_chat_messages_session_id;

DROP INDEX IF EXISTS idx_chat_sessions_created_at;
DROP INDEX IF EXISTS idx_chat_sessions_agent_id;
DROP INDEX IF EXISTS idx_chat_sessions_project_user;

DROP INDEX IF EXISTS idx_agents_llm_model_id;
DROP INDEX IF EXISTS idx_agents_created_by;
DROP INDEX IF EXISTS idx_agents_project_id;

DROP INDEX IF EXISTS idx_llm_models_active;
DROP INDEX IF EXISTS idx_llm_models_provider_id;
DROP INDEX IF EXISTS idx_llm_providers_active;

DROP INDEX IF EXISTS idx_vector_records_unique;
DROP INDEX IF EXISTS idx_vector_records_index_id;
DROP INDEX IF EXISTS idx_vector_records_chunk_id;

DROP INDEX IF EXISTS idx_vector_indexes_status;
DROP INDEX IF EXISTS idx_vector_indexes_project_id;

DROP INDEX IF EXISTS idx_chunks_sequence;
DROP INDEX IF EXISTS idx_chunks_embedding_status;
DROP INDEX IF EXISTS idx_chunks_document_version_id;

DROP INDEX IF EXISTS idx_document_versions_status;
DROP INDEX IF EXISTS idx_document_versions_file_id;

DROP INDEX IF EXISTS idx_files_created_at;
DROP INDEX IF EXISTS idx_files_uploader_id;
DROP INDEX IF EXISTS idx_files_sha256;
DROP INDEX IF EXISTS idx_files_status;
DROP INDEX IF EXISTS idx_files_project_id_not_deleted;

DROP INDEX IF EXISTS idx_project_members_unique;
DROP INDEX IF EXISTS idx_project_members_user_id;
DROP INDEX IF EXISTS idx_project_members_project_id;

DROP INDEX IF EXISTS idx_projects_owner_id;
DROP INDEX IF EXISTS idx_projects_not_deleted;