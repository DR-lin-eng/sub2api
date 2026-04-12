package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

type adminUsageRepoStub struct {
	UsageLogRepository
	statsByAccount map[int64]*usagestats.UsageStats
	statsByUser    map[int64]*usagestats.UsageStats
}

func (s *adminUsageRepoStub) GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	if filters.AccountID > 0 {
		if stats := s.statsByAccount[filters.AccountID]; stats != nil {
			return stats, nil
		}
		return &usagestats.UsageStats{}, nil
	}
	if filters.UserID > 0 {
		if stats := s.statsByUser[filters.UserID]; stats != nil {
			return stats, nil
		}
		return &usagestats.UsageStats{}, nil
	}
	return &usagestats.UsageStats{}, nil
}

type adminProxyRepoStub struct {
	ProxyRepository
	accountSummaries []ProxyAccountSummary
}

func (s *adminProxyRepoStub) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]ProxyAccountSummary, error) {
	return append([]ProxyAccountSummary(nil), s.accountSummaries...), nil
}

func (s *adminProxyRepoStub) Create(ctx context.Context, proxy *Proxy) error {
	panic("unexpected Create call")
}
func (s *adminProxyRepoStub) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	panic("unexpected GetByID call")
}
func (s *adminProxyRepoStub) ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error) {
	panic("unexpected ListByIDs call")
}
func (s *adminProxyRepoStub) Update(ctx context.Context, proxy *Proxy) error {
	panic("unexpected Update call")
}
func (s *adminProxyRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}
func (s *adminProxyRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *adminProxyRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (s *adminProxyRepoStub) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFiltersAndAccountCount call")
}
func (s *adminProxyRepoStub) ListActive(ctx context.Context) ([]Proxy, error) {
	panic("unexpected ListActive call")
}
func (s *adminProxyRepoStub) ListActiveWithAccountCount(ctx context.Context) ([]ProxyWithAccountCount, error) {
	panic("unexpected ListActiveWithAccountCount call")
}
func (s *adminProxyRepoStub) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	panic("unexpected ExistsByHostPortAuth call")
}
func (s *adminProxyRepoStub) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	panic("unexpected CountAccountsByProxyID call")
}

func TestAdminService_GetUserUsageStats_UsesRealUsageRepo(t *testing.T) {
	repo := &adminUsageRepoStub{
		statsByUser: map[int64]*usagestats.UsageStats{
			7: {
				TotalRequests:     12,
				TotalCost:         3.5,
				TotalTokens:       456,
				AverageDurationMs: 123,
			},
		},
	}
	svc := NewAdminService(nil, nil, nil, nil, nil, nil, nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	got, err := svc.GetUserUsageStats(context.Background(), 7, "month")
	require.NoError(t, err)
	payload, ok := got.(map[string]any)
	require.True(t, ok)
	require.Equal(t, int64(12), payload["total_requests"])
	require.Equal(t, 3.5, payload["total_cost"])
	require.Equal(t, int64(456), payload["total_tokens"])
	require.Equal(t, 123.0, payload["avg_duration_ms"])
}

func TestAdminService_GetProxyUsageStats_AggregatesAccountUsage(t *testing.T) {
	proxyRepo := &adminProxyRepoStub{}
	usageRepo := &adminUsageRepoStub{
		statsByAccount: map[int64]*usagestats.UsageStats{
			1: {TotalRequests: 10, TotalCost: 1.5, TotalTokens: 100, AverageDurationMs: 100},
			2: {TotalRequests: 30, TotalCost: 2.5, TotalTokens: 300, AverageDurationMs: 300},
		},
	}
	svc := &adminServiceImpl{
		proxyRepo:    proxyRepo,
		usageLogRepo: usageRepo,
	}
	proxyRepo.accountSummaries = []ProxyAccountSummary{
		{ID: 1, Name: "a1"},
		{ID: 2, Name: "a2"},
	}

	stats, err := svc.GetProxyUsageStats(context.Background(), 4)
	require.NoError(t, err)
	require.NotNil(t, stats)
	require.Equal(t, int64(40), stats.TotalRequests)
	require.Equal(t, 4.0, stats.TotalCost)
	require.Equal(t, int64(400), stats.TotalTokens)
	require.Equal(t, 250.0, stats.AverageDurationMs)
}
