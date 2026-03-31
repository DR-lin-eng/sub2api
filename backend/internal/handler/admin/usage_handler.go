package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UsageHandler handles admin usage-related requests
type UsageHandler struct {
	usageService   *service.UsageService
	apiKeyService  *service.APIKeyService
	adminService   service.AdminService
	cleanupService *service.UsageCleanupService
}

// NewUsageHandler creates a new admin usage handler
func NewUsageHandler(
	usageService *service.UsageService,
	apiKeyService *service.APIKeyService,
	adminService service.AdminService,
	cleanupService *service.UsageCleanupService,
) *UsageHandler {
	return &UsageHandler{
		usageService:   usageService,
		apiKeyService:  apiKeyService,
		adminService:   adminService,
		cleanupService: cleanupService,
	}
}

// CreateUsageCleanupTaskRequest represents cleanup task creation request
type CreateUsageCleanupTaskRequest struct {
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	UserID      *int64  `json:"user_id"`
	APIKeyID    *int64  `json:"api_key_id"`
	AccountID   *int64  `json:"account_id"`
	GroupID     *int64  `json:"group_id"`
	Model       *string `json:"model"`
	RequestType *string `json:"request_type"`
	Stream      *bool   `json:"stream"`
	BillingType *int8   `json:"billing_type"`
	Timezone    string  `json:"timezone"`
}

// List handles listing all usage records with filters
// GET /api/v1/admin/usage
func (h *UsageHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) ListGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	exactTotal := false
	if exactTotalRaw := strings.TrimSpace(c.QueryValue("exact_total")); exactTotalRaw != "" {
		parsed, err := strconv.ParseBool(exactTotalRaw)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid exact_total value, use true or false")
			return
		}
		exactTotal = parsed
	}

	// Parse filters
	var userID, apiKeyID, accountID, groupID int64
	if userIDStr := c.QueryValue("user_id"); userIDStr != "" {
		id, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid user_id")
			return
		}
		userID = id
	}

	if apiKeyIDStr := c.QueryValue("api_key_id"); apiKeyIDStr != "" {
		id, err := strconv.ParseInt(apiKeyIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid api_key_id")
			return
		}
		apiKeyID = id
	}

	if accountIDStr := c.QueryValue("account_id"); accountIDStr != "" {
		id, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid account_id")
			return
		}
		accountID = id
	}

	if groupIDStr := c.QueryValue("group_id"); groupIDStr != "" {
		id, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
			return
		}
		groupID = id
	}

	model := c.QueryValue("model")

	var requestType *int16
	var stream *bool
	if requestTypeStr := strings.TrimSpace(c.QueryValue("request_type")); requestTypeStr != "" {
		parsed, err := service.ParseUsageRequestType(requestTypeStr)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		value := int16(parsed)
		requestType = &value
	} else if streamStr := c.QueryValue("stream"); streamStr != "" {
		val, err := strconv.ParseBool(streamStr)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid stream value, use true or false")
			return
		}
		stream = &val
	}

	var billingType *int8
	if billingTypeStr := c.QueryValue("billing_type"); billingTypeStr != "" {
		val, err := strconv.ParseInt(billingTypeStr, 10, 8)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid billing_type")
			return
		}
		bt := int8(val)
		billingType = &bt
	}

	// Parse date range
	var startTime, endTime *time.Time
	userTZ := c.QueryValue("timezone")
	if startDateStr := c.QueryValue("start_date"); startDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid start_date format, use YYYY-MM-DD")
			return
		}
		startTime = &t
	}

	if endDateStr := c.QueryValue("end_date"); endDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid end_date format, use YYYY-MM-DD")
			return
		}
		// Use half-open range [start, end), move to next calendar day start (DST-safe).
		t = t.AddDate(0, 0, 1)
		endTime = &t
	}

	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	filters := usagestats.UsageLogFilters{
		UserID:      userID,
		APIKeyID:    apiKeyID,
		AccountID:   accountID,
		GroupID:     groupID,
		Model:       model,
		RequestType: requestType,
		Stream:      stream,
		BillingType: billingType,
		StartTime:   startTime,
		EndTime:     endTime,
		ExactTotal:  exactTotal,
	}

	records, result, err := h.usageService.ListWithFilters(c.Request().Context(), params, filters)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	out := make([]dto.AdminUsageLog, 0, len(records))
	for i := range records {
		out = append(out, *dto.UsageLogFromServiceAdmin(&records[i]))
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, out, result.Total, page, pageSize)
}

// Stats handles getting usage statistics with filters
// GET /api/v1/admin/usage/stats
func (h *UsageHandler) Stats(c *gin.Context) {
	h.StatsGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) StatsGateway(c gatewayctx.GatewayContext) {
	// Parse filters - same as List endpoint
	var userID, apiKeyID, accountID, groupID int64
	if userIDStr := c.QueryValue("user_id"); userIDStr != "" {
		id, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid user_id")
			return
		}
		userID = id
	}

	if apiKeyIDStr := c.QueryValue("api_key_id"); apiKeyIDStr != "" {
		id, err := strconv.ParseInt(apiKeyIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid api_key_id")
			return
		}
		apiKeyID = id
	}

	if accountIDStr := c.QueryValue("account_id"); accountIDStr != "" {
		id, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid account_id")
			return
		}
		accountID = id
	}

	if groupIDStr := c.QueryValue("group_id"); groupIDStr != "" {
		id, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid group_id")
			return
		}
		groupID = id
	}

	model := c.QueryValue("model")

	var requestType *int16
	var stream *bool
	if requestTypeStr := strings.TrimSpace(c.QueryValue("request_type")); requestTypeStr != "" {
		parsed, err := service.ParseUsageRequestType(requestTypeStr)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		value := int16(parsed)
		requestType = &value
	} else if streamStr := c.QueryValue("stream"); streamStr != "" {
		val, err := strconv.ParseBool(streamStr)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid stream value, use true or false")
			return
		}
		stream = &val
	}

	var billingType *int8
	if billingTypeStr := c.QueryValue("billing_type"); billingTypeStr != "" {
		val, err := strconv.ParseInt(billingTypeStr, 10, 8)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid billing_type")
			return
		}
		bt := int8(val)
		billingType = &bt
	}

	// Parse date range
	userTZ := c.QueryValue("timezone")
	now := timezone.NowInUserLocation(userTZ)
	var startTime, endTime time.Time

	startDateStr := c.QueryValue("start_date")
	endDateStr := c.QueryValue("end_date")

	if startDateStr != "" && endDateStr != "" {
		var err error
		startTime, err = timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid start_date format, use YYYY-MM-DD")
			return
		}
		endTime, err = timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid end_date format, use YYYY-MM-DD")
			return
		}
		// 与 SQL 条件 created_at < end 对齐，使用次日 00:00 作为上边界（DST-safe）。
		endTime = endTime.AddDate(0, 0, 1)
	} else {
		period := defaultQueryValue(c, "period", "today")
		switch period {
		case "today":
			startTime = timezone.StartOfDayInUserLocation(now, userTZ)
		case "week":
			startTime = now.AddDate(0, 0, -7)
		case "month":
			startTime = now.AddDate(0, -1, 0)
		default:
			startTime = timezone.StartOfDayInUserLocation(now, userTZ)
		}
		endTime = now
	}

	// Build filters and call GetStatsWithFilters
	filters := usagestats.UsageLogFilters{
		UserID:      userID,
		APIKeyID:    apiKeyID,
		AccountID:   accountID,
		GroupID:     groupID,
		Model:       model,
		RequestType: requestType,
		Stream:      stream,
		BillingType: billingType,
		StartTime:   &startTime,
		EndTime:     &endTime,
	}

	stats, err := h.usageService.GetStatsWithFilters(c.Request().Context(), filters)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, stats)
}

// SearchUsers handles searching users by email keyword
// GET /api/v1/admin/usage/search-users
func (h *UsageHandler) SearchUsers(c *gin.Context) {
	h.SearchUsersGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) SearchUsersGateway(c gatewayctx.GatewayContext) {
	keyword := c.QueryValue("q")
	if keyword == "" {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, []any{})
		return
	}

	// Limit to 30 results
	users, _, err := h.adminService.ListUsers(c.Request().Context(), 1, 30, service.UserListFilters{Search: keyword})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	// Return simplified user list (only id and email)
	type SimpleUser struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}

	result := make([]SimpleUser, len(users))
	for i, u := range users {
		result[i] = SimpleUser{
			ID:    u.ID,
			Email: u.Email,
		}
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

// SearchAPIKeys handles searching API keys by user
// GET /api/v1/admin/usage/search-api-keys
func (h *UsageHandler) SearchAPIKeys(c *gin.Context) {
	h.SearchAPIKeysGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) SearchAPIKeysGateway(c gatewayctx.GatewayContext) {
	userIDStr := c.QueryValue("user_id")
	keyword := c.QueryValue("q")

	var userID int64
	if userIDStr != "" {
		id, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid user_id")
			return
		}
		userID = id
	}

	keys, err := h.apiKeyService.SearchAPIKeys(c.Request().Context(), userID, keyword, 30)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	// Return simplified API key list (only id and name)
	type SimpleAPIKey struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		UserID int64  `json:"user_id"`
	}

	result := make([]SimpleAPIKey, len(keys))
	for i, k := range keys {
		result[i] = SimpleAPIKey{
			ID:     k.ID,
			Name:   k.Name,
			UserID: k.UserID,
		}
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

// ListCleanupTasks handles listing usage cleanup tasks
// GET /api/v1/admin/usage/cleanup-tasks
func (h *UsageHandler) ListCleanupTasks(c *gin.Context) {
	h.ListCleanupTasksGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) ListCleanupTasksGateway(c gatewayctx.GatewayContext) {
	if h.cleanupService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Usage cleanup service unavailable")
		return
	}
	operator := int64(0)
	if subject, ok := middleware.GetAuthSubjectFromGatewayContext(c); ok {
		operator = subject.UserID
	}
	page, pageSize := response.ParsePaginationValues(c)
	logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 请求清理任务列表: operator=%d page=%d page_size=%d", operator, page, pageSize)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	tasks, result, err := h.cleanupService.ListTasks(c.Request().Context(), params)
	if err != nil {
		logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 查询清理任务列表失败: operator=%d page=%d page_size=%d err=%v", operator, page, pageSize, err)
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	out := make([]dto.UsageCleanupTask, 0, len(tasks))
	for i := range tasks {
		out = append(out, *dto.UsageCleanupTaskFromService(&tasks[i]))
	}
	logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 返回清理任务列表: operator=%d total=%d items=%d page=%d page_size=%d", operator, result.Total, len(out), page, pageSize)
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, out, result.Total, page, pageSize)
}

// CreateCleanupTask handles creating a usage cleanup task
// POST /api/v1/admin/usage/cleanup-tasks
func (h *UsageHandler) CreateCleanupTask(c *gin.Context) {
	h.CreateCleanupTaskGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) CreateCleanupTaskGateway(c gatewayctx.GatewayContext) {
	if h.cleanupService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Usage cleanup service unavailable")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateUsageCleanupTaskRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	req.StartDate = strings.TrimSpace(req.StartDate)
	req.EndDate = strings.TrimSpace(req.EndDate)
	if req.StartDate == "" || req.EndDate == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	startTime, err := timezone.ParseInUserLocation("2006-01-02", req.StartDate, req.Timezone)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid start_date format, use YYYY-MM-DD")
		return
	}
	endTime, err := timezone.ParseInUserLocation("2006-01-02", req.EndDate, req.Timezone)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid end_date format, use YYYY-MM-DD")
		return
	}
	endTime = endTime.Add(24*time.Hour - time.Nanosecond)

	var requestType *int16
	stream := req.Stream
	if req.RequestType != nil {
		parsed, err := service.ParseUsageRequestType(*req.RequestType)
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		value := int16(parsed)
		requestType = &value
		stream = nil
	}

	filters := service.UsageCleanupFilters{
		StartTime:   startTime,
		EndTime:     endTime,
		UserID:      req.UserID,
		APIKeyID:    req.APIKeyID,
		AccountID:   req.AccountID,
		GroupID:     req.GroupID,
		Model:       req.Model,
		RequestType: requestType,
		Stream:      stream,
		BillingType: req.BillingType,
	}

	var userID any
	if filters.UserID != nil {
		userID = *filters.UserID
	}
	var apiKeyID any
	if filters.APIKeyID != nil {
		apiKeyID = *filters.APIKeyID
	}
	var accountID any
	if filters.AccountID != nil {
		accountID = *filters.AccountID
	}
	var groupID any
	if filters.GroupID != nil {
		groupID = *filters.GroupID
	}
	var model any
	if filters.Model != nil {
		model = *filters.Model
	}
	var streamValue any
	if filters.Stream != nil {
		streamValue = *filters.Stream
	}
	var requestTypeName any
	if filters.RequestType != nil {
		requestTypeName = service.RequestTypeFromInt16(*filters.RequestType).String()
	}
	var billingType any
	if filters.BillingType != nil {
		billingType = *filters.BillingType
	}

	idempotencyPayload := struct {
		OperatorID int64                         `json:"operator_id"`
		Body       CreateUsageCleanupTaskRequest `json:"body"`
	}{
		OperatorID: subject.UserID,
		Body:       req,
	}
	executeAdminIdempotentGatewayJSON(c, "admin.usage.cleanup_tasks.create", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 请求创建清理任务: operator=%d start=%s end=%s user_id=%v api_key_id=%v account_id=%v group_id=%v model=%v request_type=%v stream=%v billing_type=%v tz=%q",
			subject.UserID,
			filters.StartTime.Format(time.RFC3339),
			filters.EndTime.Format(time.RFC3339),
			userID,
			apiKeyID,
			accountID,
			groupID,
			model,
			requestTypeName,
			streamValue,
			billingType,
			req.Timezone,
		)

		task, err := h.cleanupService.CreateTask(ctx, filters, subject.UserID)
		if err != nil {
			logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 创建清理任务失败: operator=%d err=%v", subject.UserID, err)
			return nil, err
		}
		logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 清理任务已创建: task=%d operator=%d status=%s", task.ID, subject.UserID, task.Status)
		return dto.UsageCleanupTaskFromService(task), nil
	})
}

// CancelCleanupTask handles canceling a usage cleanup task
// POST /api/v1/admin/usage/cleanup-tasks/:id/cancel
func (h *UsageHandler) CancelCleanupTask(c *gin.Context) {
	h.CancelCleanupTaskGateway(gatewayctx.FromGin(c))
}

func (h *UsageHandler) CancelCleanupTaskGateway(c gatewayctx.GatewayContext) {
	if h.cleanupService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Usage cleanup service unavailable")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}
	idStr := strings.TrimSpace(c.PathParam("id"))
	taskID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || taskID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid task id")
		return
	}
	logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 请求取消清理任务: task=%d operator=%d", taskID, subject.UserID)
	if err := h.cleanupService.CancelTask(c.Request().Context(), taskID, subject.UserID); err != nil {
		logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 取消清理任务失败: task=%d operator=%d err=%v", taskID, subject.UserID, err)
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	logger.LegacyPrintf("handler.admin.usage", "[UsageCleanup] 清理任务已取消: task=%d operator=%d", taskID, subject.UserID)
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"id": taskID, "status": service.UsageCleanupStatusCanceled})
}
