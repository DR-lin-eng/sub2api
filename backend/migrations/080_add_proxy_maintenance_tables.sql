CREATE TABLE IF NOT EXISTS proxy_maintenance_plans (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    cron_expression TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    source_proxy_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    max_results INTEGER NOT NULL DEFAULT 50,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_failure_reason TEXT NOT NULL DEFAULT '',
    paused_at TIMESTAMPTZ NULL,
    pause_reason TEXT NOT NULL DEFAULT '',
    max_failures_before_pause INTEGER NOT NULL DEFAULT 3,
    last_run_at TIMESTAMPTZ NULL,
    next_run_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proxy_maintenance_plans_due
    ON proxy_maintenance_plans (enabled, next_run_at);

CREATE TABLE IF NOT EXISTS proxy_maintenance_results (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES proxy_maintenance_plans(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    moved_accounts INTEGER NOT NULL DEFAULT 0,
    checked_proxies INTEGER NOT NULL DEFAULT 0,
    healthy_proxies INTEGER NOT NULL DEFAULT 0,
    failed_proxies INTEGER NOT NULL DEFAULT 0,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proxy_maintenance_results_plan_created
    ON proxy_maintenance_results (plan_id, created_at DESC);
