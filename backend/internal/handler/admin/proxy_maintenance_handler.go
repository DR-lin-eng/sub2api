package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ProxyMaintenanceHandler struct {
	svc *service.ProxyMaintenanceService
}

func NewProxyMaintenanceHandler(svc *service.ProxyMaintenanceService) *ProxyMaintenanceHandler {
	return &ProxyMaintenanceHandler{svc: svc}
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
	plans, err := h.svc.ListPlans(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, plans)
}

func (h *ProxyMaintenanceHandler) Create(c *gin.Context) {
	var req createProxyMaintenancePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
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
	created, err := h.svc.CreatePlan(c.Request.Context(), plan)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, created)
}

func (h *ProxyMaintenanceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}
	plan, err := h.svc.GetPlan(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "plan not found")
		return
	}
	var req updateProxyMaintenancePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
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
	updated, err := h.svc.UpdatePlan(c.Request.Context(), plan)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *ProxyMaintenanceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}
	if err := h.svc.DeletePlan(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *ProxyMaintenanceHandler) ListResults(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}
	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	results, err := h.svc.ListResults(c.Request.Context(), id, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, results)
}

func (h *ProxyMaintenanceHandler) RunNow(c *gin.Context) {
	var req runProxyMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.RunNow(c.Request.Context(), req.SourceProxyIDs)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}
