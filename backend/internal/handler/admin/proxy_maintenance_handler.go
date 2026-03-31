package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ProxyMaintenanceHandler struct {
	svc         *service.ProxyMaintenanceService
	taskManager *proxyMaintenanceTaskManager
}

func NewProxyMaintenanceHandler(svc *service.ProxyMaintenanceService) *ProxyMaintenanceHandler {
	return &ProxyMaintenanceHandler{svc: svc, taskManager: defaultProxyMaintenanceTaskManager()}
}

type createProxyMaintenancePlanRequest struct {
	Name                   string  `json:"name"`
	CronExpression         string  `json:"cron_expression" binding:"required"`
	Enabled                *bool   `json:"enabled"`
	SourceProxyIDs         []int64 `json:"source_proxy_ids"`
	MaxResults             int     `json:"max_results"`
	MaxFailuresBeforePause int     `json:"max_failures_before_pause"`
}

type updateProxyMaintenancePlanRequest struct {
	Name                   string  `json:"name"`
	CronExpression         string  `json:"cron_expression"`
	Enabled                *bool   `json:"enabled"`
	SourceProxyIDs         []int64 `json:"source_proxy_ids"`
	MaxResults             int     `json:"max_results"`
	MaxFailuresBeforePause int     `json:"max_failures_before_pause"`
}

type runProxyMaintenanceRequest struct {
	SourceProxyIDs []int64 `json:"source_proxy_ids"`
}

func (h *ProxyMaintenanceHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *ProxyMaintenanceHandler) ListGateway(c gatewayctx.GatewayContext) {
	plans, err := h.svc.ListPlans(c.Request().Context())
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, plans)
}

func (h *ProxyMaintenanceHandler) Create(c *gin.Context) {
	h.CreateGateway(gatewayctx.FromGin(c))
}

func (h *ProxyMaintenanceHandler) CreateGateway(c gatewayctx.GatewayContext) {
	var req createProxyMaintenancePlanRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	plan := &service.ProxyMaintenancePlan{
		Name:                   req.Name,
		CronExpression:         req.CronExpression,
		Enabled:                true,
		SourceProxyIDs:         req.SourceProxyIDs,
		MaxResults:             req.MaxResults,
		MaxFailuresBeforePause: req.MaxFailuresBeforePause,
	}
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	created, err := h.svc.CreatePlan(c.Request().Context(), plan)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, created)
}

func (h *ProxyMaintenanceHandler) Update(c *gin.Context) {
	h.UpdateGateway(gatewayctx.FromGin(c))
}

func (h *ProxyMaintenanceHandler) UpdateGateway(c gatewayctx.GatewayContext) {
	id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "invalid plan id")
		return
	}
	plan, err := h.svc.GetPlan(c.Request().Context(), id)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusNotFound, "plan not found")
		return
	}
	var req updateProxyMaintenancePlanRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != "" {
		plan.Name = req.Name
	}
	if req.CronExpression != "" {
		plan.CronExpression = req.CronExpression
	}
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	if req.SourceProxyIDs != nil {
		plan.SourceProxyIDs = req.SourceProxyIDs
	}
	if req.MaxResults > 0 {
		plan.MaxResults = req.MaxResults
	}
	if req.MaxFailuresBeforePause > 0 {
		plan.MaxFailuresBeforePause = req.MaxFailuresBeforePause
	}
	updated, err := h.svc.UpdatePlan(c.Request().Context(), plan)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}

func (h *ProxyMaintenanceHandler) Delete(c *gin.Context) {
	h.DeleteGateway(gatewayctx.FromGin(c))
}

func (h *ProxyMaintenanceHandler) DeleteGateway(c gatewayctx.GatewayContext) {
	id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "invalid plan id")
		return
	}
	if err := h.svc.DeletePlan(c.Request().Context(), id); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "deleted"})
}

func (h *ProxyMaintenanceHandler) ListResults(c *gin.Context) {
	h.ListResultsGateway(gatewayctx.FromGin(c))
}

func (h *ProxyMaintenanceHandler) ListResultsGateway(c gatewayctx.GatewayContext) {
	id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "invalid plan id")
		return
	}
	limit := 50
	if l, err := strconv.Atoi(c.QueryValue("limit")); err == nil && l > 0 {
		limit = l
	}
	results, err := h.svc.ListResults(c.Request().Context(), id, limit)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, results)
}

func (h *ProxyMaintenanceHandler) RunNow(c *gin.Context) {
	h.RunNowGateway(gatewayctx.FromGin(c))
}

func (h *ProxyMaintenanceHandler) RunNowGateway(c gatewayctx.GatewayContext) {
	var req runProxyMaintenanceRequest
	if c.Request() != nil && c.Request().ContentLength > 0 {
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
	}
	task := h.taskManager.createTask()
	if task == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "proxy maintenance task manager not available")
		return
	}
	task.execute = func(ctx context.Context, task *proxyMaintenanceTask) (*service.ProxyMaintenanceResult, error) {
		return h.svc.RunNow(ctx, req.SourceProxyIDs)
	}
	if err := h.taskManager.submitTask(task); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusTooManyRequests, err.Error())
		return
	}
	response.AcceptedContext(gatewayJSONResponder{ctx: c}, task.state)
}

func (h *ProxyMaintenanceHandler) GetTask(c *gin.Context) {
	h.GetTaskGateway(gatewayctx.FromGin(c))
}

func (h *ProxyMaintenanceHandler) GetTaskGateway(c gatewayctx.GatewayContext) {
	taskID := strings.TrimSpace(c.PathParam("task_id"))
	if taskID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "task_id is required")
		return
	}
	task, ok := h.taskManager.getTask(taskID)
	if !ok || task == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusNotFound, "task not found")
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, task)
}
