package repository

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newSchedulerCacheForTest(t *testing.T) (*schedulerCache, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewSchedulerCache(client).(*schedulerCache), client
}

func TestSchedulerCache_GetSnapshot_UsesBucketHashWithoutAccountKeys(t *testing.T) {
	cache, client := newSchedulerCacheForTest(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{GroupID: 1, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	accounts := []service.Account{
		{ID: 1, Name: "acc-1", Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{1}},
		{ID: 2, Name: "acc-2", Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{1}},
	}

	require.NoError(t, cache.SetSnapshot(ctx, bucket, accounts))
	require.NoError(t, client.Del(ctx, schedulerAccountKey("1"), schedulerAccountKey("2")).Err())

	got, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, got, 2)
	require.Equal(t, int64(1), got[0].ID)
	require.Equal(t, int64(2), got[1].ID)
}

func TestSchedulerCache_SetSnapshot_LargeBucketPreservesOrder(t *testing.T) {
	cache, _ := newSchedulerCacheForTest(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{GroupID: 1, Platform: service.PlatformAnthropic, Mode: service.SchedulerModeMixed}
	accounts := make([]service.Account, 0, 1200)
	for i := 1; i <= 1200; i++ {
		accounts = append(accounts, service.Account{
			ID:          int64(i),
			Name:        "acc-" + strconv.Itoa(i),
			Platform:    service.PlatformAnthropic,
			Status:      service.StatusActive,
			Schedulable: true,
			GroupIDs:    []int64{1},
			LastUsedAt:  ptrTime(time.Unix(int64(i), 0)),
		})
	}

	require.NoError(t, cache.SetSnapshot(ctx, bucket, accounts))

	got, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, got, 1200)
	require.Equal(t, int64(1), got[0].ID)
	require.Equal(t, int64(600), got[599].ID)
	require.Equal(t, int64(1200), got[1199].ID)
}

func TestSchedulerCache_UpdateLastUsed_UpdatesActiveSnapshotHash(t *testing.T) {
	cache, _ := newSchedulerCacheForTest(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{GroupID: 1, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	accounts := []service.Account{
		{ID: 11, Name: "acc-11", Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{1}},
	}
	require.NoError(t, cache.SetSnapshot(ctx, bucket, accounts))

	usedAt := time.Unix(1712345678, 0)
	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{11: usedAt}))

	got, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, got, 1)
	require.NotNil(t, got[0].LastUsedAt)
	require.WithinDuration(t, usedAt, *got[0].LastUsedAt, time.Second)
}

func TestSchedulerCache_GetSnapshot_DoesNotReadLegacyZSetAnymore(t *testing.T) {
	cache, client := newSchedulerCacheForTest(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{GroupID: 9, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}

	require.NoError(t, client.Set(ctx, schedulerBucketKey(schedulerReadyPrefix, bucket), "1", 0).Err())
	require.NoError(t, client.Set(ctx, schedulerBucketKey(schedulerActivePrefix, bucket), "legacy", 0).Err())
	require.NoError(t, client.ZAdd(ctx, schedulerSnapshotLegacyKey(bucket, "legacy"),
		redis.Z{Score: 0, Member: "101"},
	).Err())
	require.NoError(t, client.Set(ctx, schedulerAccountKey("101"), `{"id":101,"name":"legacy","platform":"openai","status":"active","schedulable":true}`, 0).Err())

	got, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.False(t, hit)
	require.Nil(t, got)
}
