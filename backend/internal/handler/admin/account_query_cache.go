package admin

import (
	"context"
	"encoding/json"
	"time"
)

var accountListSnapshotCache = newSnapshotCache(5 * time.Second)

type accountListCacheKey struct {
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	Platform    string `json:"platform"`
	AccountType string `json:"type"`
	Status      string `json:"status"`
	Search      string `json:"search"`
	Plan        string `json:"plan"`
	OAuthType   string `json:"oauth_type"`
	TierID      string `json:"tier_id"`
	Group       int64  `json:"group"`
	Lite        bool   `json:"lite"`
}

type accountListCachePayload struct {
	Items    []AccountWithConcurrency `json:"items"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

func (h *AccountHandler) getAccountListCached(
	ctx context.Context,
	page, pageSize int,
	platform, accountType, status, search, plan, oauthType, tierID string,
	groupID int64,
	lite bool,
	load func(context.Context) ([]AccountWithConcurrency, int64, error),
) (accountListCachePayload, bool, error) {
	keyRaw, _ := json.Marshal(accountListCacheKey{
		Page:        page,
		PageSize:    pageSize,
		Platform:    platform,
		AccountType: accountType,
		Status:      status,
		Search:      search,
		Plan:        plan,
		OAuthType:   oauthType,
		TierID:      tierID,
		Group:       groupID,
		Lite:        lite,
	})
	entry, hit, err := accountListSnapshotCache.GetOrLoad(string(keyRaw), func() (any, error) {
		items, total, err := load(ctx)
		if err != nil {
			return nil, err
		}
		return accountListCachePayload{
			Items:    items,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	})
	if err != nil {
		return accountListCachePayload{}, hit, err
	}
	payload, err := snapshotPayloadAs[accountListCachePayload](entry.Payload)
	return payload, hit, err
}
