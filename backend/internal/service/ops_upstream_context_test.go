package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAppendOpsUpstreamError_UsesPreparedRequestBodyFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	setOpsUpstreamRequestBody(c, []byte(`{"model":"gpt-5"}`))
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Kind:    "http_error",
		Message: "upstream failed",
	})

	v, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := v.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, `{"model":"gpt-5"}`, events[0].UpstreamRequestBody)
}

func TestAppendOpsUpstreamError_UsesRequestBodyStringFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	c.Set(OpsUpstreamRequestBodyKey, `{"model":"gpt-4"}`)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Kind:    "request_error",
		Message: "dial timeout",
	})

	v, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := v.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, `{"model":"gpt-4"}`, events[0].UpstreamRequestBody)
}

func TestAppendOpsUpstreamError_MarksPreparedRequestBodyTruncated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	c.Set(OpsUpstreamRequestBodyKey, &OpsPreparedRequestBody{
		JSON:      `{"request_body_truncated":true}`,
		Truncated: true,
		Bytes:     1024 * 1024,
	})
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Kind:    "failover",
		Message: "upstream failed",
	})

	v, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := v.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, `{"request_body_truncated":true}`, events[0].UpstreamRequestBody)
	require.Contains(t, events[0].Kind, "request_body_truncated")
}

func TestAppendOpsUpstreamError_DeduplicatesPreparedRequestBodyAcrossEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	setOpsUpstreamRequestBody(c, []byte(`{"model":"gpt-5","stream":true}`))
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Kind:    "request_error",
		Message: "timeout",
	})
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Kind:    "request_error",
		Message: "connection reset",
	})

	v, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := v.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 2)
	require.NotEmpty(t, events[0].UpstreamRequestBody)
	require.Empty(t, events[1].UpstreamRequestBody, "same prepared body should not be duplicated on every failover event")
}
