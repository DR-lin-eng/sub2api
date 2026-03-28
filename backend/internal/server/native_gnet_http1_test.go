package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
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
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"error","error":{"type":"api_error","message":"User context not found"}}`, string(body))

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
