package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/curlcffi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

type openaiOAuthClientNoopStub struct{}

func (s *openaiOAuthClientNoopStub) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openaiOAuthClientNoopStub) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openaiOAuthClientNoopStub) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func TestOpenAIOAuthService_ExchangeSoraSessionToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.Header.Get("Cookie"), "__Secure-next-auth.session-token=st-token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"at-token","expires":"2099-01-01T00:00:00Z","user":{"email":"demo@example.com"}}`))
	}))
	defer server.Close()

	origin := openAISoraSessionAuthURL
	openAISoraSessionAuthURL = server.URL
	defer func() { openAISoraSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	info, err := svc.ExchangeSoraSessionToken(context.Background(), "st-token", nil)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, "at-token", info.AccessToken)
	require.Equal(t, "demo@example.com", info.Email)
	require.Greater(t, info.ExpiresAt, int64(0))
}

func TestOpenAIOAuthService_ExchangeSoraSessionToken_MissingAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"expires":"2099-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	origin := openAISoraSessionAuthURL
	openAISoraSessionAuthURL = server.URL
	defer func() { openAISoraSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	_, err := svc.ExchangeSoraSessionToken(context.Background(), "st-token", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing access token")
}

func TestOpenAIOAuthService_ExchangeSoraSessionToken_AcceptsSetCookieLine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.Header.Get("Cookie"), "__Secure-next-auth.session-token=st-cookie-value")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"at-token","expires":"2099-01-01T00:00:00Z","user":{"email":"demo@example.com"}}`))
	}))
	defer server.Close()

	origin := openAISoraSessionAuthURL
	openAISoraSessionAuthURL = server.URL
	defer func() { openAISoraSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	raw := "__Secure-next-auth.session-token.0=st-cookie-value; Domain=.chatgpt.com; Path=/; HttpOnly; Secure; SameSite=Lax"
	info, err := svc.ExchangeSoraSessionToken(context.Background(), raw, nil)
	require.NoError(t, err)
	require.Equal(t, "at-token", info.AccessToken)
}

func TestOpenAIOAuthService_ExchangeSoraSessionToken_MergesChunkedSetCookieLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.Header.Get("Cookie"), "__Secure-next-auth.session-token=chunk-0chunk-1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"at-token","expires":"2099-01-01T00:00:00Z","user":{"email":"demo@example.com"}}`))
	}))
	defer server.Close()

	origin := openAISoraSessionAuthURL
	openAISoraSessionAuthURL = server.URL
	defer func() { openAISoraSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	raw := strings.Join([]string{
		"Set-Cookie: __Secure-next-auth.session-token.1=chunk-1; Path=/; HttpOnly",
		"Set-Cookie: __Secure-next-auth.session-token.0=chunk-0; Path=/; HttpOnly",
	}, "\n")
	info, err := svc.ExchangeSoraSessionToken(context.Background(), raw, nil)
	require.NoError(t, err)
	require.Equal(t, "at-token", info.AccessToken)
}

func TestOpenAIOAuthService_ExchangeSoraSessionToken_PrefersLatestDuplicateChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.Header.Get("Cookie"), "__Secure-next-auth.session-token=new-0new-1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"at-token","expires":"2099-01-01T00:00:00Z","user":{"email":"demo@example.com"}}`))
	}))
	defer server.Close()

	origin := openAISoraSessionAuthURL
	openAISoraSessionAuthURL = server.URL
	defer func() { openAISoraSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	raw := strings.Join([]string{
		"Set-Cookie: __Secure-next-auth.session-token.0=old-0; Path=/; HttpOnly",
		"Set-Cookie: __Secure-next-auth.session-token.1=old-1; Path=/; HttpOnly",
		"Set-Cookie: __Secure-next-auth.session-token.0=new-0; Path=/; HttpOnly",
		"Set-Cookie: __Secure-next-auth.session-token.1=new-1; Path=/; HttpOnly",
	}, "\n")
	info, err := svc.ExchangeSoraSessionToken(context.Background(), raw, nil)
	require.NoError(t, err)
	require.Equal(t, "at-token", info.AccessToken)
}

func TestOpenAIOAuthService_ExchangeSoraSessionToken_UsesLatestCompleteChunkGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.Header.Get("Cookie"), "__Secure-next-auth.session-token=ok-0ok-1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"at-token","expires":"2099-01-01T00:00:00Z","user":{"email":"demo@example.com"}}`))
	}))
	defer server.Close()

	origin := openAISoraSessionAuthURL
	openAISoraSessionAuthURL = server.URL
	defer func() { openAISoraSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	raw := strings.Join([]string{
		"set-cookie",
		"__Secure-next-auth.session-token.0=ok-0; Domain=.chatgpt.com; Path=/",
		"set-cookie",
		"__Secure-next-auth.session-token.1=ok-1; Domain=.chatgpt.com; Path=/",
		"set-cookie",
		"__Secure-next-auth.session-token.0=partial-0; Domain=.chatgpt.com; Path=/",
	}, "\n")
	info, err := svc.ExchangeSoraSessionToken(context.Background(), raw, nil)
	require.NoError(t, err)
	require.Equal(t, "at-token", info.AccessToken)
}

func TestOpenAIOAuthService_ExchangeChatGPTSessionToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.Header.Get("Cookie"), "__Secure-next-auth.session-token=chatweb-st")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"eyJhbGciOiJub25lIn0.eyJodHRwczovL2FwaS5vcGVuYWkuY29tL2F1dGgiOnsiY2hhdGdwdF9hY2NvdW50X2lkIjoiYWNjX2NoYXR3ZWIiLCJjaGF0Z3B0X3VzZXJfaWQiOiJ1c2VyX2NoYXR3ZWIifSwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9wcm9maWxlIjp7ImVtYWlsIjoiY2hhdHdlYkBleGFtcGxlLmNvbSJ9LCJleHAiOjQxMDI0NDQ4MDB9.","expires":"2099-01-01T00:00:00Z","user":{"email":"chatweb@example.com"}}`))
	}))
	defer server.Close()

	origin := openAIChatGPTSessionAuthURL
	openAIChatGPTSessionAuthURL = server.URL
	defer func() { openAIChatGPTSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	info, err := svc.ExchangeChatGPTSessionToken(context.Background(), "chatweb-st", nil)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, "chatweb@example.com", info.Email)
	require.Equal(t, "acc_chatweb", info.ChatGPTAccountID)
	require.Equal(t, "user_chatweb", info.ChatGPTUserID)
}

func TestOpenAIOAuthService_InspectAccessToken_Success(t *testing.T) {
	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	accessToken := "eyJhbGciOiJub25lIn0." +
		"eyJjbGllbnRfaWQiOiJhcHBfY2hhdHdlYiIsImV4cCI6NDEwMjQ0NDgwMCwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9hdXRoIjp7ImNoYXRncHRfYWNjb3VudF9pZCI6ImFjY19jaGF0d2ViIiwiY2hhdGdwdF91c2VyX2lkIjoidXNlcl9jaGF0d2ViIiwiY2hhdGdwdF9wbGFuX3R5cGUiOiJwbHVzIiwib3JnYW5pemF0aW9ucyI6W3siaWQiOiJvcmdfMSIsImlzX2RlZmF1bHQiOnRydWV9XX0sImh0dHBzOi8vYXBpLm9wZW5haS5jb20vcHJvZmlsZSI6eyJlbWFpbCI6ImNoYXR3ZWJAZXhhbXBsZS5jb20ifX0."

	info, err := svc.InspectAccessToken(accessToken)
	require.NoError(t, err)
	require.Equal(t, accessToken, info.AccessToken)
	require.Equal(t, "app_chatweb", info.ClientID)
	require.Equal(t, "chatweb@example.com", info.Email)
	require.Equal(t, "acc_chatweb", info.ChatGPTAccountID)
	require.Equal(t, "user_chatweb", info.ChatGPTUserID)
	require.Equal(t, "org_1", info.OrganizationID)
	require.Equal(t, "plus", info.PlanType)
	require.Greater(t, info.ExpiresAt, int64(0))
	require.GreaterOrEqual(t, info.ExpiresIn, int64(0))
}

func TestParseOpenAIChatWebSessionInput_PreservesBrowserCookies(t *testing.T) {
	raw := strings.Join([]string{
		"Cookie: cf_clearance=cf-token-1; Path=/",
		"Cookie: _cfuvid=cfuvid-1; Path=/",
		"Cookie: oai-did=device-123; Path=/",
		"Cookie: __Secure-next-auth.session-token=session-abc; Path=/; HttpOnly",
	}, "\n")

	parsed := parseOpenAIChatWebSessionInput(raw)
	require.Equal(t, "session-abc", parsed.SessionToken)
	require.Equal(t, "device-123", parsed.DeviceID)
	require.Contains(t, parsed.CookieHeader, "cf_clearance=cf-token-1")
	require.Contains(t, parsed.CookieHeader, "_cfuvid=cfuvid-1")
	require.Contains(t, parsed.CookieHeader, "oai-did=device-123")
	require.Contains(t, parsed.CookieHeader, "__Secure-next-auth.session-token=session-abc")
}

func TestOpenAIOAuthService_ExchangeChatGPTSessionToken_CloudflareChallenge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<html><script>window._cf_chl_opt={cRay:'abc-123'};</script>Enable JavaScript and cookies to continue</html>`))
	}))
	defer server.Close()

	origin := openAIChatGPTSessionAuthURL
	openAIChatGPTSessionAuthURL = server.URL
	defer func() { openAIChatGPTSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()

	_, err := svc.ExchangeChatGPTSessionToken(context.Background(), "chatweb-st", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "OPENAI_CHATWEB_SESSION_CLOUDFLARE_CHALLENGE")
	require.Contains(t, err.Error(), "Cloudflare")
}

func TestOpenAIOAuthService_ExchangeChatGPTSessionToken_PrefersCurlCFFISidecar(t *testing.T) {
	var sidecarCalls int32
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&sidecarCalls, 1)

		var payload struct {
			Method  string            `json:"method"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, http.MethodGet, payload.Method)
		require.Equal(t, openAIChatGPTSessionAuthURL, payload.URL)
		require.Contains(t, payload.Headers["Cookie"], "__Secure-next-auth.session-token=chatweb-st")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":200,"headers":{"content-type":"application/json"},"body":"{\"accessToken\":\"eyJhbGciOiJub25lIn0.eyJodHRwczovL2FwaS5vcGVuYWkuY29tL2F1dGgiOnsiY2hhdGdwdF9hY2NvdW50X2lkIjoiYWNjX3NpZGVjYXIiLCJjaGF0Z3B0X3VzZXJfaWQiOiJ1c2VyX3NpZGVjYXIifSwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9wcm9maWxlIjp7ImVtYWlsIjoic2lkZWNhckBleGFtcGxlLmNvbSJ9LCJleHAiOjQxMDI0NDQ4MDB9.\",\"expires\":\"2099-01-01T00:00:00Z\",\"user\":{\"email\":\"sidecar@example.com\"}}"}`))
	}))
	defer sidecarServer.Close()

	client, err := curlcffi.NewClient(curlcffi.Config{
		BaseURL:             sidecarServer.URL,
		Impersonate:         "chrome131",
		TimeoutSeconds:      30,
		SessionReuseEnabled: true,
		SessionTTLSeconds:   3600,
	})
	require.NoError(t, err)

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()
	svc.SetOpenAIChatWebCurlCFFISidecarClient(client)

	info, err := svc.ExchangeChatGPTSessionToken(context.Background(), "chatweb-st", nil)
	require.NoError(t, err)
	require.Equal(t, "sidecar@example.com", info.Email)
	require.Equal(t, "acc_sidecar", info.ChatGPTAccountID)
	require.Equal(t, "user_sidecar", info.ChatGPTUserID)
	require.EqualValues(t, 1, atomic.LoadInt32(&sidecarCalls))
}

func TestOpenAIOAuthService_ExchangeChatGPTSessionToken_SidecarFailureFallbackToHTTP(t *testing.T) {
	var sidecarCalls int32
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&sidecarCalls, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"sidecar unavailable"}`))
	}))
	defer sidecarServer.Close()

	client, err := curlcffi.NewClient(curlcffi.Config{
		BaseURL:             sidecarServer.URL,
		Impersonate:         "chrome131",
		TimeoutSeconds:      30,
		SessionReuseEnabled: true,
		SessionTTLSeconds:   3600,
	})
	require.NoError(t, err)

	var upstreamCalls int32
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"eyJhbGciOiJub25lIn0.eyJodHRwczovL2FwaS5vcGVuYWkuY29tL2F1dGgiOnsiY2hhdGdwdF9hY2NvdW50X2lkIjoiYWNjX2ZhbGxiYWNrIiwiY2hhdGdwdF91c2VyX2lkIjoidXNlcl9mYWxsYmFjayJ9LCJodHRwczovL2FwaS5vcGVuYWkuY29tL3Byb2ZpbGUiOnsiZW1haWwiOiJmYWxsYmFja0BleGFtcGxlLmNvbSJ9LCJleHAiOjQxMDI0NDQ4MDB9.","expires":"2099-01-01T00:00:00Z","user":{"email":"fallback@example.com"}}`))
	}))
	defer upstreamServer.Close()

	origin := openAIChatGPTSessionAuthURL
	openAIChatGPTSessionAuthURL = upstreamServer.URL
	defer func() { openAIChatGPTSessionAuthURL = origin }()

	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientNoopStub{})
	defer svc.Stop()
	svc.SetOpenAIChatWebCurlCFFISidecarClient(client)

	info, err := svc.ExchangeChatGPTSessionToken(context.Background(), "chatweb-st", nil)
	require.NoError(t, err)
	require.Equal(t, "fallback@example.com", info.Email)
	require.Equal(t, "acc_fallback", info.ChatGPTAccountID)
	require.EqualValues(t, 1, atomic.LoadInt32(&sidecarCalls))
	require.EqualValues(t, 1, atomic.LoadInt32(&upstreamCalls))
}
