-- 回滚审计日志表的创建

-- 删除表注释
COMMENT ON TABLE audit_logs IS NULL;
COMMENT ON COLUMN audit_logs.action IS NULL;
COMMENT ON COLUMN audit_logs.resource_type IS NULL;
COMMENT ON COLUMN audit_logs.resource_id IS NULL;
COMMENT ON COLUMN audit_logs.details IS NULL;
COMMENT ON COLUMN audit_logs.success IS NULL;
COMMENT ON COLUMN audit_logs.error_message IS NULL;

-- 删除索引
DROP INDEX IF EXISTS idx_audit_logs_success;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_user_id;

-- 删除表
DROP TABLE IF EXISTS audit_logs;