package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterGatewayRoutes 注册 API 网关路由（Claude/OpenAI/Gemini 兼容）
func RegisterGatewayRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	cfg *config.Config,
) {
	bodyLimit := middleware.RequestBodyLimit(cfg.Gateway.MaxBodySize)
	soraMaxBodySize := cfg.Gateway.SoraMaxBodySize
	if soraMaxBodySize <= 0 {
		soraMaxBodySize = cfg.Gateway.MaxBodySize
	}
	soraBodyLimit := middleware.RequestBodyLimit(soraMaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	opsErrorLogger := handler.OpsErrorLoggerMiddleware(opsService)
	endpointNorm := handler.InboundEndpointMiddleware()

	// 未分组 Key 拦截中间件（按协议格式区分错误响应）
	requireGroupAnthropic := middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter)
	requireGroupGoogle := middleware.RequireGroupAssignment(settingService, middleware.GoogleErrorWriter)

	// API网关（Claude API兼容）
	gateway := r.Group("/v1")
	gateway.Use(bodyLimit)
	gateway.Use(clientRequestID)
	gateway.Use(opsErrorLogger)
	gateway.Use(endpointNorm)
	gateway.Use(gin.HandlerFunc(apiKeyAuth))
	gateway.Use(requireGroupAnthropic)
	{
		// /v1/messages: auto-route based on group platform
		gateway.POST("/messages", func(c *gin.Context) {
			if getGroupPlatform(c) == service.PlatformOpenAI {
				h.OpenAIGateway.Messages(c)
				return
			}
			h.Gateway.Messages(c)
		})
		// /v1/messages/count_tokens: OpenAI groups get 404
		gateway.POST("/messages/count_tokens", func(c *gin.Context) {
			if getGroupPlatform(c) == service.PlatformOpenAI {
				c.JSON(http.StatusNotFound, gin.H{
					"type": "error",
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Token counting is not supported for this platform",
					},
				})
				return
			}
			h.Gateway.CountTokens(c)
		})
		gateway.GET("/models", h.Gateway.Models)
		gateway.GET("/usage", h.Gateway.Usage)
		// OpenAI Responses API
		gateway.POST("/responses", h.OpenAIGateway.Responses)
		gateway.POST("/responses/*subpath", h.OpenAIGateway.Responses)
		gateway.GET("/responses", h.OpenAIGateway.ResponsesWebSocket)
		// OpenAI Chat Completions API
		gateway.POST("/chat/completions", h.OpenAIGateway.ChatCompletions)
		gateway.POST("/images/generations", func(c *gin.Context) {
			if getGroupPlatform(c) != service.PlatformOpenAI {
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Images API is not supported for this platform",
					},
				})
				return
			}
			h.OpenAIGateway.Images(c)
		})
		gateway.POST("/images/edits", func(c *gin.Context) {
			if getGroupPlatform(c) != service.PlatformOpenAI {
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Images API is not supported for this platform",
					},
				})
				return
			}
			h.OpenAIGateway.Images(c)
		})
	}

	// Gemini 原生 API 兼容层（Gemini SDK/CLI 直连）
	gemini := r.Group("/v1beta")
	gemini.Use(bodyLimit)
	gemini.Use(clientRequestID)
	gemini.Use(opsErrorLogger)
	gemini.Use(endpointNorm)
	gemini.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	gemini.Use(requireGroupGoogle)
	{
		gemini.GET("/models", h.Gateway.GeminiV1BetaListModels)
		gemini.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		// Gin treats ":" as a param marker, but Gemini uses "{model}:{action}" in the same segment.
		gemini.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// OpenAI Responses API（不带v1前缀的别名）
	r.POST("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.OpenAIGateway.Responses)
	r.POST("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.OpenAIGateway.Responses)
	r.GET("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.OpenAIGateway.ResponsesWebSocket)
	// OpenAI Chat Completions API（不带v1前缀的别名）
	r.POST("/chat/completions", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.OpenAIGateway.ChatCompletions)
	r.POST("/images/generations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if getGroupPlatform(c) != service.PlatformOpenAI {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
			return
		}
		h.OpenAIGateway.Images(c)
	})
	r.POST("/images/edits", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if getGroupPlatform(c) != service.PlatformOpenAI {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
			return
		}
		h.OpenAIGateway.Images(c)
	})
	// Claude Code bootstrap / telemetry compatibility endpoints.
	r.GET("/api/claude_cli/bootstrap", clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeBootstrap)
	r.GET("/api/claude_code/organizations/metrics_enabled", clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeMetricsEnabled)
	r.GET("/api/claude_code/settings", clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeManagedSettings)
	r.GET("/api/claude_code/policy_limits", clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudePolicyLimits)
	r.GET("/api/claude_code/user_settings", clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeUserSettings)
	r.PUT("/api/claude_code/user_settings", bodyLimit, clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeUpdateUserSettings)

	// Antigravity 模型列表
	r.GET("/antigravity/models", gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.AntigravityModels)

	// Antigravity 专用路由（仅使用 antigravity 账户，不混合调度）
	antigravityV1 := r.Group("/antigravity/v1")
	antigravityV1.Use(bodyLimit)
	antigravityV1.Use(clientRequestID)
	antigravityV1.Use(opsErrorLogger)
	antigravityV1.Use(endpointNorm)
	antigravityV1.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1.Use(gin.HandlerFunc(apiKeyAuth))
	antigravityV1.Use(requireGroupAnthropic)
	{
		antigravityV1.POST("/messages", h.Gateway.Messages)
		antigravityV1.POST("/messages/count_tokens", h.Gateway.CountTokens)
		antigravityV1.GET("/models", h.Gateway.AntigravityModels)
		antigravityV1.GET("/usage", h.Gateway.Usage)
	}

	antigravityV1Beta := r.Group("/antigravity/v1beta")
	antigravityV1Beta.Use(bodyLimit)
	antigravityV1Beta.Use(clientRequestID)
	antigravityV1Beta.Use(opsErrorLogger)
	antigravityV1Beta.Use(endpointNorm)
	antigravityV1Beta.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1Beta.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	antigravityV1Beta.Use(requireGroupGoogle)
	{
		antigravityV1Beta.GET("/models", h.Gateway.GeminiV1BetaListModels)
		antigravityV1Beta.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		antigravityV1Beta.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// Sora 专用路由（强制使用 sora 平台）
	soraV1 := r.Group("/sora/v1")
	soraV1.Use(soraBodyLimit)
	soraV1.Use(clientRequestID)
	soraV1.Use(opsErrorLogger)
	soraV1.Use(endpointNorm)
	soraV1.Use(middleware.ForcePlatform(service.PlatformSora))
	soraV1.Use(gin.HandlerFunc(apiKeyAuth))
	soraV1.Use(requireGroupAnthropic)
	{
		soraV1.POST("/chat/completions", h.SoraGateway.ChatCompletions)
		soraV1.GET("/models", h.Gateway.Models)
	}

	// Sora 媒体代理（可选 API Key 验证）
	if cfg.Gateway.SoraMediaRequireAPIKey {
		r.GET("/sora/media/*filepath", gin.HandlerFunc(apiKeyAuth), h.SoraGateway.MediaProxy)
	} else {
		r.GET("/sora/media/*filepath", h.SoraGateway.MediaProxy)
	}
	// Sora 媒体代理（签名 URL，无需 API Key）
	r.GET("/sora/media-signed/*filepath", h.SoraGateway.MediaProxySigned)
}

func ExecutableGatewayRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil || h.Gateway == nil {
		return nil
	}
	return []gatewayctx.RouteDef{
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_cli/bootstrap",
			Handler: h.Gateway.ClaudeBootstrapGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_code/organizations/metrics_enabled",
			Handler: h.Gateway.ClaudeMetricsEnabledGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_code/settings",
			Handler: h.Gateway.ClaudeManagedSettingsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_code/policy_limits",
			Handler: h.Gateway.ClaudePolicyLimitsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_code/user_settings",
			Handler: h.Gateway.ClaudeUserSettingsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/claude_code/user_settings",
			Handler: h.Gateway.ClaudeUpdateUserSettingsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/messages",
			Handler: openAIMessagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/messages/count_tokens",
			Handler: openAICountTokensDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/v1/models",
			Handler: h.Gateway.ModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/chat/completions",
			Handler: h.OpenAIGateway.ChatCompletionsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/images/generations",
			Handler: openAIImagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/images/edits",
			Handler: openAIImagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/v1/responses",
			Handler: h.OpenAIGateway.ResponsesWebSocketGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/responses",
			Handler: h.OpenAIGateway.ResponsesGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/responses/*subpath",
			Handler: h.OpenAIGateway.ResponsesGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/chat/completions",
			Handler: h.OpenAIGateway.ChatCompletionsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/images/generations",
			Handler: openAIImagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/images/edits",
			Handler: openAIImagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/responses",
			Handler: h.OpenAIGateway.ResponsesWebSocketGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/responses",
			Handler: h.OpenAIGateway.ResponsesGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/responses/*subpath",
			Handler: h.OpenAIGateway.ResponsesGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/v1beta/models",
			Handler: h.Gateway.GeminiV1BetaListModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"inbound_endpoint",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/v1beta/models/:model",
			Handler: h.Gateway.GeminiV1BetaGetModelGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"inbound_endpoint",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1beta/models/*modelAction",
			Handler: h.Gateway.GeminiV1BetaModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/antigravity/models",
			Handler: h.Gateway.AntigravityModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/antigravity/v1beta/models",
			Handler: h.Gateway.GeminiV1BetaListModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"inbound_endpoint",
				"force_platform_antigravity",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/antigravity/v1beta/models/:model",
			Handler: h.Gateway.GeminiV1BetaGetModelGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"inbound_endpoint",
				"force_platform_antigravity",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/antigravity/v1beta/models/*modelAction",
			Handler: h.Gateway.GeminiV1BetaModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"force_platform_antigravity",
				"google_api_key_auth",
				"require_group_google",
			},
		},
	}
}

func openAIMessagesDispatchGateway(h *handler.Handlers) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		if c == nil || h == nil {
			writeGatewayDispatchUnavailable(c)
			return
		}
		apiKey, ok := middleware.GetAPIKeyFromGatewayContext(c)
		if !ok || apiKey == nil {
			if h.OpenAIGateway != nil {
				h.OpenAIGateway.MessagesGateway(c)
			}
			return
		}
		if apiKey.Group != nil && apiKey.Group.Platform == service.PlatformOpenAI {
			if h.OpenAIGateway != nil {
				h.OpenAIGateway.MessagesGateway(c)
			}
			return
		}
		if h.Gateway != nil {
			h.Gateway.MessagesGateway(c)
			return
		}
		writeGatewayDispatchUnavailable(c)
	}
}

func openAICountTokensDispatchGateway(h *handler.Handlers) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		if c == nil || h == nil {
			writeGatewayDispatchUnavailable(c)
			return
		}
		apiKey, ok := middleware.GetAPIKeyFromGatewayContext(c)
		if !ok || apiKey == nil {
			if h.Gateway != nil {
				h.Gateway.CountTokensGateway(c)
				return
			}
			writeGatewayDispatchUnavailable(c)
			return
		}
		if apiKey.Group != nil && apiKey.Group.Platform == service.PlatformOpenAI {
			c.WriteJSON(http.StatusNotFound, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Token counting is not supported for this platform",
				},
			})
			return
		}
		if h.Gateway != nil {
			h.Gateway.CountTokensGateway(c)
			return
		}
		writeGatewayDispatchUnavailable(c)
	}
}

func openAIImagesDispatchGateway(h *handler.Handlers) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		if c == nil || h == nil {
			writeGatewayDispatchUnavailable(c)
			return
		}
		apiKey, ok := middleware.GetAPIKeyFromGatewayContext(c)
		if !ok || apiKey == nil {
			if h.OpenAIGateway != nil {
				h.OpenAIGateway.ImagesGateway(c)
				return
			}
			writeGatewayDispatchUnavailable(c)
			return
		}
		if apiKey.Group != nil && apiKey.Group.Platform != service.PlatformOpenAI {
			c.WriteJSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
			return
		}
		if h.OpenAIGateway != nil {
			h.OpenAIGateway.ImagesGateway(c)
			return
		}
		writeGatewayDispatchUnavailable(c)
	}
}

func writeGatewayDispatchUnavailable(c gatewayctx.GatewayContext) {
	if c == nil {
		return
	}
	c.WriteJSON(http.StatusServiceUnavailable, gin.H{
		"error": gin.H{
			"type":    "api_error",
			"message": "Service temporarily unavailable",
		},
	})
}

// getGroupPlatform extracts the group platform from the API Key stored in context.
func getGroupPlatform(c *gin.Context) string {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}
