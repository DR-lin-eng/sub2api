package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

const (
	scheduledTestDefaultMaxWorkers = 3
	scheduledTestPerPlanTimeout    = 45 * time.Second
)

// ScheduledTestRunnerService periodically scans due test plans and executes them.
type ScheduledTestRunnerService struct {
	planRepo       ScheduledTestPlanRepository
	scheduledSvc   *ScheduledTestService
	accountTestSvc *AccountTestService
	rateLimitSvc   *RateLimitService
	cfg            *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewScheduledTestRunnerService creates a new runner.
func NewScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc *AccountTestService,
	rateLimitSvc *RateLimitService,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	return &ScheduledTestRunnerService{
		planRepo:       planRepo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: accountTestSvc,
		rateLimitSvc:   rateLimitSvc,
		cfg:            cfg,
	}
}

// Start begins the cron ticker (every minute).
func (s *ScheduledTestRunnerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(scheduledTestCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc("* * * * *", func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] not started (invalid schedule): %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] started (tick=every minute)")
	})
}

// Stop gracefully shuts down the cron scheduler.
func (s *ScheduledTestRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] cron stop timed out")
			}
		}
	})
}

func (s *ScheduledTestRunnerService) runScheduled() {
	// Delay 10s so execution lands at ~:10 of each minute instead of :00.
	time.Sleep(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()
	plans, err := s.planRepo.ListDue(ctx, now)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] ListDue error: %v", err)
		return
	}
	if len(plans) == 0 {
		return
	}

	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] found %d due plans", len(plans))

	sem := make(chan struct{}, scheduledTestDefaultMaxWorkers)
	var wg sync.WaitGroup

	for _, plan := range plans {
		sem <- struct{}{}
		wg.Add(1)
		go func(p *ScheduledTestPlan) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOnePlan(ctx, p)
		}(plan)
	}

	wg.Wait()
}

func (s *ScheduledTestRunnerService) runOnePlan(ctx context.Context, plan *ScheduledTestPlan) {
	if plan == nil {
		return
	}
	planCtx := ctx
	var cancel context.CancelFunc
	if scheduledTestPerPlanTimeout > 0 {
		planCtx, cancel = context.WithTimeout(ctx, scheduledTestPerPlanTimeout)
		defer cancel()
	}

	runFinishedAt := time.Now()
	result, err := s.accountTestSvc.RunTestBackground(planCtx, plan.AccountID, plan.ModelID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d RunTestBackground error: %v", plan.ID, err)
		result = &ScheduledTestResult{
			Status:       "failed",
			ErrorMessage: err.Error(),
			StartedAt:    runFinishedAt,
			FinishedAt:   runFinishedAt,
		}
	}
	runFinishedAt = time.Now()

	if err := s.scheduledSvc.SaveResult(ctx, plan.ID, plan.MaxResults, result); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d SaveResult error: %v", plan.ID, err)
	}

	s.applyFailureSideEffects(ctx, plan, result)

	// Auto-recover account if test succeeded and auto_recover is enabled.
	if result.Status == "success" && plan.AutoRecover {
		s.tryRecoverAccount(ctx, plan.AccountID, plan.ID)
	}

	update, err := buildScheduledTestRunUpdate(plan, result, runFinishedAt)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d build run update error: %v", plan.ID, err)
		return
	}
	if err := s.planRepo.UpdateAfterRun(ctx, update); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d UpdateAfterRun error: %v", plan.ID, err)
	}
}

func (s *ScheduledTestRunnerService) applyFailureSideEffects(ctx context.Context, plan *ScheduledTestPlan, result *ScheduledTestResult) {
	if s == nil || plan == nil || result == nil || result.Status == "success" {
		return
	}
	if s.rateLimitSvc == nil || s.accountTestSvc == nil || s.accountTestSvc.accountRepo == nil {
		return
	}

	statusCode, body, ok := extractScheduledTestHTTPStatusAndBody(result.ErrorMessage)
	if !ok {
		return
	}
	if statusCode != http.StatusUnauthorized {
		return
	}

	account, err := s.accountTestSvc.accountRepo.GetByID(ctx, plan.AccountID)
	if err != nil || account == nil {
		return
	}
	shouldDisable := s.rateLimitSvc.HandleUpstreamError(ctx, account, statusCode, http.Header{}, body)
	logger.LegacyPrintf(
		"service.scheduled_test_runner",
		"[ScheduledTestRunner] plan=%d applied 401 side effects: account=%d should_disable=%v",
		plan.ID,
		plan.AccountID,
		shouldDisable,
	)
}

func shouldPauseScheduledTestImmediately(reason string) bool {
	reason = strings.ToLower(strings.TrimSpace(reason))
	if reason == "" {
		return false
	}
	return strings.Contains(reason, "account not found") ||
		strings.Contains(reason, "token_invalidated") ||
		strings.Contains(reason, "authentication token has been invalidated")
}

func extractScheduledTestHTTPStatusAndBody(reason string) (int, []byte, bool) {
	const prefix = "api returned "
	trimmed := strings.TrimSpace(reason)
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, prefix) {
		return 0, nil, false
	}

	rest := strings.TrimSpace(trimmed[len(prefix):])
	colon := strings.Index(rest, ":")
	if colon <= 0 {
		return 0, nil, false
	}

	statusCode, err := strconv.Atoi(strings.TrimSpace(rest[:colon]))
	if err != nil || statusCode <= 0 {
		return 0, nil, false
	}

	body := strings.TrimSpace(rest[colon+1:])
	return statusCode, []byte(body), true
}

func buildScheduledTestRunUpdate(plan *ScheduledTestPlan, result *ScheduledTestResult, finishedAt time.Time) (ScheduledTestRunUpdate, error) {
	update := ScheduledTestRunUpdate{
		ID:        plan.ID,
		LastRunAt: finishedAt,
		Enabled:   plan.Enabled,
	}
	if result == nil {
		result = &ScheduledTestResult{
			Status:       "failed",
			ErrorMessage: "scheduled test returned no result",
		}
	}

	if result.Status == "success" {
		nextRun, err := computeNextRun(plan.CronExpression, finishedAt)
		if err != nil {
			return ScheduledTestRunUpdate{}, err
		}
		update.NextRunAt = &nextRun
		update.ConsecutiveFailures = 0
		update.LastFailureReason = ""
		update.PausedAt = nil
		update.PauseReason = ""
		return update, nil
	}

	update.ConsecutiveFailures = plan.ConsecutiveFailures + 1
	update.LastFailureReason = strings.TrimSpace(result.ErrorMessage)
	if update.LastFailureReason == "" {
		update.LastFailureReason = strings.TrimSpace(result.ResponseText)
	}
	if update.LastFailureReason == "" {
		update.LastFailureReason = "scheduled test failed"
	}
	update.PausedAt = nil
	update.PauseReason = ""

	if shouldPauseScheduledTestImmediately(update.LastFailureReason) {
		update.Enabled = false
		pausedAt := finishedAt
		update.PausedAt = &pausedAt
		update.PauseReason = "paused after unrecoverable scheduled test failure"
		update.NextRunAt = nil
		return update, nil
	}

	if plan.MaxFailuresBeforePause > 0 && update.ConsecutiveFailures >= plan.MaxFailuresBeforePause {
		update.Enabled = false
		pausedAt := finishedAt
		update.PausedAt = &pausedAt
		update.PauseReason = fmt.Sprintf("paused after %d consecutive scheduled test failures", update.ConsecutiveFailures)
		return update, nil
	}

	nextRun, err := computeNextRun(plan.CronExpression, finishedAt)
	if err != nil {
		return ScheduledTestRunUpdate{}, err
	}
	update.NextRunAt = &nextRun
	return update, nil
}

// tryRecoverAccount attempts to recover an account from recoverable runtime state.
func (s *ScheduledTestRunnerService) tryRecoverAccount(ctx context.Context, accountID int64, planID int64) {
	if s.rateLimitSvc == nil {
		return
	}

	recovery, err := s.rateLimitSvc.RecoverAccountAfterSuccessfulTest(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover failed: %v", planID, err)
		return
	}
	if recovery == nil {
		return
	}

	if recovery.ClearedError {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d recovered from error status", planID, accountID)
	}
	if recovery.ClearedRateLimit {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d cleared rate-limit/runtime state", planID, accountID)
	}
}
