package admin

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RedeemHandler handles admin redeem code management
type RedeemHandler struct {
	adminService  service.AdminService
	redeemService *service.RedeemService
}

// NewRedeemHandler creates a new admin redeem handler
func NewRedeemHandler(adminService service.AdminService, redeemService *service.RedeemService) *RedeemHandler {
	return &RedeemHandler{
		adminService:  adminService,
		redeemService: redeemService,
	}
}

// GenerateRedeemCodesRequest represents generate redeem codes request
type GenerateRedeemCodesRequest struct {
	Count        int     `json:"count" binding:"required,min=1,max=100"`
	Type         string  `json:"type" binding:"required,oneof=balance concurrency subscription invitation"`
	Value        float64 `json:"value" binding:"min=0"`
	GroupID      *int64  `json:"group_id"`                                    // 订阅类型必填
	ValidityDays int     `json:"validity_days" binding:"omitempty,max=36500"` // 订阅类型使用，默认30天，最大100年
}

// CreateAndRedeemCodeRequest represents creating a fixed code and redeeming it for a target user.
// Type 为 omitempty 而非 required 是为了向后兼容旧版调用方（不传 type 时默认 balance）。
type CreateAndRedeemCodeRequest struct {
	Code         string  `json:"code" binding:"required,min=3,max=128"`
	Type         string  `json:"type" binding:"omitempty,oneof=balance concurrency subscription invitation"` // 不传时默认 balance（向后兼容）
	Value        float64 `json:"value" binding:"required,gt=0"`
	UserID       int64   `json:"user_id" binding:"required,gt=0"`
	GroupID      *int64  `json:"group_id"`                                    // subscription 类型必填
	ValidityDays int     `json:"validity_days" binding:"omitempty,max=36500"` // subscription 类型必填，>0
	Notes        string  `json:"notes"`
}

// List handles listing all redeem codes with pagination
// GET /api/v1/admin/redeem-codes
func (h *RedeemHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) ListGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	codeType := c.QueryValue("type")
	status := c.QueryValue("status")
	search := c.QueryValue("search")
	// 标准化和验证 search 参数
	search = strings.TrimSpace(search)
	if len(search) > 100 {
		search = search[:100]
	}

	codes, total, err := h.adminService.ListRedeemCodes(c.Request().Context(), page, pageSize, codeType, status, search)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	out := make([]dto.AdminRedeemCode, 0, len(codes))
	for i := range codes {
		out = append(out, *dto.RedeemCodeFromServiceAdmin(&codes[i]))
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, out, total, page, pageSize)
}

// GetByID handles getting a redeem code by ID
// GET /api/v1/admin/redeem-codes/:id
func (h *RedeemHandler) GetByID(c *gin.Context) {
	h.GetByIDGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) GetByIDGateway(c gatewayctx.GatewayContext) {
	codeID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid redeem code ID")
		return
	}

	code, err := h.adminService.GetRedeemCode(c.Request().Context(), codeID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.RedeemCodeFromServiceAdmin(code))
}

// Generate handles generating new redeem codes
// POST /api/v1/admin/redeem-codes/generate
func (h *RedeemHandler) Generate(c *gin.Context) {
	h.GenerateGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) GenerateGateway(c gatewayctx.GatewayContext) {
	var req GenerateRedeemCodesRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	executeAdminIdempotentGatewayJSON(c, "admin.redeem_codes.generate", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		codes, execErr := h.adminService.GenerateRedeemCodes(ctx, &service.GenerateRedeemCodesInput{
			Count:        req.Count,
			Type:         req.Type,
			Value:        req.Value,
			GroupID:      req.GroupID,
			ValidityDays: req.ValidityDays,
		})
		if execErr != nil {
			return nil, execErr
		}

		out := make([]dto.AdminRedeemCode, 0, len(codes))
		for i := range codes {
			out = append(out, *dto.RedeemCodeFromServiceAdmin(&codes[i]))
		}
		return out, nil
	})
}

// CreateAndRedeem creates a fixed redeem code and redeems it for a target user in one step.
// POST /api/v1/admin/redeem-codes/create-and-redeem
func (h *RedeemHandler) CreateAndRedeem(c *gin.Context) {
	h.CreateAndRedeemGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) CreateAndRedeemGateway(c gatewayctx.GatewayContext) {
	if h.redeemService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "redeem service not configured")
		return
	}

	var req CreateAndRedeemCodeRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	// 向后兼容：旧版调用方（如 Sub2ApiPay）不传 type 字段，默认当作 balance 充值处理。
	// 请勿删除此默认值逻辑，否则会导致旧版调用方 400 报错。
	if req.Type == "" {
		req.Type = "balance"
	}

	if req.Type == "subscription" {
		if req.GroupID == nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "group_id is required for subscription type")
			return
		}
		if req.ValidityDays <= 0 {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "validity_days must be greater than 0 for subscription type")
			return
		}
	}

	executeAdminIdempotentGatewayJSON(c, "admin.redeem_codes.create_and_redeem", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		existing, err := h.redeemService.GetByCode(ctx, req.Code)
		if err == nil {
			return h.resolveCreateAndRedeemExisting(ctx, existing, req.UserID)
		}
		if !errors.Is(err, service.ErrRedeemCodeNotFound) {
			return nil, err
		}

		createErr := h.redeemService.CreateCode(ctx, &service.RedeemCode{
			Code:         req.Code,
			Type:         req.Type,
			Value:        req.Value,
			Status:       service.StatusUnused,
			Notes:        req.Notes,
			GroupID:      req.GroupID,
			ValidityDays: req.ValidityDays,
		})
		if createErr != nil {
			// Unique code race: if code now exists, use idempotent semantics by used_by.
			existingAfterCreateErr, getErr := h.redeemService.GetByCode(ctx, req.Code)
			if getErr == nil {
				return h.resolveCreateAndRedeemExisting(ctx, existingAfterCreateErr, req.UserID)
			}
			return nil, createErr
		}

		redeemed, redeemErr := h.redeemService.Redeem(ctx, req.UserID, req.Code)
		if redeemErr != nil {
			return nil, redeemErr
		}
		return map[string]any{"redeem_code": dto.RedeemCodeFromServiceAdmin(redeemed)}, nil
	})
}

func (h *RedeemHandler) resolveCreateAndRedeemExisting(ctx context.Context, existing *service.RedeemCode, userID int64) (any, error) {
	if existing == nil {
		return nil, infraerrors.Conflict("REDEEM_CODE_CONFLICT", "redeem code conflict")
	}

	// If previous run created the code but crashed before redeem, redeem it now.
	if existing.CanUse() {
		redeemed, err := h.redeemService.Redeem(ctx, userID, existing.Code)
		if err == nil {
			return map[string]any{"redeem_code": dto.RedeemCodeFromServiceAdmin(redeemed)}, nil
		}
		if !errors.Is(err, service.ErrRedeemCodeUsed) {
			return nil, err
		}
		latest, getErr := h.redeemService.GetByCode(ctx, existing.Code)
		if getErr == nil {
			existing = latest
		}
	}

	if existing.UsedBy != nil && *existing.UsedBy == userID {
		return map[string]any{"redeem_code": dto.RedeemCodeFromServiceAdmin(existing)}, nil
	}

	return nil, infraerrors.Conflict("REDEEM_CODE_CONFLICT", "redeem code already used by another user")
}

// Delete handles deleting a redeem code
// DELETE /api/v1/admin/redeem-codes/:id
func (h *RedeemHandler) Delete(c *gin.Context) {
	h.DeleteGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) DeleteGateway(c gatewayctx.GatewayContext) {
	codeID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid redeem code ID")
		return
	}

	err = h.adminService.DeleteRedeemCode(c.Request().Context(), codeID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "Redeem code deleted successfully"})
}

// BatchDelete handles batch deleting redeem codes
// POST /api/v1/admin/redeem-codes/batch-delete
func (h *RedeemHandler) BatchDelete(c *gin.Context) {
	h.BatchDeleteGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) BatchDeleteGateway(c gatewayctx.GatewayContext) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required,min=1"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	deleted, err := h.adminService.BatchDeleteRedeemCodes(c.Request().Context(), req.IDs)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"deleted": deleted,
		"message": "Redeem codes deleted successfully",
	})
}

// Expire handles expiring a redeem code
// POST /api/v1/admin/redeem-codes/:id/expire
func (h *RedeemHandler) Expire(c *gin.Context) {
	h.ExpireGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) ExpireGateway(c gatewayctx.GatewayContext) {
	codeID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid redeem code ID")
		return
	}

	code, err := h.adminService.ExpireRedeemCode(c.Request().Context(), codeID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.RedeemCodeFromServiceAdmin(code))
}

// GetStats handles getting redeem code statistics
// GET /api/v1/admin/redeem-codes/stats
func (h *RedeemHandler) GetStats(c *gin.Context) {
	h.GetStatsGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) GetStatsGateway(c gatewayctx.GatewayContext) {
	const pageSize = 1000
	page := 1
	totalCodes := int64(0)
	activeCodes := int64(0)
	usedCodes := int64(0)
	expiredCodes := int64(0)
	totalValueDistributed := 0.0
	byType := map[string]int64{
		service.RedeemTypeBalance:      0,
		service.RedeemTypeConcurrency:  0,
		service.RedeemTypeSubscription: 0,
		service.RedeemTypeInvitation:   0,
	}

	for {
		codes, total, err := h.adminService.ListRedeemCodes(c.Request().Context(), page, pageSize, "", "", "")
		if err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
		if page == 1 {
			totalCodes = total
		}
		for _, code := range codes {
			byType[code.Type]++
			switch code.Status {
			case service.StatusUnused:
				activeCodes++
			case service.StatusUsed:
				usedCodes++
				totalValueDistributed += code.Value
			case service.StatusExpired:
				expiredCodes++
			}
		}
		if len(codes) < pageSize || int64(page*pageSize) >= total {
			break
		}
		page++
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"total_codes":             totalCodes,
		"active_codes":            activeCodes,
		"used_codes":              usedCodes,
		"expired_codes":           expiredCodes,
		"total_value_distributed": totalValueDistributed,
		"by_type":                 byType,
	})
}

// Export handles exporting redeem codes to CSV
// GET /api/v1/admin/redeem-codes/export
func (h *RedeemHandler) Export(c *gin.Context) {
	h.ExportGateway(gatewayctx.FromGin(c))
}

func (h *RedeemHandler) ExportGateway(c gatewayctx.GatewayContext) {
	codeType := c.QueryValue("type")
	status := c.QueryValue("status")

	// Get all codes without pagination (use large page size)
	codes, _, err := h.adminService.ListRedeemCodes(c.Request().Context(), 1, 10000, codeType, status, "")
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	// Create CSV buffer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	if err := writer.Write([]string{"id", "code", "type", "value", "status", "used_by", "used_by_email", "used_at", "created_at"}); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to export redeem codes: "+err.Error())
		return
	}

	// Write data rows
	for _, code := range codes {
		usedBy := ""
		if code.UsedBy != nil {
			usedBy = fmt.Sprintf("%d", *code.UsedBy)
		}
		usedByEmail := ""
		if code.User != nil {
			usedByEmail = code.User.Email
		}
		usedAt := ""
		if code.UsedAt != nil {
			usedAt = code.UsedAt.Format("2006-01-02 15:04:05")
		}
		if err := writer.Write([]string{
			fmt.Sprintf("%d", code.ID),
			code.Code,
			code.Type,
			fmt.Sprintf("%.2f", code.Value),
			code.Status,
			usedBy,
			usedByEmail,
			usedAt,
			code.CreatedAt.Format("2006-01-02 15:04:05"),
		}); err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to export redeem codes: "+err.Error())
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to export redeem codes: "+err.Error())
		return
	}

	c.SetHeader("Content-Disposition", "attachment; filename=redeem_codes.csv")
	if err := c.WriteReader(http.StatusOK, "text/csv", bytes.NewReader(buf.Bytes()), int64(buf.Len())); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to export redeem codes: "+err.Error())
	}
}
