package service

import (
	"context"
	"time"
)

type ProxyMaintenancePlan struct {
	ID                     int64      `json:"id"`
	Name                   string     `json:"name"`
	CronExpression         string     `json:"cron_expression"`
	Enabled                bool       `json:"enabled"`
	SourceProxyIDs         []int64    `json:"source_proxy_ids"`
	MaxResults             int        `json:"max_results"`
	ConsecutiveFailures    int        `json:"consecutive_failures"`
	LastFailureReason      string     `json:"last_failure_reason"`
	PausedAt               *time.Time `json:"paused_at"`
	PauseReason            string     `json:"pause_reason"`
	MaxFailuresBeforePause int        `json:"max_failures_before_pause"`
	LastRunAt              *time.Time `json:"last_run_at"`
	NextRunAt              *time.Time `json:"next_run_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type ProxyMaintenanceAssignment struct {
	SourceProxyID  int64   `json:"source_proxy_id"`
	TargetProxyID  int64   `json:"target_proxy_id"`
	AccountIDs     []int64 `json:"account_ids"`
	AccountCount   int     `json:"account_count"`
	SourceProxy    string  `json:"source_proxy,omitempty"`
	TargetProxy    string  `json:"target_proxy,omitempty"`
}

type ProxyMaintenanceFailure struct {
	ProxyID       int64  `json:"proxy_id"`
	ProxyName     string `json:"proxy_name,omitempty"`
	Message       string `json:"message,omitempty"`
	AffectedCount int    `json:"affected_count"`
}

type ProxyMaintenanceResult struct {
	ID             int64                        `json:"id"`
	PlanID         int64                        `json:"plan_id,omitempty"`
	Status         string                       `json:"status"`
	Summary        string                       `json:"summary"`
	MovedAccounts  int                          `json:"moved_accounts"`
	CheckedProxies int                          `json:"checked_proxies"`
	HealthyProxies int                          `json:"healthy_proxies"`
	FailedProxies  int                          `json:"failed_proxies"`
	Details        map[string]any               `json:"details,omitempty"`
	Assignments    []ProxyMaintenanceAssignment `json:"assignments,omitempty"`
	Failures       []ProxyMaintenanceFailure    `json:"failures,omitempty"`
	ErrorMessage   string                       `json:"error_message,omitempty"`
	StartedAt      time.Time                    `json:"started_at"`
	FinishedAt     time.Time                    `json:"finished_at"`
	CreatedAt      time.Time                    `json:"created_at"`
}

type ProxyMaintenanceRunUpdate struct {
	ID                  int64
	LastRunAt           time.Time
	NextRunAt           *time.Time
	Enabled             bool
	ConsecutiveFailures int
	LastFailureReason   string
	PausedAt            *time.Time
	PauseReason         string
}

type ProxyMaintenancePlanRepository interface {
	Create(ctx context.Context, plan *ProxyMaintenancePlan) (*ProxyMaintenancePlan, error)
	GetByID(ctx context.Context, id int64) (*ProxyMaintenancePlan, error)
	List(ctx context.Context) ([]*ProxyMaintenancePlan, error)
	ListDue(ctx context.Context, now time.Time) ([]*ProxyMaintenancePlan, error)
	Update(ctx context.Context, plan *ProxyMaintenancePlan) (*ProxyMaintenancePlan, error)
	Delete(ctx context.Context, id int64) error
	UpdateAfterRun(ctx context.Context, update ProxyMaintenanceRunUpdate) error
}

type ProxyMaintenanceResultRepository interface {
	Create(ctx context.Context, result *ProxyMaintenanceResult) (*ProxyMaintenanceResult, error)
	ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ProxyMaintenanceResult, error)
	PruneOldResults(ctx context.Context, planID int64, keepCount int) error
}
