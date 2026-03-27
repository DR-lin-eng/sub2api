package service

import (
	"context"
	"time"
)

type TaskStateCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type AdminTaskStateRepository interface {
	UpsertState(ctx context.Context, taskKind string, taskID string, stateJSON string, status string, expiresAt *time.Time, finishedAt *time.Time) error
	GetState(ctx context.Context, taskKind string, taskID string) (string, error)
	DeleteExpired(ctx context.Context, now time.Time, limit int) (int64, error)
}
