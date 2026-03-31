package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	gocache "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
)

const (
	stickySessionPrefix         = "sticky_session:"
	gatewayCacheLocalMaxTTL     = time.Second
	gatewayCacheCleanupInterval = time.Minute
	gatewayCacheMinRefreshGap   = time.Second
	gatewayCacheMaxRefreshGap   = 30 * time.Second
)

type gatewayCacheEntry struct {
	AccountID                 int64
	NextRemoteRefreshUnixNano int64
}

type gatewayCache struct {
	rdb        *redis.Client
	localCache *gocache.Cache
}

func NewGatewayCache(rdb *redis.Client) service.GatewayCache {
	return &gatewayCache{
		rdb:        rdb,
		localCache: gocache.New(gatewayCacheLocalMaxTTL, gatewayCacheCleanupInterval),
	}
}

// buildSessionKey 构建 session key，包含 groupID 实现分组隔离
// 格式: sticky_session:{groupID}:{sessionHash}
func buildSessionKey(groupID int64, sessionHash string) string {
	return fmt.Sprintf("%s%d:%s", stickySessionPrefix, groupID, sessionHash)
}

func normalizeGatewayCacheTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 || ttl > gatewayCacheLocalMaxTTL {
		return gatewayCacheLocalMaxTTL
	}
	return ttl
}

func gatewayCacheRemoteRefreshInterval(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return gatewayCacheMaxRefreshGap
	}
	interval := ttl / 4
	if interval < gatewayCacheMinRefreshGap {
		return gatewayCacheMinRefreshGap
	}
	if interval > gatewayCacheMaxRefreshGap {
		return gatewayCacheMaxRefreshGap
	}
	return interval
}

func gatewayCacheNextRemoteRefreshUnixNano(now time.Time, ttl time.Duration) int64 {
	return now.Add(gatewayCacheRemoteRefreshInterval(ttl)).UnixNano()
}

func (c *gatewayCache) getLocalSessionEntry(key string) (gatewayCacheEntry, bool) {
	if c == nil || c.localCache == nil || key == "" {
		return gatewayCacheEntry{}, false
	}
	value, ok := c.localCache.Get(key)
	if !ok {
		return gatewayCacheEntry{}, false
	}
	entry, ok := value.(gatewayCacheEntry)
	if !ok || entry.AccountID <= 0 {
		c.localCache.Delete(key)
		return gatewayCacheEntry{}, false
	}
	return entry, true
}

func (c *gatewayCache) getLocalSessionAccountID(key string) (int64, bool) {
	entry, ok := c.getLocalSessionEntry(key)
	if !ok {
		return 0, false
	}
	return entry.AccountID, true
}

func (c *gatewayCache) cacheSessionAccountID(key string, accountID int64, ttl time.Duration) {
	c.cacheSessionAccountIDAt(key, accountID, ttl, time.Now())
}

func (c *gatewayCache) cacheSessionAccountIDAt(key string, accountID int64, ttl time.Duration, now time.Time) {
	if c == nil || c.localCache == nil || key == "" || accountID <= 0 {
		return
	}
	c.localCache.Set(key, gatewayCacheEntry{
		AccountID:                 accountID,
		NextRemoteRefreshUnixNano: gatewayCacheNextRemoteRefreshUnixNano(now, ttl),
	}, normalizeGatewayCacheTTL(ttl))
}

func (c *gatewayCache) refreshLocalSessionTTL(key string, ttl time.Duration) {
	c.refreshLocalSessionTTLAt(key, ttl, time.Now())
}

func (c *gatewayCache) refreshLocalSessionTTLAt(key string, ttl time.Duration, now time.Time) {
	if c == nil || c.localCache == nil || key == "" {
		return
	}
	entry, ok := c.getLocalSessionEntry(key)
	if !ok {
		return
	}
	if entry.NextRemoteRefreshUnixNano <= 0 {
		entry.NextRemoteRefreshUnixNano = gatewayCacheNextRemoteRefreshUnixNano(now, ttl)
	}
	c.localCache.Set(key, entry, normalizeGatewayCacheTTL(ttl))
}

func (c *gatewayCache) shouldRefreshRemoteSessionTTL(key string, ttl time.Duration, now time.Time) bool {
	entry, ok := c.getLocalSessionEntry(key)
	if !ok {
		return true
	}
	if entry.NextRemoteRefreshUnixNano <= now.UnixNano() {
		return true
	}
	c.refreshLocalSessionTTLAt(key, ttl, now)
	return false
}

func (c *gatewayCache) shouldWriteRemoteSessionAccount(key string, accountID int64, ttl time.Duration, now time.Time) bool {
	if accountID <= 0 {
		return false
	}
	entry, ok := c.getLocalSessionEntry(key)
	if !ok {
		return true
	}
	if entry.AccountID != accountID {
		return true
	}
	if entry.NextRemoteRefreshUnixNano <= now.UnixNano() {
		return true
	}
	c.refreshLocalSessionTTLAt(key, ttl, now)
	return false
}

func (c *gatewayCache) GetSessionAccountID(ctx context.Context, groupID int64, sessionHash string) (int64, error) {
	key := buildSessionKey(groupID, sessionHash)
	if accountID, ok := c.getLocalSessionAccountID(key); ok {
		return accountID, nil
	}
	accountID, err := c.rdb.Get(ctx, key).Int64()
	if err != nil {
		return 0, err
	}
	c.cacheSessionAccountID(key, accountID, gatewayCacheLocalMaxTTL)
	return accountID, nil
}

func (c *gatewayCache) SetSessionAccountID(ctx context.Context, groupID int64, sessionHash string, accountID int64, ttl time.Duration) error {
	key := buildSessionKey(groupID, sessionHash)
	now := time.Now()
	if !c.shouldWriteRemoteSessionAccount(key, accountID, ttl, now) {
		return nil
	}
	if err := c.rdb.Set(ctx, key, accountID, ttl).Err(); err != nil {
		return err
	}
	c.cacheSessionAccountIDAt(key, accountID, ttl, now)
	return nil
}

func (c *gatewayCache) RefreshSessionTTL(ctx context.Context, groupID int64, sessionHash string, ttl time.Duration) error {
	key := buildSessionKey(groupID, sessionHash)
	now := time.Now()
	if !c.shouldRefreshRemoteSessionTTL(key, ttl, now) {
		return nil
	}
	if err := c.rdb.Expire(ctx, key, ttl).Err(); err != nil {
		return err
	}
	if entry, ok := c.getLocalSessionEntry(key); ok {
		c.cacheSessionAccountIDAt(key, entry.AccountID, ttl, now)
	}
	return nil
}

// DeleteSessionAccountID 删除粘性会话与账号的绑定关系。
// 当检测到绑定的账号不可用（如状态错误、禁用、不可调度等）时调用，
// 以便下次请求能够重新选择可用账号。
//
// DeleteSessionAccountID removes the sticky session binding for the given session.
// Called when the bound account becomes unavailable (e.g., error status, disabled,
// or unschedulable), allowing subsequent requests to select a new available account.
func (c *gatewayCache) DeleteSessionAccountID(ctx context.Context, groupID int64, sessionHash string) error {
	key := buildSessionKey(groupID, sessionHash)
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return err
	}
	if c.localCache != nil {
		c.localCache.Delete(key)
	}
	return nil
}
