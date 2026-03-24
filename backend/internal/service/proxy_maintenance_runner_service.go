package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

const proxyMaintenanceDefaultMaxWorkers = 2

type ProxyMaintenanceRunnerService struct {
	planRepo *ProxyMaintenanceService
	cfg      *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

func NewProxyMaintenanceRunnerService(planSvc *ProxyMaintenanceService, cfg *config.Config) *ProxyMaintenanceRunnerService {
	return &ProxyMaintenanceRunnerService{
		planRepo: planSvc,
		cfg:      cfg,
	}
}

func (s *ProxyMaintenanceRunnerService) Start() {
	if s == nil || s.planRepo == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}
		c := cron.New(cron.WithParser(proxyMaintenanceCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc("* * * * *", func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] not started (invalid schedule): %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] started (tick=every minute)")
	})
}

func (s *ProxyMaintenanceRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] cron stop timed out")
			}
		}
	})
}

func (s *ProxyMaintenanceRunnerService) runScheduled() {
	time.Sleep(10 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	plans, err := s.planRepo.planRepo.ListDue(ctx, time.Now())
	if err != nil {
		logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] ListDue error: %v", err)
		return
	}
	if len(plans) == 0 {
		return
	}
	logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] found %d due plans", len(plans))

	sem := make(chan struct{}, proxyMaintenanceDefaultMaxWorkers)
	var wg sync.WaitGroup
	for _, plan := range plans {
		sem <- struct{}{}
		wg.Add(1)
		go func(plan *ProxyMaintenancePlan) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOnePlan(ctx, plan)
		}(plan)
	}
	wg.Wait()
}

func (s *ProxyMaintenanceRunnerService) runOnePlan(ctx context.Context, plan *ProxyMaintenancePlan) {
	if plan == nil {
		return
	}
	startedAt := time.Now().UTC()
	result, err := s.planRepo.RunPlan(ctx, plan)
	finishedAt := time.Now().UTC()
	if err != nil {
		logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] plan=%d run error: %v", plan.ID, err)
		result = &ProxyMaintenanceResult{
			Status:       "failed",
			Summary:      "proxy maintenance failed",
			ErrorMessage: err.Error(),
			StartedAt:    startedAt,
			FinishedAt:   finishedAt,
			CreatedAt:    finishedAt,
		}
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = startedAt
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = finishedAt
	}
	result.CreatedAt = finishedAt

	if err := s.planRepo.SaveResult(ctx, plan.ID, plan.MaxResults, result); err != nil {
		logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] plan=%d save result error: %v", plan.ID, err)
	}

	update, err := buildProxyMaintenanceRunUpdate(plan, result, finishedAt)
	if err != nil {
		logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] plan=%d build update error: %v", plan.ID, err)
		return
	}
	if err := s.planRepo.planRepo.UpdateAfterRun(ctx, update); err != nil {
		logger.LegacyPrintf("service.proxy_maintenance_runner", "[ProxyMaintenanceRunner] plan=%d update after run error: %v", plan.ID, err)
	}
}

func buildProxyMaintenanceRunUpdate(plan *ProxyMaintenancePlan, result *ProxyMaintenanceResult, finishedAt time.Time) (ProxyMaintenanceRunUpdate, error) {
	update := ProxyMaintenanceRunUpdate{
		ID:        plan.ID,
		LastRunAt: finishedAt,
		Enabled:   plan.Enabled,
	}
	if result == nil {
		result = &ProxyMaintenanceResult{Status: "failed", ErrorMessage: "proxy maintenance returned no result"}
	}
	if result.Status == "success" {
		nextRun, err := computeProxyMaintenanceNextRun(plan.CronExpression, finishedAt)
		if err != nil {
			return ProxyMaintenanceRunUpdate{}, err
		}
		update.NextRunAt = &nextRun
		update.ConsecutiveFailures = 0
		update.LastFailureReason = ""
		update.PausedAt = nil
		update.PauseReason = ""
		return update, nil
	}
	update.ConsecutiveFailures = plan.ConsecutiveFailures + 1
	update.LastFailureReason = stringsTrimSpaceOr(result.ErrorMessage, result.Summary, "proxy maintenance failed")
	if update.ConsecutiveFailures >= plan.MaxFailuresBeforePause && plan.MaxFailuresBeforePause > 0 {
		update.Enabled = false
		update.NextRunAt = nil
		pausedAt := finishedAt
		update.PausedAt = &pausedAt
		update.PauseReason = fmt.Sprintf("paused after %d consecutive proxy maintenance failures", update.ConsecutiveFailures)
	} else {
		nextRun, err := computeProxyMaintenanceNextRun(plan.CronExpression, finishedAt)
		if err != nil {
			return ProxyMaintenanceRunUpdate{}, err
		}
		update.NextRunAt = &nextRun
	}
	return update, nil
}

func stringsTrimSpaceOr(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
