package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func ExecutableUserRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil {
		return nil
	}
	out := make([]gatewayctx.RouteDef, 0, 10)
	if h.User != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/profile", Handler: h.User.GetProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/user/password", Handler: h.User.ChangePasswordGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/user", Handler: h.User.UpdateProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.APIKey != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/keys", Handler: h.APIKey.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/keys/:id", Handler: h.APIKey.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/keys", Handler: h.APIKey.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/keys/:id", Handler: h.APIKey.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/keys/:id", Handler: h.APIKey.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/groups/available", Handler: h.APIKey.GetAvailableGroupsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/groups/rates", Handler: h.APIKey.GetUserGroupRatesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Announcement != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/announcements", Handler: h.Announcement.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/announcements/:id/read", Handler: h.Announcement.MarkReadGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Redeem != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/redeem", Handler: h.Redeem.RedeemGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/redeem/history", Handler: h.Redeem.GetHistoryGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Subscription != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/subscriptions", Handler: h.Subscription.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/subscriptions/active", Handler: h.Subscription.GetActiveGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/subscriptions/progress", Handler: h.Subscription.GetProgressGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/subscriptions/summary", Handler: h.Subscription.GetSummaryGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Usage != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage", Handler: h.Usage.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/:id", Handler: h.Usage.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/stats", Handler: h.Usage.StatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/dashboard/stats", Handler: h.Usage.DashboardStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/dashboard/trend", Handler: h.Usage.DashboardTrendGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/dashboard/models", Handler: h.Usage.DashboardModelsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/usage/dashboard/api-keys-usage", Handler: h.Usage.DashboardAPIKeysUsageGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Totp != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/totp/status", Handler: h.Totp.GetStatusGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/totp/verification-method", Handler: h.Totp.GetVerificationMethodGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/totp/send-code", Handler: h.Totp.SendVerifyCodeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/totp/setup", Handler: h.Totp.InitiateSetupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/totp/enable", Handler: h.Totp.EnableGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/totp/disable", Handler: h.Totp.DisableGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	return out
}

// RegisterUserRoutes 注册用户相关路由（需要认证）
func RegisterUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
) {
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		// 用户接口
		user := authenticated.Group("/user")
		{
			user.GET("/profile", h.User.GetProfile)
			user.PUT("/password", h.User.ChangePassword)
			user.PUT("", h.User.UpdateProfile)

			// TOTP 双因素认证
			totp := user.Group("/totp")
			{
				totp.GET("/status", h.Totp.GetStatus)
				totp.GET("/verification-method", h.Totp.GetVerificationMethod)
				totp.POST("/send-code", h.Totp.SendVerifyCode)
				totp.POST("/setup", h.Totp.InitiateSetup)
				totp.POST("/enable", h.Totp.Enable)
				totp.POST("/disable", h.Totp.Disable)
			}
		}

		// API Key管理
		keys := authenticated.Group("/keys")
		{
			keys.GET("", h.APIKey.List)
			keys.GET("/:id", h.APIKey.GetByID)
			keys.POST("", h.APIKey.Create)
			keys.PUT("/:id", h.APIKey.Update)
			keys.DELETE("/:id", h.APIKey.Delete)
		}

		// 用户可用分组（非管理员接口）
		groups := authenticated.Group("/groups")
		{
			groups.GET("/available", h.APIKey.GetAvailableGroups)
			groups.GET("/rates", h.APIKey.GetUserGroupRates)
		}

		// 使用记录
		usage := authenticated.Group("/usage")
		{
			usage.GET("", h.Usage.List)
			usage.GET("/:id", h.Usage.GetByID)
			usage.GET("/stats", h.Usage.Stats)
			// User dashboard endpoints
			usage.GET("/dashboard/stats", h.Usage.DashboardStats)
			usage.GET("/dashboard/trend", h.Usage.DashboardTrend)
			usage.GET("/dashboard/models", h.Usage.DashboardModels)
			usage.POST("/dashboard/api-keys-usage", h.Usage.DashboardAPIKeysUsage)
		}

		// 公告（用户可见）
		announcements := authenticated.Group("/announcements")
		{
			announcements.GET("", h.Announcement.List)
			announcements.POST("/:id/read", h.Announcement.MarkRead)
		}

		// 卡密兑换
		redeem := authenticated.Group("/redeem")
		{
			redeem.POST("", h.Redeem.Redeem)
			redeem.GET("/history", h.Redeem.GetHistory)
		}

		// 用户订阅
		subscriptions := authenticated.Group("/subscriptions")
		{
			subscriptions.GET("", h.Subscription.List)
			subscriptions.GET("/active", h.Subscription.GetActive)
			subscriptions.GET("/progress", h.Subscription.GetProgress)
			subscriptions.GET("/summary", h.Subscription.GetSummary)
		}
	}
}
