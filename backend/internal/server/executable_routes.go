package server

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	sermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/routes"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/setup"
)

const (
	middlewareTagRequestLogger    = "request_logger"
	middlewareTagClientReqID      = "client_request_id"
	middlewareTagSecurity         = "security_headers"
	middlewareTagCORS             = "cors"
	middlewareTagSetupGuard       = "setup_guard"
	middlewareTagGoogleAPIKey     = "google_api_key_auth"
	middlewareTagStandardAPIKey   = "standard_api_key_auth"
	middlewareTagRequireGoogle    = "require_group_google"
	middlewareTagRequireAnthropic = "require_group_anthropic"
	middlewareTagInboundEP        = "inbound_endpoint"
	middlewareTagForceAG          = "force_platform_antigravity"
	middlewareTagBodyLimitGW      = "gateway_body_limit"
	middlewareTagMessageDispatch  = "message_dispatch"
)

const nativeRouteFallbackToHTTPHandlerKey = "_native_route_fallback_http_handler"

type executableRoute struct {
	method     string
	path       string
	handler    gatewayctx.HandlerFunc
	middleware []string
}

type executableRuntimeConfig struct {
	cfg                 *config.Config
	routes              []executableRoute
	apiKeyService       *service.APIKeyService
	subscriptionService *service.SubscriptionService
	settingService      *service.SettingService
}

func buildExecutableRuntimeConfig(
	cfg *config.Config,
	handlers *handler.Handlers,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	settingService *service.SettingService,
) *executableRuntimeConfig {
	rawDefs := make([]gatewayctx.RouteDef, 0, 8)
	rawDefs = append(rawDefs, routes.ExecutableCommonRoutes()...)
	rawDefs = append(rawDefs, setup.ExecutableRoutes()...)
	rawDefs = append(rawDefs, routes.ExecutableGatewayRoutes(handlers)...)

	out := make([]executableRoute, 0, len(rawDefs))
	for _, def := range rawDefs {
		out = append(out, executableRoute{
			method:     strings.ToUpper(strings.TrimSpace(def.Method)),
			path:       strings.TrimSpace(def.Path),
			handler:    def.Handler,
			middleware: append([]string(nil), def.Middleware...),
		})
	}
	return &executableRuntimeConfig{
		cfg:                 cfg,
		routes:              out,
		apiKeyService:       apiKeyService,
		subscriptionService: subscriptionService,
		settingService:      settingService,
	}
}

func matchExecutableRoute(defs []executableRoute, method, path string) (executableRoute, map[string]string, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	for _, def := range defs {
		if def.method != method {
			continue
		}
		if params, ok := matchRoutePath(def.path, path); ok {
			return def, params, true
		}
	}
	return executableRoute{}, nil, false
}

func dispatchExecutableRoute(runtimeCfg *executableRuntimeConfig, req *http.Request, writer any, clientIP string) bool {
	if req == nil {
		return false
	}
	if runtimeCfg == nil {
		return false
	}
	route, params, ok := matchExecutableRoute(runtimeCfg.routes, req.Method, req.URL.Path)
	if !ok || route.handler == nil {
		return false
	}

	ctx := gatewayctx.NewNative(req, writer, params, clientIP)
	if !applyExecutableMiddlewares(runtimeCfg, ctx, route.middleware) {
		return true
	}
	route.handler(ctx)
	if shouldFallbackToHTTPHandler(ctx) {
		return false
	}
	return true
}

func shouldFallbackToHTTPHandler(c gatewayctx.GatewayContext) bool {
	if c == nil || c.ResponseWritten() {
		return false
	}
	value, ok := c.Value(nativeRouteFallbackToHTTPHandlerKey)
	if !ok {
		return false
	}
	fallback, ok := value.(bool)
	return ok && fallback
}

func applyExecutableMiddlewares(runtimeCfg *executableRuntimeConfig, c gatewayctx.GatewayContext, tags []string) bool {
	if c == nil {
		return false
	}
	var corsCfg sermiddleware.CORSRuntimeConfig
	var cspCfg config.CSPConfig
	if runtimeCfg != nil && runtimeCfg.cfg != nil {
		corsCfg = sermiddleware.PrepareCORSRuntimeConfig(runtimeCfg.cfg.CORS)
		cspCfg = runtimeCfg.cfg.Security.CSP
	} else {
		corsCfg = sermiddleware.PrepareCORSRuntimeConfig(config.CORSConfig{})
		cspCfg = config.CSPConfig{Enabled: true, Policy: config.DefaultCSPPolicy}
	}

	for _, tag := range tags {
		switch tag {
		case middlewareTagRequestLogger:
			sermiddleware.ApplyRequestLoggerContext(c)
		case middlewareTagClientReqID:
			sermiddleware.ApplyClientRequestIDContext(c)
		case middlewareTagCORS:
			if !sermiddleware.ApplyCORSContext(c, corsCfg) {
				return false
			}
		case middlewareTagSecurity:
			sermiddleware.ApplySecurityHeadersContext(c, cspCfg, "", nil)
		case middlewareTagSetupGuard:
			if !setup.SetupGuardContext(c) {
				return false
			}
		case middlewareTagInboundEP:
			handler.ApplyInboundEndpointContext(c)
		case middlewareTagGoogleAPIKey:
			if runtimeCfg == nil || !sermiddleware.ApplyAPIKeyAuthWithSubscriptionGoogleContext(runtimeCfg.apiKeyService, runtimeCfg.subscriptionService, runtimeCfg.cfg, c) {
				return false
			}
		case middlewareTagStandardAPIKey:
			if runtimeCfg == nil || !sermiddleware.ApplyAPIKeyAuthWithSubscriptionContext(runtimeCfg.apiKeyService, runtimeCfg.subscriptionService, runtimeCfg.cfg, c) {
				return false
			}
		case middlewareTagRequireGoogle:
			if runtimeCfg == nil || !sermiddleware.RequireGroupAssignmentContext(runtimeCfg.settingService, sermiddleware.GoogleErrorWriterContext, c) {
				return false
			}
		case middlewareTagRequireAnthropic:
			if runtimeCfg == nil || !sermiddleware.RequireGroupAssignmentContext(runtimeCfg.settingService, sermiddleware.AnthropicErrorWriterContext, c) {
				return false
			}
		case middlewareTagForceAG:
			sermiddleware.SetForcePlatformContext(c, service.PlatformAntigravity)
		case middlewareTagBodyLimitGW:
			if runtimeCfg != nil && runtimeCfg.cfg != nil {
				sermiddleware.ApplyRequestBodyLimitContext(c, runtimeCfg.cfg.Gateway.MaxBodySize)
			}
		case middlewareTagMessageDispatch:
			// No-op here; route handler performs platform-aware dispatch.
		}
	}
	return true
}

func matchRoutePath(pattern, path string) (map[string]string, bool) {
	if pattern == path {
		return nil, true
	}
	pattern = strings.Trim(pattern, "/")
	path = strings.Trim(path, "/")
	if pattern == "" || path == "" {
		return nil, false
	}
	pp := strings.Split(pattern, "/")
	sp := strings.Split(path, "/")
	params := make(map[string]string)
	i, j := 0, 0
	for i < len(pp) && j < len(sp) {
		switch {
		case strings.HasPrefix(pp[i], ":"):
			params[strings.TrimPrefix(pp[i], ":")] = sp[j]
		case strings.HasPrefix(pp[i], "*"):
			params[strings.TrimPrefix(pp[i], "*")] = strings.Join(sp[j:], "/")
			return params, true
		case pp[i] != sp[j]:
			return nil, false
		}
		i++
		j++
	}
	if i == len(pp) && j == len(sp) {
		return params, true
	}
	return nil, false
}
