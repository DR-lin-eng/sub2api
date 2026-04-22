package service

import (
	"testing"

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
