package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"

	"github.com/gin-gonic/gin"
)

// AdminOnly 管理员权限中间件
// 必须在JWTAuth中间件之后使用
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ApplyAdminOnlyContext(gatewayctx.FromGin(c)) {
			c.Next()
		}
	}
}

func ApplyAdminOnlyContext(c gatewayctx.GatewayContext) bool {
	role, ok := GetUserRoleFromGatewayContext(c)
	if !ok {
		AbortWithErrorContext(c, 401, "UNAUTHORIZED", "User not found in context")
		return false
	}
	if role != service.RoleAdmin {
		AbortWithErrorContext(c, 403, "FORBIDDEN", "Admin access required")
		return false
	}
	return true
}
