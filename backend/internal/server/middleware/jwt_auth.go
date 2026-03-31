package middleware

import (
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"

	"github.com/gin-gonic/gin"
)

// NewJWTAuthMiddleware 创建 JWT 认证中间件
func NewJWTAuthMiddleware(authService *service.AuthService, userService *service.UserService) JWTAuthMiddleware {
	return JWTAuthMiddleware(jwtAuth(authService, userService))
}

// jwtAuth JWT认证中间件实现
func jwtAuth(authService *service.AuthService, userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ApplyJWTAuthContext(authService, userService, gatewayctx.FromGin(c)) {
			c.Next()
		}
	}
}

func ApplyJWTAuthContext(authService *service.AuthService, userService *service.UserService, c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
		// 从Authorization header中提取token
		authHeader := c.HeaderValue("Authorization")
		if authHeader == "" {
			AbortWithErrorContext(c, 401, "UNAUTHORIZED", "Authorization header is required")
			return false
		}

		// 验证Bearer scheme
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			AbortWithErrorContext(c, 401, "INVALID_AUTH_HEADER", "Authorization header format must be 'Bearer {token}'")
			return false
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			AbortWithErrorContext(c, 401, "EMPTY_TOKEN", "Token cannot be empty")
			return false
		}

		// 验证token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, service.ErrTokenExpired) {
				AbortWithErrorContext(c, 401, "TOKEN_EXPIRED", "Token has expired")
				return false
			}
			AbortWithErrorContext(c, 401, "INVALID_TOKEN", "Invalid token")
			return false
		}

		// 从数据库获取最新的用户信息
		user, err := userService.GetByID(c.Request().Context(), claims.UserID)
		if err != nil {
			AbortWithErrorContext(c, 401, "USER_NOT_FOUND", "User not found")
			return false
		}

		// 检查用户状态
		if !user.IsActive() {
			AbortWithErrorContext(c, 401, "USER_INACTIVE", "User account is not active")
			return false
		}

		// Security: Validate TokenVersion to ensure token hasn't been invalidated
		// This check ensures tokens issued before a password change are rejected
		if claims.TokenVersion != user.TokenVersion {
			AbortWithErrorContext(c, 401, "TOKEN_REVOKED", "Token has been revoked (password changed)")
			return false
		}

		c.SetValue(string(ContextKeyUser), AuthSubject{
			UserID:      user.ID,
			Concurrency: user.Concurrency,
		})
		c.SetValue(string(ContextKeyUserRole), user.Role)
		return true
}

// Deprecated: prefer GetAuthSubjectFromContext in auth_subject.go.
