package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type taskStateCache struct {
	rdb *redis.Client
}

func NewTaskStateCache(rdb *redis.Client) service.TaskStateCache {
	if rdb == nil {
		return nil
	}
	return &taskStateCache{rdb: rdb}
}

func (c *taskStateCache) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.rdb == nil || key == "" {
		return "", nil
	}
	value, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return value, err
}

func (c *taskStateCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if c == nil || c.rdb == nil || key == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *taskStateCache) Delete(ctx context.Context, key string) error {
	if c == nil || c.rdb == nil || key == "" {
		return nil
	}
	return c.rdb.Del(ctx, key).Err()
}
