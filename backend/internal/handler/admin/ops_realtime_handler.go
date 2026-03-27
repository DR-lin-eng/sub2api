package admin

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

var (
	opsConcurrencySnapshotCache     = newSnapshotCache(10 * time.Second)
	opsUserConcurrencySnapshotCache = newSnapshotCache(10 * time.Second)
	opsAvailabilitySnapshotCache    = newSnapshotCache(10 * time.Second)
)

// GetConcurrencyStats returns real-time concurrency usage aggregated by platform/group/account.
// GET /api/v1/admin/ops/concurrency
func (h *OpsHandler) GetConcurrencyStats(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if !h.opsService.IsRealtimeMonitoringEnabled(c.Request.Context()) {
		response.Success(c, gin.H{
			"enabled":   false,
			"platform":  map[string]*service.PlatformConcurrencyInfo{},
			"group":     map[int64]*service.GroupConcurrencyInfo{},
			"account":   map[int64]*service.AccountConcurrencyInfo{},
			"timestamp": time.Now().UTC(),
		})
		return
	}

	platformFilter := strings.TrimSpace(c.Query("platform"))
	var groupID *int64
	if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		groupID = &id
	}
	includeAccount := parseBoolQueryWithDefault(c.Query("include_account"), true)
	accountLimit := 0
	if v := strings.TrimSpace(c.Query("account_limit")); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 0 {
			response.BadRequest(c, "Invalid account_limit")
			return
		}
		accountLimit = parsed
	}

	cacheKey := "concurrency:" + platformFilter + ":" + strconv.FormatBool(includeAccount) + ":" + strconv.Itoa(accountLimit)
	if groupID != nil {
		cacheKey += ":" + strconv.FormatInt(*groupID, 10)
	}
	entry, _, err := opsConcurrencySnapshotCache.GetOrLoad(cacheKey, func() (any, error) {
		platform, group, account, collectedAt, loadErr := h.opsService.GetConcurrencyStats(c.Request.Context(), platformFilter, groupID)
		if loadErr != nil {
			return nil, loadErr
		}
		if !includeAccount {
			account = map[int64]*service.AccountConcurrencyInfo{}
		} else if accountLimit > 0 {
			account = limitAccountConcurrencyMap(account, accountLimit)
		}
		payload := gin.H{
			"enabled":  true,
			"platform": platform,
			"group":    group,
			"account":  account,
		}
		if collectedAt != nil {
			payload["timestamp"] = collectedAt.UTC()
		}
		return payload, nil
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, entry.Payload)
}

// GetUserConcurrencyStats returns real-time concurrency usage for all active users.
// GET /api/v1/admin/ops/user-concurrency
func (h *OpsHandler) GetUserConcurrencyStats(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if !h.opsService.IsRealtimeMonitoringEnabled(c.Request.Context()) {
		response.Success(c, gin.H{
			"enabled":   false,
			"user":      map[int64]*service.UserConcurrencyInfo{},
			"timestamp": time.Now().UTC(),
		})
		return
	}

	userLimit := 0
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 0 {
			response.BadRequest(c, "Invalid limit")
			return
		}
		userLimit = parsed
	}

	cacheKey := "user_concurrency:" + strconv.Itoa(userLimit)
	entry, _, err := opsUserConcurrencySnapshotCache.GetOrLoad(cacheKey, func() (any, error) {
		users, collectedAt, loadErr := h.opsService.GetUserConcurrencyStats(c.Request.Context())
		if loadErr != nil {
			return nil, loadErr
		}
		if userLimit > 0 {
			users = limitUserConcurrencyMap(users, userLimit)
		}
		payload := gin.H{
			"enabled": true,
			"user":    users,
		}
		if collectedAt != nil {
			payload["timestamp"] = collectedAt.UTC()
		}
		return payload, nil
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, entry.Payload)
}

// GetAccountAvailability returns account availability statistics.
// GET /api/v1/admin/ops/account-availability
//
// Query params:
// - platform: optional
// - group_id: optional
func (h *OpsHandler) GetAccountAvailability(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if !h.opsService.IsRealtimeMonitoringEnabled(c.Request.Context()) {
		response.Success(c, gin.H{
			"enabled":   false,
			"platform":  map[string]*service.PlatformAvailability{},
			"group":     map[int64]*service.GroupAvailability{},
			"account":   map[int64]*service.AccountAvailability{},
			"timestamp": time.Now().UTC(),
		})
		return
	}

	platform := strings.TrimSpace(c.Query("platform"))
	var groupID *int64
	if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		groupID = &id
	}
	includeAccount := parseBoolQueryWithDefault(c.Query("include_account"), true)
	accountLimit := 0
	if v := strings.TrimSpace(c.Query("account_limit")); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 0 {
			response.BadRequest(c, "Invalid account_limit")
			return
		}
		accountLimit = parsed
	}

	cacheKey := "availability:" + platform + ":" + strconv.FormatBool(includeAccount) + ":" + strconv.Itoa(accountLimit)
	if groupID != nil {
		cacheKey += ":" + strconv.FormatInt(*groupID, 10)
	}
	entry, _, err := opsAvailabilitySnapshotCache.GetOrLoad(cacheKey, func() (any, error) {
		platformStats, groupStats, accountStats, collectedAt, loadErr := h.opsService.GetAccountAvailabilityStats(c.Request.Context(), platform, groupID)
		if loadErr != nil {
			return nil, loadErr
		}
		if !includeAccount {
			accountStats = map[int64]*service.AccountAvailability{}
		} else if accountLimit > 0 {
			accountStats = limitAccountAvailabilityMap(accountStats, accountLimit)
		}
		payload := gin.H{
			"enabled":  true,
			"platform": platformStats,
			"group":    groupStats,
			"account":  accountStats,
		}
		if collectedAt != nil {
			payload["timestamp"] = collectedAt.UTC()
		}
		return payload, nil
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, entry.Payload)
}

func limitAccountConcurrencyMap(src map[int64]*service.AccountConcurrencyInfo, limit int) map[int64]*service.AccountConcurrencyInfo {
	if limit <= 0 || len(src) <= limit {
		return src
	}
	rows := make([]*service.AccountConcurrencyInfo, 0, len(src))
	for _, item := range src {
		if item != nil {
			rows = append(rows, item)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].LoadPercentage != rows[j].LoadPercentage {
			return rows[i].LoadPercentage > rows[j].LoadPercentage
		}
		if rows[i].WaitingInQueue != rows[j].WaitingInQueue {
			return rows[i].WaitingInQueue > rows[j].WaitingInQueue
		}
		return rows[i].AccountID < rows[j].AccountID
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make(map[int64]*service.AccountConcurrencyInfo, len(rows))
	for _, item := range rows {
		out[item.AccountID] = item
	}
	return out
}

func limitUserConcurrencyMap(src map[int64]*service.UserConcurrencyInfo, limit int) map[int64]*service.UserConcurrencyInfo {
	if limit <= 0 || len(src) <= limit {
		return src
	}
	rows := make([]*service.UserConcurrencyInfo, 0, len(src))
	for _, item := range src {
		if item != nil {
			rows = append(rows, item)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].LoadPercentage != rows[j].LoadPercentage {
			return rows[i].LoadPercentage > rows[j].LoadPercentage
		}
		if rows[i].WaitingInQueue != rows[j].WaitingInQueue {
			return rows[i].WaitingInQueue > rows[j].WaitingInQueue
		}
		return rows[i].UserID < rows[j].UserID
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make(map[int64]*service.UserConcurrencyInfo, len(rows))
	for _, item := range rows {
		out[item.UserID] = item
	}
	return out
}

func limitAccountAvailabilityMap(src map[int64]*service.AccountAvailability, limit int) map[int64]*service.AccountAvailability {
	if limit <= 0 || len(src) <= limit {
		return src
	}
	rows := make([]*service.AccountAvailability, 0, len(src))
	for _, item := range src {
		if item != nil {
			rows = append(rows, item)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].HasError != rows[j].HasError {
			return rows[i].HasError
		}
		if rows[i].IsRateLimited != rows[j].IsRateLimited {
			return rows[i].IsRateLimited
		}
		if rows[i].IsAvailable != rows[j].IsAvailable {
			return !rows[i].IsAvailable
		}
		return rows[i].AccountID < rows[j].AccountID
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make(map[int64]*service.AccountAvailability, len(rows))
	for _, item := range rows {
		out[item.AccountID] = item
	}
	return out
}

func parseOpsRealtimeWindow(v string) (time.Duration, string, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "1min", "1m":
		return 1 * time.Minute, "1min", true
	case "5min", "5m":
		return 5 * time.Minute, "5min", true
	case "30min", "30m":
		return 30 * time.Minute, "30min", true
	case "1h", "60m", "60min":
		return 1 * time.Hour, "1h", true
	default:
		return 0, "", false
	}
}

// GetRealtimeTrafficSummary returns QPS/TPS current/peak/avg for the selected window.
// GET /api/v1/admin/ops/realtime-traffic
//
// Query params:
// - window: 1min|5min|30min|1h (default: 1min)
// - platform: optional
// - group_id: optional
func (h *OpsHandler) GetRealtimeTrafficSummary(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	windowDur, windowLabel, ok := parseOpsRealtimeWindow(c.Query("window"))
	if !ok {
		response.BadRequest(c, "Invalid window")
		return
	}

	platform := strings.TrimSpace(c.Query("platform"))
	var groupID *int64
	if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		groupID = &id
	}

	endTime := time.Now().UTC()
	startTime := endTime.Add(-windowDur)

	if !h.opsService.IsRealtimeMonitoringEnabled(c.Request.Context()) {
		disabledSummary := &service.OpsRealtimeTrafficSummary{
			Window:    windowLabel,
			StartTime: startTime,
			EndTime:   endTime,
			Platform:  platform,
			GroupID:   groupID,
			QPS:       service.OpsRateSummary{},
			TPS:       service.OpsRateSummary{},
		}
		response.Success(c, gin.H{
			"enabled":   false,
			"summary":   disabledSummary,
			"timestamp": endTime,
		})
		return
	}

	filter := &service.OpsDashboardFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Platform:  platform,
		GroupID:   groupID,
		QueryMode: service.OpsQueryModeRaw,
	}

	summary, err := h.opsService.GetRealtimeTrafficSummary(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if summary != nil {
		summary.Window = windowLabel
	}
	response.Success(c, gin.H{
		"enabled":   true,
		"summary":   summary,
		"timestamp": endTime,
	})
}

// GetGatewaySchedulerRuntime returns generic Gateway scheduler runtime metrics.
// GET /api/v1/admin/ops/gateway-scheduler
func (h *OpsHandler) GetGatewaySchedulerRuntime(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	platform := strings.TrimSpace(c.Query("platform"))
	limit := 6
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			response.BadRequest(c, "Invalid limit")
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}
	var groupID *int64
	if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		groupID = &id
	}

	data, err := h.opsService.GetGatewaySchedulerRuntime(c.Request.Context(), platform, groupID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

// GetOpenAIWSRuntime returns OpenAI WS runtime diagnostics.
// GET /api/v1/admin/ops/openai-ws-runtime
func (h *OpsHandler) GetOpenAIWSRuntime(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	platform := strings.TrimSpace(c.Query("platform"))
	data, err := h.opsService.GetOpenAIWSRuntime(c.Request.Context(), platform)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}
