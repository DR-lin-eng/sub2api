package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestApplyOpenAICompactTransportOverride_UsesGatewayHeaderTimeout(t *testing.T) {
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				ResponseHeaderTimeout: 90,
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	req = svc.applyOpenAICompactTransportOverride(req)

	override, ok := GetUpstreamTransportOverride(req.Context())
	require.True(t, ok)
	require.Equal(t, defaultOpenAIStreamingConnectQuickFail, override.DialTimeout)
	require.Equal(t, 90*time.Second, override.ResponseHeaderTimeout)
}

func TestHandleNonStreamingResponse_CompactSSEPartialFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"X-Request-Id": []string{"rid-compact-partial"},
		},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.in_progress","response":{"id":"resp_partial"}}`,
			`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"message","role":"assistant","status":"incomplete"}}`,
			`data: {"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"compacted summary"}`,
		}, "\n"))),
	}

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	usage, err := svc.handleNonStreamingResponse(context.Background(), resp, c, &Account{Type: AccountTypeOAuth}, "gpt-5.3-codex", "gpt-5.3-codex")
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"output":[`)
	require.Contains(t, rec.Body.String(), `compacted summary`)
	require.Equal(t, "sse", rec.Header().Get("X-Sub2API-Compact-Upstream-Format"))
	require.Equal(t, "true", rec.Header().Get("X-Sub2API-Compact-Partial"))
	require.Equal(t, "stream_disconnected", rec.Header().Get("X-Sub2API-Compact-Terminal-Event"))
}

func TestHandleNonStreamingResponse_CompactSSEDisconnectWithoutOutputReturnsFailoverError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"X-Request-Id": []string{"rid-compact-disconnect"},
		},
		Body: io.NopCloser(strings.NewReader(`data: {"type":"response.in_progress","response":{"id":"resp_in_progress"}}`)),
	}

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	usage, err := svc.handleNonStreamingResponse(context.Background(), resp, c, &Account{Type: AccountTypeOAuth}, "gpt-5.3-codex", "gpt-5.3-codex")
	require.Nil(t, usage)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "stream disconnected before completion")
	require.Empty(t, rec.Body.String())
}

func TestHandleNonStreamingResponsePassthrough_CompactSSEPartialFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"X-Request-Id": []string{"rid-compact-pass"},
		},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"message","role":"assistant","status":"incomplete"}}`,
			`data: {"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"passthrough compact"}`,
		}, "\n"))),
	}

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	usage, err := svc.handleNonStreamingResponsePassthrough(context.Background(), resp, c, &Account{Type: AccountTypeOAuth})
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `passthrough compact`)
	require.Equal(t, "true", rec.Header().Get("X-Sub2API-Compact-Partial"))
}
