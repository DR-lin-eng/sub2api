package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// NewAPIKeyAuthMiddleware 创建 API Key 认证中间件
func NewAPIKeyAuthMiddleware(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) APIKeyAuthMiddleware {
	return APIKeyAuthMiddleware(apiKeyAuthWithSubscription(apiKeyService, subscriptionService, cfg))
}

// apiKeyAuthWithSubscription API Key认证中间件（支持订阅验证）
//
// 中间件职责分为两层：
//   - 鉴权（Authentication）：验证 Key 有效性、用户状态、IP 限制 —— 始终执行
//   - 计费执行（Billing Enforcement）：过期/配额/订阅/余额检查 —— skipBilling 时整块跳过
//
// /v1/usage 端点只需鉴权，不需要计费执行（允许过期/配额耗尽的 Key 查询自身用量）。
func apiKeyAuthWithSubscription(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ApplyAPIKeyAuthWithSubscriptionContext(apiKeyService, subscriptionService, cfg, gatewayctx.FromGin(c)) {
			return
		}
		c.Next()
	}
}

func ApplyAPIKeyAuthWithSubscriptionContext(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config, c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil || apiKeyService == nil {
		return false
	}

	queryKey := strings.TrimSpace(c.QueryValue("key"))
	queryAPIKey := strings.TrimSpace(c.QueryValue("api_key"))
	if queryKey != "" || queryAPIKey != "" {
		abortWithErrorContext(c, 400, "api_key_in_query_deprecated", "API key in query parameter is deprecated. Please use Authorization header instead.")
		return false
	}

	apiKeyString := extractAPIKeyFromGatewayContext(c)
	if apiKeyString == "" {
		abortWithErrorContext(c, 401, "API_KEY_REQUIRED", "API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header")
		return false
	}

	apiKey, err := apiKeyService.GetByKey(c.Request().Context(), apiKeyString)
	if err != nil {
		if errors.Is(err, service.ErrAPIKeyNotFound) {
			abortWithErrorContext(c, 401, "INVALID_API_KEY", "Invalid API key")
			return false
		}
		abortWithErrorContext(c, 500, "INTERNAL_ERROR", "Failed to validate API key")
		return false
	}

	if !apiKey.IsActive() &&
		apiKey.Status != service.StatusAPIKeyExpired &&
		apiKey.Status != service.StatusAPIKeyQuotaExhausted {
		abortWithErrorContext(c, 401, "API_KEY_DISABLED", "API key is disabled")
		return false
	}

	if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
		allowed, _ := ip.CheckIPRestrictionWithCompiledRules(strings.TrimSpace(c.ClientIP()), apiKey.CompiledIPWhitelist, apiKey.CompiledIPBlacklist)
		if !allowed {
			abortWithErrorContext(c, 403, "ACCESS_DENIED", "Access denied")
			return false
		}
	}

	if apiKey.User == nil {
		abortWithErrorContext(c, 401, "USER_NOT_FOUND", "User associated with API key not found")
		return false
	}
	if !apiKey.User.IsActive() {
		abortWithErrorContext(c, 401, "USER_INACTIVE", "User account is not active")
		return false
	}

	if cfg != nil && cfg.RunMode == config.RunModeSimple {
		setAPIKeyAuthContextValues(c, apiKey, nil)
		_ = apiKeyService.TouchLastUsed(c.Request().Context(), apiKey.ID)
		return true
	}

	skipBilling := c.Path() == "/v1/usage"
	var subscription *service.UserSubscription
	isSubscriptionType := apiKey.Group != nil && apiKey.Group.IsSubscriptionType()

	if isSubscriptionType && subscriptionService != nil {
		sub, subErr := subscriptionService.GetActiveSubscription(
			c.Request().Context(),
			apiKey.User.ID,
			apiKey.Group.ID,
		)
		if subErr != nil {
			if !skipBilling {
				abortWithErrorContext(c, 403, "SUBSCRIPTION_NOT_FOUND", "No active subscription found for this group")
				return false
			}
		} else {
			subscription = sub
		}
	}

	if !skipBilling {
		switch apiKey.Status {
		case service.StatusAPIKeyQuotaExhausted:
			abortWithErrorContext(c, 429, "API_KEY_QUOTA_EXHAUSTED", "API key 额度已用完")
			return false
		case service.StatusAPIKeyExpired:
			abortWithErrorContext(c, 403, "API_KEY_EXPIRED", "API key 已过期")
			return false
		}

		if apiKey.IsExpired() {
			abortWithErrorContext(c, 403, "API_KEY_EXPIRED", "API key 已过期")
			return false
		}
		if apiKey.IsQuotaExhausted() {
			abortWithErrorContext(c, 429, "API_KEY_QUOTA_EXHAUSTED", "API key 额度已用完")
			return false
		}

		if subscription != nil {
			needsMaintenance, validateErr := subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
			if validateErr != nil {
				code := "SUBSCRIPTION_INVALID"
				status := 403
				if errors.Is(validateErr, service.ErrDailyLimitExceeded) ||
					errors.Is(validateErr, service.ErrWeeklyLimitExceeded) ||
					errors.Is(validateErr, service.ErrMonthlyLimitExceeded) {
					code = "USAGE_LIMIT_EXCEEDED"
					status = 429
				}
				abortWithErrorContext(c, status, code, validateErr.Error())
				return false
			}
			if needsMaintenance {
				maintenanceCopy := *subscription
				subscriptionService.DoWindowMaintenance(&maintenanceCopy)
			}
		} else if apiKey.User.Balance <= 0 {
			abortWithErrorContext(c, 403, "INSUFFICIENT_BALANCE", "Insufficient account balance")
			return false
		}
	}

	setAPIKeyAuthContextValues(c, apiKey, subscription)
	_ = apiKeyService.TouchLastUsed(c.Request().Context(), apiKey.ID)
	return true
}

func extractAPIKeyFromGatewayContext(c gatewayctx.GatewayContext) string {
	if c == nil {
		return ""
	}
	authHeader := strings.TrimSpace(c.HeaderValue("Authorization"))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if token := strings.TrimSpace(parts[1]); token != "" {
				return token
			}
		}
	}
	if token := strings.TrimSpace(c.HeaderValue("x-api-key")); token != "" {
		return token
	}
	if token := strings.TrimSpace(c.HeaderValue("x-goog-api-key")); token != "" {
		return token
	}
	return ""
}

func setAPIKeyAuthContextValues(c gatewayctx.GatewayContext, apiKey *service.APIKey, subscription *service.UserSubscription) {
	if c == nil || apiKey == nil || apiKey.User == nil {
		return
	}
	if subscription != nil {
		c.SetValue(string(ContextKeySubscription), subscription)
	}
	c.SetValue(string(ContextKeyAPIKey), apiKey)
	c.SetValue(string(ContextKeyUser), AuthSubject{
		UserID:      apiKey.User.ID,
		Concurrency: apiKey.User.Concurrency,
	})
	c.SetValue(string(ContextKeyUserRole), apiKey.User.Role)
	setGroupContextGateway(c, apiKey.Group)
}

func abortWithErrorContext(c gatewayctx.GatewayContext, statusCode int, code, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(statusCode, NewErrorResponse(code, message))
	c.Abort()
}

// GetAPIKeyFromContext 从上下文中获取API key
func GetAPIKeyFromContext(c *gin.Context) (*service.APIKey, bool) {
	return GetAPIKeyFromGatewayContext(gatewayctx.FromGin(c))
}

func GetAPIKeyFromGatewayContext(c gatewayctx.GatewayContext) (*service.APIKey, bool) {
	if c == nil {
		return nil, false
	}
	value, exists := c.Value(string(ContextKeyAPIKey))
	if !exists {
		return nil, false
	}
	apiKey, ok := value.(*service.APIKey)
	return apiKey, ok
}

// GetSubscriptionFromContext 从上下文中获取订阅信息
func GetSubscriptionFromContext(c *gin.Context) (*service.UserSubscription, bool) {
	return GetSubscriptionFromGatewayContext(gatewayctx.FromGin(c))
}

func GetSubscriptionFromGatewayContext(c gatewayctx.GatewayContext) (*service.UserSubscription, bool) {
	if c == nil {
		return nil, false
	}
	value, exists := c.Value(string(ContextKeySubscription))
	if !exists {
		return nil, false
	}
	subscription, ok := value.(*service.UserSubscription)
	return subscription, ok
}

func setGroupContext(c *gin.Context, group *service.Group) {
	setGroupContextGateway(gatewayctx.FromGin(c), group)
}

func setGroupContextGateway(c gatewayctx.GatewayContext, group *service.Group) {
	if c == nil || c.Request() == nil {
		return
	}
	if !service.IsGroupContextValid(group) {
		return
	}
	if existing, ok := c.Request().Context().Value(ctxkey.Group).(*service.Group); ok && existing != nil && existing.ID == group.ID && service.IsGroupContextValid(existing) {
		return
	}
	ctx := context.WithValue(c.Request().Context(), ctxkey.Group, group)
	c.SetRequest(c.Request().WithContext(ctx))
}
