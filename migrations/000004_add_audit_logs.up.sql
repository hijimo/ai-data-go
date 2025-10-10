-- 添加审计日志表，用于记录敏感操作
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 为审计日志表创建索引
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_audit_logs_action ON audit_logs(action) WHERE is_deleted = FALSE;
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at) WHERE is_deleted = FALSE;
CREATE INDEX idx_audit_logs_success ON audit_logs(success) WHERE is_deleted = FALSE;

-- 添加审计日志表的约束
ALTER TABLE audit_logs ADD CONSTRAINT check_audit_logs_action 
    CHECK (action IN ('create', 'update', 'delete', 'view', 'export', 'import', 'login', 'logout', 'config_change'));

ALTER TABLE audit_logs ADD CONSTRAINT check_audit_logs_resource_type 
    CHECK (resource_type IN ('project', 'file', 'agent', 'llm_provider', 'vector_index', 'training_job', 'user', 'system'));

-- 添加表和列注释
COMMENT ON TABLE audit_logs IS '审计日志表 - 记录用户的敏感操作和系统事件';
COMMENT ON COLUMN audit_logs.action IS '操作类型: create, update, delete, view, export, import, login, logout, config_change';
COMMENT ON COLUMN audit_logs.resource_type IS '资源类型: project, file, agent, llm_provider, vector_index, training_job, user, system';
COMMENT ON COLUMN audit_logs.resource_id IS '资源ID，对于系统级操作可为空';
COMMENT ON COLUMN audit_logs.details IS '操作详情，包含操作前后的数据变化';
COMMENT ON COLUMN audit_logs.success IS '操作是否成功';
COMMENT ON COLUMN audit_logs.error_message IS '失败时的错误信息';