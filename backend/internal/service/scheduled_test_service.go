package service

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var scheduledTestCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

const scheduledTestDefaultMaxFailuresBeforePause = 3

func applyScheduledTestPlanDefaults(plan *ScheduledTestPlan) {
	if plan.MaxResults <= 0 {
		plan.MaxResults = 50
	}
	if plan.MaxFailuresBeforePause <= 0 {
		plan.MaxFailuresBeforePause = scheduledTestDefaultMaxFailuresBeforePause
	}
}

func resetScheduledTestFailureState(plan *ScheduledTestPlan) {
	plan.ConsecutiveFailures = 0
	plan.LastFailureReason = ""
	plan.PausedAt = nil
	plan.PauseReason = ""
}

func preserveScheduledTestRuntimeState(dst *ScheduledTestPlan, src *ScheduledTestPlan) {
	if dst == nil || src == nil {
		return
	}
	dst.ConsecutiveFailures = src.ConsecutiveFailures
	dst.LastFailureReason = src.LastFailureReason
	dst.PausedAt = src.PausedAt
	dst.PauseReason = src.PauseReason
}

// ScheduledTestService provides CRUD operations for scheduled test plans and results.
type ScheduledTestService struct {
	planRepo   ScheduledTestPlanRepository
	resultRepo ScheduledTestResultRepository
}

// NewScheduledTestService creates a new ScheduledTestService.
func NewScheduledTestService(
	planRepo ScheduledTestPlanRepository,
	resultRepo ScheduledTestResultRepository,
) *ScheduledTestService {
	return &ScheduledTestService{
		planRepo:   planRepo,
		resultRepo: resultRepo,
	}
}

// CreatePlan validates the cron expression, computes next_run_at, and persists the plan.
func (s *ScheduledTestService) CreatePlan(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun

	applyScheduledTestPlanDefaults(plan)
	resetScheduledTestFailureState(plan)

	return s.planRepo.Create(ctx, plan)
}

// GetPlan retrieves a plan by ID.
func (s *ScheduledTestService) GetPlan(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	return s.planRepo.GetByID(ctx, id)
}

// ListPlansByAccount returns all plans for a given account.
func (s *ScheduledTestService) ListPlansByAccount(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	return s.planRepo.ListByAccountID(ctx, accountID)
}

// UpdatePlan validates cron and updates the plan.
func (s *ScheduledTestService) UpdatePlan(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	current, err := s.planRepo.GetByID(ctx, plan.ID)
	if err != nil {
		return nil, err
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun

	applyScheduledTestPlanDefaults(plan)
	preserveScheduledTestRuntimeState(plan, current)

	// Only a real re-enable should clear accumulated failures and pause metadata.
	if plan.Enabled && !current.Enabled {
		resetScheduledTestFailureState(plan)
	} else if plan.PausedAt != nil {
		plan.NextRunAt = nil
	}

	return s.planRepo.Update(ctx, plan)
}

// DeletePlan removes a plan and its results (via CASCADE).
func (s *ScheduledTestService) DeletePlan(ctx context.Context, id int64) error {
	return s.planRepo.Delete(ctx, id)
}

// ListResults returns the most recent results for a plan.
func (s *ScheduledTestService) ListResults(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.resultRepo.ListByPlanID(ctx, planID, limit)
}

// SaveResult inserts a result and prunes old entries beyond maxResults.
func (s *ScheduledTestService) SaveResult(ctx context.Context, planID int64, maxResults int, result *ScheduledTestResult) error {
	result.PlanID = planID
	if _, err := s.resultRepo.Create(ctx, result); err != nil {
		return err
	}
	return s.resultRepo.PruneOldResults(ctx, planID, maxResults)
}

func computeNextRun(cronExpr string, from time.Time) (time.Time, error) {
	sched, err := scheduledTestCronParser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(from), nil
}
