package server

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
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

func TestClassifyPreface(t *testing.T) {
	require.Equal(t, protocolTargetH2C, classifyPreface([]byte(http2.ClientPreface)))
	require.Equal(t, protocolTargetH2C, classifyPreface([]byte("GET / HTTP/1.1\r\nHost: example.com\r\nUpgrade: h2c\r\nHTTP2-Settings: AAMAAABkAAQCAAAAAAIAAAAA\r\n\r\n")))
	require.Equal(t, protocolTargetHTTP1, classifyPreface([]byte("GET /health HTTP/1.1\r\nHost: example.com\r\n\r\n")))
}
