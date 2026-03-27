package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type adminTaskStateRepository struct {
	db *sql.DB
}

func NewAdminTaskStateRepository(_ *dbent.Client, db *sql.DB) service.AdminTaskStateRepository {
	return &adminTaskStateRepository{db: db}
}

func (r *adminTaskStateRepository) UpsertState(ctx context.Context, taskKind string, taskID string, stateJSON string, status string, expiresAt *time.Time, finishedAt *time.Time) error {
	if r == nil || r.db == nil || taskKind == "" || taskID == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO admin_task_states (task_kind, task_id, state_json, status, expires_at, finished_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (task_kind, task_id)
		DO UPDATE SET
			state_json = EXCLUDED.state_json,
			status = EXCLUDED.status,
			expires_at = EXCLUDED.expires_at,
			finished_at = EXCLUDED.finished_at,
			updated_at = NOW()
	`, taskKind, taskID, stateJSON, status, expiresAt, finishedAt)
	return err
}

func (r *adminTaskStateRepository) GetState(ctx context.Context, taskKind string, taskID string) (string, error) {
	if r == nil || r.db == nil || taskKind == "" || taskID == "" {
		return "", nil
	}
	var state string
	err := r.db.QueryRowContext(ctx, `
		SELECT state_json
		FROM admin_task_states
		WHERE task_kind = $1 AND task_id = $2
	`, taskKind, taskID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return state, err
}

func (r *adminTaskStateRepository) DeleteExpired(ctx context.Context, now time.Time, limit int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 500
	}
	res, err := r.db.ExecContext(ctx, `
		WITH victims AS (
			SELECT task_kind, task_id
			FROM admin_task_states
			WHERE expires_at IS NOT NULL AND expires_at <= $1
			ORDER BY updated_at ASC
			LIMIT $2
		)
		DELETE FROM admin_task_states s
		USING victims v
		WHERE s.task_kind = v.task_kind AND s.task_id = v.task_id
	`, now, limit)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
