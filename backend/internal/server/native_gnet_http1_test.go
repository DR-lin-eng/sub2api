package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	admin "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/web"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveIngressRuntimeUsesNativeGnetForHTTP1(t *testing.T) {
	base := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: http.NewServeMux(),
	}

	cfg := &config.Config{}
	cfg.Server.RuntimeMode = config.ServerRuntimeModeGnet
	rt := ResolveIngressRuntime(cfg, base)

	split, ok := rt.(*hybridRuntime)
	require.True(t, ok)
	require.Equal(t, config.ServerRuntimeModeGnet, split.Name())
	_, isNative := split.http1Runtime.(*nativeGnetHTTPRuntime)
	require.True(t, isNative)
}

func TestDecodeBufferedRequestHandlesPipelinedRequests(t *testing.T) {
	raw := []byte(
		"POST /health HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\n\r\nhello" +
			"GET /next HTTP/1.1\r\nHost: example.com\r\n\r\n",
	)

	req, consumed, complete, err := decodeBufferedRequest(raw, nil, nil)
	require.NoError(t, err)
	require.True(t, complete)
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "/health", req.URL.Path)
	expectedRemaining := "GET /next HTTP/1.1\r\nHost: example.com\r\n\r\n"
	require.Equal(t, len(raw)-len(expectedRemaining), consumed)
	require.Equal(t, expectedRemaining, string(raw[consumed:]))
}

func TestDecodeBufferedRequestReturnsIncompleteForPartialBody(t *testing.T) {
	raw := []byte("POST /health HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\n\r\nhel")

	req, consumed, complete, err := decodeBufferedRequest(raw, nil, nil)
	require.NoError(t, err)
	require.False(t, complete)
	require.Nil(t, req)
	require.Zero(t, consumed)
}

func TestDecodeBufferedRequestFastPathsBodylessRequests(t *testing.T) {
	raw := []byte("GET /health HTTP/1.1\r\nHost: example.com\r\n\r\n")

	req, consumed, complete, err := decodeBufferedRequest(raw, nil, nil)
	require.NoError(t, err)
	require.True(t, complete)
	require.NotNil(t, req)
	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, int64(0), req.ContentLength)
	require.Same(t, http.NoBody, req.Body)
	require.Equal(t, len(raw), consumed)
}

func TestNativeGnetHTTPRuntimeServesHTTP1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
	}
	httpServer := NewHTTPServer(cfg, router)
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + listener.Addr().String() + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, resp.Header.Get("Transfer-Encoding"))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, len(body), mustAtoi(t, resp.Header.Get("Content-Length")))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableHealthRouteWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + listener.Addr().String() + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "15", resp.Header.Get("Content-Length"))
	require.Empty(t, resp.Header.Get("Transfer-Encoding"))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"ok"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	n, err := strconv.Atoi(value)
	require.NoError(t, err)
	return n
}

func TestNativeGnetHTTPRuntimeServesExecutableClaudeBootstrapAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + listener.Addr().String() + "/api/claude_cli/bootstrap")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableClaudeMetricsEnabledAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + listener.Addr().String() + "/api/claude_code/organizations/metrics_enabled")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableClaudePolicyLimitsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + listener.Addr().String() + "/api/claude_code/policy_limits")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableClaudeUserSettingsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPut, "http://"+listener.Addr().String()+"/api/claude_code/user_settings", strings.NewReader(`{"entries":{"~/.claude/settings.json":"{}"}}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableChatCompletionsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-5.1","stream":false}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableResponsesAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/v1/responses", strings.NewReader(`{"model":"gpt-5.1","stream":false}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableResponsesWebSocketAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/v1/responses", nil)
	require.NoError(t, err)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGVzdC13cy1rZXk=")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

type nativeRouteAPIKeyRepoStub struct {
	getByKeyForAuth func(ctx context.Context, key string) (*service.APIKey, error)
}

func (s *nativeRouteAPIKeyRepoStub) Create(context.Context, *service.APIKey) error {
	panic("unexpected Create call")
}
func (s *nativeRouteAPIKeyRepoStub) GetByID(context.Context, int64) (*service.APIKey, error) {
	panic("unexpected GetByID call")
}
func (s *nativeRouteAPIKeyRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	panic("unexpected GetKeyAndOwnerID call")
}
func (s *nativeRouteAPIKeyRepoStub) GetByKey(context.Context, string) (*service.APIKey, error) {
	panic("unexpected GetByKey call")
}
func (s *nativeRouteAPIKeyRepoStub) GetByKeyForAuth(ctx context.Context, key string) (*service.APIKey, error) {
	if s.getByKeyForAuth == nil {
		panic("unexpected GetByKeyForAuth call")
	}
	return s.getByKeyForAuth(ctx, key)
}
func (s *nativeRouteAPIKeyRepoStub) Update(context.Context, *service.APIKey) error {
	panic("unexpected Update call")
}
func (s *nativeRouteAPIKeyRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}
func (s *nativeRouteAPIKeyRepoStub) ListByUserID(context.Context, int64, pagination.PaginationParams, service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserID call")
}
func (s *nativeRouteAPIKeyRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	panic("unexpected VerifyOwnership call")
}
func (s *nativeRouteAPIKeyRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	panic("unexpected CountByUserID call")
}
func (s *nativeRouteAPIKeyRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	panic("unexpected ExistsByKey call")
}
func (s *nativeRouteAPIKeyRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]service.APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}
func (s *nativeRouteAPIKeyRepoStub) SearchAPIKeys(context.Context, int64, string, int) ([]service.APIKey, error) {
	panic("unexpected SearchAPIKeys call")
}
func (s *nativeRouteAPIKeyRepoStub) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected ClearGroupIDByGroupID call")
}
func (s *nativeRouteAPIKeyRepoStub) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	panic("unexpected UpdateGroupIDByUserAndGroup call")
}
func (s *nativeRouteAPIKeyRepoStub) CountByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected CountByGroupID call")
}
func (s *nativeRouteAPIKeyRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByUserID call")
}
func (s *nativeRouteAPIKeyRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByGroupID call")
}
func (s *nativeRouteAPIKeyRepoStub) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	panic("unexpected IncrementQuotaUsed call")
}
func (s *nativeRouteAPIKeyRepoStub) UpdateLastUsed(context.Context, int64, time.Time) error {
	return nil
}
func (s *nativeRouteAPIKeyRepoStub) IncrementRateLimitUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementRateLimitUsage call")
}
func (s *nativeRouteAPIKeyRepoStub) ResetRateLimitWindows(context.Context, int64) error {
	panic("unexpected ResetRateLimitWindows call")
}
func (s *nativeRouteAPIKeyRepoStub) GetRateLimitData(context.Context, int64) (*service.APIKeyRateLimitData, error) {
	panic("unexpected GetRateLimitData call")
}

func TestNativeGnetHTTPRuntimeServesExecutableMessagesOpenAIDispatchWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	groupID := int64(42)
	apiKeyRepo := &nativeRouteAPIKeyRepoStub{
		getByKeyForAuth: func(ctx context.Context, key string) (*service.APIKey, error) {
			return &service.APIKey{
				ID:      101,
				Key:     key,
				Status:  service.StatusActive,
				GroupID: &groupID,
				Group: &service.Group{
					ID:                    groupID,
					Platform:              service.PlatformOpenAI,
					AllowMessagesDispatch: true,
				},
				User: &service.User{
					ID:          7,
					Status:      service.StatusActive,
					Role:        service.RoleUser,
					Concurrency: 2,
					Balance:     10,
				},
			}, nil
		},
	}
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, cfg)

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		apiKeyService,
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/v1/messages", strings.NewReader(`{"model":"gpt-5.1","stream":false}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"error":{"type":"api_error","message":"Service temporarily unavailable"}}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableCountTokensOpenAI404WithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	groupID := int64(43)
	apiKeyRepo := &nativeRouteAPIKeyRepoStub{
		getByKeyForAuth: func(ctx context.Context, key string) (*service.APIKey, error) {
			return &service.APIKey{
				ID:      102,
				Key:     key,
				Status:  service.StatusActive,
				GroupID: &groupID,
				Group: &service.Group{
					ID:                    groupID,
					Platform:              service.PlatformOpenAI,
					AllowMessagesDispatch: true,
				},
				User: &service.User{
					ID:          8,
					Status:      service.StatusActive,
					Role:        service.RoleUser,
					Concurrency: 2,
					Balance:     10,
				},
			}, nil
		},
	}
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, cfg)

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		apiKeyService,
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/v1/messages/count_tokens", strings.NewReader(`{"model":"gpt-5.1","messages":[]}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"error","error":{"type":"not_found_error","message":"Token counting is not supported for this platform"}}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableMessagesAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/v1/messages", strings.NewReader(`{"model":"claude-3.7-sonnet","messages":[],"stream":false}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableMessagesCountTokensAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		&service.APIKeyService{},
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/v1/messages/count_tokens", strings.NewReader(`{"model":"claude-3.7-sonnet","messages":[]}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"API_KEY_REQUIRED","message":"API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminTLSSettingsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{
				Setting: &admin.SettingHandler{},
			},
		},
		nil,
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/settings/tls-fingerprint", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminSettingsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{
				Setting: &admin.SettingHandler{},
			},
		},
		nil,
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/settings", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminSoraS3SettingsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{
				Setting: &admin.SettingHandler{},
			},
		},
		nil,
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/settings/sora-s3", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminImportTaskAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{
				Account: &admin.AccountHandler{},
			},
		},
		nil,
		nil,
		&service.SettingService{},
		nil,
		nil,
		nil,
		nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Serve(listener)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/admin/accounts/data/tasks", strings.NewReader(`{"data":{"type":"sub2api-data","version":1}}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminDashboardStatsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{Dashboard: &admin.DashboardHandler{}},
		},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/dashboard/stats", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminGroupsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{Group: &admin.GroupHandler{}},
		},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/groups", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminGroupCreateAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{Group: &admin.GroupHandler{}},
		},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/admin/groups", strings.NewReader(`{"name":"demo"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminUserCreateAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{User: &admin.UserHandler{}},
		},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/admin/users", strings.NewReader(`{"email":"a@example.com","password":"secret123"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminSubscriptionsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{Subscription: &admin.SubscriptionHandler{}},
		},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/subscriptions", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminUsageAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{
			Admin: &handler.AdminHandlers{Usage: &admin.UsageHandler{}},
		},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/usage", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminUsageCleanupTasksAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Usage: &admin.UsageHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/usage/cleanup-tasks", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminAPIKeysAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{APIKey: &admin.AdminAPIKeyHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPut, "http://"+listener.Addr().String()+"/api/v1/admin/api-keys/1", strings.NewReader(`{"group_id":1}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminErrorPassthroughAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{ErrorPassthrough: &admin.ErrorPassthroughHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/error-passthrough-rules", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminProxyMaintenanceAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{ProxyMaintenance: &admin.ProxyMaintenanceHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/proxy-maintenance-plans", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminAnnouncementsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Announcement: &admin.AnnouncementHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/announcements", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminPromoCodesAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Promo: &admin.PromoHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/promo-codes", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminUserAttributesAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{UserAttribute: &admin.UserAttributeHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/user-attributes", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminRedeemCodesAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Redeem: &admin.RedeemHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/redeem-codes", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminBackupsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Backup: &admin.BackupHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/backups", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminScheduledTestsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{ScheduledTest: &admin.ScheduledTestHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/admin/scheduled-test-plans", strings.NewReader(`{"account_id":1,"cron_expression":"0 * * * *"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminProxiesAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Proxy: &admin.ProxyHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/proxies", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminOpenAIOAuthAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{OpenAIOAuth: &admin.OpenAIOAuthHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/admin/openai/generate-auth-url", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminGeminiOAuthAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{GeminiOAuth: &admin.GeminiOAuthHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/gemini/oauth/capabilities", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminAntigravityOAuthAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{AntigravityOAuth: &admin.AntigravityOAuthHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/admin/antigravity/oauth/auth-url", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminDataManagementAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{DataManagement: &admin.DataManagementHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/data-management/agent/health", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminOpsAlertRulesAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Ops: &admin.OpsHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/ops/alert-rules", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminOpsRuntimeLoggingAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Ops: &admin.OpsHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/ops/runtime/logging", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminOpsErrorsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Ops: &admin.OpsHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/ops/errors", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminOpsQPSWSAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}
	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Ops: &admin.OpsHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/ops/ws/qps", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminUserAPIKeysAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{User: &admin.UserHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/users/1/api-keys", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminGroupAPIKeysAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Group: &admin.GroupHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/groups/1/api-keys", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutablePublicAccountExportDownloadWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Account: &admin.AccountHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/public/account-export-tasks/task-1/download?token=x", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":404,"message":"Export artifact not found"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAuthMeUnauthorizedWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Auth: &handler.AuthHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{},
		nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/auth/me", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization header is required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableLinuxDoOAuthStartWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Auth: &handler.AuthHandler{}},
		nil, nil, nil, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/auth/oauth/linuxdo/start?redirect=%2Fdashboard", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), `"reason":"CONFIG_NOT_READY"`)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableUserProfileUnauthorizedWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{User: &handler.UserHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{},
		nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/user/profile", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization header is required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAuthLoginUnauthorizedRateLimitEnvelopeWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Auth: &handler.AuthHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{}, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/auth/login", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableKeysUnauthorizedWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{APIKey: &handler.APIKeyHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{}, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/keys", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization header is required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableRedeemUnauthorizedWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Redeem: &handler.RedeemHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{}, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/redeem", strings.NewReader(`{"code":"demo"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization header is required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableSubscriptionsUnauthorizedWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Subscription: &handler.SubscriptionHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{}, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/subscriptions", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization header is required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableSoraGenerateUnauthorizedWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{SoraClient: &handler.SoraClientHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{}, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/sora/generate", strings.NewReader(`{"model":"sora-2","prompt":"test"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization header is required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableUsageStatsUnauthorizedWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Usage: &handler.UsageHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{}, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/usage/stats", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization header is required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableTotpStatusUnauthorizedWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Totp: &handler.TotpHandler{}},
		nil, nil, &service.SettingService{}, &service.AuthService{}, &service.UserService{}, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/user/totp/status", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization header is required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminOpsConcurrencyAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Ops: &admin.OpsHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/ops/concurrency", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminSystemVersionAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{System: &admin.SystemHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/system/version", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableFrontendCatchAllWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{},
		nil, nil, nil, nil, nil, nil, &web.FrontendServer{},
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/dashboard", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"error":"Frontend not embedded. Build with -tags embed to include frontend."}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminAccountsAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Account: &admin.AccountHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/v1/admin/accounts", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminAccountTestAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{Account: &admin.AccountHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/admin/accounts/1/test", strings.NewReader(`{"model_id":"claude-sonnet-4-5"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}

func TestNativeGnetHTTPRuntimeServesExecutableAdminAccountsOAuthAuthFailureWithoutFallbackHandler(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Host: "127.0.0.1", Port: 0, ReadHeaderTimeout: 5, IdleTimeout: 30},
		Security: config.SecurityConfig{CSP: config.CSPConfig{Enabled: false, Policy: config.DefaultCSPPolicy}},
	}

	httpServer := NewHTTPServer(cfg, http.NewServeMux())
	registerHTTPServerExecutableRuntimeConfig(httpServer, buildExecutableRuntimeConfig(
		cfg,
		&handler.Handlers{Admin: &handler.AdminHandlers{OAuth: &admin.OAuthHandler{}}},
		nil, nil, &service.SettingService{}, nil, nil, nil, nil,
	))
	runtime := newNativeGnetHTTPRuntime(cfg, httpServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- runtime.Serve(listener) }()

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/api/v1/admin/accounts/generate-auth-url", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"code":"UNAUTHORIZED","message":"Authorization required"}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, runtime.Shutdown(ctx))

	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("native gnet runtime did not exit in time")
	}
}
