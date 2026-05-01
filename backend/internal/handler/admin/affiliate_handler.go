package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type AffiliateHandler struct {
	affiliateService *service.AffiliateService
	adminService     service.AdminService
}

func NewAffiliateHandler(affiliateService *service.AffiliateService, adminService service.AdminService) *AffiliateHandler {
	return &AffiliateHandler{
		affiliateService: affiliateService,
		adminService:     adminService,
	}
}

func (h *AffiliateHandler) ListUsers(c *gin.Context) {
	h.ListUsersGateway(gatewayctx.FromGin(c))
}

func (h *AffiliateHandler) ListUsersGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	entries, total, err := h.affiliateService.AdminListCustomUsers(c.Request().Context(), service.AffiliateAdminFilter{
		Search:   c.QueryValue("search"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, entries, total, page, pageSize)
}

type UpdateAffiliateUserRequest struct {
	AffCode              *string  `json:"aff_code"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent"`
	ClearRebateRate      bool     `json:"clear_rebate_rate"`
}

func (h *AffiliateHandler) UpdateUserSettings(c *gin.Context) {
	h.UpdateUserSettingsGateway(gatewayctx.FromGin(c))
}

func (h *AffiliateHandler) UpdateUserSettingsGateway(c gatewayctx.GatewayContext) {
	userID, err := strconv.ParseInt(c.PathParam("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid user_id")
		return
	}

	var req UpdateAffiliateUserRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.AffCode != nil {
		if err := h.affiliateService.AdminUpdateUserAffCode(c.Request().Context(), userID, *req.AffCode); err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
	}
	if req.ClearRebateRate {
		if err := h.affiliateService.AdminSetUserRebateRate(c.Request().Context(), userID, nil); err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
	} else if req.AffRebateRatePercent != nil {
		if err := h.affiliateService.AdminSetUserRebateRate(c.Request().Context(), userID, req.AffRebateRatePercent); err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"user_id": userID})
}

func (h *AffiliateHandler) ClearUserSettings(c *gin.Context) {
	h.ClearUserSettingsGateway(gatewayctx.FromGin(c))
}

func (h *AffiliateHandler) ClearUserSettingsGateway(c gatewayctx.GatewayContext) {
	userID, err := strconv.ParseInt(c.PathParam("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid user_id")
		return
	}
	if err := h.affiliateService.AdminSetUserRebateRate(c.Request().Context(), userID, nil); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if _, err := h.affiliateService.AdminResetUserAffCode(c.Request().Context(), userID); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"user_id": userID})
}

type BatchSetRateRequest struct {
	UserIDs              []int64  `json:"user_ids" binding:"required"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent"`
	Clear                bool     `json:"clear"`
}

func (h *AffiliateHandler) BatchSetRate(c *gin.Context) {
	h.BatchSetRateGateway(gatewayctx.FromGin(c))
}

func (h *AffiliateHandler) BatchSetRateGateway(c gatewayctx.GatewayContext) {
	var req BatchSetRateRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if len(req.UserIDs) == 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "user_ids cannot be empty")
		return
	}
	if !req.Clear && req.AffRebateRatePercent == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "aff_rebate_rate_percent is required unless clear=true")
		return
	}
	rate := req.AffRebateRatePercent
	if req.Clear {
		rate = nil
	}
	if err := h.affiliateService.AdminBatchSetUserRebateRate(c.Request().Context(), req.UserIDs, rate); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"affected": len(req.UserIDs)})
}

type AffiliateUserSummary struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (h *AffiliateHandler) LookupUsers(c *gin.Context) {
	h.LookupUsersGateway(gatewayctx.FromGin(c))
}

func (h *AffiliateHandler) LookupUsersGateway(c gatewayctx.GatewayContext) {
	keyword := c.QueryValue("q")
	if keyword == "" {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, []AffiliateUserSummary{})
		return
	}
	users, _, err := h.adminService.ListUsers(c.Request().Context(), 1, 20, service.UserListFilters{Search: keyword})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	result := make([]AffiliateUserSummary, len(users))
	for i, u := range users {
		result[i] = AffiliateUserSummary{ID: u.ID, Email: u.Email, Username: u.Username}
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}
