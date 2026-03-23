package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type proxyFailoverAccountRepoStub struct {
	AccountRepository

	accounts        map[int64]*Account
	updatedAccounts []*Account
	tempUnschedule  []proxyFailoverTempUnscheduleCall
	clearedIDs      []int64
}

type proxyFailoverTempUnscheduleCall struct {
	accountID int64
	until     time.Time
	reason    string
}

func (s *proxyFailoverAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	account, ok := s.accounts[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return cloneAccountForProxyFailoverTest(account), nil
}

func (s *proxyFailoverAccountRepoStub) Update(_ context.Context, account *Account) error {
	cloned := cloneAccountForProxyFailoverTest(account)
	s.accounts[account.ID] = cloned
	s.updatedAccounts = append(s.updatedAccounts, cloneAccountForProxyFailoverTest(account))
	return nil
}

func (s *proxyFailoverAccountRepoStub) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	s.tempUnschedule = append(s.tempUnschedule, proxyFailoverTempUnscheduleCall{
		accountID: id,
		until:     until,
		reason:    reason,
	})
	if account, ok := s.accounts[id]; ok {
		account.TempUnschedulableUntil = ptrTimeForProxyFailoverTest(until)
		account.TempUnschedulableReason = reason
	}
	return nil
}

func (s *proxyFailoverAccountRepoStub) ClearTempUnschedulable(_ context.Context, id int64) error {
	s.clearedIDs = append(s.clearedIDs, id)
	if account, ok := s.accounts[id]; ok {
		account.TempUnschedulableUntil = nil
		account.TempUnschedulableReason = ""
	}
	return nil
}

type proxyFailoverProxyRepoStub struct {
	ProxyRepository

	proxies []ProxyWithAccountCount
}

func (s *proxyFailoverProxyRepoStub) ListActiveWithAccountCount(_ context.Context) ([]ProxyWithAccountCount, error) {
	out := make([]ProxyWithAccountCount, len(s.proxies))
	copy(out, s.proxies)
	return out, nil
}

type proxyFailoverLatencyCacheStub struct {
	ProxyLatencyCache

	latencies map[int64]*ProxyLatencyInfo
}

func (s *proxyFailoverLatencyCacheStub) GetProxyLatencies(_ context.Context, proxyIDs []int64) (map[int64]*ProxyLatencyInfo, error) {
	out := make(map[int64]*ProxyLatencyInfo, len(proxyIDs))
	for _, id := range proxyIDs {
		if info, ok := s.latencies[id]; ok {
			cloned := *info
			out[id] = &cloned
		}
	}
	return out, nil
}

type proxyFailoverSchedulerCacheStub struct {
	SchedulerCache

	lastAccount *Account
	setCalls    int
}

func (s *proxyFailoverSchedulerCacheStub) SetAccount(_ context.Context, account *Account) error {
	s.setCalls++
	s.lastAccount = cloneAccountForProxyFailoverTest(account)
	return nil
}

func TestGatewayServiceTempUnscheduleRetryableError_ReassignsFailedProxy(t *testing.T) {
	failedProxyID := int64(10)
	accountRepo := &proxyFailoverAccountRepoStub{
		accounts: map[int64]*Account{
			1: {
				ID:                      1,
				Name:                    "openai-1",
				Platform:                PlatformOpenAI,
				Type:                    AccountTypeOAuth,
				Status:                  StatusActive,
				Schedulable:             true,
				Concurrency:             1,
				Priority:                1,
				ProxyID:                 &failedProxyID,
				TempUnschedulableUntil:  ptrTimeForProxyFailoverTest(time.Now().Add(3 * time.Minute)),
				TempUnschedulableReason: "previous failure",
			},
		},
	}
	proxyRepo := &proxyFailoverProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 10, Name: "failed", Status: StatusActive}, AccountCount: 8},
			{Proxy: Proxy{ID: 20, Name: "healthy-low-load", Status: StatusActive}, AccountCount: 2},
			{Proxy: Proxy{ID: 30, Name: "failed-quality", Status: StatusActive}, AccountCount: 1},
		},
	}
	latencyCache := &proxyFailoverLatencyCacheStub{
		latencies: map[int64]*ProxyLatencyInfo{
			20: {Success: true, QualityStatus: "healthy", UpdatedAt: time.Now()},
			30: {Success: true, QualityStatus: "failed", UpdatedAt: time.Now()},
		},
	}
	schedulerCache := &proxyFailoverSchedulerCacheStub{}
	snapshot := NewSchedulerSnapshotService(schedulerCache, nil, nil, nil, nil)

	svc := NewGatewayService(
		accountRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		snapshot,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	svc.SetProxyFailoverDeps(proxyRepo, latencyCache)

	failoverErr := newProxyRequestFailoverError(accountRepo.accounts[1], "http://failed-proxy:8080", errors.New("dial tcp timeout"))
	svc.TempUnscheduleRetryableError(context.Background(), 1, failoverErr)

	require.Len(t, accountRepo.updatedAccounts, 1)
	require.Empty(t, accountRepo.tempUnschedule)
	require.Equal(t, []int64{1}, accountRepo.clearedIDs)

	updated := accountRepo.accounts[1]
	require.NotNil(t, updated.ProxyID)
	require.Equal(t, int64(20), *updated.ProxyID)
	require.Nil(t, updated.TempUnschedulableUntil)
	require.Empty(t, updated.TempUnschedulableReason)
	require.NotNil(t, updated.Proxy)
	require.Equal(t, int64(20), updated.Proxy.ID)

	require.Equal(t, 1, schedulerCache.setCalls)
	require.NotNil(t, schedulerCache.lastAccount)
	require.NotNil(t, schedulerCache.lastAccount.ProxyID)
	require.Equal(t, int64(20), *schedulerCache.lastAccount.ProxyID)
	require.Nil(t, schedulerCache.lastAccount.TempUnschedulableUntil)
}

func TestGatewayServiceTempUnscheduleRetryableError_FallsBackWhenNoHealthyProxy(t *testing.T) {
	failedProxyID := int64(10)
	accountRepo := &proxyFailoverAccountRepoStub{
		accounts: map[int64]*Account{
			1: {
				ID:          1,
				Name:        "openai-1",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				ProxyID:     &failedProxyID,
			},
		},
	}
	proxyRepo := &proxyFailoverProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 10, Name: "failed", Status: StatusActive}, AccountCount: 8},
			{Proxy: Proxy{ID: 20, Name: "cf-challenge", Status: StatusActive}, AccountCount: 1},
			{Proxy: Proxy{ID: 30, Name: "unreachable", Status: StatusActive}, AccountCount: 1},
		},
	}
	latencyCache := &proxyFailoverLatencyCacheStub{
		latencies: map[int64]*ProxyLatencyInfo{
			20: {Success: true, QualityStatus: "challenge", UpdatedAt: time.Now()},
			30: {Success: false, QualityStatus: "failed", UpdatedAt: time.Now()},
		},
	}

	svc := NewGatewayService(
		accountRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	svc.SetProxyFailoverDeps(proxyRepo, latencyCache)

	failoverErr := newProxyRequestFailoverError(accountRepo.accounts[1], "http://failed-proxy:8080", errors.New("context deadline exceeded"))
	before := time.Now()
	svc.TempUnscheduleRetryableError(context.Background(), 1, failoverErr)

	require.Empty(t, accountRepo.updatedAccounts)
	require.Empty(t, accountRepo.clearedIDs)
	require.Len(t, accountRepo.tempUnschedule, 1)
	require.Equal(t, int64(1), accountRepo.tempUnschedule[0].accountID)
	require.Equal(t, "upstream request failed via proxy/network (auto temp-unschedule 10m)", accountRepo.tempUnschedule[0].reason)
	require.WithinDuration(t, before.Add(10*time.Minute), accountRepo.tempUnschedule[0].until, 3*time.Second)
	require.NotNil(t, accountRepo.accounts[1].ProxyID)
	require.Equal(t, int64(10), *accountRepo.accounts[1].ProxyID)
}

func TestGatewayServiceTempUnscheduleRetryableError_IgnoresStaleFailedProxy(t *testing.T) {
	currentProxyID := int64(20)
	accountRepo := &proxyFailoverAccountRepoStub{
		accounts: map[int64]*Account{
			1: {
				ID:          1,
				Name:        "openai-1",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				ProxyID:     &currentProxyID,
			},
		},
	}
	schedulerCache := &proxyFailoverSchedulerCacheStub{}
	snapshot := NewSchedulerSnapshotService(schedulerCache, nil, nil, nil, nil)

	svc := NewGatewayService(
		accountRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		snapshot,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	svc.SetProxyFailoverDeps(&proxyFailoverProxyRepoStub{}, &proxyFailoverLatencyCacheStub{})

	failoverErr := &UpstreamFailoverError{
		StatusCode:           502,
		TempUnscheduleFor:    10 * time.Minute,
		TempUnscheduleReason: "upstream request failed via proxy/network (auto temp-unschedule 10m)",
		FailedProxyID:        10,
		FailedProxyURL:       "http://failed-proxy:8080",
	}
	svc.TempUnscheduleRetryableError(context.Background(), 1, failoverErr)

	require.Empty(t, accountRepo.updatedAccounts)
	require.Empty(t, accountRepo.tempUnschedule)
	require.Equal(t, []int64{1}, accountRepo.clearedIDs)
	require.NotNil(t, accountRepo.accounts[1].ProxyID)
	require.Equal(t, int64(20), *accountRepo.accounts[1].ProxyID)
	require.Equal(t, 1, schedulerCache.setCalls)
	require.NotNil(t, schedulerCache.lastAccount)
	require.NotNil(t, schedulerCache.lastAccount.ProxyID)
	require.Equal(t, int64(20), *schedulerCache.lastAccount.ProxyID)
}

func cloneAccountForProxyFailoverTest(account *Account) *Account {
	if account == nil {
		return nil
	}
	cloned := *account
	if account.ProxyID != nil {
		proxyID := *account.ProxyID
		cloned.ProxyID = &proxyID
	}
	if account.Proxy != nil {
		proxyCopy := *account.Proxy
		cloned.Proxy = &proxyCopy
	}
	if account.TempUnschedulableUntil != nil {
		t := *account.TempUnschedulableUntil
		cloned.TempUnschedulableUntil = &t
	}
	if account.LastUsedAt != nil {
		t := *account.LastUsedAt
		cloned.LastUsedAt = &t
	}
	if account.ExpiresAt != nil {
		t := *account.ExpiresAt
		cloned.ExpiresAt = &t
	}
	if account.RateLimitedAt != nil {
		t := *account.RateLimitedAt
		cloned.RateLimitedAt = &t
	}
	if account.RateLimitResetAt != nil {
		t := *account.RateLimitResetAt
		cloned.RateLimitResetAt = &t
	}
	if account.OverloadUntil != nil {
		t := *account.OverloadUntil
		cloned.OverloadUntil = &t
	}
	if account.SessionWindowStart != nil {
		t := *account.SessionWindowStart
		cloned.SessionWindowStart = &t
	}
	if account.SessionWindowEnd != nil {
		t := *account.SessionWindowEnd
		cloned.SessionWindowEnd = &t
	}
	if account.Credentials != nil {
		cloned.Credentials = make(map[string]any, len(account.Credentials))
		for k, v := range account.Credentials {
			cloned.Credentials[k] = v
		}
	}
	if account.Extra != nil {
		cloned.Extra = make(map[string]any, len(account.Extra))
		for k, v := range account.Extra {
			cloned.Extra[k] = v
		}
	}
	if account.GroupIDs != nil {
		cloned.GroupIDs = append([]int64(nil), account.GroupIDs...)
	}
	return &cloned
}

func ptrTimeForProxyFailoverTest(t time.Time) *time.Time {
	return &t
}
