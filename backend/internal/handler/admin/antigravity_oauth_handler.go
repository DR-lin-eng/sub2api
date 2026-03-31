package admin

import (
	"encoding/json"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AntigravityOAuthHandler struct {
	antigravityOAuthService *service.AntigravityOAuthService
}

func NewAntigravityOAuthHandler(antigravityOAuthService *service.AntigravityOAuthService) *AntigravityOAuthHandler {
	return &AntigravityOAuthHandler{antigravityOAuthService: antigravityOAuthService}
}

type AntigravityGenerateAuthURLRequest struct {
	ProxyID *int64 `json:"proxy_id"`
}

// GenerateAuthURL generates Google OAuth authorization URL
// POST /api/v1/admin/antigravity/oauth/auth-url
func (h *AntigravityOAuthHandler) GenerateAuthURL(c *gin.Context) {
	h.GenerateAuthURLGateway(gatewayctx.FromGin(c))
}

func (h *AntigravityOAuthHandler) GenerateAuthURLGateway(c gatewayctx.GatewayContext) {
	var req AntigravityGenerateAuthURLRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "请求无效: "+err.Error())
		return
	}

	result, err := h.antigravityOAuthService.GenerateAuthURL(c.Request().Context(), req.ProxyID)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "生成授权链接失败: "+err.Error())
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

type AntigravityExchangeCodeRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	State     string `json:"state" binding:"required"`
	Code      string `json:"code" binding:"required"`
	ProxyID   *int64 `json:"proxy_id"`
}

// ExchangeCode 用 authorization code 交换 token
// POST /api/v1/admin/antigravity/oauth/exchange-code
func (h *AntigravityOAuthHandler) ExchangeCode(c *gin.Context) {
	h.ExchangeCodeGateway(gatewayctx.FromGin(c))
}

func (h *AntigravityOAuthHandler) ExchangeCodeGateway(c gatewayctx.GatewayContext) {
	var req AntigravityExchangeCodeRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "请求无效: "+err.Error())
		return
	}

	tokenInfo, err := h.antigravityOAuthService.ExchangeCode(c.Request().Context(), &service.AntigravityExchangeCodeInput{
		SessionID: req.SessionID,
		State:     req.State,
		Code:      req.Code,
		ProxyID:   req.ProxyID,
	})
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Token 交换失败: "+err.Error())
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, tokenInfo)
}

// AntigravityRefreshTokenRequest represents the request for validating Antigravity refresh token
type AntigravityRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	ProxyID      *int64 `json:"proxy_id"`
}

// RefreshToken validates an Antigravity refresh token and returns full token info
// POST /api/v1/admin/antigravity/oauth/refresh-token
func (h *AntigravityOAuthHandler) RefreshToken(c *gin.Context) {
	h.RefreshTokenGateway(gatewayctx.FromGin(c))
}

func (h *AntigravityOAuthHandler) RefreshTokenGateway(c gatewayctx.GatewayContext) {
	var req AntigravityRefreshTokenRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "请求无效: "+err.Error())
		return
	}

	tokenInfo, err := h.antigravityOAuthService.ValidateRefreshToken(c.Request().Context(), req.RefreshToken, req.ProxyID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, tokenInfo)
}
