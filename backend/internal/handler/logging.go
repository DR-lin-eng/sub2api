package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func requestLogger(c *gin.Context, component string, fields ...zap.Field) *zap.Logger {
	base := logger.L()
	if c != nil && c.Request != nil {
		base = logger.FromContext(c.Request.Context())
	}
	return requestLoggerFromBase(base, component, fields...)
}

func requestLoggerContext(c gatewayctx.GatewayContext, component string, fields ...zap.Field) *zap.Logger {
	base := logger.L()
	if c != nil {
		base = logger.FromContext(c.Context())
	}
	return requestLoggerFromBase(base, component, fields...)
}

func requestLoggerFromBase(base *zap.Logger, component string, fields ...zap.Field) *zap.Logger {
	if component != "" {
		fields = append([]zap.Field{zap.String("component", component)}, fields...)
	}
	return base.With(fields...)
}
