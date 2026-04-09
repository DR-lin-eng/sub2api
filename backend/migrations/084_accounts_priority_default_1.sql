-- 账号默认优先级从 50 调整为 1。
-- 0 仍然保留为可手动设置的更高优先级。

ALTER TABLE accounts
ALTER COLUMN priority SET DEFAULT 1;

COMMENT ON COLUMN accounts.priority IS '调度优先级(0-100，越小越高，默认1)';
