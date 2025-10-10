-- 删除所有表（按依赖关系逆序）
DROP TABLE IF EXISTS training_jobs;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS answers;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_sessions;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS llm_models;
DROP TABLE IF EXISTS llm_providers;
DROP TABLE IF EXISTS vector_records;
DROP TABLE IF EXISTS vector_indexes;
DROP TABLE IF EXISTS chunks;
DROP TABLE IF EXISTS document_versions;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;

-- 删除UUID扩展
DROP EXTENSION IF EXISTS "uuid-ossp";