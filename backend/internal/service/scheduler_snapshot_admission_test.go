//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type schedulerAdmissionCacheStub struct {
	accountsByID map[int64]*Account
	snapshotsByBucket map[string][]*Account
	setSnapshotBuckets []SchedulerBucket
}

func (s *schedulerAdmissionCacheStub) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	if s.snapshotsByBucket == nil {
		return nil, false, nil
	}
	cached, ok := s.snapshotsByBucket[bucket.String()]
	if !ok {
		return nil, false, nil
	}
	out := make([]*Account, 0, len(cached))
	for _, account := range cached {
		if account == nil {
			continue
		}
		cloned := *account
		out = append(out, &cloned)
	}
	return out, true, nil
}

func (s *schedulerAdmissionCacheStub) SetSnapshot(ctx context.Context, bucket SchedulerBucket, accounts []Account) error {
	s.setSnapshotBuckets = append(s.setSnapshotBuckets, bucket)
	if s.snapshotsByBucket == nil {
		s.snapshotsByBucket = make(map[string][]*Account)
	}
	cloned := make([]*Account, 0, len(accounts))
	for i := range accounts {
		account := accounts[i]
		copied := account
		cloned = append(cloned, &copied)
	}
	s.snapshotsByBucket[bucket.String()] = cloned
	return nil
}

func (s *schedulerAdmissionCacheStub) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s.accountsByID == nil {
		return nil, nil
	}
	account := s.accountsByID[accountID]
	if account == nil {
		return nil, nil
	}
	cloned := *account
	return &cloned, nil
}

func (s *schedulerAdmissionCacheStub) SetAccount(ctx context.Context, account *Account) error {
	if s.accountsByID == nil {
		s.accountsByID = make(map[int64]*Account)
	}
	if account == nil {
		return nil
	}
	cloned := *account
	s.accountsByID[account.ID] = &cloned
	return nil
}

func (s *schedulerAdmissionCacheStub) DeleteAccount(ctx context.Context, accountID int64) error {
	delete(s.accountsByID, accountID)
	return nil
}

func (s *schedulerAdmissionCacheStub) UpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	return nil
}

func (s *schedulerAdmissionCacheStub) TryLockBucket(ctx context.Context, bucket SchedulerBucket, ttl time.Duration) (bool, error) {
	return true, nil
}

func (s *schedulerAdmissionCacheStub) ListBuckets(ctx context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}

func (s *schedulerAdmissionCacheStub) GetOutboxWatermark(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *schedulerAdmissionCacheStub) SetOutboxWatermark(ctx context.Context, id int64) error {
	return nil
}

type schedulerAdmissionTesterRecorder struct {
	ids []int64
}

func (r *schedulerAdmissionTesterRecorder) EnqueueSchedulerAdmissionTest(accountID int64) {
	r.ids = append(r.ids, accountID)
}

func TestSchedulerSnapshotService_ListSchedulableAccounts_CacheMissRebuildsHashSnapshot(t *testing.T) {
	cache := &schedulerAdmissionCacheStub{}
	repo := &mockAccountRepoForPlatform{
		listGroupPlatformFunc: func(ctx context.Context, groupID int64, platform string) ([]Account, error) {
			if groupID != 9 {
				t.Fatalf("unexpected groupID %d", groupID)
			}
			if platform != PlatformOpenAI {
				t.Fatalf("unexpected platform %s", platform)
			}
			return []Account{
				{ID: 901, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, GroupIDs: []int64{9}},
				{ID: 902, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, GroupIDs: []int64{9}},
			}, nil
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.DbFallbackEnabled = true
	groupID := int64(9)
	svc := NewSchedulerSnapshotService(cache, nil, repo, nil, cfg)

	accounts, useMixed, err := svc.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	require.NoError(t, err)
	require.False(t, useMixed)
	require.Len(t, accounts, 2)
	require.Equal(t, int64(901), accounts[0].ID)
	require.Equal(t, int64(902), accounts[1].ID)
	require.Len(t, cache.setSnapshotBuckets, 1)
	require.Equal(t, SchedulerBucket{GroupID: 9, Platform: PlatformOpenAI, Mode: SchedulerModeSingle}, cache.setSnapshotBuckets[0])

	cached, hit, err := cache.GetSnapshot(context.Background(), SchedulerBucket{GroupID: 9, Platform: PlatformOpenAI, Mode: SchedulerModeSingle})
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, cached, 2)
	require.Equal(t, int64(901), cached[0].ID)
	require.Equal(t, int64(902), cached[1].ID)
}

func TestSchedulerSnapshotService_MaybeEnqueueAdmissionProbe_ForNewSchedulableOAuth(t *testing.T) {
	tester := &schedulerAdmissionTesterRecorder{}
	svc := NewSchedulerSnapshotService(nil, nil, nil, nil, nil)
	svc.SetAdmissionTester(tester)

	account := &Account{
		ID:          301,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"refresh_token": "rt-301",
		},
	}

	svc.maybeEnqueueAdmissionProbe(nil, account)
	require.Equal(t, []int64{301}, tester.ids)
}

func TestSchedulerSnapshotService_MaybeEnqueueAdmissionProbe_DoesNotEnqueueForAlreadySchedulableAccount(t *testing.T) {
	accountID := int64(302)
	previous := &Account{
		ID:          accountID,
		Platform:    PlatformGemini,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{"refresh_token": "rt-old"},
	}
	current := &Account{
		ID:          accountID,
		Platform:    PlatformGemini,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{"refresh_token": "rt-new"},
	}
	tester := &schedulerAdmissionTesterRecorder{}
	svc := NewSchedulerSnapshotService(nil, nil, nil, nil, nil)
	svc.SetAdmissionTester(tester)

	svc.maybeEnqueueAdmissionProbe(previous, current)
	require.Empty(t, tester.ids)
}
