package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/cespare/xxhash/v2"
)

type gatewayAPIKeySelection struct {
	APIKey    *service.APIKey
	Selection *service.AccountSelectionResult
}

type openAIAPIKeySelection struct {
	APIKey    *service.APIKey
	Selection *service.AccountSelectionResult
	Decision  service.OpenAIAccountScheduleDecision
}

func apiKeySupportsMultiGroupRouting(apiKey *service.APIKey) bool {
	return apiKey != nil && len(apiKey.GroupIDs) > 1 && len(apiKey.Groups) > 1
}

func orderedAPIKeyGroups(apiKey *service.APIKey, seed string) []*service.Group {
	if apiKey == nil {
		return nil
	}

	var groups []*service.Group
	seen := make(map[int64]struct{})
	appendGroup := func(group *service.Group) {
		if group == nil || group.ID <= 0 {
			return
		}
		if _, exists := seen[group.ID]; exists {
			return
		}
		seen[group.ID] = struct{}{}
		groups = append(groups, group)
	}

	for _, group := range apiKey.Groups {
		appendGroup(group)
	}
	appendGroup(apiKey.Group)

	if len(groups) <= 1 {
		return groups
	}

	if strings.TrimSpace(seed) == "" {
		seed = "apikey-multi-group"
	}
	start := int(xxhash.Sum64String(seed) % uint64(len(groups)))
	rotated := make([]*service.Group, 0, len(groups))
	rotated = append(rotated, groups[start:]...)
	rotated = append(rotated, groups[:start]...)
	return rotated
}

func isNoAvailableSelectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, service.ErrNoAvailableAccounts) {
		return true
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "no available")
}

func selectGatewayAPIKeyGroup(
	ctx context.Context,
	apiKey *service.APIKey,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	metadataUserID string,
	selectFn func(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}, metadataUserID string) (*service.AccountSelectionResult, error),
) (*gatewayAPIKeySelection, error) {
	if !apiKeySupportsMultiGroupRouting(apiKey) {
		selection, err := selectFn(ctx, apiKey.GroupID, sessionHash, requestedModel, excludedIDs, metadataUserID)
		if err != nil {
			return nil, err
		}
		return &gatewayAPIKeySelection{APIKey: apiKey, Selection: selection}, nil
	}

	var firstWait *gatewayAPIKeySelection
	var lastErr error
	for _, group := range orderedAPIKeyGroups(apiKey, sessionHash+":"+requestedModel) {
		currentAPIKey := cloneAPIKeyWithGroup(apiKey, group)
		selection, err := selectFn(ctx, currentAPIKey.GroupID, sessionHash, requestedModel, excludedIDs, metadataUserID)
		if err == nil {
			if selection != nil && selection.WaitPlan != nil && !selection.Acquired {
				if firstWait == nil {
					firstWait = &gatewayAPIKeySelection{APIKey: currentAPIKey, Selection: selection}
				}
				continue
			}
			return &gatewayAPIKeySelection{APIKey: currentAPIKey, Selection: selection}, nil
		}
		if isNoAvailableSelectionError(err) {
			lastErr = err
			continue
		}
		return nil, err
	}
	if firstWait != nil {
		return firstWait, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, service.ErrNoAvailableAccounts
}

func selectOpenAIAPIKeyGroup(
	ctx context.Context,
	apiKey *service.APIKey,
	previousResponseID string,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport service.OpenAIUpstreamTransport,
	selectFn func(ctx context.Context, groupID *int64, previousResponseID, sessionHash, requestedModel string, excludedIDs map[int64]struct{}, requiredTransport service.OpenAIUpstreamTransport) (*service.AccountSelectionResult, service.OpenAIAccountScheduleDecision, error),
) (*openAIAPIKeySelection, error) {
	if !apiKeySupportsMultiGroupRouting(apiKey) {
		selection, decision, err := selectFn(ctx, apiKey.GroupID, previousResponseID, sessionHash, requestedModel, excludedIDs, requiredTransport)
		if err != nil {
			return nil, err
		}
		return &openAIAPIKeySelection{APIKey: apiKey, Selection: selection, Decision: decision}, nil
	}

	var firstWait *openAIAPIKeySelection
	var lastErr error
	for _, group := range orderedAPIKeyGroups(apiKey, sessionHash+":"+requestedModel+":"+previousResponseID) {
		currentAPIKey := cloneAPIKeyWithGroup(apiKey, group)
		selection, decision, err := selectFn(ctx, currentAPIKey.GroupID, previousResponseID, sessionHash, requestedModel, excludedIDs, requiredTransport)
		if err == nil {
			if selection != nil && selection.WaitPlan != nil && !selection.Acquired {
				if firstWait == nil {
					firstWait = &openAIAPIKeySelection{APIKey: currentAPIKey, Selection: selection, Decision: decision}
				}
				continue
			}
			return &openAIAPIKeySelection{APIKey: currentAPIKey, Selection: selection, Decision: decision}, nil
		}
		if isNoAvailableSelectionError(err) {
			lastErr = err
			continue
		}
		return nil, err
	}
	if firstWait != nil {
		return firstWait, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, service.ErrNoAvailableAccounts
}

func selectAccountForModelAcrossAPIKeyGroups(
	ctx context.Context,
	apiKey *service.APIKey,
	sessionHash string,
	requestedModel string,
	selectFn func(ctx context.Context, groupID *int64, sessionHash, requestedModel string) (*service.Account, error),
) (*service.APIKey, *service.Account, error) {
	if !apiKeySupportsMultiGroupRouting(apiKey) {
		account, err := selectFn(ctx, apiKey.GroupID, sessionHash, requestedModel)
		return apiKey, account, err
	}

	var lastErr error
	for _, group := range orderedAPIKeyGroups(apiKey, sessionHash+":"+requestedModel) {
		currentAPIKey := cloneAPIKeyWithGroup(apiKey, group)
		account, err := selectFn(ctx, currentAPIKey.GroupID, sessionHash, requestedModel)
		if err == nil {
			return currentAPIKey, account, nil
		}
		if isNoAvailableSelectionError(err) {
			lastErr = err
			continue
		}
		return nil, nil, err
	}
	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, service.ErrNoAvailableAccounts
}

func selectGeminiAIStudioAccountAcrossAPIKeyGroups(
	ctx context.Context,
	apiKey *service.APIKey,
	selectFn func(ctx context.Context, groupID *int64) (*service.Account, error),
) (*service.APIKey, *service.Account, error) {
	if !apiKeySupportsMultiGroupRouting(apiKey) {
		account, err := selectFn(ctx, apiKey.GroupID)
		return apiKey, account, err
	}

	var lastErr error
	for _, group := range orderedAPIKeyGroups(apiKey, "gemini-ai-studio") {
		currentAPIKey := cloneAPIKeyWithGroup(apiKey, group)
		account, err := selectFn(ctx, currentAPIKey.GroupID)
		if err == nil {
			return currentAPIKey, account, nil
		}
		if isNoAvailableSelectionError(err) {
			lastErr = err
			continue
		}
		return nil, nil, err
	}
	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, service.ErrNoAvailableAccounts
}
