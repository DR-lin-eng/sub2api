//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type schedulerAdmissionCacheStub struct {
	accountsByID map[int64]*Account
}

func (s *schedulerAdmissionCacheStub) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	return nil, false, nil
}

func (s *schedulerAdmissionCacheStub) SetSnapshot(ctx context.Context, bucket SchedulerBucket, accounts []Account) error {
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
