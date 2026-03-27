package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyProcessingPrefix = "idempotency:processing:"
	idempotencyReplayPrefix     = "idempotency:replay:"
)

type idempotencyCache struct {
	rdb *redis.Client
}

func NewIdempotencyCache(rdb *redis.Client) service.IdempotencyCache {
	if rdb == nil {
		return nil
	}
	return &idempotencyCache{rdb: rdb}
}

func (c *idempotencyCache) GetProcessing(ctx context.Context, scope, keyHash string) (*service.IdempotencyCacheState, error) {
	return c.get(ctx, idempotencyProcessingKey(scope, keyHash))
}

func (c *idempotencyCache) GetReplay(ctx context.Context, scope, keyHash string) (*service.IdempotencyCacheState, error) {
	return c.get(ctx, idempotencyReplayKey(scope, keyHash))
}

func (c *idempotencyCache) SetProcessing(ctx context.Context, scope, keyHash string, state *service.IdempotencyCacheState, ttl time.Duration) error {
	return c.set(ctx, idempotencyProcessingKey(scope, keyHash), state, ttl)
}

func (c *idempotencyCache) SetReplay(ctx context.Context, scope, keyHash string, state *service.IdempotencyCacheState, ttl time.Duration) error {
	return c.set(ctx, idempotencyReplayKey(scope, keyHash), state, ttl)
}

func (c *idempotencyCache) ClearProcessing(ctx context.Context, scope, keyHash string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, idempotencyProcessingKey(scope, keyHash)).Err()
}

func (c *idempotencyCache) ClearReplay(ctx context.Context, scope, keyHash string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, idempotencyReplayKey(scope, keyHash)).Err()
}

func (c *idempotencyCache) get(ctx context.Context, key string) (*service.IdempotencyCacheState, error) {
	if c == nil || c.rdb == nil || key == "" {
		return nil, nil
	}
	raw, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var state service.IdempotencyCacheState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		_ = c.rdb.Del(ctx, key).Err()
		return nil, nil
	}
	return &state, nil
}

func (c *idempotencyCache) set(ctx context.Context, key string, state *service.IdempotencyCacheState, ttl time.Duration) error {
	if c == nil || c.rdb == nil || key == "" || state == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = time.Second
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, raw, ttl).Err()
}

func idempotencyProcessingKey(scope, keyHash string) string {
	return idempotencyProcessingPrefix + scope + ":" + keyHash
}

func idempotencyReplayKey(scope, keyHash string) string {
	return idempotencyReplayPrefix + scope + ":" + keyHash
}
