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

	out := make([]gatewayctx.RouteDef, 0, 24)
	if h.Admin.TLSFingerprintProfile != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/tls-fingerprint-profiles", Handler: adaptLegacyGinRoute("/api/v1/admin/tls-fingerprint-profiles", h.Admin.TLSFingerprintProfile.List), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/tls-fingerprint-profiles/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/tls-fingerprint-profiles/:id", h.Admin.TLSFingerprintProfile.GetByID), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/tls-fingerprint-profiles", Handler: adaptLegacyGinRoute("/api/v1/admin/tls-fingerprint-profiles", h.Admin.TLSFingerprintProfile.Create), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/tls-fingerprint-profiles/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/tls-fingerprint-profiles/:id", h.Admin.TLSFingerprintProfile.Update), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/tls-fingerprint-profiles/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/tls-fingerprint-profiles/:id", h.Admin.TLSFingerprintProfile.Delete), Middleware: mw},
		)
	}
	if h.Admin.Channel != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channels", Handler: adaptLegacyGinRoute("/api/v1/admin/channels", h.Admin.Channel.List), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channels/model-pricing", Handler: adaptLegacyGinRoute("/api/v1/admin/channels/model-pricing", h.Admin.Channel.GetModelDefaultPricing), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channels/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channels/:id", h.Admin.Channel.GetByID), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channels", Handler: adaptLegacyGinRoute("/api/v1/admin/channels", h.Admin.Channel.Create), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/channels/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channels/:id", h.Admin.Channel.Update), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/channels/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channels/:id", h.Admin.Channel.Delete), Middleware: mw},
		)
	}
	if h.Admin.ChannelMonitor != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitors", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitors", h.Admin.ChannelMonitor.List), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitors/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitors/:id", h.Admin.ChannelMonitor.Get), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channel-monitors", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitors", h.Admin.ChannelMonitor.Create), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/channel-monitors/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitors/:id", h.Admin.ChannelMonitor.Update), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/channel-monitors/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitors/:id", h.Admin.ChannelMonitor.Delete), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channel-monitors/:id/run", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitors/:id/run", h.Admin.ChannelMonitor.Run), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitors/:id/history", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitors/:id/history", h.Admin.ChannelMonitor.History), Middleware: mw},
		)
	}
	if h.Admin.ChannelMonitorTemplate != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitor-templates", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitor-templates", h.Admin.ChannelMonitorTemplate.List), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitor-templates/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitor-templates/:id", h.Admin.ChannelMonitorTemplate.Get), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channel-monitor-templates", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitor-templates", h.Admin.ChannelMonitorTemplate.Create), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/channel-monitor-templates/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitor-templates/:id", h.Admin.ChannelMonitorTemplate.Update), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/channel-monitor-templates/:id", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitor-templates/:id", h.Admin.ChannelMonitorTemplate.Delete), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/channel-monitor-templates/:id/apply", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitor-templates/:id/apply", h.Admin.ChannelMonitorTemplate.Apply), Middleware: mw},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/channel-monitor-templates/:id/monitors", Handler: adaptLegacyGinRoute("/api/v1/admin/channel-monitor-templates/:id/monitors", h.Admin.ChannelMonitorTemplate.AssociatedMonitors), Middleware: mw},
		)
	}

	return out
}

func registerAdminFeatureRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	if admin == nil || h == nil || h.Admin == nil {
		return
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
