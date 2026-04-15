//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type modelsRefreshAccountRepoStub struct {
	accountRepoStub
	account       *Account
	updateExtra   map[string]any
	updateExtraID int64
}

func (s *modelsRefreshAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.account == nil || s.account.ID != id {
		return nil, fmt.Errorf("account %d not found", id)
	}
	cloned := *s.account
	return &cloned, nil
}

func (s *modelsRefreshAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	s.updateExtraID = id
	s.updateExtra = updates
	if s.account != nil && s.account.ID == id {
		if s.account.Extra == nil {
			s.account.Extra = make(map[string]any, len(updates))
		}
		for key, value := range updates {
			s.account.Extra[key] = value
		}
	}
	return nil
}

type modelFetchHTTPUpstreamStub struct {
	resp *http.Response
	err  error
	req  *http.Request
}

func (s *modelFetchHTTPUpstreamStub) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected Do call")
}

func (s *modelFetchHTTPUpstreamStub) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ bool) (*http.Response, error) {
	s.req = req
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func TestAccountTestService_FetchAndCacheAvailableModels_OpenAISuccess(t *testing.T) {
	repo := &modelsRefreshAccountRepoStub{
		account: &Account{
			ID:       7,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key": "sk-test",
			},
		},
	}
	upstream := &modelFetchHTTPUpstreamStub{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-5"},{"id":"gpt-5-mini"}]}`)),
		},
	}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	result, err := svc.FetchAndCacheAvailableModels(context.Background(), 7)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []string{"gpt-5", "gpt-5-mini"}, result.Models)
	require.Equal(t, int64(7), repo.updateExtraID)
	require.Equal(t, []string{"gpt-5", "gpt-5-mini"}, repo.updateExtra[AccountExtraFetchedModelsKey])
	require.NotEmpty(t, repo.updateExtra[AccountExtraModelsFetchedAtKey])
	require.Equal(t, "openai_v1_models", repo.updateExtra[AccountExtraModelsSourceKey])
	require.Equal(t, "", repo.updateExtra[AccountExtraModelsRefreshErrorKey])
	require.NotNil(t, upstream.req)
	require.Equal(t, "https://api.openai.com/v1/models", upstream.req.URL.String())
	require.Equal(t, "Bearer sk-test", upstream.req.Header.Get("Authorization"))
}

func TestAccountTestService_FetchAndCacheAvailableModels_PreservesOldModelsOnFailure(t *testing.T) {
	repo := &modelsRefreshAccountRepoStub{
		account: &Account{
			ID:       8,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key": "sk-test",
			},
			Extra: map[string]any{
				AccountExtraFetchedModelsKey: []any{"gpt-5"},
			},
		},
	}
	upstream := &modelFetchHTTPUpstreamStub{
		err: fmt.Errorf("dial tcp 127.0.0.1:8080: connectex: connection refused"),
	}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	result, err := svc.FetchAndCacheAvailableModels(context.Background(), 8)
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, []string{"gpt-5"}, result.Models)
	require.Equal(t, int64(8), repo.updateExtraID)
	require.Equal(t, truncateModelsRefreshError(err), repo.updateExtra[AccountExtraModelsRefreshErrorKey])
	require.Equal(t, []any{"gpt-5"}, repo.account.Extra[AccountExtraFetchedModelsKey])
}

func TestAccountTestService_FetchAndCacheAvailableModels_OpenAIChatWebUsesTokenProvider(t *testing.T) {
	repo := &modelsRefreshAccountRepoStub{
		account: &Account{
			ID:       9,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"session_token":      "st-live",
				"chatgpt_account_id": "acc-live",
			},
			Extra: map[string]any{
				"openai_auth_mode": OpenAIAuthModeChatWeb,
			},
		},
	}
	upstream := &modelFetchHTTPUpstreamStub{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-5"},{"id":"gpt-5-mini"}]}`)),
		},
	}
	tokenProvider := &openAIAccountTokenProviderStub{token: "fresh-chatweb-token"}
	svc := &AccountTestService{
		accountRepo:         repo,
		httpUpstream:        upstream,
		openAITokenProvider: tokenProvider,
	}

	result, err := svc.FetchAndCacheAvailableModels(context.Background(), 9)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []string{"gpt-5", "gpt-5-mini"}, result.Models)
	require.Equal(t, 1, tokenProvider.calls)
	require.NotNil(t, upstream.req)
	require.Equal(t, "Bearer fresh-chatweb-token", upstream.req.Header.Get("Authorization"))
	require.Equal(t, "acc-live", upstream.req.Header.Get("chatgpt-account-id"))
	require.Equal(t, "fresh-chatweb-token", repo.account.GetOpenAIAccessToken())
}
