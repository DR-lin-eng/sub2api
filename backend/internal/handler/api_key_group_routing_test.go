package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSelectOpenAIAPIKeyGroupWithCompact_PropagatesRequireCompact(t *testing.T) {
	groupID := int64(42)
	apiKey := &service.APIKey{ID: 7, GroupID: &groupID}
	var gotRequireCompact bool

	selection, err := selectOpenAIAPIKeyGroupWithCompact(
		context.Background(),
		apiKey,
		"resp_123",
		"session_123",
		"gpt-5.4",
		nil,
		service.OpenAIUpstreamTransportAny,
		true,
		func(ctx context.Context, groupID *int64, previousResponseID, sessionHash, requestedModel string, excludedIDs map[int64]struct{}, requiredTransport service.OpenAIUpstreamTransport, requireCompact bool) (*service.AccountSelectionResult, service.OpenAIAccountScheduleDecision, error) {
			gotRequireCompact = requireCompact
			return &service.AccountSelectionResult{
				Account: &service.Account{ID: 99, Platform: service.PlatformOpenAI},
			}, service.OpenAIAccountScheduleDecision{}, nil
		},
	)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.True(t, gotRequireCompact)
}
