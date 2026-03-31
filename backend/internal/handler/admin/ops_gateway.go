package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func parseOpsViewParamGateway(c gatewayctx.GatewayContext) string {
	if c == nil {
		return ""
	}
	v := strings.ToLower(strings.TrimSpace(c.QueryValue("view")))
	switch v {
	case "", opsListViewErrors:
		return opsListViewErrors
	case opsListViewExcluded:
		return opsListViewExcluded
	case opsListViewAll:
		return opsListViewAll
	default:
		return opsListViewErrors
	}
}

func (h *OpsHandler) GetErrorLogByIDGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	id, ok := parseRequiredPositiveInt64PathGateway(c, "id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid error id")
		return
	}
	detail, err := h.opsService.GetErrorLogByID(c.Request().Context(), id)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, detail)
}

func (h *OpsHandler) GetErrorLogsGateway(c gatewayctx.GatewayContext) {
	result, etag, hit, err := h.listOpsErrorLogsGateway(c, "1h", false, false)
	if err != nil {
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))
	if applySnapshotETagHeaders(c, etag) {
		return
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, result.Errors, int64(result.Total), result.Page, result.PageSize)
}

func (h *OpsHandler) ListRequestErrorsGateway(c gatewayctx.GatewayContext) {
	result, etag, hit, err := h.listOpsErrorLogsGateway(c, "1h", false, false)
	if err != nil {
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))
	if applySnapshotETagHeaders(c, etag) {
		return
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, result.Errors, int64(result.Total), result.Page, result.PageSize)
}

func (h *OpsHandler) GetRequestErrorGateway(c gatewayctx.GatewayContext) {
	h.GetErrorLogByIDGateway(c)
}

func (h *OpsHandler) ListRequestErrorUpstreamErrorsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	id, ok := parseRequiredPositiveInt64PathGateway(c, "id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid error id")
		return
	}
	detail, err := h.opsService.GetErrorLogByID(c.Request().Context(), id)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	requestID := strings.TrimSpace(detail.RequestID)
	clientRequestID := strings.TrimSpace(detail.ClientRequestID)
	if requestID == "" && clientRequestID == "" {
		response.PaginatedContext(gatewayJSONResponder{ctx: c}, []*service.OpsErrorLog{}, 0, 1, 10)
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	if pageSize > 500 {
		pageSize = 500
	}
	startTime, endTime, err := parseOpsTimeRangeGateway(c, "30d")
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	filter := &service.OpsErrorLogFilter{
		Page:      page,
		PageSize:  pageSize,
		StartTime: &startTime,
		EndTime:   &endTime,
		View:      "all",
		Phase:     "upstream",
		Owner:     "provider",
		Source:    strings.TrimSpace(c.QueryValue("error_source")),
		Query:     strings.TrimSpace(c.QueryValue("q")),
	}
	if platform := strings.TrimSpace(c.QueryValue("platform")); platform != "" {
		filter.Platform = platform
	}
	if requestID != "" {
		filter.RequestID = requestID
	} else {
		filter.ClientRequestID = clientRequestID
	}
	result, err := h.opsService.GetErrorLogs(c.Request().Context(), filter)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	includeDetail := strings.TrimSpace(c.QueryValue("include_detail"))
	if includeDetail == "1" || strings.EqualFold(includeDetail, "true") || strings.EqualFold(includeDetail, "yes") {
		details := make([]*service.OpsErrorLogDetail, 0, len(result.Errors))
		for _, item := range result.Errors {
			if item == nil {
				continue
			}
			d, err := h.opsService.GetErrorLogByID(c.Request().Context(), item.ID)
			if err != nil || d == nil {
				continue
			}
			details = append(details, d)
		}
		response.PaginatedContext(gatewayJSONResponder{ctx: c}, details, int64(result.Total), result.Page, result.PageSize)
		return
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, result.Errors, int64(result.Total), result.Page, result.PageSize)
}

func (h *OpsHandler) RetryRequestErrorClientGateway(c gatewayctx.GatewayContext) {
	h.retryOpsErrorGateway(c, service.OpsRetryModeClient)
}

func (h *OpsHandler) RetryRequestErrorUpstreamEventGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, ok := parseRequiredPositiveInt64PathGateway(c, "id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid error id")
		return
	}
	idxRaw := strings.TrimSpace(c.PathParam("idx"))
	idx, err := strconv.Atoi(idxRaw)
	if err != nil || idx < 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid upstream idx")
		return
	}
	result, err := h.opsService.RetryUpstreamEvent(c.Request().Context(), subject.UserID, id, idx)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

func (h *OpsHandler) ResolveRequestErrorGateway(c gatewayctx.GatewayContext) {
	h.UpdateErrorResolutionGateway(c)
}

func (h *OpsHandler) ListUpstreamErrorsGateway(c gatewayctx.GatewayContext) {
	result, etag, hit, err := h.listOpsErrorLogsGateway(c, "1h", true, true)
	if err != nil {
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))
	if applySnapshotETagHeaders(c, etag) {
		return
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, result.Errors, int64(result.Total), result.Page, result.PageSize)
}

func (h *OpsHandler) GetUpstreamErrorGateway(c gatewayctx.GatewayContext) {
	h.GetErrorLogByIDGateway(c)
}

func (h *OpsHandler) RetryUpstreamErrorGateway(c gatewayctx.GatewayContext) {
	h.retryOpsErrorGateway(c, service.OpsRetryModeUpstream)
}

func (h *OpsHandler) ResolveUpstreamErrorGateway(c gatewayctx.GatewayContext) {
	h.UpdateErrorResolutionGateway(c)
}

func (h *OpsHandler) ListRequestDetailsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	if pageSize > 100 {
		pageSize = 100
	}
	startTime, endTime, err := parseOpsTimeRangeGateway(c, "1h")
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	filter := &service.OpsRequestDetailFilter{
		Page:      page,
		PageSize:  pageSize,
		StartTime: &startTime,
		EndTime:   &endTime,
		Kind:      strings.TrimSpace(c.QueryValue("kind")),
		Platform:  strings.TrimSpace(c.QueryValue("platform")),
		Model:     strings.TrimSpace(c.QueryValue("model")),
		RequestID: strings.TrimSpace(c.QueryValue("request_id")),
		Query:     strings.TrimSpace(c.QueryValue("q")),
		Sort:      strings.TrimSpace(c.QueryValue("sort")),
	}
	var ok bool
	if filter.UserID, ok = parseOptionalInt64QueryGateway(c, "user_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid user_id")
		return
	}
	if filter.APIKeyID, ok = parseOptionalInt64QueryGateway(c, "api_key_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid api_key_id")
		return
	}
	if filter.AccountID, ok = parseOptionalInt64QueryGateway(c, "account_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid account_id")
		return
	}
	if filter.GroupID, ok = parseOptionalInt64QueryGateway(c, "group_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
		return
	}
	if raw := strings.TrimSpace(c.QueryValue("min_duration_ms")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid min_duration_ms")
			return
		}
		filter.MinDurationMs = &n
	}
	if raw := strings.TrimSpace(c.QueryValue("max_duration_ms")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid max_duration_ms")
			return
		}
		filter.MaxDurationMs = &n
	}
	payload, etag, hit, err := getOpsRequestDetailCached(c.Request().Context(), filter, func(ctx context.Context) (*service.OpsRequestDetailList, error) {
		return h.opsService.ListRequestDetails(ctx, filter)
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid") {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to list request details")
		return
	}
	c.SetHeader("X-Snapshot-Cache", cacheStatusValue(hit))
	if applySnapshotETagHeaders(c, etag) {
		return
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, payload.Items, payload.Total, payload.Page, payload.PageSize)
}

func (h *OpsHandler) RetryErrorRequestGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, ok := parseRequiredPositiveInt64PathGateway(c, "id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid error id")
		return
	}
	req := opsRetryRequest{Mode: service.OpsRetryModeClient}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Mode) == "" {
		req.Mode = service.OpsRetryModeClient
	}
	if strings.EqualFold(strings.TrimSpace(req.Mode), service.OpsRetryModeUpstream) {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "upstream retry is not supported on this endpoint")
		return
	}
	result, err := h.opsService.RetryError(c.Request().Context(), subject.UserID, id, req.Mode, req.PinnedAccountID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

func (h *OpsHandler) ListRetryAttemptsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	id, ok := parseRequiredPositiveInt64PathGateway(c, "id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid error id")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(c.QueryValue("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid limit")
			return
		}
		limit = n
	}
	items, err := h.opsService.ListRetryAttemptsByErrorID(c.Request().Context(), id, limit)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, items)
}

func (h *OpsHandler) UpdateErrorResolutionGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, ok := parseRequiredPositiveInt64PathGateway(c, "id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid error id")
		return
	}
	var req opsResolveRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	uid := subject.UserID
	if err := h.opsService.UpdateErrorResolution(c.Request().Context(), id, req.Resolved, &uid, nil); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"ok": true})
}

func (h *OpsHandler) listOpsErrorLogsGateway(c gatewayctx.GatewayContext, defaultRange string, upstreamOnly bool, fixedUpstreamPhase bool) (*service.OpsErrorLogList, string, bool, error) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return nil, "", false, fmt.Errorf("ops service unavailable")
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return nil, "", false, err
	}
	page, pageSize := response.ParsePaginationValues(c)
	if pageSize > 500 {
		pageSize = 500
	}
	startTime, endTime, err := parseOpsTimeRangeGateway(c, defaultRange)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return nil, "", false, err
	}
	filter := &service.OpsErrorLogFilter{Page: page, PageSize: pageSize}
	if !startTime.IsZero() {
		filter.StartTime = &startTime
	}
	if !endTime.IsZero() {
		filter.EndTime = &endTime
	}
	filter.View = parseOpsViewParamGateway(c)
	filter.Phase = strings.TrimSpace(c.QueryValue("phase"))
	filter.Owner = strings.TrimSpace(c.QueryValue("error_owner"))
	filter.Source = strings.TrimSpace(c.QueryValue("error_source"))
	filter.Query = strings.TrimSpace(c.QueryValue("q"))
	filter.UserQuery = strings.TrimSpace(c.QueryValue("user_query"))
	if fixedUpstreamPhase {
		filter.Phase = "upstream"
		filter.Owner = "provider"
	} else if strings.EqualFold(strings.TrimSpace(filter.Phase), "upstream") {
		filter.Phase = ""
	}
	if platform := strings.TrimSpace(c.QueryValue("platform")); platform != "" {
		filter.Platform = platform
	}
	var ok bool
	if filter.GroupID, ok = parseOptionalInt64QueryGateway(c, "group_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
		return nil, "", false, fmt.Errorf("invalid group_id")
	}
	if filter.AccountID, ok = parseOptionalInt64QueryGateway(c, "account_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid account_id")
		return nil, "", false, fmt.Errorf("invalid account_id")
	}
	if v := strings.TrimSpace(c.QueryValue("resolved")); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes":
			b := true
			filter.Resolved = &b
		case "0", "false", "no":
			b := false
			filter.Resolved = &b
		default:
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid resolved")
			return nil, "", false, fmt.Errorf("invalid resolved")
		}
	}
	if statusCodesStr := strings.TrimSpace(c.QueryValue("status_codes")); statusCodesStr != "" {
		parts := strings.Split(statusCodesStr, ",")
		out := make([]int, 0, len(parts))
		for _, part := range parts {
			p := strings.TrimSpace(part)
			if p == "" {
				continue
			}
			n, err := strconv.Atoi(p)
			if err != nil || n < 0 {
				response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid status_codes")
				return nil, "", false, fmt.Errorf("invalid status_codes")
			}
			out = append(out, n)
		}
		filter.StatusCodes = out
	}
	payload, etag, hit, err := getOpsErrorListCached(c.Request().Context(), struct {
		DefaultRange      string                     `json:"default_range"`
		UpstreamOnly      bool                       `json:"upstream_only"`
		FixedUpstreamOnly bool                       `json:"fixed_upstream_only"`
		Filter            *service.OpsErrorLogFilter `json:"filter"`
	}{
		DefaultRange:      defaultRange,
		UpstreamOnly:      upstreamOnly,
		FixedUpstreamOnly: fixedUpstreamPhase,
		Filter:            filter,
	}, func(ctx context.Context) (*service.OpsErrorLogList, error) {
		return h.opsService.GetErrorLogs(ctx, filter)
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return nil, "", false, err
	}
	_ = upstreamOnly
	return payload.Result, etag, hit, nil
}

func applySnapshotETagHeaders(c gatewayctx.GatewayContext, etag string) bool {
	if c == nil || etag == "" {
		return false
	}
	c.SetHeader("ETag", etag)
	c.SetHeader("Vary", "If-None-Match")
	if ifNoneMatchMatched(c.HeaderValue("If-None-Match"), etag) {
		_, _ = c.WriteBytes(http.StatusNotModified, nil)
		return true
	}
	return false
}

func (h *OpsHandler) retryOpsErrorGateway(c gatewayctx.GatewayContext, mode string) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, ok := parseRequiredPositiveInt64PathGateway(c, "id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid error id")
		return
	}
	result, err := h.opsService.RetryError(c.Request().Context(), subject.UserID, id, mode, nil)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

func parseRequiredPositiveInt64PathGateway(c gatewayctx.GatewayContext, key string) (int64, bool) {
	raw := strings.TrimSpace(c.PathParam(key))
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (h *OpsHandler) GetConcurrencyStatsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if !h.opsService.IsRealtimeMonitoringEnabled(c.Request().Context()) {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
			"enabled":   false,
			"platform":  map[string]*service.PlatformConcurrencyInfo{},
			"group":     map[int64]*service.GroupConcurrencyInfo{},
			"account":   map[int64]*service.AccountConcurrencyInfo{},
			"timestamp": time.Now().UTC(),
		})
		return
	}

	platformFilter := strings.TrimSpace(c.QueryValue("platform"))
	groupID, ok := parseOptionalInt64QueryGateway(c, "group_id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
		return
	}
	includeAccount := parseBoolQueryWithDefault(c.QueryValue("include_account"), true)
	accountLimit, ok := parseOptionalNonNegativeIntGateway(c, "account_limit")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid account_limit")
		return
	}

	cacheKey := "concurrency:" + platformFilter + ":" + strconv.FormatBool(includeAccount) + ":" + strconv.Itoa(accountLimit)
	if groupID != nil {
		cacheKey += ":" + strconv.FormatInt(*groupID, 10)
	}
	entry, _, err := opsConcurrencySnapshotCache.GetOrLoad(cacheKey, func() (any, error) {
		platform, group, account, collectedAt, loadErr := h.opsService.GetConcurrencyStats(c.Request().Context(), platformFilter, groupID)
		if loadErr != nil {
			return nil, loadErr
		}
		if !includeAccount {
			account = map[int64]*service.AccountConcurrencyInfo{}
		} else if accountLimit > 0 {
			account = limitAccountConcurrencyMap(account, accountLimit)
		}
		payload := map[string]any{
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
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, entry.Payload)
}

func (h *OpsHandler) GetUserConcurrencyStatsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if !h.opsService.IsRealtimeMonitoringEnabled(c.Request().Context()) {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
			"enabled":   false,
			"user":      map[int64]*service.UserConcurrencyInfo{},
			"timestamp": time.Now().UTC(),
		})
		return
	}

	userLimit, ok := parseOptionalNonNegativeIntGateway(c, "limit")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid limit")
		return
	}

	cacheKey := "user_concurrency:" + strconv.Itoa(userLimit)
	entry, _, err := opsUserConcurrencySnapshotCache.GetOrLoad(cacheKey, func() (any, error) {
		users, collectedAt, loadErr := h.opsService.GetUserConcurrencyStats(c.Request().Context())
		if loadErr != nil {
			return nil, loadErr
		}
		if userLimit > 0 {
			users = limitUserConcurrencyMap(users, userLimit)
		}
		payload := map[string]any{
			"enabled": true,
			"user":    users,
		}
		if collectedAt != nil {
			payload["timestamp"] = collectedAt.UTC()
		}
		return payload, nil
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, entry.Payload)
}

func (h *OpsHandler) GetAccountAvailabilityGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if !h.opsService.IsRealtimeMonitoringEnabled(c.Request().Context()) {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
			"enabled":   false,
			"platform":  map[string]*service.PlatformAvailability{},
			"group":     map[int64]*service.GroupAvailability{},
			"account":   map[int64]*service.AccountAvailability{},
			"timestamp": time.Now().UTC(),
		})
		return
	}

	platform := strings.TrimSpace(c.QueryValue("platform"))
	groupID, ok := parseOptionalInt64QueryGateway(c, "group_id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
		return
	}
	includeAccount := parseBoolQueryWithDefault(c.QueryValue("include_account"), true)
	accountLimit, ok := parseOptionalNonNegativeIntGateway(c, "account_limit")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid account_limit")
		return
	}

	cacheKey := "availability:" + platform + ":" + strconv.FormatBool(includeAccount) + ":" + strconv.Itoa(accountLimit)
	if groupID != nil {
		cacheKey += ":" + strconv.FormatInt(*groupID, 10)
	}
	entry, _, err := opsAvailabilitySnapshotCache.GetOrLoad(cacheKey, func() (any, error) {
		platformStats, groupStats, accountStats, collectedAt, loadErr := h.opsService.GetAccountAvailabilityStats(c.Request().Context(), platform, groupID)
		if loadErr != nil {
			return nil, loadErr
		}
		if !includeAccount {
			accountStats = map[int64]*service.AccountAvailability{}
		} else if accountLimit > 0 {
			accountStats = limitAccountAvailabilityMap(accountStats, accountLimit)
		}
		payload := map[string]any{
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
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, entry.Payload)
}

func (h *OpsHandler) GetRealtimeTrafficSummaryGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	windowDur, windowLabel, ok := parseOpsRealtimeWindow(c.QueryValue("window"))
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid window")
		return
	}
	platform := strings.TrimSpace(c.QueryValue("platform"))
	groupID, ok := parseOptionalInt64QueryGateway(c, "group_id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
		return
	}

	endTime := time.Now().UTC()
	startTime := endTime.Add(-windowDur)
	if !h.opsService.IsRealtimeMonitoringEnabled(c.Request().Context()) {
		disabledSummary := &service.OpsRealtimeTrafficSummary{
			Window:    windowLabel,
			StartTime: startTime,
			EndTime:   endTime,
			Platform:  platform,
			GroupID:   groupID,
			QPS:       service.OpsRateSummary{},
			TPS:       service.OpsRateSummary{},
		}
		response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
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
	summary, err := h.opsService.GetRealtimeTrafficSummary(c.Request().Context(), filter)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if summary != nil {
		summary.Window = windowLabel
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"enabled":   true,
		"summary":   summary,
		"timestamp": endTime,
	})
}

func (h *OpsHandler) GetGatewaySchedulerRuntimeGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	platform := strings.TrimSpace(c.QueryValue("platform"))
	limit := 6
	if raw := strings.TrimSpace(c.QueryValue("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid limit")
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}
	groupID, ok := parseOptionalInt64QueryGateway(c, "group_id")
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
		return
	}

	data, err := h.opsService.GetGatewaySchedulerRuntime(c.Request().Context(), platform, groupID, limit)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, data)
}

func (h *OpsHandler) GetOpenAIWSRuntimeGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	data, err := h.opsService.GetOpenAIWSRuntime(c.Request().Context(), strings.TrimSpace(c.QueryValue("platform")))
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, data)
}

func (h *OpsHandler) GetDashboardOverviewGateway(c gatewayctx.GatewayContext) {
	h.handleDashboardFilterGateway(c, func(filter *service.OpsDashboardFilter) (any, error) {
		return h.opsService.GetDashboardOverview(c.Request().Context(), filter)
	})
}

func (h *OpsHandler) GetDashboardThroughputTrendGateway(c gatewayctx.GatewayContext) {
	h.handleDashboardFilterGateway(c, func(filter *service.OpsDashboardFilter) (any, error) {
		return h.opsService.GetThroughputTrend(c.Request().Context(), filter, pickThroughputBucketSeconds(filter.EndTime.Sub(filter.StartTime)))
	})
}

func (h *OpsHandler) GetDashboardLatencyHistogramGateway(c gatewayctx.GatewayContext) {
	h.handleDashboardFilterGateway(c, func(filter *service.OpsDashboardFilter) (any, error) {
		return h.opsService.GetLatencyHistogram(c.Request().Context(), filter)
	})
}

func (h *OpsHandler) GetDashboardErrorTrendGateway(c gatewayctx.GatewayContext) {
	h.handleDashboardFilterGateway(c, func(filter *service.OpsDashboardFilter) (any, error) {
		return h.opsService.GetErrorTrend(c.Request().Context(), filter, pickThroughputBucketSeconds(filter.EndTime.Sub(filter.StartTime)))
	})
}

func (h *OpsHandler) GetDashboardErrorDistributionGateway(c gatewayctx.GatewayContext) {
	h.handleDashboardFilterGateway(c, func(filter *service.OpsDashboardFilter) (any, error) {
		return h.opsService.GetErrorDistribution(c.Request().Context(), filter)
	})
}

func (h *OpsHandler) GetDashboardOpenAITokenStatsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	filter, err := parseOpsOpenAITokenStatsFilterGateway(c)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.opsService.GetOpenAITokenStats(c.Request().Context(), filter)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, data)
}

func (h *OpsHandler) ListSystemLogsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	page, pageSize := response.ParsePaginationValues(c)
	if pageSize > 200 {
		pageSize = 200
	}
	start, end, err := parseOpsTimeRangeGateway(c, "1h")
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	filter := &service.OpsSystemLogFilter{
		Page:            page,
		PageSize:        pageSize,
		StartTime:       &start,
		EndTime:         &end,
		Level:           strings.TrimSpace(c.QueryValue("level")),
		Component:       strings.TrimSpace(c.QueryValue("component")),
		RequestID:       strings.TrimSpace(c.QueryValue("request_id")),
		ClientRequestID: strings.TrimSpace(c.QueryValue("client_request_id")),
		Platform:        strings.TrimSpace(c.QueryValue("platform")),
		Model:           strings.TrimSpace(c.QueryValue("model")),
		Query:           strings.TrimSpace(c.QueryValue("q")),
	}
	var ok bool
	if filter.UserID, ok = parseOptionalInt64QueryGateway(c, "user_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid user_id")
		return
	}
	if filter.AccountID, ok = parseOptionalInt64QueryGateway(c, "account_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid account_id")
		return
	}
	result, err := h.opsService.ListSystemLogs(c.Request().Context(), filter)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, result.Logs, int64(result.Total), result.Page, result.PageSize)
}

func (h *OpsHandler) CleanupSystemLogsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req opsSystemLogCleanupRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request body")
		return
	}
	start, err := parseOptionalRFC3339(req.StartTime)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid start_time")
		return
	}
	end, err := parseOptionalRFC3339(req.EndTime)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid end_time")
		return
	}
	filter := &service.OpsSystemLogCleanupFilter{
		StartTime:       start,
		EndTime:         end,
		Level:           strings.TrimSpace(req.Level),
		Component:       strings.TrimSpace(req.Component),
		RequestID:       strings.TrimSpace(req.RequestID),
		ClientRequestID: strings.TrimSpace(req.ClientRequestID),
		UserID:          req.UserID,
		AccountID:       req.AccountID,
		Platform:        strings.TrimSpace(req.Platform),
		Model:           strings.TrimSpace(req.Model),
		Query:           strings.TrimSpace(req.Query),
	}
	deleted, err := h.opsService.CleanupSystemLogs(c.Request().Context(), filter, subject.UserID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"deleted": deleted})
}

func (h *OpsHandler) GetSystemLogIngestionHealthGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, h.opsService.GetSystemLogSinkHealth())
}

func (h *OpsHandler) handleDashboardFilterGateway(c gatewayctx.GatewayContext, execute func(*service.OpsDashboardFilter) (any, error)) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	startTime, endTime, err := parseOpsTimeRangeGateway(c, "1h")
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	filter := &service.OpsDashboardFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Platform:  strings.TrimSpace(c.QueryValue("platform")),
		QueryMode: parseOpsQueryModeGateway(c),
	}
	var ok bool
	if filter.GroupID, ok = parseOptionalInt64QueryGateway(c, "group_id"); !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
		return
	}
	data, err := execute(filter)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, data)
}

func parseOpsTimeRangeGateway(c gatewayctx.GatewayContext, defaultRange string) (time.Time, time.Time, error) {
	startStr := strings.TrimSpace(c.QueryValue("start_time"))
	endStr := strings.TrimSpace(c.QueryValue("end_time"))
	parseTS := func(s string) (time.Time, error) {
		if s == "" {
			return time.Time{}, nil
		}
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t, nil
		}
		return time.Parse(time.RFC3339, s)
	}
	start, err := parseTS(startStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseTS(endStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if startStr != "" || endStr != "" {
		if end.IsZero() {
			end = time.Now()
		}
		if start.IsZero() {
			dur, _ := parseOpsDuration(defaultRange)
			start = end.Add(-dur)
		}
		if start.After(end) {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid time range: start_time must be <= end_time")
		}
		if end.Sub(start) > 30*24*time.Hour {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid time range: max window is 30 days")
		}
		return start, end, nil
	}
	tr := strings.TrimSpace(c.QueryValue("time_range"))
	if tr == "" {
		tr = defaultRange
	}
	dur, ok := parseOpsDuration(tr)
	if !ok {
		dur, _ = parseOpsDuration(defaultRange)
	}
	end = time.Now()
	start = end.Add(-dur)
	if end.Sub(start) > 30*24*time.Hour {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time range: max window is 30 days")
	}
	return start, end, nil
}

func parseOpsQueryModeGateway(c gatewayctx.GatewayContext) service.OpsQueryMode {
	raw := strings.TrimSpace(c.QueryValue("mode"))
	if raw == "" {
		return ""
	}
	return service.ParseOpsQueryMode(raw)
}

func parseOpsOpenAITokenStatsFilterGateway(c gatewayctx.GatewayContext) (*service.OpsOpenAITokenStatsFilter, error) {
	timeRange := strings.TrimSpace(c.QueryValue("time_range"))
	if timeRange == "" {
		timeRange = "30d"
	}
	dur, ok := parseOpsOpenAITokenStatsDuration(timeRange)
	if !ok {
		return nil, fmt.Errorf("invalid time_range")
	}
	end := time.Now().UTC()
	start := end.Add(-dur)
	filter := &service.OpsOpenAITokenStatsFilter{
		TimeRange: timeRange,
		StartTime: start,
		EndTime:   end,
		Platform:  strings.TrimSpace(c.QueryValue("platform")),
	}
	groupID, ok := parseOptionalInt64QueryGateway(c, "group_id")
	if !ok {
		return nil, fmt.Errorf("invalid group_id")
	}
	filter.GroupID = groupID
	topNRaw := strings.TrimSpace(c.QueryValue("top_n"))
	pageRaw := strings.TrimSpace(c.QueryValue("page"))
	pageSizeRaw := strings.TrimSpace(c.QueryValue("page_size"))
	if topNRaw != "" && (pageRaw != "" || pageSizeRaw != "") {
		return nil, fmt.Errorf("invalid query: top_n cannot be used with page/page_size")
	}
	if topNRaw != "" {
		topN, err := strconv.Atoi(topNRaw)
		if err != nil || topN < 1 || topN > 100 {
			return nil, fmt.Errorf("invalid top_n")
		}
		filter.TopN = topN
		return filter, nil
	}
	filter.Page = 1
	filter.PageSize = 20
	if pageRaw != "" {
		page, err := strconv.Atoi(pageRaw)
		if err != nil || page < 1 {
			return nil, fmt.Errorf("invalid page")
		}
		filter.Page = page
	}
	if pageSizeRaw != "" {
		pageSize, err := strconv.Atoi(pageSizeRaw)
		if err != nil || pageSize < 1 || pageSize > 100 {
			return nil, fmt.Errorf("invalid page_size")
		}
		filter.PageSize = pageSize
	}
	return filter, nil
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return &t, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseOptionalInt64QueryGateway(c gatewayctx.GatewayContext, key string) (*int64, bool) {
	raw := strings.TrimSpace(c.QueryValue(key))
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, false
	}
	return &id, true
}

func parseOptionalNonNegativeIntGateway(c gatewayctx.GatewayContext, key string) (int, bool) {
	raw := strings.TrimSpace(c.QueryValue(key))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}
