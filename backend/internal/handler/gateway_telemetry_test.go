package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandler_ClaudeBootstrapGateway_ReturnsMockedBootstrapForAnthropicGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/claude_cli/bootstrap", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformAnthropic},
	})

	h := &GatewayHandler{}
	h.ClaudeBootstrap(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "private, max-age=3600", rec.Header().Get("Cache-Control"))
	require.JSONEq(t, `{"client_data":{},"additional_model_options":[]}`, rec.Body.String())
}

func TestGatewayHandler_ClaudeMetricsEnabledGateway_ReturnsDisabledForAnthropicCompatibleGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/claude_code/organizations/metrics_enabled", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformAntigravity},
	})

	h := &GatewayHandler{}
	h.ClaudeMetricsEnabled(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "private, max-age=86400", rec.Header().Get("Cache-Control"))
	require.JSONEq(t, `{"metrics_logging_enabled":false}`, rec.Body.String())
}

func TestGatewayHandler_ClaudeBootstrapGateway_ReturnsNotFoundForOpenAIGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/claude_cli/bootstrap", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI},
	})

	h := &GatewayHandler{}
	h.ClaudeBootstrap(c)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.JSONEq(t, `{"type":"error","error":{"type":"not_found_error","message":"Bootstrap is not supported for this platform"}}`, rec.Body.String())
}

func TestGatewayHandler_ClaudeManagedSettingsGateway_UsesETagAndNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/claude_code/settings", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformAnthropic},
	})

	h := &GatewayHandler{}
	h.ClaudeManagedSettings(c)

	require.Equal(t, http.StatusNoContent, rec.Code)
	etag := rec.Header().Get("ETag")
	require.NotEmpty(t, etag)
	require.Equal(t, "private, max-age=3600", rec.Header().Get("Cache-Control"))

	rec2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(rec2)
	req2 := httptest.NewRequest(http.MethodGet, "/api/claude_code/settings", nil)
	req2.Header.Set("If-None-Match", etag)
	c2.Request = req2
	c2.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformAnthropic},
	})
	h.ClaudeManagedSettings(c2)
	require.Equal(t, http.StatusNotModified, rec2.Code)
}

func TestGatewayHandler_ClaudePolicyLimitsGateway_ReturnsEmptyRestrictions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/claude_code/policy_limits", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformAnthropic},
	})

	h := &GatewayHandler{}
	h.ClaudePolicyLimits(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Header().Get("ETag"))
	require.JSONEq(t, `{"restrictions":{}}`, rec.Body.String())
}

func TestGatewayHandler_ClaudeUserSettingsGateway_PutThenGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &GatewayHandler{}
	key := &service.APIKey{Group: &service.Group{Platform: service.PlatformAnthropic}}
	subject := middleware.AuthSubject{UserID: 42, Concurrency: 1}

	putRec := httptest.NewRecorder()
	putCtx, _ := gin.CreateTestContext(putRec)
	putCtx.Request = httptest.NewRequest(http.MethodPut, "/api/claude_code/user_settings", bytes.NewBufferString(`{"entries":{"~/.claude/settings.json":"{}","~/.claude/CLAUDE.md":"memo"}}`))
	putCtx.Request.Header.Set("Content-Type", "application/json")
	putCtx.Set(string(middleware.ContextKeyAPIKey), key)
	putCtx.Set(string(middleware.ContextKeyUser), subject)

	h.ClaudeUpdateUserSettings(putCtx)
	require.Equal(t, http.StatusOK, putRec.Code)
	require.Contains(t, putRec.Body.String(), `"checksum"`)
	require.Contains(t, putRec.Body.String(), `"lastModified"`)

	getRec := httptest.NewRecorder()
	getCtx, _ := gin.CreateTestContext(getRec)
	getCtx.Request = httptest.NewRequest(http.MethodGet, "/api/claude_code/user_settings", nil)
	getCtx.Set(string(middleware.ContextKeyAPIKey), key)
	getCtx.Set(string(middleware.ContextKeyUser), subject)

	h.ClaudeUserSettings(getCtx)
	require.Equal(t, http.StatusOK, getRec.Code)
	require.Contains(t, getRec.Body.String(), `"userId":"42"`)
	require.Contains(t, getRec.Body.String(), `"version":1`)
	require.Contains(t, getRec.Body.String(), `"lastModified":"`)
	require.Contains(t, getRec.Body.String(), `"checksum":"cdbb10254979e5b29306b3e721195f84"`)
	require.Contains(t, getRec.Body.String(), `"~/.claude/settings.json":"{}"`)
	require.Contains(t, getRec.Body.String(), `"~/.claude/CLAUDE.md":"memo"`)
}
