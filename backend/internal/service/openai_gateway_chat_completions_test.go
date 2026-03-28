package service

import (
	"bytes"
	"context"
	"encoding/json"
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

type chatCompletionsHTTPUpstreamRecorder struct {
	lastReq         *http.Request
	lastBody        []byte
	resp            *http.Response
	err             error
	override        UpstreamTransportOverride
	overrideApplied bool
	deadlineIn      time.Duration
	deadlineSet     bool
}

func (u *chatCompletionsHTTPUpstreamRecorder) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	u.lastReq = req
	if req != nil && req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		u.lastBody = body
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	if req != nil {
		u.override, u.overrideApplied = GetUpstreamTransportOverride(req.Context())
		if deadline, ok := req.Context().Deadline(); ok {
			u.deadlineSet = true
			u.deadlineIn = time.Until(deadline)
		}
	}
	if u.err != nil {
		return nil, u.err
	}
	return u.resp, nil
}

func (u *chatCompletionsHTTPUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, enableTLSFingerprint bool) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func newOpenAIAPIKeyAccountForChatCompatTest() *Account {
	return &Account{
		ID:          301,
		Name:        "openai-chat-compat-test",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func TestForwardAsChatCompletions_StreamAppliesTransportOverrideAndIncludesUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &chatCompletionsHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"x-request-id": []string{"rid-chat-stream"}},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				`data: {"type":"response.created","response":{"id":"resp_chat_stream","model":"gpt-5.4"}}`,
				``,
				`data: {"type":"response.output_text.delta","delta":"hello"}`,
				``,
				`data: {"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":3,"output_tokens":5,"input_tokens_details":{"cached_tokens":2}}}}`,
				``,
				`data: [DONE]`,
				``,
			}, "\n"))),
		},
	}
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				MaxLineSize: defaultMaxLineSize,
			},
		},
		httpUpstream: upstream,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	body := []byte(`{"model":"gpt-5.4","stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"hello"}]}`)
	result, err := svc.ForwardAsChatCompletions(context.Background(), c, newOpenAIAPIKeyAccountForChatCompatTest(), body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
	require.Equal(t, 2, result.Usage.CacheReadInputTokens)

	require.NotNil(t, upstream.lastReq)
	require.True(t, upstream.overrideApplied)
	require.Greater(t, upstream.override.DialTimeout, time.Duration(0))
	require.Greater(t, upstream.override.ResponseHeaderTimeout, time.Duration(0))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
	require.Contains(t, rec.Body.String(), `"object":"chat.completion.chunk"`)
	require.Contains(t, rec.Body.String(), `"content":"hello"`)
	require.Contains(t, rec.Body.String(), `"usage":{"prompt_tokens":3,"completion_tokens":5,"total_tokens":8`)
	require.Contains(t, rec.Body.String(), "data: [DONE]")
}

func TestForwardAsChatCompletions_NonStreamUsesProxyQuickFailAndBuffersJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &chatCompletionsHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"x-request-id": []string{"rid-chat-buffered"}},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				`data: {"type":"response.completed","response":{"id":"resp_buffered","status":"completed","usage":{"input_tokens":7,"output_tokens":4,"input_tokens_details":{"cached_tokens":1}},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"buffered ok"}]}]}}`,
				``,
				`data: [DONE]`,
				``,
			}, "\n"))),
		},
	}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}

	account := newOpenAIAPIKeyAccountForChatCompatTest()
	account.Proxy = &Proxy{
		ID:       88,
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     8080,
		Status:   StatusActive,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	body := []byte(`{"model":"gpt-5.4","stream":false,"messages":[{"role":"user","content":"hello"}]}`)
	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Stream)
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 4, result.Usage.OutputTokens)
	require.Equal(t, 1, result.Usage.CacheReadInputTokens)

	require.NotNil(t, upstream.lastReq)
	require.False(t, upstream.overrideApplied)
	require.True(t, upstream.deadlineSet)
	require.Greater(t, upstream.deadlineIn, time.Duration(0))
	require.LessOrEqual(t, upstream.deadlineIn, proxyRequestQuickFailTimeout+time.Second)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &parsed))
	choices, ok := parsed["choices"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, choices)

	firstChoice, ok := choices[0].(map[string]any)
	require.True(t, ok)
	message, ok := firstChoice["message"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "buffered ok", message["content"])
}

func TestForwardAsChatCompletions_UpstreamFailoverReturnsFailoverError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &chatCompletionsHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"overloaded"}}`)),
		},
	}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	body := []byte(`{"model":"gpt-5.4","stream":true,"messages":[{"role":"user","content":"hello"}]}`)
	result, err := svc.ForwardAsChatCompletions(context.Background(), c, newOpenAIAPIKeyAccountForChatCompatTest(), body, "", "")
	require.Nil(t, result)

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "overloaded")
	require.Equal(t, 0, rec.Body.Len())
}
