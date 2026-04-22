package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const tinyPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+4Q0AAAAASUVORK5CYII="

func TestAccountSupportsOpenAIImageCapability_ChatWeb(t *testing.T) {
	chatweb := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"openai_auth_mode": OpenAIAuthModeChatWeb,
		},
	}
	apiKey := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	}

	require.True(t, chatweb.SupportsOpenAIImageCapability(OpenAIImagesCapabilityBasic))
	require.True(t, chatweb.SupportsOpenAIImageCapability(OpenAIImagesCapabilityChatWebEdit))
	require.False(t, chatweb.SupportsOpenAIImageCapability(OpenAIImagesCapabilityNative))

	require.True(t, apiKey.SupportsOpenAIImageCapability(OpenAIImagesCapabilityBasic))
	require.True(t, apiKey.SupportsOpenAIImageCapability(OpenAIImagesCapabilityChatWebEdit))
	require.True(t, apiKey.SupportsOpenAIImageCapability(OpenAIImagesCapabilityNative))
}

func TestParseOpenAIChatWebChatCompletionsImageRequest(t *testing.T) {
	body := []byte(`{
		"model":"gpt-image-1",
		"n":2,
		"messages":[
			{
				"role":"user",
				"content":[
					{"type":"text","text":"画一只猫"},
					{"type":"image_url","image_url":{"url":"` + tinyPNGDataURL + `"}}
				]
			}
		]
	}`)

	req, handled, err := parseOpenAIChatWebChatCompletionsImageRequest(body)
	require.NoError(t, err)
	require.True(t, handled)
	require.NotNil(t, req)
	require.Equal(t, "gpt-image-1", req.RequestModel)
	require.Equal(t, "画一只猫", req.Prompt)
	require.Equal(t, 2, req.N)
	require.Len(t, req.Uploads, 1)
	require.Equal(t, 1, req.Uploads[0].Width)
	require.Equal(t, 1, req.Uploads[0].Height)
}

func TestParseOpenAIChatWebResponsesImageRequest(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5",
		"input":[
			{"type":"input_text","text":"生成一张海边日落"},
			{"type":"input_image","image_url":"` + tinyPNGDataURL + `"}
		],
		"tools":[{"type":"image_generation"}]
	}`)

	req, handled, err := parseOpenAIChatWebResponsesImageRequest(body)
	require.NoError(t, err)
	require.True(t, handled)
	require.NotNil(t, req)
	require.Equal(t, "gpt-5", req.RequestModel)
	require.Equal(t, "生成一张海边日落", req.Prompt)
	require.Equal(t, 1, req.N)
	require.Len(t, req.Uploads, 1)
	require.Equal(t, "image/png", req.Uploads[0].ContentType)
}

func TestParseOpenAIChatWebImageStreamBodyAndFilter(t *testing.T) {
	body := []byte("data: {\"conversation_id\":\"conv_123\",\"message\":{\"content\":{\"content_type\":\"text\",\"parts\":[\"done\"]}}}\n\n" +
		"data: {\"pointer\":\"file-service://file_out\"}\n\n" +
		"data: {\"pointer\":\"sediment://file_in\"}\n\n")

	summary := parseOpenAIChatWebImageStreamBody(body)
	require.NotNil(t, summary)
	require.Equal(t, "conv_123", summary.ConversationID)
	require.Contains(t, summary.FileIDs, "file_out")
	require.Contains(t, summary.FileIDs, "sed:file_in")
	require.Equal(t, "done", summary.Text)

	filtered := filterOpenAIChatWebOutputFileIDs(summary.FileIDs, []openAIChatWebUploadedImage{
		{FileID: "file_in"},
	})
	require.Equal(t, []string{"file_out"}, filtered)
}

func TestResolveOpenAIChatWebImageUpstreamModel(t *testing.T) {
	free := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "free",
		},
	}
	plus := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "plus",
		},
	}

	require.Equal(t, "auto", resolveOpenAIChatWebImageUpstreamModel(free, "gpt-image-1", ""))
	require.Equal(t, "auto", resolveOpenAIChatWebImageUpstreamModel(free, "gpt-image-2", ""))
	require.Equal(t, "gpt-5-3", resolveOpenAIChatWebImageUpstreamModel(plus, "gpt-image-2", ""))
	require.Equal(t, "mapped-model", resolveOpenAIChatWebImageUpstreamModel(plus, "gpt-image-2", "mapped-model"))
}

func TestOpenAIGatewayService_ChatWebImagesFlow_UploadConversationDownload(t *testing.T) {
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
		case "/backend-api/sentinel/chat-requirements/prepare":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"token":"chat-req-token","proofofwork":{"required":false}}`))),
			}
		case "/backend-api/sentinel/chat-requirements/finalize":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
			}
		case "/backend-api/files":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"upload_url":"https://upload.example.com/blob/1","file_id":"file_input_1"}`))),
			}
		case "/blob/1":
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}
		case "/backend-api/files/process_upload_stream":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
			}
		case "/backend-api/conversation":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"text/event-stream"},
					"x-request-id": []string{"rid-chatweb-image"},
				},
				Body: io.NopCloser(bytes.NewReader([]byte(
					"data: {\"conversation_id\":\"conv_img_1\"}\n\n" +
						"data: {\"pointer\":\"file-service://file_output_1\"}\n\n" +
						"data: {\"message\":{\"content\":{\"content_type\":\"text\",\"parts\":[\"done\"]}}}\n\n",
				))),
			}
		case "/backend-api/files/file_output_1/download":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"download_url":"https://download.example.com/result.png"}`))),
			}
		case "/result.png":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       io.NopCloser(bytes.NewReader([]byte("png-binary"))),
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

	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          2001,
		Name:        "chatweb-image",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
			"plan_type":          "plus",
		},
		Extra: map[string]any{
			"openai_auth_mode": OpenAIAuthModeChatWeb,
		},
		Status:         StatusActive,
		Schedulable:    true,
		RateMultiplier: f64p(1),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(nil))
	c.Request.Header.Set("User-Agent", "Mozilla/5.0")
	c.Request.Header.Set("Content-Type", "multipart/form-data")

	parsed := &OpenAIImagesRequest{
		Endpoint:           openAIImagesEditsEndpoint,
		Multipart:          true,
		Model:              "gpt-image-2",
		Prompt:             "把这张图改成赛博朋克风格",
		N:                  1,
		RequiredCapability: OpenAIImagesCapabilityChatWebEdit,
		Uploads: []OpenAIImagesUpload{
			{
				FieldName:   "image",
				FileName:    "source.png",
				ContentType: "image/png",
				Data:        []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1},
				Width:       1,
				Height:      1,
			},
		},
		SizeTier: "2K",
	}

	result, err := svc.ForwardImagesContext(context.Background(), gatewayctx.FromGin(c), account, nil, parsed, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "gpt-image-2", result.Model)
	require.Equal(t, "gpt-5-3", result.UpstreamModel)
	require.Equal(t, 1, result.ImageCount)
	require.Contains(t, rec.Body.String(), `"created"`)
	require.Contains(t, rec.Body.String(), `"b64_json":"cG5nLWJpbmFyeQ=="`)
	require.Equal(t, []string{
		"/backend-api/sentinel/chat-requirements/prepare",
		"/backend-api/sentinel/chat-requirements/finalize",
		"/backend-api/files",
		"/blob/1",
		"/backend-api/files/process_upload_stream",
		"/backend-api/conversation",
		"/backend-api/files/file_output_1/download",
		"/result.png",
		"/backend-api/sentinel/ping",
	}, upstream.paths)
	require.Contains(t, string(upstream.bodies["/backend-api/conversation"]), `"system_hints":["picture_v2"]`)
	require.Contains(t, string(upstream.bodies["/backend-api/conversation"]), `"asset_pointer":"sediment://file_input_1"`)
	require.Equal(t, "chat-req-token", upstream.headers["/backend-api/conversation"].Get("openai-sentinel-chat-requirements-token"))
}
