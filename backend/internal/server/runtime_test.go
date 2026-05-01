package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func TestBuildRouteManifestIncludesHintsAndSorts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/responses", func(c *gin.Context) {
		c.Status(http.StatusSwitchingProtocols)
	})

	manifest := BuildRouteManifest(router)
	require.Len(t, manifest, 2)
	require.Equal(t, "/health", manifest[0].Path)
	require.Equal(t, "/responses", manifest[1].Path)
	require.NotEmpty(t, manifest[0].Handler)
	require.True(t, manifest[0].Executable)
	require.Contains(t, manifest[0].Middleware, "request_logger")
	require.True(t, manifest[1].Hints.Streaming)
	require.True(t, manifest[1].Hints.WebSocket)
}

func TestProvideHTTPServerRegistersRouteManifestMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              8080,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
			RuntimeMode:       config.ServerRuntimeModeNetHTTP,
		},
	}
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	httpServer := ProvideHTTPServer(cfg, router)
	runtime := ResolveIngressRuntime(cfg, httpServer)

	require.Equal(t, config.ServerRuntimeModeNetHTTP, runtime.Name())
	require.Equal(t, cfg.Server.Address(), runtime.Addr())
	require.Len(t, runtime.RouteManifest(), 1)
	require.Equal(t, "/health", runtime.RouteManifest()[0].Path)
}

func TestResolveIngressRuntimeHonorsConfiguredMode(t *testing.T) {
	server := &http.Server{Addr: "127.0.0.1:8080"}

	cfg := &config.Config{}
	cfg.Server.RuntimeMode = config.ServerRuntimeModeHybrid
	require.Equal(t, config.ServerRuntimeModeHybrid, ResolveIngressRuntime(cfg, server).Name())

	cfg.Server.RuntimeMode = config.ServerRuntimeModeGnet
	require.Equal(t, config.ServerRuntimeModeGnet, ResolveIngressRuntime(cfg, server).Name())
}

func TestNewRustSidecarIngressRuntimeWhenEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.SocketPath = "/tmp/sub2api-rust-sidecar.sock"
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 1
	cfg.Rust.Sidecar.HealthcheckTimeoutSeconds = 1

	rt := newRustSidecarIngressRuntime(cfg, &http.Server{Addr: "127.0.0.1:8080"}, nil)
	require.NotNil(t, rt)
	require.Equal(t, "rust-sidecar-h2c", rt.Name())
	require.Equal(t, "127.0.0.1:8080", rt.Addr())
}

func TestRustSidecarIngressRuntimeHealthcheck(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "rust-sidecar.sock")
	ln, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer ln.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"test-sidecar","version":"v0"}`))
	})
	srv := &http.Server{Handler: mux}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	go func() {
		_ = srv.Serve(ln)
	}()

	cfg := &config.Config{}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.SocketPath = socketPath
	cfg.Rust.Sidecar.FailClosed = true
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 1
	cfg.Rust.Sidecar.HealthcheckTimeoutSeconds = 1

	rt := newRustSidecarIngressRuntime(cfg, &http.Server{Addr: "127.0.0.1:8080"}, nil)
	require.NotNil(t, rt)
	require.NoError(t, rt.healthcheck())
}

func TestRustSidecarIngressRuntimeProxiesRawConnectionToUnixSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "rust-sidecar.sock")
	ln, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer ln.Close()

	sidecarDone := make(chan struct{})
	go func() {
		defer close(sidecarDone)
		healthBody := `{"status":"ok","service":"test-sidecar","version":"v0"}`
		for i := 0; i < 2; i++ {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				peek, err := reader.Peek(4)
				if err == nil && string(peek) == "GET " {
					for {
						line, readErr := reader.ReadString('\n')
						if readErr != nil || line == "\r\n" {
							break
						}
					}
					_, _ = c.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(healthBody), healthBody)))
					return
				}
				buf := make([]byte, 4)
				_, _ = io.ReadFull(reader, buf)
				_, _ = c.Write([]byte("pong"))
			}(conn)
		}
	}()

	cfg := &config.Config{}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.SocketPath = socketPath
	cfg.Rust.Sidecar.FailClosed = true
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 1
	cfg.Rust.Sidecar.HealthcheckTimeoutSeconds = 1

	listener := newClassifiedListener(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	rt := newRustSidecarIngressRuntime(cfg, &http.Server{Addr: "127.0.0.1:8080"}, nil)
	require.NotNil(t, rt)

	errCh := make(chan error, 1)
	go func() {
		errCh <- rt.Serve(listener)
	}()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	require.True(t, listener.enqueue(serverConn))

	_, err = clientConn.Write([]byte("ping"))
	require.NoError(t, err)
	buf := make([]byte, 4)
	_, err = io.ReadFull(clientConn, buf)
	require.NoError(t, err)
	require.Equal(t, "pong", string(buf))

	require.NoError(t, listener.Close())
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("rust sidecar runtime did not exit in time")
	}

	select {
	case <-sidecarDone:
	case <-time.After(2 * time.Second):
		t.Fatal("sidecar stub did not exit in time")
	}
}

func TestClassifyPreface(t *testing.T) {
	require.Equal(t, protocolTargetH2C, classifyPreface([]byte(http2.ClientPreface)))
	require.Equal(t, protocolTargetH2C, classifyPreface([]byte("GET / HTTP/1.1\r\nHost: example.com\r\nUpgrade: h2c\r\nHTTP2-Settings: AAMAAABkAAQCAAAAAAIAAAAA\r\n\r\n")))
	require.Equal(t, protocolTargetHTTP1, classifyPreface([]byte("GET /health HTTP/1.1\r\nHost: example.com\r\n\r\n")))
}

func TestClassifyPrefaceRoutesResponsesWebSocketToSidecarWhenEnabled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	mux := newProtocolMux(ln, 0, true, false, true)
	target := classifyPrefaceWithOptions([]byte("GET /v1/responses HTTP/1.1\r\nHost: example.com\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"), mux)
	require.Equal(t, protocolTargetSidecar, target)
}

func TestClassifyPrefaceDoesNotRouteResponsesWebSocketToSidecarWhenDisabled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	mux := newProtocolMux(ln, 0, false, false, false)
	target := classifyPrefaceWithOptions([]byte("GET /v1/responses HTTP/1.1\r\nHost: example.com\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"), mux)
	require.Equal(t, protocolTargetHTTP1, target)
}

func TestClassifyPrefaceMatchesMixedCaseHeadersWithoutStringLowerAlloc(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	mux := newProtocolMux(ln, 0, true, false, true)
	target := classifyPrefaceWithOptions([]byte("GeT /V1/Responses HTTP/1.1\r\nHost: example.com\r\nUpGrAdE: WebSocket\r\nConnection: Upgrade\r\n\r\n"), mux)
	require.Equal(t, protocolTargetSidecar, target)
}

func TestHasHTTPHeaderTerminatorIncrementalDetectsBoundaryAcrossReadChunks(t *testing.T) {
	buf := []byte("GET /health HTTP/1.1\r\nHost: example.com\r\n")
	require.False(t, hasHTTPHeaderTerminatorIncremental(buf, 0))

	buf = append(buf, '\r', '\n')
	require.True(t, hasHTTPHeaderTerminatorIncremental(buf, len(buf)-2))
}

func TestClassifiedListenerBuffersBurstWithoutConcurrentAccept(t *testing.T) {
	listener := newClassifiedListener(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	require.True(t, listener.enqueue(serverConn))
}

func TestHybridRuntimeRoutesResponsesWebSocketUpgradeToSidecar(t *testing.T) {
	sidecarSocket := filepath.Join(t.TempDir(), "rust-sidecar.sock")
	sidecarListener, err := net.Listen("unix", sidecarSocket)
	require.NoError(t, err)
	defer sidecarListener.Close()

	receivedReqCh := make(chan string, 1)
	sidecarDone := make(chan struct{})
	go func() {
		defer close(sidecarDone)
		healthBody := `{"status":"ok","service":"test-sidecar","version":"v0"}`
		sidecarBody := `{"code":"SIDE_CAR","message":"routed to sidecar"}`
		for i := 0; i < 2; i++ {
			conn, err := sidecarListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				line, readErr := reader.ReadString('\n')
				if readErr != nil {
					return
				}
				if line == "GET /healthz HTTP/1.1\r\n" {
					for {
						headerLine, err := reader.ReadString('\n')
						if err != nil || headerLine == "\r\n" {
							break
						}
					}
					_, _ = c.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(healthBody), healthBody)))
					return
				}
				receivedReqCh <- line
				for {
					headerLine, err := reader.ReadString('\n')
					if err != nil || headerLine == "\r\n" {
						break
					}
				}
				_, _ = c.Write([]byte(fmt.Sprintf("HTTP/1.1 401 Unauthorized\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(sidecarBody), sidecarBody)))
			}(conn)
		}
	}()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
			RuntimeMode:       config.ServerRuntimeModeHybrid,
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.FailClosed = true
	cfg.Rust.Sidecar.SocketPath = sidecarSocket
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 1
	cfg.Rust.Sidecar.HealthcheckTimeoutSeconds = 1
	cfg.Rust.Sidecar.ResponsesWSEnabled = true

	base := &http.Server{Addr: "127.0.0.1:0", Handler: http.NewServeMux()}
	rt := ResolveIngressRuntime(cfg, base)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- rt.Serve(listener) }()

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
	require.JSONEq(t, `{"code":"SIDE_CAR","message":"routed to sidecar"}`, string(body))

	select {
	case line := <-receivedReqCh:
		require.Equal(t, "GET /v1/responses HTTP/1.1\r\n", line)
	case <-time.After(2 * time.Second):
		t.Fatal("sidecar did not receive websocket request")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, rt.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("hybrid runtime did not exit in time")
	}

	select {
	case <-sidecarDone:
	case <-time.After(2 * time.Second):
		t.Fatal("sidecar stub did not exit in time")
	}
}

func TestHybridRuntimeRoutesH2CToSidecar(t *testing.T) {
	sidecarSocket := filepath.Join(t.TempDir(), "rust-sidecar-h2c.sock")
	sidecarListener, err := net.Listen("unix", sidecarSocket)
	require.NoError(t, err)
	defer sidecarListener.Close()

	sidecarMux := http.NewServeMux()
	sidecarMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"test-sidecar","version":"v0"}`))
	})
	sidecarMux.HandleFunc("/h2c-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Sidecar", "h2c")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("routed-via-sidecar"))
	})

	sidecarServer := &http.Server{
		Handler: h2c.NewHandler(sidecarMux, &http2.Server{}),
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = sidecarServer.Shutdown(ctx)
	}()
	go func() {
		_ = sidecarServer.Serve(sidecarListener)
	}()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              0,
			ReadHeaderTimeout: 5,
			IdleTimeout:       10,
			RuntimeMode:       config.ServerRuntimeModeHybrid,
			H2C: config.H2CConfig{
				Enabled:                      true,
				MaxConcurrentStreams:         16,
				IdleTimeout:                  10,
				MaxReadFrameSize:             1 << 20,
				MaxUploadBufferPerConnection: 2 << 20,
				MaxUploadBufferPerStream:     512 << 10,
			},
		},
		Security: config.SecurityConfig{
			CSP: config.CSPConfig{
				Enabled: false,
				Policy:  config.DefaultCSPPolicy,
			},
		},
	}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.FailClosed = true
	cfg.Rust.Sidecar.SocketPath = sidecarSocket
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 1
	cfg.Rust.Sidecar.HealthcheckTimeoutSeconds = 1
	cfg.Rust.Sidecar.H2CDelegateEnabled = true

	base := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	rt := ResolveIngressRuntime(cfg, base)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- rt.Serve(listener) }()

	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, listener.Addr().String())
			},
		},
	}
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/h2c-check", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	require.Equal(t, 2, resp.ProtoMajor)
	require.Equal(t, "h2c", resp.Header.Get("X-Sidecar"))
	require.Equal(t, "routed-via-sidecar", string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, rt.Shutdown(ctx))
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("hybrid runtime did not exit in time")
	}
}
