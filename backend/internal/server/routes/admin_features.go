package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"

	"github.com/gin-gonic/gin"
)

func executableAdminFeatureRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil || h.Admin == nil {
		return nil
	}

	mw := []string{
		"request_logger",
		"cors",
		"security_headers",
		"client_request_id",
		"admin_auth",
	}

	out := make([]gatewayctx.RouteDef, 0, 30)
	if h.Admin.Affiliate != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/affiliates/users", Handler: h.Admin.Affiliate.ListUsersGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/affiliates/users/lookup", Handler: h.Admin.Affiliate.LookupUsersGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/affiliates/users/:user_id", Handler: h.Admin.Affiliate.UpdateUserSettingsGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/affiliates/users/:user_id", Handler: h.Admin.Affiliate.ClearUserSettingsGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/affiliates/users/batch-rate", Handler: h.Admin.Affiliate.BatchSetRateGateway, Middleware: mw},
		)
	}
	if h.Admin.TLSFingerprintProfile != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/tls-fingerprint-profiles", Handler: h.Admin.TLSFingerprintProfile.ListGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/tls-fingerprint-profiles/:id", Handler: h.Admin.TLSFingerprintProfile.GetByIDGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/tls-fingerprint-profiles", Handler: h.Admin.TLSFingerprintProfile.CreateGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/tls-fingerprint-profiles/:id", Handler: h.Admin.TLSFingerprintProfile.UpdateGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/tls-fingerprint-profiles/:id", Handler: h.Admin.TLSFingerprintProfile.DeleteGateway, Middleware: mw},
		)
	}
	if h.Admin.Channel != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channels", Handler: h.Admin.Channel.ListGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channels/model-pricing", Handler: h.Admin.Channel.GetModelDefaultPricingGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channels/:id", Handler: h.Admin.Channel.GetByIDGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channels", Handler: h.Admin.Channel.CreateGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/channels/:id", Handler: h.Admin.Channel.UpdateGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/channels/:id", Handler: h.Admin.Channel.DeleteGateway, Middleware: mw},
		)
	}
	if h.Admin.ChannelMonitor != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitors", Handler: h.Admin.ChannelMonitor.ListGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitors/:id", Handler: h.Admin.ChannelMonitor.GetGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channel-monitors", Handler: h.Admin.ChannelMonitor.CreateGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/channel-monitors/:id", Handler: h.Admin.ChannelMonitor.UpdateGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/channel-monitors/:id", Handler: h.Admin.ChannelMonitor.DeleteGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channel-monitors/:id/run", Handler: h.Admin.ChannelMonitor.RunGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitors/:id/history", Handler: h.Admin.ChannelMonitor.HistoryGateway, Middleware: mw},
		)
	}
	if h.Admin.ChannelMonitorTemplate != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitor-templates", Handler: h.Admin.ChannelMonitorTemplate.ListGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitor-templates/:id", Handler: h.Admin.ChannelMonitorTemplate.GetGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channel-monitor-templates", Handler: h.Admin.ChannelMonitorTemplate.CreateGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/channel-monitor-templates/:id", Handler: h.Admin.ChannelMonitorTemplate.UpdateGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/channel-monitor-templates/:id", Handler: h.Admin.ChannelMonitorTemplate.DeleteGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channel-monitor-templates/:id/apply", Handler: h.Admin.ChannelMonitorTemplate.ApplyGateway, Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitor-templates/:id/monitors", Handler: h.Admin.ChannelMonitorTemplate.AssociatedMonitorsGateway, Middleware: mw},
		)
	}

	return out
}

func registerAdminFeatureRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	if admin == nil || h == nil || h.Admin == nil {
		return
	}
	if h.Admin.Affiliate != nil {
		affiliates := admin.Group("/affiliates")
		{
			users := affiliates.Group("/users")
			{
				users.GET("", h.Admin.Affiliate.ListUsers)
				users.GET("/lookup", h.Admin.Affiliate.LookupUsers)
				users.PUT("/:user_id", h.Admin.Affiliate.UpdateUserSettings)
				users.DELETE("/:user_id", h.Admin.Affiliate.ClearUserSettings)
				users.POST("/batch-rate", h.Admin.Affiliate.BatchSetRate)
			}
		}
	}
	if h.Admin.TLSFingerprintProfile != nil {
		profiles := admin.Group("/tls-fingerprint-profiles")
		{
			profiles.GET("", h.Admin.TLSFingerprintProfile.List)
			profiles.GET("/:id", h.Admin.TLSFingerprintProfile.GetByID)
			profiles.POST("", h.Admin.TLSFingerprintProfile.Create)
			profiles.PUT("/:id", h.Admin.TLSFingerprintProfile.Update)
			profiles.DELETE("/:id", h.Admin.TLSFingerprintProfile.Delete)
		}
	}
	if h.Admin.Channel != nil {
		channels := admin.Group("/channels")
		{
			channels.GET("", h.Admin.Channel.List)
			channels.GET("/model-pricing", h.Admin.Channel.GetModelDefaultPricing)
			channels.GET("/:id", h.Admin.Channel.GetByID)
			channels.POST("", h.Admin.Channel.Create)
			channels.PUT("/:id", h.Admin.Channel.Update)
			channels.DELETE("/:id", h.Admin.Channel.Delete)
		}
	}
	if h.Admin.ChannelMonitor != nil {
		monitors := admin.Group("/channel-monitors")
		{
			monitors.GET("", h.Admin.ChannelMonitor.List)
			monitors.GET("/:id", h.Admin.ChannelMonitor.Get)
			monitors.POST("", h.Admin.ChannelMonitor.Create)
			monitors.PUT("/:id", h.Admin.ChannelMonitor.Update)
			monitors.DELETE("/:id", h.Admin.ChannelMonitor.Delete)
			monitors.POST("/:id/run", h.Admin.ChannelMonitor.Run)
			monitors.GET("/:id/history", h.Admin.ChannelMonitor.History)
		}
	}
	if h.Admin.ChannelMonitorTemplate != nil {
		templates := admin.Group("/channel-monitor-templates")
		{
			templates.GET("", h.Admin.ChannelMonitorTemplate.List)
			templates.GET("/:id", h.Admin.ChannelMonitorTemplate.Get)
			templates.POST("", h.Admin.ChannelMonitorTemplate.Create)
			templates.PUT("/:id", h.Admin.ChannelMonitorTemplate.Update)
			templates.DELETE("/:id", h.Admin.ChannelMonitorTemplate.Delete)
			templates.POST("/:id/apply", h.Admin.ChannelMonitorTemplate.Apply)
			templates.GET("/:id/monitors", h.Admin.ChannelMonitorTemplate.AssociatedMonitors)
		}
	}
}
