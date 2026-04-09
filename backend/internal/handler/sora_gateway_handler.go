package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/soraerror"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

// SoraGatewayHandler handles Sora chat completions requests
type SoraGatewayHandler struct {
	gatewayService        *service.GatewayService
	soraGatewayService    *service.SoraGatewayService
	billingCacheService   *service.BillingCacheService
	usageRecordWorkerPool *service.UsageRecordWorkerPool
	concurrencyHelper     *ConcurrencyHelper
	maxAccountSwitches    int
	streamMode            string
	soraTLSEnabled        bool
	soraMediaSigningKey   string
	soraMediaRoot         string
}

// NewSoraGatewayHandler creates a new SoraGatewayHandler
func NewSoraGatewayHandler(
	gatewayService *service.GatewayService,
	soraGatewayService *service.SoraGatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	cfg *config.Config,
) *SoraGatewayHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 3
	streamMode := "force"
	soraTLSEnabled := true
	signKey := ""
	mediaRoot := "/app/data/sora"
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
		if mode := strings.TrimSpace(cfg.Gateway.SoraStreamMode); mode != "" {
			streamMode = mode
		}
		soraTLSEnabled = !cfg.Sora.Client.DisableTLSFingerprint
		signKey = strings.TrimSpace(cfg.Gateway.SoraMediaSigningKey)
		if root := strings.TrimSpace(cfg.Sora.Storage.LocalPath); root != "" {
			mediaRoot = root
		}
	}
	return &SoraGatewayHandler{
		gatewayService:        gatewayService,
		soraGatewayService:    soraGatewayService,
		billingCacheService:   billingCacheService,
		usageRecordWorkerPool: usageRecordWorkerPool,
		concurrencyHelper:     NewConcurrencyHelper(concurrencyService, SSEPingFormatComment, pingInterval),
		maxAccountSwitches:    maxAccountSwitches,
		streamMode:            strings.ToLower(streamMode),
		soraTLSEnabled:        soraTLSEnabled,
		soraMediaSigningKey:   signKey,
		soraMediaRoot:         mediaRoot,
	}
}

// ChatCompletions handles Sora /v1/chat/completions endpoint
func (h *SoraGatewayHandler) ChatCompletions(c *gin.Context) {
	transportCtx := gatewayctx.FromGin(c)
	apiKey, ok := middleware2.GetAPIKeyFromGatewayContext(transportCtx)
	if !ok {
		h.errorResponseGateway(transportCtx, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(transportCtx)
	if !ok {
		h.errorResponseGateway(transportCtx, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLoggerContext(
		transportCtx,
		"handler.sora_gateway.chat_completions",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	requestStart := time.Now()

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(transportCtx.Request())
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponseGateway(transportCtx, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	setOpsRequestContextGateway(transportCtx, "", false, body)

	// 校验请求体 JSON 合法性
	if !gjson.ValidBytes(body) {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}

	// 使用 gjson 只读提取字段做校验，避免完整 Unmarshal
	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()

	msgsResult := gjson.GetBytes(body, "messages")
	if !msgsResult.IsArray() || len(msgsResult.Array()) == 0 {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "messages is required")
		return
	}

	clientStream := gjson.GetBytes(body, "stream").Bool()
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", clientStream))
	if !clientStream {
		if h.streamMode == "error" {
			h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Sora requires stream=true")
			return
		}
		var err error
		body, err = sjson.SetBytes(body, "stream", true)
		if err != nil {
			h.errorResponseGateway(transportCtx, http.StatusInternalServerError, "api_error", "Failed to process request")
			return
		}
	}

	setOpsRequestContextGateway(transportCtx, reqModel, clientStream, body)
	setOpsEndpointContextGateway(transportCtx, "", int16(service.RequestTypeFromLegacy(clientStream, false)))

	platform := ""
	if forced, ok := middleware2.GetForcePlatformFromGatewayContext(transportCtx); ok {
		platform = forced
	} else if apiKey.Group != nil {
		platform = apiKey.Group.Platform
	}
	if platform != service.PlatformSora {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "This endpoint only supports Sora platform")
		return
	}

	streamStarted := false
	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(transportCtx)
	service.SetOpsLatencyMsContext(transportCtx, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	maxWait := service.CalculateMaxWait(subject.Concurrency)
	canWait, err := h.concurrencyHelper.IncrementWaitCount(transportCtx.Context(), subject.UserID, maxWait)
	waitCounted := false
	if err != nil {
		reqLog.Warn("sora.user_wait_counter_increment_failed", zap.Error(err))
	} else if !canWait {
		reqLog.Info("sora.user_wait_queue_full", zap.Int("max_wait", maxWait))
		h.errorResponseGateway(transportCtx, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return
	}
	if err == nil && canWait {
		waitCounted = true
	}
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementWaitCount(transportCtx.Context(), subject.UserID)
		}
	}()

	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWaitContext(transportCtx, subject.UserID, subject.Concurrency, clientStream, &streamStarted)
	if err != nil {
		reqLog.Warn("sora.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyErrorContext(transportCtx, err, "user", streamStarted)
		return
	}
	if waitCounted {
		h.concurrencyHelper.DecrementWaitCount(transportCtx.Context(), subject.UserID)
		waitCounted = false
	}
	userReleaseFunc = wrapReleaseOnDone(transportCtx.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(transportCtx.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("sora.billing_eligibility_check_failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.handleStreamingAwareErrorContext(transportCtx, status, code, message, streamStarted)
		return
	}

	sessionHash := generateOpenAISessionHashContext(transportCtx, body)

	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	lastFailoverStatus := 0
	var lastFailoverBody []byte
	var lastFailoverHeaders http.Header

	for {
		groupSelection, err := selectGatewayAPIKeyGroup(
			transportCtx.Context(),
			apiKey,
			sessionHash,
			reqModel,
			failedAccountIDs,
			"",
			h.gatewayService.SelectAccountWithLoadAwareness,
		)
		if err != nil {
			reqLog.Warn("sora.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				h.handleStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error(), streamStarted)
				return
			}
			rayID, mitigated, contentType := extractSoraFailoverHeaderInsights(lastFailoverHeaders, lastFailoverBody)
			fields := []zap.Field{
				zap.Int("last_upstream_status", lastFailoverStatus),
			}
			if rayID != "" {
				fields = append(fields, zap.String("last_upstream_cf_ray", rayID))
			}
			if mitigated != "" {
				fields = append(fields, zap.String("last_upstream_cf_mitigated", mitigated))
			}
			if contentType != "" {
				fields = append(fields, zap.String("last_upstream_content_type", contentType))
			}
			reqLog.Warn("sora.failover_exhausted_no_available_accounts", fields...)
			h.handleFailoverExhaustedContext(transportCtx, lastFailoverStatus, lastFailoverHeaders, lastFailoverBody, streamStarted)
			return
		}
		selectedAPIKey := groupSelection.APIKey
		selection := groupSelection.Selection
		account := selection.Account
		setOpsSelectedAccountGateway(transportCtx, account.ID, account.Platform)
		proxyBound := account.ProxyID != nil
		proxyID := int64(0)
		if account.ProxyID != nil {
			proxyID = *account.ProxyID
		}
		tlsFingerprintEnabled := h.soraTLSEnabled

		accountReleaseFunc := selection.ReleaseFunc
		if !selection.Acquired {
			if selection.WaitPlan == nil {
				h.handleStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
				return
			}
			accountWaitCounted := false
			canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(transportCtx.Context(), account.ID, selection.WaitPlan.MaxWaiting)
			if err != nil {
				reqLog.Warn("sora.account_wait_counter_increment_failed",
					zap.Int64("account_id", account.ID),
					zap.Int64("proxy_id", proxyID),
					zap.Bool("proxy_bound", proxyBound),
					zap.Bool("tls_fingerprint_enabled", tlsFingerprintEnabled),
					zap.Error(err),
				)
			} else if !canWait {
				reqLog.Info("sora.account_wait_queue_full",
					zap.Int64("account_id", account.ID),
					zap.Int64("proxy_id", proxyID),
					zap.Bool("proxy_bound", proxyBound),
					zap.Bool("tls_fingerprint_enabled", tlsFingerprintEnabled),
					zap.Int("max_waiting", selection.WaitPlan.MaxWaiting),
				)
				h.handleStreamingAwareErrorContext(transportCtx, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", streamStarted)
				return
			}
			if err == nil && canWait {
				accountWaitCounted = true
			}
			defer func() {
				if accountWaitCounted {
					h.concurrencyHelper.DecrementAccountWaitCount(transportCtx.Context(), account.ID)
				}
			}()

			accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeoutContext(
				transportCtx,
				account.ID,
				selection.WaitPlan.MaxConcurrency,
				selection.WaitPlan.Timeout,
				clientStream,
				&streamStarted,
			)
			if err != nil {
				reqLog.Warn("sora.account_slot_acquire_failed",
					zap.Int64("account_id", account.ID),
					zap.Int64("proxy_id", proxyID),
					zap.Bool("proxy_bound", proxyBound),
					zap.Bool("tls_fingerprint_enabled", tlsFingerprintEnabled),
					zap.Error(err),
				)
				h.handleConcurrencyErrorContext(transportCtx, err, "account", streamStarted)
				return
			}
			if accountWaitCounted {
				h.concurrencyHelper.DecrementAccountWaitCount(transportCtx.Context(), account.ID)
				accountWaitCounted = false
			}
		}
		accountReleaseFunc = wrapReleaseOnDone(transportCtx.Context(), accountReleaseFunc)

		service.SetOpsLatencyMsContext(transportCtx, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		result, err := h.soraGatewayService.ForwardContext(transportCtx.Context(), transportCtx, account, body, clientStream)
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		service.SetOpsLatencyMsContext(transportCtx, service.OpsResponseLatencyMsKey, forwardDurationMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMsContext(transportCtx, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportGatewayAccountScheduleResult(account.ID, false, nil)
				h.gatewayService.RecordGatewayAccountSwitch(account.ID)
				failedAccountIDs[account.ID] = struct{}{}
				if switchCount >= maxAccountSwitches {
					lastFailoverStatus = failoverErr.StatusCode
					lastFailoverHeaders = cloneHTTPHeaders(failoverErr.ResponseHeaders)
					lastFailoverBody = failoverErr.ResponseBody
					rayID, mitigated, contentType := extractSoraFailoverHeaderInsights(lastFailoverHeaders, lastFailoverBody)
					fields := []zap.Field{
						zap.Int64("account_id", account.ID),
						zap.Int64("proxy_id", proxyID),
						zap.Bool("proxy_bound", proxyBound),
						zap.Bool("tls_fingerprint_enabled", tlsFingerprintEnabled),
						zap.Int("upstream_status", failoverErr.StatusCode),
						zap.Int("switch_count", switchCount),
						zap.Int("max_switches", maxAccountSwitches),
					}
					if rayID != "" {
						fields = append(fields, zap.String("upstream_cf_ray", rayID))
					}
					if mitigated != "" {
						fields = append(fields, zap.String("upstream_cf_mitigated", mitigated))
					}
					if contentType != "" {
						fields = append(fields, zap.String("upstream_content_type", contentType))
					}
					reqLog.Warn("sora.upstream_failover_exhausted", fields...)
					h.handleFailoverExhaustedContext(transportCtx, lastFailoverStatus, lastFailoverHeaders, lastFailoverBody, streamStarted)
					return
				}
				lastFailoverStatus = failoverErr.StatusCode
				lastFailoverHeaders = cloneHTTPHeaders(failoverErr.ResponseHeaders)
				lastFailoverBody = failoverErr.ResponseBody
				switchCount++
				upstreamErrCode, upstreamErrMsg := extractUpstreamErrorCodeAndMessage(lastFailoverBody)
				rayID, mitigated, contentType := extractSoraFailoverHeaderInsights(lastFailoverHeaders, lastFailoverBody)
				fields := []zap.Field{
					zap.Int64("account_id", account.ID),
					zap.Int64("proxy_id", proxyID),
					zap.Bool("proxy_bound", proxyBound),
					zap.Bool("tls_fingerprint_enabled", tlsFingerprintEnabled),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.String("upstream_error_code", upstreamErrCode),
					zap.String("upstream_error_message", upstreamErrMsg),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				}
				if rayID != "" {
					fields = append(fields, zap.String("upstream_cf_ray", rayID))
				}
				if mitigated != "" {
					fields = append(fields, zap.String("upstream_cf_mitigated", mitigated))
				}
				if contentType != "" {
					fields = append(fields, zap.String("upstream_content_type", contentType))
				}
				reqLog.Warn("sora.upstream_failover_switching", fields...)
				continue
			}
			h.gatewayService.ReportGatewayAccountScheduleResult(account.ID, false, nil)
			reqLog.Error("sora.forward_failed",
				zap.Int64("account_id", account.ID),
				zap.Int64("proxy_id", proxyID),
				zap.Bool("proxy_bound", proxyBound),
				zap.Bool("tls_fingerprint_enabled", tlsFingerprintEnabled),
				zap.Error(err),
			)
			return
		}
		if result != nil && result.FirstTokenMs != nil {
			h.gatewayService.ReportGatewayAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportGatewayAccountScheduleResult(account.ID, true, nil)
		}

		userAgent := strings.TrimSpace(transportCtx.HeaderValue("User-Agent"))
		clientIP := strings.TrimSpace(transportCtx.ClientIP())
		requestPayloadHash := service.HashUsageRequestPayload(body)
		inboundEndpoint := GetInboundEndpointContext(transportCtx)
		upstreamEndpoint := GetUpstreamEndpointContext(transportCtx, account.Platform)

		// 使用量记录通过有界 worker 池提交，避免请求热路径创建无界 goroutine。
		h.submitUsageRecordTask(func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
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
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.sora_gateway.chat_completions"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", selectedAPIKey.ID),
					zap.Any("group_id", selectedAPIKey.GroupID),
					zap.String("model", reqModel),
					zap.Int64("account_id", account.ID),
				).Error("sora.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("sora.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int64("proxy_id", proxyID),
			zap.Bool("proxy_bound", proxyBound),
			zap.Bool("tls_fingerprint_enabled", tlsFingerprintEnabled),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func generateOpenAISessionHash(c *gin.Context, body []byte) string {
	return generateOpenAISessionHashContext(gatewayctx.FromGin(c), body)
}

func generateOpenAISessionHashContext(c gatewayctx.GatewayContext, body []byte) string {
	if c == nil {
		return ""
	}
	sessionID := strings.TrimSpace(c.HeaderValue("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.HeaderValue("conversation_id"))
	}
	if sessionID == "" && len(body) > 0 {
		sessionID = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	}
	if sessionID == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(hash[:])
}

func (h *SoraGatewayHandler) submitUsageRecordTask(task service.UsageRecordTask) {
	if task == nil {
		return
	}
	if h.usageRecordWorkerPool != nil {
		h.usageRecordWorkerPool.Submit(task)
		return
	}
	// 回退路径：worker 池未注入时同步执行，避免退回到无界 goroutine 模式。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().With(
				zap.String("component", "handler.sora_gateway.chat_completions"),
				zap.Any("panic", recovered),
			).Error("sora.usage_record_task_panic_recovered")
		}
	}()
	task(ctx)
}

func (h *SoraGatewayHandler) handleConcurrencyError(c *gin.Context, err error, slotType string, streamStarted bool) {
	h.handleConcurrencyErrorContext(gatewayctx.FromGin(c), err, slotType, streamStarted)
}

func (h *SoraGatewayHandler) handleConcurrencyErrorContext(c gatewayctx.GatewayContext, err error, slotType string, streamStarted bool) {
	h.handleStreamingAwareErrorContext(c, http.StatusTooManyRequests, "rate_limit_error",
		fmt.Sprintf("Concurrency limit exceeded for %s, please retry later", slotType), streamStarted)
}

func (h *SoraGatewayHandler) handleFailoverExhausted(c *gin.Context, statusCode int, responseHeaders http.Header, responseBody []byte, streamStarted bool) {
	h.handleFailoverExhaustedContext(gatewayctx.FromGin(c), statusCode, responseHeaders, responseBody, streamStarted)
}

func (h *SoraGatewayHandler) handleFailoverExhaustedContext(c gatewayctx.GatewayContext, statusCode int, responseHeaders http.Header, responseBody []byte, streamStarted bool) {
	upstreamMsg := service.ExtractUpstreamErrorMessage(responseBody)
	service.SetOpsUpstreamErrorContext(c, statusCode, upstreamMsg, "")

	status, errType, errMsg := h.mapUpstreamError(statusCode, responseHeaders, responseBody)
	h.handleStreamingAwareErrorContext(c, status, errType, errMsg, streamStarted)
}

func (h *SoraGatewayHandler) mapUpstreamError(statusCode int, responseHeaders http.Header, responseBody []byte) (int, string, string) {
	if isSoraCloudflareChallengeResponse(statusCode, responseHeaders, responseBody) {
		baseMsg := fmt.Sprintf("Sora request blocked by Cloudflare challenge (HTTP %d). Please switch to a clean proxy/network and retry.", statusCode)
		return http.StatusBadGateway, "upstream_error", formatSoraCloudflareChallengeMessage(baseMsg, responseHeaders, responseBody)
	}

	upstreamCode, upstreamMessage := extractUpstreamErrorCodeAndMessage(responseBody)
	if strings.EqualFold(upstreamCode, "cf_shield_429") {
		baseMsg := "Sora request blocked by Cloudflare shield (429). Please switch to a clean proxy/network and retry."
		return http.StatusTooManyRequests, "rate_limit_error", formatSoraCloudflareChallengeMessage(baseMsg, responseHeaders, responseBody)
	}
	if shouldPassthroughSoraUpstreamMessage(statusCode, upstreamMessage) {
		switch statusCode {
		case 401, 403, 404, 500, 502, 503, 504:
			return http.StatusBadGateway, "upstream_error", upstreamMessage
		case 429:
			return http.StatusTooManyRequests, "rate_limit_error", upstreamMessage
		}
	}

	switch statusCode {
	case 401:
		return http.StatusBadGateway, "upstream_error", "Upstream authentication failed, please contact administrator"
	case 403:
		return http.StatusBadGateway, "upstream_error", "Upstream access forbidden, please contact administrator"
	case 404:
		if strings.EqualFold(upstreamCode, "unsupported_country_code") {
			return http.StatusBadGateway, "upstream_error", "Upstream region capability unavailable for this account, please contact administrator"
		}
		return http.StatusBadGateway, "upstream_error", "Upstream capability unavailable for this account, please contact administrator"
	case 429:
		return http.StatusTooManyRequests, "rate_limit_error", "Upstream rate limit exceeded, please retry later"
	case 529:
		return http.StatusServiceUnavailable, "upstream_error", "Upstream service overloaded, please retry later"
	case 500, 502, 503, 504:
		return http.StatusBadGateway, "upstream_error", "Upstream service temporarily unavailable"
	default:
		return http.StatusBadGateway, "upstream_error", "Upstream request failed"
	}
}

func cloneHTTPHeaders(headers http.Header) http.Header {
	if headers == nil {
		return nil
	}
	return headers.Clone()
}

func extractSoraFailoverHeaderInsights(headers http.Header, body []byte) (rayID, mitigated, contentType string) {
	if headers != nil {
		mitigated = strings.TrimSpace(headers.Get("cf-mitigated"))
		contentType = strings.TrimSpace(headers.Get("content-type"))
		if contentType == "" {
			contentType = strings.TrimSpace(headers.Get("Content-Type"))
		}
	}
	rayID = soraerror.ExtractCloudflareRayID(headers, body)
	return rayID, mitigated, contentType
}

func isSoraCloudflareChallengeResponse(statusCode int, headers http.Header, body []byte) bool {
	return soraerror.IsCloudflareChallengeResponse(statusCode, headers, body)
}

func shouldPassthroughSoraUpstreamMessage(statusCode int, message string) bool {
	message = strings.TrimSpace(message)
	if message == "" {
		return false
	}
	if statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests {
		lower := strings.ToLower(message)
		if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") || strings.Contains(lower, "window._cf_chl_opt") {
			return false
		}
	}
	return true
}

func formatSoraCloudflareChallengeMessage(base string, headers http.Header, body []byte) string {
	return soraerror.FormatCloudflareChallengeMessage(base, headers, body)
}

func extractUpstreamErrorCodeAndMessage(body []byte) (string, string) {
	return soraerror.ExtractUpstreamErrorCodeAndMessage(body)
}

func (h *SoraGatewayHandler) handleStreamingAwareError(c *gin.Context, status int, errType, message string, streamStarted bool) {
	h.handleStreamingAwareErrorContext(gatewayctx.FromGin(c), status, errType, message, streamStarted)
}

func (h *SoraGatewayHandler) handleStreamingAwareErrorContext(c gatewayctx.GatewayContext, status int, errType, message string, streamStarted bool) {
	if c == nil {
		return
	}
	if streamStarted {
		_ = gatewayctx.WriteSSEEvent(c, "error", map[string]any{
			"error": map[string]string{
				"type":    errType,
				"message": message,
			},
		})
		return
	}
	h.errorResponseGateway(c, status, errType, message)
}

func (h *SoraGatewayHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	h.errorResponseGateway(gatewayctx.FromGin(c), status, errType, message)
}

func (h *SoraGatewayHandler) errorResponseGateway(c gatewayctx.GatewayContext, status int, errType, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

// MediaProxy serves local Sora media files.
func (h *SoraGatewayHandler) MediaProxy(c *gin.Context) {
	h.proxySoraMedia(c, false)
}

// MediaProxySigned serves local Sora media files with signature verification.
func (h *SoraGatewayHandler) MediaProxySigned(c *gin.Context) {
	h.proxySoraMedia(c, true)
}

func (h *SoraGatewayHandler) proxySoraMedia(c *gin.Context, requireSignature bool) {
	rawPath := c.Param("filepath")
	if rawPath == "" {
		c.Status(http.StatusNotFound)
		return
	}
	cleaned := path.Clean(rawPath)
	if !strings.HasPrefix(cleaned, "/image/") && !strings.HasPrefix(cleaned, "/video/") {
		c.Status(http.StatusNotFound)
		return
	}

	query := c.Request.URL.Query()
	if requireSignature {
		if h.soraMediaSigningKey == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"type":    "api_error",
					"message": "Sora 媒体签名未配置",
				},
			})
			return
		}
		expiresStr := strings.TrimSpace(query.Get("expires"))
		signature := strings.TrimSpace(query.Get("sig"))
		expires, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil || expires <= time.Now().Unix() {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"type":    "authentication_error",
					"message": "Sora 媒体签名已过期",
				},
			})
			return
		}
		query.Del("sig")
		query.Del("expires")
		signingQuery := query.Encode()
		if !service.VerifySoraMediaURL(cleaned, signingQuery, expires, signature, h.soraMediaSigningKey) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"type":    "authentication_error",
					"message": "Sora 媒体签名无效",
				},
			})
			return
		}
	}
	if strings.TrimSpace(h.soraMediaRoot) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"type":    "api_error",
				"message": "Sora 媒体目录未配置",
			},
		})
		return
	}

	relative := strings.TrimPrefix(cleaned, "/")
	localPath := filepath.Join(h.soraMediaRoot, filepath.FromSlash(relative))
	if _, err := os.Stat(localPath); err != nil {
		if os.IsNotExist(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	c.File(localPath)
}
