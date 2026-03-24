package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type proxyMaintenancePlanRepository struct {
	db *sql.DB
}

func NewProxyMaintenancePlanRepository(db *sql.DB) service.ProxyMaintenancePlanRepository {
	return &proxyMaintenancePlanRepository{db: db}
}

func (r *proxyMaintenancePlanRepository) Create(ctx context.Context, plan *service.ProxyMaintenancePlan) (*service.ProxyMaintenancePlan, error) {
	sourceProxyIDs, err := json.Marshal(plan.SourceProxyIDs)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO proxy_maintenance_plans (
			name, cron_expression, enabled, source_proxy_ids, max_results,
			consecutive_failures, last_failure_reason, paused_at, pause_reason,
			max_failures_before_pause, next_run_at, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())
		RETURNING
			id, name, cron_expression, enabled, source_proxy_ids, max_results,
			consecutive_failures, last_failure_reason, paused_at, pause_reason,
			max_failures_before_pause, last_run_at, next_run_at, created_at, updated_at
	`, plan.Name, plan.CronExpression, plan.Enabled, string(sourceProxyIDs), plan.MaxResults, plan.ConsecutiveFailures, plan.LastFailureReason, plan.PausedAt, plan.PauseReason, plan.MaxFailuresBeforePause, plan.NextRunAt)
	return scanProxyMaintenancePlan(row)
}

func (r *proxyMaintenancePlanRepository) GetByID(ctx context.Context, id int64) (*service.ProxyMaintenancePlan, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, name, cron_expression, enabled, source_proxy_ids, max_results,
			consecutive_failures, last_failure_reason, paused_at, pause_reason,
			max_failures_before_pause, last_run_at, next_run_at, created_at, updated_at
		FROM proxy_maintenance_plans WHERE id = $1
	`, id)
	return scanProxyMaintenancePlan(row)
}

func (r *proxyMaintenancePlanRepository) List(ctx context.Context) ([]*service.ProxyMaintenancePlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, name, cron_expression, enabled, source_proxy_ids, max_results,
			consecutive_failures, last_failure_reason, paused_at, pause_reason,
			max_failures_before_pause, last_run_at, next_run_at, created_at, updated_at
		FROM proxy_maintenance_plans
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanProxyMaintenancePlans(rows)
}

func (r *proxyMaintenancePlanRepository) ListDue(ctx context.Context, now time.Time) ([]*service.ProxyMaintenancePlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, name, cron_expression, enabled, source_proxy_ids, max_results,
			consecutive_failures, last_failure_reason, paused_at, pause_reason,
			max_failures_before_pause, last_run_at, next_run_at, created_at, updated_at
		FROM proxy_maintenance_plans
		WHERE enabled = true AND next_run_at <= $1
		ORDER BY next_run_at ASC
	`, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanProxyMaintenancePlans(rows)
}

func (r *proxyMaintenancePlanRepository) Update(ctx context.Context, plan *service.ProxyMaintenancePlan) (*service.ProxyMaintenancePlan, error) {
	sourceProxyIDs, err := json.Marshal(plan.SourceProxyIDs)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE proxy_maintenance_plans
		SET
			name = $2,
			cron_expression = $3,
			enabled = $4,
			source_proxy_ids = $5::jsonb,
			max_results = $6,
			consecutive_failures = $7,
			last_failure_reason = $8,
			paused_at = $9,
			pause_reason = $10,
			max_failures_before_pause = $11,
			next_run_at = $12,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id, name, cron_expression, enabled, source_proxy_ids, max_results,
			consecutive_failures, last_failure_reason, paused_at, pause_reason,
			max_failures_before_pause, last_run_at, next_run_at, created_at, updated_at
	`, plan.ID, plan.Name, plan.CronExpression, plan.Enabled, string(sourceProxyIDs), plan.MaxResults, plan.ConsecutiveFailures, plan.LastFailureReason, plan.PausedAt, plan.PauseReason, plan.MaxFailuresBeforePause, plan.NextRunAt)
	return scanProxyMaintenancePlan(row)
}

func (r *proxyMaintenancePlanRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM proxy_maintenance_plans WHERE id = $1`, id)
	return err
}

func (r *proxyMaintenancePlanRepository) UpdateAfterRun(ctx context.Context, update service.ProxyMaintenanceRunUpdate) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE proxy_maintenance_plans
		SET
			last_run_at = $2,
			next_run_at = $3,
			enabled = $4,
			consecutive_failures = $5,
			last_failure_reason = $6,
			paused_at = $7,
			pause_reason = $8,
			updated_at = NOW()
		WHERE id = $1
	`, update.ID, update.LastRunAt, update.NextRunAt, update.Enabled, update.ConsecutiveFailures, update.LastFailureReason, update.PausedAt, update.PauseReason)
	return err
}

type proxyMaintenanceResultRepository struct {
	db *sql.DB
}

func NewProxyMaintenanceResultRepository(db *sql.DB) service.ProxyMaintenanceResultRepository {
	return &proxyMaintenanceResultRepository{db: db}
}

func (r *proxyMaintenanceResultRepository) Create(ctx context.Context, result *service.ProxyMaintenanceResult) (*service.ProxyMaintenanceResult, error) {
	details, err := json.Marshal(result.Details)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO proxy_maintenance_results (
			plan_id, status, summary, moved_accounts, checked_proxies, healthy_proxies,
			failed_proxies, details, error_message, started_at, finished_at, created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,NOW())
		RETURNING
			id, plan_id, status, summary, moved_accounts, checked_proxies, healthy_proxies,
			failed_proxies, details, error_message, started_at, finished_at, created_at
	`, result.PlanID, result.Status, result.Summary, result.MovedAccounts, result.CheckedProxies, result.HealthyProxies, result.FailedProxies, string(details), result.ErrorMessage, result.StartedAt, result.FinishedAt)
	return scanProxyMaintenanceResult(row)
}

func (r *proxyMaintenanceResultRepository) ListByPlanID(ctx context.Context, planID int64, limit int) ([]*service.ProxyMaintenanceResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, plan_id, status, summary, moved_accounts, checked_proxies, healthy_proxies,
			failed_proxies, details, error_message, started_at, finished_at, created_at
		FROM proxy_maintenance_results
		WHERE plan_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, planID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanProxyMaintenanceResults(rows)
}

func (r *proxyMaintenanceResultRepository) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM proxy_maintenance_results
		WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY plan_id ORDER BY created_at DESC) AS rn
				FROM proxy_maintenance_results
				WHERE plan_id = $1
			) ranked
			WHERE rn > $2
		)
	`, planID, keepCount)
	return err
}

type proxyMaintenanceScannable interface {
	Scan(dest ...any) error
}

func scanProxyMaintenancePlan(row proxyMaintenanceScannable) (*service.ProxyMaintenancePlan, error) {
	plan := &service.ProxyMaintenancePlan{}
	var sourceProxyIDsRaw []byte
	if err := row.Scan(
		&plan.ID, &plan.Name, &plan.CronExpression, &plan.Enabled, &sourceProxyIDsRaw, &plan.MaxResults,
		&plan.ConsecutiveFailures, &plan.LastFailureReason, &plan.PausedAt, &plan.PauseReason,
		&plan.MaxFailuresBeforePause, &plan.LastRunAt, &plan.NextRunAt, &plan.CreatedAt, &plan.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(sourceProxyIDsRaw) > 0 {
		_ = json.Unmarshal(sourceProxyIDsRaw, &plan.SourceProxyIDs)
	}
	if plan.SourceProxyIDs == nil {
		plan.SourceProxyIDs = []int64{}
	}
	return plan, nil
}

func scanProxyMaintenancePlans(rows *sql.Rows) ([]*service.ProxyMaintenancePlan, error) {
	var plans []*service.ProxyMaintenancePlan
	for rows.Next() {
		plan, err := scanProxyMaintenancePlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func scanProxyMaintenanceResult(row proxyMaintenanceScannable) (*service.ProxyMaintenanceResult, error) {
	result := &service.ProxyMaintenanceResult{}
	var detailsRaw []byte
	if err := row.Scan(
		&result.ID, &result.PlanID, &result.Status, &result.Summary, &result.MovedAccounts,
		&result.CheckedProxies, &result.HealthyProxies, &result.FailedProxies, &detailsRaw,
		&result.ErrorMessage, &result.StartedAt, &result.FinishedAt, &result.CreatedAt,
	); err != nil {
		return nil, err
	}
	if len(detailsRaw) > 0 {
		_ = json.Unmarshal(detailsRaw, &result.Details)
	}
	if result.Details == nil {
		result.Details = map[string]any{}
	}
	if assignmentsRaw, ok := result.Details["assignments"]; ok {
		buf, _ := json.Marshal(assignmentsRaw)
		_ = json.Unmarshal(buf, &result.Assignments)
	}
	if failuresRaw, ok := result.Details["failures"]; ok {
		buf, _ := json.Marshal(failuresRaw)
		_ = json.Unmarshal(buf, &result.Failures)
	}
	return result, nil
}

func scanProxyMaintenanceResults(rows *sql.Rows) ([]*service.ProxyMaintenanceResult, error) {
	var results []*service.ProxyMaintenanceResult
	for rows.Next() {
		result, err := scanProxyMaintenanceResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}
