package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	cfg           *config.Config
	authService   *service.AuthService
	userService   *service.UserService
	settingSvc    *service.SettingService
	promoService  *service.PromoService
	redeemService *service.RedeemService
	totpService   *service.TotpService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(cfg *config.Config, authService *service.AuthService, userService *service.UserService, settingService *service.SettingService, promoService *service.PromoService, redeemService *service.RedeemService, totpService *service.TotpService) *AuthHandler {
	return &AuthHandler{
		cfg:           cfg,
		authService:   authService,
		userService:   userService,
		settingSvc:    settingService,
		promoService:  promoService,
		redeemService: redeemService,
		totpService:   totpService,
	}
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	VerifyCode     string `json:"verify_code"`
	TurnstileToken string `json:"turnstile_token"`
	PromoCode      string `json:"promo_code"`      // 注册优惠码
	InvitationCode string `json:"invitation_code"` // 邀请码
}

// SendVerifyCodeRequest 发送验证码请求
type SendVerifyCodeRequest struct {
	Email          string `json:"email" binding:"required,email"`
	TurnstileToken string `json:"turnstile_token"`
}

// SendVerifyCodeResponse 发送验证码响应
type SendVerifyCodeResponse struct {
	Message   string `json:"message"`
	Countdown int    `json:"countdown"` // 倒计时秒数
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	TurnstileToken string `json:"turnstile_token"`
}

// AuthResponse 认证响应格式（匹配前端期望）
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"` // 新增：Refresh Token
	ExpiresIn    int       `json:"expires_in,omitempty"`    // 新增：Access Token有效期（秒）
	TokenType    string    `json:"token_type"`
	User         *dto.User `json:"user"`
}

type authJSONResponder struct {
	ctx gatewayctx.GatewayContext
}

func (g authJSONResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g authJSONResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}

func (h *AuthHandler) respondInvalidCredentials(c gatewayctx.GatewayContext) {
	response.ErrorFromContext(authJSONResponder{ctx: c}, service.ErrInvalidCredentials)
}

// respondWithTokenPair 生成 Token 对并返回认证响应
// 如果 Token 对生成失败，回退到只返回 Access Token（向后兼容）
func (h *AuthHandler) respondWithTokenPair(c *gin.Context, user *service.User) {
	h.respondWithTokenPairGateway(gatewayctx.FromGin(c), user)
}

func (h *AuthHandler) respondWithTokenPairGateway(c gatewayctx.GatewayContext, user *service.User) {
	tokenPair, err := h.authService.GenerateTokenPair(c.Request().Context(), user, "")
	if err != nil {
		slog.Error("failed to generate token pair", "error", err, "user_id", user.ID)
		// 回退到只返回Access Token
		token, tokenErr := h.authService.GenerateToken(user)
		if tokenErr != nil {
			response.ErrorContext(authJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to generate token")
			return
		}
		response.SuccessContext(authJSONResponder{ctx: c}, AuthResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			User:        dto.UserFromService(user),
		})
		return
	}
	response.SuccessContext(authJSONResponder{ctx: c}, AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		TokenType:    "Bearer",
		User:         dto.UserFromService(user),
	})
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	h.RegisterGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) RegisterGateway(c gatewayctx.GatewayContext) {
	var req RegisterRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Turnstile 验证（邮箱验证码注册场景避免重复校验一次性 token）
	if err := h.authService.VerifyTurnstileForRegister(c.Request().Context(), req.TurnstileToken, ip.GetClientIPContext(c), req.VerifyCode); err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	_, user, err := h.authService.RegisterWithVerification(c.Request().Context(), req.Email, req.Password, req.VerifyCode, req.PromoCode, req.InvitationCode)
	if err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	h.respondWithTokenPairGateway(c, user)
}

// SendVerifyCode 发送邮箱验证码
// POST /api/v1/auth/send-verify-code
func (h *AuthHandler) SendVerifyCode(c *gin.Context) {
	h.SendVerifyCodeGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) SendVerifyCodeGateway(c gatewayctx.GatewayContext) {
	var req SendVerifyCodeRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Turnstile 验证
	if err := h.authService.VerifyTurnstile(c.Request().Context(), req.TurnstileToken, ip.GetClientIPContext(c)); err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	result, err := h.authService.SendVerifyCodeAsync(c.Request().Context(), req.Email)
	if err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(authJSONResponder{ctx: c}, SendVerifyCodeResponse{
		Message:   "Verification code sent successfully",
		Countdown: result.Countdown,
	})
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	h.LoginGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) LoginGateway(c gatewayctx.GatewayContext) {
	var req LoginRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Turnstile 验证
	if err := h.authService.VerifyTurnstile(c.Request().Context(), req.TurnstileToken, ip.GetClientIPContext(c)); err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	token, user, err := h.authService.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}
	_ = token // token 由 authService.Login 返回但此处由 respondWithTokenPair 重新生成

	if h.settingSvc.IsBackendModeEnabled(c.Request().Context()) && !user.IsAdmin() {
		h.respondInvalidCredentials(c)
		return
	}

	// Check if TOTP 2FA is enabled for this user
	if h.totpService != nil && h.settingSvc.IsTotpEnabled(c.Request().Context()) && user.TotpEnabled {
		// Create a temporary login session for 2FA
		tempToken, err := h.totpService.CreateLoginSession(c.Request().Context(), user.ID, user.Email)
		if err != nil {
			response.ErrorContext(authJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to create 2FA session")
			return
		}

		response.SuccessContext(authJSONResponder{ctx: c}, TotpLoginResponse{
			Requires2FA:     true,
			TempToken:       tempToken,
			UserEmailMasked: service.MaskEmail(user.Email),
		})
		return
	}

	h.respondWithTokenPairGateway(c, user)
}

// TotpLoginResponse represents the response when 2FA is required
type TotpLoginResponse struct {
	Requires2FA     bool   `json:"requires_2fa"`
	TempToken       string `json:"temp_token,omitempty"`
	UserEmailMasked string `json:"user_email_masked,omitempty"`
}

// Login2FARequest represents the 2FA login request
type Login2FARequest struct {
	TempToken string `json:"temp_token" binding:"required"`
	TotpCode  string `json:"totp_code" binding:"required,len=6"`
}

// Login2FA completes the login with 2FA verification
// POST /api/v1/auth/login/2fa
func (h *AuthHandler) Login2FA(c *gin.Context) {
	h.Login2FAGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) Login2FAGateway(c gatewayctx.GatewayContext) {
	var req Login2FARequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	slog.Debug("login_2fa_request",
		"temp_token_len", len(req.TempToken),
		"totp_code_len", len(req.TotpCode))

	// Get the login session
	session, err := h.totpService.GetLoginSession(c.Request().Context(), req.TempToken)
	if err != nil || session == nil {
		tokenPrefix := ""
		if len(req.TempToken) >= 8 {
			tokenPrefix = req.TempToken[:8]
		}
		slog.Debug("login_2fa_session_invalid",
			"temp_token_prefix", tokenPrefix,
			"error", err)
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid or expired 2FA session")
		return
	}

	slog.Debug("login_2fa_session_found",
		"user_id", session.UserID,
		"email", session.Email)

	// Get the user before verification so backend mode can fail closed without
	// preserving a pending 2FA session for non-admin users.
	user, err := h.userService.GetByID(c.Request().Context(), session.UserID)
	if err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	// Backend mode: non-admin 2FA sessions must be invalidated and respond the
	// same as invalid credentials.
	if h.settingSvc.IsBackendModeEnabled(c.Request().Context()) && !user.IsAdmin() {
		if err := h.totpService.DeleteLoginSession(c.Request().Context(), req.TempToken); err != nil {
			slog.Debug("login_2fa_delete_session_failed",
				"user_id", session.UserID,
				"error", err)
		}
		h.respondInvalidCredentials(c)
		return
	}

	// Verify the TOTP code
	if err := h.totpService.VerifyCode(c.Request().Context(), session.UserID, req.TotpCode); err != nil {
		slog.Debug("login_2fa_verify_failed",
			"user_id", session.UserID,
			"error", err)
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	// Delete the login session after all checks pass.
	_ = h.totpService.DeleteLoginSession(c.Request().Context(), req.TempToken)

	h.respondWithTokenPairGateway(c, user)
}

// GetCurrentUser handles getting current authenticated user
// GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	h.GetCurrentUserGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) GetCurrentUserGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(authGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	user, err := h.userService.GetByID(c.Request().Context(), subject.UserID)
	if err != nil {
		response.ErrorFromContext(authGatewayResponder{ctx: c}, err)
		return
	}

	type UserResponse struct {
		*dto.User
		RunMode string `json:"run_mode"`
	}

	runMode := config.RunModeStandard
	if h.cfg != nil {
		runMode = h.cfg.RunMode
	}

	response.SuccessContext(authGatewayResponder{ctx: c}, UserResponse{User: dto.UserFromService(user), RunMode: runMode})
}

// ValidatePromoCodeRequest 验证优惠码请求
type ValidatePromoCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// ValidatePromoCodeResponse 验证优惠码响应
type ValidatePromoCodeResponse struct {
	Valid       bool    `json:"valid"`
	BonusAmount float64 `json:"bonus_amount,omitempty"`
	ErrorCode   string  `json:"error_code,omitempty"`
	Message     string  `json:"message,omitempty"`
}

// ValidatePromoCode 验证优惠码（公开接口，注册前调用）
// POST /api/v1/auth/validate-promo-code
func (h *AuthHandler) ValidatePromoCode(c *gin.Context) {
	h.ValidatePromoCodeGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) ValidatePromoCodeGateway(c gatewayctx.GatewayContext) {
	// 检查优惠码功能是否启用
	if h.settingSvc != nil && !h.settingSvc.IsPromoCodeEnabled(c.Request().Context()) {
		response.SuccessContext(authJSONResponder{ctx: c}, ValidatePromoCodeResponse{
			Valid:     false,
			ErrorCode: "PROMO_CODE_DISABLED",
		})
		return
	}

	var req ValidatePromoCodeRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	promoCode, err := h.promoService.ValidatePromoCode(c.Request().Context(), req.Code)
	if err != nil {
		// 根据错误类型返回对应的错误码
		errorCode := "PROMO_CODE_INVALID"
		switch err {
		case service.ErrPromoCodeNotFound:
			errorCode = "PROMO_CODE_NOT_FOUND"
		case service.ErrPromoCodeExpired:
			errorCode = "PROMO_CODE_EXPIRED"
		case service.ErrPromoCodeDisabled:
			errorCode = "PROMO_CODE_DISABLED"
		case service.ErrPromoCodeMaxUsed:
			errorCode = "PROMO_CODE_MAX_USED"
		case service.ErrPromoCodeAlreadyUsed:
			errorCode = "PROMO_CODE_ALREADY_USED"
		}

		response.SuccessContext(authJSONResponder{ctx: c}, ValidatePromoCodeResponse{
			Valid:     false,
			ErrorCode: errorCode,
		})
		return
	}

	if promoCode == nil {
		response.SuccessContext(authJSONResponder{ctx: c}, ValidatePromoCodeResponse{
			Valid:     false,
			ErrorCode: "PROMO_CODE_INVALID",
		})
		return
	}

	response.SuccessContext(authJSONResponder{ctx: c}, ValidatePromoCodeResponse{
		Valid:       true,
		BonusAmount: promoCode.BonusAmount,
	})
}

// ValidateInvitationCodeRequest 验证邀请码请求
type ValidateInvitationCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// ValidateInvitationCodeResponse 验证邀请码响应
type ValidateInvitationCodeResponse struct {
	Valid     bool   `json:"valid"`
	ErrorCode string `json:"error_code,omitempty"`
}

// ValidateInvitationCode 验证邀请码（公开接口，注册前调用）
// POST /api/v1/auth/validate-invitation-code
func (h *AuthHandler) ValidateInvitationCode(c *gin.Context) {
	h.ValidateInvitationCodeGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) ValidateInvitationCodeGateway(c gatewayctx.GatewayContext) {
	// 检查邀请码功能是否启用
	if h.settingSvc == nil || !h.settingSvc.IsInvitationCodeEnabled(c.Request().Context()) {
		response.SuccessContext(authJSONResponder{ctx: c}, ValidateInvitationCodeResponse{
			Valid:     false,
			ErrorCode: "INVITATION_CODE_DISABLED",
		})
		return
	}

	var req ValidateInvitationCodeRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// 验证邀请码
	redeemCode, err := h.redeemService.GetByCode(c.Request().Context(), req.Code)
	if err != nil {
		response.SuccessContext(authJSONResponder{ctx: c}, ValidateInvitationCodeResponse{
			Valid:     false,
			ErrorCode: "INVITATION_CODE_NOT_FOUND",
		})
		return
	}

	// 检查类型和状态
	if redeemCode.Type != service.RedeemTypeInvitation {
		response.SuccessContext(authJSONResponder{ctx: c}, ValidateInvitationCodeResponse{
			Valid:     false,
			ErrorCode: "INVITATION_CODE_INVALID",
		})
		return
	}

	if redeemCode.Status != service.StatusUnused {
		response.SuccessContext(authJSONResponder{ctx: c}, ValidateInvitationCodeResponse{
			Valid:     false,
			ErrorCode: "INVITATION_CODE_USED",
		})
		return
	}

	response.SuccessContext(authJSONResponder{ctx: c}, ValidateInvitationCodeResponse{
		Valid: true,
	})
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
	Email          string `json:"email" binding:"required,email"`
	TurnstileToken string `json:"turnstile_token"`
}

// ForgotPasswordResponse 忘记密码响应
type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

// ForgotPassword 请求密码重置
// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	h.ForgotPasswordGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) ForgotPasswordGateway(c gatewayctx.GatewayContext) {
	var req ForgotPasswordRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Turnstile 验证
	if err := h.authService.VerifyTurnstile(c.Request().Context(), req.TurnstileToken, ip.GetClientIPContext(c)); err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	frontendBaseURL := strings.TrimSpace(h.settingSvc.GetFrontendURL(c.Request().Context()))
	if frontendBaseURL == "" {
		slog.Error("frontend_url not configured in settings or config; cannot build password reset link")
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusInternalServerError, "Password reset is not configured")
		return
	}

	// Request password reset (async)
	// Note: This returns success even if email doesn't exist (to prevent enumeration)
	if err := h.authService.RequestPasswordResetAsync(c.Request().Context(), req.Email, frontendBaseURL); err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(authJSONResponder{ctx: c}, ForgotPasswordResponse{
		Message: "If your email is registered, you will receive a password reset link shortly.",
	})
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPasswordResponse 重置密码响应
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// ResetPassword 重置密码
// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	h.ResetPasswordGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) ResetPasswordGateway(c gatewayctx.GatewayContext) {
	var req ResetPasswordRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Reset password
	if err := h.authService.ResetPassword(c.Request().Context(), req.Email, req.Token, req.NewPassword); err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(authJSONResponder{ctx: c}, ResetPasswordResponse{
		Message: "Your password has been reset successfully. You can now log in with your new password.",
	})
}

// ==================== Token Refresh Endpoints ====================

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // Access Token有效期（秒）
	TokenType    string `json:"token_type"`
}

// RefreshToken 刷新Token
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	h.RefreshTokenGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) RefreshTokenGateway(c gatewayctx.GatewayContext) {
	var req RefreshTokenRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	result, err := h.authService.RefreshTokenPair(c.Request().Context(), req.RefreshToken)
	if err != nil {
		response.ErrorFromContext(authJSONResponder{ctx: c}, err)
		return
	}

	// Backend mode: block non-admin token refresh
	if h.settingSvc.IsBackendModeEnabled(c.Request().Context()) && result.UserRole != "admin" {
		response.ErrorContext(authJSONResponder{ctx: c}, http.StatusForbidden, "Backend mode is active. Only admin login is allowed.")
		return
	}

	response.SuccessContext(authJSONResponder{ctx: c}, RefreshTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"` // 可选：撤销指定的Refresh Token
}

// LogoutResponse 登出响应
type LogoutResponse struct {
	Message string `json:"message"`
}

// Logout 用户登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	h.LogoutGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) LogoutGateway(c gatewayctx.GatewayContext) {
	var req LogoutRequest
	// 允许空请求体（向后兼容）
	_ = c.BindJSON(&req)

	// 如果提供了Refresh Token，撤销它
	if req.RefreshToken != "" {
		if err := h.authService.RevokeRefreshToken(c.Request().Context(), req.RefreshToken); err != nil {
			slog.Debug("failed to revoke refresh token", "error", err)
			// 不影响登出流程
		}
	}

	response.SuccessContext(authJSONResponder{ctx: c}, LogoutResponse{
		Message: "Logged out successfully",
	})
}

// RevokeAllSessionsResponse 撤销所有会话响应
type RevokeAllSessionsResponse struct {
	Message string `json:"message"`
}

// RevokeAllSessions 撤销当前用户的所有会话
// POST /api/v1/auth/revoke-all-sessions
func (h *AuthHandler) RevokeAllSessions(c *gin.Context) {
	h.RevokeAllSessionsGateway(gatewayctx.FromGin(c))
}

func (h *AuthHandler) RevokeAllSessionsGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(authGatewayResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	if err := h.authService.RevokeAllUserSessions(c.Request().Context(), subject.UserID); err != nil {
		slog.Error("failed to revoke all sessions", "user_id", subject.UserID, "error", err)
		response.ErrorContext(authGatewayResponder{ctx: c}, http.StatusInternalServerError, "Failed to revoke sessions")
		return
	}

	response.SuccessContext(authGatewayResponder{ctx: c}, RevokeAllSessionsResponse{
		Message: "All sessions have been revoked. Please log in again.",
	})
}

type authGatewayResponder struct {
	ctx gatewayctx.GatewayContext
}

func (g authGatewayResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request()
}

func (g authGatewayResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.WriteJSON(status, payload)
}
