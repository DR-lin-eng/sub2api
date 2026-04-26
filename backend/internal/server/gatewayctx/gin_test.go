package gatewayctx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdaptGinHandlerExposesTransportNeutralOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/items/:id", AdaptGinHandler(func(c GatewayContext) {
		require.Equal(t, http.MethodGet, c.Method())
		require.Equal(t, "/items/42", c.Path())
		require.Equal(t, "42", c.PathParam("id"))
		require.Equal(t, "abc", c.QueryValue("q"))
		require.Equal(t, "demo", c.HeaderValue("X-Test"))
		c.SetHeader("X-Handled-By", "gatewayctx")
		c.WriteJSON(http.StatusAccepted, map[string]any{
			"id":     c.PathParam("id"),
			"query":  c.QueryValue("q"),
			"method": c.Method(),
		})
	}))

	req := httptest.NewRequest(http.MethodGet, "/items/42?q=abc", nil)
	req.Header.Set("X-Test", "demo")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, "gatewayctx", rec.Header().Get("X-Handled-By"))
	require.JSONEq(t, `{"id":"42","query":"abc","method":"GET"}`, rec.Body.String())
}

func TestWriteSSECommentAndData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/stream", AdaptGinHandler(func(c GatewayContext) {
		require.NoError(t, WriteSSEComment(c, "ping"))
		require.NoError(t, WriteSSEData(c, "hello\nworld"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	require.Equal(t, ":ping\n\ndata: hello\ndata: world\n\n", rec.Body.String())
}

func TestGinGatewayContextClientIP_IgnoresSpoofedForwardedHeadersWhenUntrusted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, engine := gin.CreateTestContext(httptest.NewRecorder())
	require.NoError(t, engine.SetTrustedProxies(nil))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "203.0.113.88:54321"
	req.Header.Set("CF-Connecting-IP", "203.0.113.7")
	req.Header.Set("X-Real-IP", "198.51.100.9")
	req.Header.Set("X-Forwarded-For", "198.51.100.10, 10.0.0.8")
	ctx.Request = req

	require.Equal(t, "203.0.113.88", FromGin(ctx).ClientIP())
}

func TestGinGatewayContextClientIP_UsesTrustedProxyChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, engine := gin.CreateTestContext(httptest.NewRecorder())
	require.NoError(t, engine.SetTrustedProxies([]string{"10.0.0.0/8"}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.8:54321"
	req.Header.Set("X-Forwarded-For", "10.1.1.1, 198.51.100.77, 10.0.0.8")
	ctx.Request = req

	require.Equal(t, "198.51.100.77", FromGin(ctx).ClientIP())
}

func TestGinGatewayContextClientIP_FallsBackToGinClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "203.0.113.88:54321"
	ctx.Request = req

	require.Equal(t, "203.0.113.88", FromGin(ctx).ClientIP())
}
