package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// adaptLegacyGinRoute bridges a Gin-only handler into the executable/gnet path.
// It keeps new upstream features reachable through Rust sidecar ingress even
// before every endpoint is fully ported to a native GatewayContext handler.
func adaptLegacyGinRoute(path string, fn gin.HandlerFunc) gatewayctx.HandlerFunc {
	paramNames := extractRouteParamNames(path)
	return func(c gatewayctx.GatewayContext) {
		runLegacyGinRoute(c, fn, paramNames)
	}
}

func withQueryValue(key, value string, fn gatewayctx.HandlerFunc) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		if c == nil || fn == nil || c.Request() == nil || c.Request().URL == nil {
			return
		}
		original := c.Request()
		cloned := original.Clone(original.Context())
		cloned.URL = cloneURL(original.URL)
		query := cloned.URL.Query()
		query.Set(key, value)
		cloned.URL.RawQuery = query.Encode()
		c.SetRequest(cloned)
		defer c.SetRequest(original)
		fn(c)
	}
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	cloned := *u
	return &cloned
}

func runLegacyGinRoute(c gatewayctx.GatewayContext, fn gin.HandlerFunc, paramNames []string) {
	if c == nil || fn == nil {
		return
	}

	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	if req := c.Request(); req != nil {
		ginCtx.Request = req
	}
	if len(paramNames) > 0 {
		params := make(gin.Params, 0, len(paramNames))
		for _, name := range paramNames {
			params = append(params, gin.Param{Key: name, Value: c.PathParam(name)})
		}
		ginCtx.Params = params
	}
	copyGatewayValuesToGin(ginCtx, c)

	fn(ginCtx)

	headers := c.Header()
	for key, values := range recorder.Header() {
		if strings.EqualFold(key, "Set-Cookie") {
			for _, value := range values {
				headers.Add(key, value)
			}
			continue
		}
		headers.Del(key)
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	status := recorder.Code
	if status == 0 {
		status = http.StatusOK
	}
	body := recorder.Body.Bytes()
	if len(body) == 0 {
		c.SetStatus(status)
		return
	}
	_, _ = c.WriteBytes(status, body)
}

func copyGatewayValuesToGin(dst *gin.Context, src gatewayctx.GatewayContext) {
	if dst == nil || src == nil {
		return
	}

	for _, key := range []string{
		string(servermiddleware.ContextKeyUser),
		string(servermiddleware.ContextKeyUserRole),
		string(servermiddleware.ContextKeyAPIKey),
		string(servermiddleware.ContextKeySubscription),
		string(servermiddleware.ContextKeyForcePlatform),
		"auth_method",
	} {
		if value, ok := src.Value(key); ok {
			dst.Set(key, value)
		}
	}
}

func extractRouteParamNames(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, ":"):
			names = append(names, strings.TrimPrefix(part, ":"))
		case strings.HasPrefix(part, "*"):
			names = append(names, strings.TrimPrefix(part, "*"))
		}
	}
	return names
}
