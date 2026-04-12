package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type chatwebSentinelHelperStub struct{}

func (chatwebSentinelHelperStub) requirementsToken(ctx context.Context, session openAIChatWebSentinelSession) (string, error) {
	return "requirements-token", nil
}

func (chatwebSentinelHelperStub) enforcementToken(ctx context.Context, session openAIChatWebSentinelSession, required bool, seed string, difficulty string) (string, error) {
	return "proof-token", nil
}

func (chatwebSentinelHelperStub) solveTurnstile(ctx context.Context, session openAIChatWebSentinelSession, requirementsToken string, dx string) (string, error) {
	return "turnstile-token", nil
}

func (chatwebSentinelHelperStub) solveSessionObserver(ctx context.Context, session openAIChatWebSentinelSession, proofToken string, collectorDX string) (string, error) {
	return "so-token", nil
}

type chatwebHTTPUpstreamRecorder struct {
	paths     []string
	headers   map[string]http.Header
	bodies    map[string][]byte
	responder func(path string) *http.Response
}

func newChatwebHTTPUpstreamRecorder() *chatwebHTTPUpstreamRecorder {
	return &chatwebHTTPUpstreamRecorder{
		headers: map[string]http.Header{},
		bodies:  map[string][]byte{},
	}
}

func (u *chatwebHTTPUpstreamRecorder) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	path := req.URL.Path
	u.paths = append(u.paths, path)
	u.headers[path] = req.Header.Clone()
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		u.bodies[path] = body
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	if u.responder != nil {
		return u.responder(path), nil
	}

	switch path {
	case "/backend-api/conversation/init":
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"conversation_id":"conv_chatweb"}`))),
		}, nil
	case "/backend-api/sentinel/chat-requirements/prepare":
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"token":"chat-req-token","proofofwork":{"required":false},"turnstile":{"required":false},"so":{"required":false}}`))),
		}, nil
	case "/backend-api/sentinel/chat-requirements/finalize":
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}, nil
	case "/backend-api/f/conversation/prepare":
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type":    []string{"application/json"},
				"X-Conduit-Token": []string{"conduit-token"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(`{"payload":{"action":"next","messages":[{"id":"msg1","author":{"role":"user"},"content":{"content_type":"text","parts":["hello"]},"metadata":{}}],"model":"gpt-5","parent_message_id":"parent1","supports_buffering":true,"timezone_offset_min":-480,"websocket_request_id":"ws1"}}`))),
		}, nil
	case "/backend-api/f/conversation":
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"rid-chatweb"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_chatweb\",\"object\":\"response\",\"model\":\"gpt-5\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":2,\"total_tokens\":3}}}\n\n"))),
		}, nil
	case "/backend-api/sentinel/ping":
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	default:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"unexpected path"}}`))),
		}, nil
	}
}

func (u *chatwebHTTPUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, enableTLSFingerprint bool) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestOpenAIGatewayService_ChatWebForward_UsesConversationChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevFactory := newOpenAIChatWebSentinelHelperRunner
	newOpenAIChatWebSentinelHelperRunner = func() openAIChatWebSentinelHelperRunner {
		return chatwebSentinelHelperStub{}
	}
	defer func() {
		newOpenAIChatWebSentinelHelperRunner = prevFactory
	}()

	upstream := newChatwebHTTPUpstreamRecorder()
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
	}

	account := &Account{
		ID:          999,
		Name:        "chatweb",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
		Extra: map[string]any{
			"openai_auth_mode": OpenAIAuthModeChatWeb,
		},
		Status:         StatusActive,
		Schedulable:    true,
		RateMultiplier: f64p(1),
	}

	body := []byte(`{"model":"gpt-5","stream":false,"input":"hello"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("User-Agent", "Mozilla/5.0")
	c.Request.Header.Set("Content-Type", "application/json")

	result, err := svc.ForwardContext(context.Background(), gatewayctx.FromGin(c), account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []string{
		"/backend-api/conversation/init",
		"/backend-api/sentinel/chat-requirements/prepare",
		"/backend-api/sentinel/chat-requirements/finalize",
		"/backend-api/f/conversation/prepare",
		"/backend-api/f/conversation",
		"/backend-api/sentinel/ping",
	}, upstream.paths)
	require.Contains(t, string(upstream.bodies["/backend-api/f/conversation"]), `"action":"next"`)
	require.Equal(t, "proof-token", upstream.headers["/backend-api/f/conversation"].Get("openai-sentinel-proof-token"))
	require.Equal(t, "chat-req-token", upstream.headers["/backend-api/f/conversation"].Get("openai-sentinel-chat-requirements-token"))
	require.Equal(t, "conduit-token", upstream.headers["/backend-api/f/conversation"].Get("x-conduit-token"))
	require.Contains(t, rec.Body.String(), `"id":"resp_chatweb"`)
}

func TestOpenAIChatWebBuildConversationPayload_PreservesToolsAndToolResults(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5",
		"tools":[{"type":"function","name":"lookup","description":"Lookup","parameters":{"type":"object"}}],
		"tool_choice":"auto",
		"input":[
			{"role":"system","content":"You are helpful"},
			{"role":"user","content":"weather?"},
			{"type":"function_call","call_id":"fc_1","name":"lookup","arguments":"{\"city\":\"Tokyo\"}"},
			{"type":"function_call_output","call_id":"fc_1","output":"sunny"}
		]
	}`)

	payload, err := openAIChatWebBuildConversationPayload(body)
	require.NoError(t, err)
	require.Equal(t, "next", payload["action"])
	require.Equal(t, "auto", payload["tool_choice"])

	tools, ok := payload["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)

	messages, ok := payload["messages"].([]map[string]any)
	if !ok {
		raw, marshalErr := json.Marshal(payload["messages"])
		require.NoError(t, marshalErr)
		var converted []map[string]any
		require.NoError(t, json.Unmarshal(raw, &converted))
		messages = converted
	}
	require.Len(t, messages, 4)
	require.Equal(t, "system", messages[0]["author"].(map[string]any)["role"])
	require.Equal(t, "user", messages[1]["author"].(map[string]any)["role"])
	require.Equal(t, "assistant", messages[2]["author"].(map[string]any)["role"])
	require.Equal(t, "tool", messages[3]["author"].(map[string]any)["role"])
	require.Equal(t, "lookup", messages[2]["metadata"].(map[string]any)["tool_name"])
	require.Equal(t, "fc_1", messages[2]["metadata"].(map[string]any)["tool_call_id"])
	require.Equal(t, "fc_1", messages[3]["metadata"].(map[string]any)["tool_call_id"])
	require.Equal(t, "sunny", messages[3]["content"].(map[string]any)["parts"].([]string)[0])
}

func TestOpenAIGatewayService_ChatWebForwardAsChatCompletions_UsesConversationChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevFactory := newOpenAIChatWebSentinelHelperRunner
	newOpenAIChatWebSentinelHelperRunner = func() openAIChatWebSentinelHelperRunner {
		return chatwebSentinelHelperStub{}
	}
	defer func() {
		newOpenAIChatWebSentinelHelperRunner = prevFactory
	}()

	upstream := newChatwebHTTPUpstreamRecorder()
	upstream.bodies["/backend-api/f/conversation"] = nil
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
	}

	account := &Account{
		ID:          1001,
		Name:        "chatweb-chat",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
		Extra: map[string]any{
			"openai_auth_mode": OpenAIAuthModeChatWeb,
		},
		Status:         StatusActive,
		Schedulable:    true,
		RateMultiplier: f64p(1),
	}

	body := []byte(`{"model":"gpt-5","stream":false,"messages":[{"role":"user","content":"hello"}]}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("User-Agent", "Mozilla/5.0")
	c.Request.Header.Set("Content-Type", "application/json")

	result, err := svc.ForwardAsChatCompletionsContext(context.Background(), gatewayctx.FromGin(c), account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/backend-api/f/conversation", upstream.paths[4])
	require.Contains(t, rec.Body.String(), `"object":"chat.completion"`)
}

func TestOpenAIGatewayService_ChatWebForwardAsAnthropic_UsesConversationChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevFactory := newOpenAIChatWebSentinelHelperRunner
	newOpenAIChatWebSentinelHelperRunner = func() openAIChatWebSentinelHelperRunner {
		return chatwebSentinelHelperStub{}
	}
	defer func() {
		newOpenAIChatWebSentinelHelperRunner = prevFactory
	}()

	upstream := newChatwebHTTPUpstreamRecorder()
	upstream.responder = func(path string) *http.Response {
		switch path {
		case "/backend-api/conversation/init":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"conversation_id":"conv_chatweb"}`))),
			}
		case "/backend-api/sentinel/chat-requirements/prepare":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"token":"chat-req-token","proofofwork":{"required":false},"turnstile":{"required":false},"so":{"required":false}}`))),
			}
		case "/backend-api/sentinel/chat-requirements/finalize":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
			}
		case "/backend-api/f/conversation/prepare":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type":    []string{"application/json"},
					"X-Conduit-Token": []string{"conduit-token"},
				},
				Body: io.NopCloser(bytes.NewReader([]byte(`{"payload":{"action":"next","messages":[{"id":"msg1","author":{"role":"user"},"content":{"content_type":"text","parts":["hello"]},"metadata":{}}],"model":"gpt-5","parent_message_id":"parent1","supports_buffering":true,"timezone_offset_min":-480,"websocket_request_id":"ws1"}}`))),
			}
		case "/backend-api/f/conversation":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"text/event-stream"},
					"x-request-id": []string{"rid-chatweb"},
				},
				Body: io.NopCloser(bytes.NewReader([]byte(
					"data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"message\",\"id\":\"msg_1\",\"role\":\"assistant\",\"content\":[],\"status\":\"in_progress\"}}\n\n" +
						"data: {\"type\":\"response.output_text.delta\",\"output_index\":0,\"content_index\":0,\"delta\":\"hello from chatweb\"}\n\n" +
						"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_chatweb\",\"object\":\"response\",\"model\":\"gpt-5\",\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"id\":\"msg_1\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"hello from chatweb\"}],\"status\":\"completed\"}],\"usage\":{\"input_tokens\":1,\"output_tokens\":2,\"total_tokens\":3}}}\n\n",
				))),
			}
		case "/backend-api/sentinel/ping":
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"unexpected path"}}`))),
			}
		}
	}
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
	}

	account := &Account{
		ID:          1002,
		Name:        "chatweb-messages",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
		Extra: map[string]any{
			"openai_auth_mode": OpenAIAuthModeChatWeb,
		},
		Status:         StatusActive,
		Schedulable:    true,
		RateMultiplier: f64p(1),
	}

	body := []byte(`{"model":"gpt-5","max_tokens":512,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request.Header.Set("User-Agent", "Mozilla/5.0")
	c.Request.Header.Set("Content-Type", "application/json")

	result, err := svc.ForwardAsAnthropicContext(context.Background(), gatewayctx.FromGin(c), account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/backend-api/f/conversation", upstream.paths[4])
	require.Contains(t, rec.Body.String(), `"type":"message"`)
	require.Contains(t, rec.Body.String(), `"hello from chatweb"`)
}
