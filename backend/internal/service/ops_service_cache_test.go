//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type opsAccountRepoCacheStub struct {
	listWithFiltersFn func(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search, plan, oauthType, tierID string, groupID int64) ([]Account, *pagination.PaginationResult, error)
}

func (s *opsAccountRepoCacheStub) Create(ctx context.Context, account *Account) error {
	panic("unexpected Create call")
}
func (s *opsAccountRepoCacheStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	panic("unexpected GetByID call")
}
func (s *opsAccountRepoCacheStub) GetByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	panic("unexpected GetByIDs call")
}
func (s *opsAccountRepoCacheStub) ExistsByID(ctx context.Context, id int64) (bool, error) {
	panic("unexpected ExistsByID call")
}
func (s *opsAccountRepoCacheStub) GetByCRSAccountID(ctx context.Context, crsAccountID string) (*Account, error) {
	panic("unexpected GetByCRSAccountID call")
}
func (s *opsAccountRepoCacheStub) FindByExtraField(ctx context.Context, key string, value any) ([]Account, error) {
	panic("unexpected FindByExtraField call")
}
func (s *opsAccountRepoCacheStub) ListCRSAccountIDs(ctx context.Context) (map[string]int64, error) {
	panic("unexpected ListCRSAccountIDs call")
}
func (s *opsAccountRepoCacheStub) Update(ctx context.Context, account *Account) error {
	panic("unexpected Update call")
}
func (s *opsAccountRepoCacheStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}
func (s *opsAccountRepoCacheStub) List(ctx context.Context, params pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *opsAccountRepoCacheStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search, plan, oauthType, tierID string, groupID int64) ([]Account, *pagination.PaginationResult, error) {
	if s.listWithFiltersFn != nil {
		return s.listWithFiltersFn(ctx, params, platform, accountType, status, search, plan, oauthType, tierID, groupID)
	}
	panic("unexpected ListWithFilters call")
}
func (s *opsAccountRepoCacheStub) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	panic("unexpected ListByGroup call")
}
func (s *opsAccountRepoCacheStub) ListActive(ctx context.Context) ([]Account, error) {
	panic("unexpected ListActive call")
}
func (s *opsAccountRepoCacheStub) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListByPlatform call")
}
func (s *opsAccountRepoCacheStub) UpdateLastUsed(ctx context.Context, id int64) error {
	panic("unexpected UpdateLastUsed call")
}
func (s *opsAccountRepoCacheStub) BatchUpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	panic("unexpected BatchUpdateLastUsed call")
}
func (s *opsAccountRepoCacheStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	panic("unexpected SetError call")
}
func (s *opsAccountRepoCacheStub) ClearError(ctx context.Context, id int64) error {
	panic("unexpected ClearError call")
}
func (s *opsAccountRepoCacheStub) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	panic("unexpected SetSchedulable call")
}
func (s *opsAccountRepoCacheStub) AutoPauseExpiredAccounts(ctx context.Context, now time.Time) (int64, error) {
	panic("unexpected AutoPauseExpiredAccounts call")
}
func (s *opsAccountRepoCacheStub) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	panic("unexpected BindGroups call")
}
func (s *opsAccountRepoCacheStub) ListSchedulable(ctx context.Context) ([]Account, error) {
	panic("unexpected ListSchedulable call")
}
func (s *opsAccountRepoCacheStub) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupID call")
}
func (s *opsAccountRepoCacheStub) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableByPlatform call")
}
func (s *opsAccountRepoCacheStub) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupIDAndPlatform call")
}
func (s *opsAccountRepoCacheStub) ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableByPlatforms call")
}
func (s *opsAccountRepoCacheStub) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupIDAndPlatforms call")
}
func (s *opsAccountRepoCacheStub) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableUngroupedByPlatform call")
}
func (s *opsAccountRepoCacheStub) ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableUngroupedByPlatforms call")
}
func (s *opsAccountRepoCacheStub) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	panic("unexpected SetRateLimited call")
}
func (s *opsAccountRepoCacheStub) SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time) error {
	panic("unexpected SetModelRateLimit call")
}
func (s *opsAccountRepoCacheStub) SetOverloaded(ctx context.Context, id int64, until time.Time) error {
	panic("unexpected SetOverloaded call")
}
func (s *opsAccountRepoCacheStub) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	panic("unexpected SetTempUnschedulable call")
}
func (s *opsAccountRepoCacheStub) ClearTempUnschedulable(ctx context.Context, id int64) error {
	panic("unexpected ClearTempUnschedulable call")
}
func (s *opsAccountRepoCacheStub) ClearRateLimit(ctx context.Context, id int64) error {
	panic("unexpected ClearRateLimit call")
}
func (s *opsAccountRepoCacheStub) ClearAntigravityQuotaScopes(ctx context.Context, id int64) error {
	panic("unexpected ClearAntigravityQuotaScopes call")
}
func (s *opsAccountRepoCacheStub) ClearModelRateLimits(ctx context.Context, id int64) error {
	panic("unexpected ClearModelRateLimits call")
}
func (s *opsAccountRepoCacheStub) UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error {
	panic("unexpected UpdateSessionWindow call")
}
func (s *opsAccountRepoCacheStub) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	panic("unexpected UpdateExtra call")
}
func (s *opsAccountRepoCacheStub) BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	panic("unexpected BulkUpdate call")
}
func (s *opsAccountRepoCacheStub) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) error {
	panic("unexpected IncrementQuotaUsed call")
}
func (s *opsAccountRepoCacheStub) ResetQuotaUsed(ctx context.Context, id int64) error {
	panic("unexpected ResetQuotaUsed call")
}

func TestListAllAccountsForOpsCached_ReturnsEmptySnapshotOnColdLoadFailure(t *testing.T) {
	repo := &opsAccountRepoCacheStub{
		listWithFiltersFn: func(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search, plan, oauthType, tierID string, groupID int64) ([]Account, *pagination.PaginationResult, error) {
			return nil, nil, errors.New("db unavailable")
		},
	}
	svc := &OpsService{accountRepo: repo}

	accounts, err := svc.listAllAccountsForOpsCached(context.Background(), PlatformOpenAI)
	require.NoError(t, err)
	require.Empty(t, accounts)
}

func TestListAllAccountsForOpsCached_UsesStaleSnapshotOnReloadFailure(t *testing.T) {
	now := time.Now().Add(-time.Second)
	svc := &OpsService{
		accountRepo: &opsAccountRepoCacheStub{
			listWithFiltersFn: func(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search, plan, oauthType, tierID string, groupID int64) ([]Account, *pagination.PaginationResult, error) {
				return nil, nil, errors.New("db unavailable")
			},
		},
	}
	svc.accountListCache.Store(PlatformOpenAI, &opsCachedAccounts{
		Accounts:  []Account{{ID: 1, Platform: PlatformOpenAI}},
		ExpiresAt: now.UnixNano(),
	})

	accounts, err := svc.listAllAccountsForOpsCached(context.Background(), PlatformOpenAI)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, int64(1), accounts[0].ID)
}
