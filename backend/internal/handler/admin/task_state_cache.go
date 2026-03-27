package admin

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var (
	defaultAdminTaskStateCache   service.TaskStateCache
	defaultAdminTaskStateCacheMu sync.RWMutex
	defaultAdminTaskStateRepo    service.AdminTaskStateRepository
	defaultAdminTaskStateRepoMu  sync.RWMutex
)

func SetDefaultTaskStateCache(cache service.TaskStateCache) {
	defaultAdminTaskStateCacheMu.Lock()
	defaultAdminTaskStateCache = cache
	defaultAdminTaskStateCacheMu.Unlock()
}

func getDefaultTaskStateCache() service.TaskStateCache {
	defaultAdminTaskStateCacheMu.RLock()
	defer defaultAdminTaskStateCacheMu.RUnlock()
	return defaultAdminTaskStateCache
}

func SetDefaultTaskStateRepository(repo service.AdminTaskStateRepository) {
	defaultAdminTaskStateRepoMu.Lock()
	defaultAdminTaskStateRepo = repo
	defaultAdminTaskStateRepoMu.Unlock()
}

func getDefaultTaskStateRepository() service.AdminTaskStateRepository {
	defaultAdminTaskStateRepoMu.RLock()
	defer defaultAdminTaskStateRepoMu.RUnlock()
	return defaultAdminTaskStateRepo
}

func loadTaskStateJSON[T any](ctx context.Context, key string) (*T, bool) {
	cache := getDefaultTaskStateCache()
	if cache == nil || key == "" {
		return nil, false
	}
	raw, err := cache.Get(ctx, key)
	if err != nil || raw == "" {
		return nil, false
	}
	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false
	}
	return &out, true
}

func loadTaskStateJSONWithRepo[T any](ctx context.Context, key string, taskKind string, taskID string) (*T, bool) {
	if out, ok := loadTaskStateJSON[T](ctx, key); ok {
		return out, true
	}
	repo := getDefaultTaskStateRepository()
	if repo == nil || taskKind == "" || taskID == "" {
		return nil, false
	}
	raw, err := repo.GetState(ctx, taskKind, taskID)
	if err != nil || raw == "" {
		return nil, false
	}
	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false
	}
	return &out, true
}

func storeTaskStateJSON(ctx context.Context, key string, ttl time.Duration, payload any) {
	cache := getDefaultTaskStateCache()
	if cache == nil || key == "" || payload == nil {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = cache.Set(ctx, key, string(raw), ttl)
}

func storeTaskStateJSONWithRepo(ctx context.Context, key string, taskKind string, taskID string, ttl time.Duration, status string, finishedAt *time.Time, payload any) {
	storeTaskStateJSON(ctx, key, ttl, payload)
	repo := getDefaultTaskStateRepository()
	if repo == nil || taskKind == "" || taskID == "" || payload == nil {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	var expiresAt *time.Time
	if ttl > 0 {
		exp := time.Now().UTC().Add(ttl)
		expiresAt = &exp
	}
	_ = repo.UpsertState(ctx, taskKind, taskID, string(raw), status, expiresAt, finishedAt)
}

func deleteTaskStateJSON(ctx context.Context, key string) {
	cache := getDefaultTaskStateCache()
	if cache == nil || key == "" {
		return
	}
	_ = cache.Delete(ctx, key)
}
