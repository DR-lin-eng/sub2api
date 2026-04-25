-- 补齐 users 表中仍缺失的运行时字段。
-- Ent 查询 users 时会选择完整 schema；任一列缺失都会导致登录/资料读取失败。

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS balance_notify_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS balance_notify_threshold_type VARCHAR NOT NULL DEFAULT 'fixed',
    ADD COLUMN IF NOT EXISTS balance_notify_threshold DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS balance_notify_extra_emails TEXT NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS total_recharged DECIMAL(20,8) NOT NULL DEFAULT 0;
