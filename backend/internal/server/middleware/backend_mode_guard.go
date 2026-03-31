package middleware

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// BackendModeUserGuard blocks non-admin users from accessing user routes when backend mode is enabled.
// Must be placed AFTER JWT auth middleware so that the user role is available in context.
func BackendModeUserGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ApplyBackendModeUserGuardContext(settingService, gatewayctx.FromGin(c)) {
			c.Next()
		}
	}
}

func ApplyBackendModeUserGuardContext(settingService *service.SettingService, c gatewayctx.GatewayContext) bool {
	if settingService == nil || c == nil || !settingService.IsBackendModeEnabled(c.Request().Context()) {
		return true
	}
	role, _ := GetUserRoleFromGatewayContext(c)
	if role == "admin" {
		return true
	}
	response.ErrorContext(gatewayResponder{ctx: c}, 403, "Backend mode is active. User self-service is disabled.")
	c.Abort()
	return false
}

// BackendModeAuthGuard selectively blocks auth endpoints when backend mode is enabled.
// Allows: login, login/2fa, logout, refresh (admin needs these).
// Blocks: register, forgot-password, reset-password, OAuth, etc.
func BackendModeAuthGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ApplyBackendModeAuthGuardContext(settingService, gatewayctx.FromGin(c)) {
			c.Next()
		}
	}
}

func ApplyBackendModeAuthGuardContext(settingService *service.SettingService, c gatewayctx.GatewayContext) bool {
	if settingService == nil || c == nil || !settingService.IsBackendModeEnabled(c.Request().Context()) {
		return true
	}
	path := c.Path()
	allowedSuffixes := []string{"/auth/login", "/auth/login/2fa", "/auth/logout", "/auth/refresh"}
	for _, suffix := range allowedSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	response.ErrorContext(gatewayResponder{ctx: c}, 403, "Backend mode is active. Registration and self-service auth flows are disabled.")
	c.Abort()
	return false
}

type gatewayResponder struct {
	ctx gatewayctx.GatewayContext
}

func (g gatewayResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g gatewayResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}
