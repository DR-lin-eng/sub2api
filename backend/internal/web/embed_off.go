//go:build !embed

// Package web provides embedded web assets for the application.
package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

// PublicSettingsProvider is an interface to fetch public settings
// This stub is needed for compilation when frontend is not embedded
type PublicSettingsProvider interface {
	GetPublicSettingsForInjection(ctx context.Context) (any, error)
}

// FrontendServer is a stub for non-embed builds
type FrontendServer struct{}

func ExecutableRoutes(server *FrontendServer) []gatewayctx.RouteDef {
	if server == nil {
		return nil
	}
	return []gatewayctx.RouteDef{
		{
			Method:     http.MethodGet,
			Path:       "/",
			Handler:    server.HandleGateway,
			Middleware: []string{"request_logger", "cors", "security_headers"},
		},
		{
			Method:     http.MethodGet,
			Path:       "/*path",
			Handler:    server.HandleGateway,
			Middleware: []string{"request_logger", "cors", "security_headers"},
		},
	}
}

// NewFrontendServer returns an error when frontend is not embedded
func NewFrontendServer(settingsProvider PublicSettingsProvider) (*FrontendServer, error) {
	return nil, errors.New("frontend not embedded")
}

// InvalidateCache is a no-op for non-embed builds
func (s *FrontendServer) InvalidateCache() {}

// Middleware returns a handler that returns 404 for non-embed builds
func (s *FrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusNotFound, "Frontend not embedded. Build with -tags embed to include frontend.")
		c.Abort()
	}
}

func (s *FrontendServer) HandleGateway(c gatewayctx.GatewayContext) {
	if c == nil {
		return
	}
	c.WriteJSON(http.StatusNotFound, map[string]any{
		"error": "Frontend not embedded. Build with -tags embed to include frontend.",
	})
}

func ServeEmbeddedFrontend() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusNotFound, "Frontend not embedded. Build with -tags embed to include frontend.")
		c.Abort()
	}
}

func HasEmbeddedFrontend() bool {
	return false
}
