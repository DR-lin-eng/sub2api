package routes

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"

	"github.com/gin-gonic/gin"
)

func executableAuthExtensionRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil || h.Auth == nil {
		return nil
	}

	common := []string{
		"request_logger",
		"cors",
		"security_headers",
		"client_request_id",
		"backend_mode_auth_guard",
	}

	defs := []gatewayctx.RouteDef{
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/linuxdo/bind/start",
			Handler:    withQueryValue("intent", "bind_current_user", h.Auth.LinuxDoOAuthStartGateway),
			Middleware: common,
		},
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/wechat/start",
			Handler:    h.Auth.WeChatOAuthStartGateway,
			Middleware: common,
		},
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/wechat/bind/start",
			Handler:    withQueryValue("intent", "bind_current_user", h.Auth.WeChatOAuthStartGateway),
			Middleware: common,
		},
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/wechat/callback",
			Handler:    h.Auth.WeChatOAuthCallbackGateway,
			Middleware: common,
		},
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/wechat/payment/start",
			Handler:    h.Auth.WeChatPaymentOAuthStartGateway,
			Middleware: common,
		},
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/wechat/payment/callback",
			Handler:    h.Auth.WeChatPaymentOAuthCallbackGateway,
			Middleware: common,
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/pending/exchange",
			Handler:    h.Auth.ExchangePendingOAuthCompletionGateway,
			Middleware: append(common, "rl_auth_oauth_pending_exchange"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/pending/send-verify-code",
			Handler:    h.Auth.SendPendingOAuthVerifyCodeGateway,
			Middleware: append(common, "rl_auth_oauth_pending_send_verify"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/pending/create-account",
			Handler:    h.Auth.CreatePendingOAuthAccountGateway,
			Middleware: append(common, "rl_auth_oauth_pending_create_account"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/pending/bind-login",
			Handler:    h.Auth.BindPendingOAuthLoginGateway,
			Middleware: append(common, "rl_auth_oauth_pending_bind_login"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/linuxdo/bind-login",
			Handler:    h.Auth.BindLinuxDoOAuthLoginGateway,
			Middleware: append(common, "rl_auth_oauth_linuxdo_bind_login"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/linuxdo/create-account",
			Handler:    h.Auth.CreateLinuxDoOAuthAccountGateway,
			Middleware: append(common, "rl_auth_oauth_linuxdo_create_account"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/wechat/complete-registration",
			Handler:    h.Auth.CompleteWeChatOAuthRegistrationGateway,
			Middleware: append(common, "rl_auth_oauth_wechat_complete"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/wechat/bind-login",
			Handler:    h.Auth.BindWeChatOAuthLoginGateway,
			Middleware: append(common, "rl_auth_oauth_wechat_bind_login"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/wechat/create-account",
			Handler:    h.Auth.CreateWeChatOAuthAccountGateway,
			Middleware: append(common, "rl_auth_oauth_wechat_create_account"),
		},
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/oidc/start",
			Handler:    h.Auth.OIDCOAuthStartGateway,
			Middleware: common,
		},
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/oidc/bind/start",
			Handler:    withQueryValue("intent", "bind_current_user", h.Auth.OIDCOAuthStartGateway),
			Middleware: common,
		},
		{
			Method:     http.MethodGet,
			Path:       "/api/v1/auth/oauth/oidc/callback",
			Handler:    h.Auth.OIDCOAuthCallbackGateway,
			Middleware: common,
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/oidc/complete-registration",
			Handler:    h.Auth.CompleteOIDCOAuthRegistrationGateway,
			Middleware: append(common, "rl_auth_oauth_oidc_complete"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/oidc/bind-login",
			Handler:    h.Auth.BindOIDCOAuthLoginGateway,
			Middleware: append(common, "rl_auth_oauth_oidc_bind_login"),
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/v1/auth/oauth/oidc/create-account",
			Handler:    h.Auth.CreateOIDCOAuthAccountGateway,
			Middleware: append(common, "rl_auth_oauth_oidc_create_account"),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/auth/oauth/bind-token",
			Handler: h.Auth.PrepareOAuthBindAccessTokenCookieGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"jwt_auth",
				"backend_mode_user_guard",
			},
		},
	}

	return defs
}

func registerAuthExtendedRoutes(auth *gin.RouterGroup, rateLimiter *appmiddleware.RateLimiter, h *handler.Handlers) {
	if auth == nil || rateLimiter == nil || h == nil || h.Auth == nil {
		return
	}

	auth.GET("/oauth/linuxdo/bind/start", func(c *gin.Context) {
		query := c.Request.URL.Query()
		query.Set("intent", "bind_current_user")
		c.Request.URL.RawQuery = query.Encode()
		h.Auth.LinuxDoOAuthStart(c)
	})
	auth.GET("/oauth/wechat/start", h.Auth.WeChatOAuthStart)
	auth.GET("/oauth/wechat/bind/start", func(c *gin.Context) {
		query := c.Request.URL.Query()
		query.Set("intent", "bind_current_user")
		c.Request.URL.RawQuery = query.Encode()
		h.Auth.WeChatOAuthStart(c)
	})
	auth.GET("/oauth/wechat/callback", h.Auth.WeChatOAuthCallback)
	auth.GET("/oauth/wechat/payment/start", h.Auth.WeChatPaymentOAuthStart)
	auth.GET("/oauth/wechat/payment/callback", h.Auth.WeChatPaymentOAuthCallback)
	auth.POST("/oauth/pending/exchange",
		rateLimiter.LimitWithOptions("oauth-pending-exchange", 20, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.ExchangePendingOAuthCompletion,
	)
	auth.POST("/oauth/pending/send-verify-code",
		rateLimiter.LimitWithOptions("oauth-pending-send-verify-code", 5, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.SendPendingOAuthVerifyCode,
	)
	auth.POST("/oauth/pending/create-account",
		rateLimiter.LimitWithOptions("oauth-pending-create-account", 10, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.CreatePendingOAuthAccount,
	)
	auth.POST("/oauth/pending/bind-login",
		rateLimiter.LimitWithOptions("oauth-pending-bind-login", 10, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.BindPendingOAuthLogin,
	)
	auth.POST("/oauth/linuxdo/bind-login",
		rateLimiter.LimitWithOptions("oauth-linuxdo-bind-login", 20, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.BindLinuxDoOAuthLogin,
	)
	auth.POST("/oauth/linuxdo/create-account",
		rateLimiter.LimitWithOptions("oauth-linuxdo-create-account", 10, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.CreateLinuxDoOAuthAccount,
	)
	auth.POST("/oauth/wechat/complete-registration",
		rateLimiter.LimitWithOptions("oauth-wechat-complete", 10, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.CompleteWeChatOAuthRegistration,
	)
	auth.POST("/oauth/wechat/bind-login",
		rateLimiter.LimitWithOptions("oauth-wechat-bind-login", 20, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.BindWeChatOAuthLogin,
	)
	auth.POST("/oauth/wechat/create-account",
		rateLimiter.LimitWithOptions("oauth-wechat-create-account", 10, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.CreateWeChatOAuthAccount,
	)
	auth.GET("/oauth/oidc/start", h.Auth.OIDCOAuthStart)
	auth.GET("/oauth/oidc/bind/start", func(c *gin.Context) {
		query := c.Request.URL.Query()
		query.Set("intent", "bind_current_user")
		c.Request.URL.RawQuery = query.Encode()
		h.Auth.OIDCOAuthStart(c)
	})
	auth.GET("/oauth/oidc/callback", h.Auth.OIDCOAuthCallback)
	auth.POST("/oauth/oidc/complete-registration",
		rateLimiter.LimitWithOptions("oauth-oidc-complete", 10, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.CompleteOIDCOAuthRegistration,
	)
	auth.POST("/oauth/oidc/bind-login",
		rateLimiter.LimitWithOptions("oauth-oidc-bind-login", 20, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.BindOIDCOAuthLogin,
	)
	auth.POST("/oauth/oidc/create-account",
		rateLimiter.LimitWithOptions("oauth-oidc-create-account", 10, time.Minute, appmiddleware.RateLimitOptions{
			FailureMode: appmiddleware.RateLimitFailClose,
		}),
		h.Auth.CreateOIDCOAuthAccount,
	)
}

func registerAuthenticatedAuthExtendedRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	if authenticated == nil || h == nil || h.Auth == nil {
		return
	}
	authenticated.POST("/auth/oauth/bind-token", h.Auth.PrepareOAuthBindAccessTokenCookie)
}
