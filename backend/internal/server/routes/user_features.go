package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"

	"github.com/gin-gonic/gin"
)

func executableUserFeatureRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil {
		return nil
	}
	mw := []string{
		"request_logger",
		"cors",
		"security_headers",
		"client_request_id",
		"jwt_auth",
		"backend_mode_user_guard",
	}
	out := make([]gatewayctx.RouteDef, 0, 4)
	if h.AvailableChannel != nil {
		out = append(out, gatewayctx.RouteDef{
			Method:     http.MethodGet,
			Path:       "/api/v1/channels/available",
			Handler:    h.AvailableChannel.ListGateway,
			Middleware: mw,
		})
	}
	if h.ChannelMonitor != nil {
		out = append(out,
			gatewayctx.RouteDef{
				Method:     http.MethodGet,
				Path:       "/api/v1/channel-monitors",
				Handler:    h.ChannelMonitor.ListGateway,
				Middleware: mw,
			},
			gatewayctx.RouteDef{
				Method:     http.MethodGet,
				Path:       "/api/v1/channel-monitors/:id/status",
				Handler:    h.ChannelMonitor.GetStatusGateway,
				Middleware: mw,
			},
		)
	}
	return out
}

func registerUserFeatureRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	if authenticated == nil || h == nil {
		return
	}
	if h.AvailableChannel != nil {
		authenticated.GET("/channels/available", h.AvailableChannel.List)
	}
	if h.ChannelMonitor != nil {
		monitors := authenticated.Group("/channel-monitors")
		{
			monitors.GET("", h.ChannelMonitor.List)
			monitors.GET("/:id/status", h.ChannelMonitor.GetStatus)
		}
	}
}
