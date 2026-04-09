package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	schedulerBucketSetKey       = "sched:buckets"
	schedulerOutboxWatermarkKey = "sched:outbox:watermark"
	schedulerAccountPrefix      = "sched:acc:"
	schedulerActivePrefix       = "sched:active:"
	schedulerReadyPrefix        = "sched:ready:"
	schedulerVersionPrefix      = "sched:ver:"
	schedulerSnapshotPrefix     = "sched:"
	schedulerLockPrefix         = "sched:lock:"
	// 防止 Redis 单次 MGET 参数过大阻塞事件循环。
	schedulerSnapshotMGetBatchSize = 500
	// 防止单次 pipeline 塞入过多 SET / ZADD 命令。
	schedulerSnapshotWriteBatchSize = 500
)

type schedulerCache struct {
	rdb *redis.Client
}

func NewSchedulerCache(rdb *redis.Client) service.SchedulerCache {
	return &schedulerCache{rdb: rdb}
}

func (c *schedulerCache) GetSnapshot(ctx context.Context, bucket service.SchedulerBucket) ([]*service.Account, bool, error) {
	readyKey := schedulerBucketKey(schedulerReadyPrefix, bucket)
	readyVal, err := c.rdb.Get(ctx, readyKey).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if readyVal != "1" {
		return nil, false, nil
	}

	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	activeVal, err := c.rdb.Get(ctx, activeKey).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	snapshotKey := schedulerSnapshotKey(bucket, activeVal)
	ids, err := c.rdb.ZRange(ctx, snapshotKey, 0, -1).Result()
	if err != nil {
		return nil, false, err
	}
	if len(ids) == 0 {
		// 空快照视为缓存未命中，触发数据库回退查询
		// 这解决了新分组创建后立即绑定账号时的竞态条件问题
		return nil, false, nil
	}

	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, schedulerAccountKey(id))
	}
	values, err := c.mGetInBatches(ctx, keys)
	if err != nil {
		return nil, false, err
	}

	accounts := make([]*service.Account, 0, len(values))
	for _, val := range values {
		if val == nil {
			return nil, false, nil
		}
		account, err := decodeCachedAccount(val)
		if err != nil {
			return nil, false, err
		}
		accounts = append(accounts, account)
	}

	return accounts, true, nil
}

func (c *schedulerCache) SetSnapshot(ctx context.Context, bucket service.SchedulerBucket, accounts []service.Account) error {
	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	oldActive, _ := c.rdb.Get(ctx, activeKey).Result()

	versionKey := schedulerBucketKey(schedulerVersionPrefix, bucket)
	version, err := c.rdb.Incr(ctx, versionKey).Result()
	if err != nil {
		return err
	}

	versionStr := strconv.FormatInt(version, 10)
	snapshotKey := schedulerSnapshotKey(bucket, versionStr)

	if len(accounts) == 0 {
		pipe := c.rdb.Pipeline()
		pipe.Del(ctx, snapshotKey)
		pipe.Set(ctx, activeKey, versionStr, 0)
		pipe.Set(ctx, schedulerBucketKey(schedulerReadyPrefix, bucket), "1", 0)
		pipe.SAdd(ctx, schedulerBucketSetKey, bucket.String())
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	} else {
		for start := 0; start < len(accounts); start += schedulerSnapshotWriteBatchSize {
			end := start + schedulerSnapshotWriteBatchSize
			if end > len(accounts) {
				end = len(accounts)
			}
			pipe := c.rdb.Pipeline()
			members := make([]redis.Z, 0, end-start)
			for idx := start; idx < end; idx++ {
				account := accounts[idx]
				payload, err := json.Marshal(account)
				if err != nil {
					return err
				}
				pipe.Set(ctx, schedulerAccountKey(strconv.FormatInt(account.ID, 10)), payload, 0)
				members = append(members, redis.Z{
					Score:  float64(idx),
					Member: strconv.FormatInt(account.ID, 10),
				})
			}
			if len(members) > 0 {
				pipe.ZAdd(ctx, snapshotKey, members...)
			}
			if _, err := pipe.Exec(ctx); err != nil {
				return err
			}
		}
		pipe := c.rdb.Pipeline()
		pipe.Set(ctx, activeKey, versionStr, 0)
		pipe.Set(ctx, schedulerBucketKey(schedulerReadyPrefix, bucket), "1", 0)
		pipe.SAdd(ctx, schedulerBucketSetKey, bucket.String())
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}

	if oldActive != "" && oldActive != versionStr {
		_ = c.rdb.Del(ctx, schedulerSnapshotKey(bucket, oldActive)).Err()
	}

	return nil
}

func (c *schedulerCache) GetAccount(ctx context.Context, accountID int64) (*service.Account, error) {
	key := schedulerAccountKey(strconv.FormatInt(accountID, 10))
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeCachedAccount(val)
}

func (c *schedulerCache) SetAccount(ctx context.Context, account *service.Account) error {
	if account == nil || account.ID <= 0 {
		return nil
	}
	payload, err := json.Marshal(account)
	if err != nil {
		return err
	}
	key := schedulerAccountKey(strconv.FormatInt(account.ID, 10))
	return c.rdb.Set(ctx, key, payload, 0).Err()
}

func (c *schedulerCache) DeleteAccount(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return nil
	}
	key := schedulerAccountKey(strconv.FormatInt(accountID, 10))
	return c.rdb.Del(ctx, key).Err()
}

func (c *schedulerCache) UpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	if len(updates) == 0 {
		return nil
	}

	keys := make([]string, 0, len(updates))
	ids := make([]int64, 0, len(updates))
	for id := range updates {
		keys = append(keys, schedulerAccountKey(strconv.FormatInt(id, 10)))
		ids = append(ids, id)
	}

	values, err := c.mGetInBatches(ctx, keys)
	if err != nil {
		return err
	}

	for start := 0; start < len(values); start += schedulerSnapshotWriteBatchSize {
		end := start + schedulerSnapshotWriteBatchSize
		if end > len(values) {
			end = len(values)
		}
		pipe := c.rdb.Pipeline()
		for i := start; i < end; i++ {
			val := values[i]
			if val == nil {
				continue
			}
			account, err := decodeCachedAccount(val)
			if err != nil {
				return err
			}
			account.LastUsedAt = ptrTime(updates[ids[i]])
			updated, err := json.Marshal(account)
			if err != nil {
				return err
			}
			pipe.Set(ctx, keys[i], updated, 0)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *schedulerCache) TryLockBucket(ctx context.Context, bucket service.SchedulerBucket, ttl time.Duration) (bool, error) {
	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	return c.rdb.SetNX(ctx, key, time.Now().UnixNano(), ttl).Result()
}

func (c *schedulerCache) ListBuckets(ctx context.Context) ([]service.SchedulerBucket, error) {
	raw, err := c.rdb.SMembers(ctx, schedulerBucketSetKey).Result()
	if err != nil {
		return nil, err
	}
	out := make([]service.SchedulerBucket, 0, len(raw))
	for _, entry := range raw {
		bucket, ok := service.ParseSchedulerBucket(entry)
		if !ok {
			continue
		}
		out = append(out, bucket)
	}
	return out, nil
}

func (c *schedulerCache) GetOutboxWatermark(ctx context.Context) (int64, error) {
	val, err := c.rdb.Get(ctx, schedulerOutboxWatermarkKey).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (c *schedulerCache) SetOutboxWatermark(ctx context.Context, id int64) error {
	return c.rdb.Set(ctx, schedulerOutboxWatermarkKey, strconv.FormatInt(id, 10), 0).Err()
}

func schedulerBucketKey(prefix string, bucket service.SchedulerBucket) string {
	return fmt.Sprintf("%s%d:%s:%s", prefix, bucket.GroupID, bucket.Platform, bucket.Mode)
}

func schedulerSnapshotKey(bucket service.SchedulerBucket, version string) string {
	return fmt.Sprintf("%s%d:%s:%s:v%s", schedulerSnapshotPrefix, bucket.GroupID, bucket.Platform, bucket.Mode, version)
}

func schedulerAccountKey(id string) string {
	return schedulerAccountPrefix + id
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func decodeCachedAccount(val any) (*service.Account, error) {
	var payload []byte
	switch raw := val.(type) {
	case string:
		payload = []byte(raw)
	case []byte:
		payload = raw
	default:
		return nil, fmt.Errorf("unexpected account cache type: %T", val)
	}
	var account service.Account
	if err := json.Unmarshal(payload, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

func (c *schedulerCache) mGetInBatches(ctx context.Context, keys []string) ([]any, error) {
	if len(keys) == 0 {
		return []any{}, nil
	}
	if len(keys) <= schedulerSnapshotMGetBatchSize {
		return c.rdb.MGet(ctx, keys...).Result()
	}

	results := make([]any, 0, len(keys))
	for start := 0; start < len(keys); start += schedulerSnapshotMGetBatchSize {
		end := start + schedulerSnapshotMGetBatchSize
		if end > len(keys) {
			end = len(keys)
		}
		values, err := c.rdb.MGet(ctx, keys[start:end]...).Result()
		if err != nil {
			return nil, err
		}
		results = append(results, values...)
	}
	return results, nil
}
