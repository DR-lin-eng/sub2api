package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountOpenAICompactCapability(t *testing.T) {
	tests := []struct {
		name          string
		account       *Account
		wantMode      string
		wantSupported bool
		wantKnown     bool
		wantAllows    bool
	}{
		{
			name:       "nil",
			account:    nil,
			wantMode:   OpenAICompactModeAuto,
			wantAllows: false,
		},
		{
			name:       "non openai",
			account:    &Account{Platform: PlatformAnthropic},
			wantMode:   OpenAICompactModeAuto,
			wantAllows: false,
		},
		{
			name:       "openai unknown auto allows",
			account:    &Account{Platform: PlatformOpenAI, Extra: map[string]any{}},
			wantMode:   OpenAICompactModeAuto,
			wantAllows: true,
		},
		{
			name:          "probe supported",
			account:       &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_compact_supported": true}},
			wantMode:      OpenAICompactModeAuto,
			wantSupported: true,
			wantKnown:     true,
			wantAllows:    true,
		},
		{
			name:       "probe unsupported",
			account:    &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_compact_supported": false}},
			wantMode:   OpenAICompactModeAuto,
			wantKnown:  true,
			wantAllows: false,
		},
		{
			name:          "force on overrides probe false",
			account:       &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_compact_mode": OpenAICompactModeForceOn, "openai_compact_supported": false}},
			wantMode:      OpenAICompactModeForceOn,
			wantSupported: true,
			wantKnown:     true,
			wantAllows:    true,
		},
		{
			name:       "force off overrides probe true",
			account:    &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_compact_mode": OpenAICompactModeForceOff, "openai_compact_supported": true}},
			wantMode:   OpenAICompactModeForceOff,
			wantKnown:  true,
			wantAllows: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantMode, tt.account.GetOpenAICompactMode())
			gotSupported, gotKnown := tt.account.OpenAICompactSupportKnown()
			require.Equal(t, tt.wantSupported, gotSupported)
			require.Equal(t, tt.wantKnown, gotKnown)
			require.Equal(t, tt.wantAllows, tt.account.AllowsOpenAICompact())
		})
	}
}

func TestAccountResolveCompactMappedModel(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Credentials: map[string]any{
			"compact_model_mapping": map[string]any{
				"gpt-5.4":       "gpt-5.4-compact",
				"gpt-5-codex-*": "gpt-5-codex-compact",
			},
		},
	}

	mapped, matched := account.ResolveCompactMappedModel("gpt-5.4")
	require.True(t, matched)
	require.Equal(t, "gpt-5.4-compact", mapped)

	mapped, matched = account.ResolveCompactMappedModel("gpt-5-codex-high")
	require.True(t, matched)
	require.Equal(t, "gpt-5-codex-compact", mapped)

	mapped, matched = account.ResolveCompactMappedModel("gpt-4.1")
	require.False(t, matched)
	require.Equal(t, "gpt-4.1", mapped)
}
