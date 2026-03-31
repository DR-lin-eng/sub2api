package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var (
	opsErrorListSnapshotCache     = newSnapshotCache(3 * time.Second)
	opsRequestDetailSnapshotCache = newSnapshotCache(3 * time.Second)
)

type opsErrorListCachePayload struct {
	Result *service.OpsErrorLogList `json:"result"`
}

type opsRequestDetailCachePayload struct {
	Items    []*service.OpsRequestDetail `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
}

func getOpsErrorListCached(
	ctx context.Context,
	key any,
	load func(context.Context) (*service.OpsErrorLogList, error),
) (opsErrorListCachePayload, string, bool, error) {
	keyRaw, _ := json.Marshal(key)
	entry, hit, err := opsErrorListSnapshotCache.GetOrLoad(string(keyRaw), func() (any, error) {
		result, err := load(ctx)
		if err != nil {
			return nil, err
		}
		return opsErrorListCachePayload{Result: result}, nil
	})
	if err != nil {
		return opsErrorListCachePayload{}, "", hit, err
	}
	payload, err := snapshotPayloadAs[opsErrorListCachePayload](entry.Payload)
	return payload, entry.ETag, hit, err
}

func getOpsRequestDetailCached(
	ctx context.Context,
	key any,
	load func(context.Context) (*service.OpsRequestDetailList, error),
) (opsRequestDetailCachePayload, string, bool, error) {
	keyRaw, _ := json.Marshal(key)
	entry, hit, err := opsRequestDetailSnapshotCache.GetOrLoad(string(keyRaw), func() (any, error) {
		result, err := load(ctx)
		if err != nil {
			return nil, err
		}
		return opsRequestDetailCachePayload{
			Items:    result.Items,
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		}, nil
	})
	if err != nil {
		return opsRequestDetailCachePayload{}, "", hit, err
	}
	payload, err := snapshotPayloadAs[opsRequestDetailCachePayload](entry.Payload)
	return payload, entry.ETag, hit, err
}
