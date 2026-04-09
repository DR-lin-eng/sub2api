package service

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memoryIdentityCacheStub struct {
	fingerprint     *Fingerprint
	maskedSessionID string
}

func (s *memoryIdentityCacheStub) GetFingerprint(_ context.Context, _ int64) (*Fingerprint, error) {
	if s.fingerprint == nil {
		return nil, nil
	}
	cloned := *s.fingerprint
	return &cloned, nil
}

func (s *memoryIdentityCacheStub) SetFingerprint(_ context.Context, _ int64, fp *Fingerprint) error {
	if fp == nil {
		s.fingerprint = nil
		return nil
	}
	cloned := *fp
	s.fingerprint = &cloned
	return nil
}

func (s *memoryIdentityCacheStub) GetMaskedSessionID(_ context.Context, _ int64) (string, error) {
	return s.maskedSessionID, nil
}

func (s *memoryIdentityCacheStub) SetMaskedSessionID(_ context.Context, _ int64, sessionID string) error {
	s.maskedSessionID = sessionID
	return nil
}

func TestIdentityService_ObserveOfficialFingerprintSample_StoresOriginator(t *testing.T) {
	cache := &memoryIdentityCacheStub{}
	svc := NewIdentityService(cache)

	headers := http.Header{}
	headers.Set("User-Agent", "codex_vscode/1.2.3")
	headers.Set("originator", "codex_vscode")
	headers.Set("x-claude-code-session-id", "11111111-2222-4333-8444-555555555555")
	headers.Set("x-claude-remote-container-id", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	headers.Set("x-claude-remote-session-id", "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee")
	headers.Set("x-client-app", "my-app/1.0.0")
	headers.Set("x-anthropic-additional-protection", "true")

	fp, err := svc.ObserveOfficialFingerprintSample(context.Background(), 101, headers)
	require.NoError(t, err)
	require.NotNil(t, fp)
	require.True(t, fp.OfficialSampled)
	require.Equal(t, "codex_vscode/1.2.3", fp.UserAgent)
	require.Equal(t, "codex_vscode", fp.Originator)
	require.Equal(t, "11111111-2222-4333-8444-555555555555", fp.ClaudeCodeSessionID)
	require.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", fp.ClaudeRemoteContainerID)
	require.Equal(t, "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", fp.ClaudeRemoteSessionID)
	require.Equal(t, "my-app/1.0.0", fp.ClientApp)
	require.Equal(t, "true", fp.AdditionalProtection)
	require.NotEmpty(t, fp.ClientID)

	cached, err := svc.GetSampledFingerprint(context.Background(), 101)
	require.NoError(t, err)
	require.NotNil(t, cached)
	require.Equal(t, "codex_vscode/1.2.3", cached.UserAgent)
	require.Equal(t, "codex_vscode", cached.Originator)
	require.Equal(t, "11111111-2222-4333-8444-555555555555", cached.ClaudeCodeSessionID)
	require.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", cached.ClaudeRemoteContainerID)
	require.Equal(t, "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", cached.ClaudeRemoteSessionID)
	require.Equal(t, "my-app/1.0.0", cached.ClientApp)
	require.Equal(t, "true", cached.AdditionalProtection)
}

func TestIdentityService_ApplyOpenAIFingerprint_SetsUserAgentAndOriginator(t *testing.T) {
	svc := NewIdentityService(&memoryIdentityCacheStub{})
	req, err := http.NewRequest(http.MethodPost, "https://example.com", nil)
	require.NoError(t, err)

	svc.ApplyOpenAIFingerprint(req, &Fingerprint{
		UserAgent:  "codex_app/2.0.0",
		Originator: "codex_chatgpt_desktop",
	})

	require.Equal(t, "codex_app/2.0.0", req.Header.Get("User-Agent"))
	require.Equal(t, "codex_chatgpt_desktop", req.Header.Get("originator"))
}

func TestIdentityService_ObserveOfficialFingerprintSample_RefreshesChangedSampleImmediately(t *testing.T) {
	cache := &memoryIdentityCacheStub{
		fingerprint: &Fingerprint{
			ClientID:        "client-1",
			UserAgent:       "codex_cli_rs/0.99.0",
			Originator:      "codex_cli_rs",
			OfficialSampled: true,
			UpdatedAt:       time.Now().Add(-time.Hour).Unix(),
		},
	}
	svc := NewIdentityService(cache)

	headers := http.Header{}
	headers.Set("User-Agent", "codex_app/2.1.0")
	headers.Set("originator", "codex_chatgpt_desktop")

	fp, err := svc.ObserveOfficialFingerprintSample(context.Background(), 101, headers)
	require.NoError(t, err)
	require.NotNil(t, fp)
	require.Equal(t, "client-1", fp.ClientID)
	require.Equal(t, "codex_app/2.1.0", fp.UserAgent)
	require.Equal(t, "codex_chatgpt_desktop", fp.Originator)
}

func TestIdentityService_ApplyAnthropicFingerprint_RewritesRemoteContainerIDForNonOfficialReuse(t *testing.T) {
	cache := &memoryIdentityCacheStub{}
	svc := NewIdentityService(cache)
	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages?beta=true", nil)
	require.NoError(t, err)

	account := &Account{
		ID:       501,
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
	}
	originalUserID := FormatMetadataUserID(
		"device-1234567890abcdefdevice-1234567890abcdefdevice-1234567890ab",
		"acc-uuid",
		"11111111-2222-4333-8444-555555555555",
		"2.1.88",
	)
	body := []byte(`{"metadata":{"user_id":` + samplingStrconvQuote(originalUserID) + `}}`)
	fp := &Fingerprint{
		UserAgent:               "claude-cli/2.1.88 (external, cli)",
		ClaudeCodeSessionID:     "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		ClaudeRemoteContainerID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ClaudeRemoteSessionID:   "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff",
		ClientApp:               "agent-sdk/test-app/1.0.0",
		AdditionalProtection:    "true",
	}

	svc.ApplyAnthropicFingerprint(req, body, account, fp, false)

	got := req.Header.Get("x-claude-remote-container-id")
	require.Len(t, got, 64)
	require.NotEqual(t, fp.ClaudeRemoteContainerID, got)
	require.NotEqual(t, fp.ClaudeCodeSessionID, req.Header.Get("x-claude-code-session-id"))
	require.NotEqual(t, fp.ClaudeRemoteSessionID, req.Header.Get("x-claude-remote-session-id"))
	require.Equal(t, "agent-sdk/test-app/1.0.0", req.Header.Get("x-client-app"))
	require.Equal(t, "true", req.Header.Get("x-anthropic-additional-protection"))
	require.NotEmpty(t, req.Header.Get("x-client-request-id"))
	require.Equal(t, "claude-cli/2.1.88 (external, cli)", req.Header.Get("User-Agent"))
}

func samplingStrconvQuote(v string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`) + `"`
}
