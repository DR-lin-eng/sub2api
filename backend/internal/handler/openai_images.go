package handler

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Images handles OpenAI Images API requests.
// POST /v1/images/generations
// POST /v1/images/edits
func (h *OpenAIGatewayHandler) Images(c *gin.Context) {
	h.ImagesGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) ImagesGateway(c gatewayctx.GatewayContext) {
	streamStarted := false
	defer h.recoverResponsesPanicContext(c, &streamStarted)

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
		"handler.openai_gateway.images",
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

	if isMultipartImagesContentType(c.HeaderValue("Content-Type")) {
		setOpsRequestContextGateway(c, "", false, nil)
	} else {
		setOpsRequestContextGateway(c, "", false, body)
	}

	parsed, err := h.gatewayService.ParseOpenAIImagesRequestContext(c, body)
	if err != nil {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	reqLog = reqLog.With(
		zap.String("model", parsed.Model),
		zap.Bool("stream", parsed.Stream),
		zap.Bool("multipart", parsed.Multipart),
		zap.String("capability", string(parsed.RequiredCapability)),
	)

	if parsed.Multipart {
		setOpsRequestContextGateway(c, parsed.Model, parsed.Stream, nil)
	} else {
		setOpsRequestContextGateway(c, parsed.Model, parsed.Stream, body)
	}
	setOpsEndpointContextGateway(c, "", int16(service.RequestTypeFromLegacy(parsed.Stream, false)))

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughServiceContext(c, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(c)
	service.SetOpsLatencyMsContext(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlotContext(c, subject.UserID, subject.Concurrency, parsed.Stream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("openai.images.billing_eligibility_check_failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.handleStreamingAwareErrorContext(c, status, code, message, streamStarted)
		return
	}

	sessionHash := h.gatewayService.GenerateSessionHashContext(c, body)
	if parsed.Multipart {
		sessionHash = h.gatewayService.GenerateSessionHashWithFallbackContext(c, nil, parsed.StickySessionSeed())
	}

	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		reqLog.Debug("openai.images.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		groupSelection, err := selectOpenAIAPIKeyGroup(
			c.Context(),
			apiKey,
			"",
			sessionHash,
			parsed.Model,
			failedAccountIDs,
			service.OpenAIUpstreamTransportHTTPSSE,
			h.gatewayService.SelectAccountWithScheduler,
		)
		var (
			selection      *service.AccountSelectionResult
			selectedAPIKey = apiKey
		)
		if err == nil && groupSelection != nil {
			selection = groupSelection.Selection
			selectedAPIKey = groupSelection.APIKey
		}
		if err != nil {
			reqLog.Warn("openai.images.account_select_failed", zap.Error(err), zap.Int("excluded_account_count", len(failedAccountIDs)))
			if lastFailoverErr != nil {
				h.handleFailoverExhaustedContext(c, lastFailoverErr, streamStarted)
			} else {
				h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", "No available compatible accounts", streamStarted)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", "No available compatible accounts", streamStarted)
			return
		}

		account := selection.Account
		if !account.SupportsOpenAIImageCapability(parsed.RequiredCapability) {
			failedAccountIDs[account.ID] = struct{}{}
			reqLog.Info("openai.images.account_incompatible",
				zap.Int64("account_id", account.ID),
				zap.String("account_type", account.Type),
				zap.String("capability", string(parsed.RequiredCapability)),
			)
			continue
		}

		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai.images.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		setOpsSelectedAccountGateway(c, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlotContext(c, selectedAPIKey.GroupID, sessionHash, selection, parsed.Stream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMsContext(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		result, err := h.gatewayService.ForwardImagesContext(c.Context(), c, account, body, parsed, "")
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
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				if failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai.images.pool_mode_same_account_retry",
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
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhaustedContext(c, failoverErr, streamStarted)
					return
				}
				switchCount++
				reqLog.Warn("openai.images.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}

			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureForwardErrorResponseContext(c, streamStarted)
			fields := []zap.Field{
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			}
			if shouldLogOpenAIForwardFailureAsWarnContext(c, wroteFallback) {
				reqLog.Warn("openai.images.forward_failed", fields...)
			} else {
				reqLog.Error("openai.images.forward_failed", fields...)
			}
			return
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)

		userAgent := c.HeaderValue("User-Agent")
		clientIP := ip.GetClientIPContext(c)
		requestPayloadHash := service.HashUsageRequestPayload(body)
		inboundEndpoint := GetInboundEndpointContext(c)
		upstreamEndpoint := GetUpstreamEndpointContext(c, account.Platform)

		h.submitUsageRecordTask(func(taskCtx context.Context) {
			_ = h.gatewayService.RecordUsage(taskCtx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             selectedAPIKey,
				User:               selectedAPIKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    inboundEndpoint,
				UpstreamEndpoint:   upstreamEndpoint,
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
			})
		})
		return
	}
}

func isMultipartImagesContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	return err == nil && strings.EqualFold(mediaType, "multipart/form-data")
}
