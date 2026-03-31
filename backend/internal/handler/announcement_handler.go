package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler handles user announcement operations
type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
}

type announcementGatewayResponder struct {
	ctx gatewayctx.GatewayContext
}

func (g announcementGatewayResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g announcementGatewayResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}

// NewAnnouncementHandler creates a new user announcement handler
func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
	}
}

// List handles listing announcements visible to current user
// GET /api/v1/announcements
func (h *AnnouncementHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *AnnouncementHandler) ListGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(announcementGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not found in context")
		return
	}

	unreadOnly := parseBoolQuery(c.QueryValue("unread_only"))

	items, err := h.announcementService.ListForUser(c.Request().Context(), subject.UserID, unreadOnly)
	if err != nil {
		response.ErrorFromContext(announcementGatewayResponder{ctx: c}, err)
		return
	}

	out := make([]dto.UserAnnouncement, 0, len(items))
	for i := range items {
		out = append(out, *dto.UserAnnouncementFromService(&items[i]))
	}
	response.SuccessContext(announcementGatewayResponder{ctx: c}, out)
}

// MarkRead marks an announcement as read for current user
// POST /api/v1/announcements/:id/read
func (h *AnnouncementHandler) MarkRead(c *gin.Context) {
	h.MarkReadGateway(gatewayctx.FromGin(c))
}

func (h *AnnouncementHandler) MarkReadGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(announcementGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not found in context")
		return
	}

	announcementID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.ErrorContext(announcementGatewayResponder{ctx: c}, http.StatusBadRequest, "Invalid announcement ID")
		return
	}

	if err := h.announcementService.MarkRead(c.Request().Context(), subject.UserID, announcementID); err != nil {
		response.ErrorFromContext(announcementGatewayResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(announcementGatewayResponder{ctx: c}, gin.H{"message": "ok"})
}

func parseBoolQuery(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
