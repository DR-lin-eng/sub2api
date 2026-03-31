package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ScheduledTestHandler handles admin scheduled-test-plan management.
type ScheduledTestHandler struct {
	scheduledTestSvc *service.ScheduledTestService
}

// NewScheduledTestHandler creates a new ScheduledTestHandler.
func NewScheduledTestHandler(scheduledTestSvc *service.ScheduledTestService) *ScheduledTestHandler {
	return &ScheduledTestHandler{scheduledTestSvc: scheduledTestSvc}
}

type createScheduledTestPlanRequest struct {
	AccountID              int64  `json:"account_id" binding:"required"`
	ModelID                string `json:"model_id"`
	CronExpression         string `json:"cron_expression" binding:"required"`
	Enabled                *bool  `json:"enabled"`
	MaxResults             int    `json:"max_results"`
	AutoRecover            *bool  `json:"auto_recover"`
	MaxFailuresBeforePause int    `json:"max_failures_before_pause"`
}

type updateScheduledTestPlanRequest struct {
	ModelID                string `json:"model_id"`
	CronExpression         string `json:"cron_expression"`
	Enabled                *bool  `json:"enabled"`
	MaxResults             int    `json:"max_results"`
	AutoRecover            *bool  `json:"auto_recover"`
	MaxFailuresBeforePause int    `json:"max_failures_before_pause"`
}

// ListByAccount GET /admin/accounts/:id/scheduled-test-plans
func (h *ScheduledTestHandler) ListByAccount(c *gin.Context) {
	h.ListByAccountGateway(gatewayctx.FromGin(c))
}

func (h *ScheduledTestHandler) ListByAccountGateway(c gatewayctx.GatewayContext) {
	accountID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "invalid account id")
		return
	}

	plans, err := h.scheduledTestSvc.ListPlansByAccount(c.Request().Context(), accountID)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, plans)
}

// Create POST /admin/scheduled-test-plans
func (h *ScheduledTestHandler) Create(c *gin.Context) {
	h.CreateGateway(gatewayctx.FromGin(c))
}

func (h *ScheduledTestHandler) CreateGateway(c gatewayctx.GatewayContext) {
	var req createScheduledTestPlanRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	plan := &service.ScheduledTestPlan{
		AccountID:              req.AccountID,
		ModelID:                req.ModelID,
		CronExpression:         req.CronExpression,
		Enabled:                true,
		MaxResults:             req.MaxResults,
		MaxFailuresBeforePause: req.MaxFailuresBeforePause,
	}
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	if req.AutoRecover != nil {
		plan.AutoRecover = *req.AutoRecover
	}

	created, err := h.scheduledTestSvc.CreatePlan(c.Request().Context(), plan)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, created)
}

// Update PUT /admin/scheduled-test-plans/:id
func (h *ScheduledTestHandler) Update(c *gin.Context) {
	h.UpdateGateway(gatewayctx.FromGin(c))
}

func (h *ScheduledTestHandler) UpdateGateway(c gatewayctx.GatewayContext) {
	planID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "invalid plan id")
		return
	}

	existing, err := h.scheduledTestSvc.GetPlan(c.Request().Context(), planID)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusNotFound, "plan not found")
		return
	}

	var req updateScheduledTestPlanRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	if req.ModelID != "" {
		existing.ModelID = req.ModelID
	}
	if req.CronExpression != "" {
		existing.CronExpression = req.CronExpression
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.MaxResults > 0 {
		existing.MaxResults = req.MaxResults
	}
	if req.AutoRecover != nil {
		existing.AutoRecover = *req.AutoRecover
	}
	if req.MaxFailuresBeforePause > 0 {
		existing.MaxFailuresBeforePause = req.MaxFailuresBeforePause
	}

	updated, err := h.scheduledTestSvc.UpdatePlan(c.Request().Context(), existing)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}

// Delete DELETE /admin/scheduled-test-plans/:id
func (h *ScheduledTestHandler) Delete(c *gin.Context) {
	h.DeleteGateway(gatewayctx.FromGin(c))
}

func (h *ScheduledTestHandler) DeleteGateway(c gatewayctx.GatewayContext) {
	planID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "invalid plan id")
		return
	}

	if err := h.scheduledTestSvc.DeletePlan(c.Request().Context(), planID); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "deleted"})
}

// ListResults GET /admin/scheduled-test-plans/:id/results
func (h *ScheduledTestHandler) ListResults(c *gin.Context) {
	h.ListResultsGateway(gatewayctx.FromGin(c))
}

func (h *ScheduledTestHandler) ListResultsGateway(c gatewayctx.GatewayContext) {
	planID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "invalid plan id")
		return
	}

	limit := 50
	if l, err := strconv.Atoi(c.QueryValue("limit")); err == nil && l > 0 {
		limit = l
	}

	results, err := h.scheduledTestSvc.ListResults(c.Request().Context(), planID, limit)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, results)
}
