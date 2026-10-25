-- 审计日志增加班级字段，用于按管辖班级隔离（fail-closed）。
-- 现有行的 class_name 置为空字符串，由后续写入补齐。
ALTER TABLE audit_logs ADD COLUMN class_name TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_audit_logs_class ON audit_logs(class_name);
