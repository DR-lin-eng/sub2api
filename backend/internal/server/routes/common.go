package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

const (
	middlewareRequestLogger = "request_logger"
	middlewareSecurity      = "security_headers"
	middlewareCORS          = "cors"
)

// RegisterCommonRoutes 注册通用路由（健康检查、状态等）
func RegisterCommonRoutes(r gin.IRoutes) {
	// 健康检查
	r.GET("/health", gatewayctx.AdaptGinHandler(commonHealth))

	// Claude Code 遥测日志（忽略，直接返回200）
	r.POST("/api/event_logging/batch", gatewayctx.AdaptGinHandler(commonEventLoggingBatch))

	// Setup status endpoint (always returns needs_setup: false in normal mode)
	// This is used by the frontend to detect when the service has restarted after setup
	r.GET("/setup/status", gatewayctx.AdaptGinHandler(commonSetupStatus))
}

func ExecutableCommonRoutes() []gatewayctx.RouteDef {
	return []gatewayctx.RouteDef{
		{
			Method:     http.MethodGet,
			Path:       "/health",
			Handler:    commonHealth,
			Middleware: []string{middlewareRequestLogger, middlewareCORS, middlewareSecurity},
		},
		{
			Method:     http.MethodPost,
			Path:       "/api/event_logging/batch",
			Handler:    commonEventLoggingBatch,
			Middleware: []string{middlewareRequestLogger, middlewareCORS, middlewareSecurity},
		},
		{
			Method:     http.MethodGet,
			Path:       "/setup/status",
			Handler:    commonSetupStatus,
			Middleware: []string{middlewareRequestLogger, middlewareCORS, middlewareSecurity},
		},
	}
}

func commonHealth(c gatewayctx.GatewayContext) {
	c.WriteJSON(http.StatusOK, map[string]any{"status": "ok"})
}

func commonEventLoggingBatch(c gatewayctx.GatewayContext) {
	c.SetStatus(http.StatusOK)
}

func commonSetupStatus(c gatewayctx.GatewayContext) {
	c.WriteJSON(http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]any{
			"needs_setup": false,
			"step":        "completed",
		},
	})
}
