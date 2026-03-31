package admin

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

type gatewayJSONResponder struct {
	ctx gatewayctx.GatewayContext
}

func (g gatewayJSONResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g gatewayJSONResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}

func defaultQueryValue(c gatewayctx.GatewayContext, key, fallback string) string {
	if c == nil {
		return fallback
	}
	if value := strings.TrimSpace(c.QueryValue(key)); value != "" {
		return value
	}
	return fallback
}
