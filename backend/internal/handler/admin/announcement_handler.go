package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler handles admin announcement management
type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
}

// NewAnnouncementHandler creates a new admin announcement handler
func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
	}
}

type CreateAnnouncementRequest struct {
	Title      string                        `json:"title" binding:"required"`
	Content    string                        `json:"content" binding:"required"`
	Status     string                        `json:"status" binding:"omitempty,oneof=draft active archived"`
	NotifyMode string                        `json:"notify_mode" binding:"omitempty,oneof=silent popup"`
	Targeting  service.AnnouncementTargeting `json:"targeting"`
	StartsAt   *int64                        `json:"starts_at"` // Unix seconds, 0/empty = immediate
	EndsAt     *int64                        `json:"ends_at"`   // Unix seconds, 0/empty = never
}

type UpdateAnnouncementRequest struct {
	Title      *string                        `json:"title"`
	Content    *string                        `json:"content"`
	Status     *string                        `json:"status" binding:"omitempty,oneof=draft active archived"`
	NotifyMode *string                        `json:"notify_mode" binding:"omitempty,oneof=silent popup"`
	Targeting  *service.AnnouncementTargeting `json:"targeting"`
	StartsAt   *int64                         `json:"starts_at"` // Unix seconds, 0 = clear
	EndsAt     *int64                         `json:"ends_at"`   // Unix seconds, 0 = clear
}

// List handles listing announcements with filters
// GET /api/v1/admin/announcements
func (h *AnnouncementHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *AnnouncementHandler) ListGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	status := strings.TrimSpace(c.QueryValue("status"))
	search := strings.TrimSpace(c.QueryValue("search"))
	if len(search) > 200 {
		search = search[:200]
	}

	params := pagination.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}

	items, paginationResult, err := h.announcementService.List(
		c.Request().Context(),
		params,
		service.AnnouncementListFilters{Status: status, Search: search},
	)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	out := make([]dto.Announcement, 0, len(items))
	for i := range items {
		out = append(out, *dto.AnnouncementFromService(&items[i]))
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, out, paginationResult.Total, page, pageSize)
}

// GetByID handles getting an announcement by ID
// GET /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) GetByID(c *gin.Context) {
	h.GetByIDGateway(gatewayctx.FromGin(c))
}

func (h *AnnouncementHandler) GetByIDGateway(c gatewayctx.GatewayContext) {
	announcementID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid announcement ID")
		return
	}

	item, err := h.announcementService.GetByID(c.Request().Context(), announcementID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.AnnouncementFromService(item))
}

// Create handles creating a new announcement
// POST /api/v1/admin/announcements
func (h *AnnouncementHandler) Create(c *gin.Context) {
	h.CreateGateway(gatewayctx.FromGin(c))
}

func (h *AnnouncementHandler) CreateGateway(c gatewayctx.GatewayContext) {
	var req CreateAnnouncementRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not found in context")
		return
	}

	input := &service.CreateAnnouncementInput{
		Title:      req.Title,
		Content:    req.Content,
		Status:     req.Status,
		NotifyMode: req.NotifyMode,
		Targeting:  req.Targeting,
		ActorID:    &subject.UserID,
	}

	if req.StartsAt != nil && *req.StartsAt > 0 {
		t := time.Unix(*req.StartsAt, 0)
		input.StartsAt = &t
	}
	if req.EndsAt != nil && *req.EndsAt > 0 {
		t := time.Unix(*req.EndsAt, 0)
		input.EndsAt = &t
	}

	created, err := h.announcementService.Create(c.Request().Context(), input)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.AnnouncementFromService(created))
}

// Update handles updating an announcement
// PUT /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) Update(c *gin.Context) {
	h.UpdateGateway(gatewayctx.FromGin(c))
}

func (h *AnnouncementHandler) UpdateGateway(c gatewayctx.GatewayContext) {
	announcementID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid announcement ID")
		return
	}

	var req UpdateAnnouncementRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not found in context")
		return
	}

	input := &service.UpdateAnnouncementInput{
		Title:      req.Title,
		Content:    req.Content,
		Status:     req.Status,
		NotifyMode: req.NotifyMode,
		Targeting:  req.Targeting,
		ActorID:    &subject.UserID,
	}

	if req.StartsAt != nil {
		if *req.StartsAt == 0 {
			var cleared *time.Time = nil
			input.StartsAt = &cleared
		} else {
			t := time.Unix(*req.StartsAt, 0)
			ptr := &t
			input.StartsAt = &ptr
		}
	}

	if req.EndsAt != nil {
		if *req.EndsAt == 0 {
			var cleared *time.Time = nil
			input.EndsAt = &cleared
		} else {
			t := time.Unix(*req.EndsAt, 0)
			ptr := &t
			input.EndsAt = &ptr
		}
	}

	updated, err := h.announcementService.Update(c.Request().Context(), announcementID, input)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.AnnouncementFromService(updated))
}

// Delete handles deleting an announcement
// DELETE /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	h.DeleteGateway(gatewayctx.FromGin(c))
}

func (h *AnnouncementHandler) DeleteGateway(c gatewayctx.GatewayContext) {
	announcementID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid announcement ID")
		return
	}

	if err := h.announcementService.Delete(c.Request().Context(), announcementID); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "Announcement deleted successfully"})
}

// ListReadStatus handles listing users read status for an announcement
// GET /api/v1/admin/announcements/:id/read-status
func (h *AnnouncementHandler) ListReadStatus(c *gin.Context) {
	h.ListReadStatusGateway(gatewayctx.FromGin(c))
}

func (h *AnnouncementHandler) ListReadStatusGateway(c gatewayctx.GatewayContext) {
	announcementID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid announcement ID")
		return
	}

	page, pageSize := response.ParsePaginationValues(c)
	params := pagination.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
	search := strings.TrimSpace(c.QueryValue("search"))
	if len(search) > 200 {
		search = search[:200]
	}

	items, paginationResult, err := h.announcementService.ListUserReadStatus(
		c.Request().Context(),
		announcementID,
		params,
		search,
	)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.PaginatedContext(gatewayJSONResponder{ctx: c}, items, paginationResult.Total, page, pageSize)
}
