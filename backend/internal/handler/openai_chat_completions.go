package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// ChatCompletions handles OpenAI Chat Completions API requests.
// POST /v1/chat/completions
func (h *OpenAIGatewayHandler) ChatCompletions(c *gin.Context) {
	h.ChatCompletionsGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) ChatCompletionsGateway(c gatewayctx.GatewayContext) {
	streamStarted := false
	defer h.recoverChatCompletionsPanicContext(c, &streamStarted)

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromGatewayContext(c)
	if !ok {
		h.errorResponseGateway(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		h.errorResponseGateway(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLoggerContext(
		c,
		"handler.openai_gateway.chat_completions",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	if !h.ensureResponsesDependenciesGateway(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request())
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponseGateway(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	if !gjson.ValidBytes(body) {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}

	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()
	reqStream := gjson.GetBytes(body, "stream").Bool()

	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))

	setOpsRequestContextGateway(c, reqModel, reqStream, body)

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughServiceContext(c, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(c)

	service.SetOpsLatencyMsContext(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlotContext(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("openai_chat_completions.billing_eligibility_check_failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.handleStreamingAwareErrorContext(c, status, code, message, streamStarted)
		return
	}

	sessionHash := h.resolveOpenAIStickySessionHashContext(c, apiKey, subject.UserID, reqModel, reqStream, body)
	promptCacheKey := h.gatewayService.ExtractSessionIDContext(c, body)

	failoverPolicy := h.resolveOpenAIFailoverPolicy(body, reqStream)
	maxAccountSwitches := failoverPolicy.MaxSwitches
	if maxAccountSwitches != h.maxAccountSwitches || !failoverPolicy.AllowSameAccountRetry {
		reqLog.Info("openai_chat_completions.streaming_failover_policy_applied",
			zap.Int("body_bytes", len(body)),
			zap.Bool("stream", reqStream),
			zap.Int("effective_max_switches", maxAccountSwitches),
			zap.Bool("allow_same_account_retry", failoverPolicy.AllowSameAccountRetry),
		)
	}
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		c.SetValue("openai_chat_completions_fallback_model", "")
		reqLog.Debug("openai_chat_completions.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithScheduler(
			c.Context(),
			apiKey.GroupID,
			"",
			sessionHash,
			reqModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
		)
		if err != nil {
			reqLog.Warn("openai_chat_completions.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				defaultModel := ""
				if apiKey.Group != nil {
					defaultModel = apiKey.Group.DefaultMappedModel
				}
				if defaultModel != "" && defaultModel != reqModel {
					reqLog.Info("openai_chat_completions.fallback_to_default_model",
						zap.String("default_mapped_model", defaultModel),
					)
					selection, scheduleDecision, err = h.gatewayService.SelectAccountWithScheduler(
						c.Context(),
						apiKey.GroupID,
						"",
						sessionHash,
						defaultModel,
						failedAccountIDs,
						service.OpenAIUpstreamTransportAny,
					)
					if err == nil && selection != nil {
						c.SetValue("openai_chat_completions_fallback_model", defaultModel)
					}
				}
				if err != nil {
					h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable", streamStarted)
					return
				}
			} else {
				if lastFailoverErr != nil {
					h.handleFailoverExhaustedContext(c, lastFailoverErr, streamStarted)
				} else {
					h.handleStreamingAwareErrorContext(c, http.StatusBadGateway, "api_error", "Upstream request failed", streamStarted)
				}
				return
			}
		}
		if selection == nil || selection.Account == nil {
			h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
			return
		}
		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai_chat_completions.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		_ = scheduleDecision
		setOpsSelectedAccountGateway(c, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlotContext(c, apiKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMsContext(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()

		defaultMappedModel := resolveOpenAIForwardDefaultMappedModel(apiKey, getContextStringGateway(c, "openai_chat_completions_fallback_model"))
		result, err := h.gatewayService.ForwardAsChatCompletionsContext(c.Context(), c, account, body, promptCacheKey, defaultMappedModel)

		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64Gateway(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMsContext(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMsContext(c, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failoverErr = adjustOpenAINetworkFailoverCooldown(failoverErr)
				failoverErr = adjustOpenAIFailoverForLargeStreamingRequest(failoverErr, body, reqStream, h.cfg)
				h.gatewayService.RegisterOpenAIRuntimeFailure(account, failoverErr)
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				// Pool mode: retry on the same account
				if failoverPolicy.AllowSameAccountRetry && failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai_chat_completions.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-c.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				_ = h.gatewayService.ClearStickySessionForAccount(c.Context(), apiKey.GroupID, sessionHash, account.ID)
				h.gatewayService.TempUnscheduleRetryableError(c.Context(), account.ID, failoverErr)
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhaustedContext(c, failoverErr, streamStarted)
					return
				}
				switchCount++
				reqLog.Warn("openai_chat_completions.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureForwardErrorResponseContext(c, streamStarted)
			reqLog.Warn("openai_chat_completions.forward_failed",
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			)
			return
		}
		if result != nil {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}
		h.gatewayService.MarkOpenAIAccountHealthy(account)
		if err := h.gatewayService.PromoteStickySession(c.Context(), apiKey.GroupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai_chat_completions.promote_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}

		userAgent := c.HeaderValue("User-Agent")
		clientIP := c.ClientIP()

		h.submitUsageRecordTask(func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:           result,
				APIKey:           apiKey,
				User:             apiKey.User,
				Account:          account,
				Subscription:     subscription,
				InboundEndpoint:  GetInboundEndpointContext(c),
				UpstreamEndpoint: GetUpstreamEndpointContext(c, account.Platform),
				UserAgent:        userAgent,
				IPAddress:        clientIP,
				APIKeyService:    h.apiKeyService,
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.chat_completions"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", apiKey.ID),
					zap.Any("group_id", apiKey.GroupID),
					zap.String("model", reqModel),
					zap.Int64("account_id", account.ID),
				).Error("openai_chat_completions.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("openai_chat_completions.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func (h *OpenAIGatewayHandler) recoverChatCompletionsPanicContext(c gatewayctx.GatewayContext, streamStarted *bool) {
	recovered := recover()
	if recovered == nil {
		return
	}
	started := streamStarted != nil && *streamStarted
	requestLoggerContext(c, "handler.openai_gateway.chat_completions").Error(
		"openai.chat_completions_panic_recovered",
		zap.Bool("stream_started", started),
		zap.Any("panic", recovered),
	)
	if !started {
		h.errorResponseGateway(c, http.StatusInternalServerError, "api_error", "Internal server error")
	}
}
