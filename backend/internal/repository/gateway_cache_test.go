package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGatewayCacheGetSessionAccountIDUsesLocalCache(t *testing.T) {
	cache := NewGatewayCache(nil).(*gatewayCache)
	key := buildSessionKey(1, "session-a")
	cache.cacheSessionAccountID(key, 42, time.Second)

	accountID, err := cache.GetSessionAccountID(context.Background(), 1, "session-a")
	require.NoError(t, err)
	require.Equal(t, int64(42), accountID)
}

func TestNormalizeGatewayCacheTTL(t *testing.T) {
	require.Equal(t, gatewayCacheLocalMaxTTL, normalizeGatewayCacheTTL(0))
	require.Equal(t, gatewayCacheLocalMaxTTL, normalizeGatewayCacheTTL(5*time.Second))
	require.Equal(t, 300*time.Millisecond, normalizeGatewayCacheTTL(300*time.Millisecond))
}

func TestGatewayCacheRemoteRefreshInterval(t *testing.T) {
	require.Equal(t, gatewayCacheMaxRefreshGap, gatewayCacheRemoteRefreshInterval(0))
	require.Equal(t, gatewayCacheMinRefreshGap, gatewayCacheRemoteRefreshInterval(2*time.Second))
	require.Equal(t, 15*time.Second, gatewayCacheRemoteRefreshInterval(time.Minute))
	require.Equal(t, gatewayCacheMaxRefreshGap, gatewayCacheRemoteRefreshInterval(10*time.Minute))
}

func TestGatewayCacheRefreshLocalSessionTTLExtendsEntry(t *testing.T) {
	cache := NewGatewayCache(nil).(*gatewayCache)
	key := buildSessionKey(1, "session-c")
	cache.cacheSessionAccountID(key, 7, 50*time.Millisecond)

	time.Sleep(20 * time.Millisecond)
	cache.refreshLocalSessionTTL(key, time.Second)
	time.Sleep(60 * time.Millisecond)

	accountID, ok := cache.getLocalSessionAccountID(key)
	require.True(t, ok)
	require.Equal(t, int64(7), accountID)
}

func TestGatewayCacheShouldRefreshRemoteSessionTTL(t *testing.T) {
	cache := NewGatewayCache(nil).(*gatewayCache)
	key := buildSessionKey(1, "session-d")
	now := time.Now()
	cache.cacheSessionAccountIDAt(key, 11, time.Minute, now)

	require.False(t, cache.shouldRefreshRemoteSessionTTL(key, time.Minute, now.Add(5*time.Second)))
	require.True(t, cache.shouldRefreshRemoteSessionTTL(key, time.Minute, now.Add(16*time.Second)))
}

func TestGatewayCacheShouldWriteRemoteSessionAccount(t *testing.T) {
	cache := NewGatewayCache(nil).(*gatewayCache)
	key := buildSessionKey(1, "session-write")
	now := time.Now()
	cache.cacheSessionAccountIDAt(key, 11, time.Minute, now)

	require.False(t, cache.shouldWriteRemoteSessionAccount(key, 11, time.Minute, now.Add(5*time.Second)))
	require.True(t, cache.shouldWriteRemoteSessionAccount(key, 22, time.Minute, now.Add(5*time.Second)))
	require.True(t, cache.shouldWriteRemoteSessionAccount(key, 11, time.Minute, now.Add(16*time.Second)))
}

func TestGatewayCacheRefreshLocalSessionTTLPreservesRefreshScheduleUntilDue(t *testing.T) {
	cache := NewGatewayCache(nil).(*gatewayCache)
	key := buildSessionKey(1, "session-e")
	now := time.Now()
	cache.cacheSessionAccountIDAt(key, 5, time.Minute, now)

	entryBefore, ok := cache.getLocalSessionEntry(key)
	require.True(t, ok)

	cache.refreshLocalSessionTTLAt(key, time.Minute, now.Add(5*time.Second))

	entryAfter, ok := cache.getLocalSessionEntry(key)
	require.True(t, ok)
	require.Equal(t, entryBefore.NextRemoteRefreshUnixNano, entryAfter.NextRemoteRefreshUnixNano)
	require.Equal(t, entryBefore.AccountID, entryAfter.AccountID)
}
