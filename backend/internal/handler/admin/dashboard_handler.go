package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// DashboardHandler handles admin dashboard statistics
type DashboardHandler struct {
	dashboardService   *service.DashboardService
	aggregationService *service.DashboardAggregationService
	opsService         *service.OpsService
	startTime          time.Time // Server start time for uptime calculation
}

// NewDashboardHandler creates a new admin dashboard handler
func NewDashboardHandler(dashboardService *service.DashboardService, aggregationService *service.DashboardAggregationService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService:   dashboardService,
		aggregationService: aggregationService,
		startTime:          time.Now(),
	}
}

func (h *DashboardHandler) SetOpsService(opsService *service.OpsService) {
	if h == nil {
		return
	}
	h.opsService = opsService
}

// parseTimeRange parses start_date, end_date query parameters
// Uses user's timezone if provided, otherwise falls back to server timezone
func parseTimeRange(c *gin.Context) (time.Time, time.Time) {
	return parseTimeRangeGateway(gatewayctx.FromGin(c))
}

func parseTimeRangeGateway(c gatewayctx.GatewayContext) (time.Time, time.Time) {
	userTZ := c.QueryValue("timezone") // Get user's timezone from request
	now := timezone.NowInUserLocation(userTZ)
	startDate := c.QueryValue("start_date")
	endDate := c.QueryValue("end_date")

	var startTime, endTime time.Time

	if startDate != "" {
		if t, err := timezone.ParseInUserLocation("2006-01-02", startDate, userTZ); err == nil {
			startTime = t
		} else {
			startTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -7), userTZ)
		}
	} else {
		startTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -7), userTZ)
	}

	if endDate != "" {
		if t, err := timezone.ParseInUserLocation("2006-01-02", endDate, userTZ); err == nil {
			endTime = t.Add(24 * time.Hour) // Include the end date
		} else {
			endTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, 1), userTZ)
		}
	} else {
		endTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, 1), userTZ)
	}

	return startTime, endTime
}

// GetStats handles getting dashboard statistics
// GET /api/v1/admin/dashboard/stats
func (h *DashboardHandler) GetStats(c *gin.Context) {
	h.GetStatsGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetStatsGateway(c gatewayctx.GatewayContext) {
	stats, err := h.dashboardService.GetDashboardStats(c.Request().Context())
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get dashboard statistics")
		return
	}

	// Calculate uptime in seconds
	uptime := int64(time.Since(h.startTime).Seconds())

	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{
		// 用户统计
		"total_users":     stats.TotalUsers,
		"today_new_users": stats.TodayNewUsers,
		"active_users":    stats.ActiveUsers,

		// API Key 统计
		"total_api_keys":  stats.TotalAPIKeys,
		"active_api_keys": stats.ActiveAPIKeys,

		// 账户统计
		"total_accounts":     stats.TotalAccounts,
		"normal_accounts":    stats.NormalAccounts,
		"error_accounts":     stats.ErrorAccounts,
		"ratelimit_accounts": stats.RateLimitAccounts,
		"overload_accounts":  stats.OverloadAccounts,

		// 累计 Token 使用统计
		"total_requests":              stats.TotalRequests,
		"total_input_tokens":          stats.TotalInputTokens,
		"total_output_tokens":         stats.TotalOutputTokens,
		"total_cache_creation_tokens": stats.TotalCacheCreationTokens,
		"total_cache_read_tokens":     stats.TotalCacheReadTokens,
		"total_tokens":                stats.TotalTokens,
		"total_cost":                  stats.TotalCost,       // 标准计费
		"total_actual_cost":           stats.TotalActualCost, // 实际扣除

		// 今日 Token 使用统计
		"today_requests":              stats.TodayRequests,
		"today_input_tokens":          stats.TodayInputTokens,
		"today_output_tokens":         stats.TodayOutputTokens,
		"today_cache_creation_tokens": stats.TodayCacheCreationTokens,
		"today_cache_read_tokens":     stats.TodayCacheReadTokens,
		"today_tokens":                stats.TodayTokens,
		"today_cost":                  stats.TodayCost,       // 今日标准计费
		"today_actual_cost":           stats.TodayActualCost, // 今日实际扣除

		// 系统运行统计
		"average_duration_ms": stats.AverageDurationMs,
		"uptime":              uptime,

		// 性能指标
		"rpm": stats.Rpm,
		"tpm": stats.Tpm,

		// 预聚合新鲜度
		"hourly_active_users": stats.HourlyActiveUsers,
		"stats_updated_at":    stats.StatsUpdatedAt,
		"stats_stale":         stats.StatsStale,
	})
}

type DashboardAggregationBackfillRequest struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// BackfillAggregation handles triggering aggregation backfill
// POST /api/v1/admin/dashboard/aggregation/backfill
func (h *DashboardHandler) BackfillAggregation(c *gin.Context) {
	h.BackfillAggregationGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) BackfillAggregationGateway(c gatewayctx.GatewayContext) {
	if h.aggregationService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Aggregation service not available")
		return
	}

	var req DashboardAggregationBackfillRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request body")
		return
	}
	start, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid start time")
		return
	}
	end, err := time.Parse(time.RFC3339, req.End)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid end time")
		return
	}

	if err := h.aggregationService.TriggerBackfill(start, end); err != nil {
		if errors.Is(err, service.ErrDashboardBackfillDisabled) {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusForbidden, "Backfill is disabled")
			return
		}
		if errors.Is(err, service.ErrDashboardBackfillTooLarge) {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Backfill range too large")
			return
		}
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to trigger backfill")
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"status": "accepted",
	})
}

// GetRealtimeMetrics handles getting real-time system metrics
// GET /api/v1/admin/dashboard/realtime
func (h *DashboardHandler) GetRealtimeMetrics(c *gin.Context) {
	h.GetRealtimeMetricsGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetRealtimeMetricsGateway(c gatewayctx.GatewayContext) {
	if h.dashboardService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Dashboard service not available")
		return
	}
	stats, err := h.dashboardService.GetDashboardStats(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	activeRequests := int64(0)
	errorRate := 0.0
	if h.opsService != nil {
		endTime := time.Now().UTC()
		startTime := endTime.Add(-5 * time.Minute)
		if overview, overviewErr := h.opsService.GetDashboardOverview(c.Request().Context(), &service.OpsDashboardFilter{
			StartTime: startTime,
			EndTime:   endTime,
			QueryMode: service.OpsQueryModeRaw,
		}); overviewErr == nil && overview != nil {
			errorRate = overview.ErrorRate
		}
		if platformStats, _, _, _, concErr := h.opsService.GetConcurrencyStats(c.Request().Context(), "", nil, false); concErr == nil {
			for _, item := range platformStats {
				if item != nil {
					activeRequests += item.CurrentInUse
				}
			}
		}
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{
		"active_requests":       activeRequests,
		"requests_per_minute":   stats.Rpm,
		"average_response_time": stats.AverageDurationMs,
		"error_rate":            errorRate,
	})
}

// GetUsageTrend handles getting usage trend data
// GET /api/v1/admin/dashboard/trend
// Query params: start_date, end_date (YYYY-MM-DD), granularity (day/hour), user_id, api_key_id, model, account_id, group_id, request_type, stream, billing_type
func (h *DashboardHandler) GetUsageTrend(c *gin.Context) {
	h.GetUsageTrendGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetUsageTrendGateway(c gatewayctx.GatewayContext) {
	startTime, endTime := parseTimeRangeGateway(c)
	granularity := defaultQueryValue(c, "granularity", "day")

	// Parse optional filter params
	var userID, apiKeyID, accountID, groupID int64
	var model string
	var requestType *int16
	var stream *bool
	var billingType *int8

	if userIDStr := c.QueryValue("user_id"); userIDStr != "" {
		if id, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			userID = id
		}
	}
	if apiKeyIDStr := c.QueryValue("api_key_id"); apiKeyIDStr != "" {
		if id, err := strconv.ParseInt(apiKeyIDStr, 10, 64); err == nil {
			apiKeyID = id
		}
	}
	if accountIDStr := c.QueryValue("account_id"); accountIDStr != "" {
		if id, err := strconv.ParseInt(accountIDStr, 10, 64); err == nil {
			accountID = id
		}
	}
	if groupIDStr := c.QueryValue("group_id"); groupIDStr != "" {
		if id, err := strconv.ParseInt(groupIDStr, 10, 64); err == nil {
			groupID = id
		}
	}
	if modelStr := c.QueryValue("model"); modelStr != "" {
		model = modelStr
	}
	if requestTypeStr := strings.TrimSpace(c.QueryValue("request_type")); requestTypeStr != "" {
		parsed, err := service.ParseUsageRequestType(requestTypeStr)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		value := int16(parsed)
		requestType = &value
	} else if streamStr := c.QueryValue("stream"); streamStr != "" {
		if streamVal, err := strconv.ParseBool(streamStr); err == nil {
			stream = &streamVal
		} else {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid stream value, use true or false")
			return
		}
	}
	if billingTypeStr := c.QueryValue("billing_type"); billingTypeStr != "" {
		if v, err := strconv.ParseInt(billingTypeStr, 10, 8); err == nil {
			bt := int8(v)
			billingType = &bt
		} else {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid billing_type")
			return
		}
	}

	trend, hit, err := h.getUsageTrendCached(c.Request().Context(), startTime, endTime, granularity, userID, apiKeyID, accountID, groupID, model, requestType, stream, billingType)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get usage trend")
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"trend":       trend,
		"start_date":  startTime.Format("2006-01-02"),
		"end_date":    endTime.Add(-24 * time.Hour).Format("2006-01-02"),
		"granularity": granularity,
	})
}

// GetModelStats handles getting model usage statistics
// GET /api/v1/admin/dashboard/models
// Query params: start_date, end_date (YYYY-MM-DD), user_id, api_key_id, account_id, group_id, request_type, stream, billing_type
func (h *DashboardHandler) GetModelStats(c *gin.Context) {
	h.GetModelStatsGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetModelStatsGateway(c gatewayctx.GatewayContext) {
	startTime, endTime := parseTimeRangeGateway(c)

	// Parse optional filter params
	var userID, apiKeyID, accountID, groupID int64
	modelSource := usagestats.ModelSourceRequested
	var requestType *int16
	var stream *bool
	var billingType *int8

	if userIDStr := c.QueryValue("user_id"); userIDStr != "" {
		if id, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			userID = id
		}
	}
	if apiKeyIDStr := c.QueryValue("api_key_id"); apiKeyIDStr != "" {
		if id, err := strconv.ParseInt(apiKeyIDStr, 10, 64); err == nil {
			apiKeyID = id
		}
	}
	if accountIDStr := c.QueryValue("account_id"); accountIDStr != "" {
		if id, err := strconv.ParseInt(accountIDStr, 10, 64); err == nil {
			accountID = id
		}
	}
	if groupIDStr := c.QueryValue("group_id"); groupIDStr != "" {
		if id, err := strconv.ParseInt(groupIDStr, 10, 64); err == nil {
			groupID = id
		}
	}
	if rawModelSource := strings.TrimSpace(c.QueryValue("model_source")); rawModelSource != "" {
		if !usagestats.IsValidModelSource(rawModelSource) {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid model_source, use requested/upstream/mapping")
			return
		}
		modelSource = rawModelSource
	}
	if requestTypeStr := strings.TrimSpace(c.QueryValue("request_type")); requestTypeStr != "" {
		parsed, err := service.ParseUsageRequestType(requestTypeStr)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		value := int16(parsed)
		requestType = &value
	} else if streamStr := c.QueryValue("stream"); streamStr != "" {
		if streamVal, err := strconv.ParseBool(streamStr); err == nil {
			stream = &streamVal
		} else {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid stream value, use true or false")
			return
		}
	}
	if billingTypeStr := c.QueryValue("billing_type"); billingTypeStr != "" {
		if v, err := strconv.ParseInt(billingTypeStr, 10, 8); err == nil {
			bt := int8(v)
			billingType = &bt
		} else {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid billing_type")
			return
		}
	}

	stats, hit, err := h.getModelStatsCached(c.Request().Context(), startTime, endTime, userID, apiKeyID, accountID, groupID, modelSource, requestType, stream, billingType)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get model statistics")
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"models":     stats,
		"start_date": startTime.Format("2006-01-02"),
		"end_date":   endTime.Add(-24 * time.Hour).Format("2006-01-02"),
	})
}

// GetGroupStats handles getting group usage statistics
// GET /api/v1/admin/dashboard/groups
// Query params: start_date, end_date (YYYY-MM-DD), user_id, api_key_id, account_id, group_id, request_type, stream, billing_type
func (h *DashboardHandler) GetGroupStats(c *gin.Context) {
	h.GetGroupStatsGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetGroupStatsGateway(c gatewayctx.GatewayContext) {
	startTime, endTime := parseTimeRangeGateway(c)

	var userID, apiKeyID, accountID, groupID int64
	var requestType *int16
	var stream *bool
	var billingType *int8

	if userIDStr := c.QueryValue("user_id"); userIDStr != "" {
		if id, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			userID = id
		}
	}
	if apiKeyIDStr := c.QueryValue("api_key_id"); apiKeyIDStr != "" {
		if id, err := strconv.ParseInt(apiKeyIDStr, 10, 64); err == nil {
			apiKeyID = id
		}
	}
	if accountIDStr := c.QueryValue("account_id"); accountIDStr != "" {
		if id, err := strconv.ParseInt(accountIDStr, 10, 64); err == nil {
			accountID = id
		}
	}
	if groupIDStr := c.QueryValue("group_id"); groupIDStr != "" {
		if id, err := strconv.ParseInt(groupIDStr, 10, 64); err == nil {
			groupID = id
		}
	}
	if requestTypeStr := strings.TrimSpace(c.QueryValue("request_type")); requestTypeStr != "" {
		parsed, err := service.ParseUsageRequestType(requestTypeStr)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		value := int16(parsed)
		requestType = &value
	} else if streamStr := c.QueryValue("stream"); streamStr != "" {
		if streamVal, err := strconv.ParseBool(streamStr); err == nil {
			stream = &streamVal
		} else {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid stream value, use true or false")
			return
		}
	}
	if billingTypeStr := c.QueryValue("billing_type"); billingTypeStr != "" {
		if v, err := strconv.ParseInt(billingTypeStr, 10, 8); err == nil {
			bt := int8(v)
			billingType = &bt
		} else {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid billing_type")
			return
		}
	}

	stats, hit, err := h.getGroupStatsCached(c.Request().Context(), startTime, endTime, userID, apiKeyID, accountID, groupID, requestType, stream, billingType)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get group statistics")
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"groups":     stats,
		"start_date": startTime.Format("2006-01-02"),
		"end_date":   endTime.Add(-24 * time.Hour).Format("2006-01-02"),
	})
}

// GetAPIKeyUsageTrend handles getting API key usage trend data
// GET /api/v1/admin/dashboard/api-keys-trend
// Query params: start_date, end_date (YYYY-MM-DD), granularity (day/hour), limit (default 5)
func (h *DashboardHandler) GetAPIKeyUsageTrend(c *gin.Context) {
	h.GetAPIKeyUsageTrendGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetAPIKeyUsageTrendGateway(c gatewayctx.GatewayContext) {
	startTime, endTime := parseTimeRangeGateway(c)
	granularity := defaultQueryValue(c, "granularity", "day")
	limitStr := defaultQueryValue(c, "limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 5
	}

	trend, hit, err := h.getAPIKeyUsageTrendCached(c.Request().Context(), startTime, endTime, granularity, limit)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get API key usage trend")
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"trend":       trend,
		"start_date":  startTime.Format("2006-01-02"),
		"end_date":    endTime.Add(-24 * time.Hour).Format("2006-01-02"),
		"granularity": granularity,
	})
}

// GetUserUsageTrend handles getting user usage trend data
// GET /api/v1/admin/dashboard/users-trend
// Query params: start_date, end_date (YYYY-MM-DD), granularity (day/hour), limit (default 12)
func (h *DashboardHandler) GetUserUsageTrend(c *gin.Context) {
	h.GetUserUsageTrendGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetUserUsageTrendGateway(c gatewayctx.GatewayContext) {
	startTime, endTime := parseTimeRangeGateway(c)
	granularity := defaultQueryValue(c, "granularity", "day")
	limitStr := defaultQueryValue(c, "limit", "12")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 12
	}

	trend, hit, err := h.getUserUsageTrendCached(c.Request().Context(), startTime, endTime, granularity, limit)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get user usage trend")
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"trend":       trend,
		"start_date":  startTime.Format("2006-01-02"),
		"end_date":    endTime.Add(-24 * time.Hour).Format("2006-01-02"),
		"granularity": granularity,
	})
}

// BatchUsersUsageRequest represents the request body for batch user usage stats
type BatchUsersUsageRequest struct {
	UserIDs []int64 `json:"user_ids" binding:"required"`
}

var dashboardUsersRankingCache = newSnapshotCache(5 * time.Minute)
var dashboardBatchUsersUsageCache = newSnapshotCache(30 * time.Second)
var dashboardBatchAPIKeysUsageCache = newSnapshotCache(30 * time.Second)

func parseRankingLimit(raw string) int {
	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || limit <= 0 {
		return 12
	}
	if limit > 50 {
		return 50
	}
	return limit
}

// GetUserSpendingRanking handles getting user spending ranking data.
// GET /api/v1/admin/dashboard/users-ranking
func (h *DashboardHandler) GetUserSpendingRanking(c *gin.Context) {
	h.GetUserSpendingRankingGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetUserSpendingRankingGateway(c gatewayctx.GatewayContext) {
	startTime, endTime := parseTimeRangeGateway(c)
	limit := parseRankingLimit(defaultQueryValue(c, "limit", "12"))

	keyRaw, _ := json.Marshal(struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Limit int    `json:"limit"`
	}{
		Start: startTime.UTC().Format(time.RFC3339),
		End:   endTime.UTC().Format(time.RFC3339),
		Limit: limit,
	})
	cacheKey := string(keyRaw)
	if cached, ok := dashboardUsersRankingCache.Get(cacheKey); ok {
		c.SetHeader("X-Snapshot-Cache", "hit")
		response.SuccessContext(gatewayJSONResponder{ctx: c}, cached.Payload)
		return
	}

	ranking, err := h.dashboardService.GetUserSpendingRanking(c.Request().Context(), startTime, endTime, limit)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get user spending ranking")
		return
	}

	payload := map[string]any{
		"ranking":           ranking.Ranking,
		"total_actual_cost": ranking.TotalActualCost,
		"total_requests":    ranking.TotalRequests,
		"total_tokens":      ranking.TotalTokens,
		"start_date":        startTime.Format("2006-01-02"),
		"end_date":          endTime.Add(-24 * time.Hour).Format("2006-01-02"),
	}
	dashboardUsersRankingCache.Set(cacheKey, payload)
	c.SetHeader("X-Snapshot-Cache", "miss")
	response.SuccessContext(gatewayJSONResponder{ctx: c}, payload)
}

// GetBatchUsersUsage handles getting usage stats for multiple users
// POST /api/v1/admin/dashboard/users-usage
func (h *DashboardHandler) GetBatchUsersUsage(c *gin.Context) {
	h.GetBatchUsersUsageGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetBatchUsersUsageGateway(c gatewayctx.GatewayContext) {
	var req BatchUsersUsageRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	userIDs := normalizeInt64IDList(req.UserIDs)
	if len(userIDs) == 0 {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"stats": map[string]any{}})
		return
	}

	keyRaw, _ := json.Marshal(struct {
		UserIDs []int64 `json:"user_ids"`
	}{
		UserIDs: userIDs,
	})
	cacheKey := string(keyRaw)
	if cached, ok := dashboardBatchUsersUsageCache.Get(cacheKey); ok {
		c.SetHeader("X-Snapshot-Cache", "hit")
		response.SuccessContext(gatewayJSONResponder{ctx: c}, cached.Payload)
		return
	}

	stats, err := h.dashboardService.GetBatchUserUsageStats(c.Request().Context(), userIDs, time.Time{}, time.Time{})
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get user usage stats")
		return
	}

	payload := map[string]any{"stats": stats}
	dashboardBatchUsersUsageCache.Set(cacheKey, payload)
	c.SetHeader("X-Snapshot-Cache", "miss")
	response.SuccessContext(gatewayJSONResponder{ctx: c}, payload)
}

// BatchAPIKeysUsageRequest represents the request body for batch api key usage stats
type BatchAPIKeysUsageRequest struct {
	APIKeyIDs []int64 `json:"api_key_ids" binding:"required"`
}

// GetBatchAPIKeysUsage handles getting usage stats for multiple API keys
// POST /api/v1/admin/dashboard/api-keys-usage
func (h *DashboardHandler) GetBatchAPIKeysUsage(c *gin.Context) {
	h.GetBatchAPIKeysUsageGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetBatchAPIKeysUsageGateway(c gatewayctx.GatewayContext) {
	var req BatchAPIKeysUsageRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	apiKeyIDs := normalizeInt64IDList(req.APIKeyIDs)
	if len(apiKeyIDs) == 0 {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"stats": map[string]any{}})
		return
	}

	keyRaw, _ := json.Marshal(struct {
		APIKeyIDs []int64 `json:"api_key_ids"`
	}{
		APIKeyIDs: apiKeyIDs,
	})
	cacheKey := string(keyRaw)
	if cached, ok := dashboardBatchAPIKeysUsageCache.Get(cacheKey); ok {
		c.SetHeader("X-Snapshot-Cache", "hit")
		response.SuccessContext(gatewayJSONResponder{ctx: c}, cached.Payload)
		return
	}

	stats, err := h.dashboardService.GetBatchAPIKeyUsageStats(c.Request().Context(), apiKeyIDs, time.Time{}, time.Time{})
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get API key usage stats")
		return
	}

	payload := map[string]any{"stats": stats}
	dashboardBatchAPIKeysUsageCache.Set(cacheKey, payload)
	c.SetHeader("X-Snapshot-Cache", "miss")
	response.SuccessContext(gatewayJSONResponder{ctx: c}, payload)
}

// GetUserBreakdown handles getting per-user usage breakdown within a dimension.
// GET /api/v1/admin/dashboard/user-breakdown
// Query params: start_date, end_date, group_id, model, endpoint, endpoint_type, limit
func (h *DashboardHandler) GetUserBreakdown(c *gin.Context) {
	h.GetUserBreakdownGateway(gatewayctx.FromGin(c))
}

func (h *DashboardHandler) GetUserBreakdownGateway(c gatewayctx.GatewayContext) {
	startTime, endTime := parseTimeRangeGateway(c)

	dim := usagestats.UserBreakdownDimension{}
	if v := c.QueryValue("group_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			dim.GroupID = id
		}
	}
	dim.Model = c.QueryValue("model")
	rawModelSource := strings.TrimSpace(defaultQueryValue(c, "model_source", usagestats.ModelSourceRequested))
	if !usagestats.IsValidModelSource(rawModelSource) {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid model_source, use requested/upstream/mapping")
		return
	}
	dim.ModelType = rawModelSource
	dim.Endpoint = c.QueryValue("endpoint")
	dim.EndpointType = defaultQueryValue(c, "endpoint_type", "inbound")

	limit := 50
	if v := c.QueryValue("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	stats, err := h.dashboardService.GetUserBreakdownStats(
		c.Request().Context(), startTime, endTime, dim, limit,
	)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get user breakdown stats")
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"users":      stats,
		"start_date": startTime.Format("2006-01-02"),
		"end_date":   endTime.Add(-24 * time.Hour).Format("2006-01-02"),
	})
}
