package server

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

func TestNewHTTPServer_UsesConfiguredAddressAndTimeouts(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              19090,
			ReadHeaderTimeout: 11,
			IdleTimeout:       37,
		},
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := NewHTTPServer(cfg, baseHandler)

	require.Equal(t, "127.0.0.1:19090", srv.Addr)
	require.Equal(t, 11*time.Second, srv.ReadHeaderTimeout)
	require.Equal(t, 37*time.Second, srv.IdleTimeout)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/health", nil)
	srv.Handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestBuildHTTPHandler_EnforcesServerMaxRequestBodySize(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxRequestBodySize: 4,
		},
		Gateway: config.GatewayConfig{
			MaxBodySize: 64,
		},
	}

	handler := BuildHTTPHandler(cfg, bodyEchoHandler())

	resp := performHTTPTestRequest(t, handler, "POST", "http://example.com/upload", "12345")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestBuildHTTPHandler_FallsBackToGatewayMaxBodySize(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxRequestBodySize: 0,
		},
		Gateway: config.GatewayConfig{
			MaxBodySize: 3,
		},
	}

	handler := BuildHTTPHandler(cfg, bodyEchoHandler())

	resp := performHTTPTestRequest(t, handler, "POST", "http://example.com/upload", "1234")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestBuildHTTPHandler_H2CEnabledSupportsHTTP2Cleartext(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       10,
			H2C: config.H2CConfig{
				Enabled:                      true,
				MaxConcurrentStreams:         16,
				IdleTimeout:                  10,
				MaxReadFrameSize:             1 << 20,
				MaxUploadBufferPerConnection: 2 << 20,
				MaxUploadBufferPerStream:     512 << 10,
			},
		},
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Transport", "h2c")
		w.WriteHeader(http.StatusAccepted)
	})

	httpHandler := BuildHTTPHandler(cfg, baseHandler)
	srv := NewHTTPServer(cfg, httpHandler)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve(ln)
	}()

	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, ln.Addr().String())
			},
		},
	}

	req, err := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String()+"/transport", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	require.Equal(t, 2, resp.ProtoMajor)
	require.Equal(t, "h2c", resp.Header.Get("X-Transport"))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, srv.Shutdown(shutdownCtx))
	require.Eventually(t, func() bool {
		select {
		case err := <-serveErrCh:
			return err == nil || err == http.ErrServerClosed
		default:
			return false
		}
	}, 5*time.Second, 20*time.Millisecond)
}

func bodyEchoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

func performHTTPTestRequest(t *testing.T, handler http.Handler, method, targetURL, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, targetURL, strings.NewReader(body))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Result()
}
