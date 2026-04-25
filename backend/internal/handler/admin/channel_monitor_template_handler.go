package admin

import (
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ChannelMonitorRequestTemplateHandler 请求模板管理后台 handler。
type ChannelMonitorRequestTemplateHandler struct {
	templateService *service.ChannelMonitorRequestTemplateService
}

// NewChannelMonitorRequestTemplateHandler 创建 handler。
func NewChannelMonitorRequestTemplateHandler(templateService *service.ChannelMonitorRequestTemplateService) *ChannelMonitorRequestTemplateHandler {
	return &ChannelMonitorRequestTemplateHandler{templateService: templateService}
}

// --- DTO ---

type channelMonitorTemplateCreateRequest struct {
	Name             string            `json:"name" binding:"required,max=100"`
	Provider         string            `json:"provider" binding:"required,oneof=openai anthropic gemini"`
	Description      string            `json:"description" binding:"max=500"`
	ExtraHeaders     map[string]string `json:"extra_headers"`
	BodyOverrideMode string            `json:"body_override_mode" binding:"omitempty,oneof=off merge replace"`
	BodyOverride     map[string]any    `json:"body_override"`
}

type channelMonitorTemplateUpdateRequest struct {
	Name             *string            `json:"name" binding:"omitempty,max=100"`
	Description      *string            `json:"description" binding:"omitempty,max=500"`
	ExtraHeaders     *map[string]string `json:"extra_headers"`
	BodyOverrideMode *string            `json:"body_override_mode" binding:"omitempty,oneof=off merge replace"`
	BodyOverride     *map[string]any    `json:"body_override"`
}

type channelMonitorTemplateResponse struct {
	ID                 int64             `json:"id"`
	Name               string            `json:"name"`
	Provider           string            `json:"provider"`
	Description        string            `json:"description"`
	ExtraHeaders       map[string]string `json:"extra_headers"`
	BodyOverrideMode   string            `json:"body_override_mode"`
	BodyOverride       map[string]any    `json:"body_override"`
	CreatedAt          string            `json:"created_at"`
	UpdatedAt          string            `json:"updated_at"`
	AssociatedMonitors int64             `json:"associated_monitors"`
}

func (h *ChannelMonitorRequestTemplateHandler) toResponse(c *gin.Context, t *service.ChannelMonitorRequestTemplate) *channelMonitorTemplateResponse {
	if t == nil {
		return nil
	}
	headers := t.ExtraHeaders
	if headers == nil {
		headers = map[string]string{}
	}
	count, _ := h.templateService.CountAssociatedMonitors(c.Request.Context(), t.ID)
	return &channelMonitorTemplateResponse{
		ID:                 t.ID,
		Name:               t.Name,
		Provider:           t.Provider,
		Description:        t.Description,
		ExtraHeaders:       headers,
		BodyOverrideMode:   t.BodyOverrideMode,
		BodyOverride:       t.BodyOverride,
		CreatedAt:          t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          t.UpdatedAt.UTC().Format(time.RFC3339),
		AssociatedMonitors: count,
	}
}

// parseTemplateID 提取并校验 :id。
func parseTemplateID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_TEMPLATE_ID", "invalid template id"))
		return 0, false
	}
	return id, true
}

func parseTemplateIDGateway(c gatewayctx.GatewayContext) (int64, bool) {
	id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, infraerrors.BadRequest("INVALID_TEMPLATE_ID", "invalid template id"))
		return 0, false
	}
	return id, true
}

// --- Handlers ---

// List GET /api/v1/admin/channel-monitor-templates?provider=anthropic
func (h *ChannelMonitorRequestTemplateHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *ChannelMonitorRequestTemplateHandler) ListGateway(c gatewayctx.GatewayContext) {
	items, err := h.templateService.List(c.Request().Context(), service.ChannelMonitorRequestTemplateListParams{
		Provider: strings.TrimSpace(c.QueryValue("provider")),
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	out := make([]*channelMonitorTemplateResponse, 0, len(items))
	for _, item := range items {
		out = append(out, h.toGatewayResponse(c, item))
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"items": out})
}

func (h *ChannelMonitorRequestTemplateHandler) ListLegacy(c *gin.Context) {
	items, err := h.templateService.List(c.Request.Context(), service.ChannelMonitorRequestTemplateListParams{
		Provider: strings.TrimSpace(c.Query("provider")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*channelMonitorTemplateResponse, 0, len(items))
	for _, t := range items {
		out = append(out, h.toResponse(c, t))
	}
	response.Success(c, gin.H{"items": out})
}

// Get GET /api/v1/admin/channel-monitor-templates/:id
func (h *ChannelMonitorRequestTemplateHandler) Get(c *gin.Context) {
	h.GetGateway(gatewayctx.FromGin(c))
}

func (h *ChannelMonitorRequestTemplateHandler) GetGateway(c gatewayctx.GatewayContext) {
	id, ok := parseTemplateIDGateway(c)
	if !ok {
		return
	}
	item, err := h.templateService.Get(c.Request().Context(), id)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, h.toGatewayResponse(c, item))
}

func (h *ChannelMonitorRequestTemplateHandler) GetLegacy(c *gin.Context) {
	id, ok := parseTemplateID(c)
	if !ok {
		return
	}
	t, err := h.templateService.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.toResponse(c, t))
}

// Create POST /api/v1/admin/channel-monitor-templates
func (h *ChannelMonitorRequestTemplateHandler) Create(c *gin.Context) {
	h.CreateGateway(gatewayctx.FromGin(c))
}

func (h *ChannelMonitorRequestTemplateHandler) CreateGateway(c gatewayctx.GatewayContext) {
	var req channelMonitorTemplateCreateRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.templateService.Create(c.Request().Context(), service.ChannelMonitorRequestTemplateCreateParams{
		Name:             req.Name,
		Provider:         req.Provider,
		Description:      req.Description,
		ExtraHeaders:     req.ExtraHeaders,
		BodyOverrideMode: req.BodyOverrideMode,
		BodyOverride:     req.BodyOverride,
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.CreatedContext(gatewayJSONResponder{ctx: c}, h.toGatewayResponse(c, item))
}

func (h *ChannelMonitorRequestTemplateHandler) CreateLegacy(c *gin.Context) {
	var req channelMonitorTemplateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	t, err := h.templateService.Create(c.Request.Context(), service.ChannelMonitorRequestTemplateCreateParams{
		Name:             req.Name,
		Provider:         req.Provider,
		Description:      req.Description,
		ExtraHeaders:     req.ExtraHeaders,
		BodyOverrideMode: req.BodyOverrideMode,
		BodyOverride:     req.BodyOverride,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, h.toResponse(c, t))
}

// Update PUT /api/v1/admin/channel-monitor-templates/:id
func (h *ChannelMonitorRequestTemplateHandler) Update(c *gin.Context) {
	h.UpdateGateway(gatewayctx.FromGin(c))
}

func (h *ChannelMonitorRequestTemplateHandler) UpdateGateway(c gatewayctx.GatewayContext) {
	id, ok := parseTemplateIDGateway(c)
	if !ok {
		return
	}
	var req channelMonitorTemplateUpdateRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.templateService.Update(c.Request().Context(), id, service.ChannelMonitorRequestTemplateUpdateParams{
		Name:             req.Name,
		Description:      req.Description,
		ExtraHeaders:     req.ExtraHeaders,
		BodyOverrideMode: req.BodyOverrideMode,
		BodyOverride:     req.BodyOverride,
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, h.toGatewayResponse(c, item))
}

func (h *ChannelMonitorRequestTemplateHandler) UpdateLegacy(c *gin.Context) {
	id, ok := parseTemplateID(c)
	if !ok {
		return
	}
	var req channelMonitorTemplateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	t, err := h.templateService.Update(c.Request.Context(), id, service.ChannelMonitorRequestTemplateUpdateParams{
		Name:             req.Name,
		Description:      req.Description,
		ExtraHeaders:     req.ExtraHeaders,
		BodyOverrideMode: req.BodyOverrideMode,
		BodyOverride:     req.BodyOverride,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.toResponse(c, t))
}

// Delete DELETE /api/v1/admin/channel-monitor-templates/:id
func (h *ChannelMonitorRequestTemplateHandler) Delete(c *gin.Context) {
	h.DeleteGateway(gatewayctx.FromGin(c))
}

func (h *ChannelMonitorRequestTemplateHandler) DeleteGateway(c gatewayctx.GatewayContext) {
	id, ok := parseTemplateIDGateway(c)
	if !ok {
		return
	}
	if err := h.templateService.Delete(c.Request().Context(), id); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, nil)
}

func (h *ChannelMonitorRequestTemplateHandler) DeleteLegacy(c *gin.Context) {
	id, ok := parseTemplateID(c)
	if !ok {
		return
	}
	if err := h.templateService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

type channelMonitorTemplateApplyRequest struct {
	// MonitorIDs 必填、非空：用户在 picker 里勾选的要被覆盖的监控 ID 列表。
	// 仅当对应监控当前 template_id == :id 时才会真的被覆盖。
	MonitorIDs []int64 `json:"monitor_ids" binding:"required,min=1"`
}

// Apply POST /api/v1/admin/channel-monitor-templates/:id/apply
// 把模板当前配置覆盖到 monitor_ids 列表里的关联监控（picker 选中的子集）。
func (h *ChannelMonitorRequestTemplateHandler) Apply(c *gin.Context) {
	h.ApplyGateway(gatewayctx.FromGin(c))
}

func (h *ChannelMonitorRequestTemplateHandler) ApplyGateway(c gatewayctx.GatewayContext) {
	id, ok := parseTemplateIDGateway(c)
	if !ok {
		return
	}
	var req channelMonitorTemplateApplyRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	affected, err := h.templateService.ApplyToMonitors(c.Request().Context(), id, req.MonitorIDs)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"affected": affected})
}

func (h *ChannelMonitorRequestTemplateHandler) ApplyLegacy(c *gin.Context) {
	id, ok := parseTemplateID(c)
	if !ok {
		return
	}
	var req channelMonitorTemplateApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	affected, err := h.templateService.ApplyToMonitors(c.Request.Context(), id, req.MonitorIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"affected": affected})
}

type associatedMonitorBriefResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
}

// AssociatedMonitors GET /api/v1/admin/channel-monitor-templates/:id/monitors
// 列出关联监控（picker 弹窗用）。
func (h *ChannelMonitorRequestTemplateHandler) AssociatedMonitors(c *gin.Context) {
	h.AssociatedMonitorsGateway(gatewayctx.FromGin(c))
}

func (h *ChannelMonitorRequestTemplateHandler) AssociatedMonitorsGateway(c gatewayctx.GatewayContext) {
	id, ok := parseTemplateIDGateway(c)
	if !ok {
		return
	}
	items, err := h.templateService.ListAssociatedMonitors(c.Request().Context(), id)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	out := make([]associatedMonitorBriefResponse, 0, len(items))
	for _, m := range items {
		out = append(out, associatedMonitorBriefResponse{
			ID: m.ID, Name: m.Name, Provider: m.Provider, Enabled: m.Enabled,
		})
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"items": out})
}

func (h *ChannelMonitorRequestTemplateHandler) AssociatedMonitorsLegacy(c *gin.Context) {
	id, ok := parseTemplateID(c)
	if !ok {
		return
	}
	items, err := h.templateService.ListAssociatedMonitors(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]associatedMonitorBriefResponse, 0, len(items))
	for _, m := range items {
		out = append(out, associatedMonitorBriefResponse{
			ID: m.ID, Name: m.Name, Provider: m.Provider, Enabled: m.Enabled,
		})
	}
	response.Success(c, gin.H{"items": out})
}

func (h *ChannelMonitorRequestTemplateHandler) toGatewayResponse(c gatewayctx.GatewayContext, t *service.ChannelMonitorRequestTemplate) *channelMonitorTemplateResponse {
	if t == nil {
		return nil
	}
	headers := t.ExtraHeaders
	if headers == nil {
		headers = map[string]string{}
	}
	count, _ := h.templateService.CountAssociatedMonitors(c.Request().Context(), t.ID)
	return &channelMonitorTemplateResponse{
		ID:                 t.ID,
		Name:               t.Name,
		Provider:           t.Provider,
		Description:        t.Description,
		ExtraHeaders:       headers,
		BodyOverrideMode:   t.BodyOverrideMode,
		BodyOverride:       t.BodyOverride,
		CreatedAt:          t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          t.UpdatedAt.UTC().Format(time.RFC3339),
		AssociatedMonitors: count,
	}
}
