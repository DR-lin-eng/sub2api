package routes

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func ExecutableAuthRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil {
		return nil
	}
	defs := make([]gatewayctx.RouteDef, 0, 16)
	if h.Setting != nil {
		defs = append(defs, gatewayctx.RouteDef{
			Method:  http.MethodGet,
			Path:    "/api/v1/settings/public",
			Handler: h.Setting.GetPublicSettingsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
			},
		})
	}
	if h.Auth != nil {
		defs = append(defs,
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/register",
				Handler: h.Auth.RegisterGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_register",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/login",
				Handler: h.Auth.LoginGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_login",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/login/2fa",
				Handler: h.Auth.Login2FAGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_login_2fa",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/send-verify-code",
				Handler: h.Auth.SendVerifyCodeGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_send_verify_code",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/refresh",
				Handler: h.Auth.RefreshTokenGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_refresh",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/logout",
				Handler: h.Auth.LogoutGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/validate-promo-code",
				Handler: h.Auth.ValidatePromoCodeGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_validate_promo",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/validate-invitation-code",
				Handler: h.Auth.ValidateInvitationCodeGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_validate_invitation",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/forgot-password",
				Handler: h.Auth.ForgotPasswordGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_forgot_password",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/reset-password",
				Handler: h.Auth.ResetPasswordGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_reset_password",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodGet,
				Path:    "/api/v1/auth/oauth/linuxdo/start",
				Handler: h.Auth.LinuxDoOAuthStartGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodGet,
				Path:    "/api/v1/auth/oauth/linuxdo/callback",
				Handler: h.Auth.LinuxDoOAuthCallbackGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/oauth/linuxdo/complete-registration",
				Handler: h.Auth.CompleteLinuxDoOAuthRegistrationGateway,
				Middleware: []string{
					"request_logger", "cors", "security_headers", "client_request_id",
					"backend_mode_auth_guard", "rl_auth_linuxdo_complete",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodGet,
				Path:    "/api/v1/auth/me",
				Handler: h.Auth.GetCurrentUserGateway,
				Middleware: []string{
					"request_logger",
					"cors",
					"security_headers",
					"client_request_id",
					"jwt_auth",
					"backend_mode_user_guard",
				},
			},
			gatewayctx.RouteDef{
				Method:  http.MethodPost,
				Path:    "/api/v1/auth/revoke-all-sessions",
				Handler: h.Auth.RevokeAllSessionsGateway,
				Middleware: []string{
					"request_logger",
					"cors",
					"security_headers",
					"client_request_id",
					"jwt_auth",
					"backend_mode_user_guard",
				},
			},
		)
	}
	if h.Admin != nil && h.Admin.Account != nil {
		defs = append(defs, gatewayctx.RouteDef{
			Method:  http.MethodGet,
			Path:    "/api/v1/public/account-export-tasks/:task_id/download",
			Handler: h.Admin.Account.DownloadExportTaskGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
			},
		})
	}
	return defs
}

// RegisterAuthRoutes 注册认证相关路由
func RegisterAuthRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth servermiddleware.JWTAuthMiddleware,
	redisClient *redis.Client,
	settingService *service.SettingService,
) {
	// 创建速率限制器
	rateLimiter := middleware.NewRateLimiter(redisClient)

	// 公开接口
	auth := v1.Group("/auth")
	auth.Use(servermiddleware.BackendModeAuthGuard(settingService))
	{
		// 注册/登录/2FA/验证码发送均属于高风险入口，增加服务端兜底限流（Redis 故障时 fail-close）
		auth.POST("/register", rateLimiter.LimitWithOptions("auth-register", 5, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.Register)
		auth.POST("/login", rateLimiter.LimitWithOptions("auth-login", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.Login)
		auth.POST("/login/2fa", rateLimiter.LimitWithOptions("auth-login-2fa", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.Login2FA)
		auth.POST("/send-verify-code", rateLimiter.LimitWithOptions("auth-send-verify-code", 5, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.SendVerifyCode)
		// Token刷新接口添加速率限制：每分钟最多 30 次（Redis 故障时 fail-close）
		auth.POST("/refresh", rateLimiter.LimitWithOptions("refresh-token", 30, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.RefreshToken)
		// 登出接口（公开，允许未认证用户调用以撤销Refresh Token）
		auth.POST("/logout", h.Auth.Logout)
		// 优惠码验证接口添加速率限制：每分钟最多 10 次（Redis 故障时 fail-close）
		auth.POST("/validate-promo-code", rateLimiter.LimitWithOptions("validate-promo", 10, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.ValidatePromoCode)
		// 邀请码验证接口添加速率限制：每分钟最多 10 次（Redis 故障时 fail-close）
		auth.POST("/validate-invitation-code", rateLimiter.LimitWithOptions("validate-invitation", 10, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.ValidateInvitationCode)
		// 忘记密码接口添加速率限制：每分钟最多 5 次（Redis 故障时 fail-close）
		auth.POST("/forgot-password", rateLimiter.LimitWithOptions("forgot-password", 5, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.ForgotPassword)
		// 重置密码接口添加速率限制：每分钟最多 10 次（Redis 故障时 fail-close）
		auth.POST("/reset-password", rateLimiter.LimitWithOptions("reset-password", 10, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.ResetPassword)
		auth.GET("/oauth/linuxdo/start", h.Auth.LinuxDoOAuthStart)
		auth.GET("/oauth/linuxdo/callback", h.Auth.LinuxDoOAuthCallback)
		auth.POST("/oauth/linuxdo/complete-registration",
			rateLimiter.LimitWithOptions("oauth-linuxdo-complete", 10, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}),
			h.Auth.CompleteLinuxDoOAuthRegistration,
		)
	}

	// 公开设置（无需认证）
	settings := v1.Group("/settings")
	{
		settings.GET("/public", h.Setting.GetPublicSettings)
	}

	public := v1.Group("/public")
	{
		public.GET("/account-export-tasks/:task_id/download", h.Admin.Account.DownloadExportTask)
	}

	// 需要认证的当前用户信息
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(servermiddleware.BackendModeUserGuard(settingService))
	{
		authenticated.GET("/auth/me", h.Auth.GetCurrentUser)
		// 撤销所有会话（需要认证）
		authenticated.POST("/auth/revoke-all-sessions", h.Auth.RevokeAllSessions)
	}
}
