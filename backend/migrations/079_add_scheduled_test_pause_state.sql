-- 079_add_scheduled_test_pause_state.sql
-- Persist scheduled-test failure counters and auto-pause metadata.

ALTER TABLE scheduled_test_plans
    ADD COLUMN IF NOT EXISTS consecutive_failures INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_failure_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS paused_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS pause_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS max_failures_before_pause INT NOT NULL DEFAULT 3;

UPDATE scheduled_test_plans
SET max_failures_before_pause = 3
WHERE max_failures_before_pause <= 0;
