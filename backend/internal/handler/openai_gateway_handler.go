package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	rustsidecar "github.com/Wei-Shaw/sub2api/internal/rustbridge/sidecar"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// OpenAIGatewayHandler handles OpenAI API gateway requests
type OpenAIGatewayHandler struct {
	gatewayService          *service.OpenAIGatewayService
	billingCacheService     *service.BillingCacheService
	apiKeyService           *service.APIKeyService
	usageRecordWorkerPool   *service.UsageRecordWorkerPool
	errorPassthroughService *service.ErrorPassthroughService
	concurrencyHelper       *ConcurrencyHelper
	maxAccountSwitches      int
	cfg                     *config.Config
}

const (
	openAIStreamLargeBodyThresholdBytes    = 64 * 1024
	openAIStreamXLargeBodyThresholdBytes   = 256 * 1024
	openAIStreamHugeBodyThresholdBytes     = 1024 * 1024
	openAIStreamProxyFailureCooldown       = 20 * time.Minute
	openAIStreamAccountNetworkCooldown     = 12 * time.Minute
	openAIStreamLongContextTimeoutCooldown = 6 * time.Minute
)

type openAIFailoverPolicy struct {
	MaxSwitches           int
	AllowSameAccountRetry bool
}

func applyOpenAIRemoteCompactFailoverPolicy(policy openAIFailoverPolicy, remoteCompact bool) openAIFailoverPolicy {
	if !remoteCompact {
		return policy
	}
	policy.MaxSwitches = 0
	policy.AllowSameAccountRetry = false
	return policy
}

func shouldRetryOpenAIRemoteCompactSilently(
	failoverErr *service.UpstreamFailoverError,
	previousResponseID string,
	switchCount int,
) bool {
	if failoverErr == nil || switchCount > 0 {
		return false
	}
	// previous_response_id binds server-side history to one account/session;
	// retrying on another account is unlikely to be seamless.
	if strings.TrimSpace(previousResponseID) != "" {
		return false
	}
	switch failoverErr.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, 554:
		return false
	}
	reason := strings.ToLower(strings.TrimSpace(failoverErr.TempUnscheduleReason))
	bodyText := strings.ToLower(strings.TrimSpace(string(failoverErr.ResponseBody)))
	if failoverErr.FailedProxyID > 0 ||
		strings.Contains(reason, "proxy") ||
		strings.Contains(reason, "network") ||
		strings.Contains(reason, "timeout") ||
		strings.Contains(bodyText, "context deadline exceeded") ||
		strings.Contains(bodyText, "timeout") ||
		strings.Contains(bodyText, "connection refused") ||
		strings.Contains(bodyText, "connection reset") ||
		strings.Contains(bodyText, "socks connect") ||
		strings.Contains(bodyText, "eof") {
		return true
	}
	switch failoverErr.StatusCode {
	case 0, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func resolveOpenAIForwardDefaultMappedModel(apiKey *service.APIKey, fallbackModel string) string {
	if fallbackModel = strings.TrimSpace(fallbackModel); fallbackModel != "" {
		return fallbackModel
	}
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return strings.TrimSpace(apiKey.Group.DefaultMappedModel)
}

func resolveOpenAISelectionFallbackModel(apiKey *service.APIKey, requestedModel string) string {
	fallbackModel := resolveOpenAIForwardDefaultMappedModel(apiKey, "")
	if fallbackModel == "" || strings.EqualFold(strings.TrimSpace(fallbackModel), strings.TrimSpace(requestedModel)) {
		return ""
	}
	return fallbackModel
}

func handleOpenAISelectionExhausted(
	ctx context.Context,
	failedAccountIDs map[int64]struct{},
	lastFailoverErr *service.UpstreamFailoverError,
	switchCount int,
	maxSwitches int,
) (map[int64]struct{}, FailoverAction) {
	fs := NewFailoverState(maxSwitches, false)
	fs.FailedAccountIDs = failedAccountIDs
	fs.LastFailoverErr = lastFailoverErr
	fs.SwitchCount = switchCount
	return fs.FailedAccountIDs, fs.HandleSelectionExhausted(ctx)
}

func clampOpenAIMaxSwitches(limit, cap int) int {
	if limit < 0 {
		limit = 0
	}
	if cap < 0 {
		cap = 0
	}
	if limit == 0 || cap == 0 {
		return 0
	}
	if limit < cap {
		return limit
	}
	return cap
}

func deriveOpenAIStreamingFallbackSeed(c *gin.Context, apiKey *service.APIKey, userID int64, reqModel string) string {
	return deriveOpenAIStreamingFallbackSeedContext(gatewayctx.FromGin(c), apiKey, userID, reqModel)
}

func deriveOpenAIStreamingFallbackSeedContext(ctx gatewayctx.GatewayContext, apiKey *service.APIKey, userID int64, reqModel string) string {
	if apiKey == nil || userID <= 0 {
		return ""
	}

	groupID := int64(0)
	if apiKey.GroupID != nil {
		groupID = *apiKey.GroupID
	}
	clientIP := strings.TrimSpace(ctx.ClientIP())

	return fmt.Sprintf(
		"openai_http_stream:%d:%d:%d:%s:%s:%s",
		userID,
		apiKey.ID,
		groupID,
		strings.TrimSpace(reqModel),
		clientIP,
		strings.TrimSpace(ctx.HeaderValue("User-Agent")),
	)
}

func (h *OpenAIGatewayHandler) resolveOpenAIStickySessionHash(
	c *gin.Context,
	apiKey *service.APIKey,
	userID int64,
	reqModel string,
	reqStream bool,
	body []byte,
) string {
	return h.resolveOpenAIStickySessionHashContext(gatewayctx.FromGin(c), apiKey, userID, reqModel, reqStream, body)
}

func (h *OpenAIGatewayHandler) resolveOpenAIStickySessionHashContext(
	ctx gatewayctx.GatewayContext,
	apiKey *service.APIKey,
	userID int64,
	reqModel string,
	reqStream bool,
	body []byte,
) string {
	if h == nil || h.gatewayService == nil {
		return ""
	}
	if !reqStream {
		return h.gatewayService.GenerateSessionHashContext(ctx, body)
	}

	seed := deriveOpenAIStreamingFallbackSeedContext(ctx, apiKey, userID, reqModel)
	return h.gatewayService.GenerateSessionHashWithFallbackContext(ctx, body, seed)
}

func (h *OpenAIGatewayHandler) resolveOpenAIFailoverPolicy(body []byte, reqStream bool) openAIFailoverPolicy {
	limit := h.maxAccountSwitches
	policy := openAIFailoverPolicy{
		MaxSwitches:           limit,
		AllowSameAccountRetry: true,
	}
	if !reqStream {
		return policy
	}

	large, xlarge, huge := h.openAIStreamingBodyThresholds()
	bodySize := len(body)
	switch {
	case huge > 0 && bodySize >= huge:
		policy.MaxSwitches = clampOpenAIMaxSwitches(limit, 0)
		policy.AllowSameAccountRetry = false
	case xlarge > 0 && bodySize >= xlarge:
		policy.MaxSwitches = clampOpenAIMaxSwitches(limit, 1)
		policy.AllowSameAccountRetry = false
	case large > 0 && bodySize >= large:
		policy.MaxSwitches = clampOpenAIMaxSwitches(limit, 2)
		policy.AllowSameAccountRetry = false
	}
	return policy
}

func adjustOpenAIFailoverForLargeStreamingRequest(
	failoverErr *service.UpstreamFailoverError,
	body []byte,
	reqStream bool,
	cfg *config.Config,
) *service.UpstreamFailoverError {
	if failoverErr == nil || !reqStream {
		return failoverErr
	}
	large, xlarge, huge := resolveOpenAIStreamingThresholds(cfg)
	bodySize := len(body)
	if large <= 0 || bodySize < large {
		return failoverErr
	}

	adjusted := *failoverErr
	adjusted.RetryableOnSameAccount = false
	if adjusted.TempUnscheduleFor <= 0 {
		switch {
		case huge > 0 && bodySize >= huge:
			adjusted.TempUnscheduleFor = openAIStreamLongContextTimeoutCooldown
		case xlarge > 0 && bodySize >= xlarge:
			adjusted.TempUnscheduleFor = 4 * time.Minute
		default:
			adjusted.TempUnscheduleFor = 2 * time.Minute
		}
		if strings.TrimSpace(adjusted.TempUnscheduleReason) == "" {
			adjusted.TempUnscheduleReason = "openai streaming long-context failover cooldown"
		}
	}
	return &adjusted
}

func resolveOpenAIStreamingThresholds(cfg *config.Config) (int, int, int) {
	large := openAIStreamLargeBodyThresholdBytes
	xlarge := openAIStreamXLargeBodyThresholdBytes
	huge := openAIStreamHugeBodyThresholdBytes
	if cfg == nil {
		return large, xlarge, huge
	}
	streaming := cfg.Gateway.OpenAI.Streaming
	if streaming.LargeBodyThresholdBytes > 0 {
		large = streaming.LargeBodyThresholdBytes
	}
	if streaming.XLargeBodyThresholdBytes > 0 {
		xlarge = streaming.XLargeBodyThresholdBytes
	}
	if streaming.HugeBodyThresholdBytes > 0 {
		huge = streaming.HugeBodyThresholdBytes
	}
	return large, xlarge, huge
}

func (h *OpenAIGatewayHandler) openAIStreamingBodyThresholds() (int, int, int) {
	if h == nil {
		return resolveOpenAIStreamingThresholds(nil)
	}
	return resolveOpenAIStreamingThresholds(h.cfg)
}

func adjustOpenAINetworkFailoverCooldown(
	failoverErr *service.UpstreamFailoverError,
) *service.UpstreamFailoverError {
	if failoverErr == nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(failoverErr.Error())), "failover") {
		return failoverErr
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(failoverErr.TempUnscheduleReason)), "proxy") &&
		!strings.Contains(strings.ToLower(strings.TrimSpace(failoverErr.TempUnscheduleReason)), "network") {
		return failoverErr
	}
	adjusted := *failoverErr
	if adjusted.FailedProxyID > 0 || strings.TrimSpace(adjusted.FailedProxyURL) != "" {
		adjusted.TempUnscheduleFor = openAIStreamProxyFailureCooldown
		adjusted.TempUnscheduleReason = "upstream request failed via proxy/network (auto temp-unschedule 20m)"
		return &adjusted
	}
	if adjusted.TempUnscheduleFor < openAIStreamAccountNetworkCooldown {
		adjusted.TempUnscheduleFor = openAIStreamAccountNetworkCooldown
		adjusted.TempUnscheduleReason = "upstream request failed via network (auto temp-unschedule 12m)"
	}
	return &adjusted
}

// NewOpenAIGatewayHandler creates a new OpenAIGatewayHandler
func NewOpenAIGatewayHandler(
	gatewayService *service.OpenAIGatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	cfg *config.Config,
) *OpenAIGatewayHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 3
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
	}
	return &OpenAIGatewayHandler{
		gatewayService:          gatewayService,
		billingCacheService:     billingCacheService,
		apiKeyService:           apiKeyService,
		usageRecordWorkerPool:   usageRecordWorkerPool,
		errorPassthroughService: errorPassthroughService,
		concurrencyHelper:       NewConcurrencyHelper(concurrencyService, SSEPingFormatComment, pingInterval),
		maxAccountSwitches:      maxAccountSwitches,
		cfg:                     cfg,
	}
}

// Responses handles OpenAI Responses API endpoint
// POST /openai/v1/responses
func (h *OpenAIGatewayHandler) Responses(c *gin.Context) {
	h.ResponsesGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) ResponsesGateway(transportCtx gatewayctx.GatewayContext) {
	// 局部兜底：确保该 handler 内部任何 panic 都不会击穿到进程级。
	streamStarted := false
	defer h.recoverResponsesPanicContext(transportCtx, &streamStarted)
	compactStartedAt := time.Now()
	defer h.logOpenAIRemoteCompactOutcomeContext(transportCtx, compactStartedAt)
	remoteCompact := isOpenAIRemoteCompactPathContext(transportCtx)
	setOpenAIClientTransportHTTPGateway(transportCtx)

	requestStart := time.Now()

	// Get apiKey and user from context (set by ApiKeyAuth middleware)
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
		"handler.openai_gateway.responses",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependenciesGateway(transportCtx, reqLog) {
		return
	}

	// Read request body
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
	sessionHashBody := body
	if service.IsOpenAIResponsesCompactPathForTestContext(transportCtx) {
		if compactSeed := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); compactSeed != "" {
			transportCtx.SetValue(service.OpenAICompactSessionSeedKeyForTest(), compactSeed)
		}
		normalizedCompactBody, normalizedCompact, compactErr := service.NormalizeOpenAICompactRequestBodyForTest(body)
		if compactErr != nil {
			h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to normalize compact request body")
			return
		}
		if normalizedCompact {
			body = normalizedCompactBody
		}
	}

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

	streamResult := gjson.GetBytes(body, "stream")
	if streamResult.Exists() && streamResult.Type != gjson.True && streamResult.Type != gjson.False {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "invalid stream field type")
		return
	}
	reqStream := streamResult.Bool()
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))
	previousResponseID := strings.TrimSpace(gjson.GetBytes(body, "previous_response_id").String())
	if previousResponseID != "" {
		previousResponseIDKind := service.ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
		reqLog = reqLog.With(
			zap.Bool("has_previous_response_id", true),
			zap.String("previous_response_id_kind", previousResponseIDKind),
			zap.Int("previous_response_id_len", len(previousResponseID)),
		)
		if previousResponseIDKind == service.OpenAIPreviousResponseIDKindMessageID {
			reqLog.Warn("openai.request_validation_failed",
				zap.String("reason", "previous_response_id_looks_like_message_id"),
			)
			h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "previous_response_id must be a response.id (resp_*), not a message id")
			return
		}
	}

	setOpsRequestContextGateway(transportCtx, reqModel, reqStream, body)
	setOpsEndpointContextGateway(transportCtx, "", int16(service.RequestTypeFromLegacy(reqStream, false)))

	// 提前校验 function_call_output 是否具备可关联上下文，避免上游 400。
	if !h.validateFunctionCallOutputRequestContext(transportCtx, body, reqLog) {
		return
	}

	// 绑定错误透传服务，允许 service 层在非 failover 错误场景复用规则。
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughServiceContext(transportCtx, h.errorPassthroughService)
	}

	// Get subscription info (may be nil)
	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(transportCtx)

	service.SetOpsLatencyMsContext(transportCtx, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlotContext(transportCtx, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	// 确保请求取消时也会释放槽位，避免长连接被动中断造成泄漏
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	// 2. Re-check billing eligibility after wait
	if err := h.billingCacheService.CheckBillingEligibility(transportCtx.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("openai.billing_eligibility_check_failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.handleStreamingAwareErrorContext(transportCtx, status, code, message, streamStarted)
		return
	}

	// Generate sticky session hash. For streaming requests, fall back to a stable
	// per-user/per-key seed when the client did not send session headers.
	sessionHash := h.resolveOpenAIStickySessionHashContext(transportCtx, apiKey, subject.UserID, reqModel, reqStream, sessionHashBody)

	failoverPolicy := h.resolveOpenAIFailoverPolicy(body, reqStream)
	failoverPolicy = applyOpenAIRemoteCompactFailoverPolicy(failoverPolicy, remoteCompact)
	maxAccountSwitches := failoverPolicy.MaxSwitches
	if maxAccountSwitches != h.maxAccountSwitches || !failoverPolicy.AllowSameAccountRetry {
		reqLog.Info("openai.streaming_failover_policy_applied",
			zap.Int("body_bytes", len(body)),
			zap.Bool("stream", reqStream),
			zap.Bool("remote_compact", remoteCompact),
			zap.Int("effective_max_switches", maxAccountSwitches),
			zap.Bool("allow_same_account_retry", failoverPolicy.AllowSameAccountRetry),
		)
	}
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		transportCtx.SetValue("openai_responses_fallback_model", "")
		// Select account supporting the requested model
		reqLog.Debug("openai.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		groupSelection, err := selectOpenAIAPIKeyGroup(
			transportCtx.Context(),
			apiKey,
			previousResponseID,
			sessionHash,
			reqModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
			h.gatewayService.SelectAccountWithScheduler,
		)
		var (
			selection        *service.AccountSelectionResult
			scheduleDecision service.OpenAIAccountScheduleDecision
			selectedAPIKey   = apiKey
		)
		if err == nil && groupSelection != nil {
			selection = groupSelection.Selection
			scheduleDecision = groupSelection.Decision
			selectedAPIKey = groupSelection.APIKey
		}
		if err != nil {
			reqLog.Warn("openai.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				fallbackModel := resolveOpenAISelectionFallbackModel(apiKey, reqModel)
				if fallbackModel != "" {
					reqLog.Info("openai.fallback_to_default_model",
						zap.String("default_mapped_model", fallbackModel),
					)
					groupSelection, err = selectOpenAIAPIKeyGroup(
						transportCtx.Context(),
						apiKey,
						previousResponseID,
						sessionHash,
						fallbackModel,
						failedAccountIDs,
						service.OpenAIUpstreamTransportAny,
						h.gatewayService.SelectAccountWithScheduler,
					)
					if err == nil && groupSelection != nil {
						selection = groupSelection.Selection
						scheduleDecision = groupSelection.Decision
						selectedAPIKey = groupSelection.APIKey
						transportCtx.SetValue("openai_responses_fallback_model", fallbackModel)
					}
				}
				if err != nil {
					h.handleStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable", streamStarted)
					return
				}
			} else {
				var action FailoverAction
				failedAccountIDs, action = handleOpenAISelectionExhausted(
					transportCtx.Context(),
					failedAccountIDs,
					lastFailoverErr,
					switchCount,
					maxAccountSwitches,
				)
				switch action {
				case FailoverContinue:
					continue
				case FailoverCanceled:
					return
				default:
					if lastFailoverErr != nil {
						h.handleFailoverExhaustedContext(transportCtx, lastFailoverErr, streamStarted)
					} else {
						h.handleFailoverExhaustedSimpleContext(transportCtx, 502, streamStarted)
					}
					return
				}
			}
		}
		if selection == nil || selection.Account == nil {
			h.handleStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
			return
		}
		if previousResponseID != "" && selection != nil && selection.Account != nil {
			reqLog.Debug("openai.account_selected_with_previous_response_id", zap.Int64("account_id", selection.Account.ID))
		}
		reqLog.Debug("openai.account_schedule_decision",
			zap.String("layer", scheduleDecision.Layer),
			zap.Bool("sticky_previous_hit", scheduleDecision.StickyPreviousHit),
			zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
			zap.Int("candidate_count", scheduleDecision.CandidateCount),
			zap.Int("top_k", scheduleDecision.TopK),
			zap.Int64("latency_ms", scheduleDecision.LatencyMs),
			zap.Float64("load_skew", scheduleDecision.LoadSkew),
		)
		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		setOpsSelectedAccountGateway(transportCtx, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlotContext(transportCtx, selectedAPIKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		// Forward request
		service.SetOpsLatencyMsContext(transportCtx, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		defaultMappedModel := resolveOpenAIForwardDefaultMappedModel(selectedAPIKey, getContextStringGateway(transportCtx, "openai_responses_fallback_model"))
		result, err := h.gatewayService.ForwardContext(transportCtx.Context(), transportCtx, account, body, defaultMappedModel)
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64Gateway(transportCtx, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMsContext(transportCtx, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMsContext(transportCtx, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failoverErr = adjustOpenAINetworkFailoverCooldown(failoverErr)
				failoverErr = adjustOpenAIFailoverForLargeStreamingRequest(failoverErr, body, reqStream, h.cfg)
				h.gatewayService.RegisterOpenAIRuntimeFailure(account, failoverErr)
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				if remoteCompact {
					if failoverErr.TempUnscheduleFor > 0 || failoverErr.StatusCode == http.StatusUnauthorized {
						_ = h.gatewayService.ClearStickySessionForAccount(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID)
						h.gatewayService.TempUnscheduleRetryableError(transportCtx.Context(), account.ID, failoverErr)
					}
					if shouldRetryOpenAIRemoteCompactSilently(failoverErr, previousResponseID, switchCount) {
						h.gatewayService.RecordOpenAIAccountSwitch()
						failedAccountIDs[account.ID] = struct{}{}
						lastFailoverErr = failoverErr
						switchCount++
						reqLog.Warn("openai.remote_compact_retrying",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("switch_count", switchCount),
							zap.String("reason", strings.TrimSpace(failoverErr.TempUnscheduleReason)),
						)
						continue
					}
					h.handleRemoteCompactFailureContext(transportCtx, failoverErr, streamStarted)
					return
				}
				// 池模式：同账号重试
				if failoverPolicy.AllowSameAccountRetry && failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-transportCtx.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				_ = h.gatewayService.ClearStickySessionForAccount(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID)
				h.gatewayService.TempUnscheduleRetryableError(transportCtx.Context(), account.ID, failoverErr)
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhaustedContext(transportCtx, failoverErr, streamStarted)
					return
				}
				switchCount++
				reqLog.Warn("openai.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureForwardErrorResponseContext(transportCtx, streamStarted)
			fields := []zap.Field{
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			}
			if shouldLogOpenAIForwardFailureAsWarnContext(transportCtx, wroteFallback) {
				reqLog.Warn("openai.forward_failed", fields...)
				return
			}
			reqLog.Error("openai.forward_failed", fields...)
			return
		}
		if result != nil {
			if account.Type == service.AccountTypeOAuth {
				h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(transportCtx.Context(), account.ID, result.ResponseHeaders)
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}
		h.gatewayService.MarkOpenAIAccountHealthy(account)
		if err := h.gatewayService.PromoteStickySession(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai.promote_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}

		// 捕获请求信息（用于异步记录，避免在 goroutine 中访问 gin.Context）
		userAgent := strings.TrimSpace(transportCtx.HeaderValue("User-Agent"))
		clientIP := strings.TrimSpace(transportCtx.ClientIP())
		requestPayloadHash := service.HashUsageRequestPayload(body)

		// 使用量记录通过有界 worker 池提交，避免请求热路径创建无界 goroutine。
		h.submitUsageRecordTask(func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             selectedAPIKey,
				User:               selectedAPIKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    GetInboundEndpointContext(transportCtx),
				UpstreamEndpoint:   GetUpstreamEndpointContext(transportCtx, account.Platform),
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.responses"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", selectedAPIKey.ID),
					zap.Any("group_id", selectedAPIKey.GroupID),
					zap.String("model", reqModel),
					zap.Int64("account_id", account.ID),
				).Error("openai.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("openai.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func isOpenAIRemoteCompactPath(c *gin.Context) bool {
	return isOpenAIRemoteCompactPathContext(gatewayctx.FromGin(c))
}

func isOpenAIRemoteCompactPathContext(c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil || c.Request().URL == nil {
		return false
	}
	normalizedPath := strings.TrimRight(strings.TrimSpace(c.Request().URL.Path), "/")
	return strings.HasSuffix(normalizedPath, "/responses/compact")
}

func (h *OpenAIGatewayHandler) logOpenAIRemoteCompactOutcome(c *gin.Context, startedAt time.Time) {
	h.logOpenAIRemoteCompactOutcomeContext(gatewayctx.FromGin(c), startedAt)
}

func (h *OpenAIGatewayHandler) logOpenAIRemoteCompactOutcomeContext(c gatewayctx.GatewayContext, startedAt time.Time) {
	if !isOpenAIRemoteCompactPathContext(c) {
		return
	}

	var (
		ctx    = context.Background()
		path   string
		status int
	)
	if c != nil {
		if req := c.Request(); req != nil {
			ctx = req.Context()
			if req.URL != nil {
				path = strings.TrimSpace(req.URL.Path)
			}
		}
		status = http.StatusOK
		if c.ResponseWritten() {
			if raw, ok := c.Value(service.OpsUpstreamStatusCodeKey); ok {
				switch typed := raw.(type) {
				case int:
					status = typed
				case int64:
					status = int(typed)
				}
			}
		}
	}

	outcome := "failed"
	if status >= 200 && status < 300 {
		outcome = "succeeded"
	}
	latencyMs := time.Since(startedAt).Milliseconds()
	if latencyMs < 0 {
		latencyMs = 0
	}

	fields := []zap.Field{
		zap.String("component", "handler.openai_gateway.responses"),
		zap.Bool("remote_compact", true),
		zap.String("compact_outcome", outcome),
		zap.Int("status_code", status),
		zap.Int64("latency_ms", latencyMs),
		zap.String("path", path),
		zap.Bool("force_codex_cli", h != nil && h.cfg != nil && h.cfg.Gateway.ForceCodexCLI),
	}

	if c != nil {
		if userAgent := strings.TrimSpace(c.HeaderValue("User-Agent")); userAgent != "" {
			fields = append(fields, zap.String("request_user_agent", userAgent))
		}
		if v, ok := c.Value(opsModelKey); ok {
			if model, ok := v.(string); ok && strings.TrimSpace(model) != "" {
				fields = append(fields, zap.String("request_model", strings.TrimSpace(model)))
			}
		}
		if v, ok := c.Value(opsAccountIDKey); ok {
			if accountID, ok := v.(int64); ok && accountID > 0 {
				fields = append(fields, zap.Int64("account_id", accountID))
			}
		}
		if upstreamRequestID := strings.TrimSpace(c.HeaderValue("x-request-id")); upstreamRequestID != "" {
			fields = append(fields, zap.String("upstream_request_id", upstreamRequestID))
		} else if upstreamRequestID := strings.TrimSpace(c.HeaderValue("X-Request-Id")); upstreamRequestID != "" {
			fields = append(fields, zap.String("upstream_request_id", upstreamRequestID))
		}
	}

	log := logger.FromContext(ctx).With(fields...)
	if outcome == "succeeded" {
		log.Info("codex.remote_compact.succeeded")
		return
	}
	log.Warn("codex.remote_compact.failed")
}

// Messages handles Anthropic Messages API requests routed to OpenAI platform.
// POST /v1/messages (when group platform is OpenAI)
func (h *OpenAIGatewayHandler) Messages(c *gin.Context) {
	h.MessagesGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) MessagesGateway(transportCtx gatewayctx.GatewayContext) {
	streamStarted := false
	defer h.recoverAnthropicMessagesPanicContext(transportCtx, &streamStarted)

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromGatewayContext(transportCtx)
	if !ok {
		h.anthropicErrorResponseContext(transportCtx, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(transportCtx)
	if !ok {
		h.anthropicErrorResponseContext(transportCtx, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLoggerContext(
		transportCtx,
		"handler.openai_gateway.messages",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	// 检查分组是否允许 /v1/messages 调度
	if apiKey.Group != nil && !apiKey.Group.AllowMessagesDispatch {
		h.anthropicErrorResponseContext(transportCtx, http.StatusForbidden, "permission_error",
			"This group does not allow /v1/messages dispatch")
		return
	}

	if !h.ensureResponsesDependenciesGateway(transportCtx, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(transportCtx.Request())
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.anthropicErrorResponseContext(transportCtx, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	if !gjson.ValidBytes(body) {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}

	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()
	reqStream := gjson.GetBytes(body, "stream").Bool()

	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))

	setOpsRequestContextGateway(transportCtx, reqModel, reqStream, body)
	setOpsEndpointContextGateway(transportCtx, "", int16(service.RequestTypeFromLegacy(reqStream, false)))

	// 绑定错误透传服务，允许 service 层在非 failover 错误场景复用规则。
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughServiceContext(transportCtx, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(transportCtx)

	service.SetOpsLatencyMsContext(transportCtx, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlotContext(transportCtx, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(transportCtx.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("openai_messages.billing_eligibility_check_failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.anthropicStreamingAwareErrorContext(transportCtx, status, code, message, streamStarted)
		return
	}

	sessionHash := h.resolveOpenAIStickySessionHashContext(transportCtx, apiKey, subject.UserID, reqModel, reqStream, body)
	promptCacheKey := h.gatewayService.ExtractSessionIDContext(transportCtx, body)

	// Anthropic 格式的请求在 metadata.user_id 中携带 session 标识，
	// 而非 OpenAI 的 session_id/conversation_id headers。
	// 从中派生 sessionHash（sticky session）和 promptCacheKey（upstream cache）。
	if sessionHash == "" || promptCacheKey == "" {
		if userID := strings.TrimSpace(gjson.GetBytes(body, "metadata.user_id").String()); userID != "" {
			seed := reqModel + "-" + userID
			if promptCacheKey == "" {
				promptCacheKey = service.GenerateSessionUUID(seed)
			}
			if sessionHash == "" {
				sessionHash = service.DeriveSessionHashFromSeed(seed)
			}
		}
	}

	failoverPolicy := h.resolveOpenAIFailoverPolicy(body, reqStream)
	maxAccountSwitches := failoverPolicy.MaxSwitches
	if maxAccountSwitches != h.maxAccountSwitches || !failoverPolicy.AllowSameAccountRetry {
		reqLog.Info("openai_messages.streaming_failover_policy_applied",
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
		// 清除上一次迭代的降级模型标记，避免残留影响本次迭代
		transportCtx.SetValue("openai_messages_fallback_model", "")
		reqLog.Debug("openai_messages.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		groupSelection, err := selectOpenAIAPIKeyGroup(
			transportCtx.Context(),
			apiKey,
			"",
			sessionHash,
			reqModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
			h.gatewayService.SelectAccountWithScheduler,
		)
		var (
			selection        *service.AccountSelectionResult
			scheduleDecision service.OpenAIAccountScheduleDecision
			selectedAPIKey   = apiKey
		)
		if err == nil && groupSelection != nil {
			selection = groupSelection.Selection
			scheduleDecision = groupSelection.Decision
			selectedAPIKey = groupSelection.APIKey
		}
		if err != nil {
			reqLog.Warn("openai_messages.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			// 首次调度失败 + 有默认映射模型 → 用默认模型重试
			if len(failedAccountIDs) == 0 {
				defaultModel := resolveOpenAISelectionFallbackModel(apiKey, reqModel)
				if defaultModel != "" && defaultModel != reqModel {
					reqLog.Info("openai_messages.fallback_to_default_model",
						zap.String("default_mapped_model", defaultModel),
					)
					groupSelection, err = selectOpenAIAPIKeyGroup(
						transportCtx.Context(),
						apiKey,
						"",
						sessionHash,
						defaultModel,
						failedAccountIDs,
						service.OpenAIUpstreamTransportAny,
						h.gatewayService.SelectAccountWithScheduler,
					)
					if err == nil && groupSelection != nil {
						selection = groupSelection.Selection
						scheduleDecision = groupSelection.Decision
						selectedAPIKey = groupSelection.APIKey
						transportCtx.SetValue("openai_messages_fallback_model", defaultModel)
					}
				}
				if err != nil {
					h.anthropicStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable", streamStarted)
					return
				}
			} else {
				var action FailoverAction
				failedAccountIDs, action = handleOpenAISelectionExhausted(
					transportCtx.Context(),
					failedAccountIDs,
					lastFailoverErr,
					switchCount,
					maxAccountSwitches,
				)
				switch action {
				case FailoverContinue:
					continue
				case FailoverCanceled:
					return
				default:
					if lastFailoverErr != nil {
						h.handleAnthropicFailoverExhaustedContext(transportCtx, lastFailoverErr, streamStarted)
					} else {
						h.anthropicStreamingAwareErrorContext(transportCtx, http.StatusBadGateway, "api_error", "Upstream request failed", streamStarted)
					}
					return
				}
			}
		}
		if selection == nil || selection.Account == nil {
			h.anthropicStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
			return
		}
		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai_messages.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		_ = scheduleDecision
		setOpsSelectedAccountGateway(transportCtx, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlotContext(transportCtx, selectedAPIKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMsContext(transportCtx, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()

		// Forward 层需要始终拿到 group 默认映射模型，这样未命中账号级映射的
		// Claude 兼容模型才不会在后续 Codex 规范化中意外退化到 gpt-5.1。
		defaultMappedModel := resolveOpenAIForwardDefaultMappedModel(selectedAPIKey, getContextStringGateway(transportCtx, "openai_messages_fallback_model"))
		result, err := h.gatewayService.ForwardAsAnthropicContext(transportCtx.Context(), transportCtx, account, body, promptCacheKey, defaultMappedModel)

		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64Gateway(transportCtx, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMsContext(transportCtx, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMsContext(transportCtx, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failoverErr = adjustOpenAINetworkFailoverCooldown(failoverErr)
				failoverErr = adjustOpenAIFailoverForLargeStreamingRequest(failoverErr, body, reqStream, h.cfg)
				h.gatewayService.RegisterOpenAIRuntimeFailure(account, failoverErr)
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				// 池模式：同账号重试
				if failoverPolicy.AllowSameAccountRetry && failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai_messages.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-transportCtx.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				_ = h.gatewayService.ClearStickySessionForAccount(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID)
				h.gatewayService.TempUnscheduleRetryableError(transportCtx.Context(), account.ID, failoverErr)
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleAnthropicFailoverExhaustedContext(transportCtx, failoverErr, streamStarted)
					return
				}
				switchCount++
				reqLog.Warn("openai_messages.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureAnthropicErrorResponseContext(transportCtx, streamStarted)
			reqLog.Warn("openai_messages.forward_failed",
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
		if err := h.gatewayService.PromoteStickySession(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai_messages.promote_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}

		userAgent := strings.TrimSpace(transportCtx.HeaderValue("User-Agent"))
		clientIP := strings.TrimSpace(transportCtx.ClientIP())
		requestPayloadHash := service.HashUsageRequestPayload(body)

		h.submitUsageRecordTask(func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             selectedAPIKey,
				User:               selectedAPIKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    GetInboundEndpointContext(transportCtx),
				UpstreamEndpoint:   GetUpstreamEndpointContext(transportCtx, account.Platform),
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.messages"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", selectedAPIKey.ID),
					zap.Any("group_id", selectedAPIKey.GroupID),
					zap.String("model", reqModel),
					zap.Int64("account_id", account.ID),
				).Error("openai_messages.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("openai_messages.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

// anthropicErrorResponse writes an error in Anthropic Messages API format.
func (h *OpenAIGatewayHandler) anthropicErrorResponse(c *gin.Context, status int, errType, message string) {
	h.anthropicErrorResponseContext(gatewayctx.FromGin(c), status, errType, message)
}

func (h *OpenAIGatewayHandler) anthropicErrorResponseContext(c gatewayctx.GatewayContext, status int, errType, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

// anthropicStreamingAwareError handles errors that may occur during streaming,
// using Anthropic SSE error format.
func (h *OpenAIGatewayHandler) anthropicStreamingAwareError(c *gin.Context, status int, errType, message string, streamStarted bool) {
	h.anthropicStreamingAwareErrorContext(gatewayctx.FromGin(c), status, errType, message, streamStarted)
}

func (h *OpenAIGatewayHandler) anthropicStreamingAwareErrorContext(c gatewayctx.GatewayContext, status int, errType, message string, streamStarted bool) {
	if c == nil {
		return
	}
	if streamStarted || service.RequestUsesAnthropicSSE(c) {
		_ = gatewayctx.WriteSSEEvent(c, "error", gin.H{
			"type": "error",
			"error": gin.H{
				"type":    errType,
				"message": message,
			},
		})
		return
	}
	h.anthropicErrorResponseContext(c, status, errType, message)
}

// handleAnthropicFailoverExhausted maps upstream failover errors to Anthropic format.
func (h *OpenAIGatewayHandler) handleAnthropicFailoverExhausted(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	h.handleAnthropicFailoverExhaustedContext(gatewayctx.FromGin(c), failoverErr, streamStarted)
}

func (h *OpenAIGatewayHandler) handleAnthropicFailoverExhaustedContext(c gatewayctx.GatewayContext, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	status, errType, errMsg := h.mapUpstreamError(failoverErr.StatusCode)
	h.anthropicStreamingAwareErrorContext(c, status, errType, errMsg, streamStarted)
}

// ensureAnthropicErrorResponse writes a fallback Anthropic error if no response was written.
func (h *OpenAIGatewayHandler) ensureAnthropicErrorResponse(c *gin.Context, streamStarted bool) bool {
	return h.ensureAnthropicErrorResponseContext(gatewayctx.FromGin(c), streamStarted)
}

func (h *OpenAIGatewayHandler) ensureAnthropicErrorResponseContext(c gatewayctx.GatewayContext, streamStarted bool) bool {
	if c == nil || service.RequestPayloadStarted(c) {
		return false
	}
	if c.ResponseWritten() && !streamStarted && !service.RequestUsesAnthropicSSE(c) && !service.RequestUsesBufferedJSON(c) {
		return false
	}
	h.anthropicStreamingAwareErrorContext(c, http.StatusBadGateway, "api_error", "Upstream request failed", streamStarted)
	return true
}

func (h *OpenAIGatewayHandler) validateFunctionCallOutputRequest(c *gin.Context, body []byte, reqLog *zap.Logger) bool {
	return h.validateFunctionCallOutputRequestContext(gatewayctx.FromGin(c), body, reqLog)
}

func (h *OpenAIGatewayHandler) validateFunctionCallOutputRequestContext(c gatewayctx.GatewayContext, body []byte, reqLog *zap.Logger) bool {
	if !gjson.GetBytes(body, `input.#(type=="function_call_output")`).Exists() {
		return true
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		// 保持原有容错语义：解析失败时跳过预校验，沿用后续上游校验结果。
		return true
	}

	c.SetValue(service.OpenAIParsedRequestBodyKey, reqBody)
	validation := service.ValidateFunctionCallOutputContext(reqBody)
	if !validation.HasFunctionCallOutput {
		return true
	}

	previousResponseID, _ := reqBody["previous_response_id"].(string)
	if strings.TrimSpace(previousResponseID) != "" || validation.HasToolCallContext {
		return true
	}

	if validation.HasFunctionCallOutputMissingCallID {
		reqLog.Warn("openai.request_validation_failed",
			zap.String("reason", "function_call_output_missing_call_id"),
		)
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "function_call_output requires call_id or previous_response_id; if relying on history, ensure store=true and reuse previous_response_id")
		return false
	}
	if validation.HasItemReferenceForAllCallIDs {
		return true
	}

	reqLog.Warn("openai.request_validation_failed",
		zap.String("reason", "function_call_output_missing_item_reference"),
	)
	h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "function_call_output requires item_reference ids matching each call_id, or previous_response_id/tool_call context; if relying on history, ensure store=true and reuse previous_response_id")
	return false
}

func (h *OpenAIGatewayHandler) acquireResponsesUserSlot(
	c *gin.Context,
	userID int64,
	userConcurrency int,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	return h.acquireResponsesUserSlotContext(gatewayctx.FromGin(c), userID, userConcurrency, reqStream, streamStarted, reqLog)
}

func (h *OpenAIGatewayHandler) acquireResponsesUserSlotContext(
	c gatewayctx.GatewayContext,
	userID int64,
	userConcurrency int,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	ctx := c.Context()
	userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlot(ctx, userID, userConcurrency)
	if err != nil {
		reqLog.Warn("openai.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyErrorContext(c, err, "user", *streamStarted)
		return nil, false
	}
	if userAcquired {
		return wrapReleaseOnDone(ctx, userReleaseFunc), true
	}

	maxWait := service.CalculateMaxWait(userConcurrency)
	canWait, waitErr := h.concurrencyHelper.IncrementWaitCount(ctx, userID, maxWait)
	if waitErr != nil {
		reqLog.Warn("openai.user_wait_counter_increment_failed", zap.Error(waitErr))
		// 按现有降级语义：等待计数异常时放行后续抢槽流程
	} else if !canWait {
		reqLog.Info("openai.user_wait_queue_full", zap.Int("max_wait", maxWait))
		h.errorResponseGateway(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return nil, false
	}

	waitCounted := waitErr == nil && canWait
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementWaitCount(ctx, userID)
		}
	}()

	userReleaseFunc, err = h.concurrencyHelper.AcquireUserSlotWithWaitContext(c, userID, userConcurrency, reqStream, streamStarted)
	if err != nil {
		reqLog.Warn("openai.user_slot_acquire_failed_after_wait", zap.Error(err))
		h.handleConcurrencyErrorContext(c, err, "user", *streamStarted)
		return nil, false
	}

	// 槽位获取成功后，立刻退出等待计数。
	if waitCounted {
		h.concurrencyHelper.DecrementWaitCount(ctx, userID)
		waitCounted = false
	}
	return wrapReleaseOnDone(ctx, userReleaseFunc), true
}

func (h *OpenAIGatewayHandler) acquireResponsesAccountSlot(
	c *gin.Context,
	groupID *int64,
	sessionHash string,
	selection *service.AccountSelectionResult,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	return h.acquireResponsesAccountSlotContext(gatewayctx.FromGin(c), groupID, sessionHash, selection, reqStream, streamStarted, reqLog)
}

func (h *OpenAIGatewayHandler) acquireResponsesAccountSlotContext(
	c gatewayctx.GatewayContext,
	groupID *int64,
	sessionHash string,
	selection *service.AccountSelectionResult,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	if selection == nil || selection.Account == nil {
		h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", "No available accounts", *streamStarted)
		return nil, false
	}

	ctx := c.Context()
	account := selection.Account
	if selection.Acquired {
		return wrapReleaseOnDone(ctx, selection.ReleaseFunc), true
	}
	if selection.WaitPlan == nil {
		h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", "No available accounts", *streamStarted)
		return nil, false
	}

	fastReleaseFunc, fastAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(
		ctx,
		account.ID,
		selection.WaitPlan.MaxConcurrency,
	)
	if err != nil {
		reqLog.Warn("openai.account_slot_quick_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleConcurrencyErrorContext(c, err, "account", *streamStarted)
		return nil, false
	}
	if fastAcquired {
		if err := h.gatewayService.BindStickySession(ctx, groupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}
		return wrapReleaseOnDone(ctx, fastReleaseFunc), true
	}

	canWait, waitErr := h.concurrencyHelper.IncrementAccountWaitCount(ctx, account.ID, selection.WaitPlan.MaxWaiting)
	if waitErr != nil {
		reqLog.Warn("openai.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(waitErr))
	} else if !canWait {
		reqLog.Info("openai.account_wait_queue_full",
			zap.Int64("account_id", account.ID),
			zap.Int("max_waiting", selection.WaitPlan.MaxWaiting),
		)
		h.handleStreamingAwareErrorContext(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", *streamStarted)
		return nil, false
	}

	accountWaitCounted := waitErr == nil && canWait
	releaseWait := func() {
		if accountWaitCounted {
			h.concurrencyHelper.DecrementAccountWaitCount(ctx, account.ID)
			accountWaitCounted = false
		}
	}
	defer releaseWait()

	accountReleaseFunc, err := h.concurrencyHelper.AcquireAccountSlotWithWaitTimeoutContext(
		c,
		account.ID,
		selection.WaitPlan.MaxConcurrency,
		selection.WaitPlan.Timeout,
		reqStream,
		streamStarted,
	)
	if err != nil {
		reqLog.Warn("openai.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleConcurrencyErrorContext(c, err, "account", *streamStarted)
		return nil, false
	}

	// Slot acquired: no longer waiting in queue.
	releaseWait()
	if err := h.gatewayService.BindStickySession(ctx, groupID, sessionHash, account.ID); err != nil {
		reqLog.Warn("openai.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
	}
	return wrapReleaseOnDone(ctx, accountReleaseFunc), true
}

// ResponsesWebSocket handles OpenAI Responses API WebSocket ingress endpoint
// GET /openai/v1/responses (Upgrade: websocket)
func (h *OpenAIGatewayHandler) ResponsesWebSocket(c *gin.Context) {
	h.ResponsesWebSocketGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) ResponsesWebSocketGateway(transportCtx gatewayctx.GatewayContext) {
	if transportCtx == nil {
		return
	}

	if !isOpenAIWSUpgradeRequest(transportCtx.Request()) {
		h.errorResponseGateway(transportCtx, http.StatusUpgradeRequired, "invalid_request_error", "WebSocket upgrade required (Upgrade: websocket)")
		return
	}
	setOpenAIClientTransportWSGateway(transportCtx)

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
		"handler.openai_gateway.responses_ws",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.Bool("openai_ws_mode", true),
	)
	if !h.ensureResponsesDependenciesGateway(transportCtx, reqLog) {
		return
	}
	reqLog.Info("openai.websocket_ingress_started")
	clientIP := strings.TrimSpace(transportCtx.ClientIP())
	userAgent := strings.TrimSpace(transportCtx.HeaderValue("User-Agent"))

	if handled := h.tryProxyResponsesWebSocketViaRustSidecar(transportCtx, reqLog); handled {
		return
	}

	acceptedConn, err := transportCtx.AcceptWebSocket(gatewayctx.WebSocketAcceptOptions{
		CompressionEnabled: true,
	})
	if err != nil {
		reqLog.Warn("openai.websocket_accept_failed",
			zap.Error(err),
			zap.String("client_ip", clientIP),
			zap.String("request_user_agent", userAgent),
			zap.String("upgrade_header", strings.TrimSpace(transportCtx.HeaderValue("Upgrade"))),
			zap.String("connection_header", strings.TrimSpace(transportCtx.HeaderValue("Connection"))),
			zap.String("sec_websocket_version", strings.TrimSpace(transportCtx.HeaderValue("Sec-WebSocket-Version"))),
			zap.Bool("has_sec_websocket_key", strings.TrimSpace(transportCtx.HeaderValue("Sec-WebSocket-Key")) != ""),
		)
		return
	}
	wsConn, ok := acceptedConn.Native().(*coderws.Conn)
	if !ok || wsConn == nil {
		reqLog.Warn("openai.websocket_accept_failed", zap.String("reason", "gatewayctx returned unexpected websocket implementation"))
		return
	}
	defer func() {
		_ = wsConn.CloseNow()
	}()
	wsConn.SetReadLimit(16 * 1024 * 1024)

	ctx := transportCtx.Context()
	readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	msgType, firstMessage, err := wsConn.Read(readCtx)
	cancel()
	if err != nil {
		closeStatus, closeReason := summarizeWSCloseErrorForLog(err)
		reqLog.Warn("openai.websocket_read_first_message_failed",
			zap.Error(err),
			zap.String("client_ip", clientIP),
			zap.String("close_status", closeStatus),
			zap.String("close_reason", closeReason),
			zap.Duration("read_timeout", 30*time.Second),
		)
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "missing first response.create message")
		return
	}
	if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "unsupported websocket message type")
		return
	}
	if !gjson.ValidBytes(firstMessage) {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "invalid JSON payload")
		return
	}

	reqModel := strings.TrimSpace(gjson.GetBytes(firstMessage, "model").String())
	if reqModel == "" {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "model is required in first response.create payload")
		return
	}
	previousResponseID := strings.TrimSpace(gjson.GetBytes(firstMessage, "previous_response_id").String())
	previousResponseIDKind := service.ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
	if previousResponseID != "" && previousResponseIDKind == service.OpenAIPreviousResponseIDKindMessageID {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "previous_response_id must be a response.id (resp_*), not a message id")
		return
	}
	reqLog = reqLog.With(
		zap.Bool("ws_ingress", true),
		zap.String("model", reqModel),
		zap.Bool("has_previous_response_id", previousResponseID != ""),
		zap.String("previous_response_id_kind", previousResponseIDKind),
	)
	setOpsRequestContextGateway(transportCtx, reqModel, true, firstMessage)
	setOpsEndpointContextGateway(transportCtx, "", int16(service.RequestTypeWSV2))

	var currentUserRelease func()
	var currentAccountRelease func()
	releaseTurnSlots := func() {
		if currentAccountRelease != nil {
			currentAccountRelease()
			currentAccountRelease = nil
		}
		if currentUserRelease != nil {
			currentUserRelease()
			currentUserRelease = nil
		}
	}
	// 必须尽早注册，确保任何 early return 都能释放已获取的并发槽位。
	defer releaseTurnSlots()

	userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlot(ctx, subject.UserID, subject.Concurrency)
	if err != nil {
		reqLog.Warn("openai.websocket_user_slot_acquire_failed", zap.Error(err))
		closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "failed to acquire user concurrency slot")
		return
	}
	if !userAcquired {
		closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "too many concurrent requests, please retry later")
		return
	}
	currentUserRelease = wrapReleaseOnDone(ctx, userReleaseFunc)

	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(transportCtx)
	if err := h.billingCacheService.CheckBillingEligibility(ctx, apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("openai.websocket_billing_eligibility_check_failed", zap.Error(err))
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "billing check failed")
		return
	}

	sessionHash := h.gatewayService.GenerateSessionHashWithFallbackContext(
		transportCtx,
		firstMessage,
		openAIWSIngressFallbackSessionSeed(subject.UserID, apiKey.ID, apiKey.GroupID),
	)
	groupSelection, err := selectOpenAIAPIKeyGroup(
		ctx,
		apiKey,
		previousResponseID,
		sessionHash,
		reqModel,
		nil,
		service.OpenAIUpstreamTransportResponsesWebsocketV2,
		h.gatewayService.SelectAccountWithScheduler,
	)
	if err != nil {
		reqLog.Warn("openai.websocket_account_select_failed", zap.Error(err))
		closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "no available account")
		return
	}
	selection := groupSelection.Selection
	scheduleDecision := groupSelection.Decision
	selectedAPIKey := groupSelection.APIKey
	if selection == nil || selection.Account == nil {
		closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "no available account")
		return
	}

	account := selection.Account
	accountMaxConcurrency := account.Concurrency
	if selection.WaitPlan != nil && selection.WaitPlan.MaxConcurrency > 0 {
		accountMaxConcurrency = selection.WaitPlan.MaxConcurrency
	}
	accountReleaseFunc := selection.ReleaseFunc
	if !selection.Acquired {
		if selection.WaitPlan == nil {
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "account is busy, please retry later")
			return
		}
		fastReleaseFunc, fastAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(
			ctx,
			account.ID,
			selection.WaitPlan.MaxConcurrency,
		)
		if err != nil {
			reqLog.Warn("openai.websocket_account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
			closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "failed to acquire account concurrency slot")
			return
		}
		if !fastAcquired {
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "account is busy, please retry later")
			return
		}
		accountReleaseFunc = fastReleaseFunc
	}
	currentAccountRelease = wrapReleaseOnDone(ctx, accountReleaseFunc)
	if err := h.gatewayService.BindStickySession(ctx, selectedAPIKey.GroupID, sessionHash, account.ID); err != nil {
		reqLog.Warn("openai.websocket_bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
	}

	token, _, err := h.gatewayService.GetAccessToken(ctx, account)
	if err != nil {
		reqLog.Warn("openai.websocket_get_access_token_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "failed to get access token")
		return
	}

	reqLog.Debug("openai.websocket_account_selected",
		zap.Int64("account_id", account.ID),
		zap.String("account_name", account.Name),
		zap.String("schedule_layer", scheduleDecision.Layer),
		zap.Int("candidate_count", scheduleDecision.CandidateCount),
	)

	hooks := &service.OpenAIWSIngressHooks{
		BeforeTurn: func(turn int) error {
			if turn == 1 {
				return nil
			}
			// 防御式清理：避免异常路径下旧槽位覆盖导致泄漏。
			releaseTurnSlots()
			// 非首轮 turn 需要重新抢占并发槽位，避免长连接空闲占槽。
			userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlot(ctx, subject.UserID, subject.Concurrency)
			if err != nil {
				return service.NewOpenAIWSClientCloseError(coderws.StatusInternalError, "failed to acquire user concurrency slot", err)
			}
			if !userAcquired {
				return service.NewOpenAIWSClientCloseError(coderws.StatusTryAgainLater, "too many concurrent requests, please retry later", nil)
			}
			accountReleaseFunc, accountAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(ctx, account.ID, accountMaxConcurrency)
			if err != nil {
				if userReleaseFunc != nil {
					userReleaseFunc()
				}
				return service.NewOpenAIWSClientCloseError(coderws.StatusInternalError, "failed to acquire account concurrency slot", err)
			}
			if !accountAcquired {
				if userReleaseFunc != nil {
					userReleaseFunc()
				}
				return service.NewOpenAIWSClientCloseError(coderws.StatusTryAgainLater, "account is busy, please retry later", nil)
			}
			currentUserRelease = wrapReleaseOnDone(ctx, userReleaseFunc)
			currentAccountRelease = wrapReleaseOnDone(ctx, accountReleaseFunc)
			return nil
		},
		AfterTurn: func(turn int, result *service.OpenAIForwardResult, turnErr error) {
			releaseTurnSlots()
			if turnErr != nil || result == nil {
				return
			}
			if account.Type == service.AccountTypeOAuth {
				h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(ctx, account.ID, result.ResponseHeaders)
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
			h.submitUsageRecordTask(func(taskCtx context.Context) {
				if err := h.gatewayService.RecordUsage(taskCtx, &service.OpenAIRecordUsageInput{
					Result:             result,
					APIKey:             selectedAPIKey,
					User:               selectedAPIKey.User,
					Account:            account,
					Subscription:       subscription,
					InboundEndpoint:    GetInboundEndpointContext(transportCtx),
					UpstreamEndpoint:   GetUpstreamEndpointContext(transportCtx, account.Platform),
					UserAgent:          userAgent,
					IPAddress:          clientIP,
					RequestPayloadHash: service.HashUsageRequestPayload(firstMessage),
					APIKeyService:      h.apiKeyService,
				}); err != nil {
					reqLog.Error("openai.websocket_record_usage_failed",
						zap.Int64("account_id", account.ID),
						zap.String("request_id", result.RequestID),
						zap.Error(err),
					)
				}
			})
		},
	}

	if err := h.gatewayService.ProxyResponsesWebSocketFromClient(ctx, transportCtx, wsConn, account, token, firstMessage, hooks); err != nil {
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		closeStatus, closeReason := summarizeWSCloseErrorForLog(err)
		reqLog.Warn("openai.websocket_proxy_failed",
			zap.Int64("account_id", account.ID),
			zap.Error(err),
			zap.String("close_status", closeStatus),
			zap.String("close_reason", closeReason),
		)
		var closeErr *service.OpenAIWSClientCloseError
		if errors.As(err, &closeErr) {
			closeOpenAIClientWS(wsConn, closeErr.StatusCode(), closeErr.Reason())
			return
		}
		closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "upstream websocket proxy failed")
		return
	}
	reqLog.Info("openai.websocket_ingress_closed", zap.Int64("account_id", account.ID))
}

func (h *OpenAIGatewayHandler) shouldUseRustSidecarResponsesWS(transportCtx gatewayctx.GatewayContext) bool {
	if h == nil || h.cfg == nil || transportCtx == nil {
		return false
	}
	if !h.cfg.Rust.Sidecar.Enabled || !h.cfg.Rust.Sidecar.ResponsesWSEnabled {
		return false
	}
	return !rustsidecar.HasBypassHeader(transportCtx.Request())
}

func (h *OpenAIGatewayHandler) tryProxyResponsesWebSocketViaRustSidecar(transportCtx gatewayctx.GatewayContext, reqLog *zap.Logger) bool {
	if !h.shouldUseRustSidecarResponsesWS(transportCtx) {
		return false
	}

	socketPath := strings.TrimSpace(h.cfg.Rust.Sidecar.SocketPath)
	var hijacker http.Hijacker
	switch native := transportCtx.Native().(type) {
	case *gin.Context:
		hj, ok := native.Writer.(http.Hijacker)
		if ok {
			hijacker = hj
		}
	case http.Hijacker:
		hijacker = native
	}
	if hijacker == nil {
		if reqLog != nil {
			reqLog.Warn("openai.websocket_rust_sidecar_unavailable", zap.String("reason", "no_hijacker"))
		}
		if h.cfg.Rust.Sidecar.FailClosed {
			return true
		}
		return false
	}

	if err := rustsidecar.TunnelUpgradedRequest(transportCtx.Request(), hijacker, socketPath); err != nil {
		if reqLog != nil {
			reqLog.Warn("openai.websocket_rust_sidecar_proxy_failed", zap.Error(err))
		}
		if h.cfg.Rust.Sidecar.FailClosed {
			return true
		}
		return false
	}
	if reqLog != nil {
		reqLog.Info("openai.websocket_routed_via_rust_sidecar")
	}
	return true
}

func (h *OpenAIGatewayHandler) recoverResponsesPanic(c *gin.Context, streamStarted *bool) {
	h.recoverResponsesPanicContext(gatewayctx.FromGin(c), streamStarted)
}

func (h *OpenAIGatewayHandler) recoverResponsesPanicContext(c gatewayctx.GatewayContext, streamStarted *bool) {
	recovered := recover()
	if recovered == nil {
		return
	}

	started := false
	if streamStarted != nil {
		started = *streamStarted
	}
	wroteFallback := h.ensureForwardErrorResponseContext(c, started)
	requestLoggerContext(c, "handler.openai_gateway.responses").Error(
		"openai.responses_panic_recovered",
		zap.Bool("fallback_error_response_written", wroteFallback),
		zap.Any("panic", recovered),
		zap.ByteString("stack", debug.Stack()),
	)
}

// recoverAnthropicMessagesPanic recovers from panics in the Anthropic Messages
// handler and returns an Anthropic-formatted error response.
func (h *OpenAIGatewayHandler) recoverAnthropicMessagesPanic(c *gin.Context, streamStarted *bool) {
	h.recoverAnthropicMessagesPanicContext(gatewayctx.FromGin(c), streamStarted)
}

func (h *OpenAIGatewayHandler) recoverAnthropicMessagesPanicContext(c gatewayctx.GatewayContext, streamStarted *bool) {
	recovered := recover()
	if recovered == nil {
		return
	}

	started := streamStarted != nil && *streamStarted
	requestLoggerContext(c, "handler.openai_gateway.messages").Error(
		"openai.messages_panic_recovered",
		zap.Bool("stream_started", started),
		zap.Any("panic", recovered),
		zap.ByteString("stack", debug.Stack()),
	)
	if !started {
		h.anthropicErrorResponseContext(c, http.StatusInternalServerError, "api_error", "Internal server error")
	}
}

func (h *OpenAIGatewayHandler) ensureResponsesDependencies(c *gin.Context, reqLog *zap.Logger) bool {
	missing := h.missingResponsesDependencies()
	if len(missing) == 0 {
		return true
	}

	if reqLog == nil {
		reqLog = requestLogger(c, "handler.openai_gateway.responses")
	}
	reqLog.Error("openai.handler_dependencies_missing", zap.Strings("missing_dependencies", missing))

	if c != nil && c.Writer != nil && !c.Writer.Written() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"type":    "api_error",
				"message": "Service temporarily unavailable",
			},
		})
	}
	return false
}

func (h *OpenAIGatewayHandler) missingResponsesDependencies() []string {
	missing := make([]string, 0, 5)
	if h == nil {
		return append(missing, "handler")
	}
	if h.gatewayService == nil {
		missing = append(missing, "gatewayService")
	}
	if h.billingCacheService == nil {
		missing = append(missing, "billingCacheService")
	}
	if h.apiKeyService == nil {
		missing = append(missing, "apiKeyService")
	}
	if h.concurrencyHelper == nil || h.concurrencyHelper.concurrencyService == nil {
		missing = append(missing, "concurrencyHelper")
	}
	return missing
}

func getContextInt64(c *gin.Context, key string) (int64, bool) {
	return getContextInt64Gateway(gatewayctx.FromGin(c), key)
}

func getContextInt64Gateway(c gatewayctx.GatewayContext, key string) (int64, bool) {
	if c == nil || key == "" {
		return 0, false
	}
	v, ok := c.Value(key)
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case int32:
		return int64(t), true
	case float64:
		return int64(t), true
	default:
		return 0, false
	}
}

func getContextStringGateway(c gatewayctx.GatewayContext, key string) string {
	if c == nil || key == "" {
		return ""
	}
	v, ok := c.Value(key)
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func shouldMaskOpenAIUpstreamError(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, 529, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func (h *OpenAIGatewayHandler) submitUsageRecordTask(task service.UsageRecordTask) {
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
				zap.String("component", "handler.openai_gateway.responses"),
				zap.Any("panic", recovered),
			).Error("openai.usage_record_task_panic_recovered")
		}
	}()
	task(ctx)
}

// handleConcurrencyError handles concurrency-related errors with proper 429 response
func (h *OpenAIGatewayHandler) handleConcurrencyError(c *gin.Context, err error, slotType string, streamStarted bool) {
	h.handleConcurrencyErrorContext(gatewayctx.FromGin(c), err, slotType, streamStarted)
}

func (h *OpenAIGatewayHandler) handleConcurrencyErrorContext(c gatewayctx.GatewayContext, err error, slotType string, streamStarted bool) {
	h.handleStreamingAwareErrorContext(c, http.StatusTooManyRequests, "rate_limit_error",
		fmt.Sprintf("Concurrency limit exceeded for %s, please retry later", slotType), streamStarted)
}

func (h *OpenAIGatewayHandler) handleFailoverExhausted(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	h.handleFailoverExhaustedContext(gatewayctx.FromGin(c), failoverErr, streamStarted)
}

func (h *OpenAIGatewayHandler) handleFailoverExhaustedContext(c gatewayctx.GatewayContext, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	statusCode := failoverErr.StatusCode
	responseBody := failoverErr.ResponseBody

	// 先检查透传规则
	if h.errorPassthroughService != nil && len(responseBody) > 0 {
		if rule := h.errorPassthroughService.MatchRule("openai", statusCode, responseBody); rule != nil {
			// 确定响应状态码
			respCode := statusCode
			if !rule.PassthroughCode && rule.ResponseCode != nil {
				respCode = *rule.ResponseCode
			}

			// 确定响应消息
			msg := service.ExtractUpstreamErrorMessage(responseBody)
			if !rule.PassthroughBody && rule.CustomMessage != nil {
				msg = *rule.CustomMessage
			}

			if rule.SkipMonitoring {
				c.SetValue(service.OpsSkipPassthroughKey, true)
			}

			h.handleStreamingAwareErrorContext(c, respCode, "upstream_error", msg, streamStarted)
			return
		}
	}

	if service.WriteOpenAICompactErrorAfterResponseStartedContext(c, "upstream_error", firstNonEmpty(service.ExtractUpstreamErrorMessage(responseBody), "Upstream compact request failed")) {
		return
	}

	// 记录原始上游状态码，以便 ops 错误日志捕获真实的上游错误
	upstreamMsg := service.ExtractUpstreamErrorMessage(responseBody)
	service.SetOpsUpstreamErrorContext(c, statusCode, upstreamMsg, "")

	// 使用默认的错误映射
	status, errType, errMsg := h.mapUpstreamError(statusCode)
	h.handleStreamingAwareErrorContext(c, status, errType, errMsg, streamStarted)
}

func (h *OpenAIGatewayHandler) handleRemoteCompactFailure(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	h.handleRemoteCompactFailureContext(gatewayctx.FromGin(c), failoverErr, streamStarted)
}

func (h *OpenAIGatewayHandler) handleRemoteCompactFailureContext(c gatewayctx.GatewayContext, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	if failoverErr == nil {
		h.handleStreamingAwareErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream compact request failed", streamStarted)
		return
	}

	statusCode := failoverErr.StatusCode
	if statusCode < 400 || statusCode > 599 {
		statusCode = http.StatusBadGateway
	}
	responseBody := failoverErr.ResponseBody

	if h.errorPassthroughService != nil && len(responseBody) > 0 {
		if rule := h.errorPassthroughService.MatchRule("openai", statusCode, responseBody); rule != nil {
			respCode := statusCode
			if !rule.PassthroughCode && rule.ResponseCode != nil {
				respCode = *rule.ResponseCode
			}

			msg := service.ExtractUpstreamErrorMessage(responseBody)
			if !rule.PassthroughBody && rule.CustomMessage != nil {
				msg = *rule.CustomMessage
			}

			if rule.SkipMonitoring {
				c.SetValue(service.OpsSkipPassthroughKey, true)
			}

			h.handleStreamingAwareErrorContext(c, respCode, "upstream_error", msg, streamStarted)
			return
		}
	}

	upstreamMsg := strings.TrimSpace(service.ExtractUpstreamErrorMessage(responseBody))
	if upstreamMsg == "" {
		upstreamMsg = strings.TrimSpace(failoverErr.Error())
	}
	if upstreamMsg == "" {
		_, _, mappedMsg := h.mapUpstreamError(statusCode)
		upstreamMsg = mappedMsg
	}
	service.SetOpsUpstreamErrorContext(c, statusCode, upstreamMsg, "")

	if shouldMaskOpenAIUpstreamError(statusCode) {
		mappedStatus, mappedType, mappedMsg := h.mapUpstreamError(statusCode)
		h.handleStreamingAwareErrorContext(c, mappedStatus, mappedType, mappedMsg, streamStarted)
		return
	}

	errType := "upstream_error"
	switch statusCode {
	case http.StatusBadRequest:
		errType = "invalid_request_error"
	case http.StatusUnauthorized:
		errType = "authentication_error"
	case http.StatusTooManyRequests:
		errType = "rate_limit_error"
	}

	h.handleStreamingAwareErrorContext(c, statusCode, errType, upstreamMsg, streamStarted)
}

// handleFailoverExhaustedSimple 简化版本，用于没有响应体的情况
func (h *OpenAIGatewayHandler) handleFailoverExhaustedSimple(c *gin.Context, statusCode int, streamStarted bool) {
	h.handleFailoverExhaustedSimpleContext(gatewayctx.FromGin(c), statusCode, streamStarted)
}

func (h *OpenAIGatewayHandler) handleFailoverExhaustedSimpleContext(c gatewayctx.GatewayContext, statusCode int, streamStarted bool) {
	status, errType, errMsg := h.mapUpstreamError(statusCode)
	service.SetOpsUpstreamErrorContext(c, statusCode, errMsg, "")
	h.handleStreamingAwareErrorContext(c, status, errType, errMsg, streamStarted)
}

func (h *OpenAIGatewayHandler) mapUpstreamError(statusCode int) (int, string, string) {
	switch statusCode {
	case 401:
		return http.StatusBadGateway, "upstream_error", "Upstream authentication failed, please contact administrator"
	case 403:
		return http.StatusBadGateway, "upstream_error", "Upstream access forbidden, please contact administrator"
	case 429:
		return http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable"
	case 529:
		return http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable"
	case 500, 502, 503, 504:
		return http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable"
	default:
		return http.StatusBadGateway, "upstream_error", "Upstream request failed"
	}
}

// handleStreamingAwareError handles errors that may occur after streaming has started
func (h *OpenAIGatewayHandler) handleStreamingAwareError(c *gin.Context, status int, errType, message string, streamStarted bool) {
	h.handleStreamingAwareErrorContext(gatewayctx.FromGin(c), status, errType, message, streamStarted)
}

func (h *OpenAIGatewayHandler) handleStreamingAwareErrorContext(c gatewayctx.GatewayContext, status int, errType, message string, streamStarted bool) {
	if c == nil {
		return
	}
	if streamStarted || service.RequestUsesOpenAISSE(c) {
		_ = gatewayctx.WriteSSEEvent(c, "error", gin.H{
			"error": gin.H{
				"type":    errType,
				"message": message,
			},
		})
		return
	}

	// Normal case: return JSON response with proper status code
	h.errorResponseGateway(c, status, errType, message)
}

// ensureForwardErrorResponse 在 Forward 返回错误但尚未写响应时补写统一错误响应。
func (h *OpenAIGatewayHandler) ensureForwardErrorResponse(c *gin.Context, streamStarted bool) bool {
	return h.ensureForwardErrorResponseContext(gatewayctx.FromGin(c), streamStarted)
}

func (h *OpenAIGatewayHandler) ensureForwardErrorResponseContext(c gatewayctx.GatewayContext, streamStarted bool) bool {
	if c == nil || service.RequestPayloadStarted(c) {
		return false
	}
	if c.ResponseWritten() && !streamStarted && !service.RequestUsesOpenAISSE(c) && !service.RequestUsesBufferedJSON(c) {
		return false
	}
	h.handleStreamingAwareErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", streamStarted)
	return true
}

func shouldLogOpenAIForwardFailureAsWarn(c *gin.Context, wroteFallback bool) bool {
	return shouldLogOpenAIForwardFailureAsWarnContext(gatewayctx.FromGin(c), wroteFallback)
}

func shouldLogOpenAIForwardFailureAsWarnContext(c gatewayctx.GatewayContext, wroteFallback bool) bool {
	if wroteFallback {
		return false
	}
	if c == nil {
		return false
	}
	return c.ResponseWritten()
}

// errorResponse returns OpenAI API format error response
func (h *OpenAIGatewayHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	h.errorResponseGateway(gatewayctx.FromGin(c), status, errType, message)
}

func setOpenAIClientTransportHTTP(c *gin.Context) {
	service.SetOpenAIClientTransport(c, service.OpenAIClientTransportHTTP)
}

func setOpenAIClientTransportHTTPGateway(c gatewayctx.GatewayContext) {
	service.SetOpenAIClientTransportContext(c, service.OpenAIClientTransportHTTP)
}

func setOpenAIClientTransportWS(c *gin.Context) {
	service.SetOpenAIClientTransport(c, service.OpenAIClientTransportWS)
}

func setOpenAIClientTransportWSGateway(c gatewayctx.GatewayContext) {
	service.SetOpenAIClientTransportContext(c, service.OpenAIClientTransportWS)
}

func (h *OpenAIGatewayHandler) errorResponseGateway(c gatewayctx.GatewayContext, status int, errType, message string) {
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

func (h *OpenAIGatewayHandler) ensureResponsesDependenciesGateway(c gatewayctx.GatewayContext, reqLog *zap.Logger) bool {
	missing := h.missingResponsesDependencies()
	if len(missing) == 0 {
		return true
	}

	if reqLog == nil {
		reqLog = requestLoggerContext(c, "handler.openai_gateway.responses")
	}
	reqLog.Error("openai.handler_dependencies_missing", zap.Strings("missing_dependencies", missing))
	h.errorResponseGateway(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
	return false
}

func ensureOpenAIPoolModeSessionHash(sessionHash string, account *service.Account) string {
	if sessionHash != "" || account == nil || !account.IsPoolMode() {
		return sessionHash
	}
	// 为当前请求生成一次性粘性会话键，确保同账号重试不会重新负载均衡到其他账号。
	return "openai-pool-retry-" + uuid.NewString()
}

func openAIWSIngressFallbackSessionSeed(userID, apiKeyID int64, groupID *int64) string {
	gid := int64(0)
	if groupID != nil {
		gid = *groupID
	}
	return fmt.Sprintf("openai_ws_ingress:%d:%d:%d", gid, userID, apiKeyID)
}

func isOpenAIWSUpgradeRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket") {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(r.Header.Get("Connection"))), "upgrade")
}

func closeOpenAIClientWS(conn *coderws.Conn, status coderws.StatusCode, reason string) {
	if conn == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 120 {
		reason = reason[:120]
	}
	_ = conn.Close(status, reason)
	_ = conn.CloseNow()
}

func summarizeWSCloseErrorForLog(err error) (string, string) {
	if err == nil {
		return "-", "-"
	}
	statusCode := coderws.CloseStatus(err)
	if statusCode == -1 {
		return "-", "-"
	}
	closeStatus := fmt.Sprintf("%d(%s)", int(statusCode), statusCode.String())
	closeReason := "-"
	var closeErr coderws.CloseError
	if errors.As(err, &closeErr) {
		reason := strings.TrimSpace(closeErr.Reason)
		if reason != "" {
			closeReason = reason
		}
	}
	return closeStatus, closeReason
}
