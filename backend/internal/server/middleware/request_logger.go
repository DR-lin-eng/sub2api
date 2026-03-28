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

const requestIDHeader = "X-Request-ID"

// RequestLogger 在请求入口注入 request-scoped logger。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ApplyRequestLoggerContext(gatewayctx.FromGin(c)) {
			c.Next()
			return
		}
		c.Next()
	}
}

func ApplyRequestLoggerContext(c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil {
		return false
	}

	requestID := strings.TrimSpace(c.HeaderValue(requestIDHeader))
	if requestID == "" {
		requestID = uuid.NewString()
	}
	c.SetHeader(requestIDHeader, requestID)

	ctx := context.WithValue(c.Request().Context(), ctxkey.RequestID, requestID)
	clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string)
	requestLogger := logger.With(
		zap.String("component", "http"),
		zap.String("request_id", requestID),
		zap.String("client_request_id", strings.TrimSpace(clientRequestID)),
		zap.String("path", c.Path()),
		zap.String("method", c.Method()),
	)

	ctx = logger.IntoContext(ctx, requestLogger)
	c.SetRequest(c.Request().WithContext(ctx))
	return true
}
