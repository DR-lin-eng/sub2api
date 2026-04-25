package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// TLSFingerprintProfileHandler 处理 TLS 指纹模板的 HTTP 请求
type TLSFingerprintProfileHandler struct {
	service *service.TLSFingerprintProfileService
}

// NewTLSFingerprintProfileHandler 创建 TLS 指纹模板处理器
func NewTLSFingerprintProfileHandler(service *service.TLSFingerprintProfileService) *TLSFingerprintProfileHandler {
	return &TLSFingerprintProfileHandler{service: service}
}

func parseTLSFingerprintProfileIDGateway(c gatewayctx.GatewayContext) (int64, bool) {
	id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid profile ID")
		return 0, false
	}
	return id, true
}

// CreateTLSFingerprintProfilePayload 创建模板请求
type CreateTLSFingerprintProfilePayload struct {
	Name                string   `json:"name" binding:"required"`
	Description         *string  `json:"description"`
	EnableGREASE        *bool    `json:"enable_grease"`
	CipherSuites        []uint16 `json:"cipher_suites"`
	Curves              []uint16 `json:"curves"`
	PointFormats        []uint16 `json:"point_formats"`
	SignatureAlgorithms []uint16 `json:"signature_algorithms"`
	ALPNProtocols       []string `json:"alpn_protocols"`
	SupportedVersions   []uint16 `json:"supported_versions"`
	KeyShareGroups      []uint16 `json:"key_share_groups"`
	PSKModes            []uint16 `json:"psk_modes"`
	Extensions          []uint16 `json:"extensions"`
}

// UpdateTLSFingerprintProfilePayload 更新模板请求（部分更新）
type UpdateTLSFingerprintProfilePayload struct {
	Name                *string  `json:"name"`
	Description         *string  `json:"description"`
	EnableGREASE        *bool    `json:"enable_grease"`
	CipherSuites        []uint16 `json:"cipher_suites"`
	Curves              []uint16 `json:"curves"`
	PointFormats        []uint16 `json:"point_formats"`
	SignatureAlgorithms []uint16 `json:"signature_algorithms"`
	ALPNProtocols       []string `json:"alpn_protocols"`
	SupportedVersions   []uint16 `json:"supported_versions"`
	KeyShareGroups      []uint16 `json:"key_share_groups"`
	PSKModes            []uint16 `json:"psk_modes"`
	Extensions          []uint16 `json:"extensions"`
}

// List 获取所有模板
// GET /api/v1/admin/tls-fingerprint-profiles
func (h *TLSFingerprintProfileHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *TLSFingerprintProfileHandler) ListGateway(c gatewayctx.GatewayContext) {
	profiles, err := h.service.List(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, profiles)
}

// GetByID 根据 ID 获取模板
// GET /api/v1/admin/tls-fingerprint-profiles/:id
func (h *TLSFingerprintProfileHandler) GetByID(c *gin.Context) {
	h.GetByIDGateway(gatewayctx.FromGin(c))
}

func (h *TLSFingerprintProfileHandler) GetByIDGateway(c gatewayctx.GatewayContext) {
	id, ok := parseTLSFingerprintProfileIDGateway(c)
	if !ok {
		return
	}

	profile, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if profile == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusNotFound, "Profile not found")
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, profile)
}

// Create 创建模板
// POST /api/v1/admin/tls-fingerprint-profiles
func (h *TLSFingerprintProfileHandler) Create(c *gin.Context) {
	h.CreateGateway(gatewayctx.FromGin(c))
}

func (h *TLSFingerprintProfileHandler) CreateGateway(c gatewayctx.GatewayContext) {
	var req CreateTLSFingerprintProfilePayload
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	profile := &model.TLSFingerprintProfile{
		Name:                req.Name,
		Description:         req.Description,
		CipherSuites:        req.CipherSuites,
		Curves:              req.Curves,
		PointFormats:        req.PointFormats,
		SignatureAlgorithms: req.SignatureAlgorithms,
		ALPNProtocols:       req.ALPNProtocols,
		SupportedVersions:   req.SupportedVersions,
		KeyShareGroups:      req.KeyShareGroups,
		PSKModes:            req.PSKModes,
		Extensions:          req.Extensions,
	}
	if req.EnableGREASE != nil {
		profile.EnableGREASE = *req.EnableGREASE
	}

	created, err := h.service.Create(c.Request().Context(), profile)
	if err != nil {
		if _, ok := err.(*model.ValidationError); ok {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, created)
}

// Update 更新模板（支持部分更新）
// PUT /api/v1/admin/tls-fingerprint-profiles/:id
func (h *TLSFingerprintProfileHandler) Update(c *gin.Context) {
	h.UpdateGateway(gatewayctx.FromGin(c))
}

func (h *TLSFingerprintProfileHandler) UpdateGateway(c gatewayctx.GatewayContext) {
	id, ok := parseTLSFingerprintProfileIDGateway(c)
	if !ok {
		return
	}

	var req UpdateTLSFingerprintProfilePayload
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	existing, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if existing == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusNotFound, "Profile not found")
		return
	}

	profile := &model.TLSFingerprintProfile{
		ID:                  id,
		Name:                existing.Name,
		Description:         existing.Description,
		EnableGREASE:        existing.EnableGREASE,
		CipherSuites:        existing.CipherSuites,
		Curves:              existing.Curves,
		PointFormats:        existing.PointFormats,
		SignatureAlgorithms: existing.SignatureAlgorithms,
		ALPNProtocols:       existing.ALPNProtocols,
		SupportedVersions:   existing.SupportedVersions,
		KeyShareGroups:      existing.KeyShareGroups,
		PSKModes:            existing.PSKModes,
		Extensions:          existing.Extensions,
	}

	if req.Name != nil {
		profile.Name = *req.Name
	}
	if req.Description != nil {
		profile.Description = req.Description
	}
	if req.EnableGREASE != nil {
		profile.EnableGREASE = *req.EnableGREASE
	}
	if req.CipherSuites != nil {
		profile.CipherSuites = req.CipherSuites
	}
	if req.Curves != nil {
		profile.Curves = req.Curves
	}
	if req.PointFormats != nil {
		profile.PointFormats = req.PointFormats
	}
	if req.SignatureAlgorithms != nil {
		profile.SignatureAlgorithms = req.SignatureAlgorithms
	}
	if req.ALPNProtocols != nil {
		profile.ALPNProtocols = req.ALPNProtocols
	}
	if req.SupportedVersions != nil {
		profile.SupportedVersions = req.SupportedVersions
	}
	if req.KeyShareGroups != nil {
		profile.KeyShareGroups = req.KeyShareGroups
	}
	if req.PSKModes != nil {
		profile.PSKModes = req.PSKModes
	}
	if req.Extensions != nil {
		profile.Extensions = req.Extensions
	}

	updated, err := h.service.Update(c.Request().Context(), profile)
	if err != nil {
		if _, ok := err.(*model.ValidationError); ok {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
			return
		}
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}

// Delete 删除模板
// DELETE /api/v1/admin/tls-fingerprint-profiles/:id
func (h *TLSFingerprintProfileHandler) Delete(c *gin.Context) {
	h.DeleteGateway(gatewayctx.FromGin(c))
}

func (h *TLSFingerprintProfileHandler) DeleteGateway(c gatewayctx.GatewayContext) {
	id, ok := parseTLSFingerprintProfileIDGateway(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, gin.H{"message": "Profile deleted successfully"})
}
