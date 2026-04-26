//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/stretchr/testify/require"
)

type adminSoraAccountRepoStub struct {
	AccountRepository
	accountToReturn *Account
	created         *Account
	updated         *Account
}

func (s *adminSoraAccountRepoStub) Create(_ context.Context, account *Account) error {
	clone := *account
	if account.Credentials != nil {
		clone.Credentials = make(map[string]any, len(account.Credentials))
		for key, value := range account.Credentials {
			clone.Credentials[key] = value
		}
	}
	s.created = &clone
	if s.accountToReturn == nil {
		s.accountToReturn = &clone
	}
	return nil
}

func (s *adminSoraAccountRepoStub) GetByID(_ context.Context, _ int64) (*Account, error) {
	if s.accountToReturn == nil {
		return nil, ErrAccountNotFound
	}
	clone := *s.accountToReturn
	if s.accountToReturn.Credentials != nil {
		clone.Credentials = make(map[string]any, len(s.accountToReturn.Credentials))
		for key, value := range s.accountToReturn.Credentials {
			clone.Credentials[key] = value
		}
	}
	return &clone, nil
}

func (s *adminSoraAccountRepoStub) Update(_ context.Context, account *Account) error {
	clone := *account
	if account.Credentials != nil {
		clone.Credentials = make(map[string]any, len(account.Credentials))
		for key, value := range account.Credentials {
			clone.Credentials[key] = value
		}
	}
	s.updated = &clone
	s.accountToReturn = &clone
	return nil
}

func (s *adminSoraAccountRepoStub) BindGroups(context.Context, int64, []int64) error { return nil }

type adminSoraGroupRepoStub struct {
	GroupRepository
}

func (s *adminSoraGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	return nil, nil
}

type soraForwardHTTPUpstreamStub struct {
	requests []*http.Request
}

func (s *soraForwardHTTPUpstreamStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	s.requests = append(s.requests, req)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}, nil
}

func (s *soraForwardHTTPUpstreamStub) DoWithTLS(*http.Request, string, int64, int, bool) (*http.Response, error) {
	panic("unexpected")
}

func TestNormalizeSoraAPIKeyBaseURLRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{name: "missing", raw: "", wantErr: "required"},
		{name: "malformed", raw: "://bad", wantErr: "invalid url"},
		{name: "userinfo", raw: "https://user:pass@example.com", wantErr: "userinfo"},
		{name: "localhost", raw: "https://localhost", wantErr: "not allowed"},
		{name: "metadata", raw: "https://metadata.google.internal", wantErr: "not allowed"},
		{name: "private_ip", raw: "https://10.0.0.1", wantErr: "not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeSoraAPIKeyBaseURL(tt.raw)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestAdminServiceCreateAccount_RejectsUnsafeSoraBaseURL(t *testing.T) {
	repo := &adminSoraAccountRepoStub{}
	svc := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &adminSoraGroupRepoStub{},
	}

	_, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:        "sora",
		Platform:    PlatformSora,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://127.0.0.1"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "base_url")
}

func TestAdminServiceUpdateAccount_RejectsUnsafeSoraBaseURL(t *testing.T) {
	repo := &adminSoraAccountRepoStub{
		accountToReturn: &Account{
			ID:       1,
			Name:     "sora",
			Platform: PlatformSora,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://sora.example.com",
			},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &adminSoraGroupRepoStub{},
	}

	_, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{
		Credentials: map[string]any{"base_url": "https://user:pass@example.com"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "base_url")
}

func TestSoraGatewayServiceForwardContext_RejectsMissingOrUnsafeExplicitBaseURL(t *testing.T) {
	upstream := &soraForwardHTTPUpstreamStub{}
	svc := NewSoraGatewayService(nil, nil, upstream, nil)

	tests := []struct {
		name        string
		credentials map[string]any
	}{
		{name: "missing", credentials: map[string]any{"api_key": "sk-test"}},
		{name: "unsafe", credentials: map[string]any{"api_key": "sk-test", "base_url": "https://169.254.169.254"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/sora", strings.NewReader(`{}`))
			req.RemoteAddr = "203.0.113.20:1234"
			ctx := gatewayctx.NewNative(req, rec, nil, req.RemoteAddr)

			_, err := svc.ForwardContext(context.Background(), ctx, &Account{
				ID:          1,
				Platform:    PlatformSora,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Credentials: tt.credentials,
			}, []byte(`{}`), false)
			require.Error(t, err)
			require.Empty(t, upstream.requests)
		})
	}
}

func TestSoraGatewayServiceForwardContext_UsesExplicitValidatedBaseURL(t *testing.T) {
	upstream := &soraForwardHTTPUpstreamStub{}
	svc := NewSoraGatewayService(nil, nil, upstream, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sora", strings.NewReader(`{"prompt":"demo"}`))
	req.RemoteAddr = "203.0.113.21:1234"
	ctx := gatewayctx.NewNative(req, rec, nil, req.RemoteAddr)

	result, err := svc.ForwardContext(context.Background(), ctx, &Account{
		ID:       1,
		Platform: PlatformSora,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://sora.example.com/custom/",
		},
	}, []byte(`{"prompt":"demo"}`), false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://sora.example.com/custom/sora/v1/chat/completions", upstream.requests[0].URL.String())
}
