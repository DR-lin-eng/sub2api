package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	sermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/routes"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/setup"
	"github.com/Wei-Shaw/sub2api/internal/web"
	"github.com/redis/go-redis/v9"
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
	middlewareTagAdminAuth        = "admin_auth"
	middlewareTagJWTAuth          = "jwt_auth"
	middlewareTagBackendModeAuth  = "backend_mode_auth_guard"
	middlewareTagBackendModeUser  = "backend_mode_user_guard"
	middlewareTagRLAuthRegister   = "rl_auth_register"
	middlewareTagRLAuthLogin      = "rl_auth_login"
	middlewareTagRLAuthLogin2FA   = "rl_auth_login_2fa"
	middlewareTagRLSendVerify     = "rl_auth_send_verify_code"
	middlewareTagRLRefresh        = "rl_auth_refresh"
	middlewareTagRLPromo          = "rl_auth_validate_promo"
	middlewareTagRLInvite         = "rl_auth_validate_invitation"
	middlewareTagRLForgot         = "rl_auth_forgot_password"
	middlewareTagRLReset          = "rl_auth_reset_password"
	middlewareTagRLLinuxDoFinish  = "rl_auth_linuxdo_complete"
	middlewareTagRLOAuthPendingExchange     = "rl_auth_oauth_pending_exchange"
	middlewareTagRLOAuthPendingSendVerify   = "rl_auth_oauth_pending_send_verify"
	middlewareTagRLOAuthPendingCreateAcct   = "rl_auth_oauth_pending_create_account"
	middlewareTagRLOAuthPendingBindLogin    = "rl_auth_oauth_pending_bind_login"
	middlewareTagRLOAuthLinuxDoBindLogin    = "rl_auth_oauth_linuxdo_bind_login"
	middlewareTagRLOAuthLinuxDoCreateAcct   = "rl_auth_oauth_linuxdo_create_account"
	middlewareTagRLOAuthWeChatComplete      = "rl_auth_oauth_wechat_complete"
	middlewareTagRLOAuthWeChatBindLogin     = "rl_auth_oauth_wechat_bind_login"
	middlewareTagRLOAuthWeChatCreateAcct    = "rl_auth_oauth_wechat_create_account"
	middlewareTagRLOAuthOIDCComplete        = "rl_auth_oauth_oidc_complete"
	middlewareTagRLOAuthOIDCBindLogin       = "rl_auth_oauth_oidc_bind_login"
	middlewareTagRLOAuthOIDCCreateAcct      = "rl_auth_oauth_oidc_create_account"
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
	authService         *service.AuthService
	userService         *service.UserService
	redisClient         *redis.Client
}

func buildExecutableRuntimeConfig(
	cfg *config.Config,
	handlers *handler.Handlers,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	settingService *service.SettingService,
	authService *service.AuthService,
	userService *service.UserService,
	redisClient *redis.Client,
	frontendServer *web.FrontendServer,
) *executableRuntimeConfig {
	rawDefs := make([]gatewayctx.RouteDef, 0, 8)
	rawDefs = append(rawDefs, routes.ExecutableCommonRoutes()...)
	rawDefs = append(rawDefs, setup.ExecutableRoutes()...)
	rawDefs = append(rawDefs, routes.ExecutableGatewayRoutes(handlers)...)
	rawDefs = append(rawDefs, routes.ExecutableAdminRoutes(handlers)...)
	rawDefs = append(rawDefs, routes.ExecutableAuthRoutes(handlers)...)
	rawDefs = append(rawDefs, routes.ExecutableUserRoutes(handlers)...)
	rawDefs = append(rawDefs, routes.ExecutablePaymentRoutes(handlers)...)
	rawDefs = append(rawDefs, routes.ExecutableSoraClientRoutes(handlers)...)
	rawDefs = append(rawDefs, web.ExecutableRoutes(frontendServer)...)

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
		authService:         authService,
		userService:         userService,
		redisClient:         redisClient,
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
		case middlewareTagAdminAuth:
			if runtimeCfg == nil || !sermiddleware.ApplyAdminAuthContext(runtimeCfg.authService, runtimeCfg.userService, runtimeCfg.settingService, c) {
				return false
			}
		case middlewareTagJWTAuth:
			if runtimeCfg == nil || !sermiddleware.ApplyJWTAuthContext(runtimeCfg.authService, runtimeCfg.userService, c) {
				return false
			}
		case middlewareTagBackendModeAuth:
			if runtimeCfg == nil || !sermiddleware.ApplyBackendModeAuthGuardContext(runtimeCfg.settingService, c) {
				return false
			}
		case middlewareTagBackendModeUser:
			if runtimeCfg == nil || !sermiddleware.ApplyBackendModeUserGuardContext(runtimeCfg.settingService, c) {
				return false
			}
		case middlewareTagRLAuthRegister:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "auth-register", 5, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLAuthLogin:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "auth-login", 20, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLAuthLogin2FA:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "auth-login-2fa", 20, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLSendVerify:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "auth-send-verify-code", 5, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLRefresh:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "refresh-token", 30, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLPromo:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "validate-promo", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLInvite:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "validate-invitation", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLForgot:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "forgot-password", 5, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLReset:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "reset-password", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLLinuxDoFinish:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-linuxdo-complete", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthPendingExchange:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-pending-exchange", 20, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthPendingSendVerify:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-pending-send-verify-code", 5, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthPendingCreateAcct:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-pending-create-account", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthPendingBindLogin:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-pending-bind-login", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthLinuxDoBindLogin:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-linuxdo-bind-login", 20, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthLinuxDoCreateAcct:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-linuxdo-create-account", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthWeChatComplete:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-wechat-complete", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthWeChatBindLogin:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-wechat-bind-login", 20, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthWeChatCreateAcct:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-wechat-create-account", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthOIDCComplete:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-oidc-complete", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthOIDCBindLogin:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-oidc-bind-login", 20, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
		case middlewareTagRLOAuthOIDCCreateAcct:
			if runtimeCfg == nil || !appmiddleware.NewRateLimiter(runtimeCfg.redisClient).AllowContext(c, "oauth-oidc-create-account", 10, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}) {
				return false
			}
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
