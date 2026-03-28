package middleware

import (
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthGoogle is a Google-style error wrapper for API key auth.
func APIKeyAuthGoogle(apiKeyService *service.APIKeyService, cfg *config.Config) gin.HandlerFunc {
	return APIKeyAuthWithSubscriptionGoogle(apiKeyService, nil, cfg)
}

// APIKeyAuthWithSubscriptionGoogle behaves like ApiKeyAuthWithSubscription but returns Google-style errors:
// {"error":{"code":401,"message":"...","status":"UNAUTHENTICATED"}}
//
// It is intended for Gemini native endpoints (/v1beta) to match Gemini SDK expectations.
func APIKeyAuthWithSubscriptionGoogle(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ApplyAPIKeyAuthWithSubscriptionGoogleContext(apiKeyService, subscriptionService, cfg, gatewayctx.FromGin(c)) {
			return
		}
		c.Next()
	}
}

func ApplyAPIKeyAuthWithSubscriptionGoogleContext(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config, c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil || apiKeyService == nil {
		return false
	}
	if v := strings.TrimSpace(c.QueryValue("api_key")); v != "" {
		abortWithGoogleErrorContext(c, 400, "Query parameter api_key is deprecated. Use Authorization header or key instead.")
		return false
	}
	apiKeyString := extractAPIKeyForGoogleContext(c)
	if apiKeyString == "" {
		abortWithGoogleErrorContext(c, 401, "API key is required")
		return false
	}

	apiKey, err := apiKeyService.GetByKey(c.Request().Context(), apiKeyString)
	if err != nil {
		if errors.Is(err, service.ErrAPIKeyNotFound) {
			abortWithGoogleErrorContext(c, 401, "Invalid API key")
			return false
		}
		abortWithGoogleErrorContext(c, 500, "Failed to validate API key")
		return false
	}

	if !apiKey.IsActive() {
		abortWithGoogleErrorContext(c, 401, "API key is disabled")
		return false
	}
	if apiKey.User == nil {
		abortWithGoogleErrorContext(c, 401, "User associated with API key not found")
		return false
	}
	if !apiKey.User.IsActive() {
		abortWithGoogleErrorContext(c, 401, "User account is not active")
		return false
	}

	if cfg != nil && cfg.RunMode == config.RunModeSimple {
		c.SetValue(string(ContextKeyAPIKey), apiKey)
		c.SetValue(string(ContextKeyUser), AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.SetValue(string(ContextKeyUserRole), apiKey.User.Role)
		setGroupContextGateway(c, apiKey.Group)
		_ = apiKeyService.TouchLastUsed(c.Request().Context(), apiKey.ID)
		return true
	}

	isSubscriptionType := apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
	if isSubscriptionType && subscriptionService != nil {
		subscription, err := subscriptionService.GetActiveSubscription(
			c.Request().Context(),
			apiKey.User.ID,
			apiKey.Group.ID,
		)
		if err != nil {
			abortWithGoogleErrorContext(c, 403, "No active subscription found for this group")
			return false
		}

		needsMaintenance, err := subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
		if err != nil {
			status := 403
			if errors.Is(err, service.ErrDailyLimitExceeded) ||
				errors.Is(err, service.ErrWeeklyLimitExceeded) ||
				errors.Is(err, service.ErrMonthlyLimitExceeded) {
				status = 429
			}
			abortWithGoogleErrorContext(c, status, err.Error())
			return false
		}

		c.SetValue(string(ContextKeySubscription), subscription)

		if needsMaintenance {
			maintenanceCopy := *subscription
			subscriptionService.DoWindowMaintenance(&maintenanceCopy)
		}
	} else if apiKey.User.Balance <= 0 {
		abortWithGoogleErrorContext(c, 403, "Insufficient account balance")
		return false
	}

	c.SetValue(string(ContextKeyAPIKey), apiKey)
	c.SetValue(string(ContextKeyUser), AuthSubject{
		UserID:      apiKey.User.ID,
		Concurrency: apiKey.User.Concurrency,
	})
	c.SetValue(string(ContextKeyUserRole), apiKey.User.Role)
	setGroupContextGateway(c, apiKey.Group)
	_ = apiKeyService.TouchLastUsed(c.Request().Context(), apiKey.ID)
	return true
}

// extractAPIKeyForGoogle extracts API key for Google/Gemini endpoints.
// Priority: x-goog-api-key > Authorization: Bearer > x-api-key > query key
// This allows OpenClaw and other clients using Bearer auth to work with Gemini endpoints.
func extractAPIKeyForGoogle(c *gin.Context) string {
	return extractAPIKeyForGoogleContext(gatewayctx.FromGin(c))
}

func extractAPIKeyForGoogleContext(c gatewayctx.GatewayContext) string {
	if c == nil {
		return ""
	}
	if k := strings.TrimSpace(c.HeaderValue("x-goog-api-key")); k != "" {
		return k
	}

	auth := strings.TrimSpace(c.HeaderValue("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if k := strings.TrimSpace(parts[1]); k != "" {
				return k
			}
		}
	}

	if k := strings.TrimSpace(c.HeaderValue("x-api-key")); k != "" {
		return k
	}
	if allowGoogleQueryKey(c.Path()) {
		if v := strings.TrimSpace(c.QueryValue("key")); v != "" {
			return v
		}
	}
	return ""
}

func abortWithGoogleError(c *gin.Context, status int, message string) {
	abortWithGoogleErrorContext(gatewayctx.FromGin(c), status, message)
}

func abortWithGoogleErrorContext(c gatewayctx.GatewayContext, status int, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleapi.HTTPStatusToGoogleStatus(status),
		},
	})
	c.Abort()
}

func allowGoogleQueryKey(path string) bool {
	return strings.HasPrefix(path, "/v1beta") || strings.HasPrefix(path, "/antigravity/v1beta")
}
