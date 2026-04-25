package routes

import (
	"net/url"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

func withQueryValue(key, value string, fn gatewayctx.HandlerFunc) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		if c == nil || fn == nil || c.Request() == nil || c.Request().URL == nil {
			return
		}
		original := c.Request()
		cloned := original.Clone(original.Context())
		cloned.URL = cloneURL(original.URL)
		query := cloned.URL.Query()
		query.Set(key, value)
		cloned.URL.RawQuery = query.Encode()
		c.SetRequest(cloned)
		defer c.SetRequest(original)
		fn(c)
	}
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	cloned := *u
	return &cloned
}
