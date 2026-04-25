package routes

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/stretchr/testify/require"
)

func TestCoreExecutableRouteParity(t *testing.T) {
	h := &handler.Handlers{
		Auth:             &handler.AuthHandler{},
		User:             &handler.UserHandler{},
		AvailableChannel: &handler.AvailableChannelHandler{},
		ChannelMonitor:   &handler.ChannelMonitorUserHandler{},
		Payment:          &handler.PaymentHandler{},
		PaymentWebhook:   &handler.PaymentWebhookHandler{},
		Admin: &handler.AdminHandlers{
			Account:                &admin.AccountHandler{},
			TLSFingerprintProfile:  &admin.TLSFingerprintProfileHandler{},
			Channel:                &admin.ChannelHandler{},
			ChannelMonitor:         &admin.ChannelMonitorHandler{},
			ChannelMonitorTemplate: &admin.ChannelMonitorRequestTemplateHandler{},
			Payment:                &admin.PaymentHandler{},
		},
	}

	defs := append([]routeKey{}, collectRouteKeys(ExecutableAuthRoutes(h))...)
	defs = append(defs, collectRouteKeys(ExecutableUserRoutes(h))...)
	defs = append(defs, collectRouteKeys(ExecutablePaymentRoutes(h))...)
	defs = append(defs, collectRouteKeys(ExecutableAdminRoutes(h))...)

	index := make(map[routeKey]struct{}, len(defs))
	for _, def := range defs {
		index[def] = struct{}{}
	}

	expected := []routeKey{
		{method: http.MethodGet, path: "/api/v1/auth/oauth/wechat/start"},
		{method: http.MethodGet, path: "/api/v1/auth/oauth/oidc/start"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/pending/exchange"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/bind-token"},
		{method: http.MethodGet, path: "/api/v1/channels/available"},
		{method: http.MethodGet, path: "/api/v1/channel-monitors"},
		{method: http.MethodGet, path: "/api/v1/channel-monitors/:id/status"},
		{method: http.MethodGet, path: "/api/v1/payment/config"},
		{method: http.MethodPost, path: "/api/v1/payment/orders"},
		{method: http.MethodPost, path: "/api/v1/payment/public/orders/resolve"},
		{method: http.MethodPost, path: "/api/v1/payment/webhook/stripe"},
		{method: http.MethodGet, path: "/api/v1/admin/payment/dashboard"},
		{method: http.MethodGet, path: "/api/v1/admin/channels"},
		{method: http.MethodGet, path: "/api/v1/admin/channel-monitors"},
		{method: http.MethodGet, path: "/api/v1/admin/channel-monitor-templates"},
		{method: http.MethodGet, path: "/api/v1/admin/tls-fingerprint-profiles"},
	}

	missing := make([]routeKey, 0)
	for _, item := range expected {
		if _, ok := index[item]; !ok {
			missing = append(missing, item)
		}
	}
	require.Empty(t, missing, "core upstream feature routes should not remain gin-only")

	nativeAuditDefs := append([]gatewayctx.RouteDef{}, executableUserFeatureRoutes(h)...)
	nativeAuditDefs = append(nativeAuditDefs, ExecutableAuthRoutes(h)...)
	nativeAuditDefs = append(nativeAuditDefs, ExecutablePaymentRoutes(h)...)
	handlerNames := collectRouteHandlerNames(nativeAuditDefs)
	for _, item := range []routeKey{
		{method: http.MethodPost, path: "/api/v1/auth/oauth/pending/exchange"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/pending/send-verify-code"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/pending/create-account"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/pending/bind-login"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/linuxdo/bind-login"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/linuxdo/create-account"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/wechat/bind-login"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/wechat/create-account"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/oidc/bind-login"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/oidc/create-account"},
		{method: http.MethodPost, path: "/api/v1/auth/oauth/bind-token"},
		{method: http.MethodGet, path: "/api/v1/channels/available"},
		{method: http.MethodGet, path: "/api/v1/channel-monitors"},
		{method: http.MethodGet, path: "/api/v1/channel-monitors/:id/status"},
		{method: http.MethodGet, path: "/api/v1/payment/config"},
		{method: http.MethodPost, path: "/api/v1/payment/orders"},
		{method: http.MethodPost, path: "/api/v1/payment/public/orders/resolve"},
		{method: http.MethodPost, path: "/api/v1/payment/webhook/stripe"},
		{method: http.MethodGet, path: "/api/v1/admin/payment/dashboard"},
		{method: http.MethodGet, path: "/api/v1/admin/payment/orders"},
	} {
		name, ok := handlerNames[item]
		require.True(t, ok, "route %s %s should be registered", item.method, item.path)
		require.False(t, strings.Contains(name, "adaptLegacyGinRoute"), "route %s %s should use native gateway handler", item.method, item.path)
	}
}

type routeKey struct {
	method string
	path   string
}

func collectRouteKeys(defs []gatewayctx.RouteDef) []routeKey {
	out := make([]routeKey, 0, len(defs))
	for _, def := range defs {
		out = append(out, routeKey{method: def.Method, path: def.Path})
	}
	return out
}

func collectRouteHandlerNames(defs []gatewayctx.RouteDef) map[routeKey]string {
	out := make(map[routeKey]string, len(defs))
	for _, def := range defs {
		name := ""
		if fn := runtime.FuncForPC(reflect.ValueOf(def.Handler).Pointer()); fn != nil {
			name = fn.Name()
		}
		out[routeKey{method: def.Method, path: def.Path}] = name
	}
	return out
}
