package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
)

var proxyMaintenanceCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

const (
	proxyMaintenanceDefaultMaxFailuresBeforePause = 3
	proxyMaintenanceDefaultMaxResults             = 50
	proxyMaintenanceProbeConcurrency              = 8
)

type ProxyMaintenanceService struct {
	planRepo   ProxyMaintenancePlanRepository
	resultRepo ProxyMaintenanceResultRepository
	adminSvc   AdminService
	settingSvc *SettingService
}

func NewProxyMaintenanceService(
	planRepo ProxyMaintenancePlanRepository,
	resultRepo ProxyMaintenanceResultRepository,
	adminSvc AdminService,
	settingSvc *SettingService,
) *ProxyMaintenanceService {
	return &ProxyMaintenanceService{
		planRepo:   planRepo,
		resultRepo: resultRepo,
		adminSvc:   adminSvc,
		settingSvc: settingSvc,
	}
}

func applyProxyMaintenancePlanDefaults(plan *ProxyMaintenancePlan) {
	if plan == nil {
		return
	}
	if plan.MaxResults <= 0 {
		plan.MaxResults = proxyMaintenanceDefaultMaxResults
	}
	if plan.MaxFailuresBeforePause <= 0 {
		plan.MaxFailuresBeforePause = proxyMaintenanceDefaultMaxFailuresBeforePause
	}
	plan.SourceProxyIDs = normalizeProxyMaintenanceIDs(plan.SourceProxyIDs)
}

func resetProxyMaintenanceFailureState(plan *ProxyMaintenancePlan) {
	if plan == nil {
		return
	}
	plan.ConsecutiveFailures = 0
	plan.LastFailureReason = ""
	plan.PausedAt = nil
	plan.PauseReason = ""
}

func preserveProxyMaintenanceRuntimeState(dst, src *ProxyMaintenancePlan) {
	if dst == nil || src == nil {
		return
	}
	dst.ConsecutiveFailures = src.ConsecutiveFailures
	dst.LastFailureReason = src.LastFailureReason
	dst.PausedAt = src.PausedAt
	dst.PauseReason = src.PauseReason
}

func (s *ProxyMaintenanceService) CreatePlan(ctx context.Context, plan *ProxyMaintenancePlan) (*ProxyMaintenancePlan, error) {
	nextRun, err := computeProxyMaintenanceNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun
	applyProxyMaintenancePlanDefaults(plan)
	resetProxyMaintenanceFailureState(plan)
	return s.planRepo.Create(ctx, plan)
}

func (s *ProxyMaintenanceService) GetPlan(ctx context.Context, id int64) (*ProxyMaintenancePlan, error) {
	return s.planRepo.GetByID(ctx, id)
}

func (s *ProxyMaintenanceService) ListPlans(ctx context.Context) ([]*ProxyMaintenancePlan, error) {
	return s.planRepo.List(ctx)
}

func (s *ProxyMaintenanceService) UpdatePlan(ctx context.Context, plan *ProxyMaintenancePlan) (*ProxyMaintenancePlan, error) {
	current, err := s.planRepo.GetByID(ctx, plan.ID)
	if err != nil {
		return nil, err
	}
	nextRun, err := computeProxyMaintenanceNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun
	applyProxyMaintenancePlanDefaults(plan)
	preserveProxyMaintenanceRuntimeState(plan, current)
	if plan.Enabled && !current.Enabled {
		resetProxyMaintenanceFailureState(plan)
	} else if plan.PausedAt != nil {
		plan.NextRunAt = nil
	}
	return s.planRepo.Update(ctx, plan)
}

func (s *ProxyMaintenanceService) DeletePlan(ctx context.Context, id int64) error {
	return s.planRepo.Delete(ctx, id)
}

func (s *ProxyMaintenanceService) ListResults(ctx context.Context, planID int64, limit int) ([]*ProxyMaintenanceResult, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.resultRepo.ListByPlanID(ctx, planID, limit)
}

func (s *ProxyMaintenanceService) SaveResult(ctx context.Context, planID int64, maxResults int, result *ProxyMaintenanceResult) error {
	result.PlanID = planID
	if _, err := s.resultRepo.Create(ctx, result); err != nil {
		return err
	}
	return s.resultRepo.PruneOldResults(ctx, planID, maxResults)
}

func computeProxyMaintenanceNextRun(cronExpr string, from time.Time) (time.Time, error) {
	sched, err := proxyMaintenanceCronParser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(from), nil
}

func (s *ProxyMaintenanceService) RunPlan(ctx context.Context, plan *ProxyMaintenancePlan) (*ProxyMaintenanceResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan is required")
	}
	return s.run(ctx, plan.SourceProxyIDs)
}

func (s *ProxyMaintenanceService) RunNow(ctx context.Context, sourceProxyIDs []int64) (*ProxyMaintenanceResult, error) {
	return s.run(ctx, sourceProxyIDs)
}

func (s *ProxyMaintenanceService) run(ctx context.Context, sourceProxyIDs []int64) (*ProxyMaintenanceResult, error) {
	if s == nil || s.adminSvc == nil {
		return nil, fmt.Errorf("proxy maintenance service unavailable")
	}
	startedAt := time.Now().UTC()
	selectedIDs := normalizeProxyMaintenanceIDs(sourceProxyIDs)

	proxies, err := s.adminSvc.GetAllProxiesWithAccountCount(ctx)
	if err != nil {
		return nil, err
	}
	selected := selectProxyMaintenanceTargets(proxies, selectedIDs)
	if len(selected) == 0 {
		return &ProxyMaintenanceResult{
			Status:       "success",
			Summary:      "No proxies selected",
			StartedAt:    startedAt,
			FinishedAt:   time.Now().UTC(),
			CreatedAt:    time.Now().UTC(),
			Assignments:  []ProxyMaintenanceAssignment{},
			Failures:     []ProxyMaintenanceFailure{},
			Details:      map[string]any{},
			MovedAccounts: 0,
		}, nil
	}

	checked, err := s.inspectProxyHealth(ctx, selected)
	if err != nil {
		return nil, err
	}

	assignments, unassigned := planProxyMaintenanceAssignments(checked)
	movedAccounts := 0
	for _, assignment := range assignments {
		if assignment.AccountCount == 0 || assignment.TargetProxyID <= 0 {
			continue
		}
		input := &BulkUpdateAccountsInput{
			AccountIDs: assignment.AccountIDs,
			ProxyID:    int64Pointer(assignment.TargetProxyID),
		}
		if _, err := s.adminSvc.BulkUpdateAccounts(ctx, input); err != nil {
			return nil, err
		}
		movedAccounts += assignment.AccountCount
	}
	deletedProxyIDs := s.cleanupFailedUnusedProxies(ctx, checked)

	healthyCount := 0
	failedCount := 0
	for _, check := range checked {
		if check.Success {
			healthyCount++
		} else {
			failedCount++
		}
	}

	status := "success"
	summary := fmt.Sprintf("Checked %d proxies, healthy=%d, failed=%d, moved_accounts=%d", len(checked), healthyCount, failedCount, movedAccounts)
	errorMessage := ""
	if len(unassigned) > 0 {
		status = "partial"
		summary = fmt.Sprintf("%s, unassigned=%d", summary, len(unassigned))
		errorMessage = "Some failed proxies had no healthy destination available"
	}

	result := &ProxyMaintenanceResult{
		Status:         status,
		Summary:        summary,
		MovedAccounts:  movedAccounts,
		CheckedProxies: len(checked),
		HealthyProxies: healthyCount,
		FailedProxies:  failedCount,
		Assignments:    assignments,
		Failures:       unassigned,
		ErrorMessage:   errorMessage,
		Details: map[string]any{
			"selected_proxy_ids": selectedIDs,
			"deleted_proxy_ids":  deletedProxyIDs,
		},
		StartedAt:  startedAt,
		FinishedAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	}
	return result, nil
}

type proxyHealthInspection struct {
	Proxy            ProxyWithAccountCount
	Success          bool
	Message          string
	AffectedAccounts []ProxyAccountSummary
}

func (s *ProxyMaintenanceService) inspectProxyHealth(ctx context.Context, proxies []ProxyWithAccountCount) ([]proxyHealthInspection, error) {
	if len(proxies) == 0 {
		return []proxyHealthInspection{}, nil
	}
	results := make([]proxyHealthInspection, len(proxies))
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(proxyMaintenanceProbeConcurrency)
	for i := range proxies {
		i := i
		eg.Go(func() error {
			proxy := proxies[i]
			inspection := proxyHealthInspection{Proxy: proxy}
			testResult, err := s.adminSvc.TestProxy(egCtx, proxy.ID)
			if err == nil && testResult != nil {
				inspection.Success = testResult.Success
				inspection.Message = strings.TrimSpace(testResult.Message)
			} else if err != nil {
				inspection.Message = err.Error()
			}
			if !inspection.Success && proxy.AccountCount > 0 {
				accounts, accErr := s.adminSvc.GetProxyAccounts(egCtx, proxy.ID)
				if accErr != nil {
					if inspection.Message != "" {
						inspection.Message += " | "
					}
					inspection.Message += "failed to list accounts: " + accErr.Error()
				} else {
					inspection.AffectedAccounts = accounts
				}
			}
			results[i] = inspection
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func selectProxyMaintenanceTargets(proxies []ProxyWithAccountCount, selectedIDs []int64) []ProxyWithAccountCount {
	if len(selectedIDs) == 0 {
		return append([]ProxyWithAccountCount(nil), proxies...)
	}
	selected := make(map[int64]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = struct{}{}
	}
	out := make([]ProxyWithAccountCount, 0, len(selectedIDs))
	for _, proxy := range proxies {
		if _, ok := selected[proxy.ID]; ok {
			out = append(out, proxy)
		}
	}
	return out
}

func planProxyMaintenanceAssignments(checked []proxyHealthInspection) ([]ProxyMaintenanceAssignment, []ProxyMaintenanceFailure) {
	healthy := make([]ProxyWithAccountCount, 0)
	projectedUsage := make(map[int64]int)
	for _, check := range checked {
		if check.Success {
			healthy = append(healthy, check.Proxy)
			projectedUsage[check.Proxy.ID] = int(check.Proxy.AccountCount)
		}
	}
	assignments := make([]ProxyMaintenanceAssignment, 0)
	unassigned := make([]ProxyMaintenanceFailure, 0)
	for _, check := range checked {
		if check.Success || len(check.AffectedAccounts) == 0 {
			continue
		}
		target := pickProxyMaintenanceTarget(healthy, projectedUsage, check.Proxy.ID)
		if target == nil {
			unassigned = append(unassigned, ProxyMaintenanceFailure{
				ProxyID:       check.Proxy.ID,
				ProxyName:     check.Proxy.Name,
				Message:       check.Message,
				AffectedCount: len(check.AffectedAccounts),
			})
			continue
		}
		accountIDs := make([]int64, 0, len(check.AffectedAccounts))
		for _, account := range check.AffectedAccounts {
			if account.ID > 0 {
				accountIDs = append(accountIDs, account.ID)
			}
		}
		if len(accountIDs) == 0 {
			continue
		}
		projectedUsage[target.ID] += len(accountIDs)
		assignments = append(assignments, ProxyMaintenanceAssignment{
			SourceProxyID: check.Proxy.ID,
			TargetProxyID: target.ID,
			AccountIDs:    accountIDs,
			AccountCount:  len(accountIDs),
			SourceProxy:   check.Proxy.Name,
			TargetProxy:   target.Name,
		})
	}
	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].SourceProxyID != assignments[j].SourceProxyID {
			return assignments[i].SourceProxyID < assignments[j].SourceProxyID
		}
		return assignments[i].TargetProxyID < assignments[j].TargetProxyID
	})
	return assignments, unassigned
}

func pickProxyMaintenanceTarget(healthy []ProxyWithAccountCount, projectedUsage map[int64]int, excludeID int64) *ProxyWithAccountCount {
	if len(healthy) == 0 {
		return nil
	}
	candidates := make([]ProxyWithAccountCount, 0, len(healthy))
	for _, proxy := range healthy {
		if proxy.ID == excludeID {
			continue
		}
		candidates = append(candidates, proxy)
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := projectedUsage[candidates[i].ID]
		right := projectedUsage[candidates[j].ID]
		if left != right {
			return left < right
		}
		return candidates[i].ID < candidates[j].ID
	})
	return &candidates[0]
}

func int64Pointer(value int64) *int64 {
	return &value
}

func (s *ProxyMaintenanceService) cleanupFailedUnusedProxies(ctx context.Context, checked []proxyHealthInspection) []int64 {
	if s == nil || s.adminSvc == nil || s.settingSvc == nil || !s.settingSvc.IsAutoDeleteUselessProxiesEnabled(ctx) {
		return nil
	}
	deleted := make([]int64, 0)
	for _, check := range checked {
		if check.Success || check.Proxy.ID <= 0 {
			continue
		}
		deleteCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.adminSvc.DeleteProxy(deleteCtx, check.Proxy.ID); err == nil {
			deleted = append(deleted, check.Proxy.ID)
		}
		cancel()
	}
	sort.Slice(deleted, func(i, j int) bool { return deleted[i] < deleted[j] })
	return deleted
}

func normalizeProxyMaintenanceIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
