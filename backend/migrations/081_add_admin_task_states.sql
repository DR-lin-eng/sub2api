CREATE TABLE IF NOT EXISTS admin_task_states (
    task_kind   TEXT NOT NULL,
    task_id     TEXT NOT NULL,
    state_json  JSONB NOT NULL DEFAULT '{}'::jsonb,
    status      TEXT NOT NULL DEFAULT '',
    expires_at  TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_kind, task_id)
);

CREATE INDEX IF NOT EXISTS idx_admin_task_states_status_updated
    ON admin_task_states (status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_task_states_expires_at
    ON admin_task_states (expires_at)
    WHERE expires_at IS NOT NULL;
