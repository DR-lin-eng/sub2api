package middleware

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ClientRequestID ensures every request has a unique client_request_id in request.Context().
//
// This is used by the Ops monitoring module for end-to-end request correlation.
func ClientRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ApplyClientRequestIDContext(gatewayctx.FromGin(c)) {
			c.Next()
			return
		}
		c.Next()
	}
}

func ApplyClientRequestIDContext(c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil {
		return false
	}
	if v := c.Request().Context().Value(ctxkey.ClientRequestID); v != nil {
		return true
	}

	id := uuid.New().String()
	ctx := context.WithValue(c.Request().Context(), ctxkey.ClientRequestID, id)
	requestLogger := logger.FromContext(ctx).With(zap.String("client_request_id", strings.TrimSpace(id)))
	ctx = logger.IntoContext(ctx, requestLogger)
	c.SetRequest(c.Request().WithContext(ctx))
	return true
}
