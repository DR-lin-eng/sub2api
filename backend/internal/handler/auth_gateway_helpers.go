package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func isRequestHTTPSGateway(c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil {
		return false
	}
	if c.Request().TLS != nil {
		return true
	}
	proto := strings.ToLower(strings.TrimSpace(c.HeaderValue("X-Forwarded-Proto")))
	return proto == "https"
}

func readCookieDecodedGateway(c gatewayctx.GatewayContext, name string) (string, error) {
	if c == nil || c.Request() == nil {
		return "", http.ErrNoCookie
	}
	ck, err := c.Request().Cookie(name)
	if err != nil {
		return "", err
	}
	return decodeCookieValue(ck.Value)
}

func clearOAuthBindAccessTokenCookieGateway(c gatewayctx.GatewayContext, secure bool) {
	if c == nil {
		return
	}
	c.SetCookie(&http.Cookie{
		Name:     oauthBindAccessTokenCookieName,
		Value:    "",
		Path:     oauthBindAccessTokenCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func setOAuthBindAccessTokenCookieGateway(c gatewayctx.GatewayContext, token string, secure bool) {
	if c == nil {
		return
	}
	c.SetCookie(&http.Cookie{
		Name:     oauthBindAccessTokenCookieName,
		Value:    url.QueryEscape(strings.TrimSpace(token)),
		Path:     oauthBindAccessTokenCookiePath,
		MaxAge:   linuxDoOAuthCookieMaxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthPendingBrowserCookieGateway(c gatewayctx.GatewayContext, secure bool) {
	if c == nil {
		return
	}
	c.SetCookie(&http.Cookie{
		Name:     oauthPendingBrowserCookieName,
		Value:    "",
		Path:     oauthPendingBrowserCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthPendingSessionCookieGateway(c gatewayctx.GatewayContext, secure bool) {
	if c == nil {
		return
	}
	c.SetCookie(&http.Cookie{
		Name:     oauthPendingSessionCookieName,
		Value:    "",
		Path:     oauthPendingSessionCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func setOAuthPendingBrowserCookieGateway(c gatewayctx.GatewayContext, sessionKey string, secure bool) {
	if c == nil {
		return
	}
	c.SetCookie(&http.Cookie{
		Name:     oauthPendingBrowserCookieName,
		Value:    encodeCookieValue(sessionKey),
		Path:     oauthPendingBrowserCookiePath,
		MaxAge:   oauthPendingCookieMaxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func setOAuthPendingSessionCookieGateway(c gatewayctx.GatewayContext, sessionToken string, secure bool) {
	if c == nil {
		return
	}
	c.SetCookie(&http.Cookie{
		Name:     oauthPendingSessionCookieName,
		Value:    encodeCookieValue(sessionToken),
		Path:     oauthPendingSessionCookiePath,
		MaxAge:   oauthPendingCookieMaxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func setOAuthCookieGateway(c gatewayctx.GatewayContext, name, value, path string, maxAgeSec int, secure bool) {
	if c == nil {
		return
	}
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthCookieGateway(c gatewayctx.GatewayContext, name, path string, secure bool) {
	if c == nil {
		return
	}
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthPendingBrowserCookieGateway(c gatewayctx.GatewayContext) (string, error) {
	return readCookieDecodedGateway(c, oauthPendingBrowserCookieName)
}

func readOAuthPendingSessionCookieGateway(c gatewayctx.GatewayContext) (string, error) {
	return readCookieDecodedGateway(c, oauthPendingSessionCookieName)
}

func (h *AuthHandler) resolveOAuthBindTargetUserIDGateway(c gatewayctx.GatewayContext) (*int64, error) {
	if subject, ok := servermiddleware.GetAuthSubjectFromGatewayContext(c); ok && subject.UserID > 0 {
		return &subject.UserID, nil
	}
	if h == nil || h.authService == nil || h.userService == nil || c == nil || c.Request() == nil {
		return nil, service.ErrInvalidToken
	}

	ck, err := c.Request().Cookie(oauthBindAccessTokenCookieName)
	clearOAuthBindAccessTokenCookieGateway(c, isRequestHTTPSGateway(c))
	if err != nil {
		return nil, err
	}

	tokenString, err := url.QueryUnescape(strings.TrimSpace(ck.Value))
	if err != nil {
		return nil, err
	}
	if tokenString == "" {
		return nil, service.ErrInvalidToken
	}

	claims, err := h.authService.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	user, err := h.userService.GetByID(c.Request().Context(), claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive() || claims.TokenVersion != user.TokenVersion {
		return nil, service.ErrInvalidToken
	}
	return &user.ID, nil
}

func (h *AuthHandler) PrepareOAuthBindAccessTokenCookieGateway(c gatewayctx.GatewayContext) {
	const bearerPrefix = "Bearer "

	authHeader := strings.TrimSpace(c.HeaderValue("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authHeader), strings.ToLower(bearerPrefix)) {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, infraerrors.Unauthorized("UNAUTHORIZED", "authentication required"))
		return
	}

	token := strings.TrimSpace(authHeader[len(bearerPrefix):])
	if token == "" {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, infraerrors.Unauthorized("UNAUTHORIZED", "authentication required"))
		return
	}

	setOAuthBindAccessTokenCookieGateway(c, token, isRequestHTTPSGateway(c))
	c.SetStatus(http.StatusNoContent)
}

func (h *AuthHandler) buildOAuthBindUserCookieFromGatewayContext(c gatewayctx.GatewayContext) (string, error) {
	userID, err := h.resolveOAuthBindTargetUserIDGateway(c)
	if err != nil || userID == nil || *userID <= 0 {
		return "", infraerrors.Unauthorized("UNAUTHORIZED", "authentication required")
	}
	return buildOAuthBindUserCookieValue(*userID, h.oauthBindCookieSecret())
}

func (h *AuthHandler) readOAuthBindUserIDFromCookieGateway(c gatewayctx.GatewayContext, cookieName string) (int64, error) {
	value, err := readCookieDecodedGateway(c, cookieName)
	if err != nil {
		return 0, err
	}
	return parseOAuthBindUserCookieValue(value, h.oauthBindCookieSecret())
}

func redirectWithFragmentGateway(c gatewayctx.GatewayContext, frontendCallback string, fragment url.Values) {
	u, err := url.Parse(frontendCallback)
	if err != nil {
		c.Redirect(http.StatusFound, linuxDoOAuthDefaultRedirectTo)
		return
	}
	if u.Scheme != "" && !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		c.Redirect(http.StatusFound, linuxDoOAuthDefaultRedirectTo)
		return
	}
	u.Fragment = fragment.Encode()
	c.SetHeader("Cache-Control", "no-store")
	c.SetHeader("Pragma", "no-cache")
	c.Redirect(http.StatusFound, u.String())
}

func redirectOAuthErrorGateway(c gatewayctx.GatewayContext, frontendCallback string, code string, message string, description string) {
	fragment := url.Values{}
	fragment.Set("error", truncateFragmentValue(code))
	if strings.TrimSpace(message) != "" {
		fragment.Set("error_message", truncateFragmentValue(message))
	}
	if strings.TrimSpace(description) != "" {
		fragment.Set("error_description", truncateFragmentValue(description))
	}
	redirectWithFragmentGateway(c, frontendCallback, fragment)
}

func redirectToFrontendCallbackGateway(c gatewayctx.GatewayContext, frontendCallback string) {
	u, err := url.Parse(frontendCallback)
	if err != nil {
		c.Redirect(http.StatusFound, linuxDoOAuthDefaultRedirectTo)
		return
	}
	if u.Scheme != "" && !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		c.Redirect(http.StatusFound, linuxDoOAuthDefaultRedirectTo)
		return
	}
	u.Fragment = ""
	c.SetHeader("Cache-Control", "no-store")
	c.SetHeader("Pragma", "no-cache")
	c.Redirect(http.StatusFound, u.String())
}

func readPendingOAuthBrowserSessionGateway(c gatewayctx.GatewayContext, h *AuthHandler) (*service.AuthPendingIdentityService, *dbent.PendingAuthSession, func(), error) {
	secureCookie := isRequestHTTPSGateway(c)
	clearCookies := func() {
		clearOAuthPendingSessionCookieGateway(c, secureCookie)
		clearOAuthPendingBrowserCookieGateway(c, secureCookie)
	}

	sessionToken, err := readOAuthPendingSessionCookieGateway(c)
	if err != nil || strings.TrimSpace(sessionToken) == "" {
		clearCookies()
		return nil, nil, clearCookies, service.ErrPendingAuthSessionNotFound
	}
	browserSessionKey, err := readOAuthPendingBrowserCookieGateway(c)
	if err != nil || strings.TrimSpace(browserSessionKey) == "" {
		clearCookies()
		return nil, nil, clearCookies, service.ErrPendingAuthBrowserMismatch
	}

	svc, err := h.pendingIdentityService()
	if err != nil {
		clearCookies()
		return nil, nil, clearCookies, err
	}

	session, err := svc.GetBrowserSession(c.Request().Context(), sessionToken, browserSessionKey)
	if err != nil {
		clearCookies()
		return nil, nil, clearCookies, err
	}

	return svc, session, clearCookies, nil
}

func transitionPendingOAuthAccountToChoiceStateWithContext(
	ctx context.Context,
	client *dbent.Client,
	session *dbent.PendingAuthSession,
	targetUser *dbent.User,
	email string,
) (*dbent.PendingAuthSession, error) {
	completionResponse := pendingOAuthChoiceCompletionResponse(session, email)
	var targetUserID *int64
	if targetUser != nil && targetUser.ID > 0 {
		targetUserID = &targetUser.ID
	}
	session, err := updatePendingOAuthSessionProgress(
		ctx,
		client,
		session,
		strings.TrimSpace(session.Intent),
		email,
		targetUserID,
		completionResponse,
	)
	if err != nil {
		return nil, infraerrors.InternalServer("PENDING_AUTH_SESSION_UPDATE_FAILED", "failed to update pending oauth session").WithCause(err)
	}
	return session, nil
}
